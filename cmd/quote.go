package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tradermade/tradermade-cli/internal/output"
)

var quoteCmd = &cobra.Command{
	Use:   "quote SYMBOL [SYMBOL...]",
	Short: "Get live quotes for one or more symbols",
	Example: `  tradermade quote EURUSD
  tradermade quote EURUSD GBPUSD XAUUSD
  tradermade quote BTCUSD --output json`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, format, err := restClient()
		if err != nil {
			return err
		}

		resp, err := client.Live(cmd.Context(), upperSymbols(args))
		if err != nil {
			return err
		}

		switch format {
		case output.JSON:
			return output.PrintJSON(os.Stdout, resp)
		case output.CSV:
			rows := make([][]string, 0, len(resp.Quotes))
			for _, q := range resp.Quotes {
				rows = append(rows, []string{q.Symbol(),
					output.Price(q.Bid), output.Price(q.Ask), output.Price(q.Mid), resp.RequestedTime})
			}
			return output.WriteCSV(os.Stdout, []string{"symbol", "bid", "ask", "mid", "requested_time"}, rows)
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
		fmt.Printf("\nas of %s\n", resp.RequestedTime)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(quoteCmd)
}
