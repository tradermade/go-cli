package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Env vars beat the config file; the scoped (REST/WS) ones beat the generic.
const (
	EnvKey     = "TRADERMADE_API_KEY"
	EnvRESTKey = "TRADERMADE_REST_API_KEY"
	EnvWSKey   = "TRADERMADE_WS_API_KEY"
)

// Config is the on-disk config file. APIKey is the single-key setup;
// RESTKey/WSKey override it per side for plans with split keys.
type Config struct {
	APIKey  string `json:"api_key,omitempty"`
	RESTKey string `json:"rest_key,omitempty"`
	WSKey   string `json:"ws_key,omitempty"`
}

// Path returns the full path of the config file, e.g.
// %AppData%\tradermade\config.json on Windows or
// ~/.config/tradermade/config.json on Linux/macOS.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot locate user config directory: %w", err)
	}
	return filepath.Join(dir, "tradermade", "config.json"), nil
}

// Load reads the config file. A missing file returns an empty Config, not an error.
func Load() (Config, error) {
	var cfg Config
	path, err := Path()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("cannot read config file %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("config file %s is not valid JSON: %w", path, err)
	}
	return cfg, nil
}

// Save writes the config file, creating the directory if needed.
func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("cannot write config file %s: %w", path, err)
	}
	return nil
}

// ResolveRESTKey returns the key REST commands should use.
// Order: scoped env var > scoped config key > generic env var > generic config key.
func ResolveRESTKey() (string, error) {
	key, _, err := resolveScoped(EnvRESTKey, func(c Config) string { return c.RESTKey }, "REST", "--rest")
	return key, err
}

// ResolveWSKey returns the key streaming commands (stream, board) should use.
func ResolveWSKey() (string, error) {
	key, _, err := resolveScoped(EnvWSKey, func(c Config) string { return c.WSKey }, "WebSocket", "--ws")
	return key, err
}

// ResolveRESTKeySource and ResolveWSKeySource also report where the key came
// from, for `config show` and `doctor`.
func ResolveRESTKeySource() (string, string, error) {
	return resolveScoped(EnvRESTKey, func(c Config) string { return c.RESTKey }, "REST", "--rest")
}

func ResolveWSKeySource() (string, string, error) {
	return resolveScoped(EnvWSKey, func(c Config) string { return c.WSKey }, "WebSocket", "--ws")
}

func resolveScoped(envName string, fromConfig func(Config) string, side, flag string) (key, source string, err error) {
	if key := strings.TrimSpace(os.Getenv(envName)); key != "" {
		return key, envName + " environment variable", nil
	}
	cfg, err := Load()
	if err != nil {
		return "", "", err
	}
	path, _ := Path()
	if key := strings.TrimSpace(fromConfig(cfg)); key != "" {
		return key, path + " (" + strings.TrimPrefix(flag, "--") + "_key)", nil
	}
	if key := strings.TrimSpace(os.Getenv(EnvKey)); key != "" {
		return key, EnvKey + " environment variable", nil
	}
	if key := strings.TrimSpace(cfg.APIKey); key != "" {
		return key, path + " (api_key)", nil
	}
	return "", "", fmt.Errorf("no %s API key found - run `tradermade config set-key %s YOUR_KEY` (or `config set-key` if one key covers both), or set %s\nGet a key at https://tradermade.com/signup", side, flag, envName)
}

// MaskKey renders a key safe for display: first 4 and last 4 characters kept.
func MaskKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}
