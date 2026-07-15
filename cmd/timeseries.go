package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/api"
	"github.com/tradermade/go-cli/internal/dates"
	"github.com/tradermade/go-cli/internal/output"
)

var (
	tsStart    string
	tsEnd      string
	tsLast     string
	tsInterval string
	tsPeriod   int
	tsWeekend  bool
	tsSave     string
)

var timeseriesCmd = &cobra.Command{
	Use:     "timeseries SYMBOL",
	Short:   "OHLC candle ranges (REST /api/v1/timeseries)",
	GroupID: "rest",
	Long: `Get an OHLC candle range from:
  GET https://marketdata.tradermade.com/api/v1/timeseries

API query construction:
  currency    Required positional SYMBOL; exactly one pair per request.
  start_date  From --start, or calculated from --last. Daily uses YYYY-MM-DD;
              hourly/minute uses YYYY-MM-DD-HH:MM. Required.
  end_date    From --end (default: current UTC time), or calculated by --last.
  interval    --interval: daily, hourly, or minute. Default: daily.
  period      --period. Daily: 1; hourly: 1,2,4,6,8,24;
              minute: 1,5,10,15,30. Omitted means the API default.
  format      Always records so table/CSV can be constructed consistently.
  weekend     --weekend sends weekend=true; supported for crypto pairs only.
  api_key     Added automatically from the configured REST key.

Range limits per API call: daily one year, hourly one month, minute two days.
--last accepts d (days), w (weeks), h (hours), or m (minutes) and cannot be
combined with --start/--end. --output json preserves the server JSON exactly.
--save requires a .csv filename. A bare filename is created in the current
working directory; a path is accepted when it includes the filename.`,
	Example: `  tradermade timeseries EURUSD --start 2026-06-01 --end 2026-07-01
  tradermade timeseries EURUSD --last 30d
  tradermade timeseries GBPUSD --last 12h --interval hourly
  tradermade timeseries EURUSD --last 90m --interval minute --period 15
  tradermade timeseries EURUSD --last 30d --output json
  tradermade timeseries EURUSD --last 30d --save eurusd-month.csv`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch tsInterval {
		case "daily", "hourly", "minute":
		default:
			return fmt.Errorf("invalid --interval %q - use daily, hourly, or minute", tsInterval)
		}

		now := time.Now().UTC()
		var start, end time.Time
		var err error
		switch {
		case tsLast != "" && (tsStart != "" || tsEnd != ""):
			return fmt.Errorf("--last cannot be combined with --start/--end")
		case tsLast != "":
			span, err := dates.ParseSpan(tsLast)
			if err != nil {
				return err
			}
			start, end = now.Add(-span), now
		case tsStart != "":
			if start, err = parsePoint(tsStart, now); err != nil {
				return err
			}
			end = now
			if tsEnd != "" {
				if end, err = parsePoint(tsEnd, now); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("provide a range: --start/--end or --last (e.g. --last 30d)")
		}
		if err := validateTimeseriesRange(tsInterval, tsPeriod, start, end); err != nil {
			return err
		}

		if tsSave != "" {
			tsSave, err = resolveSavePath(tsSave)
			if err != nil {
				return err
			}
		}
		client, format, err := restClient(false)
		if err != nil {
			return err
		}
		symbol := strings.ToUpper(strings.TrimSpace(args[0]))
		startStr := dates.FormatFor(start, tsInterval)
		endStr := dates.FormatFor(end, tsInterval)

		var resp *api.TimeseriesResponse
		wireJSON := format == output.JSON
		if wireJSON {
			body, err := client.TimeseriesRaw(cmd.Context(), symbol, startStr, endStr, tsInterval, tsPeriod, tsWeekend)
			if err != nil {
				return err
			}
			if err := printServerBody(body); err != nil {
				return err
			}
			if tsSave == "" {
				return nil
			}
			// Still parse so --save can write the CSV alongside the raw view.
			resp = &api.TimeseriesResponse{}
			if err := jsonUnmarshal(body, resp); err != nil {
				return err
			}
		} else {
			resp, err = client.Timeseries(cmd.Context(), symbol, startStr, endStr, tsInterval, tsPeriod, tsWeekend)
			if err != nil {
				return err
			}
		}

		header := []string{"date", "open", "high", "low", "close"}
		rows := make([][]string, 0, len(resp.Quotes))
		for _, q := range resp.Quotes {
			rows = append(rows, []string{q.Date,
				q.Open.String(), q.High.String(), q.Low.String(), q.Close.String()})
		}

		if tsSave != "" {
			if err := saveCSV(tsSave, header, rows); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "saved %d rows to %s\n", len(rows), tsSave)
			if wireJSON {
				return nil
			}
		}
		if wireJSON {
			return nil
		}

		switch format {
		case output.CSV:
			return output.WriteCSV(os.Stdout, header, rows)
		}

		w := output.TableWriter(os.Stdout)
		fmt.Fprintln(w, "DATE\tOPEN\tHIGH\tLOW\tCLOSE")
		for _, q := range resp.Quotes {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", q.Date, q.Open, q.High, q.Low, q.Close)
		}
		if err := w.Flush(); err != nil {
			return err
		}
		fmt.Printf("\n%s %s candles, %d rows\n", symbol, tsInterval, len(resp.Quotes))
		return nil
	},
}

