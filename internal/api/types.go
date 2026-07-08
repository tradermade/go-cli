package api

import "strings"

// Num absorbs a JSON value the API sends sometimes as a number and sometimes
// as a quoted string (OHLC fields vary by endpoint). Keeps the wire text as-is.
type Num string

func (n *Num) UnmarshalJSON(b []byte) error {
	*n = Num(strings.Trim(string(b), `"`))
	return nil
}

func (n Num) String() string {
	if n == "" {
		return "-"
	}
	return string(n)
}

// symbolFrom builds a display symbol from either a currency pair or a CFD
// instrument name, whichever the endpoint returned.
func symbolFrom(base, quote, instrument string) string {
	if instrument != "" {
		return instrument
	}
	return base + quote
}
