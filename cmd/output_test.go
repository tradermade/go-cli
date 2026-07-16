package cmd

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/output"
)

func TestRawOutput(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("output", "table", "")

	raw, err := rawOutput(cmd, output.Raw, false)
	if err != nil || !raw {
		t.Fatalf("--output raw: raw=%v err=%v", raw, err)
	}

	raw, err = rawOutput(cmd, output.Table, true)
	if err != nil || !raw {
		t.Fatalf("legacy --raw: raw=%v err=%v", raw, err)
	}

	if err := cmd.Flags().Set("output", "csv"); err != nil {
		t.Fatal(err)
	}
	if _, err := rawOutput(cmd, output.CSV, true); err == nil {
		t.Fatal("expected conflicting --raw --output csv to fail")
	}
}
