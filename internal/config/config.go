package config

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds the resolved hass-cli configuration.
type Config struct {
	URL   string
	Token string
}

// configDir returns the directory where the config file lives.
func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "hass-cli"), nil
}

// ConfigFilePath returns the path to the saved .env config file.
func ConfigFilePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".env"), nil
}

// Load reads the config file and returns the values. Returns an empty Config
// (no error) if the file doesn't exist yet.
func Load() (*Config, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return &Config{}, nil
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	defer f.Close()

	cfg := &Config{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "HASS_URL":
			cfg.URL = strings.TrimSpace(val)
		case "HASS_TOKEN":
			cfg.Token = strings.TrimSpace(val)
		}
	}
	return cfg, scanner.Err()
}

// Save writes the config to the config file, creating directories as needed.
func Save(cfg *Config) error {
	dir, err := configDir()
	if err != nil {
		return fmt.Errorf("find config dir: %w", err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	path := filepath.Join(dir, ".env")
	content := fmt.Sprintf("HASS_URL=%s\nHASS_TOKEN=%s\n", cfg.URL, cfg.Token)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// DiscoverURL tries common HA addresses and returns the first one that
// responds with any HTTP reply (even 401 — that means HA is there).
// Returns "" if nothing is found.
func DiscoverURL() string {
	candidates := []string{
		"http://homeassistant.local:8123",
		"http://homeassistant:8123",
		"http://localhost:8123",
	}
	client := &http.Client{Timeout: 2 * time.Second}
	for _, addr := range candidates {
		resp, err := client.Get(addr + "/api/")
		if err == nil {
			resp.Body.Close()
			return addr
		}
	}
	return ""
}
