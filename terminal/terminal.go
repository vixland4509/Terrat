package terminal

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

type parserState int

const (
	stateNormal parserState = iota
	stateEscape
	stateCSI
	stateOSC
	stateOSCEscape
	stateCharset
)

type Terminal struct {
	mu sync.RWMutex

	cols int
	rows int

	lines      [][]Cell
	scrollback [][]Cell
	maxHistory int
	scrollOff  int

	altLines [][]Cell
	isAlt    bool

	bracketedPaste bool

	cursorX       int
	cursorY       int
	cursorVisible bool
	savedX        int
	savedY        int
	savedFG       Color
	savedBG       Color
	savedBold     bool
	savedUnderline bool
	savedInverse  bool

	currFG        Color
	currBG        Color
	currBold      bool
	currUnderline bool
	currInverse   bool

	scrollTop    int
	scrollBottom int

	appCursor     bool
	mouseTracking bool
	mouseSGR      bool

	state      parserState
	csiParams  []string
	csiPrivate bool
	csiLeader  rune
	oscBuffer  []rune

	title        string
	TitleChan    chan string
	ResponseChan chan []byte

	sel Selection

	theme *Theme
}

func New(cols, rows int) *Terminal {
	th := ThemeTokyoNight
	t := &Terminal{
		cols:          cols,
		rows:          rows,
		maxHistory:    5000,
		cursorVisible: true,
		theme:         th,
		currFG:        ColorDefaultFG,
		currBG:        ColorDefaultBG,
		savedFG:       ColorDefaultFG,
		savedBG:       ColorDefaultBG,
		scrollTop:     0,
		scrollBottom:  rows - 1,
		TitleChan:     make(chan string, 16),
		ResponseChan:  make(chan []byte, 16),
	}

	t.lines = make([][]Cell, rows)
	t.altLines = make([][]Cell, rows)
	for i := 0; i < rows; i++ {
		t.lines[i] = t.makeRow()
		t.altLines[i] = t.makeRow()
	}

	return t
}

func (t *Terminal) emptyCell() Cell {
	return Cell{
		Char: ' ',
		FG:   t.currFG,
		BG:   t.currBG,
	}
}

func (t *Terminal) emptyRow() []Cell {
	row := make([]Cell, t.cols)
	cell := t.emptyCell()
	for i := range row {
		row[i] = cell
	}
	return row
}

func (t *Terminal) makeRow() []Cell {
	row := make([]Cell, t.cols)
	for i := range row {
		row[i] = EmptyCell()
	}
	return row
}

func (t *Terminal) RLock()   { t.mu.RLock() }
func (t *Terminal) RUnlock() { t.mu.RUnlock() }

func (t *Terminal) Lock()   { t.mu.Lock() }
func (t *Terminal) Unlock() { t.mu.Unlock() }

func (t *Terminal) Cursor() (int, int, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.CursorLocked()
}

func (t *Terminal) CursorLocked() (int, int, bool) {
	return t.cursorX, t.cursorY, t.cursorVisible
}

func (t *Terminal) GetCell(x, y int) Cell {
	if y < 0 || y >= t.rows || x < 0 || x >= t.cols {
		return EmptyCell()
	}

	if t.scrollOff > 0 && !t.isAlt {
		targetIdx := (len(t.scrollback) - t.scrollOff) + y
		if targetIdx < 0 {
			return EmptyCell()
		}
		if targetIdx < len(t.scrollback) {
			if x < len(t.scrollback[targetIdx]) {
				return t.scrollback[targetIdx][x]
			}
			return EmptyCell()
		}
		gridRow := targetIdx - len(t.scrollback)
		grid := t.activeGrid()
		if gridRow >= 0 && gridRow < len(grid) && x < len(grid[gridRow]) {
			return grid[gridRow][x]
		}
		return EmptyCell()
	}

	grid := t.activeGrid()
	if y < len(grid) && x < len(grid[y]) {
		return grid[y][x]
	}
	return EmptyCell()
}

func (t *Terminal) IsAlt() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isAlt
}

func (t *Terminal) IsAltLocked() bool {
	return t.isAlt
}

func (t *Terminal) BracketedPaste() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.bracketedPaste
}

func (t *Terminal) AppCursorLocked() bool {
	return t.appCursor
}

func (t *Terminal) MouseTrackingLocked() bool {
	return t.mouseTracking
}

