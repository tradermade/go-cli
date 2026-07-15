package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient points a Client at a stub server.
func newTestClient(t *testing.T, key string, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := New(key)
	c.BaseURL = srv.URL
	return c
}

func TestLiveParsesPairsAndCFDs(t *testing.T) {
	c := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("api_key"); got != "k" {
			t.Errorf("api_key = %q", got)
		}
		if got := r.URL.Query().Get("currency"); got != "EURUSD,UK100" {
			t.Errorf("currency = %q", got)
		}
		w.Write([]byte(`{"endpoint":"live","quotes":[
			{"base_currency":"EUR","quote_currency":"USD","bid":1.1,"ask":1.2,"mid":1.15},
			{"instrument":"UK100","bid":8000,"ask":8002,"mid":8001}
		],"requested_time":"now","timestamp":1}`))
	})

	resp, err := c.Live(context.Background(), []string{"EURUSD", "UK100"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Quotes) != 2 {
		t.Fatalf("quotes = %d", len(resp.Quotes))
	}
	if resp.Quotes[0].Symbol() != "EURUSD" || resp.Quotes[1].Symbol() != "UK100" {
		t.Errorf("symbols: %s, %s", resp.Quotes[0].Symbol(), resp.Quotes[1].Symbol())
	}
}

func TestErrorSurfacesServerMessage(t *testing.T) {
	c := newTestClient(t, "bad", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"api key is invalid"}`))
	})

	_, err := c.Live(context.Background(), []string{"EURUSD"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Message, "api key is invalid") {
		t.Errorf("server message missing: %s", apiErr.Message)
	}
}

func TestWSKeyHintOnRESTFailure(t *testing.T) {
	c := newTestClient(t, "wsSomeStreamKey", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.Live(context.Background(), []string{"EURUSD"})
	if err == nil || !strings.Contains(err.Error(), "WebSocket streaming key") {
		t.Errorf("expected ws-key hint, got: %v", err)
	}
}

func TestNoHintForNormalKeys(t *testing.T) {
	c := newTestClient(t, "regular-key", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.Live(context.Background(), []string{"EURUSD"})
	if err == nil || strings.Contains(err.Error(), "WebSocket streaming key") {
		t.Errorf("hint should not appear for non-ws keys: %v", err)
	}
}

func TestConvertBuildsCorrectRequest(t *testing.T) {
	c := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("from") != "USD" || q.Get("to") != "INR" || q.Get("amount") != "1000" {
			t.Errorf("params: %v", q)
		}
		w.Write([]byte(`{"base_currency":"USD","quote_currency":"INR","quote":94.8,"total":94800,"requested_time":"now","timestamp":1}`))
	})

	resp, err := c.Convert(context.Background(), "USD", "INR", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 94800 {
		t.Errorf("total = %v", resp.Total)
	}
}

func TestTimeseriesMixedNumberEncodings(t *testing.T) {
	// The API mixes number and string OHLC encodings; Num must absorb both.
	c := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"base_currency":"EUR","quote_currency":"USD","endpoint":"timeseries","quotes":[
			{"date":"2026-07-01","open":1.1,"high":"1.2","low":1.05,"close":"1.15"}
		],"request_time":"now"}`))
	})

	resp, err := c.Timeseries(context.Background(), "EURUSD", "2026-06-01", "2026-07-01", "daily", 0, false)
	if err != nil {
		t.Fatal(err)
	}
	q := resp.Quotes[0]
	if q.Open.String() != "1.1" || q.High.String() != "1.2" || q.Close.String() != "1.15" {
		t.Errorf("mixed encodings parsed wrong: %+v", q)
	}
}

func TestTimeseriesBuildsDocumentedParameters(t *testing.T) {
	c := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		for key, want := range map[string]string{
			"currency": "BTCUSD", "start_date": "2026-07-01-00:00",
			"end_date": "2026-07-02-00:00", "interval": "minute",
			"period": "15", "format": "records", "weekend": "true", "api_key": "k",
		} {
			if got := q.Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		w.Write([]byte(`{"base_currency":"BTC","quote_currency":"USD","endpoint":"timeseries","quotes":[],"request_time":"now"}`))
	})

	if _, err := c.Timeseries(context.Background(), "BTCUSD", "2026-07-01-00:00", "2026-07-02-00:00", "minute", 15, true); err != nil {
		t.Fatal(err)
	}
}

func TestLiveRawPreservesServerJSON(t *testing.T) {
	want := []byte(`{"endpoint":"live", "new_server_field":42}`)
	c := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		w.Write(want)
	})
	got, err := c.LiveRaw(context.Background(), []string{"EURUSD"})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("got %q, want exact %q", got, want)
	}
}

func TestRetriesTransportErrors(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Kill the connection mid-response to simulate transport failure.
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("hijack unsupported")
			}
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}
		w.Write([]byte(`{"endpoint":"live","quotes":[],"requested_time":"now","timestamp":1}`))
	}))
	t.Cleanup(srv.Close)

	c := New("k")
	c.BaseURL = srv.URL
	if _, err := c.Live(context.Background(), []string{"EURUSD"}); err != nil {
		t.Fatalf("expected retry to succeed, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}
