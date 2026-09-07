package render

import (
	"fmt"
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
	minDotPixel := th.MinDot.ToPixel()
	maxDotPixel := th.MaxDot.ToPixel()
	badgeTextPixel := th.BadgeText.ToPixel()
	mutedTextPixel := th.MutedText.ToPixel()

	FillRect(c.Pixels, c.Stride, 0, 0, c.Width, c.Height, defaultBGPixel)

	FillRect(c.Pixels, c.Stride, 0, 0, c.Width, HeaderHeight, headerBGPixel)
	DrawHLine(c.Pixels, c.Stride, 0, HeaderHeight-1, c.Width, headerLinePixel)

	dotY := HeaderHeight / 2
	DrawCircle(c.Pixels, c.Stride, 20, dotY, 5, closeDotPixel)
	DrawCircle(c.Pixels, c.Stride, 36, dotY, 5, minDotPixel)
	DrawCircle(c.Pixels, c.Stride, 52, dotY, 5, maxDotPixel)

	titleY := (HeaderHeight - charH) / 2
	c.TabHitBoxes = nil
	c.NewTabHitBox = [4]int{0, 0, 0, 0}

	curXTab := 70
	availTabWidth := c.Width - 140 - curXTab - 40
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

		tabBGPixel := headerBGPixel
		tabFGPixel := mutedTextPixel
		if tab.Active {
			tabBGPixel = defaultBGPixel
			tabFGPixel = th.FG.ToPixel()
			FillRect(c.Pixels, c.Stride, tStartX, 0, tabW, HeaderHeight, tabBGPixel)
			DrawVLine(c.Pixels, c.Stride, tStartX, 0, HeaderHeight, headerLinePixel)
			DrawVLine(c.Pixels, c.Stride, tEndX-1, 0, HeaderHeight, headerLinePixel)
		} else {
			DrawVLine(c.Pixels, c.Stride, tEndX-1, 6, HeaderHeight-12, headerLinePixel)
		}

		maxChars := (tabW - 32) / charW
		disp := fmt.Sprintf("%d: %s", i+1, tab.Title)
		if tab.Title == "" {
			disp = fmt.Sprintf("%d: bash", i+1)
		}
		if len(disp) > maxChars && maxChars > 2 {
			disp = disp[:maxChars-1] + "…"
		}
		tTextY := (HeaderHeight - charH) / 2
		c.fontEngine.DrawString(c.Pixels, c.Stride, tStartX+10, tTextY, disp, tabFGPixel, tabBGPixel, tab.Active)

		closeX := tEndX - 18
		closeEndX := tEndX - 4
		c.fontEngine.DrawString(c.Pixels, c.Stride, closeX, tTextY, "×", mutedTextPixel, tabBGPixel, false)

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
	DrawRectBorder(c.Pixels, c.Stride, btnX, btnY, btnW, btnH, headerLinePixel)
	plusCharX := btnX + (btnW-charW)/2
	plusCharY := (HeaderHeight - charH) / 2
	c.fontEngine.DrawString(c.Pixels, c.Stride, plusCharX, plusCharY, "+", th.BadgeText.ToPixel(), headerBGPixel, true)
	c.NewTabHitBox = [4]int{btnX - 2, btnX + btnW + 2, 0, HeaderHeight}

	if hudInfo == "" {
		hudInfo = fmt.Sprintf("%dx%d", c.cols, c.rows)
	}
	hudLen := len(hudInfo)
	hudX := c.Width - (hudLen * charW) - 18
	if hudX > 70+(12*charW) {
		c.fontEngine.DrawString(c.Pixels, c.Stride, hudX, titleY, hudInfo, badgeTextPixel, headerBGPixel, false)
	}

	// Render diagnostic notification chip in header bar between tabs and grid size
	if diag != nil && !term.IsAltLocked() {
		diagMsg := diag.Message
		if diag.Suggestion != "" {
			diagMsg = fmt.Sprintf("%s (%s)", diag.Message, diag.Suggestion)
		}
		maxDiagChars := (hudX - (btnX + btnW + 24)) / charW
		if maxDiagChars > 10 {
			if len(diagMsg) > maxDiagChars {
				diagMsg = diagMsg[:maxDiagChars-1] + "…"
			}
			chipW := len(diagMsg)*charW + 16
			chipH := 20
			chipX := hudX - chipW - 14
			if chipX > btnX+btnW+12 {
				chipY := (HeaderHeight - chipH) / 2
				chipColor := th.MinDot.ToPixel() // Amber warning
				if diag.IsError {
					chipColor = th.CloseDot.ToPixel() // Red error
				}
				DrawRectBorder(c.Pixels, c.Stride, chipX, chipY, chipW, chipH, chipColor)
				c.fontEngine.DrawString(c.Pixels, c.Stride, chipX+8, chipY+(chipH-charH)/2, diagMsg, chipColor, headerBGPixel, true)
			}
		}
	}

	gridStartY := HeaderHeight + PaddingTop
	gridStartX := PaddingLeft
	curX, curY, curVis := term.CursorLocked()
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

			isCursor := (x == curX && y == effectiveCurY && effectiveCurY < c.rows && curVis && cursorBlink)
			if isCursor {
				fg, bg = bg, th.Cursor
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
			DrawGlyphBlit(c.Pixels, c.Stride, cellX, cellY, mask, fgPixel, bgPixel, cell.Underline || isHoveredURL, baseline)
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

	if scrollOff > 0 {
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
				thumbX := c.Width - 6
				FillRect(c.Pixels, c.Stride, thumbX, thumbY, 3, thumbH, th.MutedText.ToPixel())
			}
		}
	}

	DrawRectBorder(c.Pixels, c.Stride, 0, 0, c.Width, c.Height, borderPixel)
}

