// Package watchlist stores the board symbols as a plain text file,
// one symbol per line, # comments allowed.
package watchlist

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Path returns the watchlist file location, e.g.
// %AppData%\tradermade\watchlist on Windows.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot locate user config directory: %w", err)
	}
	return filepath.Join(dir, "tradermade", "watchlist"), nil
}

// Load reads the watchlist. A missing file returns an empty list, not an error.
// Symbols are trimmed, uppercased, and deduplicated; blank lines and
// #-comments are skipped.
func Load() ([]string, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot read watchlist %s: %w", path, err)
	}
	return Normalize(strings.Split(string(data), "\n")), nil
}

// Save writes the symbols back, creating the directory if needed.
func Save(symbols []string) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}
	content := strings.Join(symbols, "\n")
	if content != "" {
		content += "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("cannot write watchlist %s: %w", path, err)
	}
	return nil
}

// Add appends symbols not already present and saves. Returns what was added.
func Add(symbols []string) ([]string, error) {
	current, err := Load()
	if err != nil {
		return nil, err
	}
	var added []string
	for _, s := range Normalize(symbols) {
		if !slices.Contains(current, s) {
			current = append(current, s)
			added = append(added, s)
		}
	}
	if len(added) == 0 {
		return nil, nil
	}
	return added, Save(current)
}

// Remove deletes symbols and saves. Returns what was actually removed.
func Remove(symbols []string) ([]string, error) {
	current, err := Load()
	if err != nil {
		return nil, err
	}
	drop := Normalize(symbols)
	var kept, removed []string
	for _, s := range current {
		if slices.Contains(drop, s) {
			removed = append(removed, s)
		} else {
			kept = append(kept, s)
		}
	}
	if len(removed) == 0 {
		return nil, nil
	}
	return removed, Save(kept)
}

// Normalize trims, uppercases, drops blanks/comments, and deduplicates
// while preserving order.
func Normalize(symbols []string) []string {
	var out []string
	for _, s := range symbols {
		s = strings.ToUpper(strings.TrimSpace(s))
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}
