package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/config"
	"github.com/tradermade/go-cli/internal/output"
	"github.com/tradermade/go-cli/internal/stream"
)

var liveCmd = &cobra.Command{
	Use:   "live SYMBOL [SYMBOL...]",
	Short: "Stream live tick data over WebSocket (Ctrl+C to stop)",
	Long: `Stream live tick data over WebSocket. Reconnects and resubscribes
automatically if the connection drops. Press Ctrl+C to stop.

With --output json each tick is printed as one JSON line (NDJSON),
ready to pipe into jq or a file.`,
	Example: `  tradermade live EURUSD
  tradermade live EURUSD GBPUSD XAUUSD
  tradermade live BTCUSD --output json > ticks.ndjson`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := outputFormat()
		if err != nil {
			return err
		}
		key, err := config.ResolveWSKey()
		if err != nil {
			return err
		}

		// Direction arrows need the previous bid per symbol.
		lastBid := map[string]float64{}
		headerPrinted := false

		opts := stream.Options{
			Key:     key,
			Symbols: args,
			// Lifecycle messages go to stderr so stdout stays clean for piping.
			Logf: func(f string, a ...any) {
				fmt.Fprintf(os.Stderr, f+"\n", a...)
			},
			OnTick: func(t stream.Tick, raw []byte) {
				switch format {
				case output.JSON:
					fmt.Println(string(raw))
					return
				case output.CSV:
					if !headerPrinted {
						fmt.Println("time,symbol,bid,ask,bid_volume,ask_volume")
						headerPrinted = true
					}
					fmt.Printf("%s,%s,%s,%s,%s,%s\n",
						t.Timestamp, t.Symbol, t.Bid, t.Ask, t.BidVolume, t.AskVolume)
					return
				}
				if !headerPrinted {
					fmt.Printf("%-22s %-10s %14s %14s\n", "TIME", "SYMBOL", "BID", "ASK")
					headerPrinted = true
				}
				dir := " "
				if bid, err := strconv.ParseFloat(t.Bid, 64); err == nil {
					if prev, ok := lastBid[t.Symbol]; ok {
						switch {
						case bid > prev:
							dir = "↑"
						case bid < prev:
							dir = "↓"
						}
					}
					lastBid[t.Symbol] = bid
				}
				fmt.Printf("%-22s %-10s %14s %14s %s\n", t.Timestamp, t.Symbol, t.Bid, t.Ask, dir)
			},
		}

		if err := stream.Run(cmd.Context(), opts); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "stopped")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(liveCmd)
}
