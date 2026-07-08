package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and build information",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("tradermade %s\n", Version)
		commit, date := Commit, Date
		if commit == "" {
			commit = "none"
		}
		if date == "" {
			date = "unknown"
		}
		fmt.Printf("commit %s, built %s, %s %s/%s\n",
			commit, date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
