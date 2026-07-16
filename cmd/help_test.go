package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestEndpointHelpDocumentsAllParameters(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{"live", restLiveCmd.Long, []string{"currency", "api_key", "--output json", "--save"}},
		{"convert", convertCmd.Long, []string{"amount", "from", "to", "api_key"}},
		{"historical", historicalCmd.Long, []string{"currency", "date", "api_key", "--save"}},
		{"timeseries", timeseriesCmd.Long, []string{"start_date", "end_date", "interval", "period", "format", "weekend", "api_key"}},
		{"candle", candleCmd.Long, []string{"currency", "date_time", "api_key", "--hour"}},
		{"symbols", symbolsCmd.Long, []string{"live_currencies_list", "live_crypto_list", "api_key"}},
		{"stream", streamCmd.Long, []string{"login.key", "login.fmt", "login.send_ladder", "subscribe.symbols", "subscribe.send_last"}},
		{"board", boardCmd.Long, []string{"WebSocket /feedAdv", "REST /historical", "REST /live"}},
		{"doctor", doctorCmd.Long, []string{"REST /api/v1/live", "WebSocket /feedAdv"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, want := range tc.want {
				if !strings.Contains(tc.text, want) {
					t.Errorf("help is missing %q", want)
				}
			}
		})
	}
}

func TestPublicLiveCommandNames(t *testing.T) {
	commands := make(map[string]*cobra.Command)
	for _, command := range rootCmd.Commands() {
		commands[command.Name()] = command
	}
	if commands["live"] != restLiveCmd {
		t.Fatal("live is not registered as the REST /api/v1/live command")
	}
	if commands["stream"] != streamCmd {
		t.Fatal("stream is not registered as the WebSocket command")
	}
	if _, exists := commands["quote"]; exists {
		t.Fatal("removed quote command is still registered")
	}
}

func TestValidateTimeseriesRange(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		interval string
		period   int
		end      time.Time
		wantErr  bool
	}{
		{"valid minute", "minute", 15, start.Add(24 * time.Hour), false},
		{"bad minute period", "minute", 4, start.Add(time.Hour), true},
		{"minute too long", "minute", 15, start.Add(49 * time.Hour), true},
		{"valid hourly", "hourly", 24, start.Add(30 * 24 * time.Hour), false},
		{"bad hourly period", "hourly", 15, start.Add(time.Hour), true},
		{"daily too long", "daily", 1, start.AddDate(1, 0, 1), true},
		{"end before start", "daily", 1, start.Add(-time.Hour), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTimeseriesRange(tc.interval, tc.period, start, tc.end)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