func (t *Terminal) MouseSGRLocked() bool {
	return t.mouseSGR
}

func (t *Terminal) ScrollOff() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.scrollOff
}

func (t *Terminal) ScrollOffLocked() int {
	return t.scrollOff
}

func (t *Terminal) ScrollbackLen() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.scrollback)
}

func (t *Terminal) ScrollbackLenLocked() int {
	return len(t.scrollback)
}

func (t *Terminal) Rows() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.rows
}

func (t *Terminal) GetRowString(y int) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if y < 0 || y >= t.rows {
		return ""
	}

	runes := make([]rune, t.cols)
	for x := 0; x < t.cols; x++ {
		cell := t.GetCell(x, y)
		r := cell.Char
		if r == 0 {
			r = ' '
		}
		runes[x] = r
	}
	return string(runes)
}

func (t *Terminal) Theme() *Theme {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.ThemeLocked()
}

func (t *Terminal) ThemeLocked() *Theme {
	if t.theme == nil {
		return ThemeTokyoNight
	}
	return t.theme
}

func (t *Terminal) SetTheme(theme *Theme) {
	if theme == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	oldTheme := t.theme
	t.theme = theme

	t.currFG = ColorDefaultFG
	t.currBG = ColorDefaultBG

	remapCell := func(cell *Cell) {
		if cell.BG == ColorDefaultBG || (oldTheme != nil && cell.BG == oldTheme.BG) {
			cell.BG = ColorDefaultBG
		}
		if cell.FG == ColorDefaultFG || (oldTheme != nil && cell.FG == oldTheme.FG) {
			cell.FG = ColorDefaultFG
		}
		if oldTheme != nil {
			for i := 0; i < 16; i++ {
				if cell.FG == oldTheme.ANSI[i] {
					cell.FG = theme.ANSI[i]
					break
				}
			}
			for i := 0; i < 16; i++ {
				if cell.BG == oldTheme.ANSI[i] {
					cell.BG = theme.ANSI[i]
					break
				}
			}
		}
	}

	for r := range t.lines {
		for c := range t.lines[r] {
			remapCell(&t.lines[r][c])
		}
	}
	for r := range t.altLines {
		for c := range t.altLines[r] {
			remapCell(&t.altLines[r][c])
		}
	}
	for r := range t.scrollback {
		for c := range t.scrollback[r] {
			remapCell(&t.scrollback[r][c])
		}
	}
}

func (t *Terminal) themeANSI(idx int) Color {
	if t.theme != nil && idx >= 0 && idx < 16 {
		return t.theme.ANSI[idx]
	}
	if idx >= 0 && idx < 16 {
		return ansi16[idx]
	}
	return ColorDefaultFG
}

func (t *Terminal) activeGrid() [][]Cell {
	if t.isAlt {
		return t.altLines
	}
	return t.lines
}

func (t *Terminal) Scroll(delta int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isAlt {
		return
	}

	t.scrollOff += delta
	if t.scrollOff > len(t.scrollback) {
		t.scrollOff = len(t.scrollback)
	}
	if t.scrollOff < 0 {
		t.scrollOff = 0
	}
	if t.sel.Mode != SelectionModeAll {
		t.sel.Active = false
	}
}

func (t *Terminal) ScrollKeepSelection(delta int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isAlt {
		return
	}

	if t.sel.Mode == SelectionModeAll {
		return
	}

	oldOff := t.scrollOff
	t.scrollOff += delta
	if t.scrollOff > len(t.scrollback) {
		t.scrollOff = len(t.scrollback)
	}
	if t.scrollOff < 0 {
		t.scrollOff = 0
	}

	actualDelta := t.scrollOff - oldOff
	if actualDelta != 0 && t.sel.Active {
		t.sel.StartY += actualDelta
		t.sel.OrigStartY += actualDelta
		if t.sel.StartY >= t.rows {
			t.sel.StartY = t.rows - 1
		}
		if t.sel.StartY < 0 {
			t.sel.StartY = 0
		}
		if t.sel.OrigStartY >= t.rows {
			t.sel.OrigStartY = t.rows - 1
		}
		if t.sel.OrigStartY < 0 {
			t.sel.OrigStartY = 0
		}
	}
}

