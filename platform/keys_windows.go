//go:build windows

package platform

import (
	"unicode/utf8"
)

// Windows Virtual Key codes
const (
	VK_BACK    = 0x08
	VK_TAB     = 0x09
	VK_RETURN  = 0x0D
	VK_SHIFT   = 0x10
	VK_CONTROL = 0x11
	VK_MENU    = 0x12 // Alt
	VK_PAUSE   = 0x13
	VK_CAPITAL = 0x14
	VK_ESCAPE  = 0x1B
	VK_SPACE   = 0x20
	VK_PRIOR   = 0x21 // Page Up
	VK_NEXT    = 0x22 // Page Down
	VK_END     = 0x23
	VK_HOME    = 0x24
	VK_LEFT    = 0x25
	VK_UP      = 0x26
	VK_RIGHT   = 0x27
	VK_DOWN    = 0x28
	VK_INSERT  = 0x2D
	VK_DELETE  = 0x2E
	VK_F1      = 0x70
	VK_F2      = 0x71
	VK_F3      = 0x72
	VK_F4      = 0x73
	VK_F5      = 0x74
	VK_F6      = 0x75
	VK_F7      = 0x76
	VK_F8      = 0x77
	VK_F9      = 0x78
	VK_F10     = 0x79
	VK_F11     = 0x7A
	VK_F12     = 0x7B
)

type KeyHandler struct{}

func NewKeyHandler() *KeyHandler {
	return &KeyHandler{}
}

func (kh *KeyHandler) LookupAction(vk uint32, state uint16) ActionType {
	isCtrl := (state & ModCtrl) != 0
	isShift := (state & ModShift) != 0
	isAlt := (state & ModAlt) != 0

	// Ctrl+Shift Shortcuts
	if isCtrl && isShift {
		switch vk {
		case 'C':
			return ActionCopy
		case 'V':
			return ActionPaste
		case 'A':
			return ActionSelectAll
		case 'T':
			return ActionNewTab
		case 'W':
			return ActionCloseTab
		case 'F':
			return ActionSearch
		case 'D':
			return ActionToggleDiagnostics
		case VK_TAB:
			return ActionPrevTab
		case VK_UP:
			return ActionScrollUp
		case VK_DOWN:
			return ActionScrollDown
		case VK_PRIOR:
			return ActionScrollTop
		case VK_NEXT:
			return ActionScrollBottom
		case VK_OEM_PLUS, '=':
			return ActionZoomIn
		case VK_OEM_MINUS, '_':
			return ActionZoomOut
		}
	}

	// Alt-only shortcuts (Alt+1..9 for tab switching like Linux)
	if isAlt && !isCtrl && !isShift {
		if vk >= '1' && vk <= '9' {
			return ActionSwitchTab1 + ActionType(vk-'1')
		}
	}

	// Ctrl-only shortcuts
	if isCtrl && !isShift && !isAlt {
		switch vk {
		case VK_OEM_COMMA, ',':
			return ActionPreferences
		case VK_TAB:
			return ActionNextTab
		case '0':
			return ActionZoomReset
		case '1':
			return ActionSwitchTab1
		case '2':
			return ActionSwitchTab2
		case '3':
			return ActionSwitchTab3
		case '4':
			return ActionSwitchTab4
		case '5':
			return ActionSwitchTab5
		case '6':
			return ActionSwitchTab6
		case '7':
			return ActionSwitchTab7
		case '8':
			return ActionSwitchTab8
		case '9':
			return ActionSwitchTab9
		}
	}

	// Shift+PageUp/PageDown
	if isShift && !isCtrl {
		switch vk {
		case VK_PRIOR:
			return ActionScrollTop
		case VK_NEXT:
			return ActionScrollBottom
		}
	}

	return ActionNone
}

func (kh *KeyHandler) LookupSpecialKey(vk uint32, state uint16) ([]byte, bool) {
	isCtrl := (state & ModCtrl) != 0
	isShift := (state & ModShift) != 0
	isAlt := (state & ModAlt) != 0

	prefix := ""
	if isAlt {
		prefix = "\x1b"
	}

	// VT sequences for special keys
	switch vk {
	case VK_RETURN:
		return []byte(prefix + "\r"), true
	case VK_BACK:
		if isCtrl {
			return []byte("\x17"), true // Ctrl+Backspace: delete word (WERASE)
		}
		return []byte(prefix + "\x7f"), true
	case VK_TAB:
		if isShift {
			return []byte("\x1b[Z"), true // Backtab
		}
		return []byte("\t"), true
	case VK_ESCAPE:
		return []byte("\x1b"), true
	case VK_UP:
		if isCtrl {
			return []byte("\x1b[1;5A"), true
		}
		return []byte(prefix + "\x1b[A"), true
	case VK_DOWN:
		if isCtrl {
			return []byte("\x1b[1;5B"), true
		}
		return []byte(prefix + "\x1b[B"), true
	case VK_RIGHT:
		if isCtrl {
			return []byte("\x1b[1;5C"), true
		}
		return []byte(prefix + "\x1b[C"), true
	case VK_LEFT:
		if isCtrl {
			return []byte("\x1b[1;5D"), true
		}
		return []byte(prefix + "\x1b[D"), true
	case VK_HOME:
		if isCtrl {
			return []byte("\x1b[1;5H"), true
		}
		return []byte(prefix + "\x1b[H"), true
	case VK_END:
		if isCtrl {
			return []byte("\x1b[1;5F"), true
		}
		return []byte(prefix + "\x1b[F"), true
	case VK_INSERT:
		if isShift {
			return nil, false // Paste
		}
		return []byte("\x1b[2~"), true
	case VK_DELETE:
		if isCtrl {
			return []byte("\x1b[3;5~"), true
		}
		return []byte(prefix + "\x1b[3~"), true
	case VK_PRIOR: // Page Up
		return []byte(prefix + "\x1b[5~"), true
	case VK_NEXT: // Page Down
		return []byte(prefix + "\x1b[6~"), true
	case VK_F1:
		return []byte("\x1bOP"), true
	case VK_F2:
		return []byte("\x1bOQ"), true
	case VK_F3:
		return []byte("\x1bOR"), true
	case VK_F4:
		return []byte("\x1bOS"), true
	case VK_F5:
		return []byte("\x1b[15~"), true
	case VK_F6:
		return []byte("\x1b[17~"), true
	case VK_F7:
		return []byte("\x1b[18~"), true
	case VK_F8:
		return []byte("\x1b[19~"), true
	case VK_F9:
		return []byte("\x1b[20~"), true
	case VK_F10:
		return []byte("\x1b[21~"), true
	case VK_F11:
		return []byte("\x1b[22~"), true
	case VK_F12:
		return []byte("\x1b[24~"), true
	}

	// Handle Ctrl+letter (C-a through C-z)
	if isCtrl && !isShift && !isAlt && vk >= 'A' && vk <= 'Z' {
		return []byte{byte(vk - 'A' + 1)}, true
	}

	return nil, false
}

const (
	VK_OEM_PLUS  = 0xBB
	VK_OEM_COMMA = 0xBC
	VK_OEM_MINUS = 0xBD
)

// CharToBytes converts a typed rune to UTF-8 bytes
func CharToBytes(r rune) []byte {
	buf := make([]byte, utf8.UTFMax)
	n := utf8.EncodeRune(buf, r)
	return buf[:n]
}
