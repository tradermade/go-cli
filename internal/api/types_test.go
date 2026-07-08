package api

import (
	"encoding/json"
	"testing"
)

func TestNumAbsorbsNumbersAndStrings(t *testing.T) {
	var c OHLC
	// Number form (most REST endpoints).
	if err := json.Unmarshal([]byte(`{"open":1.09,"high":1.1,"low":1.08,"close":1.095}`), &c); err != nil {
		t.Fatalf("number form: %v", err)
	}
	if c.Open.String() != "1.09" || c.Close.String() != "1.095" {
		t.Errorf("number form parsed wrong: %+v", c)
	}
	// String form (some endpoints quote their prices).
	if err := json.Unmarshal([]byte(`{"open":"1.09","high":"1.1","low":"1.08","close":"1.095"}`), &c); err != nil {
		t.Fatalf("string form: %v", err)
	}
	if c.Open.String() != "1.09" {
		t.Errorf("string form parsed wrong: %+v", c)
	}
}

func TestSymbolFrom(t *testing.T) {
	if got := symbolFrom("EUR", "USD", ""); got != "EURUSD" {
		t.Errorf("pair: %s", got)
	}
	if got := symbolFrom("", "", "UK100"); got != "UK100" {
		t.Errorf("cfd: %s", got)
	}
}
