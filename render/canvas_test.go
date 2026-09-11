package render

import (
	"testing"
	"terrat/terminal"
)

func TestRenderPreferencesModal(t *testing.T) {
	fe, err := NewFontEngine(13.0)
	if err != nil {
		t.Fatalf("failed to init font engine: %v", err)
	}
	defer fe.Close()

	c := NewCanvas(fe)
	c.Resize(800, 600)

	th := terminal.ThemeTokyoNight

	options := []PrefOption{
		{ID: "diagnostics", Label: "Live Diagnostics", IsToggle: true, Enabled: true},
		{ID: "ghost_text", Label: "Ghost Autocomplete", IsToggle: true, Enabled: false},
		{ID: "theme", Label: "Color Theme", Value: "Tokyo Night"},
	}

	modalX, modalY, modalW, modalH, rowH := c.RenderPreferencesModal(th, 0, options, "tokyo-night")

	if modalW <= 0 || modalH <= 0 || rowH <= 0 {
		t.Fatalf("expected positive modal dimensions, got w=%d, h=%d, rowH=%d", modalW, modalH, rowH)
	}

	// Modal should be anchored in top-right corner under header
	if modalX < c.Width/2 {
		t.Fatalf("expected modal to be in the right corner, got modalX=%d in width %d", modalX, c.Width)
	}
	if modalY < HeaderHeight {
		t.Fatalf("expected modal to be below header, got modalY=%d, HeaderHeight=%d", modalY, HeaderHeight)
	}
}
