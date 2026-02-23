package sprite

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds persistent smol configuration.
type Config struct {
	Token  string `json:"token"`
	APIURL string `json:"api_url"`
	Org    string `json:"org"`
	Email  string `json:"email,omitempty"`
}

// ConfigPath returns the path to the config file.
func ConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine config directory: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "smol", "config.json"), nil
}

// LoadConfig reads the config from disk. Returns a zero Config if no file exists.
func LoadConfig() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// IsLoggedIn returns true if a token is configured.
func (c *Config) IsLoggedIn() bool {
	return c.Token != ""
}
