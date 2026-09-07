package terminal

import "testing"

func TestThemeResolution(t *testing.T) {
	if th := ResolveTheme("tokyo-night", false); th.ID != "tokyo-night" {
		t.Fatalf("expected tokyo-night, got %s", th.ID)
	}
	if th := ResolveTheme("catppuccin-mocha", false); th.ID != "catppuccin-mocha" {
		t.Fatalf("expected catppuccin-mocha, got %s", th.ID)
	}
	if th := ResolveTheme("minecraft", false); th.ID != "minecraft" {
		t.Fatalf("expected minecraft, got %s", th.ID)
	}
	if th := ResolveTheme("tokyo-day", true); th.ID != "tokyo-day" {
		t.Fatalf("expected tokyo-day, got %s", th.ID)
	}
	if th := ResolveTheme("solarized-light", true); th.ID != "solarized-light" {
		t.Fatalf("expected solarized-light, got %s", th.ID)
	}

	if th := ResolveTheme("auto", true); th.ID != "tokyo-night" {
		t.Fatalf("expected auto dark to resolve to tokyo-night, got %s", th.ID)
	}
	if th := ResolveTheme("auto", false); th.ID != "tokyo-day" {
		t.Fatalf("expected auto light to resolve to tokyo-day, got %s", th.ID)
	}
	if th := ResolveTheme("", true); th.ID != "tokyo-night" {
		t.Fatalf("expected empty pref to resolve to tokyo-night on dark, got %s", th.ID)
	}
}

func TestTerminalSetTheme(t *testing.T) {
	term := New(80, 24)
	if term.Theme().ID != "tokyo-night" {
		t.Fatalf("expected default tokyo-night, got %s", term.Theme().ID)
	}

	term.SetTheme(ThemeCatppuccinMocha)
	if term.Theme().ID != "catppuccin-mocha" {
		t.Fatalf("expected catppuccin-mocha, got %s", term.Theme().ID)
	}

	term.SetTheme(ThemeMinecraft)
	if term.Theme().ID != "minecraft" {
		t.Fatalf("expected minecraft, got %s", term.Theme().ID)
	}

	term.SetTheme(ThemeTokyoDay)
	if term.Theme().ID != "tokyo-day" {
		t.Fatalf("expected tokyo-day, got %s", term.Theme().ID)
	}
}

func TestClearWithLightTheme(t *testing.T) {
	term := New(80, 24)
	term.SetTheme(ThemeTokyoDay)

	_, _ = term.Write([]byte("vixland@vixland-Compute: ~/Documents/Terrat$ clear\r\n"))
	_, _ = term.Write([]byte("\x1b[H\x1b[2J\x1b[3J"))

	for y := 0; y < 24; y++ {
		for x := 0; x < 80; x++ {
			cell := term.GetCell(x, y)
			if cell.BG != ColorDefaultBG {
				t.Fatalf("cell (%d, %d) has non-default BG: 0x%08x (expected ColorDefaultBG)", x, y, cell.BG)
			}
		}
	}
}