type PrefOption struct {
	ID       string
	Label    string
	Sublabel string
}

func (c *Canvas) RenderPreferencesModal(th *terminal.Theme, selectedIdx int, options []PrefOption, savedID string) (modalX, modalY, modalW, modalH, rowH int) {
	charW := c.fontEngine.CharWidth()
	charH := c.fontEngine.CharHeight()

	modalW = 440
	rowH = 28
	headerH := 36
	footerH := 32
	modalH = headerH + len(options)*rowH + footerH

	if modalW > c.Width-40 {
		modalW = c.Width - 40
	}
	if modalH > c.Height-40 {
		modalH = c.Height - 40
	}

	modalX = (c.Width - modalW) / 2
	modalY = (c.Height - modalH) / 2

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
	c.fontEngine.DrawString(c.Pixels, c.Stride, modalX+16, modalY+(headerH-charH)/2, "PREFERENCES // THEME SELECTOR", badgeTextPixel, modalHeaderBGPixel, true)

	optStartY := modalY + headerH + 4
	for i, opt := range options {
		rowY := optStartY + i*rowH
		isSelected := (i == selectedIdx)
		isSaved := (opt.ID == savedID)

		bgPix := modalBGPixel
		fgPix := fgPixel
		if isSelected {
			bgPix = selBGPixel
			fgPix = selFGPixel
			FillRect(c.Pixels, c.Stride, modalX+8, rowY, modalW-16, rowH-2, selBGPixel)
		}

		textY := rowY + (rowH-charH)/2

		prefix := "  "
		if isSelected {
			prefix = "> "
		}
		c.fontEngine.DrawString(c.Pixels, c.Stride, modalX+14, textY, prefix+opt.Label, fgPix, bgPix, isSelected)

		subX := modalX + 14 + len(prefix+opt.Label)*charW + 12
		if subX < modalX+modalW-120 {
			subColor := mutedTextPixel
			if isSelected {
				subColor = selFGPixel
			}
			c.fontEngine.DrawString(c.Pixels, c.Stride, subX, textY, "("+opt.Sublabel+")", subColor, bgPix, false)
		}

		if isSaved {
			activeBadge := "[SAVED]"
			badgeX := modalX + modalW - (len(activeBadge) * charW) - 20
			c.fontEngine.DrawString(c.Pixels, c.Stride, badgeX, textY, activeBadge, badgeTextPixel, bgPix, true)
		}
	}

	footerY := modalY + modalH - footerH
	DrawHLine(c.Pixels, c.Stride, modalX, footerY, modalW, th.HeaderLine.ToPixel())
	hints := "UP/DOWN: Preview  ENTER: Save  ESC: Exit"
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
