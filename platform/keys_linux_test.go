//go:build !windows

package platform

import (
	"testing"
)

func TestKeysymToRune(t *testing.T) {
	// ASCII
	r, ok := KeysymToRune('a')
	if !ok || r != 'a' {
		t.Fatalf("expected 'a', got %q, %v", r, ok)
	}

	// Latin-1
	r, ok = KeysymToRune(0x00e9) // é
	if !ok || r != 'é' {
		t.Fatalf("expected 'é', got %q, %v", r, ok)
	}

	// Direct Unicode (Persian 'پ' = U+067E)
	r, ok = KeysymToRune(0x067e)
	if !ok || r != 'پ' {
		t.Fatalf("expected 'پ', got %q, %v", r, ok)
	}

	// Direct Unicode with 0x01000000 flag
	r, ok = KeysymToRune(0x0100067e)
	if !ok || r != 'پ' {
		t.Fatalf("expected 'پ' from 0x0100067e, got %q, %v", r, ok)
	}

	// Standard X11 Arabic comma (0x05ac -> U+060C)
	r, ok = KeysymToRune(0x05ac)
	if !ok || r != '\u060C' {
		t.Fatalf("expected Arabic comma, got %q, %v", r, ok)
	}

	// Standard X11 Arabic alef (0x05c7 -> U+0627)
	r, ok = KeysymToRune(0x05c7)
	if !ok || r != 'ا' {
		t.Fatalf("expected 'ا', got %q, %v", r, ok)
	}
}
