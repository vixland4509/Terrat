//go:build !windows

package platform

import (
	"unicode/utf8"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type KeyHandler struct {
	minKeycode        xproto.Keycode
	maxKeycode        xproto.Keycode
	keysymsPerKeycode byte
	keysyms           []xproto.Keysym
	numLockMask       uint16
}

func NewKeyHandler(X *xgb.Conn) (*KeyHandler, error) {
	setup := xproto.Setup(X)
	minKey := setup.MinKeycode
	maxKey := setup.MaxKeycode
	count := byte(maxKey - minKey + 1)

	mapping, err := xproto.GetKeyboardMapping(X, minKey, count).Reply()
	if err != nil {
		return nil, err
	}

	kh := &KeyHandler{
		minKeycode:        minKey,
		maxKeycode:        maxKey,
		keysymsPerKeycode: mapping.KeysymsPerKeycode,
		keysyms:           mapping.Keysyms,
		numLockMask:       0x10, // Mod2 default
	}
	kh.detectNumLockMask(X)
	return kh, nil
}

func (kh *KeyHandler) detectNumLockMask(X *xgb.Conn) {
	modMap, err := xproto.GetModifierMapping(X).Reply()
	if err != nil || modMap == nil {
		kh.numLockMask = 0x10
		return
	}
	kpm := int(modMap.KeycodesPerModifier)
	for mod := 0; mod < 8; mod++ {
		mask := uint16(1 << mod)
		for k := 0; k < kpm; k++ {
			kc := modMap.Keycodes[mod*kpm+k]
			if kc == 0 || kc < kh.minKeycode || kc > kh.maxKeycode {
				continue
			}
			idx := int(kc-kh.minKeycode) * int(kh.keysymsPerKeycode)
			for c := 0; c < int(kh.keysymsPerKeycode); c++ {
				if idx+c < len(kh.keysyms) && kh.keysyms[idx+c] == 0xff7f { // XK_Num_Lock
					kh.numLockMask = mask
					return
				}
			}
		}
	}
}

func (kh *KeyHandler) KeySym(ev xproto.KeyPressEvent) xproto.Keysym {
	keycode := ev.Detail
	if keycode < kh.minKeycode || keycode > kh.maxKeycode {
		return 0
	}
	idx := int(keycode-kh.minKeycode) * int(kh.keysymsPerKeycode)
	if idx >= len(kh.keysyms) {
		return 0
	}

	// In X11, bits 13-14 specify keyboard group (0-3) for layout switching
	group := int((ev.State >> 13) & 3)
	col := group * 2
	if col >= int(kh.keysymsPerKeycode) {
		col = 0
	}

	isShift := (ev.State & ModShift) != 0
	isLock := (ev.State & ModLock) != 0 // CapsLock

	isKP := false
	for c := 0; c < int(kh.keysymsPerKeycode); c++ {
		s := kh.keysyms[idx+c]
		if (s >= 0xff80 && s <= 0xffbd) || (s >= 0xffb0 && s <= 0xffb9) {
			isKP = true
			break
		}
	}

	keysym := kh.keysyms[idx+col]
	shiftCol := col + 1

	if isKP {
		isNumLock := (ev.State & kh.numLockMask) != 0
		effectiveShift := isShift != isNumLock
		if effectiveShift && shiftCol < int(kh.keysymsPerKeycode) && kh.keysyms[idx+shiftCol] != 0 {
			keysym = kh.keysyms[idx+shiftCol]
		}
	} else {
		if shiftCol < int(kh.keysymsPerKeycode) && kh.keysyms[idx+shiftCol] != 0 {
			if isShift {
				keysym = kh.keysyms[idx+shiftCol]
			}
		}
	}

	// Fallback to Group 0 if current group has no keysym
	if keysym == 0 && col != 0 {
		keysym = kh.keysyms[idx]
		if isShift && kh.keysymsPerKeycode > 1 && kh.keysyms[idx+1] != 0 {
			keysym = kh.keysyms[idx+1]
		}
	}

	// Apply CapsLock for ASCII letters
	if isLock && !isShift && keysym >= 'a' && keysym <= 'z' {
		keysym -= ('a' - 'A')
	} else if isLock && isShift && keysym >= 'A' && keysym <= 'Z' {
		keysym += ('a' - 'A')
	}

	return keysym
}

func (kh *KeyHandler) RefreshMapping(X *xgb.Conn) error {
	setup := xproto.Setup(X)
	minKey := setup.MinKeycode
	maxKey := setup.MaxKeycode
	count := byte(maxKey - minKey + 1)

	mapping, err := xproto.GetKeyboardMapping(X, minKey, count).Reply()
	if err != nil {
		return err
	}
	kh.minKeycode = minKey
	kh.maxKeycode = maxKey
	kh.keysymsPerKeycode = mapping.KeysymsPerKeycode
	kh.keysyms = mapping.Keysyms
	kh.detectNumLockMask(X)
	return nil
}

func (kh *KeyHandler) Translate(ev xproto.KeyPressEvent, appCursor ...bool) ([]byte, ActionType) {
	keysym := kh.KeySym(ev)
	if keysym == 0 {
		return nil, ActionNone
	}

	isAppCursor := len(appCursor) > 0 && appCursor[0]
	state := ev.State
	isShift := (state & ModShift) != 0
	isCtrl := (state & ModCtrl) != 0
	isAlt := (state & ModAlt) != 0

	if isAlt && !isCtrl && keysym >= '1' && keysym <= '9' {
		return nil, ActionSwitchTab1 + ActionType(keysym-'1')
	}

	if isCtrl && !isShift {
		if keysym == '=' || keysym == '+' {
			return nil, ActionZoomIn
		}
		if keysym == '-' || keysym == '_' {
			return nil, ActionZoomOut
		}
		if keysym == '0' {
			return nil, ActionZoomReset
		}
	}
	if isCtrl && isShift {
		if keysym == '+' || keysym == '=' {
			return nil, ActionZoomIn
		}
		if keysym == '_' || keysym == '-' {
			return nil, ActionZoomOut
		}
	}

	if isCtrl {
		if keysym == ',' || keysym == '<' {
			return nil, ActionPreferences
		}
		if isShift && (keysym == 'P' || keysym == 'p') {
			return nil, ActionPreferences
		}
	}

	if isCtrl && isShift {
		if keysym == 'T' || keysym == 't' {
			return nil, ActionNewTab
		}
		if keysym == 'W' || keysym == 'w' {
			return nil, ActionCloseTab
		}
		if keysym == 'F' || keysym == 'f' {
			return nil, ActionSearch
		}
		if keysym == 'D' || keysym == 'd' {
			return nil, ActionToggleDiagnostics
		}
		if keysym == 0xff09 || keysym == 0xfe20 {
			return nil, ActionPrevTab
		}
		if keysym == 0xff55 {
			return nil, ActionPrevTab
		}
		if keysym == 0xff56 {
			return nil, ActionNextTab
		}
	}

	if isCtrl && !isShift {
		if keysym == 0xff09 {
			return nil, ActionNextTab
		}
		if keysym == 0xff55 {
			return nil, ActionPrevTab
		}
		if keysym == 0xff56 {
			return nil, ActionNextTab
		}
	}

	if (keysym == 0xff55 || keysym == 0xff9a) && isShift {
		return nil, ActionScrollUp
	}
	if (keysym == 0xff56 || keysym == 0xff9b) && isShift {
		return nil, ActionScrollDown
	}
	if (keysym == 0xff50 || keysym == 0xff95) && isShift {
		return nil, ActionScrollTop
	}
	if (keysym == 0xff57 || keysym == 0xff9c) && isShift {
		return nil, ActionScrollBottom
	}

	if isCtrl && isShift {
		if keysym == 'C' || keysym == 'c' {
			return nil, ActionCopy
		}
		if keysym == 'V' || keysym == 'v' {
			return nil, ActionPaste
		}
		if keysym == 'A' || keysym == 'a' {
			return nil, ActionSelectAll
		}
	}
	if isShift && keysym == 0xff63 {
		return nil, ActionPaste
	}

	if isCtrl {
		if keysym >= 'a' && keysym <= 'z' {
			return []byte{byte(keysym - 'a' + 1)}, ActionNone
		}
		if keysym >= 'A' && keysym <= 'Z' {
			return []byte{byte(keysym - 'A' + 1)}, ActionNone
		}
		switch keysym {
		case '@':
			return []byte{0x00}, ActionNone
		case '[':
			return []byte{0x1b}, ActionNone
		case '\\':
			return []byte{0x1c}, ActionNone
		case ']':
			return []byte{0x1d}, ActionNone
		case '^':
			return []byte{0x1e}, ActionNone
		case '_':
			return []byte{0x1f}, ActionNone
		}
	}

	switch keysym {
	// Keypad numbers (0-9)
	case 0xffb0, 0xffb1, 0xffb2, 0xffb3, 0xffb4, 0xffb5, 0xffb6, 0xffb7, 0xffb8, 0xffb9:
		return []byte{byte('0' + (keysym - 0xffb0))}, ActionNone

	// Keypad arithmetic and separators
	case 0xffaa: // KP_Multiply
		return []byte("*"), ActionNone
	case 0xffab: // KP_Add
		return []byte("+"), ActionNone
	case 0xffac: // KP_Separator
		return []byte(","), ActionNone
	case 0xffad: // KP_Subtract
		return []byte("-"), ActionNone
	case 0xffae: // KP_Decimal
		return []byte("."), ActionNone
	case 0xffaf: // KP_Divide
		return []byte("/"), ActionNone
	case 0xffbd: // KP_Equal
		return []byte("="), ActionNone
	case 0xff80: // KP_Space
		return []byte(" "), ActionNone
	case 0xff89: // KP_Tab
		return []byte("\t"), ActionNone

	// Keypad navigation (when NumLock is OFF)
	case 0xff95: // KP_Home
		if isAppCursor {
			return []byte("\x1bOH"), ActionNone
		}
		return []byte("\x1b[H"), ActionNone
	case 0xff96: // KP_Left
		if isAppCursor {
			return []byte("\x1bOD"), ActionNone
		}
		return []byte("\x1b[D"), ActionNone
	case 0xff97: // KP_Up
		if isAppCursor {
			return []byte("\x1bOA"), ActionNone
		}
		return []byte("\x1b[A"), ActionNone
	case 0xff98: // KP_Right
		if isAppCursor {
			return []byte("\x1bOC"), ActionNone
		}
		return []byte("\x1b[C"), ActionNone
	case 0xff99: // KP_Down
		if isAppCursor {
			return []byte("\x1bOB"), ActionNone
		}
		return []byte("\x1b[B"), ActionNone
	case 0xff9a: // KP_Prior (Page Up)
		return []byte("\x1b[5~"), ActionNone
	case 0xff9b: // KP_Next (Page Down)
		return []byte("\x1b[6~"), ActionNone
	case 0xff9c: // KP_End
		if isAppCursor {
			return []byte("\x1bOF"), ActionNone
		}
		return []byte("\x1b[F"), ActionNone
	case 0xff9d: // KP_Begin (Center 5)
		return []byte("\x1b[E"), ActionNone
	case 0xff9e: // KP_Insert
		return []byte("\x1b[2~"), ActionNone
	case 0xff9f: // KP_Delete
		return []byte("\x1b[3~"), ActionNone
	case 0xff0d, 0xff8d:
		return []byte("\r"), ActionNone
	case 0xff08:
		return []byte("\x7f"), ActionNone
	case 0xff09:
		if isShift {
			return []byte("\x1b[Z"), ActionNone
		}
		return []byte("\t"), ActionNone
	case 0xff1b:
		return []byte("\x1b"), ActionNone
	case 0xffff:
		return []byte("\x1b[3~"), ActionNone
	case 0xff63:
		return []byte("\x1b[2~"), ActionNone
	case 0xff50:
		if isAppCursor {
			return []byte("\x1bOH"), ActionNone
		}
		return []byte("\x1b[H"), ActionNone
	case 0xff57:
		if isAppCursor {
			return []byte("\x1bOF"), ActionNone
		}
		return []byte("\x1b[F"), ActionNone
	case 0xff55:
		return []byte("\x1b[5~"), ActionNone
	case 0xff56:
		return []byte("\x1b[6~"), ActionNone
	case 0xff52:
		if isAppCursor {
			return []byte("\x1bOA"), ActionNone
		}
		return []byte("\x1b[A"), ActionNone
	case 0xff54:
		if isAppCursor {
			return []byte("\x1bOB"), ActionNone
		}
		return []byte("\x1b[B"), ActionNone
	case 0xff53:
		if isAppCursor {
			return []byte("\x1bOC"), ActionNone
		}
		return []byte("\x1b[C"), ActionNone
	case 0xff51:
		if isAppCursor {
			return []byte("\x1bOD"), ActionNone
		}
		return []byte("\x1b[D"), ActionNone

	case 0xffbe:
		return []byte("\x1bOP"), ActionNone
	case 0xffbf:
		return []byte("\x1bOQ"), ActionNone
	case 0xffc0:
		return []byte("\x1bOR"), ActionNone
	case 0xffc1:
		return []byte("\x1bOS"), ActionNone
	case 0xffc2:
		return []byte("\x1b[15~"), ActionNone
	case 0xffc3:
		return []byte("\x1b[17~"), ActionNone
	case 0xffc4:
		return []byte("\x1b[18~"), ActionNone
	case 0xffc5:
		return []byte("\x1b[19~"), ActionNone
	case 0xffc6:
		return []byte("\x1b[20~"), ActionNone
	case 0xffc7:
		return []byte("\x1b[21~"), ActionNone
	case 0xffc8:
		return []byte("\x1b[23~"), ActionNone
	case 0xffc9:
		return []byte("\x1b[24~"), ActionNone
	}

	if r, ok := KeysymToRune(keysym); ok {
		buf := make([]byte, 4)
		n := utf8.EncodeRune(buf, r)
		return buf[:n], ActionNone
	}

	return nil, ActionNone
}

// KeysymToRune converts an X11 keysym to its corresponding Unicode rune.
func KeysymToRune(keysym xproto.Keysym) (rune, bool) {
	// Keypad numbers (0-9)
	if keysym >= 0xffb0 && keysym <= 0xffb9 {
		return rune('0' + (keysym - 0xffb0)), true
	}

	// Keypad arithmetic and special
	switch keysym {
	case 0xffaa:
		return '*', true
	case 0xffab:
		return '+', true
	case 0xffac:
		return ',', true
	case 0xffad:
		return '-', true
	case 0xffae:
		return '.', true
	case 0xffaf:
		return '/', true
	case 0xffbd:
		return '=', true
	case 0xff80:
		return ' ', true
	case 0xff89:
		return '\t', true
	case 0xff8d:
		return '\r', true
	}
	// Standard ASCII printable
	if keysym >= 0x0020 && keysym <= 0x007e {
		return rune(keysym), true
	}

	// Latin-1 Supplement
	if keysym >= 0x00a0 && keysym <= 0x00ff {
		return rune(keysym), true
	}

	// Direct Unicode keysyms (0x01000000 to 0x0110ffff)
	if (keysym & 0xff000000) == 0x01000000 {
		cp := rune(keysym & 0x00ffffff)
		if cp >= 0x20 && cp <= 0x10ffff {
			return cp, true
		}
	}

	// Standard X11 Arabic / Persian keysyms (0x05ac - 0x05fa)
	if keysym >= 0x05ac && keysym <= 0x05fa {
		switch keysym {
		case 0x05ac:
			return '\u060C', true // Arabic comma
		case 0x05bb:
			return '\u061B', true // Arabic semicolon
		case 0x05bf:
			return '\u061F', true // Arabic question mark
		default:
			if keysym >= 0x05c1 && keysym <= 0x05fa {
				return rune(0x0621 + (keysym - 0x05c1)), true
			}
		}
	}

	// Standard X11 Cyrillic keysyms (0x06a0 - 0x06ff)
	if keysym >= 0x06a0 && keysym <= 0x06ff {
		return rune(0x0400 + (keysym - 0x06a0)), true
	}

	// Standard X11 Greek keysyms (0x07a1 - 0x07fe)
	if keysym >= 0x07a1 && keysym <= 0x07fe {
		return rune(0x0380 + (keysym - 0x07a1)), true
	}

	// Unicode in standard range >= 0x0100 (excluding standard X11 special function keys 0xff00-0xffff)
	// Modern XKB layouts (including Persian, Arabic, Cyrillic) often map directly to Unicode codepoints
	if keysym >= 0x0100 && keysym <= 0x10ffff && (keysym < 0xff00 || keysym > 0xffff) {
		return rune(keysym), true
	}

	return 0, false
}