func (t *Terminal) SetScrollOff(off int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isAlt {
		return
	}

	t.scrollOff = off
	if t.scrollOff > len(t.scrollback) {
		t.scrollOff = len(t.scrollback)
	}
	if t.scrollOff < 0 {
		t.scrollOff = 0
	}
}

func (t *Terminal) ScrollToTop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isAlt {
		return
	}

	t.scrollOff = len(t.scrollback)
	if t.sel.Mode != SelectionModeAll {
		t.sel.Active = false
	}
}

func (t *Terminal) ResetScroll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.scrollOff != 0 {
		t.scrollOff = 0
		if t.sel.Mode != SelectionModeAll {
			t.sel.Active = false
		}
	}
}

func (t *Terminal) Resize(newCols, newRows int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if newCols == t.cols && newRows == t.rows {
		return
	}
	if newCols <= 0 || newRows <= 0 {
		return
	}

	newLines := make([][]Cell, newRows)
	newAlt := make([][]Cell, newRows)

	for r := 0; r < newRows; r++ {
		newLines[r] = make([]Cell, newCols)
		newAlt[r] = make([]Cell, newCols)

		for c := 0; c < newCols; c++ {
			if r < len(t.lines) && c < len(t.lines[r]) {
				newLines[r][c] = t.lines[r][c]
			} else {
				newLines[r][c] = EmptyCell()
			}

			if r < len(t.altLines) && c < len(t.altLines[r]) {
				newAlt[r][c] = t.altLines[r][c]
			} else {
				newAlt[r][c] = EmptyCell()
			}
		}
	}

	t.cols = newCols
	t.rows = newRows
	t.lines = newLines
	t.altLines = newAlt

	t.scrollTop = 0
	t.scrollBottom = newRows - 1

	if t.cursorY >= newRows {
		t.cursorY = newRows - 1
	}
	if t.cursorX >= newCols {
		t.cursorX = newCols - 1
	}
	t.sel.Active = false
}

func (t *Terminal) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	idx := 0
	for idx < len(p) {
		r, size := utf8.DecodeRune(p[idx:])
		idx += size
		t.processRune(r)
	}

	return len(p), nil
}

func (t *Terminal) processRune(r rune) {
	switch t.state {
	case stateNormal:
		switch r {
		case '\x1b':
			t.state = stateEscape
		case '\n':
			t.lineFeed()
		case '\r':
			t.cursorX = 0
		case '\b':
			if t.cursorX > 0 {
				t.cursorX--
			}
		case '\t':
			t.cursorX = (t.cursorX/8 + 1) * 8
			if t.cursorX >= t.cols {
				t.cursorX = t.cols - 1
			}
		case '\a':
		case '\x00':
		default:
			if r >= ' ' {
				t.putChar(r)
			}
		}

	case stateEscape:
		switch r {
		case '[':
			t.state = stateCSI
			t.csiParams = nil
			t.csiPrivate = false
			t.csiLeader = 0
		case ']':
			t.state = stateOSC
			t.oscBuffer = nil
		case '(', ')', '*', '+':
			t.state = stateCharset
		case '7':
			t.saveCursor()
			t.state = stateNormal
		case '8':
			t.restoreCursor()
			t.state = stateNormal
		case 'M':
			t.reverseIndex()
			t.state = stateNormal
		case '=':
			t.state = stateNormal
		case '>':
			t.state = stateNormal
		default:
			t.state = stateNormal
		}

	case stateCharset:
		t.state = stateNormal

	case stateCSI:
		if r == '?' || r == '>' || r == '=' {
			t.csiLeader = r
			if r == '?' {
				t.csiPrivate = true
			}
			return
		}
		if (r >= '0' && r <= '9') || r == ';' {
			if len(t.csiParams) == 0 {
				if r == ';' {
					t.csiParams = []string{"", ""}
				} else {
					t.csiParams = []string{string(r)}
				}
			} else {
				last := len(t.csiParams) - 1
				if r == ';' {
					t.csiParams = append(t.csiParams, "")
				} else {
					t.csiParams[last] += string(r)
				}
			}
			return
		}

		t.executeCSI(r)
		t.state = stateNormal

	case stateOSC:
		if r == '\a' {
			t.executeOSC()
			t.state = stateNormal
		} else if r == '\x1b' {
			t.state = stateOSCEscape
		} else {
			if len(t.oscBuffer) < 4096 {
				t.oscBuffer = append(t.oscBuffer, r)
			}
		}

	case stateOSCEscape:
		if r == '\\' {
			t.executeOSC()
			t.state = stateNormal
		} else {
			t.executeOSC()
			t.state = stateEscape
			t.processRune(r)
		}
	}
}

