package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const PluginID = "herdr-cliproxyapi-quotas"

var ErrConfigNotFound = errors.New("configuration not found")

type Config struct {
	BaseURL       string `json:"base_url"`
	ManagementKey string `json:"management_key"`
}

func DefaultConfig() Config {
	return Config{BaseURL: "http://127.0.0.1:8317"}
}

func Path() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("HERDR_PLUGIN_CONFIG_DIR")); dir != "" {
		return filepath.Join(dir, "config.json"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: %w", err)
	}
	return filepath.Join(dir, "herdr", "plugins", PluginID, "config.json"), nil
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultConfig(), ErrConfigNotFound
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	cfg = cfg.Normalized()
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	cfg = cfg.Normalized()
	if err := cfg.Validate(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil && !errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("secure config: %w", err)
	}
	return nil
}

func (c Config) Normalized() Config {
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	c.ManagementKey = strings.TrimSpace(c.ManagementKey)
	return c
}

func (c Config) Validate() error {
	if c.BaseURL == "" {
		return errors.New("Base URL is required")
	}
	parsed, err := url.Parse(c.BaseURL)
	if err != nil || parsed.Host == "" {
		return errors.New("Base URL must be an absolute HTTP(S) URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("Base URL must use http or https")
	}
	if parsed.User != nil {
		return errors.New("Base URL must not contain credentials")
	}
	if c.ManagementKey == "" {
		return errors.New("Management key is required")
	}
	return nil
}
