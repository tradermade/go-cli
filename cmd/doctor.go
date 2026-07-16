package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/api"
	"github.com/tradermade/go-cli/internal/config"
	"github.com/tradermade/go-cli/internal/output"
	"github.com/tradermade/go-cli/internal/stream"
)

// check is one doctor line - also the JSON output shape.
type check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Short:   "Connectivity check (REST /live; WebSocket /feedAdv)",
	GroupID: "combined",
	Long: `Runs four independent checks and reports each:

  key      is an API key configured, and where does it come from
  rest     can the REST API be reached, and does the key work there
  stream   can the WebSocket be reached, does the key work there, plan limits
  config   is the config file present and valid

Exit code is 0 when everything passes, 1 otherwise - usable as a CI smoke
test. Support can ask customers to paste the output of
"tradermade doctor --output json".

Probe construction:
  REST /api/v1/live  currency=EURUSD; api_key is the configured REST key.
  WebSocket /feedAdv login action with key and fmt=JSON; no market-data
                      subscription is required for the capability check.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := outputFormat()
		if err != nil {
			return err
		}
		if format != output.Table && format != output.JSON {
			return fmt.Errorf("doctor uses table by default and only accepts --output json")
		}
		ctx := cmd.Context()
		var checks []check

		// 1+2. REST key present, and does it work against /live?
		restKey, restSource, restKeyErr := config.ResolveRESTKeySource()
		if restKeyErr != nil {
			checks = append(checks, check{"rest-key", false, "not configured - `tradermade config set-key --rest YOUR_KEY` or set " + config.EnvRESTKey})
			checks = append(checks, check{"rest", false, "skipped - no key"})
		} else {
			checks = append(checks, check{"rest-key", true, fmt.Sprintf("%s (from %s)", config.MaskKey(restKey), restSource)})
			if _, err := api.New(restKey).Live(ctx, []string{"EURUSD"}); err != nil {
				checks = append(checks, check{"rest", false, oneLine(err.Error())})
			} else {
				checks = append(checks, check{"rest", true, "live quote ok"})
			}
		}

		// 3+4. WebSocket key present, and does it log in?
		wsKey, wsSource, wsKeyErr := config.ResolveWSKeySource()
		if wsKeyErr != nil {
			checks = append(checks, check{"ws-key", false, "not configured - `tradermade config set-key --ws YOUR_KEY` or set " + config.EnvWSKey})
			checks = append(checks, check{"stream", false, "skipped - no key"})
		} else {
			checks = append(checks, check{"ws-key", true, fmt.Sprintf("%s (from %s)", config.MaskKey(wsKey), wsSource)})
			plan, _, err := stream.Probe(ctx, "", wsKey)
			if err != nil {
				checks = append(checks, check{"stream", false, oneLine(err.Error())})
			} else {
				detail := fmt.Sprintf("login ok - plan allows %d symbols", plan.SymbolLimit)
				if plan.CFDs {
					detail += ", CFDs enabled"
				}
				if plan.TraderLadder {
					detail += ", trader ladder enabled"
				}
				checks = append(checks, check{"stream", true, detail})
			}
		}

		// 4. Config file health.
		path, _ := config.Path()
		if data, err := os.ReadFile(path); err != nil {
			if os.IsNotExist(err) {
				checks = append(checks, check{"config", true, "no config file (env-only setup) - " + path})
			} else {
				checks = append(checks, check{"config", false, err.Error()})
			}
		} else if !json.Valid(data) {
			checks = append(checks, check{"config", false, "config file is not valid JSON: " + path})
		} else {
			checks = append(checks, check{"config", true, path})
		}

		allOK := true
		for _, c := range checks {
			if !c.OK {
				allOK = false
			}
		}

		if format == output.JSON {
			if err := output.PrintJSON(os.Stdout, checks); err != nil {
				return err
			}
		} else {
			w := output.TableWriter(os.Stdout)
			for _, c := range checks {
				status := "ok"
				if !c.OK {
					status = "FAIL"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", c.Name, status, c.Detail)
			}
			if err := w.Flush(); err != nil {
				return err
			}
		}

		if !allOK {
			// Non-zero exit without repeating the details as an error message.
			os.Exit(1)
		}
		return nil
	},
}

// oneLine flattens multi-line error messages for the aligned table.
func oneLine(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\n", " ")), " ")
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
