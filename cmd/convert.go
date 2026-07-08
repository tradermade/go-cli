package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tradermade/tradermade-cli/internal/output"
)

var convertCmd = &cobra.Command{
	Use:   "convert AMOUNT FROM TO",
	Short: "Convert an amount between currencies at the live rate",
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

		client, format, err := restClient()
		if err != nil {
			return err
		}

		resp, err := client.Convert(cmd.Context(), from, to, amount)
		if err != nil {
			return err
		}

		switch format {
		case output.JSON:
			return output.PrintJSON(os.Stdout, resp)
		case output.CSV:
			return output.WriteCSV(os.Stdout,
				[]string{"from", "to", "amount", "rate", "total", "requested_time"},
				[][]string{{resp.BaseCurrency, resp.QuoteCurrency, output.Price(amount),
					output.Price(resp.Quote), output.Price(resp.Total), resp.RequestedTime}})
		}

		fmt.Printf("%s %s = %s %s\n", output.Price(amount), resp.BaseCurrency,
			output.Price(resp.Total), resp.QuoteCurrency)
		fmt.Printf("rate  1 %s = %s %s\n", resp.BaseCurrency, output.Price(resp.Quote), resp.QuoteCurrency)
		fmt.Printf("as of %s\n", resp.RequestedTime)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)
}
