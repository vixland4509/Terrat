package render

import (
	"testing"
)

func TestFontEngineZoom(t *testing.T) {
	fe, err := NewFontEngine(13.0)
	if err != nil {
		t.Fatalf("failed to initialize font engine: %v", err)
	}
	defer fe.Close()

	if fe.FontSize() != 13.0 {
		t.Fatalf("expected initial font size 13.0, got %f", fe.FontSize())
	}

	if !fe.ZoomIn() {
		t.Fatal("expected ZoomIn to succeed")
	}
	if fe.FontSize() != 14.0 {
		t.Fatalf("expected font size 14.0 after zoom in, got %f", fe.FontSize())
	}

	if !fe.ZoomOut() || !fe.ZoomOut() {
		t.Fatal("expected ZoomOut to succeed")
	}
	if fe.FontSize() != 12.0 {
		t.Fatalf("expected font size 12.0 after two zoom outs, got %f", fe.FontSize())
	}

	if !fe.ZoomReset() {
		t.Fatal("expected ZoomReset to succeed")
	}
	if fe.FontSize() != 13.0 {
		t.Fatalf("expected font size 13.0 after reset, got %f", fe.FontSize())
	}
}

func TestFallbackAndBoxDrawing(t *testing.T) {
	fe, err := NewFontEngine(13.0)
	if err != nil {
		t.Fatalf("failed to initialize font engine: %v", err)
	}
	defer fe.Close()

	// 1. Box drawing characters
	boxRunes := []rune{0x2500, 0x2502, 0x250C, 0x2510, 0x2514, 0x2518, 0x253C, 0x2550, 0x2551}
	for _, r := range boxRunes {
		mask := fe.GetGlyph(r, false)
		if mask == nil || mask.Width != fe.CharWidth() || mask.Height != fe.CharHeight() {
			t.Fatalf("expected valid mask for box rune U+%04X", r)
		}
		nonZero := 0
		for _, a := range mask.Alpha {
			if a > 0 {
				nonZero++
			}
		}
		if nonZero == 0 {
			t.Fatalf("expected box rune U+%04X to have rendered pixels, got 0", r)
		}
	}

	// 2. Block element
	fullBlock := fe.GetGlyph(0x2588, false)
	for i, a := range fullBlock.Alpha {
		if a != 0xff {
			t.Fatalf("expected full block pixel %d to be 0xff, got 0x%02x", i, a)
		}
	}

	// 3. Persian / Arabic fallback
	persianRunes := []rune{0x0633, 0x0644, 0x0627, 0x0645} // سلام
	for _, r := range persianRunes {
		mask := fe.GetGlyph(r, false)
		if mask == nil {
			t.Fatalf("expected non-nil mask for Persian rune U+%04X", r)
		}
		nonZero := 0
		for _, a := range mask.Alpha {
			if a > 0 {
				nonZero++
			}
		}
		if nonZero == 0 {
			t.Fatalf("expected Persian rune U+%04X to render non-zero pixels", r)
		}
	}
}
