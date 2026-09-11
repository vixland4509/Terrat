package render

import (
	"fmt"
	"strings"
	"terrat/terminal"
)

const (
	HeaderHeight  = 34
	PaddingLeft   = 16
	PaddingRight  = 16
	PaddingTop    = 12
	PaddingBottom = 14
)

type TabInfo struct {
	ID      int
	Title   string
	Active  bool
	HasBell bool
}

type TabHitBox struct {
	ID        int
	StartX    int
	EndX      int
	CloseX    int
	CloseEndX int
}

type SearchMatch struct {
	Row      int
	StartCol int
	EndCol   int
}

type URLRange struct {
	Row      int
	StartCol int
	EndCol   int
	URL      string
}

type DiagnosticInfo struct {
	IsError    bool
	Message    string
	Suggestion string
	QuickFix   string
}

var mcDirt16x16 = [16][16]uint32{
	{0x231810, 0x271a12, 0x241810, 0x1f150e, 0x241810, 0x2b1d14, 0x271a12, 0x241810, 0x231810, 0x20160f, 0x241810, 0x271a12, 0x2b1d14, 0x241810, 0x1f150e, 0x231810},
	{0x271a12, 0x2b1d14, 0x271a12, 0x241810, 0x1c130d, 0x241810, 0x2f2017, 0x2b1d14, 0x241810, 0x271a12, 0x2b1d14, 0x332319, 0x271a12, 0x1c130d, 0x241810, 0x271a12},
	{0x241810, 0x271a12, 0x1a110b, 0x170f09, 0x241810, 0x2b1d14, 0x271a12, 0x241810, 0x1c130d, 0x2b1d14, 0x37261a, 0x2b1d14, 0x241810, 0x241810, 0x2b1d14, 0x241810},
	{0x20160f, 0x1c130d, 0x241810, 0x241810, 0x2b1d14, 0x241810, 0x1f150e, 0x1c130d, 0x241810, 0x241810, 0x2b1d14, 0x241810, 0x1f150e, 0x271a12, 0x2f2017, 0x271a12},
	{0x241810, 0x241810, 0x2b1d14, 0x2f2017, 0x271a12, 0x1c130d, 0x241810, 0x271a12, 0x2b1d14, 0x1c130d, 0x241810, 0x1c130d, 0x241810, 0x2b1d14, 0x271a12, 0x20160f},
	{0x2b1d14, 0x2f2017, 0x271a12, 0x241810, 0x1f150e, 0x241810, 0x2b1d14, 0x332319, 0x271a12, 0x241810, 0x2b1d14, 0x271a12, 0x241810, 0x20160f, 0x241810, 0x241810},
	{0x271a12, 0x241810, 0x1c130d, 0x241810, 0x271a12, 0x2b1d14, 0x271a12, 0x2b1d14, 0x241810, 0x2f2017, 0x2b1d14, 0x241810, 0x1c130d, 0x241810, 0x2b1d14, 0x271a12},
	{0x241810, 0x1c130d, 0x241810, 0x2b1d14, 0x332319, 0x271a12, 0x1c130d, 0x241810, 0x271a12, 0x271a12, 0x20160f, 0x241810, 0x271a12, 0x2b1d14, 0x271a12, 0x241810},
	{0x20160f, 0x241810, 0x2b1d14, 0x271a12, 0x241810, 0x1f150e, 0x241810, 0x2b1d14, 0x37261a, 0x2b1d14, 0x241810, 0x1c130d, 0x241810, 0x241810, 0x1f150e, 0x20160f},
	{0x241810, 0x2b1d14, 0x271a12, 0x20160f, 0x241810, 0x271a12, 0x2b1d14, 0x241810, 0x2b1d14, 0x241810, 0x1c130d, 0x241810, 0x2b1d14, 0x2f2017, 0x271a12, 0x241810},
	{0x2b1d14, 0x271a12, 0x1c130d, 0x241810, 0x2b1d14, 0x332319, 0x271a12, 0x1c130d, 0x241810, 0x271a12, 0x2b1d14, 0x271a12, 0x241810, 0x2b1d14, 0x241810, 0x2b1d14},
	{0x271a12, 0x241810, 0x271a12, 0x2b1d14, 0x241810, 0x271a12, 0x20160f, 0x241810, 0x2b1d14, 0x2f2017, 0x241810, 0x20160f, 0x271a12, 0x241810, 0x1c130d, 0x271a12},
	{0x241810, 0x20160f, 0x2b1d14, 0x271a12, 0x1c130d, 0x241810, 0x271a12, 0x2b1d14, 0x271a12, 0x241810, 0x1c130d, 0x241810, 0x2b1d14, 0x332319, 0x271a12, 0x241810},
	{0x1f150e, 0x241810, 0x241810, 0x1c130d, 0x241810, 0x2b1d14, 0x37261a, 0x271a12, 0x20160f, 0x241810, 0x271a12, 0x2b1d14, 0x271a12, 0x241810, 0x20160f, 0x1f150e},
	{0x241810, 0x271a12, 0x2b1d14, 0x241810, 0x271a12, 0x271a12, 0x2b1d14, 0x241810, 0x241810, 0x2b1d14, 0x2f2017, 0x241810, 0x1c130d, 0x241810, 0x271a12, 0x241810},
	{0x271a12, 0x2b1d14, 0x241810, 0x20160f, 0x241810, 0x20160f, 0x241810, 0x271a12, 0x2b1d14, 0x271a12, 0x241810, 0x1c130d, 0x241810, 0x2b1d14, 0x241810, 0x271a12},
}

