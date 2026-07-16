// Package api is a thin client for the TraderMade REST API
// (https://tradermade.com/docs/restful-api). No CLI dependencies here so
// it can be pulled out as an SDK later.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL is the production REST endpoint root.
const DefaultBaseURL = "https://marketdata.tradermade.com/api/v1"

// UserAgent is sent with every request; the CLI stamps its version in at startup.
var UserAgent = "tradermade-cli"

// Client calls the TraderMade REST API.
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

// New returns a Client with sane defaults.
func New(apiKey string) *Client {
	return &Client{
		APIKey:  apiKey,
		BaseURL: DefaultBaseURL,
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

// APIError is a non-200 response mapped to a human-readable explanation.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return e.Message
}

// statusHint translates TraderMade's documented status codes into actionable text.
func statusHint(code int) string {
	switch code {
	case http.StatusUnauthorized:
		return "invalid API key, or your plan does not include this endpoint - check the key with `tradermade config show`"
	case http.StatusNoContent:
		return "no data available for the requested parameters"
	case http.StatusBadRequest:
		return "invalid request parameters - check the symbol and date formats"
	case http.StatusForbidden:
		return "data not available - markets closed (weekend) or the date is outside your plan's historical range"
	default:
		return fmt.Sprintf("unexpected API response (HTTP %d)", code)
	}
}

// Quote is one instrument's live price. Forex/crypto pairs populate
// BaseCurrency/QuoteCurrency; CFDs populate Instrument instead.
type Quote struct {
	BaseCurrency  string  `json:"base_currency,omitempty"`
	QuoteCurrency string  `json:"quote_currency,omitempty"`
	Instrument    string  `json:"instrument,omitempty"`
	Bid           float64 `json:"bid"`
	Ask           float64 `json:"ask"`
	Mid           float64 `json:"mid"`
}

// Symbol returns the display symbol regardless of instrument type.
func (q Quote) Symbol() string {
	if q.Instrument != "" {
		return q.Instrument
	}
	return q.BaseCurrency + q.QuoteCurrency
}

// LiveResponse is the /live endpoint payload.
type LiveResponse struct {
	Endpoint      string  `json:"endpoint"`
	Quotes        []Quote `json:"quotes"`
	RequestedTime string  `json:"requested_time"`
	Timestamp     int64   `json:"timestamp"`
}

// Live fetches real-time quotes for one or more symbols (e.g. EURUSD, XAUUSD, BTCUSD).
func liveParams(symbols []string) url.Values {
	params := url.Values{}
	params.Set("currency", strings.Join(symbols, ","))
	return params
}

func (c *Client) Live(ctx context.Context, symbols []string) (*LiveResponse, error) {
	var out LiveResponse
	if err := c.get(ctx, "/live", liveParams(symbols), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// LiveRaw returns the /live response body exactly as sent by the server.
func (c *Client) LiveRaw(ctx context.Context, symbols []string) ([]byte, error) {
	return c.getBody(ctx, "/live", liveParams(symbols))
}

// ConvertResponse is the /convert endpoint payload.
type ConvertResponse struct {
	BaseCurrency  string  `json:"base_currency"`
	QuoteCurrency string  `json:"quote_currency"`
	Quote         float64 `json:"quote"`
	Total         float64 `json:"total"`
	RequestedTime string  `json:"requested_time"`
	Timestamp     int64   `json:"timestamp"`
}

// Convert converts an amount from one currency to another at the live rate.
func convertParams(from, to string, amount float64) url.Values {
	params := url.Values{}
	params.Set("from", from)
	params.Set("to", to)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	return params
}

func (c *Client) Convert(ctx context.Context, from, to string, amount float64) (*ConvertResponse, error) {
	var out ConvertResponse
	if err := c.get(ctx, "/convert", convertParams(from, to, amount), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ConvertRaw returns the /convert response body exactly as sent by the server.
func (c *Client) ConvertRaw(ctx context.Context, from, to string, amount float64) ([]byte, error) {
	return c.getBody(ctx, "/convert", convertParams(from, to, amount))
}

// get performs a GET request against the API and decodes the JSON response.
func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	body, err := c.getBody(ctx, path, params)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("cannot parse API response: %w", err)
	}
	return nil
}

// getBody performs a GET request and returns the raw response body.
func (c *Client) getBody(ctx context.Context, path string, params url.Values) ([]byte, error) {
	params.Set("api_key", c.APIKey)
	reqURL := c.BaseURL + path + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	// Retry transient transport errors a couple of times. HTTP error
	// statuses are real answers and are not retried.
	var resp *http.Response
	for attempt := 0; ; attempt++ {
		resp, err = c.HTTP.Do(req)
		if err == nil || attempt == 2 || ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 300 * time.Millisecond):
		}
	}
	if err != nil {
		return nil, fmt.Errorf("network error calling TraderMade API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("cannot read API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := statusHint(resp.StatusCode)
		// Prefer the API's own message when the error body carries one.
		var apiMsg struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		if json.Unmarshal(body, &apiMsg) == nil {
			if apiMsg.Message != "" {
				msg = apiMsg.Message + " (" + msg + ")"
			} else if apiMsg.Error != "" {
				msg = apiMsg.Error + " (" + msg + ")"
			}
		}
		// TraderMade issues separate REST and WebSocket keys; a "ws"-prefixed
		// key hitting REST typically fails with 401 or 500.
		if (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode >= 500) &&
			strings.HasPrefix(strings.ToLower(c.APIKey), "ws") {
			msg += `` + "\nnote: your key starts with \"ws\" - that is usually a WebSocket streaming key; REST endpoints (live, convert, historical...) need a REST API key"
		}
		return nil, &APIError{StatusCode: resp.StatusCode, Message: msg}
	}

	return body, nil
}
