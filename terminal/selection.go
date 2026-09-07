package terminal

import (
	"strings"
	"unicode"
)

type SelectionMode int

const (
	SelectionModeNone SelectionMode = iota
	SelectionModeNormal
	SelectionModeWord
	SelectionModeLine
)

type Selection struct {
	Active     bool
	Mode       SelectionMode
	StartX     int
	StartY     int
	EndX       int
	EndY       int
	OrigStartX int
	OrigStartY int
}

func (s *Selection) Normalized() (sX, sY, eX, eY int) {
	if s.StartY < s.EndY || (s.StartY == s.EndY && s.StartX <= s.EndX) {
		return s.StartX, s.StartY, s.EndX, s.EndY
	}
	return s.EndX, s.EndY, s.StartX, s.StartY
}

func (s *Selection) HasSelection() bool {
	if !s.Active {
		return false
	}
	if s.Mode == SelectionModeWord || s.Mode == SelectionModeLine {
		return true
	}
	return s.StartX != s.EndX || s.StartY != s.EndY
}

func (s *Selection) IsSelected(x, y int) bool {
	if !s.HasSelection() {
		return false
	}
	sX, sY, eX, eY := s.Normalized()
	if y < sY || y > eY {
		return false
	}
	if sY == eY {
		return x >= sX && x <= eX
	}
	if y == sY {
		return x >= sX
	}
	if y == eY {
		return x <= eX
	}
	return true
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' || r == '/' || r == ':' || r == '~' || r == '@'
}

func (t *Terminal) StartSelection(x, y int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if x < 0 {
		x = 0
	}
	if x >= t.cols {
		x = t.cols - 1
	}
	if y < 0 {
		y = 0
	}
	if y >= t.rows {
		y = t.rows - 1
	}

	t.sel = Selection{
		Active:     true,
		Mode:       SelectionModeNormal,
		StartX:     x,
		StartY:     y,
		EndX:       x,
		EndY:       y,
		OrigStartX: x,
		OrigStartY: y,
	}
	t.dirtyAll = true
}

func (t *Terminal) UpdateSelection(x, y int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.sel.Active {
		return
	}

	if x < 0 {
		x = 0
	}
	if x >= t.cols {
		x = t.cols - 1
	}
	if y < 0 {
		y = 0
	}
	if y >= t.rows {
		y = t.rows - 1
	}

	if t.sel.Mode == SelectionModeWord {
		wLeft, wRight := t.findWordBounds(x, y)
		if y < t.sel.OrigStartY || (y == t.sel.OrigStartY && x < t.sel.OrigStartX) {
			t.sel.StartX = wLeft
			t.sel.StartY = y
			origLeft, _ := t.findWordBounds(t.sel.OrigStartX, t.sel.OrigStartY)
			t.sel.EndX = origLeft
			t.sel.EndY = t.sel.OrigStartY
		} else {
			origLeft, _ := t.findWordBounds(t.sel.OrigStartX, t.sel.OrigStartY)
			t.sel.StartX = origLeft
			t.sel.StartY = t.sel.OrigStartY
			t.sel.EndX = wRight
			t.sel.EndY = y
		}
	} else if t.sel.Mode == SelectionModeLine {
		if y < t.sel.OrigStartY {
			t.sel.StartX = 0
			t.sel.StartY = y
			t.sel.EndX = t.cols - 1
			t.sel.EndY = t.sel.OrigStartY
		} else {
			t.sel.StartX = 0
			t.sel.StartY = t.sel.OrigStartY
			t.sel.EndX = t.cols - 1
			t.sel.EndY = y
		}
	} else {
		t.sel.EndX = x
		t.sel.EndY = y
	}

	t.dirtyAll = true
}

func (t *Terminal) SelectWord(x, y int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if y < 0 || y >= t.rows || x < 0 || x >= t.cols {
		return
	}

	wLeft, wRight := t.findWordBounds(x, y)
	t.sel = Selection{
		Active:     true,
		Mode:       SelectionModeWord,
		StartX:     wLeft,
		StartY:     y,
		EndX:       wRight,
		EndY:       y,
		OrigStartX: x,
		OrigStartY: y,
	}
	t.dirtyAll = true
}

func (t *Terminal) SelectLine(y int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if y < 0 || y >= t.rows {
		return
	}

	t.sel = Selection{
		Active:     true,
		Mode:       SelectionModeLine,
		StartX:     0,
		StartY:     y,
		EndX:       t.cols - 1,
		EndY:       y,
		OrigStartX: 0,
		OrigStartY: y,
	}
	t.dirtyAll = true
}

func (t *Terminal) findWordBounds(x, y int) (int, int) {
	if y < 0 || y >= t.rows || x < 0 || x >= t.cols {
		return x, x
	}

	targetRune := t.GetCell(x, y).Char
	targetIsWord := isWordChar(targetRune)
	targetIsSpace := targetRune == 0 || targetRune == ' '

	left := x
	for left > 0 {
		prev := t.GetCell(left-1, y).Char
		if targetIsWord && isWordChar(prev) {
			left--
		} else if targetIsSpace && (prev == 0 || prev == ' ') {
			left--
		} else if !targetIsWord && !targetIsSpace && !isWordChar(prev) && prev != 0 && prev != ' ' {
			left--
		} else {
			break
		}
	}

	right := x
	for right < t.cols-1 {
		next := t.GetCell(right+1, y).Char
		if targetIsWord && isWordChar(next) {
			right++
		} else if targetIsSpace && (next == 0 || next == ' ') {
			right++
		} else if !targetIsWord && !targetIsSpace && !isWordChar(next) && next != 0 && next != ' ' {
			right++
		} else {
			break
		}
	}

	return left, right
}

func (t *Terminal) ClearSelection() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.sel.Active {
		t.sel.Active = false
		t.dirtyAll = true
	}
}

func (t *Terminal) HasSelection() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.sel.HasSelection()
}

func (t *Terminal) IsSelected(x, y int) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.sel.IsSelected(x, y)
}

func (t *Terminal) IsSelectedLocked(x, y int) bool {
	return t.sel.IsSelected(x, y)
}

func (t *Terminal) GetSelectedText() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.sel.HasSelection() {
		return ""
	}

	sX, sY, eX, eY := t.sel.Normalized()

	var sb strings.Builder
	for y := sY; y <= eY; y++ {
		if y < 0 || y >= t.rows {
			continue
		}

		colStart := 0
		if y == sY {
			colStart = sX
		}
		colEnd := t.cols - 1
		if y == eY {
			colEnd = eX
		}

		if colStart < 0 {
			colStart = 0
		}
		if colEnd >= t.cols {
			colEnd = t.cols - 1
		}

		var lineRunes []rune
		for x := colStart; x <= colEnd; x++ {
			r := t.GetCell(x, y).Char
			if r == 0 {
				r = ' '
			}
			lineRunes = append(lineRunes, r)
		}

		lineStr := string(lineRunes)
		if colEnd == t.cols-1 {
			lineStr = strings.TrimRight(lineStr, " ")
		}

		sb.WriteString(lineStr)
		if y < eY {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}
