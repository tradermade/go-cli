package dates

import (
	"testing"
	"time"
)

var now = time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)

func TestParseDay(t *testing.T) {
	cases := []struct {
		in   string
		want string
		err  bool
	}{
		{"today", "2026-07-07", false},
		{"YESTERDAY", "2026-07-06", false},
		{"2026-07-01", "2026-07-01", false},
		{"01-07-2026", "", true},
		{"tomorrow", "", true},
	}
	for _, tc := range cases {
		got, err := ParseDay(tc.in, now)
		if tc.err {
			if err == nil {
				t.Errorf("ParseDay(%q): expected error, got %v", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseDay(%q): %v", tc.in, err)
			continue
		}
		if got.Format(DayFormat) != tc.want {
			t.Errorf("ParseDay(%q) = %s, want %s", tc.in, got.Format(DayFormat), tc.want)
		}
	}
}

func TestParseSpan(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		err  bool
	}{
		{"30d", 30 * 24 * time.Hour, false},
		{"2w", 14 * 24 * time.Hour, false},
		{"12h", 12 * time.Hour, false},
		{"90m", 90 * time.Minute, false},
		{"0d", 0, true},
		{"d", 0, true},
		{"30x", 0, true},
		{"", 0, true},
	}
	for _, tc := range cases {
		got, err := ParseSpan(tc.in)
		if tc.err != (err != nil) {
			t.Errorf("ParseSpan(%q): err = %v, want err = %v", tc.in, err, tc.err)
			continue
		}
		if !tc.err && got != tc.want {
			t.Errorf("ParseSpan(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestFormatFor(t *testing.T) {
	if got := FormatFor(now, "daily"); got != "2026-07-07" {
		t.Errorf("daily = %s", got)
	}
	if got := FormatFor(now, "hourly"); got != "2026-07-07-12:00" {
		t.Errorf("hourly = %s", got)
	}
}
