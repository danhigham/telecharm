package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danhigham/telecharm/internal/config"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := []byte(`telegram:
  api_id: 12345
  api_hash: "abcdef0123456789"
log_level: debug
`)
	if err := os.WriteFile(cfgPath, content, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Telegram.APIID != 12345 {
		t.Errorf("APIID = %d, want 12345", cfg.Telegram.APIID)
	}
	if cfg.Telegram.APIHash != "abcdef0123456789" {
		t.Errorf("APIHash = %q, want %q", cfg.Telegram.APIHash, "abcdef0123456789")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := config.Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestConfigDir(t *testing.T) {
	dir := config.Dir()
	if dir == "" {
		t.Error("Dir() returned empty string")
	}
}
