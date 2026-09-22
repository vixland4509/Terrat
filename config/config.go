package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Shell                 string  `json:"shell"`
	DefaultShell          string  `json:"default_shell,omitempty"`
	Theme                 string  `json:"theme"`
	FontSize              float64 `json:"font_size"`
	Opacity               float64 `json:"opacity"`
	GhostText             bool    `json:"ghost_text"`
	Diagnostics           bool    `json:"diagnostics"`
	SanitizePaste         bool    `json:"sanitize_paste"`
	BracketedPaste        bool    `json:"bracketed_paste"`
}

func DefaultConfig() *Config {
	return &Config{
		Shell:          "",
		Theme:          "auto",
		FontSize:       13.0,
		Opacity:        0.95,
		GhostText:      true,
		Diagnostics:    true,
		SanitizePaste:  true,
		BracketedPaste: true,
	}
}

func ConfigPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "terrat", "config.json"), nil
	}
	if exe, err := os.Executable(); err == nil {
		portableCfg := filepath.Join(filepath.Dir(exe), "config.json")
		if _, err := os.Stat(portableCfg); err == nil {
			return portableCfg, nil
		}
	}
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

	if cfg.Shell == "" && cfg.DefaultShell != "" {
		cfg.Shell = cfg.DefaultShell
	}
	cfg.DefaultShell = ""

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
