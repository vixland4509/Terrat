package terminal

import (
	"testing"
)

func TestBracketedPasteMode(t *testing.T) {
	term := New(80, 24)

	if term.BracketedPaste() {
		t.Fatalf("expected initial bracketedPaste to be false, got true")
	}

	// Enable bracketed paste: CSI ? 2004 h
	_, err := term.Write([]byte("\x1b[?2004h"))
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if !term.BracketedPaste() {
		t.Fatalf("expected bracketedPaste to be true after \\x1b[?2004h")
	}

	// Disable bracketed paste: CSI ? 2004 l
	_, err = term.Write([]byte("\x1b[?2004l"))
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if term.BracketedPaste() {
		t.Fatalf("expected bracketedPaste to be false after \\x1b[?2004l")
	}
}