func (t *Terminal) putChar(r rune) {
	if t.cursorX >= t.cols {
		t.cursorX = 0
		t.lineFeed()
	}

	grid := t.activeGrid()
	if t.cursorY >= 0 && t.cursorY < t.rows && t.cursorX >= 0 && t.cursorX < t.cols {
		grid[t.cursorY][t.cursorX] = Cell{
			Char:      r,
			FG:        t.currFG,
			BG:        t.currBG,
			Bold:      t.currBold,
			Underline: t.currUnderline,
			Inverse:   t.currInverse,
		}
	}

	t.cursorX++
}

func (t *Terminal) lineFeed() {
	if t.cursorY >= t.scrollBottom {
		t.scrollUp(1)
	} else {
		t.cursorY++
	}
}

func (t *Terminal) scrollUp(count int) {
	grid := t.activeGrid()

	for c := 0; c < count; c++ {
		if t.scrollTop == 0 && !t.isAlt {
			saved := make([]Cell, t.cols)
			copy(saved, grid[0])
			t.scrollback = append(t.scrollback, saved)
			if len(t.scrollback) > t.maxHistory {
				t.scrollback[0] = nil
				t.scrollback = t.scrollback[1:]
			} else if t.scrollOff > 0 {
				t.scrollOff++
			}
		}

		for r := t.scrollTop; r < t.scrollBottom; r++ {
			grid[r] = grid[r+1]
		}

		grid[t.scrollBottom] = t.makeRow()
	}
	if t.scrollOff > len(t.scrollback) {
		t.scrollOff = len(t.scrollback)
	}
}

func (t *Terminal) reverseIndex() {
	if t.cursorY <= t.scrollTop {
		grid := t.activeGrid()
		for r := t.scrollBottom; r > t.scrollTop; r-- {
			grid[r] = grid[r-1]
		}
		grid[t.scrollTop] = t.makeRow()
	} else {
		t.cursorY--
	}
}

func (t *Terminal) saveCursor() {
	t.savedX = t.cursorX
	t.savedY = t.cursorY
	t.savedFG = t.currFG
	t.savedBG = t.currBG
	t.savedBold = t.currBold
	t.savedUnderline = t.currUnderline
	t.savedInverse = t.currInverse
}

func (t *Terminal) restoreCursor() {
	t.cursorX = t.savedX
	t.cursorY = t.savedY
	if t.cursorX >= t.cols {
		t.cursorX = t.cols - 1
	}
	if t.cursorX < 0 {
		t.cursorX = 0
	}
	if t.cursorY >= t.rows {
		t.cursorY = t.rows - 1
	}
	if t.cursorY < 0 {
		t.cursorY = 0
	}
	t.currFG = t.savedFG
	t.currBG = t.savedBG
	t.currBold = t.savedBold
	t.currUnderline = t.savedUnderline
	t.currInverse = t.savedInverse
}

