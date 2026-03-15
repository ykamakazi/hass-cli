package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir) // redirect config dir resolution

	// Override configDir to use temp dir directly.
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg := &Config{
		URL:   "http://homeassistant.local:8123",
		Token: "test-token-abc123",
	}

	// Save.
	path := filepath.Join(dir, "hass-cli", ".env")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}

	content := "HASS_URL=" + cfg.URL + "\nHASS_TOKEN=" + cfg.Token + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	// Load back.
	loaded, err := loadFromPath(path)
	if err != nil {
		t.Fatalf("loadFromPath() error = %v", err)
	}
	if loaded.URL != cfg.URL {
		t.Errorf("URL = %q, want %q", loaded.URL, cfg.URL)
	}
	if loaded.Token != cfg.Token {
		t.Errorf("Token = %q, want %q", loaded.Token, cfg.Token)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := loadFromPath("/nonexistent/path/.env")
	if err != nil {
		t.Fatalf("loadFromPath() on missing file returned error: %v", err)
	}
	if cfg.URL != "" || cfg.Token != "" {
		t.Errorf("expected empty config for missing file, got %+v", cfg)
	}
}

func TestLoad_IgnoresComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# This is a comment\nHASS_URL=http://ha.local:8123\n# Another comment\nHASS_TOKEN=mytoken\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.URL != "http://ha.local:8123" {
		t.Errorf("URL = %q, want %q", cfg.URL, "http://ha.local:8123")
	}
	if cfg.Token != "mytoken" {
		t.Errorf("Token = %q, want %q", cfg.Token, "mytoken")
	}
}

func TestLoad_IgnoresUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "HASS_URL=http://ha.local:8123\nUNKNOWN_KEY=value\nHASS_TOKEN=tok\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.URL == "" || cfg.Token == "" {
		t.Errorf("expected URL and Token to be set, got %+v", cfg)
	}
}
