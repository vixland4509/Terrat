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