var mcDeepslate16x16 = [16][16]uint32{
	{0x181a18, 0x1c1f1c, 0x181a18, 0x141614, 0x181a18, 0x1e221e, 0x1c1f1c, 0x181a18, 0x181a18, 0x151715, 0x181a18, 0x1c1f1c, 0x222622, 0x181a18, 0x141614, 0x181a18},
	{0x1c1f1c, 0x222622, 0x1c1f1c, 0x181a18, 0x121412, 0x181a18, 0x242824, 0x1e221e, 0x181a18, 0x1c1f1c, 0x1e221e, 0x262a26, 0x1c1f1c, 0x121412, 0x181a18, 0x1c1f1c},
	{0x181a18, 0x1c1f1c, 0x121412, 0x101210, 0x181a18, 0x1e221e, 0x1c1f1c, 0x181a18, 0x121412, 0x1e221e, 0x262b26, 0x1e221e, 0x181a18, 0x181a18, 0x1e221e, 0x181a18},
	{0x151715, 0x121412, 0x181a18, 0x181a18, 0x1e221e, 0x181a18, 0x141614, 0x121412, 0x181a18, 0x181a18, 0x1e221e, 0x181a18, 0x141614, 0x1c1f1c, 0x242824, 0x1c1f1c},
	{0x181a18, 0x181a18, 0x1e221e, 0x242824, 0x1c1f1c, 0x121412, 0x181a18, 0x1c1f1c, 0x1e221e, 0x121412, 0x181a18, 0x121412, 0x181a18, 0x1e221e, 0x1c1f1c, 0x151715},
	{0x1e221e, 0x242824, 0x1c1f1c, 0x181a18, 0x141614, 0x181a18, 0x1e221e, 0x262a26, 0x1c1f1c, 0x181a18, 0x1e221e, 0x1c1f1c, 0x181a18, 0x151715, 0x181a18, 0x181a18},
	{0x1c1f1c, 0x181a18, 0x121412, 0x181a18, 0x1c1f1c, 0x1e221e, 0x1c1f1c, 0x1e221e, 0x181a18, 0x242824, 0x1e221e, 0x181a18, 0x121412, 0x181a18, 0x1e221e, 0x1c1f1c},
	{0x181a18, 0x121412, 0x181a18, 0x1e221e, 0x262a26, 0x1c1f1c, 0x121412, 0x181a18, 0x1c1f1c, 0x1c1f1c, 0x151715, 0x181a18, 0x1c1f1c, 0x1e221e, 0x1c1f1c, 0x181a18},
	{0x151715, 0x181a18, 0x1e221e, 0x1c1f1c, 0x181a18, 0x141614, 0x181a18, 0x1e221e, 0x282e28, 0x1e221e, 0x181a18, 0x121412, 0x181a18, 0x181a18, 0x141614, 0x151715},
	{0x181a18, 0x1e221e, 0x1c1f1c, 0x151715, 0x181a18, 0x1c1f1c, 0x1e221e, 0x181a18, 0x1e221e, 0x181a18, 0x121412, 0x181a18, 0x1e221e, 0x242824, 0x1c1f1c, 0x181a18},
	{0x1e221e, 0x1c1f1c, 0x121412, 0x181a18, 0x1e221e, 0x262a26, 0x1c1f1c, 0x121412, 0x181a18, 0x1c1f1c, 0x1e221e, 0x1c1f1c, 0x181a18, 0x1e221e, 0x181a18, 0x1e221e},
	{0x1c1f1c, 0x181a18, 0x1c1f1c, 0x1e221e, 0x181a18, 0x1c1f1c, 0x151715, 0x181a18, 0x1e221e, 0x242824, 0x181a18, 0x151715, 0x1c1f1c, 0x181a18, 0x121412, 0x1c1f1c},
	{0x181a18, 0x151715, 0x1e221e, 0x1c1f1c, 0x121412, 0x181a18, 0x1c1f1c, 0x1e221e, 0x1c1f1c, 0x181a18, 0x121412, 0x181a18, 0x1e221e, 0x262a26, 0x1c1f1c, 0x181a18},
	{0x141614, 0x181a18, 0x181a18, 0x121412, 0x181a18, 0x1e221e, 0x282e28, 0x1c1f1c, 0x151715, 0x181a18, 0x1c1f1c, 0x1e221e, 0x1c1f1c, 0x181a18, 0x151715, 0x141614},
	{0x181a18, 0x1c1f1c, 0x1e221e, 0x181a18, 0x1c1f1c, 0x1c1f1c, 0x1e221e, 0x181a18, 0x181a18, 0x1e221e, 0x242824, 0x181a18, 0x121412, 0x181a18, 0x1c1f1c, 0x181a18},
	{0x1c1f1c, 0x1e221e, 0x181a18, 0x151715, 0x181a18, 0x151715, 0x181a18, 0x1c1f1c, 0x1e221e, 0x1c1f1c, 0x181a18, 0x121412, 0x181a18, 0x1e221e, 0x181a18, 0x1c1f1c},
}

func FillPattern16x16(buf []byte, stride, x, y, w, h int, pattern *[16][16]uint32) {
	if w <= 0 || h <= 0 {
		return
	}
	for row := 0; row < h; row++ {
		py := y + row
		if py < 0 || py*stride >= len(buf) {
			continue
		}
		bufRowOffset := py * stride
		patY := py & 15

		for col := 0; col < w; col++ {
			px := x + col
			patX := px & 15
			pixel := pattern[patY][patX]

			p := bufRowOffset + px*4
			if p >= 0 && p+3 < len(buf) {
				buf[p+0] = byte(pixel)
				buf[p+1] = byte(pixel >> 8)
				buf[p+2] = byte(pixel >> 16)
				buf[p+3] = 0xff
			}
		}
	}
}

func DrawMinecraftButton(buf []byte, stride, x, y, w, h int, selected bool) {
	DrawRectBorder(buf, stride, x, y, w, h, 0x000000)

	hiColor := uint32(0x8a8a8a)
	shColor := uint32(0x282828)
	topFill := uint32(0x525252)
	botFill := uint32(0x424242)

	if selected {
		hiColor = 0xffffff
		shColor = 0x5a5a5a
		topFill = 0x6e6e6e
		botFill = 0x5a5a5a
	}

	halfH := (h - 2) / 2
	FillRect(buf, stride, x+1, y+1, w-2, halfH, topFill)
	FillRect(buf, stride, x+1, y+1+halfH, w-2, h-2-halfH, botFill)

	DrawHLine(buf, stride, x+1, y+1, w-2, hiColor)
	DrawVLine(buf, stride, x+1, y+1, h-2, hiColor)

	DrawHLine(buf, stride, x+1, y+h-2, w-2, shColor)
	DrawVLine(buf, stride, x+w-2, y+1, h-2, shColor)
	DrawHLine(buf, stride, x+2, y+h-3, w-4, shColor)
}

func DrawMinecraftBlock(buf []byte, stride, x, y int, blockType string) {
	size := 12
	DrawRectBorder(buf, stride, x, y, size, size, 0x000000)

	var hiCol, shCol, fillCol, dotCol uint32
	switch blockType {
	case "redstone":
		hiCol = 0xff6666
		shCol = 0x660000
		fillCol = 0xb81818
		dotCol = 0xffaaaa
	case "gold":
		hiCol = 0xffea75
		shCol = 0x7a5700
		fillCol = 0xdca316
		dotCol = 0xfff6a8
	case "emerald":
		hiCol = 0x70ff94
		shCol = 0x0c6922
		fillCol = 0x1db347
		dotCol = 0xa8ffbe
	}

	DrawHLine(buf, stride, x+1, y+1, size-2, hiCol)
	DrawVLine(buf, stride, x+1, y+1, size-2, hiCol)
	DrawHLine(buf, stride, x+1, y+size-2, size-2, shCol)
	DrawVLine(buf, stride, x+size-2, y+1, size-2, shCol)

	FillRect(buf, stride, x+2, y+2, size-4, size-4, fillCol)

	switch blockType {
	case "redstone":
		FillRect(buf, stride, x+5, y+4, 2, 4, dotCol)
		FillRect(buf, stride, x+4, y+5, 4, 2, dotCol)
	case "gold":
		DrawHLine(buf, stride, x+4, y+4, 3, dotCol)
		DrawHLine(buf, stride, x+5, y+7, 3, dotCol)
	case "emerald":
		FillRect(buf, stride, x+4, y+4, 2, 2, dotCol)
		FillRect(buf, stride, x+6, y+6, 2, 2, dotCol)
	}
}

type Canvas struct {
	Width  int
	Height int
	Stride int
	Pixels []byte

	fontEngine *FontEngine
	cols       int
	rows       int

	TabHitBoxes  []TabHitBox
	NewTabHitBox [4]int
	PrefModalBox [5]int // [0]: x, [1]: y, [2]: w, [3]: h, [4]: rowH
}

func NewCanvas(fe *FontEngine) *Canvas {
	return &Canvas{
		fontEngine: fe,
	}
}

func (c *Canvas) Resize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	c.Width = width
	c.Height = height
	c.Stride = width * 4

	needed := height * c.Stride
	if len(c.Pixels) < needed {
		c.Pixels = make([]byte, needed)
	} else {
		c.Pixels = c.Pixels[:needed]
	}

	availW := width - (PaddingLeft + PaddingRight)
	availH := height - (HeaderHeight + PaddingTop + PaddingBottom)

	if availW < c.fontEngine.CharWidth() {
		availW = c.fontEngine.CharWidth()
	}
	if availH < c.fontEngine.CharHeight() {
		availH = c.fontEngine.CharHeight()
	}

	c.cols = availW / c.fontEngine.CharWidth()
	c.rows = availH / c.fontEngine.CharHeight()
}

