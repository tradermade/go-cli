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
	Use:     "candle SYMBOL --at YYYY-MM-DD-HH:MM",
	Short:   "One candle (REST /api/v1/minute_historical or /api/v1/hour_historical)",
	GroupID: "rest",
	Long: `Get one exact-time OHLC candle.

Endpoint selection:
  default  GET https://marketdata.tradermade.com/api/v1/minute_historical
  --hour   GET https://marketdata.tradermade.com/api/v1/hour_historical

API query construction:
  currency   Required positional SYMBOL; uppercased.
  date_time  Required --at value in YYYY-MM-DD-HH:MM form.
  api_key    Added automatically from the configured REST key.

--output json prints the JSON response exactly as the server sent it.`,
	Example: `  tradermade candle EURUSD --at 2026-07-01-14:30
  tradermade candle EURUSD --at 2026-07-01-14:00 --hour`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		at, err := dates.ParseAt(candleAt)
		if err != nil {
			return err
		}
		client, format, err := restClient(false)
		if err != nil {
			return err
		}
		symbol := strings.ToUpper(strings.TrimSpace(args[0]))
		atString := at.Format(dates.DateTimeFormat)
		if format == output.JSON {
			body, err := client.CandleRaw(cmd.Context(), symbol, atString, candleHour)
			if err != nil {
				return err
			}
			return printServerBody(body)
		}

		var resp *api.CandleResponse
		if candleHour {
			resp, err = client.HourHistorical(cmd.Context(), symbol, atString)
		} else {
			resp, err = client.MinuteHistorical(cmd.Context(), symbol, atString)
		}
		if err != nil {
			return err
		}

		switch format {
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
