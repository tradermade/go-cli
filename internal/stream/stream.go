// Package stream is a client for the TraderMade WebSocket streaming API
// (wss://stream.tradermade.com/feedAdv).
//
// Protocol summary (https://tradermade.com/docs/streaming-data-api):
//
//	→ {"action":"login","key":"...","fmt":"JSON"}
//	← {"type":"login_ok","symbol_limit":54,...}   or {"type":"login_reject",...}
//	→ {"action":"subscribe","symbols":["EURUSD:QUOTE"]}
//	← {"type":"sub_ack","accepted":[...],"denied":[...],"denied_reasons":{...}}
//	← {"t":"QUOTE","s":"EURUSD","b":"1.16270","a":"1.16272","ts":"20260515-12:36:35.588",...}
//
// The server does not persist subscriptions across disconnects, so Run
// resubscribes after every reconnect. Login rejections abort instead of
// retrying forever.
package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// DefaultURL is the production streaming endpoint.
const DefaultURL = "wss://stream.tradermade.com/feedAdv"

// ErrAuth marks a login rejection - retrying will not help.
var ErrAuth = errors.New("login rejected: invalid API key or streaming not enabled on your plan")

// Tick is a live market update. Mid and Ladder are only present when the
// login asked for ladder data.
//
// Price fields are strings, but careful: plain frames quote every value
// while ladder frames send b/a/m as bare JSON numbers. The custom
// unmarshaller below absorbs both so neither frame type gets dropped.
type Tick struct {
	Type      string  `json:"t"`
	Symbol    string  `json:"s"`
	Bid       string  `json:"b"`
	Ask       string  `json:"a"`
	Mid       string  `json:"m"`
	BidVolume string  `json:"bv"`
	AskVolume string  `json:"av"`
	Timestamp string  `json:"ts"`
	Ladder    *Ladder `json:"ladder"`
}

func (t *Tick) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type      string          `json:"t"`
		Symbol    string          `json:"s"`
		Bid       json.RawMessage `json:"b"`
		Ask       json.RawMessage `json:"a"`
		Mid       json.RawMessage `json:"m"`
		BidVolume json.RawMessage `json:"bv"`
		AskVolume json.RawMessage `json:"av"`
		Timestamp string          `json:"ts"`
		Ladder    *Ladder         `json:"ladder"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	t.Type, t.Symbol, t.Timestamp, t.Ladder = raw.Type, raw.Symbol, raw.Timestamp, raw.Ladder
	t.Bid = unquote(raw.Bid)
	t.Ask = unquote(raw.Ask)
	t.Mid = unquote(raw.Mid)
	t.BidVolume = unquote(raw.BidVolume)
	t.AskVolume = unquote(raw.AskVolume)
	return nil
}

// unquote keeps the wire text of a value that may or may not be quoted.
func unquote(r json.RawMessage) string {
	return strings.Trim(string(r), `"`)
}

// Ladder is market depth: [price, volume] levels, best first.
type Ladder struct {
	Asks [][]string `json:"a"`
	Bids [][]string `json:"b"`
}

// control covers every non-tick message the server can send.
type control struct {
	Type          string            `json:"type"`
	SymbolLimit   int               `json:"symbol_limit"`
	Accepted      []string          `json:"accepted"`
	Denied        []string          `json:"denied"`
	DeniedReasons map[string]string `json:"denied_reasons"`
	Invalid       []string          `json:"invalid"`
	Reason        string            `json:"reason"`
}

// Ping every 30s; if nothing arrives (message or pong) within readTimeout
// the read fails and we reconnect. Catches half-open connections when the
// market is quiet.
const (
	pingInterval = 30 * time.Second
	readTimeout  = 75 * time.Second
)

// Options configures a streaming session.
type Options struct {
	URL     string   // defaults to DefaultURL
	Key     string   // TraderMade API key
	Symbols []string // plain symbols, e.g. EURUSD - ":QUOTE" is appended automatically
	// SendLast asks the server to send the cached last tick on subscribe.
	SendLast bool
	// SendLadder asks for market depth on login. Needs the trader ladder
	// enabled on the plan; ticks then carry a Ladder and a mid price.
	SendLadder bool
	// OnTick gets every market update; raw is the original wire payload.
	OnTick func(t Tick, raw []byte)
	// OnRaw, if set, gets every frame exactly as received - control
	// messages, greetings, ticks - before any parsing or filtering.
	OnRaw func(raw []byte)
	// Logf, if set, receives connection lifecycle messages.
	Logf func(format string, args ...any)
}

func (o *Options) logf(format string, args ...any) {
	if o.Logf != nil {
		o.Logf(format, args...)
	}
}