func (c *Canvas) Cols() int {
	return c.cols
}

func (c *Canvas) Rows() int {
	return c.rows
}

func (c *Canvas) setPixel(x, y int, pixel uint32) {
	if x < 0 || x >= c.Width || y < 0 || y >= c.Height {
		return
	}
	p := y*c.Stride + x*4
	if p >= 0 && p+3 < len(c.Pixels) {
		c.Pixels[p+0] = byte(pixel)
		c.Pixels[p+1] = byte(pixel >> 8)
		c.Pixels[p+2] = byte(pixel >> 16)
		c.Pixels[p+3] = 0xff
	}
}

func (c *Canvas) Render(term *terminal.Terminal, cursorBlink bool, title string, hudInfo string, tabs []TabInfo, searchMatches []SearchMatch, activeSearchIdx int, hoveredURL *URLRange, ghostText string, diag *DiagnosticInfo) {
	term.RLock()
	defer term.RUnlock()

	charW := c.fontEngine.CharWidth()
	charH := c.fontEngine.CharHeight()
	baseline := c.fontEngine.Baseline()

	th := term.ThemeLocked()
	defaultBGPixel := th.BG.ToPixel()
	headerBGPixel := th.HeaderBG.ToPixel()
	headerLinePixel := th.HeaderLine.ToPixel()
	borderPixel := th.Border.ToPixel()
	closeDotPixel := th.CloseDot.ToPixel()
	badgeTextPixel := th.BadgeText.ToPixel()
	mutedTextPixel := th.MutedText.ToPixel()

	isMC := (th.ID == "minecraft")

	if isMC {
		FillPattern16x16(c.Pixels, c.Stride, 0, 0, c.Width, c.Height, &mcDeepslate16x16)
		DrawHLine(c.Pixels, c.Stride, 0, HeaderHeight-1, c.Width, 0x000000)
		DrawHLine(c.Pixels, c.Stride, 0, HeaderHeight, c.Width, 0x282f28)
	} else {
		FillRect(c.Pixels, c.Stride, 0, 0, c.Width, c.Height, defaultBGPixel)

		FillRect(c.Pixels, c.Stride, 0, 0, c.Width, HeaderHeight, headerBGPixel)
		DrawHLine(c.Pixels, c.Stride, 0, HeaderHeight-1, c.Width, headerLinePixel)
	}

	titleY := (HeaderHeight - charH) / 2
	c.TabHitBoxes = nil
	c.NewTabHitBox = [4]int{0, 0, 0, 0}

	curXTab := 12
	availTabWidth := c.Width - 120 - curXTab - 40
	if availTabWidth < 80 {
		availTabWidth = 80
	}

	tabCount := len(tabs)
	if tabCount == 0 {
		tabCount = 1
	}
	tabW := availTabWidth / tabCount
	if tabW > 180 {
		tabW = 180
	}
	if tabW < 75 {
		tabW = 75
	}

	for i, tab := range tabs {
		tStartX := curXTab
		tEndX := curXTab + tabW

		if isMC {
			if tab.Active {
				FillPattern16x16(c.Pixels, c.Stride, tStartX, 0, tabW, HeaderHeight, &mcDeepslate16x16)
				DrawRectBorder(c.Pixels, c.Stride, tStartX, 0, tabW, HeaderHeight, 0x000000)
				DrawHLine(c.Pixels, c.Stride, tStartX+1, 1, tabW-2, 0x5a5e5a)
				DrawVLine(c.Pixels, c.Stride, tStartX+1, 1, HeaderHeight-2, 0x5a5e5a)
				DrawVLine(c.Pixels, c.Stride, tEndX-2, 1, HeaderHeight-2, 0x181a18)
				DrawHLine(c.Pixels, c.Stride, tStartX+1, HeaderHeight-1, tabW-2, 0x181a18)
			} else {
				FillRect(c.Pixels, c.Stride, tStartX+1, 3, tabW-2, HeaderHeight-5, 0x121412)
				DrawRectBorder(c.Pixels, c.Stride, tStartX+1, 3, tabW-2, HeaderHeight-5, 0x0a0c0a)
				DrawHLine(c.Pixels, c.Stride, tStartX+2, 4, tabW-4, 0x1e221e)
			}
		} else {
			if tab.Active {
				FillRect(c.Pixels, c.Stride, tStartX, 0, tabW, HeaderHeight, defaultBGPixel)
				DrawVLine(c.Pixels, c.Stride, tStartX, 0, HeaderHeight, headerLinePixel)
				DrawVLine(c.Pixels, c.Stride, tEndX-1, 0, HeaderHeight, headerLinePixel)
			} else {
				DrawVLine(c.Pixels, c.Stride, tEndX-1, 6, HeaderHeight-12, headerLinePixel)
			}
		}

		maxChars := (tabW - 32) / charW
		disp := fmt.Sprintf("%d: %s", i+1, tab.Title)
		if tab.Title == "" {
			disp = fmt.Sprintf("%d: bash", i+1)
		}
		disp = truncateString(disp, maxChars, "…")
		tTextY := (HeaderHeight - charH) / 2
		closeX := tEndX - 18
		closeEndX := tEndX - 4

		if isMC {
			if tab.Active {
				c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, tStartX+10, tTextY, disp, 0xffffff, 0x3f3f3f, true)
				c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, closeX, tTextY, "×", 0xff5555, 0x3f1515, true)
			} else {
				c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, tStartX+10, tTextY, disp, 0x888888, 0x222222, false)
				c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, closeX, tTextY, "×", 0x555555, 0x151515, false)
			}
		} else {
			tabBGPixel := headerBGPixel
			tabFGPixel := mutedTextPixel
			if tab.Active {
				tabBGPixel = defaultBGPixel
				tabFGPixel = th.FG.ToPixel()
			}
			c.fontEngine.DrawString(c.Pixels, c.Stride, tStartX+10, tTextY, disp, tabFGPixel, tabBGPixel, tab.Active)
			c.fontEngine.DrawString(c.Pixels, c.Stride, closeX, tTextY, "×", mutedTextPixel, tabBGPixel, false)
		}

		c.TabHitBoxes = append(c.TabHitBoxes, TabHitBox{
			ID:        tab.ID,
			StartX:    tStartX,
			EndX:      tEndX,
			CloseX:    closeX - 4,
			CloseEndX: closeEndX + 4,
		})

		curXTab += tabW
	}

	btnW := 26
	btnH := 20
	btnX := curXTab + 6
	btnY := (HeaderHeight - btnH) / 2
	plusCharX := btnX + (btnW-charW)/2
	plusCharY := (HeaderHeight - charH) / 2
	if isMC {
		DrawMinecraftButton(c.Pixels, c.Stride, btnX, btnY, btnW, btnH, false)
		c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, plusCharX, plusCharY, "+", 0x55ff55, 0x153f15, true)
	} else {
		DrawRectBorder(c.Pixels, c.Stride, btnX, btnY, btnW, btnH, headerLinePixel)
		c.fontEngine.DrawString(c.Pixels, c.Stride, plusCharX, plusCharY, "+", th.BadgeText.ToPixel(), headerBGPixel, true)
	}
	c.NewTabHitBox = [4]int{btnX - 2, btnX + btnW + 2, 0, HeaderHeight}

	// Window controls on the top-right (Settings, Minimize, Maximize, Close)
	winBtnW := 36
	settingsBtnX := c.Width - winBtnW*4
	minBtnX := c.Width - winBtnW*3
	maxBtnX := c.Width - winBtnW*2
	closeBtnX := c.Width - winBtnW

	if isMC {
		DrawMinecraftBlock(c.Pixels, c.Stride, settingsBtnX+12, 11, "diamond")
		DrawMinecraftBlock(c.Pixels, c.Stride, minBtnX+12, 11, "gold")
		DrawMinecraftBlock(c.Pixels, c.Stride, maxBtnX+12, 11, "emerald")
		DrawMinecraftBlock(c.Pixels, c.Stride, closeBtnX+12, 11, "redstone")
	} else {
		midY := HeaderHeight / 2
		btnFG := mutedTextPixel

		// 0. Settings / Toggles: clean minimalist sliders icon
		sliderX := settingsBtnX + 13
		DrawHLine(c.Pixels, c.Stride, sliderX, midY-3, 10, btnFG)
		DrawVLine(c.Pixels, c.Stride, sliderX+7, midY-5, 5, btnFG)
		DrawHLine(c.Pixels, c.Stride, sliderX, midY+3, 10, btnFG)
		DrawVLine(c.Pixels, c.Stride, sliderX+2, midY+1, 5, btnFG)

		// 1. Minimize: clean horizontal line
		DrawHLine(c.Pixels, c.Stride, minBtnX+13, midY, 10, btnFG)
		DrawHLine(c.Pixels, c.Stride, minBtnX+13, midY+1, 10, btnFG)

		// 2. Maximize / Fullscreen: clean square outline
		sqSize := 10
		sqX := maxBtnX + 13
		sqY := (HeaderHeight - sqSize) / 2
		DrawRectBorder(c.Pixels, c.Stride, sqX, sqY, sqSize, sqSize, btnFG)

		// 3. Close: clean diagonal cross ✕
		crossSize := 9
		crossX := closeBtnX + 13
		crossY := (HeaderHeight - crossSize) / 2
		closeFG := closeDotPixel
		for i := 0; i < crossSize; i++ {
			c.setPixel(crossX+i, crossY+i, closeFG)
			c.setPixel(crossX+i+1, crossY+i, closeFG)
			c.setPixel(crossX+crossSize-1-i, crossY+i, closeFG)
			c.setPixel(crossX+crossSize-i, crossY+i, closeFG)
		}
	}

	if hudInfo == "" {
		hudInfo = fmt.Sprintf("%dx%d", c.cols, c.rows)
	}
	hudLen := len(hudInfo)
	hudX := settingsBtnX - (hudLen * charW) - 16
	if hudX > btnX+btnW+16 {
		if isMC {
			c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, hudX, titleY, hudInfo, 0x55ffff, 0x153f3f, true)
		} else {
			c.fontEngine.DrawString(c.Pixels, c.Stride, hudX, titleY, hudInfo, badgeTextPixel, headerBGPixel, false)
		}
	}

	// Render diagnostic notification chip in header bar between tabs and grid size
	if diag != nil && !term.IsAltLocked() {
		diagMsg := diag.Message
		if diag.Suggestion != "" {
			diagMsg = fmt.Sprintf("%s (%s)", diag.Message, diag.Suggestion)
		}
		maxDiagChars := (hudX - (btnX + btnW + 24)) / charW
		if maxDiagChars > 10 {
			diagMsg = truncateString(diagMsg, maxDiagChars, "…")
			chipW := len([]rune(diagMsg))*charW + 16
			chipH := 20
			chipX := hudX - chipW - 14
			if chipX > btnX+btnW+12 {
				chipY := (HeaderHeight - chipH) / 2
				chipColor := th.MinDot.ToPixel() // Amber warning
				if diag.IsError {
					chipColor = th.CloseDot.ToPixel() // Red error
				}
				if isMC {
					DrawMinecraftButton(c.Pixels, c.Stride, chipX, chipY, chipW, chipH, false)
					c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, chipX+8, chipY+(chipH-charH)/2, diagMsg, chipColor, MinecraftShadow(chipColor), true)
				} else {
					DrawRectBorder(c.Pixels, c.Stride, chipX, chipY, chipW, chipH, chipColor)
					c.fontEngine.DrawString(c.Pixels, c.Stride, chipX+8, chipY+(chipH-charH)/2, diagMsg, chipColor, headerBGPixel, true)
				}
			}
		}
	}

	gridStartY := HeaderHeight + PaddingTop
	gridStartX := PaddingLeft
	curX, curY, curVis := term.CursorLocked()
	renderCurX := curX
	if renderCurX >= c.cols && c.cols > 0 {
		renderCurX = c.cols - 1
	}
	scrollOff := term.ScrollOffLocked()
	effectiveCurY := curY
	if scrollOff > 0 {
		effectiveCurY = curY + scrollOff
	}

	for y := 0; y < c.rows; y++ {
		cellY := gridStartY + (y * charH)

		for x := 0; x < c.cols; x++ {
			cell := term.GetCell(x, y)
			cellX := gridStartX + (x * charW)

			fg := cell.FG
			if fg == terminal.ColorDefaultFG || fg == 0 {
				fg = th.FG
			}
			bg := cell.BG
			if bg == terminal.ColorDefaultBG || bg == 0 {
				bg = th.BG
			}

			if cell.Inverse {
				fg, bg = bg, fg
			}

			isSearchMatch := false
			isActiveSearchMatch := false
			for idx, sm := range searchMatches {
				if y == sm.Row && x >= sm.StartCol && x <= sm.EndCol {
					isSearchMatch = true
					if idx == activeSearchIdx {
						isActiveSearchMatch = true
					}
					break
				}
			}

			if isSearchMatch {
				if isActiveSearchMatch {
					bg = terminal.Color(0x9ece6a)
					fg = terminal.Color(0x15161e)
				} else {
					bg = terminal.Color(0xe0af68)
					fg = terminal.Color(0x15161e)
				}
			}

			if term.IsSelectedLocked(x, y) {
				bg = th.SelectionBG
				fg = th.SelectionFG
			}

			isCursor := (x == renderCurX && y == effectiveCurY && effectiveCurY < c.rows && curVis && cursorBlink)
			if isCursor {
				if isMC {
					DrawRectBorder(c.Pixels, c.Stride, cellX, cellY, charW, charH, 0x000000)
					FillRect(c.Pixels, c.Stride, cellX+1, cellY+1, charW-2, charH-2, 0x55ffff)
					DrawHLine(c.Pixels, c.Stride, cellX+1, cellY+1, charW-2, 0xaaffff)
					DrawVLine(c.Pixels, c.Stride, cellX+1, cellY+1, charH-2, 0xaaffff)
					DrawHLine(c.Pixels, c.Stride, cellX+1, cellY+charH-2, charW-2, 0x008888)
					DrawVLine(c.Pixels, c.Stride, cellX+charW-2, cellY+1, charH-2, 0x008888)
					fgPixel := uint32(0x000000)
					bgPixel := uint32(0x55ffff)
					char := cell.Char
					if char == 0 {
						char = ' '
					}
					if char != ' ' {
						mask := c.fontEngine.GetGlyph(char, cell.Bold)
						DrawGlyphBlit(c.Pixels, c.Stride, cellX, cellY, mask, fgPixel, bgPixel, false, baseline)
					}
					continue
				} else {
					fg, bg = bg, th.Cursor
				}
			}

			fgPixel := fg.ToPixel()
			bgPixel := bg.ToPixel()

			char := cell.Char
			if char == 0 {
				char = ' '
			}

			isHoveredURL := (hoveredURL != nil && y == hoveredURL.Row && x >= hoveredURL.StartCol && x <= hoveredURL.EndCol)

			if char == ' ' && !isCursor && !cell.Underline && !isHoveredURL && !isSearchMatch {
				if bg != th.BG {
					FillRect(c.Pixels, c.Stride, cellX, cellY, charW, charH, bgPixel)
				}
				continue
			}

			mask := c.fontEngine.GetGlyph(char, cell.Bold)
			if isMC && bg == th.BG && !isSearchMatch && !term.IsSelectedLocked(x, y) {
				DrawGlyphBlitTransparent(c.Pixels, c.Stride, cellX, cellY, mask, fgPixel, cell.Underline || isHoveredURL, baseline)
			} else {
				DrawGlyphBlit(c.Pixels, c.Stride, cellX, cellY, mask, fgPixel, bgPixel, cell.Underline || isHoveredURL, baseline)
			}
		}
	}

	// Render Ghost Text (inline autosuggestion) directly following the cursor on normal screen
	if ghostText != "" && !term.IsAltLocked() && scrollOff == 0 && effectiveCurY >= 0 && effectiveCurY < c.rows {
		ghostY := gridStartY + (effectiveCurY * charH)
		ghostFGPixel := th.MutedText.ToPixel()
		ghostBGPixel := defaultBGPixel

		for idx, r := range ghostText {
			gx := curX + idx
			if gx >= c.cols {
				break
			}
			// Only draw ghost text on empty cells so we never collide with existing characters
			cell := term.GetCell(gx, effectiveCurY)
			if cell.Char != 0 && cell.Char != ' ' {
				break
			}

			cellX := gridStartX + (gx * charW)
			mask := c.fontEngine.GetGlyph(r, false)
			DrawGlyphBlit(c.Pixels, c.Stride, cellX, ghostY, mask, ghostFGPixel, ghostBGPixel, false, baseline)
		}
	}

	// Render Floating Diagnostic Tooltip directly below cursor row
	if diag != nil && !term.IsAltLocked() && scrollOff == 0 && effectiveCurY >= 0 && effectiveCurY < c.rows {
		diagMsg := "💡 " + diag.Message
		if diag.Suggestion != "" {
			diagMsg += "  •  " + diag.Suggestion
		}
		if diag.QuickFix != "" {
			diagMsg += "  [Alt+Enter to fix]"
		}

		tipPadX := 10
		tipPadY := 4
		tipW := len(diagMsg)*charW + tipPadX*2
		tipH := charH + tipPadY*2

		if tipW > c.Width-30 {
			tipW = c.Width - 30
		}

		tipX := gridStartX + (curX * charW)
		if tipX+tipW > c.Width-16 {
			tipX = c.Width - tipW - 16
		}
		if tipX < gridStartX {
			tipX = gridStartX
		}

		// Show below cursor, or above cursor if at the bottom row
		tipY := gridStartY + ((effectiveCurY + 1) * charH) + 4
		if tipY+tipH > c.Height-PaddingBottom {
			tipY = gridStartY + ((effectiveCurY - 1) * charH) - 4
		}

		tipBGPix := th.HeaderBG.ToPixel()
		tipBorderPix := th.MinDot.ToPixel() // Amber
		if diag.IsError {
			tipBorderPix = th.CloseDot.ToPixel() // Red
		}

		FillRect(c.Pixels, c.Stride, tipX, tipY, tipW, tipH, tipBGPix)
		DrawRectBorder(c.Pixels, c.Stride, tipX, tipY, tipW, tipH, tipBorderPix)

		textY := tipY + tipPadY
		c.fontEngine.DrawString(c.Pixels, c.Stride, tipX+tipPadX, textY, diagMsg, tipBorderPix, tipBGPix, true)
	}

	maxScroll := term.ScrollbackLenLocked()
	if maxScroll > 0 {
		trackY := HeaderHeight + 4
		trackH := c.Height - HeaderHeight - 8
		if trackH > 24 {
			totalLines := maxScroll + c.rows
			thumbH := (c.rows * trackH) / totalLines
			if thumbH < 20 {
				thumbH = 20
			}
			if thumbH > trackH {
				thumbH = trackH
			}
			progress := float64(maxScroll-scrollOff) / float64(maxScroll)
			thumbY := trackY + int(progress*float64(trackH-thumbH))
			thumbW := 5
			thumbX := c.Width - thumbW - 2

			// Draw subtle track
			FillRect(c.Pixels, c.Stride, thumbX, trackY, thumbW, trackH, th.HeaderBG.ToPixel())

			// Draw thumb (brighter when scrolled)
			thumbColor := th.MutedText.ToPixel()
			if scrollOff > 0 {
				thumbColor = th.BadgeText.ToPixel()
			}
			FillRect(c.Pixels, c.Stride, thumbX, thumbY, thumbW, thumbH, thumbColor)
		}
	}

	DrawRectBorder(c.Pixels, c.Stride, 0, 0, c.Width, c.Height, borderPixel)
}

