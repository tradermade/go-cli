// Package output renders API data as tables, JSON, or CSV.
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"
)

// Format selects the rendering mode.
type Format string

const (
	Table Format = "table"
	JSON  Format = "json"
	CSV   Format = "csv"
)

// ParseFormat validates the --output flag value.
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case Table, JSON, CSV:
		return Format(s), nil
	default:
		return "", fmt.Errorf("invalid output format %q - use table, json, or csv", s)
	}
}

// PrintJSON writes v as indented JSON.
func PrintJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// WriteCSV writes an RFC 4180 CSV with a header row.
func WriteCSV(w io.Writer, header []string, rows [][]string) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, r := range rows {
		if err := cw.Write(r); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// TableWriter returns a tabwriter configured for aligned CLI tables.
// Call Flush on the returned writer when done.
func TableWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
}

// Price formats a float without artificial rounding or trailing zeros.
func Price(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
