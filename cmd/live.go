package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/config"
	"github.com/tradermade/go-cli/internal/output"
	"github.com/tradermade/go-cli/internal/stream"
)

var (
	streamLadder   bool
	streamSendLast bool
	streamRaw      bool
	streamSave     string
)

var streamCmd = &cobra.Command{
	Use:     "stream SYMBOL [SYMBOL...]",
	Short:   "Live ticks (WebSocket wss://stream.tradermade.com/feedAdv)",
	GroupID: "websocket",
	Long: `Stream ticks from wss://stream.tradermade.com/feedAdv (WebSocket v2).

Protocol construction:
  login.action       Always "login".
  login.key          Added from the configured WebSocket key.
  login.fmt          Always "JSON". CSV table/file output is converted locally.
  login.send_ladder  --ladder adds true; requires trader-ladder plan access.
  subscribe.action   Always "subscribe" after login_ok.
  subscribe.symbols  Positional symbols normalized to SYMBOL:QUOTE.
  subscribe.send_last  --send-last adds true for one cached LAST_QUOTE.

The server supports JSON/CSV/SSV market frames, but control frames are always
JSON. This CLI requests JSON so reconnect, acknowledgements, ladder data, and
CSV saving share one parser. --output json prints original market tick frames
as NDJSON; --output raw also includes greeting and control frames.

The client reconnects with backoff and resubscribes because subscriptions do
not persist. --save requires a .csv filename and appends on restart. A bare
filename uses the current working directory; the absolute path is reported
when the first tick is saved. Press Ctrl+C to stop.`,
	Example: `  tradermade stream EURUSD
  tradermade stream EURUSD GBPUSD XAUUSD
  tradermade stream EURUSD --ladder
  tradermade stream EURUSD --output raw
  tradermade stream EURUSD --save ticks.csv
  tradermade stream BTCUSD --output json > ticks.ndjson`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()
		format, err := outputFormat()
		if err != nil {
			return err
		}
		rawOutput, err := rawOutput(cmd, format, streamRaw)
		if err != nil {
			return err
		}
		key, err := config.ResolveWSKey()
		if err != nil {
			return err
		}

		// Open the capture file before connecting, so a bad path fails
		// instantly instead of after the stream is already up.
		var csvW *csv.Writer
		saved := 0
		var saveErr error
		if streamSave != "" {
			streamSave, err = resolveSavePath(streamSave)
			if err != nil {
				return err
			}
			f, w, needHeader, err := openCSVAppend(streamSave)
			if err != nil {
				return err
			}
			defer f.Close()
			if needHeader {
				if err := w.Write([]string{"time", "symbol", "bid", "ask", "bid_volume", "ask_volume"}); err != nil {
					return err
				}
				w.Flush()
				if err := w.Error(); err != nil {
					return err
				}
			}
			csvW = w
		}

		// Direction arrows need the previous bid per symbol.
		lastBid := map[string]float64{}
		headerPrinted := false

		opts := stream.Options{
			Key:        key,
			Symbols:    args,
			SendLadder: streamLadder,
			SendLast:   streamSendLast,
			// Lifecycle messages go to stderr so stdout stays clean for piping.
			Logf: func(f string, a ...any) {
				fmt.Fprintf(os.Stderr, f+"\n", a...)
			},
			OnTick: func(t stream.Tick, raw []byte) {
				if csvW != nil && saveErr == nil {
					if err := csvW.Write([]string{t.Timestamp, t.Symbol, t.Bid, t.Ask, t.BidVolume, t.AskVolume}); err != nil {
						saveErr = fmt.Errorf("cannot save CSV to %s: %w", streamSave, err)
						cancel()
						return
					}
					csvW.Flush()
					if err := csvW.Error(); err != nil {
						saveErr = fmt.Errorf("cannot save CSV to %s: %w", streamSave, err)
						cancel()
						return
					}
					saved++
					if saved == 1 {
						fmt.Fprintf(os.Stderr, "saving CSV to %s\n", streamSave)
					}
				}
				if rawOutput {
					return // display is handled frame-by-frame via OnRaw
				}
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
					fmt.Printf("%-22s %-10s %14s %14s %10s %10s\n", "TIME", "SYMBOL", "BID", "ASK", "BID-VOL", "ASK-VOL")
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
				bid, ask := t.Bid, t.Ask
				if t.Ladder != nil {
					// Ladder frames carry prices as floats and can arrive with
					// artifacts like 1.1425399999999999 - shorten for display.
					bid, ask = shortNum(bid), shortNum(ask)
				}
				bv, av := t.BidVolume, t.AskVolume
				if bv == "" {
					bv = "-" // LAST_QUOTE messages carry no volumes
				}
				if av == "" {
					av = "-"
				}
				fmt.Printf("%-22s %-10s %14s %14s %10s %10s %s\n",
					t.Timestamp, t.Symbol, bid, ask, bv, av, dir)
				if t.Ladder != nil {
					fmt.Printf("%22s %-8s %s\n", "", "  bids", ladderLevels(t.Ladder.Bids))
					fmt.Printf("%22s %-8s %s\n", "", "  asks", ladderLevels(t.Ladder.Asks))
				}
			},
		}

		if rawOutput {
			// Every frame exactly as received - control messages included.
			opts.OnRaw = func(raw []byte) {
				fmt.Println(string(raw))
			}
		}

		if err := stream.Run(ctx, opts); err != nil {
			return err
		}
		if saveErr != nil {
			return saveErr
		}
		if streamSave != "" {
			fmt.Fprintf(os.Stderr, "saved %d ticks to %s\n", saved, streamSave)
		}
		fmt.Fprintln(os.Stderr, "stopped")
		return nil
	},
}

// ladderLevels renders up to five depth levels as "price x volume" pairs.
func ladderLevels(levels [][]string) string {
	const max = 5
	parts := make([]string, 0, max)
	for i, l := range levels {
		if i == max {
			break
		}
		if len(l) >= 2 {
			parts = append(parts, l[0]+" x "+l[1])
		}
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, "   ")
}

// shortNum rewrites a numeric string in its shortest round-trip form,
// removing float artifacts (1.1425399999999999 -> 1.14254).
func shortNum(s string) string {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func init() {
	streamCmd.Flags().BoolVar(&streamLadder, "ladder", false,
		"request market depth (trader ladder plans only)")
	streamCmd.Flags().BoolVar(&streamSendLast, "send-last", false,
		"receive the cached last tick immediately on subscribe")
	streamCmd.Flags().BoolVar(&streamRaw, "raw", false,
		"deprecated alias for --output raw")
	_ = streamCmd.Flags().MarkDeprecated("raw", "use --output raw instead")
	streamCmd.Flags().StringVar(&streamSave, "save", "",
		"append CSV to a .csv filename")
	rootCmd.AddCommand(streamCmd)
}
