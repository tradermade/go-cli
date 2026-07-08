// Package cmd wires the CLI commands. Command logic stays thin here;
// API and streaming behavior lives in internal/.
package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/api"
	"github.com/tradermade/go-cli/internal/output"
)

// Build metadata, stamped at release time via:
//
//	go build -ldflags "-X github.com/tradermade/go-cli/cmd.Version=v1.2.3 \
//	                   -X github.com/tradermade/go-cli/cmd.Commit=abc1234 \
//	                   -X github.com/tradermade/go-cli/cmd.Date=2026-07-07"
var (
	Version = "0.1.0-dev"
	Commit  = ""
	Date    = ""
)

var outputFlag string

var rootCmd = &cobra.Command{
	Use:     "tradermade",
	Short:   "TraderMade market data from your terminal",
	Version: Version,
	Long: `tradermade - live, historical, and streaming FX / CFD / crypto data
from the TraderMade API (https://tradermade.com), in your terminal.

Get started:
  tradermade config set-key YOUR_API_KEY
  tradermade quote EURUSD GBPUSD
  tradermade convert 1000 USD INR
  tradermade live EURUSD GBPUSD

An API key from https://tradermade.com/signup is required. The key is read
from the TRADERMADE_API_KEY environment variable first, then from the saved
config file. Plans with separate REST and WebSocket keys: see
"tradermade config set-key --help".`,
	SilenceUsage:  true,
	SilenceErrors: false,
}

// Execute runs the CLI. First Ctrl+C cancels the command context; a second
// one force-kills.
func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", string(output.Table),
		"output format: table, json, or csv")
	rootCmd.Version = Version
	api.UserAgent = "tradermade-cli/" + Version
}

// outputFormat parses the --output flag, failing fast on invalid values.
func outputFormat() (output.Format, error) {
	return output.ParseFormat(outputFlag)
}
