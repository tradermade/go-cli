package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/api"
	"github.com/tradermade/go-cli/internal/output"
)

var (
	symbolsMarket string
)

var symbolsCmd = &cobra.Command{
	Use:     "symbols",
	Short:   "Codes (REST /api/v1/live_currencies_list or /api/v1/live_crypto_list)",
	GroupID: "rest",
	Long: `List codes supported by live endpoints.

Endpoint selection:
  --market forex   GET /api/v1/live_currencies_list (default)
  --market crypto  GET /api/v1/live_crypto_list

API query construction:
  api_key  Added automatically from the configured REST key.

The command returns the complete list from the selected endpoint.`,
	Example: `  tradermade symbols
  tradermade symbols --market crypto
  tradermade symbols --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, format, err := restClient(false)
		if err != nil {
			return err
		}

		crypto := false
		switch symbolsMarket {
		case "forex":
		case "crypto":
			crypto = true
		default:
			return fmt.Errorf("invalid --market %q - use forex or crypto", symbolsMarket)
		}
		if format == output.JSON {
			body, err := client.SymbolListRaw(cmd.Context(), crypto)
			if err != nil {
				return err
			}
			return printServerBody(body)
		}

		var resp *api.SymbolsResponse
		if crypto {
			resp, err = client.LiveCryptoList(cmd.Context())
		} else {
			resp, err = client.LiveCurrenciesList(cmd.Context())
		}
		if err != nil {
			return err
		}

		codes := make([]string, 0, len(resp.AvailableCurrencies))
		for code := range resp.AvailableCurrencies {
			codes = append(codes, code)
		}
		sort.Strings(codes)

		if format == output.CSV {
			rows := make([][]string, 0, len(codes))
			for _, code := range codes {
				rows = append(rows, []string{code, resp.AvailableCurrencies[code]})
			}
			return output.WriteCSV(os.Stdout, []string{"code", "name"}, rows)
		}

		w := output.TableWriter(os.Stdout)
		for _, code := range codes {
			fmt.Fprintf(w, "%s\t%s\n", code, resp.AvailableCurrencies[code])
		}
		if err := w.Flush(); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "\n%d codes\n", len(codes))
		return nil
	},
}

func init() {
	symbolsCmd.Flags().StringVar(&symbolsMarket, "market", "forex", "asset class: forex or crypto")
	rootCmd.AddCommand(symbolsCmd)
}
