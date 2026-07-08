package watchlist

import (
	"reflect"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"uppercases and trims", []string{" eurusd ", "GBPUSD"}, []string{"EURUSD", "GBPUSD"}},
		{"dedupes preserving order", []string{"EURUSD", "gbpusd", "EURUSD"}, []string{"EURUSD", "GBPUSD"}},
		{"skips blanks and comments", []string{"", "# my majors", "EURUSD"}, []string{"EURUSD"}},
		{"empty input", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Normalize(tc.in); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Normalize(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
