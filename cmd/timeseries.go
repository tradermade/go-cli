package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/tradermade/tradermade-cli/internal/dates"
	"github.com/tradermade/tradermade-cli/internal/output"
)

var (
	tsStart    string
	tsEnd      string
	tsLast     string
	tsInterval string
	tsPeriod   int
)

var timeseriesCmd = &cobra.Command{
	Use:   "timeseries SYMBOL",
	Short: "A range of OHLC candles (daily, hourly, or minute)",
	Example: `  tradermade timeseries EURUSD --start 2026-06-01 --end 2026-07-01
  tradermade timeseries EURUSD --last 30d
  tradermade timeseries GBPUSD --last 12h --interval hourly
  tradermade timeseries EURUSD --last 90m --interval minute --period 15`,
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

		client, format, err := restClient()
		if err != nil {
			return err
		}
		symbol := strings.ToUpper(strings.TrimSpace(args[0]))

		resp, err := client.Timeseries(cmd.Context(), symbol,
			dates.FormatFor(start, tsInterval), dates.FormatFor(end, tsInterval),
			tsInterval, tsPeriod)
		if err != nil {
			return err
		}

		switch format {
		case output.JSON:
			return output.PrintJSON(os.Stdout, resp)
		case output.CSV:
			rows := make([][]string, 0, len(resp.Quotes))
			for _, q := range resp.Quotes {
				rows = append(rows, []string{q.Date,
					q.Open.String(), q.High.String(), q.Low.String(), q.Close.String()})
			}
			return output.WriteCSV(os.Stdout, []string{"date", "open", "high", "low", "close"}, rows)
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
	rootCmd.AddCommand(timeseriesCmd)
}
