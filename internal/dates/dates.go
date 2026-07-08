// Package dates parses the date inputs the CLI accepts (yesterday,
// 2026-07-01, --last 30d) into the formats the API wants.
package dates

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// API wire formats.
const (
	DayFormat      = "2006-01-02"
	DateTimeFormat = "2006-01-02-15:04"
)

// ParseDay resolves "today", "yesterday", or an explicit 2006-01-02 date.
func ParseDay(s string, now time.Time) (time.Time, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "today":
		return now, nil
	case "yesterday":
		return now.AddDate(0, 0, -1), nil
	}
	t, err := time.Parse(DayFormat, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q - use 2006-01-02, today, or yesterday", s)
	}
	return t, nil
}

// ParseAt resolves an intraday point: 2006-01-02-15:04.
func ParseAt(s string) (time.Time, error) {
	t, err := time.Parse(DateTimeFormat, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time %q - use YYYY-MM-DD-HH:MM, e.g. 2026-07-01-14:30", s)
	}
	return t, nil
}

// ParseSpan converts "30d", "2w", "12h", or "45m" into a duration.
func ParseSpan(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) < 2 {
		return 0, spanErr(s)
	}
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil || n <= 0 {
		return 0, spanErr(s)
	}
	switch s[len(s)-1] {
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'm':
		return time.Duration(n) * time.Minute, nil
	}
	return 0, spanErr(s)
}

func spanErr(s string) error {
	return fmt.Errorf("invalid span %q - use a number plus d(ays), w(eeks), h(ours), or m(inutes), e.g. 30d", s)
}

// FormatFor renders t in the wire format the given timeseries interval needs.
func FormatFor(t time.Time, interval string) string {
	if interval == "daily" {
		return t.Format(DayFormat)
	}
	return t.Format(DateTimeFormat)
}
