package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Theme    string  `json:"theme"`
	FontSize float64 `json:"font_size"`
	Opacity  float64 `json:"opacity"`
}

func DefaultConfig() *Config {
	return &Config{
		Theme:    "auto",
		FontSize: 13.0,
		Opacity:  0.95,
	}
}

func ConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "terrat", "config.json"), nil
}

func Load() *Config {
	path, err := ConfigPath()
	if err != nil {
		return DefaultConfig()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		cfg := DefaultConfig()
		_ = Save(cfg)
		return cfg
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return DefaultConfig()
	}

	if cfg.Theme == "" {
		cfg.Theme = "auto"
	}
	if cfg.FontSize < 7.0 || cfg.FontSize > 36.0 {
		cfg.FontSize = 13.0
	}
	if cfg.Opacity < 0.2 || cfg.Opacity > 1.0 {
		cfg.Opacity = 0.95
	}

	return cfg
}

func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
