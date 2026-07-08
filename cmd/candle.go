package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/api"
	"github.com/tradermade/go-cli/internal/dates"
	"github.com/tradermade/go-cli/internal/output"
)

var (
	candleAt   string
	candleHour bool
)

var candleCmd = &cobra.Command{
	Use:   "candle SYMBOL --at YYYY-MM-DD-HH:MM",
	Short: "A single minute or hour candle at an exact time",
	Example: `  tradermade candle EURUSD --at 2026-07-01-14:30
  tradermade candle EURUSD --at 2026-07-01-14:00 --hour`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		at, err := dates.ParseAt(candleAt)
		if err != nil {
			return err
		}
		client, format, err := restClient()
		if err != nil {
			return err
		}
		symbol := strings.ToUpper(strings.TrimSpace(args[0]))

		var resp *api.CandleResponse
		if candleHour {
			resp, err = client.HourHistorical(cmd.Context(), symbol, at.Format(dates.DateTimeFormat))
		} else {
			resp, err = client.MinuteHistorical(cmd.Context(), symbol, at.Format(dates.DateTimeFormat))
		}
		if err != nil {
			return err
		}

		switch format {
		case output.JSON:
			return output.PrintJSON(os.Stdout, resp)
		case output.CSV:
			return output.WriteCSV(os.Stdout,
				[]string{"symbol", "date_time", "open", "high", "low", "close"},
				[][]string{{resp.Currency, resp.DateTime,
					resp.Open.String(), resp.High.String(), resp.Low.String(), resp.Close.String()}})
		}

		w := output.TableWriter(os.Stdout)
		fmt.Fprintln(w, "SYMBOL\tTIME\tOPEN\tHIGH\tLOW\tCLOSE")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			resp.Currency, resp.DateTime, resp.Open, resp.High, resp.Low, resp.Close)
		return w.Flush()
	},
}

func init() {
	candleCmd.Flags().StringVar(&candleAt, "at", "", "exact time, e.g. 2026-07-01-14:30 (required)")
	candleCmd.Flags().BoolVar(&candleHour, "hour", false, "fetch an hour candle instead of a minute candle")
	_ = candleCmd.MarkFlagRequired("at")
	rootCmd.AddCommand(candleCmd)
}
