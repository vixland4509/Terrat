package terminal

import (
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

	cursorX       int
	cursorY       int
	cursorVisible bool
	savedX        int
	savedY        int

	currFG        Color
	currBG        Color
	currBold      bool
	currUnderline bool
	currInverse   bool

	scrollTop    int
	scrollBottom int

	state      parserState
	csiParams  []string
	csiPrivate bool
	oscBuffer  []rune

	title     string
	TitleChan chan string

	dirtyRows []bool
	dirtyAll  bool

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
		scrollTop:     0,
		scrollBottom:  rows - 1,
		TitleChan:     make(chan string, 16),
		dirtyRows:     make([]bool, rows),
		dirtyAll:      true,
	}

	t.lines = make([][]Cell, rows)
	t.altLines = make([][]Cell, rows)
	for i := 0; i < rows; i++ {
		t.lines[i] = t.makeRow()
		t.altLines[i] = t.makeRow()
		t.dirtyRows[i] = true
	}

	return t
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

func (t *Terminal) Size() (int, int) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.cols, t.rows
}

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


func (t *Terminal) Cols() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.cols
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

func (t *Terminal) IsRowDirty(y int) bool {
	if t.dirtyAll {
		return true
	}
	if y >= 0 && y < len(t.dirtyRows) {
		return t.dirtyRows[y]
	}
	return true
}

func (t *Terminal) ClearDirty() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.dirtyAll = false
	for i := range t.dirtyRows {
		t.dirtyRows[i] = false
	}
}

func (t *Terminal) MarkAllDirty() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.dirtyAll = true
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

	t.dirtyAll = true
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

func (t *Terminal) themeFG() Color {
	if t.theme != nil {
		return t.theme.FG
	}
	return Color(0xc0caf5)
}

func (t *Terminal) themeBG() Color {
	if t.theme != nil {
		return t.theme.BG
	}
	return Color(0x1a1b26)
}

func (t *Terminal) activeGrid() [][]Cell {
	if t.isAlt {
		return t.altLines
	}
	return t.lines
}

func (t *Terminal) markRowDirty(row int) {
	if row >= 0 && row < len(t.dirtyRows) {
		t.dirtyRows[row] = true
	}
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
	t.sel.Active = false
	t.dirtyAll = true
}

func (t *Terminal) ScrollToTop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isAlt {
		return
	}

	t.scrollOff = len(t.scrollback)
	t.sel.Active = false
	t.dirtyAll = true
}

