package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/config"
)

var (
	setKeyRest bool
	setKeyWS   bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage the saved API keys and config file",
}

var configSetKeyCmd = &cobra.Command{
	Use:   "set-key KEY",
	Short: "Save a TraderMade API key",
	Long: `Save a TraderMade API key.

Some plans issue separately-scoped keys: one for REST, one for WebSocket
streaming. Save each with its flag:

  tradermade config set-key --rest YOUR_REST_KEY
  tradermade config set-key --ws   YOUR_WS_KEY

If one key covers both APIs, save it without flags and every command uses it.

If your key starts with a dash, put -- before it so it is not read as a flag:

  tradermade config set-key --rest -- -YOUR_KEY`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := strings.TrimSpace(args[0])
		if key == "" {
			return fmt.Errorf("key is empty")
		}
		if setKeyRest && setKeyWS {
			return fmt.Errorf("pass --rest or --ws, not both - run set-key twice for two different keys")
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		var label string
		switch {
		case setKeyRest:
			cfg.RESTKey, label = key, "REST key"
		case setKeyWS:
			cfg.WSKey, label = key, "WebSocket key"
		default:
			cfg.APIKey, label = key, "API key (used for both REST and WebSocket)"
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		path, _ := config.Path()
		fmt.Printf("%s saved to %s\n", label, path)
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
	configSetKeyCmd.Flags().BoolVar(&setKeyRest, "rest", false, "save as the REST-only key")
	configSetKeyCmd.Flags().BoolVar(&setKeyWS, "ws", false, "save as the WebSocket-only key")
	configCmd.AddCommand(configSetKeyCmd, configShowCmd, configPathCmd)
	rootCmd.AddCommand(configCmd)
}
