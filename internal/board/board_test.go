package board

import (
	"testing"
	"time"
)

func TestSpread(t *testing.T) {
	cases := []struct {
		bid, ask, want string
	}{
		{"1.162700000", "1.162720000", "0.00002000"}, // wire precision, capped at 8 decimals
		{"148.25", "148.27", "0.02"},
		{"not-a-number", "1.0", "-"},
	}
	for _, tc := range cases {
		if got := spread(tc.bid, tc.ask); got != tc.want {
			t.Errorf("spread(%s, %s) = %s, want %s", tc.bid, tc.ask, got, tc.want)
		}
	}
}

func TestChangePct(t *testing.T) {
	if got := changePct(1.0, 1.01); got != "+1.000%" {
		t.Errorf("changePct = %s, want +1.000%%", got)
	}
	if got := changePct(0, 1.0); got != "-" {
		t.Errorf("changePct with no open = %s, want -", got)
	}
}

func TestSortedOrder(t *testing.T) {
	order := []string{"GBPUSD", "EURUSD", "XAUUSD"}
	change := map[string]float64{"GBPUSD": 0.001, "EURUSD": 0.02, "XAUUSD": -0.01}
	changeVal := func(s string) float64 { return change[s] }

	if got := sortedOrder(order, changeVal, SortList); got[0] != "GBPUSD" || got[2] != "XAUUSD" {
		t.Errorf("list order changed: %v", got)
	}
	if got := sortedOrder(order, changeVal, SortSymbol); got[0] != "EURUSD" || got[2] != "XAUUSD" {
		t.Errorf("symbol sort wrong: %v", got)
	}
	if got := sortedOrder(order, changeVal, SortChange); got[0] != "EURUSD" || got[2] != "XAUUSD" {
		t.Errorf("change sort wrong: %v", got)
	}
	// Input slice must not be mutated.
	if order[0] != "GBPUSD" {
		t.Errorf("input order mutated: %v", order)
	}
}

func TestNextSort(t *testing.T) {
	if nextSort(SortList) != SortSymbol || nextSort(SortSymbol) != SortChange || nextSort(SortChange) != SortList {
		t.Error("sort cycle broken")
	}
}

func TestAge(t *testing.T) {
	now := time.Now()
	cases := []struct {
		ago  time.Duration
		want string
	}{
		{500 * time.Millisecond, "now"},
		{5 * time.Second, "5s"},
		{3 * time.Minute, "3m"},
		{2 * time.Hour, "2h"},
	}
	for _, tc := range cases {
		if got := age(now, now.Add(-tc.ago)); got != tc.want {
			t.Errorf("age(-%s) = %s, want %s", tc.ago, got, tc.want)
		}
	}
}
