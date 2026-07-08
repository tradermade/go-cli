package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/api"
	"github.com/tradermade/go-cli/internal/output"
)

var (
	symbolsMarket string
	symbolsGrep   string
)

var symbolsCmd = &cobra.Command{
	Use:   "symbols",
	Short: "List available currency or crypto codes",
	Example: `  tradermade symbols
  tradermade symbols --grep GBP
  tradermade symbols --market crypto --grep BTC`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, format, err := restClient()
		if err != nil {
			return err
		}

		var resp *api.SymbolsResponse
		switch symbolsMarket {
		case "forex":
			resp, err = client.LiveCurrenciesList(cmd.Context())
		case "crypto":
			resp, err = client.LiveCryptoList(cmd.Context())
		default:
			return fmt.Errorf("invalid --market %q - use forex or crypto", symbolsMarket)
		}
		if err != nil {
			return err
		}

		// Filter client-side, then sort for stable output.
		grep := strings.ToUpper(strings.TrimSpace(symbolsGrep))
		filtered := make(map[string]string)
		for code, name := range resp.AvailableCurrencies {
			if grep == "" ||
				strings.Contains(strings.ToUpper(code), grep) ||
				strings.Contains(strings.ToUpper(name), grep) {
				filtered[code] = name
			}
		}

		if format == output.JSON {
			return output.PrintJSON(os.Stdout, filtered)
		}

		codes := make([]string, 0, len(filtered))
		for code := range filtered {
			codes = append(codes, code)
		}
		sort.Strings(codes)

		if format == output.CSV {
			rows := make([][]string, 0, len(codes))
			for _, code := range codes {
				rows = append(rows, []string{code, filtered[code]})
			}
			return output.WriteCSV(os.Stdout, []string{"code", "name"}, rows)
		}

		w := output.TableWriter(os.Stdout)
		for _, code := range codes {
			fmt.Fprintf(w, "%s\t%s\n", code, filtered[code])
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
	symbolsCmd.Flags().StringVar(&symbolsGrep, "grep", "", "filter by code or name (case-insensitive)")
	rootCmd.AddCommand(symbolsCmd)
}