func (t *Terminal) executeCSI(cmd rune) {
	args := make([]int, len(t.csiParams))
	for i, p := range t.csiParams {
		val, _ := strconv.Atoi(p)
		args[i] = val
	}

	arg := func(idx, defaultVal int) int {
		if idx < len(args) && args[idx] > 0 {
			return args[idx]
		}
		return defaultVal
	}

	grid := t.activeGrid()

	switch cmd {
	case 'A':
		n := arg(0, 1)
		t.cursorY -= n
		if t.cursorY < 0 {
			t.cursorY = 0
		}

	case 'B':
		n := arg(0, 1)
		t.cursorY += n
		if t.cursorY >= t.rows {
			t.cursorY = t.rows - 1
		}

	case 'C':
		n := arg(0, 1)
		t.cursorX += n
		if t.cursorX >= t.cols {
			t.cursorX = t.cols - 1
		}

	case 'D':
		n := arg(0, 1)
		t.cursorX -= n
		if t.cursorX < 0 {
			t.cursorX = 0
		}

	case 'E':
		n := arg(0, 1)
		t.cursorY += n
		if t.cursorY >= t.rows {
			t.cursorY = t.rows - 1
		}
		t.cursorX = 0

	case 'F':
		n := arg(0, 1)
		t.cursorY -= n
		if t.cursorY < 0 {
			t.cursorY = 0
		}
		t.cursorX = 0

	case 'G':
		col := arg(0, 1) - 1
		if col < 0 {
			col = 0
		}
		if col >= t.cols {
			col = t.cols - 1
		}
		t.cursorX = col

	case 'H', 'f':
		row := arg(0, 1) - 1
		col := arg(1, 1) - 1
		if row < 0 {
			row = 0
		}
		if row >= t.rows {
			row = t.rows - 1
		}
		if col < 0 {
			col = 0
		}
		if col >= t.cols {
			col = t.cols - 1
		}
		t.cursorX = col
		t.cursorY = row

	case 'J':
		mode := 0
		if len(args) > 0 {
			mode = args[0]
		}
		switch mode {
		case 0:
			for c := t.cursorX; c < t.cols; c++ {
				grid[t.cursorY][c] = t.emptyCell()
			}
			for r := t.cursorY + 1; r < t.rows; r++ {
				grid[r] = t.emptyRow()
			}
		case 1:
			for r := 0; r < t.cursorY; r++ {
				grid[r] = t.emptyRow()
			}
			for c := 0; c <= t.cursorX && c < t.cols; c++ {
				grid[t.cursorY][c] = t.emptyCell()
			}
		case 2:
			for r := 0; r < t.rows; r++ {
				grid[r] = t.emptyRow()
			}
		case 3:
			t.scrollback = nil
			t.scrollOff = 0
		}

	case 'K':
		mode := 0
		if len(args) > 0 {
			mode = args[0]
		}
		switch mode {
		case 0:
			for c := t.cursorX; c < t.cols; c++ {
				grid[t.cursorY][c] = t.emptyCell()
			}
		case 1:
			for c := 0; c <= t.cursorX && c < t.cols; c++ {
				grid[t.cursorY][c] = t.emptyCell()
			}
		case 2:
			for c := 0; c < t.cols; c++ {
				grid[t.cursorY][c] = t.emptyCell()
			}
		}

	case 'd':
		row := arg(0, 1) - 1
		if row < 0 {
			row = 0
		}
		if row >= t.rows {
			row = t.rows - 1
		}
		t.cursorY = row

	case 'S':
		n := arg(0, 1)
		t.scrollUp(n)

	case 'T':
		n := arg(0, 1)
		for i := 0; i < n; i++ {
			for r := t.scrollBottom; r > t.scrollTop; r-- {
				grid[r] = grid[r-1]
			}
			grid[t.scrollTop] = t.emptyRow()
		}

	case '@':
		n := arg(0, 1)
		row := grid[t.cursorY]
		for c := t.cols - 1; c >= t.cursorX; c-- {
			if c-n >= t.cursorX {
				row[c] = row[c-n]
			} else {
				row[c] = t.emptyCell()
			}
		}

	case 'X':
		n := arg(0, 1)
		row := grid[t.cursorY]
		for c := t.cursorX; c < t.cursorX+n && c < t.cols; c++ {
			row[c] = t.emptyCell()
		}

	case 's':
		t.saveCursor()

	case 'u':
		t.restoreCursor()

	case 'L':
		if t.cursorY >= t.scrollTop && t.cursorY <= t.scrollBottom {
			n := arg(0, 1)
			for i := 0; i < n; i++ {
				for r := t.scrollBottom; r > t.cursorY; r-- {
					grid[r] = grid[r-1]
				}
				grid[t.cursorY] = t.emptyRow()
			}
		}

	case 'M':
		if t.cursorY >= t.scrollTop && t.cursorY <= t.scrollBottom {
			n := arg(0, 1)
			for i := 0; i < n; i++ {
				for r := t.cursorY; r < t.scrollBottom; r++ {
					grid[r] = grid[r+1]
				}
				grid[t.scrollBottom] = t.emptyRow()
			}
		}

	case 'P':
		n := arg(0, 1)
		row := grid[t.cursorY]
		for c := t.cursorX; c < t.cols; c++ {
			if c+n < t.cols {
				row[c] = row[c+n]
			} else {
				row[c] = t.emptyCell()
			}
		}

	case 'r':
		top := arg(0, 1) - 1
		bottom := arg(1, t.rows) - 1
		if top >= 0 && bottom < t.rows && top < bottom {
			t.scrollTop = top
			t.scrollBottom = bottom
		} else {
			t.scrollTop = 0
			t.scrollBottom = t.rows - 1
		}
		t.cursorX = 0
		t.cursorY = 0

	case 'm':
		t.executeSGR(args)

	case 'n':
		mode := arg(0, 0)
		if mode == 6 {
			resp := fmt.Sprintf("\x1b[%d;%dR", t.cursorY+1, t.cursorX+1)
			select {
			case t.ResponseChan <- []byte(resp):
			default:
			}
		} else if mode == 5 {
			select {
			case t.ResponseChan <- []byte("\x1b[0n"):
			default:
			}
		}

	case 'c':
		if t.csiLeader == '>' {
			select {
			case t.ResponseChan <- []byte("\x1b[>0;10;1c"):
			default:
			}
		} else {
			select {
			case t.ResponseChan <- []byte("\x1b[?62;c"):
			default:
			}
		}

	case 'h', 'l':
		enable := (cmd == 'h')
		if t.csiPrivate {
			for _, mode := range args {
				switch mode {
				case 1:
					t.appCursor = enable
				case 25:
					t.cursorVisible = enable
				case 1047, 47:
					t.isAlt = enable
					t.scrollTop = 0
					t.scrollBottom = t.rows - 1
					t.scrollOff = 0
				case 1048:
					if enable {
						t.saveCursor()
					} else {
						t.restoreCursor()
					}
				case 1049:
					if enable {
						t.saveCursor()
						t.isAlt = true
						for r := 0; r < t.rows; r++ {
							t.altLines[r] = t.makeRow()
						}
						t.scrollTop = 0
						t.scrollBottom = t.rows - 1
						t.scrollOff = 0
					} else {
						t.isAlt = false
						t.appCursor = false
						t.mouseTracking = false
						t.mouseSGR = false
						t.restoreCursor()
						t.scrollTop = 0
						t.scrollBottom = t.rows - 1
						t.scrollOff = 0
					}
				case 1000, 1002:
					t.mouseTracking = enable
				case 1006:
					t.mouseSGR = enable
				case 2004:
					t.bracketedPaste = enable
				}
			}
		}
	}
}

