package api

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// OHLC is one candle. Fields use Num because the API mixes number and
// string encodings across endpoints.
type OHLC struct {
	Open  Num `json:"open"`
	High  Num `json:"high"`
	Low   Num `json:"low"`
	Close Num `json:"close"`
}

// HistoricalQuote is one instrument's daily candle from /historical.
type HistoricalQuote struct {
	BaseCurrency  string `json:"base_currency,omitempty"`
	QuoteCurrency string `json:"quote_currency,omitempty"`
	Instrument    string `json:"instrument,omitempty"`
	OHLC
}

// Symbol returns the display symbol regardless of instrument type.
func (q HistoricalQuote) Symbol() string {
	return symbolFrom(q.BaseCurrency, q.QuoteCurrency, q.Instrument)
}

// HistoricalResponse is the /historical endpoint payload.
type HistoricalResponse struct {
	Date        string            `json:"date"`
	Endpoint    string            `json:"endpoint"`
	Quotes      []HistoricalQuote `json:"quotes"`
	RequestTime string            `json:"request_time"`
}

// Historical fetches daily OHLC for one or more symbols on a given day
// (date format 2006-01-02).
func (c *Client) Historical(ctx context.Context, symbols []string, date string) (*HistoricalResponse, error) {
	params := url.Values{}
	params.Set("currency", strings.Join(symbols, ","))
	params.Set("date", date)
	var out HistoricalResponse
	if err := c.get(ctx, "/historical", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TimeseriesQuote is one candle in a /timeseries range.
type TimeseriesQuote struct {
	Date string `json:"date"`
	OHLC
}

// TimeseriesResponse is the /timeseries endpoint payload.
type TimeseriesResponse struct {
	BaseCurrency  string            `json:"base_currency"`
	QuoteCurrency string            `json:"quote_currency"`
	Endpoint      string            `json:"endpoint"`
	Quotes        []TimeseriesQuote `json:"quotes"`
	RequestTime   string            `json:"request_time"`
}

// Timeseries fetches a candle range. interval is daily, hourly, or minute;
// period is the interval multiplier (e.g. hourly period 4 = 4-hour candles).
// Dates are 2006-01-02 for daily and 2006-01-02-15:04 for intraday.
func (c *Client) Timeseries(ctx context.Context, symbol, start, end, interval string, period int) (*TimeseriesResponse, error) {
	params := url.Values{}
	params.Set("currency", symbol)
	params.Set("start_date", start)
	params.Set("end_date", end)
	params.Set("interval", interval)
	if period > 0 {
		params.Set("period", strconv.Itoa(period))
	}
	var out TimeseriesResponse
	if err := c.get(ctx, "/timeseries", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CandleResponse is the /minute_historical and /hour_historical payload -
// a single candle with OHLC at the top level.
type CandleResponse struct {
	Currency    string `json:"currency"`
	DateTime    string `json:"date_time"`
	Endpoint    string `json:"endpoint"`
	RequestTime string `json:"request_time"`
	OHLC
}

// MinuteHistorical fetches one minute candle (date_time 2006-01-02-15:04).
func (c *Client) MinuteHistorical(ctx context.Context, symbol, dateTime string) (*CandleResponse, error) {
	return c.candle(ctx, "/minute_historical", symbol, dateTime)
}

// HourHistorical fetches one hour candle (date_time 2006-01-02-15:04).
func (c *Client) HourHistorical(ctx context.Context, symbol, dateTime string) (*CandleResponse, error) {
	return c.candle(ctx, "/hour_historical", symbol, dateTime)
}

func (c *Client) candle(ctx context.Context, path, symbol, dateTime string) (*CandleResponse, error) {
	params := url.Values{}
	params.Set("currency", symbol)
	params.Set("date_time", dateTime)
	var out CandleResponse
	if err := c.get(ctx, path, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