type PrefOption struct {
	ID       string
	Label    string
	Sublabel string
	Value    string
	IsToggle bool
	Enabled  bool
}

func (c *Canvas) RenderPreferencesModal(th *terminal.Theme, selectedIdx int, options []PrefOption, savedID string) (modalX, modalY, modalW, modalH, rowH int) {
	charW := c.fontEngine.CharWidth()
	charH := c.fontEngine.CharHeight()

	maxChars := 28
	for _, opt := range options {
		l := len(opt.Label) + 4
		if opt.IsToggle {
			l += 8 // for "   [ON]"
		} else if opt.Value != "" {
			l += len(opt.Value) + 3
		}
		if l > maxChars {
			maxChars = l
		}
	}

	if th.ID == "minecraft" {
		modalW = maxChars*charW + 44
		if modalW < 300 {
			modalW = 300
		}
		rowH = charH + 14
		if rowH < 30 {
			rowH = 30
		}
		headerH := 40
		footerH := 36
		modalH = headerH + len(options)*rowH + footerH

		if modalW > c.Width-20 {
			modalW = c.Width - 20
		}
		if modalH > c.Height-HeaderHeight-10 {
			modalH = c.Height - HeaderHeight - 10
		}

		modalX = c.Width - modalW - 12
		if modalX < 6 {
			modalX = 6
		}
		modalY = HeaderHeight + 6

		c.PrefModalBox = [5]int{modalX, modalY, modalW, modalH, rowH}

		// 1. Tiled Minecraft Dirt menu background
		FillPattern16x16(c.Pixels, c.Stride, modalX, modalY, modalW, modalH, &mcDirt16x16)

		// 2. 3-layer beveled Minecraft GUI Container border
		DrawRectBorder(c.Pixels, c.Stride, modalX, modalY, modalW, modalH, 0x000000)
		DrawRectBorder(c.Pixels, c.Stride, modalX+1, modalY+1, modalW-2, modalH-2, 0x000000)
		DrawHLine(c.Pixels, c.Stride, modalX+2, modalY+2, modalW-4, 0xc6c6c6)
		DrawVLine(c.Pixels, c.Stride, modalX+2, modalY+2, modalH-4, 0xc6c6c6)
		DrawHLine(c.Pixels, c.Stride, modalX+2, modalY+modalH-3, modalW-4, 0x373737)
		DrawVLine(c.Pixels, c.Stride, modalX+modalW-3, modalY+2, modalH-4, 0x373737)

		// Header separator groove
		DrawHLine(c.Pixels, c.Stride, modalX+4, modalY+headerH-2, modalW-8, 0x140e09)
		DrawHLine(c.Pixels, c.Stride, modalX+4, modalY+headerH-1, modalW-8, 0x3e291c)

		// Header Title
		title := "SETTINGS // QUICK TOGGLES"
		titleX := modalX + (modalW-len(title)*charW)/2
		c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, titleX, modalY+(headerH-charH)/2, title, 0xffff55, 0x3f3f15, true)

		// Option Rows as Real Minecraft 3D Buttons
		optStartY := modalY + headerH + 4
		for i, opt := range options {
			rowY := optStartY + i*rowH
			isSelected := (i == selectedIdx)

			btnX := modalX + 12
			btnW := modalW - 24
			btnH := rowH - 4

			DrawMinecraftButton(c.Pixels, c.Stride, btnX, rowY, btnW, btnH, isSelected)

			textY := rowY + (btnH-charH)/2
			prefix := "  "
			labelColor := uint32(0xe0e0e0)
			labelShadow := uint32(0x383838)

			if isSelected {
				prefix = "> "
				labelColor = 0xffffa0
				labelShadow = 0x3f3f20
			}

			c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, btnX+10, textY, prefix+opt.Label, labelColor, labelShadow, isSelected)

			if opt.IsToggle {
				badge := "[OFF]"
				badgeColor := uint32(0x777777)
				badgeShadow := uint32(0x1f1f1f)
				if opt.Enabled {
					badge = "[ON]"
					badgeColor = uint32(0x55ff55)
					badgeShadow = uint32(0x153f15)
				}
				badgeX := btnX + btnW - (len(badge) * charW) - 12
				c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, badgeX, textY, badge, badgeColor, badgeShadow, true)
			} else if opt.Value != "" {
				badgeX := btnX + btnW - (len(opt.Value) * charW) - 12
				c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, badgeX, textY, opt.Value, 0x55ffff, 0x153f3f, true)
			} else if opt.ID == savedID {
				activeBadge := "[SAVED]"
				badgeX := btnX + btnW - (len(activeBadge) * charW) - 12
				c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, badgeX, textY, activeBadge, 0x55ff55, 0x153f15, true)
			}
		}

		// Footer separator groove
		footerY := modalY + modalH - footerH
		DrawHLine(c.Pixels, c.Stride, modalX+4, footerY, modalW-8, 0x140e09)
		DrawHLine(c.Pixels, c.Stride, modalX+4, footerY+1, modalW-8, 0x3e291c)

		hints := "Click/Enter: Toggle  ESC: Close"
		hintsX := modalX + (modalW-len(hints)*charW)/2
		c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, hintsX, footerY+(footerH-charH)/2, hints, 0xa0a0a0, 0x282828, false)

		return modalX, modalY, modalW, modalH, rowH
	}

	modalW = maxChars*charW + 36
	if modalW < 280 {
		modalW = 280
	}
	rowH = charH + 12
	if rowH < 26 {
		rowH = 26
	}
	headerH := 36
	footerH := 28
	modalH = headerH + len(options)*rowH + footerH

	if modalW > c.Width-20 {
		modalW = c.Width - 20
	}
	if modalH > c.Height-HeaderHeight-10 {
		modalH = c.Height - HeaderHeight - 10
	}

	// Minimal and in the top-right corner under the settings button!
	modalX = c.Width - modalW - 12
	if modalX < 6 {
		modalX = 6
	}
	modalY = HeaderHeight + 6

	c.PrefModalBox = [5]int{modalX, modalY, modalW, modalH, rowH}

	modalBGPixel := th.HeaderBG.ToPixel()
	modalHeaderBGPixel := th.BG.ToPixel()
	borderPixel := th.Border.ToPixel()
	badgeTextPixel := th.BadgeText.ToPixel()
	mutedTextPixel := th.MutedText.ToPixel()
	fgPixel := th.FG.ToPixel()
	selBGPixel := th.SelectionBG.ToPixel()
	selFGPixel := th.SelectionFG.ToPixel()

	FillRect(c.Pixels, c.Stride, modalX, modalY, modalW, modalH, modalBGPixel)

	FillRect(c.Pixels, c.Stride, modalX, modalY, modalW, headerH, modalHeaderBGPixel)
	DrawHLine(c.Pixels, c.Stride, modalX, modalY+headerH-1, modalW, th.HeaderLine.ToPixel())
	c.fontEngine.DrawString(c.Pixels, c.Stride, modalX+14, modalY+(headerH-charH)/2, "SETTINGS // QUICK TOGGLES", badgeTextPixel, modalHeaderBGPixel, true)

	optStartY := modalY + headerH + 4
	for i, opt := range options {
		rowY := optStartY + i*rowH
		isSelected := (i == selectedIdx)

		bgPix := modalBGPixel
		fgPix := fgPixel
		if isSelected {
			bgPix = selBGPixel
			fgPix = selFGPixel
			FillRect(c.Pixels, c.Stride, modalX+6, rowY, modalW-12, rowH-2, selBGPixel)
		}

		textY := rowY + (rowH-charH)/2

		prefix := "  "
		if isSelected {
			prefix = "> "
		}
		c.fontEngine.DrawString(c.Pixels, c.Stride, modalX+12, textY, prefix+opt.Label, fgPix, bgPix, isSelected)

		if opt.IsToggle {
			badge := "[OFF]"
			badgeColor := mutedTextPixel
			if opt.Enabled {
				badge = "[ON]"
				badgeColor = th.MinDot.ToPixel() // Vivid emerald/green
			}
			badgeX := modalX + modalW - (len(badge) * charW) - 14
			c.fontEngine.DrawString(c.Pixels, c.Stride, badgeX, textY, badge, badgeColor, bgPix, true)
		} else if opt.Value != "" {
			badgeX := modalX + modalW - (len(opt.Value) * charW) - 14
			c.fontEngine.DrawString(c.Pixels, c.Stride, badgeX, textY, opt.Value, badgeTextPixel, bgPix, true)
		} else if opt.ID == savedID {
			activeBadge := "[SAVED]"
			badgeX := modalX + modalW - (len(activeBadge) * charW) - 14
			c.fontEngine.DrawString(c.Pixels, c.Stride, badgeX, textY, activeBadge, badgeTextPixel, bgPix, true)
		}
	}

	footerY := modalY + modalH - footerH
	DrawHLine(c.Pixels, c.Stride, modalX, footerY, modalW, th.HeaderLine.ToPixel())
	hints := "Click/Enter: Toggle  ESC: Close"
	hintsX := modalX + (modalW-len(hints)*charW)/2
	if hintsX < modalX+10 {
		hintsX = modalX + 10
	}
	c.fontEngine.DrawString(c.Pixels, c.Stride, hintsX, footerY+(footerH-charH)/2, hints, mutedTextPixel, modalBGPixel, false)

	DrawRectBorder(c.Pixels, c.Stride, modalX, modalY, modalW, modalH, borderPixel)

	return modalX, modalY, modalW, modalH, rowH
}

