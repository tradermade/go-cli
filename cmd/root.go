// Package cmd wires the CLI commands. Command logic stays thin here;
// API and streaming behavior lives in internal/.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

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
  tradermade config set-key --rest YOUR_REST_KEY --ws YOUR_WS_KEY
  tradermade quote EURUSD GBPUSD
  tradermade convert 1000 USD INR
  tradermade live EURUSD GBPUSD

API keys from https://tradermade.com/signup are required. Environment
variables override saved keys: TRADERMADE_REST_API_KEY and
TRADERMADE_WS_API_KEY (or TRADERMADE_API_KEY for both).

The command list below shows every supported REST and WebSocket endpoint. Run
"tradermade COMMAND --help" for its arguments, flags, and examples.`,
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
	rootCmd.AddGroup(
		&cobra.Group{ID: "rest", Title: "REST API:"},
		&cobra.Group{ID: "websocket", Title: "WebSocket API:"},
		&cobra.Group{ID: "combined", Title: "REST + WebSocket:"},
		&cobra.Group{ID: "local", Title: "Local commands (no endpoint):"},
	)
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", string(output.Table),
		"select json or csv; live also supports raw (omit for table)")
	rootCmd.PersistentFlags().Bool("help", false, "show help for this command")
	rootCmd.PersistentFlags().BoolFuncP("short-help-disabled", "h", "", func(string) error {
		return fmt.Errorf("-h is not supported; use --help")
	})
	_ = rootCmd.PersistentFlags().MarkHidden("short-help-disabled")
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		if strings.Contains(err.Error(), "short-help-disabled") {
			return fmt.Errorf("-h is not supported; use --help")
		}
		return err
	})
	// Suppress Cobra's "help" subcommand. Help is exposed only through the
	// long --help flag; the hidden alternate name prevents default creation.
	rootCmd.SetHelpCommand(&cobra.Command{Use: "__help", Hidden: true})
	rootCmd.Version = Version
	api.UserAgent = "tradermade-cli/" + Version
}

// outputFormat parses the --output flag, failing fast on invalid values.
func outputFormat() (output.Format, error) {
	format, err := output.ParseFormat(outputFlag)
	if err != nil {
		return "", err
	}
	flag := rootCmd.PersistentFlags().Lookup("output")
	if err := validateOutputSelection(format, flag != nil && flag.Changed); err != nil {
		return "", err
	}
	return format, nil
}

func validateOutputSelection(format output.Format, explicitlySet bool) error {
	if format == output.Table && explicitlySet {
		return fmt.Errorf("table is already the default; omit --output table")
	}
	return nil
}
