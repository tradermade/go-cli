package output

import (
	"strings"
	"testing"
)

func TestParseFormat(t *testing.T) {
	for _, ok := range []string{"table", "json", "csv"} {
		if _, err := ParseFormat(ok); err != nil {
			t.Errorf("ParseFormat(%q): %v", ok, err)
		}
	}
	if _, err := ParseFormat("xml"); err == nil {
		t.Error("ParseFormat(xml) should fail")
	}
}

func TestWriteCSV(t *testing.T) {
	var b strings.Builder
	err := WriteCSV(&b, []string{"symbol", "bid"}, [][]string{
		{"EURUSD", "1.14"},
		{"GBPUSD", "1.33"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "symbol,bid\nEURUSD,1.14\nGBPUSD,1.33\n"
	if b.String() != want {
		t.Errorf("got %q, want %q", b.String(), want)
	}
}
