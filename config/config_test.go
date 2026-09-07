package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDefault(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Theme != "auto" {
		t.Fatalf("expected default theme to be 'auto', got %s", cfg.Theme)
	}
	if cfg.FontSize != 13.0 {
		t.Fatalf("expected default font size 13.0, got %f", cfg.FontSize)
	}
	if cfg.Opacity != 0.95 {
		t.Fatalf("expected default opacity 0.95, got %f", cfg.Opacity)
	}
	if !cfg.GhostText {
		t.Fatalf("expected default GhostText to be true")
	}
	if !cfg.ConfirmMultilinePaste {
		t.Fatalf("expected default ConfirmMultilinePaste to be true")
	}
	if !cfg.SanitizePaste {
		t.Fatalf("expected default SanitizePaste to be true")
	}
	if !cfg.BracketedPaste {
		t.Fatalf("expected default BracketedPaste to be true")
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "terrat-config-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origConfigHome)
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfgPath, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(cfgPath) != filepath.Join(tmpDir, "terrat") {
		t.Fatalf("unexpected config path: %s", cfgPath)
	}

	loaded := Load()
	if loaded.Theme != "auto" || loaded.FontSize != 13.0 || loaded.Opacity != 0.95 || !loaded.GhostText || !loaded.Diagnostics {
		t.Fatalf("expected loaded default values, got theme=%s, size=%f, opacity=%f, ghost_text=%v, diagnostics=%v", loaded.Theme, loaded.FontSize, loaded.Opacity, loaded.GhostText, loaded.Diagnostics)
	}

	loaded.Theme = "catppuccin-mocha"
	loaded.FontSize = 15.5
	loaded.Opacity = 0.90
	loaded.GhostText = false
	loaded.Diagnostics = false
	if err := Save(loaded); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	reloaded := Load()
	if reloaded.Theme != "catppuccin-mocha" {
		t.Fatalf("expected reloaded theme to be 'catppuccin-mocha', got %s", reloaded.Theme)
	}
	if reloaded.FontSize != 15.5 {
		t.Fatalf("expected reloaded font size to be 15.5, got %f", reloaded.FontSize)
	}
	if reloaded.Opacity != 0.90 {
		t.Fatalf("expected reloaded opacity to be 0.90, got %f", reloaded.Opacity)
	}
	if reloaded.GhostText != false {
		t.Fatalf("expected reloaded GhostText to be false, got %v", reloaded.GhostText)
	}
	if reloaded.Diagnostics != false {
		t.Fatalf("expected reloaded Diagnostics to be false, got %v", reloaded.Diagnostics)
	}
}
