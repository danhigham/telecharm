package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Telegram TelegramConfig `yaml:"telegram"`
	LogLevel string         `yaml:"log_level"`
	Bubbles  *bool          `yaml:"bubbles,omitempty"`
}

// BubblesEnabled returns the bubbles preference, defaulting to true.
func (c *Config) BubblesEnabled() bool {
	if c.Bubbles == nil {
		return true
	}
	return *c.Bubbles
}

// SetBubbles sets the bubbles preference.
func (c *Config) SetBubbles(v bool) {
	c.Bubbles = &v
}

// Save writes the config to the given path.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

type TelegramConfig struct {
	APIID   int    `yaml:"api_id"`
	APIHash string `yaml:"api_hash"`
}

func Dir() string {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		cfgDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(cfgDir, "telecharm")
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	return &cfg, nil
}