func (c *Canvas) RenderSearchBar(th *terminal.Theme, query string, matchIdx, totalMatches int) (barX, barY, barW, barH int) {
	charW := c.fontEngine.CharWidth()
	charH := c.fontEngine.CharHeight()

	barW = 380
	barH = 34
	if barW > c.Width-40 {
		barW = c.Width - 40
	}
	barX = c.Width - barW - 20
	barY = HeaderHeight + 8

	if th.ID == "minecraft" {
		FillRect(c.Pixels, c.Stride, barX, barY, barW, barH, 0x141614)
		DrawRectBorder(c.Pixels, c.Stride, barX, barY, barW, barH, 0x000000)
		DrawHLine(c.Pixels, c.Stride, barX+1, barY+1, barW-2, 0x101010)
		DrawVLine(c.Pixels, c.Stride, barX+1, barY+1, barH-2, 0x101010)
		DrawHLine(c.Pixels, c.Stride, barX+1, barY+barH-2, barW-2, 0x555555)
		DrawVLine(c.Pixels, c.Stride, barX+barW-2, barY+1, barH-2, 0x555555)

		prefix := "FIND: "
		c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, barX+12, barY+(barH-charH)/2, prefix, 0x55ffff, 0x153f3f, true)

		textX := barX + 12 + len(prefix)*charW
		dispQuery := query
		if dispQuery == "" {
			dispQuery = "_"
		}
		maxQueryChars := (barW - (len(prefix)+14)*charW)
		if len(dispQuery)*charW > maxQueryChars && maxQueryChars > 0 {
			dispQuery = dispQuery[len(dispQuery)-(maxQueryChars/charW):]
		}
		c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, textX, barY+(barH-charH)/2, dispQuery, 0xffffff, 0x3f3f3f, false)

		countStr := fmt.Sprintf("(%d/%d)", matchIdx, totalMatches)
		if totalMatches == 0 && query != "" {
			countStr = "(0/0)"
		} else if query == "" {
			countStr = ""
		}
		if countStr != "" {
			countX := barX + barW - len(countStr)*charW - 12
			if countX > textX+len(dispQuery)*charW+8 {
				cntColor := uint32(0x888888)
				if totalMatches > 0 {
					cntColor = 0x55ff55
				}
				c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, countX, barY+(barH-charH)/2, countStr, cntColor, MinecraftShadow(cntColor), false)
			}
		}

		return barX, barY, barW, barH
	}

	bgPix := th.HeaderBG.ToPixel()
	borderPix := th.Border.ToPixel()

	FillRect(c.Pixels, c.Stride, barX, barY, barW, barH, bgPix)
	DrawRectBorder(c.Pixels, c.Stride, barX, barY, barW, barH, borderPix)

	prefix := "FIND: "
	c.fontEngine.DrawString(c.Pixels, c.Stride, barX+12, barY+(barH-charH)/2, prefix, th.BadgeText.ToPixel(), bgPix, true)

	textX := barX + 12 + len(prefix)*charW
	dispQuery := query
	if dispQuery == "" {
		dispQuery = "_"
	}
	maxQueryChars := (barW - (len(prefix)+14)*charW)
	if len(dispQuery)*charW > maxQueryChars && maxQueryChars > 0 {
		dispQuery = dispQuery[len(dispQuery)-(maxQueryChars/charW):]
	}
	c.fontEngine.DrawString(c.Pixels, c.Stride, textX, barY+(barH-charH)/2, dispQuery, th.FG.ToPixel(), bgPix, false)

	countStr := fmt.Sprintf("(%d/%d)", matchIdx, totalMatches)
	if totalMatches == 0 && query != "" {
		countStr = "(0/0)"
	} else if query == "" {
		countStr = ""
	}
	if countStr != "" {
		countX := barX + barW - len(countStr)*charW - 12
		if countX > textX+len(dispQuery)*charW+8 {
			cntColor := th.MutedText.ToPixel()
			if totalMatches > 0 {
				cntColor = th.BadgeText.ToPixel()
			}
			c.fontEngine.DrawString(c.Pixels, c.Stride, countX, barY+(barH-charH)/2, countStr, cntColor, bgPix, false)
		}
	}

	return barX, barY, barW, barH
}

