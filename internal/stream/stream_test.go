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

func TestTickUnmarshalWithLadder(t *testing.T) {
	// Verbatim 4-level ladder QUOTE payload from the streaming docs.
	raw := `{"a":"1.16189","av":"100000","b":"1.16185","bv":"100000","ladder":{"a":[["1.1619000","2600000"],["1.1619200","250000"],["1.1619300","52350000"],["1.1619400","3000000"]],"b":[["1.1618400","2681000"],["1.1618200","2250000"],["1.1618100","51500000"],["1.1618000","3500000"]]},"m":"1.16187","s":"EURUSD","t":"QUOTE","ts":"20260522-17:30:12.842"}`
	var tick Tick
	if err := json.Unmarshal([]byte(raw), &tick); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if tick.Mid != "1.16187" {
		t.Errorf("mid = %q", tick.Mid)
	}
	if tick.Ladder == nil {
		t.Fatal("ladder missing")
	}
	if len(tick.Ladder.Asks) != 4 || len(tick.Ladder.Bids) != 4 {
		t.Fatalf("levels: %d asks, %d bids", len(tick.Ladder.Asks), len(tick.Ladder.Bids))
	}
	if tick.Ladder.Bids[0][0] != "1.1618400" || tick.Ladder.Bids[0][1] != "2681000" {
		t.Errorf("best bid level: %v", tick.Ladder.Bids[0])
	}
	if tick.Ladder.Asks[3][0] != "1.1619400" || tick.Ladder.Asks[3][1] != "3000000" {
		t.Errorf("deepest ask level: %v", tick.Ladder.Asks[3])
	}
}

func TestTickUnmarshalNumericLadderFrame(t *testing.T) {
	// Verbatim ladder frame captured from the live feed: b/a/m arrive as
	// bare JSON numbers, unlike plain frames which quote everything.
	raw := `{"a":1.34122,"av":"1000000","b":1.34111,"bv":"1000000","ladder":{"a":[["1.3412300","4500000"],["1.3412400","6000000"]],"b":[["1.3411000","3000000"],["1.3410900","4000000"]]},"m":1.3411650000000002,"s":"GBPUSD","t":"QUOTE","ts":"20260710-13:34:27.801"}`
	var tick Tick
	if err := json.Unmarshal([]byte(raw), &tick); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if tick.Bid != "1.34111" || tick.Ask != "1.34122" {
		t.Errorf("numeric prices parsed wrong: bid=%q ask=%q", tick.Bid, tick.Ask)
	}
	if tick.Mid != "1.3411650000000002" {
		t.Errorf("mid = %q", tick.Mid)
	}
	if tick.BidVolume != "1000000" {
		t.Errorf("bv = %q", tick.BidVolume)
	}
	if tick.Ladder == nil || len(tick.Ladder.Bids) != 2 {
		t.Fatalf("ladder missing or wrong: %+v", tick.Ladder)
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
