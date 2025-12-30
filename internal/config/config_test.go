// internal/config/config_test.go
package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Digest.DefaultDays != 7 {
		t.Errorf("expected default days 7, got %d", cfg.Digest.DefaultDays)
	}

	if cfg.Fetch.Concurrency != 5 {
		t.Errorf("expected concurrency 5, got %d", cfg.Fetch.Concurrency)
	}
}

func TestConfigDir(t *testing.T) {
	dir := ConfigDir()
	if dir == "" {
		t.Error("config dir should not be empty")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := DefaultConfig()
	cfg.GitHub.Token = "test-token"

	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.GitHub.Token != "test-token" {
		t.Errorf("expected token 'test-token', got '%s'", loaded.GitHub.Token)
	}
}
