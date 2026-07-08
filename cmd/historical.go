package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/tradermade/tradermade-cli/internal/dates"
	"github.com/tradermade/tradermade-cli/internal/output"
)

var historicalDate string

var historicalCmd = &cobra.Command{
	Use:   "historical SYMBOL [SYMBOL...]",
	Short: "Daily open/high/low/close for a given date",
	Example: `  tradermade historical EURUSD --date 2026-07-01
  tradermade historical EURUSD GBPUSD --date yesterday
  tradermade historical XAUUSD --date 2026-07-01 --output json`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		day, err := dates.ParseDay(historicalDate, time.Now().UTC())
		if err != nil {
			return err
		}
		client, format, err := restClient()
		if err != nil {
			return err
		}

		resp, err := client.Historical(cmd.Context(), upperSymbols(args), day.Format(dates.DayFormat))
		if err != nil {
			return err
		}

		switch format {
		case output.JSON:
			return output.PrintJSON(os.Stdout, resp)
		case output.CSV:
			rows := make([][]string, 0, len(resp.Quotes))
			for _, q := range resp.Quotes {
				rows = append(rows, []string{q.Symbol(), resp.Date,
					q.Open.String(), q.High.String(), q.Low.String(), q.Close.String()})
			}
			return output.WriteCSV(os.Stdout, []string{"symbol", "date", "open", "high", "low", "close"}, rows)
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
	rootCmd.AddCommand(historicalCmd)
}
