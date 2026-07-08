package stream

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestWireSymbols(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"appends quote suffix", []string{"EURUSD"}, []string{"EURUSD:QUOTE"}},
		{"uppercases", []string{"eurusd", "gbpusd"}, []string{"EURUSD:QUOTE", "GBPUSD:QUOTE"}},
		{"keeps explicit suffix", []string{"EURUSD:QUOTE"}, []string{"EURUSD:QUOTE"}},
		{"trims and drops empties", []string{" EURUSD ", "", "  "}, []string{"EURUSD:QUOTE"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := wireSymbols(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("wireSymbols(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestTickUnmarshal(t *testing.T) {
	// Verbatim QUOTE payload from the TraderMade streaming docs.
	raw := `{"a":"1.162720000","av":"100000","b":"1.162700000","bv":"100000","s":"EURUSD","t":"QUOTE","ts":"20260515-12:36:35.588"}`
	var tick Tick
	if err := json.Unmarshal([]byte(raw), &tick); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if tick.Symbol != "EURUSD" || tick.Bid != "1.162700000" || tick.Ask != "1.162720000" || tick.Type != "QUOTE" {
		t.Errorf("unexpected tick: %+v", tick)
	}
}