func (c *Canvas) RenderPasteConfirmModal(th *terminal.Theme, content string, warnings []string, isURL bool, isLarge bool) (modalX, modalY, modalW, modalH int) {
	charW := c.fontEngine.CharWidth()
	charH := c.fontEngine.CharHeight()

	allLines := strings.Split(content, "\n")
	totalLines := len(allLines)

	maxPreviewLines := 6
	previewLines := allLines
	moreLinesCount := 0
	if len(previewLines) > maxPreviewLines {
		previewLines = allLines[:maxPreviewLines-1]
		moreLinesCount = totalLines - (maxPreviewLines - 1)
	}

	previewRowH := charH + 4
	numDisplayRows := len(previewLines)
	if moreLinesCount > 0 {
		numDisplayRows++
	}
	if numDisplayRows == 0 {
		numDisplayRows = 1
	}

	headerH := 36
	bannerH := 28
	previewBoxH := numDisplayRows*previewRowH + 16
	footerH := 36
	modalH = headerH + bannerH + previewBoxH + footerH + 16

	modalW = 560
	if modalW > c.Width-40 {
		modalW = c.Width - 40
	}
	if modalH > c.Height-40 {
		modalH = c.Height - 40
	}

	modalX = (c.Width - modalW) / 2
	modalY = (c.Height - modalH) / 2

	if th.ID == "minecraft" {
		FillPattern16x16(c.Pixels, c.Stride, modalX, modalY, modalW, modalH, &mcDirt16x16)
		DrawRectBorder(c.Pixels, c.Stride, modalX, modalY, modalW, modalH, 0x000000)
		DrawRectBorder(c.Pixels, c.Stride, modalX+1, modalY+1, modalW-2, modalH-2, 0x000000)
		DrawHLine(c.Pixels, c.Stride, modalX+2, modalY+2, modalW-4, 0xc6c6c6)
		DrawVLine(c.Pixels, c.Stride, modalX+2, modalY+2, modalH-4, 0xc6c6c6)
		DrawHLine(c.Pixels, c.Stride, modalX+2, modalY+modalH-3, modalW-4, 0x373737)
		DrawVLine(c.Pixels, c.Stride, modalX+modalW-3, modalY+2, modalH-4, 0x373737)

		titleStr := "PASTE CONFIRMATION // SAFE REVIEW"
		c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, modalX+16, modalY+(headerH-charH)/2, titleStr, 0xffff55, 0x3f3f15, true)
		DrawHLine(c.Pixels, c.Stride, modalX+4, modalY+headerH-1, modalW-8, 0x140e09)

		bannerY := modalY + headerH + 6
		bannerX := modalX + 14
		bannerW := modalW - 28
		DrawMinecraftButton(c.Pixels, c.Stride, bannerX, bannerY, bannerW, bannerH, false)

		bannerMsg := "! Multiline paste detected: commands will execute immediately without confirmation!"
		if len(warnings) > 0 {
			bannerMsg = "! " + warnings[0]
		} else if isLarge {
			bannerMsg = "! Large payload detected: pasting may freeze or slow the shell!"
		}
		maxBannerChars := (bannerW - 20) / charW
		bannerMsg = truncateString(bannerMsg, maxBannerChars, "...")
		c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, bannerX+10, bannerY+(bannerH-charH)/2, bannerMsg, 0xffaa00, 0x3f2a00, true)

		boxX := modalX + 14
		boxY := bannerY + bannerH + 8
		boxW := modalW - 28
		boxH := modalH - (boxY - modalY) - footerH - 10
		if boxH < 30 {
			boxH = 30
		}
		FillRect(c.Pixels, c.Stride, boxX, boxY, boxW, boxH, 0x141614)
		DrawRectBorder(c.Pixels, c.Stride, boxX, boxY, boxW, boxH, 0x000000)
		DrawHLine(c.Pixels, c.Stride, boxX+1, boxY+1, boxW-2, 0x101010)
		DrawVLine(c.Pixels, c.Stride, boxX+1, boxY+1, boxH-2, 0x101010)
		DrawHLine(c.Pixels, c.Stride, boxX+1, boxY+boxH-2, boxW-2, 0x555555)
		DrawVLine(c.Pixels, c.Stride, boxX+boxW-2, boxY+1, boxH-2, 0x555555)

		maxLineChars := (boxW - 48) / charW
		if maxLineChars < 10 {
			maxLineChars = 10
		}
		rowStartY := boxY + 8
		for i, l := range previewLines {
			lineY := rowStartY + i*previewRowH
			if lineY+charH > boxY+boxH {
				break
			}
			numStr := fmt.Sprintf("%2d ", i+1)
			c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, boxX+10, lineY, numStr, 0x888888, 0x222222, false)

			cleanL := strings.ReplaceAll(l, "\t", "    ")
			cleanL = truncateString(cleanL, maxLineChars, "...")
			c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, boxX+10+len(numStr)*charW, lineY, cleanL, 0xe0e0e0, 0x383838, false)
		}

		footerY := modalY + modalH - footerH
		DrawHLine(c.Pixels, c.Stride, modalX+4, footerY, modalW-8, 0x140e09)

		hints := "[ENTER] Paste All   [S] Single Line"
		if isURL {
			hints += "   [Q] Quoted"
		}
		hints += "   [ESC] Cancel"

		hintsX := modalX + (modalW-len(hints)*charW)/2
		if hintsX < modalX+12 {
			hintsX = modalX + 12
		}
		c.fontEngine.DrawStringShadow(c.Pixels, c.Stride, hintsX, footerY+(footerH-charH)/2, hints, 0xa0a0a0, 0x282828, false)

		return modalX, modalY, modalW, modalH
	}

	modalBGPixel := th.HeaderBG.ToPixel()
	modalHeaderBGPixel := th.BG.ToPixel()
	borderPixel := th.Border.ToPixel()
	badgeTextPixel := th.BadgeText.ToPixel()
	mutedTextPixel := th.MutedText.ToPixel()
	fgPixel := th.FG.ToPixel()
	warnPixel := th.MinDot.ToPixel() // Amber
	if len(warnings) > 0 {
		warnPixel = th.CloseDot.ToPixel() // Red
	}

	// 1. Modal background and frame
	FillRect(c.Pixels, c.Stride, modalX, modalY, modalW, modalH, modalBGPixel)

	// 2. Header
	FillRect(c.Pixels, c.Stride, modalX, modalY, modalW, headerH, modalHeaderBGPixel)
	DrawHLine(c.Pixels, c.Stride, modalX, modalY+headerH-1, modalW, th.HeaderLine.ToPixel())

	dotSize := 8
	dotX := modalX + 14
	dotY := modalY + (headerH-dotSize)/2
	FillRect(c.Pixels, c.Stride, dotX, dotY, dotSize, dotSize, warnPixel)

	titleStr := "PASTE CONFIRMATION // SAFE REVIEW"
	c.fontEngine.DrawString(c.Pixels, c.Stride, dotX+dotSize+10, modalY+(headerH-charH)/2, titleStr, badgeTextPixel, modalHeaderBGPixel, true)

	metaStr := fmt.Sprintf("[%d lines • %d B]", totalLines, len(content))
	metaX := modalX + modalW - len(metaStr)*charW - 14
	if metaX > dotX+dotSize+10+len(titleStr)*charW+8 {
		c.fontEngine.DrawString(c.Pixels, c.Stride, metaX, modalY+(headerH-charH)/2, metaStr, mutedTextPixel, modalHeaderBGPixel, false)
	}

	// 3. Safety Warning Banner
	bannerY := modalY + headerH + 6
	bannerX := modalX + 14
	bannerW := modalW - 28
	FillRect(c.Pixels, c.Stride, bannerX, bannerY, bannerW, bannerH, modalHeaderBGPixel)
	DrawRectBorder(c.Pixels, c.Stride, bannerX, bannerY, bannerW, bannerH, warnPixel)

	bannerMsg := "Multiline paste detected: commands will execute immediately without confirmation!"
	if len(warnings) > 0 {
		bannerMsg = "! " + warnings[0]
	} else if isLarge {
		bannerMsg = "! Large payload detected: pasting may freeze or slow the shell!"
	}
	maxBannerChars := (bannerW - 20) / charW
	bannerMsg = truncateString(bannerMsg, maxBannerChars, "...")
	c.fontEngine.DrawString(c.Pixels, c.Stride, bannerX+10, bannerY+(bannerH-charH)/2, bannerMsg, warnPixel, modalHeaderBGPixel, true)

	// 4. Preview Box
	boxX := modalX + 14
	boxY := bannerY + bannerH + 8
	boxW := modalW - 28
	boxH := modalH - (boxY - modalY) - footerH - 10
	if boxH < 30 {
		boxH = 30
	}
	FillRect(c.Pixels, c.Stride, boxX, boxY, boxW, boxH, modalHeaderBGPixel)
	DrawRectBorder(c.Pixels, c.Stride, boxX, boxY, boxW, boxH, borderPixel)

	maxLineChars := (boxW - 48) / charW
	if maxLineChars < 10 {
		maxLineChars = 10
	}

	rowStartY := boxY + 8
	for i, l := range previewLines {
		lineY := rowStartY + i*previewRowH
		if lineY+charH > boxY+boxH {
			break
		}
		numStr := fmt.Sprintf("%2d ", i+1)
		c.fontEngine.DrawString(c.Pixels, c.Stride, boxX+10, lineY, numStr, mutedTextPixel, modalHeaderBGPixel, false)

		cleanL := strings.ReplaceAll(l, "\t", "    ")
		cleanL = truncateString(cleanL, maxLineChars, "...")
		c.fontEngine.DrawString(c.Pixels, c.Stride, boxX+10+len(numStr)*charW, lineY, cleanL, fgPixel, modalHeaderBGPixel, false)
	}

	if moreLinesCount > 0 {
		moreY := rowStartY + len(previewLines)*previewRowH
		if moreY+charH <= boxY+boxH {
			moreStr := fmt.Sprintf("... (+%d more lines) ...", moreLinesCount)
			c.fontEngine.DrawString(c.Pixels, c.Stride, boxX+24, moreY, moreStr, mutedTextPixel, modalHeaderBGPixel, false)
		}
	}

	// 5. Footer action hints
	footerY := modalY + modalH - footerH
	DrawHLine(c.Pixels, c.Stride, modalX, footerY, modalW, th.HeaderLine.ToPixel())

	hints := "[ENTER] Paste All   [S] Single Line"
	if isURL {
		hints += "   [Q] Quoted"
	}
	hints += "   [ESC] Cancel"

	hintsX := modalX + (modalW-len(hints)*charW)/2
	if hintsX < modalX+12 {
		hintsX = modalX + 12
	}
	c.fontEngine.DrawString(c.Pixels, c.Stride, hintsX, footerY+(footerH-charH)/2, hints, mutedTextPixel, modalBGPixel, false)

	// 6. Outer Border
	DrawRectBorder(c.Pixels, c.Stride, modalX, modalY, modalW, modalH, borderPixel)

	return modalX, modalY, modalW, modalH
}

func truncateString(s string, maxChars int, ellipsis string) string {
	if maxChars <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	eRunes := []rune(ellipsis)
	if maxChars <= len(eRunes) {
		return string(runes[:maxChars])
	}
	return string(runes[:maxChars-len(eRunes)]) + ellipsis
}