func (t *Terminal) executeSGR(args []int) {
	if len(args) == 0 {
		args = []int{0}
	}

	for i := 0; i < len(args); i++ {
		code := args[i]
		switch {
		case code == 0:
			t.currFG = ColorDefaultFG
			t.currBG = ColorDefaultBG
			t.currBold = false
			t.currUnderline = false
			t.currInverse = false

		case code == 1:
			t.currBold = true
		case code == 4:
			t.currUnderline = true
		case code == 7:
			t.currInverse = true
		case code == 22:
			t.currBold = false
		case code == 24:
			t.currUnderline = false
		case code == 27:
			t.currInverse = false

		case code >= 30 && code <= 37:
			t.currFG = t.themeANSI(code - 30)
		case code == 39:
			t.currFG = ColorDefaultFG

		case code >= 40 && code <= 47:
			t.currBG = t.themeANSI(code - 40)
		case code == 49:
			t.currBG = ColorDefaultBG

		case code >= 90 && code <= 97:
			t.currFG = t.themeANSI(code - 90 + 8)
		case code >= 100 && code <= 107:
			t.currBG = t.themeANSI(code - 100 + 8)

		case code == 38:
			if i+2 < len(args) && args[i+1] == 5 {
				t.currFG = ANSI256(args[i+2])
				i += 2
			} else if i+4 < len(args) && args[i+1] == 2 {
				t.currFG = RGB(byte(args[i+2]), byte(args[i+3]), byte(args[i+4]))
				i += 4
			}

		case code == 48:
			if i+2 < len(args) && args[i+1] == 5 {
				t.currBG = ANSI256(args[i+2])
				i += 2
			} else if i+4 < len(args) && args[i+1] == 2 {
				t.currBG = RGB(byte(args[i+2]), byte(args[i+3]), byte(args[i+4]))
				i += 4
			}
		}
	}
}

func (t *Terminal) executeOSC() {
	content := string(t.oscBuffer)
	parts := strings.SplitN(content, ";", 2)
	if len(parts) == 2 {
		cmd := parts[0]
		val := parts[1]
		if cmd == "0" || cmd == "2" {
			t.title = val
			select {
			case t.TitleChan <- val:
			default:
			}
		}
	}
}