// wireSymbols normalizes user symbols to the SYMBOL:QUOTE wire format.
func wireSymbols(symbols []string) []string {
	out := make([]string, 0, len(symbols))
	for _, s := range symbols {
		s = strings.ToUpper(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		if !strings.Contains(s, ":") {
			s += ":QUOTE"
		}
		out = append(out, s)
	}
	return out
}

// Run connects, logs in, subscribes, and dispatches ticks until ctx is
// cancelled. Transient failures reconnect with exponential backoff (1s → 30s);
// auth failures return ErrAuth immediately.
func Run(ctx context.Context, opts Options) error {
	if opts.URL == "" {
		opts.URL = DefaultURL
	}
	symbols := wireSymbols(opts.Symbols)
	if len(symbols) == 0 {
		return errors.New("no symbols to subscribe to")
	}

	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		err := runOnce(ctx, opts, symbols, &backoff)
		if ctx.Err() != nil {
			return nil // clean shutdown via Ctrl+C
		}
		if errors.Is(err, ErrAuth) {
			return err
		}
		opts.logf("connection lost: %v - reconnecting in %s", err, backoff)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, maxBackoff)
	}
}

// runOnce is a single connect → login → subscribe → read-loop cycle.
// It resets *backoff once login succeeds so a stable session that later
// drops starts reconnecting from 1s again.
func runOnce(ctx context.Context, opts Options, symbols []string, backoff *time.Duration) error {
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.DialContext(ctx, opts.URL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	// Unblock the blocking ReadMessage below when the context is cancelled.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-done:
		}
	}()

	// Any message or pong pushes the deadline out. WriteControl is safe to
	// call concurrently with the read loop's writes.
	_ = conn.SetReadDeadline(time.Now().Add(readTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(readTimeout))
	})
	go func() {
		t := time.NewTicker(pingInterval)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-t.C:
				_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second))
			}
		}
	}()

	// TRADERMADE_WS_DEBUG=1 logs the raw frames we send, for support cases.
	wsDebug := os.Getenv("TRADERMADE_WS_DEBUG") != ""

	login := map[string]any{"action": "login", "key": opts.Key, "fmt": "JSON"}
	if opts.SendLadder {
		login["send_ladder"] = true
	}
	if wsDebug {
		raw, _ := json.Marshal(login)
		opts.logf("debug send: %s", string(raw))
	}
	if err := conn.WriteJSON(login); err != nil {
		return fmt.Errorf("send login: %w", err)
	}

	loggedIn := false
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(readTimeout))

		if opts.OnRaw != nil {
			opts.OnRaw(msg)
		}

		// The server may send plain-text greetings (e.g. "Connected") - skip them.
		if !json.Valid(msg) {
			continue
		}

		// Ticks use "t"; control messages use "type". Sniff both.
		var kind struct {
			T    string `json:"t"`
			Type string `json:"type"`
		}
		if err := json.Unmarshal(msg, &kind); err != nil {
			continue
		}

		switch {
		case kind.T == "QUOTE" || kind.T == "LAST_QUOTE":
			var tick Tick
			if err := json.Unmarshal(msg, &tick); err != nil {
				// Never drop a frame silently - surface parse problems.
				opts.logf("unparseable tick: %v: %s", err, string(msg))
				continue
			}
			if opts.OnTick != nil {
				opts.OnTick(tick, msg)
			}

		case kind.Type == "login_ok":
			loggedIn = true
			*backoff = time.Second
			var c control
			_ = json.Unmarshal(msg, &c)
			if wsDebug {
				opts.logf("debug recv: %s", string(msg))
			}
			opts.logf("connected - plan allows %d simultaneous symbols", c.SymbolLimit)
			sub := map[string]any{"action": "subscribe", "symbols": symbols}
			if opts.SendLast {
				sub["send_last"] = true
			}
			if wsDebug {
				raw, _ := json.Marshal(sub)
				opts.logf("debug send: %s", string(raw))
			}
			if err := conn.WriteJSON(sub); err != nil {
				return fmt.Errorf("send subscribe: %w", err)
			}

		case kind.Type == "login_reject":
			return ErrAuth

		case kind.Type == "sub_ack":
			var c control
			_ = json.Unmarshal(msg, &c)
			if len(c.Accepted) > 0 {
				opts.logf("subscribed: %s", strings.Join(c.Accepted, ", "))
			}
			for _, d := range c.Denied {
				opts.logf("denied: %s (%s)", d, c.DeniedReasons[d])
			}
			if len(c.Invalid) > 0 {
				opts.logf("invalid symbols: %s", strings.Join(c.Invalid, ", "))
			}

		case kind.Type == "error":
			var c control
			_ = json.Unmarshal(msg, &c)
			opts.logf("server error: %s", c.Reason)

		default:
			// Unknown control message before login completed likely means a
			// protocol problem; after login, ignore quietly.
			if !loggedIn {
				opts.logf("unexpected message before login: %s", string(msg))
			}
		}
	}
}
