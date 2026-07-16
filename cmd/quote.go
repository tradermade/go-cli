package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/api"
	"github.com/tradermade/go-cli/internal/output"
)

var restLiveSave string

var restLiveCmd = &cobra.Command{
	Use:     "live SYMBOL [SYMBOL...]",
	Short:   "Live rates (REST /api/v1/live)",
	GroupID: "rest",
	Long: `Get live bid, ask, and mid prices from:
  GET https://marketdata.tradermade.com/api/v1/live

API query construction:
  currency  Required. Positional SYMBOL arguments joined with commas.
            Example: EURUSD GBPUSD becomes currency=EURUSD,GBPUSD.
  api_key   Added automatically from the configured REST key.

Symbols are uppercased. Forex, crypto pairs, and enabled CFD instruments are
accepted. --output json prints the JSON response exactly as the server sent it.
--save requires a .csv filename. A bare filename is created in the current
working directory; a path is accepted when it includes the filename.`,
	Example: `  tradermade live EURUSD
  tradermade live EURUSD GBPUSD XAUUSD
  tradermade live BTCUSD --output json`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if restLiveSave != "" {
			restLiveSave, err = resolveSavePath(restLiveSave)
			if err != nil {
				return err
			}
		}
		client, format, err := restClient(false)
		if err != nil {
			return err
		}

		symbols := upperSymbols(args)
		var resp *api.LiveResponse
		wireJSON := format == output.JSON
		if wireJSON {
			body, err := client.LiveRaw(cmd.Context(), symbols)
			if err != nil {
				return err
			}
			if err := printServerBody(body); err != nil {
				return err
			}
			if restLiveSave == "" {
				return nil
			}
			resp = &api.LiveResponse{}
			if err := jsonUnmarshal(body, resp); err != nil {
				return err
			}
		} else {
			resp, err = client.Live(cmd.Context(), symbols)
			if err != nil {
				return err
			}
		}

		header := []string{"symbol", "bid", "ask", "mid", "time"}
		rows := make([][]string, 0, len(resp.Quotes))
		for _, q := range resp.Quotes {
			rows = append(rows, []string{q.Symbol(),
				output.Price(q.Bid), output.Price(q.Ask), output.Price(q.Mid),
				serverTime(resp.Timestamp, resp.RequestedTime)})
		}

		if restLiveSave != "" {
			if err := saveCSV(restLiveSave, header, rows); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "saved %d rows to %s\n", len(rows), restLiveSave)
			if wireJSON {
				return nil
			}
		}

		switch format {
		case output.CSV:
			return output.WriteCSV(os.Stdout, header, rows)
		}

		w := output.TableWriter(os.Stdout)
		fmt.Fprintln(w, "SYMBOL\tBID\tASK\tMID")
		for _, q := range resp.Quotes {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				q.Symbol(), output.Price(q.Bid), output.Price(q.Ask), output.Price(q.Mid))
		}
		if err := w.Flush(); err != nil {
			return err
		}
		fmt.Printf("\nas of %s\n", serverTime(resp.Timestamp, resp.RequestedTime))
		return nil
	},
}

func init() {
	restLiveCmd.Flags().StringVar(&restLiveSave, "save", "",
		"write CSV to a .csv filename (overwrites)")
	rootCmd.AddCommand(restLiveCmd)
}