func (t *Terminal) ResetScroll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.scrollOff != 0 {
		t.scrollOff = 0
		t.sel.Active = false
		t.dirtyAll = true
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
	newDirty := make([]bool, newRows)

	for r := 0; r < newRows; r++ {
		newDirty[r] = true
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
	t.dirtyRows = newDirty
	t.dirtyAll = true

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
		case ']':
			t.state = stateOSC
			t.oscBuffer = nil
		case '(':
			t.state = stateCharset
		case '7':
			t.savedX = t.cursorX
			t.savedY = t.cursorY
			t.state = stateNormal
		case '8':
			t.cursorX = t.savedX
			t.cursorY = t.savedY
			t.state = stateNormal
		case 'M':
			t.reverseIndex()
			t.state = stateNormal
		default:
			t.state = stateNormal
		}

	case stateCharset:
		t.state = stateNormal

	case stateCSI:
		if r == '?' {
			t.csiPrivate = true
			return
		}
		if (r >= '0' && r <= '9') || r == ';' {
			if len(t.csiParams) == 0 {
				t.csiParams = append(t.csiParams, string(r))
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
		if r == '\a' || r == '\x1b' {
			t.executeOSC()
			t.state = stateNormal
		} else {
			t.oscBuffer = append(t.oscBuffer, r)
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
		t.markRowDirty(t.cursorY)
	}

	t.cursorX++
}

func (t *Terminal) lineFeed() {
	if t.cursorY >= t.scrollBottom {
		t.scrollUp(1)
	} else {
		t.cursorY++
	}
	t.markRowDirty(t.cursorY)
}

func (t *Terminal) scrollUp(count int) {
	grid := t.activeGrid()

	for c := 0; c < count; c++ {
		if t.scrollTop == 0 && !t.isAlt {
			saved := make([]Cell, t.cols)
			copy(saved, grid[0])
			t.scrollback = append(t.scrollback, saved)
			if len(t.scrollback) > t.maxHistory {
				t.scrollback = t.scrollback[1:]
			} else if t.scrollOff > 0 {
				t.scrollOff++
			}
		}

		for r := t.scrollTop; r < t.scrollBottom; r++ {
			grid[r] = grid[r+1]
			t.markRowDirty(r)
		}

		grid[t.scrollBottom] = t.makeRow()
		t.markRowDirty(t.scrollBottom)
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
			t.markRowDirty(r)
		}
		grid[t.scrollTop] = t.makeRow()
		t.markRowDirty(t.scrollTop)
	} else {
		t.cursorY--
		t.markRowDirty(t.cursorY)
	}
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
		t.markRowDirty(t.cursorY)

	case 'B':
		n := arg(0, 1)
		t.cursorY += n
		if t.cursorY >= t.rows {
			t.cursorY = t.rows - 1
		}
		t.markRowDirty(t.cursorY)

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
		t.markRowDirty(t.cursorY)

	case 'J':
		mode := 0
		if len(args) > 0 {
			mode = args[0]
		}
		switch mode {
		case 0:
			for c := t.cursorX; c < t.cols; c++ {
				grid[t.cursorY][c] = EmptyCell()
			}
			t.markRowDirty(t.cursorY)
			for r := t.cursorY + 1; r < t.rows; r++ {
				grid[r] = t.makeRow()
				t.markRowDirty(r)
			}
		case 1:
			for r := 0; r < t.cursorY; r++ {
				grid[r] = t.makeRow()
				t.markRowDirty(r)
			}
			for c := 0; c <= t.cursorX && c < t.cols; c++ {
				grid[t.cursorY][c] = EmptyCell()
			}
			t.markRowDirty(t.cursorY)
		case 2:
			for r := 0; r < t.rows; r++ {
				grid[r] = t.makeRow()
				t.markRowDirty(r)
			}
			t.dirtyAll = true
		case 3:
			t.scrollback = nil
			t.scrollOff = 0
			t.dirtyAll = true
		}

	case 'K':
		mode := 0
		if len(args) > 0 {
			mode = args[0]
		}
		switch mode {
		case 0:
			for c := t.cursorX; c < t.cols; c++ {
				grid[t.cursorY][c] = EmptyCell()
			}
		case 1:
			for c := 0; c <= t.cursorX && c < t.cols; c++ {
				grid[t.cursorY][c] = EmptyCell()
			}
		case 2:
			grid[t.cursorY] = t.makeRow()
		}
		t.markRowDirty(t.cursorY)

	case 'L':
		n := arg(0, 1)
		for i := 0; i < n; i++ {
			for r := t.scrollBottom; r > t.cursorY; r-- {
				grid[r] = grid[r-1]
				t.markRowDirty(r)
			}
			grid[t.cursorY] = t.makeRow()
			t.markRowDirty(t.cursorY)
		}

	case 'M':
		n := arg(0, 1)
		for i := 0; i < n; i++ {
			for r := t.cursorY; r < t.scrollBottom; r++ {
				grid[r] = grid[r+1]
				t.markRowDirty(r)
			}
			grid[t.scrollBottom] = t.makeRow()
			t.markRowDirty(t.scrollBottom)
		}

	case 'P':
		n := arg(0, 1)
		row := grid[t.cursorY]
		for c := t.cursorX; c < t.cols; c++ {
			if c+n < t.cols {
				row[c] = row[c+n]
			} else {
				row[c] = EmptyCell()
			}
		}
		t.markRowDirty(t.cursorY)

	case 'r':
		top := arg(0, 1) - 1
		bottom := arg(1, t.rows) - 1
		if top >= 0 && bottom < t.rows && top < bottom {
			t.scrollTop = top
			t.scrollBottom = bottom
		}

	case 'm':
		t.executeSGR(args)

	case 'h', 'l':
		enable := (cmd == 'h')
		if t.csiPrivate {
			mode := arg(0, 0)
			switch mode {
			case 25:
				t.cursorVisible = enable
			case 1049, 47:
				t.isAlt = enable
				t.dirtyAll = true
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
