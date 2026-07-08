package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tradermade/tradermade-cli/internal/board"
	"github.com/tradermade/tradermade-cli/internal/config"
	"github.com/tradermade/tradermade-cli/internal/watchlist"
)

var boardSort string

var boardCmd = &cobra.Command{
	Use:   "board [SYMBOL...]",
	Short: "Live watchlist dashboard in the terminal",
	Long: `A full-screen dashboard where your watchlist symbols update in place
as ticks arrive: bid/ask, spread, day change, and green/red flashes
on movement.

The DAY% column is change vs the previous daily close (fetched via REST at
startup); without a REST key it falls back to change since the session began.

Keys:
  q  quit
  s  cycle sort (list / symbol / change)
  c  rest-check: fetch a REST /live snapshot and show each symbol's REST
     mid next to the stream price, with the deviation

With no arguments the saved watchlist is used (manage it with
"board add", "board remove", "board list"). Passing symbols uses those
for this run without touching the saved list.`,
	Example: `  tradermade board add EURUSD GBPUSD XAUUSD
  tradermade board
  tradermade board BTCUSD ETHUSD    # one-off board, ignores the watchlist
  tradermade board --sort change    # biggest mover first`,
	RunE: func(cmd *cobra.Command, args []string) error {
		symbols := watchlist.Normalize(args)
		if len(symbols) == 0 {
			saved, err := watchlist.Load()
			if err != nil {
				return err
			}
			symbols = saved
		}
		if len(symbols) == 0 {
			return fmt.Errorf("watchlist is empty - add symbols with `tradermade board add EURUSD GBPUSD` or pass them directly: `tradermade board EURUSD`")
		}

		key, err := config.ResolveWSKey()
		if err != nil {
			return err
		}
		// REST key is optional: with it the Δ column is day change vs previous
		// close; without it the board falls back to session change.
		restKey, _ := config.ResolveRESTKey()

		switch boardSort {
		case board.SortList, board.SortSymbol, board.SortChange:
		default:
			return fmt.Errorf("invalid --sort %q - use list, symbol, or change", boardSort)
		}

		return board.Run(cmd.Context(), board.Options{
			Key: key, RESTKey: restKey, Symbols: symbols, Sort: boardSort,
		})
	},
}

var boardAddCmd = &cobra.Command{
	Use:   "add SYMBOL [SYMBOL...]",
	Short: "Add symbols to the saved watchlist",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		added, err := watchlist.Add(args)
		if err != nil {
			return err
		}
		if len(added) == 0 {
			fmt.Println("already on the watchlist")
			return nil
		}
		fmt.Printf("added: %s\n", strings.Join(added, ", "))
		return nil
	},
}

var boardRemoveCmd = &cobra.Command{
	Use:   "remove SYMBOL [SYMBOL...]",
	Short: "Remove symbols from the saved watchlist",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		removed, err := watchlist.Remove(args)
		if err != nil {
			return err
		}
		if len(removed) == 0 {
			fmt.Println("nothing matched the watchlist")
			return nil
		}
		fmt.Printf("removed: %s\n", strings.Join(removed, ", "))
		return nil
	},
}

var boardListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show the saved watchlist",
	RunE: func(cmd *cobra.Command, args []string) error {
		symbols, err := watchlist.Load()
		if err != nil {
			return err
		}
		if len(symbols) == 0 {
			path, _ := watchlist.Path()
			fmt.Printf("watchlist is empty (%s)\n", path)
			return nil
		}
		for _, s := range symbols {
			fmt.Println(s)
		}
		return nil
	},
}

func init() {
	boardCmd.Flags().StringVar(&boardSort, "sort", board.SortList,
		"row order: list (watchlist order), symbol (alphabetical), or change (biggest mover first)")
	boardCmd.AddCommand(boardAddCmd, boardRemoveCmd, boardListCmd)
	rootCmd.AddCommand(boardCmd)
}
