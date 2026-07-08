package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

// Plan is the account capability set the server reports on login.
type Plan struct {
	SymbolLimit  int  `json:"symbol_limit"`
	CFDs         bool `json:"cfds"`
	TraderLadder bool `json:"trader_ladder"`
}

// Probe dials the streaming endpoint, logs in, and returns the plan
// capabilities plus how long connect+login took. Used by `doctor`.
func Probe(ctx context.Context, wsURL, key string) (Plan, time.Duration, error) {
	if wsURL == "" {
		wsURL = DefaultURL
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	start := time.Now()
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return Plan{}, 0, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	login := map[string]string{"action": "login", "key": key, "fmt": "JSON"}
	if err := conn.WriteJSON(login); err != nil {
		return Plan{}, 0, fmt.Errorf("send login: %w", err)
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return Plan{}, 0, fmt.Errorf("read: %w", err)
		}
		if !json.Valid(msg) {
			continue // plain-text greeting
		}
		var c struct {
			Type string `json:"type"`
			Plan
		}
		if err := json.Unmarshal(msg, &c); err != nil {
			continue
		}
		switch c.Type {
		case "login_ok":
			return c.Plan, time.Since(start), nil
		case "login_reject":
			return Plan{}, 0, ErrAuth
		}
	}
}
