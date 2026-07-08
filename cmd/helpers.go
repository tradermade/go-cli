package cmd

import (
	"strings"

	"github.com/tradermade/tradermade-cli/internal/api"
	"github.com/tradermade/tradermade-cli/internal/config"
	"github.com/tradermade/tradermade-cli/internal/output"
)

// restClient parses the --output flag and resolves the REST key.
func restClient() (*api.Client, output.Format, error) {
	format, err := outputFormat()
	if err != nil {
		return nil, "", err
	}
	key, err := config.ResolveRESTKey()
	if err != nil {
		return nil, "", err
	}
	return api.New(key), format, nil
}

// upperSymbols trims and uppercases symbol args, dropping blanks.
func upperSymbols(args []string) []string {
	out := make([]string, 0, len(args))
	for _, s := range args {
		s = strings.ToUpper(strings.TrimSpace(s))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
