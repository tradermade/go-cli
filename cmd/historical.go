package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/api"
	"github.com/tradermade/go-cli/internal/dates"
	"github.com/tradermade/go-cli/internal/output"
)

var (
	historicalDate string
	historicalSave string
)

var historicalCmd = &cobra.Command{
	Use:     "historical SYMBOL [SYMBOL...]",
	Short:   "Daily OHLC (REST /api/v1/historical)",
	GroupID: "rest",
	Long: `Get one daily OHLC candle for one or more symbols from:
  GET https://marketdata.tradermade.com/api/v1/historical

API query construction:
  currency  Required. Positional SYMBOL arguments joined with commas.
  date      --date in YYYY-MM-DD form. "today" and "yesterday" are resolved
            in UTC before the request. Default: yesterday.
  api_key   Added automatically from the configured REST key.

--output json prints the JSON response exactly as the server sent it.
--save requires a .csv filename. A bare filename is created in the current
working directory; a path is accepted when it includes the filename. Saving
appends: re-running with the same filename adds rows and keeps earlier data,
writing the header only once.`,
	Example: `  tradermade historical EURUSD --date 2026-07-01
  tradermade historical EURUSD GBPUSD --date yesterday
  tradermade historical EURUSD --output json
  tradermade historical EURUSD --save eurusd.csv`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		day, err := dates.ParseDay(historicalDate, time.Now().UTC())
		if err != nil {
			return err
		}
		if historicalSave != "" {
			historicalSave, err = resolveSavePath(historicalSave)
			if err != nil {
				return err
			}
		}
		client, format, err := restClient(false)
		if err != nil {
			return err
		}
		symbols := upperSymbols(args)
		date := day.Format(dates.DayFormat)

		var resp *api.HistoricalResponse
		wireJSON := format == output.JSON
		if wireJSON {
			body, err := client.HistoricalRaw(cmd.Context(), symbols, date)
			if err != nil {
				return err
			}
			if err := printServerBody(body); err != nil {
				return err
			}
			if historicalSave == "" {
				return nil
			}
			// Still parse so --save can write the CSV alongside the raw view.
			resp = &api.HistoricalResponse{}
			if err := jsonUnmarshal(body, resp); err != nil {
				return err
			}
		} else {
			resp, err = client.Historical(cmd.Context(), symbols, date)
			if err != nil {
				return err
			}
		}

		header := []string{"symbol", "date", "open", "high", "low", "close"}
		rows := make([][]string, 0, len(resp.Quotes))
		for _, q := range resp.Quotes {
			rows = append(rows, []string{q.Symbol(), resp.Date,
				q.Open.String(), q.High.String(), q.Low.String(), q.Close.String()})
		}

		if historicalSave != "" {
			if err := saveCSV(historicalSave, header, rows); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "saved %d rows to %s\n", len(rows), historicalSave)
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
		fmt.Fprintln(w, "SYMBOL\tOPEN\tHIGH\tLOW\tCLOSE")
		for _, q := range resp.Quotes {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", q.Symbol(), q.Open, q.High, q.Low, q.Close)
		}
		if err := w.Flush(); err != nil {
			return err
		}
		fmt.Printf("\ndaily candle for %s\n", resp.Date)
		return nil
	},
}

func init() {
	historicalCmd.Flags().StringVar(&historicalDate, "date", "yesterday",
		"date: 2006-01-02, today, or yesterday")
	historicalCmd.Flags().StringVar(&historicalSave, "save", "",
		"write CSV to a .csv filename (appends; header written once)")
	rootCmd.AddCommand(historicalCmd)
}
