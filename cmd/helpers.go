package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tradermade/go-cli/internal/api"
	"github.com/tradermade/go-cli/internal/config"
	"github.com/tradermade/go-cli/internal/output"
)

// jsonUnmarshal wraps encoding/json with a friendlier error.
func jsonUnmarshal(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("cannot parse API response: %w", err)
	}
	return nil
}

// printServerBody preserves the server payload and only adds a terminal
// newline when the response did not include one.
func printServerBody(data []byte) error {
	if _, err := os.Stdout.Write(data); err != nil {
		return err
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		_, err := fmt.Fprintln(os.Stdout)
		return err
	}
	return nil
}

// restClient parses the --output flag and resolves the REST key.
func restClient(allowRaw bool) (*api.Client, output.Format, error) {
	format, err := outputFormat()
	if err != nil {
		return nil, "", err
	}
	if format == output.Raw && !allowRaw {
		return nil, "", fmt.Errorf("--output raw is not supported by this command")
	}
	key, err := config.ResolveRESTKey()
	if err != nil {
		return nil, "", err
	}
	return api.New(key), format, nil
}

// rawOutput resolves --output raw and the deprecated --raw compatibility
// alias. An explicitly selected structured format must not be silently ignored.
func rawOutput(cmd *cobra.Command, format output.Format, legacy bool) (bool, error) {
	if !legacy {
		return format == output.Raw, nil
	}
	if flag := cmd.Flag("output"); flag != nil && flag.Changed && format != output.Raw && format != output.JSON {
		return false, fmt.Errorf("--raw cannot be combined with --output %s; use --output json", format)
	}
	return true, nil
}

// serverTime prefers the server's quote timestamp (when the price was set)
// over the request-processing time, which can lag behind it.
func serverTime(ts int64, fallback string) string {
	if ts == 0 {
		return fallback
	}
	return output.UnixUTC(ts)
}

// resolveSavePath requires a CSV filename. Bare filenames are resolved against
// the process working directory; directory-only targets are rejected.
func resolveSavePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("--save needs a .csv filename, e.g. --save data.csv")
	}

	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return "", fmt.Errorf("--save target %q is a directory; include a .csv filename", path)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("cannot access --save target %q: %w", path, err)
	}
	if !strings.EqualFold(filepath.Ext(path), ".csv") {
		return "", fmt.Errorf("--save target %q must include a .csv filename", path)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve --save target %q: %w", path, err)
	}
	if err := checkSavePath(abs); err != nil {
		return "", err
	}
	return abs, nil
}

// checkSavePath validates a --save target before any network work, so a
// bad path fails instantly with a clear message instead of after a fetch.
// A bare filename is fine - it saves into the current directory.
func checkSavePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("--save needs a .csv filename, e.g. --save data.csv")
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return fmt.Errorf("cannot save to %q: that is a directory - give a file name, e.g. %s", path, filepath.Join(path, "data.csv"))
	}
	dir := filepath.Dir(path)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return fmt.Errorf("cannot save to %q: directory %q does not exist - create it first or check the path", path, dir)
	}
	return nil
}

// saveCSV writes header+rows to path, overwriting any existing file.
func saveCSV(path string, header []string, rows [][]string) error {
	if err := checkSavePath(path); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create %q: %w", path, err)
	}
	w := csv.NewWriter(f)
	werr := w.Write(header)
	for _, r := range rows {
		if werr == nil {
			werr = w.Write(r)
		}
	}
	w.Flush()
	if werr == nil {
		werr = w.Error()
	}
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	return werr
}

// openCSVAppend opens path for appending CSV rows (used by stream capture -
// restarting a capture continues the file rather than wiping it). Reports
// whether the file is new/empty so the caller writes the header only once.
func openCSVAppend(path string) (f *os.File, w *csv.Writer, needHeader bool, err error) {
	if err := checkSavePath(path); err != nil {
		return nil, nil, false, err
	}
	f, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, false, fmt.Errorf("cannot open %q: %w", path, err)
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, false, err
	}
	return f, csv.NewWriter(f), st.Size() == 0, nil
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
