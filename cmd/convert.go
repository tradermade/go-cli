package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/output"
)

var convertCmd = &cobra.Command{
	Use:     "convert AMOUNT FROM TO",
	Short:   "Currency conversion (REST /api/v1/convert)",
	GroupID: "rest",
	Long: `Convert an amount using:
  GET https://marketdata.tradermade.com/api/v1/convert

API query construction:
  amount   Required. First positional argument; must be numeric.
  from     Required. Second positional argument; uppercased currency code.
  to       Required. Third positional argument; uppercased currency code.
  api_key  Added automatically from the configured REST key.

--output json prints the JSON response exactly as the server sent it.`,
	Example: `  tradermade convert 1000 USD INR
  tradermade convert 250.50 EUR GBP --output json`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		amount, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			return fmt.Errorf("invalid amount %q - expected a number like 1000 or 250.50", args[0])
		}
		from := strings.ToUpper(strings.TrimSpace(args[1]))
		to := strings.ToUpper(strings.TrimSpace(args[2]))

		client, format, err := restClient(false)
		if err != nil {
			return err
		}

		if format == output.JSON {
			body, err := client.ConvertRaw(cmd.Context(), from, to, amount)
			if err != nil {
				return err
			}
			return printServerBody(body)
		}

		resp, err := client.Convert(cmd.Context(), from, to, amount)
		if err != nil {
			return err
		}

		switch format {
		case output.CSV:
			return output.WriteCSV(os.Stdout,
				[]string{"from", "to", "amount", "rate", "total", "time"},
				[][]string{{resp.BaseCurrency, resp.QuoteCurrency, output.Price(amount),
					output.Price(resp.Quote), output.Price(resp.Total),
					serverTime(resp.Timestamp, resp.RequestedTime)}})
		}

		fmt.Printf("%s %s = %s %s\n", output.Price(amount), resp.BaseCurrency,
			output.Price(resp.Total), resp.QuoteCurrency)
		fmt.Printf("rate  1 %s = %s %s\n", resp.BaseCurrency, output.Price(resp.Quote), resp.QuoteCurrency)
		fmt.Printf("as of %s\n", serverTime(resp.Timestamp, resp.RequestedTime))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)
}
