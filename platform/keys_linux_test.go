//go:build !windows

package platform

import (
	"testing"

	"github.com/jezek/xgb/xproto"
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

	// Numpad / Keypad digits 0-9
	for i := 0; i <= 9; i++ {
		sym := 0xffb0 + i
		r, ok := KeysymToRune(xproto.Keysym(sym))
		expected := rune('0' + i)
		if !ok || r != expected {
			t.Fatalf("expected KP_%d to be %q, got %q", i, expected, r)
		}
	}

	// Numpad operators
	kpOps := map[xproto.Keysym]rune{
		0xffaa: '*',
		0xffab: '+',
		0xffac: ',',
		0xffad: '-',
		0xffae: '.',
		0xffaf: '/',
		0xffbd: '=',
		0xff80: ' ',
		0xff89: '\t',
		0xff8d: '\r',
	}
	for sym, expected := range kpOps {
		r, ok := KeysymToRune(sym)
		if !ok || r != expected {
			t.Fatalf("expected keysym 0x%x to be %q, got %q", sym, expected, r)
		}
	}
}

func TestTranslateNumpadAndAppCursor(t *testing.T) {
	kh := &KeyHandler{}

	// Test Numpad digits translate
	for i := 0; i <= 9; i++ {
		sym := uint32(0xffb0 + i)
		ev := xproto.KeyPressEvent{}
		kh.keysyms = []xproto.Keysym{xproto.Keysym(sym)}
		kh.keysymsPerKeycode = 1
		kh.minKeycode = 10
		kh.maxKeycode = 10
		ev.Detail = 10

		b, act := kh.Translate(ev)
		if act != ActionNone || len(b) != 1 || b[0] != byte('0'+i) {
			t.Fatalf("expected numpad digit %d, got b=%q, act=%v", i, string(b), act)
		}
	}

	// Test AppCursor Mode for arrow keys
	kh.keysyms = []xproto.Keysym{0xff52} // Up Arrow
	ev := xproto.KeyPressEvent{Detail: 10}

	// Normal mode (appCursor = false)
	b, _ := kh.Translate(ev, false)
	if string(b) != "\x1b[A" {
		t.Fatalf("expected \\x1b[A in normal cursor mode, got %q", string(b))
	}

	// AppCursor mode (appCursor = true)
	b, _ = kh.Translate(ev, true)
	if string(b) != "\x1bOA" {
		t.Fatalf("expected \\x1bOA in appCursor mode, got %q", string(b))
	}
}