func validateTimeseriesRange(interval string, period int, start, end time.Time) error {
	if end.Before(start) {
		return fmt.Errorf("range end must not be before start")
	}
	validPeriod := period == 0
	switch interval {
	case "daily":
		validPeriod = validPeriod || period == 1
		if end.After(start.AddDate(1, 0, 0)) {
			return fmt.Errorf("daily timeseries requests are limited to one year per call")
		}
	case "hourly":
		validPeriod = validPeriod || period == 1 || period == 2 || period == 4 || period == 6 || period == 8 || period == 24
		if end.Sub(start) > 31*24*time.Hour {
			return fmt.Errorf("hourly timeseries requests are limited to one month per call")
		}
	case "minute":
		validPeriod = validPeriod || period == 1 || period == 5 || period == 10 || period == 15 || period == 30
		if end.Sub(start) > 48*time.Hour {
			return fmt.Errorf("minute timeseries requests are limited to two days per call")
		}
	}
	if !validPeriod {
		return fmt.Errorf("invalid --period %d for %s interval; use daily: 1, hourly: 1/2/4/6/8/24, minute: 1/5/10/15/30", period, interval)
	}
	return nil
}

// parsePoint accepts either a day (2006-01-02 / today / yesterday) or an
// intraday point (2006-01-02-15:04).
func parsePoint(s string, now time.Time) (time.Time, error) {
	if t, err := dates.ParseAt(s); err == nil {
		return t, nil
	}
	return dates.ParseDay(s, now)
}

func init() {
	timeseriesCmd.Flags().StringVar(&tsStart, "start", "", "range start: 2006-01-02 or 2006-01-02-15:04")
	timeseriesCmd.Flags().StringVar(&tsEnd, "end", "", "range end (default: now)")
	timeseriesCmd.Flags().StringVar(&tsLast, "last", "", "relative range, e.g. 30d, 2w, 12h, 90m")
	timeseriesCmd.Flags().StringVar(&tsInterval, "interval", "daily", "candle size: daily, hourly, or minute")
	timeseriesCmd.Flags().IntVar(&tsPeriod, "period", 0, "interval multiplier, e.g. --interval minute --period 15")
	timeseriesCmd.Flags().BoolVar(&tsWeekend, "weekend", false, "include weekend crypto data (crypto pairs only)")
	timeseriesCmd.Flags().StringVar(&tsSave, "save", "",
		"write CSV to a .csv filename (overwrites)")
	rootCmd.AddCommand(timeseriesCmd)
}
