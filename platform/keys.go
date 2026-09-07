package platform

import (
	"unicode/utf8"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

const (
	ModShift = 1 << 0
	ModLock  = 1 << 1
	ModCtrl  = 1 << 2
	ModAlt   = 1 << 3
)

type KeyHandler struct {
	minKeycode        xproto.Keycode
	maxKeycode        xproto.Keycode
	keysymsPerKeycode byte
	keysyms           []xproto.Keysym
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

	return &KeyHandler{
		minKeycode:        minKey,
		maxKeycode:        maxKey,
		keysymsPerKeycode: mapping.KeysymsPerKeycode,
		keysyms:           mapping.Keysyms,
	}, nil
}

type ActionType int

const (
	ActionNone ActionType = iota
	ActionScrollUp
	ActionScrollDown
	ActionScrollTop
	ActionScrollBottom
	ActionCopy
	ActionPaste
	ActionSelectAll
	ActionPreferences
	ActionZoomIn
	ActionZoomOut
	ActionZoomReset
	ActionNewTab
	ActionCloseTab
	ActionNextTab
	ActionPrevTab
	ActionSearch
	ActionSwitchTab1
	ActionSwitchTab2
	ActionSwitchTab3
	ActionSwitchTab4
	ActionSwitchTab5
	ActionSwitchTab6
	ActionSwitchTab7
	ActionSwitchTab8
	ActionSwitchTab9
)

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

	keysym := kh.keysyms[idx+col]

	shiftCol := col + 1
	if shiftCol < int(kh.keysymsPerKeycode) && kh.keysyms[idx+shiftCol] != 0 {
		if isShift {
			keysym = kh.keysyms[idx+shiftCol]
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
	return nil
}

func (kh *KeyHandler) Translate(ev xproto.KeyPressEvent) ([]byte, ActionType) {
	keysym := kh.KeySym(ev)
	if keysym == 0 {
		return nil, ActionNone
	}

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
		return []byte("\x1b[H"), ActionNone
	case 0xff57:
		return []byte("\x1b[F"), ActionNone
	case 0xff55:
		return []byte("\x1b[5~"), ActionNone
	case 0xff56:
		return []byte("\x1b[6~"), ActionNone
	case 0xff52:
		return []byte("\x1b[A"), ActionNone
	case 0xff54:
		return []byte("\x1b[B"), ActionNone
	case 0xff53:
		return []byte("\x1b[C"), ActionNone
	case 0xff51:
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
