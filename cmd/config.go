package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/config"
)

var (
	setKeyRest string
	setKeyWS   string
)

var configCmd = &cobra.Command{
	Use:     "config",
	Short:   "Manage the saved API keys and config file",
	GroupID: "local",
}

var configSetKeyCmd = &cobra.Command{
	Use:   "set-key --rest KEY | --ws KEY",
	Short: "Save your TraderMade API keys",
	Long: `Save your TraderMade API keys: --rest for the REST key, --ws for the
WebSocket streaming key. One at a time or both together:

  tradermade config set-key --rest YOUR_REST_KEY
  tradermade config set-key --ws   YOUR_WS_KEY
  tradermade config set-key --rest YOUR_REST_KEY --ws YOUR_WS_KEY

Keys that start with a dash work as-is.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		rest := strings.TrimSpace(setKeyRest)
		ws := strings.TrimSpace(setKeyWS)
		if rest == "" && ws == "" {
			return fmt.Errorf("nothing to save - pass --rest KEY and/or --ws KEY")
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		var saved []string
		if rest != "" {
			cfg.RESTKey = rest
			saved = append(saved, "REST key")
		}
		if ws != "" {
			cfg.WSKey = ws
			saved = append(saved, "WebSocket key")
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		path, _ := config.Path()
		fmt.Printf("%s saved to %s\n", strings.Join(saved, " and "), path)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the active keys (masked) and where they come from",
	RunE: func(cmd *cobra.Command, args []string) error {
		show := func(side string, resolve func() (string, string, error)) {
			key, source, err := resolve()
			if err != nil {
				fmt.Printf("%-7s not configured\n", side)
				return
			}
			fmt.Printf("%-7s %s  (from %s)\n", side, config.MaskKey(key), source)
		}
		show("rest", config.ResolveRESTKeySource)
		show("stream", config.ResolveWSKeySource)
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the config file location",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.Path()
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	},
}

func init() {
	configSetKeyCmd.Flags().StringVar(&setKeyRest, "rest", "", "the REST API key")
	configSetKeyCmd.Flags().StringVar(&setKeyWS, "ws", "", "the WebSocket streaming key")
	configCmd.AddCommand(configSetKeyCmd, configShowCmd, configPathCmd)
	rootCmd.AddCommand(configCmd)
}
