package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"

	"terrat/autosuggest"
	"terrat/config"
	"terrat/diagnostics"
	"terrat/paste"
	"terrat/platform"
	"terrat/pty"
	"terrat/render"
	"terrat/terminal"
)

//go:embed icon.png
var embeddedIconPNG []byte

const (
	AppName = "TerraTerminal"
)

var (
	Version   = "0.1.5"
	AppBanner = "TerraTerminal (Terrat) v" + Version + " - Blazing fast minimalist terminal for Linux"
)

type Tab struct {
	ID      int
	Title   string
	Term    *terminal.Terminal
	PTY     *pty.TerminalPTY
	ExitCh  chan struct{}
	HasBell bool
}

type tabTitleMsg struct {
	tabID int
	title string
}

var urlRegex = regexp.MustCompile(`https?://[^\s<>"'()]+`)

func main() {
	var customCmd []string
	var remainingArgs []string
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-e" {
			if i+1 < len(os.Args) {
				customCmd = os.Args[i+1:]
			}
			break
		}
		remainingArgs = append(remainingArgs, os.Args[i])
	}

	fs := flag.NewFlagSet("terrat", flag.ContinueOnError)
	verFlag := fs.Bool("v", false, "Print version and exit")
	versionFlag := fs.Bool("version", false, "Print version and exit")
	titleFlag := fs.String("title", AppName, "Set initial window title")
	fontSizeFlag := fs.Float64("font-size", 0.0, "Font size in points (defaults to config)")
	_ = fs.Parse(remainingArgs)

	if *verFlag || *versionFlag {
		fmt.Println(AppBanner)
		return
	}

	appConfig := config.Load()
	if *fontSizeFlag > 0 {
		appConfig.FontSize = *fontSizeFlag
	}

	fontEngine, err := render.NewFontEngine(appConfig.FontSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing font engine: %v\n", err)
		os.Exit(1)
	}
	defer fontEngine.Close()

	initCols := 80
	initRows := 24
	initWidth := uint16(render.PaddingLeft + (initCols * fontEngine.CharWidth()) + render.PaddingRight)
	initHeight := uint16(render.HeaderHeight + render.PaddingTop + (initRows * fontEngine.CharHeight()) + render.PaddingBottom)

	win, err := platform.NewWindow(*titleFlag, initWidth, initHeight, "", embeddedIconPNG)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating window: %v\n", err)
		os.Exit(1)
	}
	defer win.Close()

	win.SetOpacity(appConfig.Opacity)

	canvas := render.NewCanvas(fontEngine)
	canvas.Resize(int(initWidth), int(initHeight))

	systemIsDark := platform.DetectSystemColorScheme()
	activeTheme := terminal.ResolveTheme(appConfig.Theme, systemIsDark)

	prefOptions := []render.PrefOption{
		{ID: "auto", Label: "System Auto Detect", Sublabel: "Follow OS Dark/Light"},
		{ID: "tokyo-night", Label: "Tokyo Night", Sublabel: "Dark / Midnight Blue"},
		{ID: "catppuccin-mocha", Label: "Catppuccin Mocha", Sublabel: "Dark / Velvet Charcoal"},
		{ID: "tokyo-day", Label: "Tokyo Day", Sublabel: "Light / Crisp Sunlight"},
		{ID: "solarized-light", Label: "Solarized Light", Sublabel: "Light / Warm Parchment"},
	}
	isPrefOpen := false
	prefIndex := 0
	savedThemeID := appConfig.Theme
	for i, opt := range prefOptions {
		if opt.ID == savedThemeID {
			prefIndex = i
			break
		}
	}

	keyHandler, err := platform.NewKeyHandler(win.X)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to query keyboard mapping: %v\n", err)
	}

	redrawCh := make(chan struct{}, 1)
	triggerRedraw := func() {
		select {
		case redrawCh <- struct{}{}:
		default:
		}
	}

	tabExitNotifyCh := make(chan int, 16)
	ptyDataNotifyCh := make(chan int, 128)
	tabTitleNotifyCh := make(chan tabTitleMsg, 32)

	nextTabID := 1
	createTab := func(cols, rows int, width, height uint16, th *terminal.Theme, cmd ...string) (*Tab, error) {
		pMaster, err := pty.Start(uint16(cols), uint16(rows), width, height, cmd...)
		if err != nil {
			return nil, err
		}

		t := terminal.New(cols, rows)
		t.SetTheme(th)

		tab := &Tab{
			ID:     nextTabID,
			Title:  "bash",
			Term:   t,
			PTY:    pMaster,
			ExitCh: make(chan struct{}),
		}
		nextTabID++

		go func(id int, pm *pty.TerminalPTY, exitCh chan struct{}) {
			_, _ = pm.Wait()
			close(exitCh)
			tabExitNotifyCh <- id
		}(tab.ID, pMaster, tab.ExitCh)

		go func(id int, pm *pty.TerminalPTY, term *terminal.Terminal) {
			buf := make([]byte, 8192)
			for {
				n, err := pm.Read(buf)
				if n > 0 {
					_, _ = term.Write(buf[:n])
					select {
					case ptyDataNotifyCh <- id:
					default:
					}
				}
				if err != nil {
					return
				}
			}
		}(tab.ID, pMaster, t)

		go func(id int, titleChan chan string) {
			for title := range titleChan {
				tabTitleNotifyCh <- tabTitleMsg{tabID: id, title: title}
			}
		}(tab.ID, t.TitleChan)

		return tab, nil
	}

	var tabs []*Tab
	activeTabIdx := 0

	initTab, err := createTab(canvas.Cols(), canvas.Rows(), initWidth, initHeight, activeTheme, customCmd...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting shell PTY: %v\n", err)
		os.Exit(1)
	}
	tabs = append(tabs, initTab)

	currentInputBuffer := ""
	activeGhostText := ""
	var activeDiag *render.DiagnosticInfo
	var updateGhostText func()
	var updateDiagnostics func()

	isSelecting := false
	isDraggingScrollbar := false
	dragScrollDelta := 0
	lastDragCol := 0

	switchTab := func(idx int) {
		if idx < 0 || idx >= len(tabs) {
			return
		}
		isSelecting = false
		isDraggingScrollbar = false
		dragScrollDelta = 0
		activeTabIdx = idx
		tabs[activeTabIdx].HasBell = false
		title := tabs[activeTabIdx].Title
		if title == "" {
			title = "bash"
		}
		win.SetTitle(AppName + " - " + title)
		currentInputBuffer = ""
		activeDiag = nil
		if updateGhostText != nil {
			updateGhostText()
		}
		if updateDiagnostics != nil {
			updateDiagnostics()
		}
		tabs[activeTabIdx].Term.MarkAllDirty()
		triggerRedraw()
	}

	closeTab := func(idx int) {
		if idx < 0 || idx >= len(tabs) {
			return
		}
		closingTab := tabs[idx]
		_ = closingTab.PTY.Close()

		tabs = append(tabs[:idx], tabs[idx+1:]...)
		if len(tabs) == 0 {
			os.Exit(0)
		}

		if activeTabIdx >= len(tabs) {
			activeTabIdx = len(tabs) - 1
		} else if idx < activeTabIdx {
			activeTabIdx--
		}

		isSelecting = false
		isDraggingScrollbar = false
		dragScrollDelta = 0

		tabs[activeTabIdx].HasBell = false
		title := tabs[activeTabIdx].Title
		if title == "" {
			title = "bash"
		}
		win.SetTitle(AppName + " - " + title)
		currentInputBuffer = ""
		activeDiag = nil
		if updateGhostText != nil {
			updateGhostText()
		}
		if updateDiagnostics != nil {
			updateDiagnostics()
		}
		tabs[activeTabIdx].Term.MarkAllDirty()
		triggerRedraw()
	}

	xEventCh := make(chan xgb.Event, 1024)
	go func() {
		for {
			ev, err := win.X.WaitForEvent()
			if ev == nil && err == nil {
				close(xEventCh)
				return
			}
			if err != nil {
				continue
			}
			xEventCh <- ev
		}
	}()

	cursorTicker := time.NewTicker(500 * time.Millisecond)
	defer cursorTicker.Stop()
	cursorBlink := true

	currentTitle := AppName
	currentWidth := initWidth
	currentHeight := initHeight
	currentCols := canvas.Cols()
	currentRows := canvas.Rows()

	charW := fontEngine.CharWidth()
	charH := fontEngine.CharHeight()
	gridStartX := render.PaddingLeft
	gridStartY := render.HeaderHeight + render.PaddingTop

	isSearchOpen := false
	searchQuery := ""
	var searchMatches []render.SearchMatch
	activeSearchIdx := 0

	isPasteModalOpen := false
	pendingPasteText := ""
	var pendingPasteWarnings []string
	pendingPasteIsURL := false
	pendingPasteIsLarge := false

	var hoveredURL *render.URLRange

	suggestEngine := autosuggest.NewEngine()
	updateGhostText = func() {
		if !appConfig.GhostText || tabs[activeTabIdx].Term.IsAlt() || isSearchOpen || isPrefOpen || isPasteModalOpen {
			activeGhostText = ""
			return
		}
		trimmed := strings.TrimLeft(currentInputBuffer, " ")
		if len(trimmed) >= 1 {
			activeGhostText = suggestEngine.Suggest(trimmed)
		} else {
			activeGhostText = ""
		}
	}

	updateDiagnostics = func() {
		if !appConfig.Diagnostics || tabs[activeTabIdx].Term.IsAlt() || isSearchOpen || isPrefOpen || isPasteModalOpen {
			activeDiag = nil
			return
		}
		trimmed := strings.TrimLeft(currentInputBuffer, " ")
		diag := diagnostics.Analyze(trimmed)
		if diag != nil {
			activeDiag = &render.DiagnosticInfo{
				IsError:    diag.Severity == diagnostics.SeverityError,
				Message:    diag.Message,
				Suggestion: diag.Suggestion,
				QuickFix:   diag.QuickFix,
			}
		} else {
			activeDiag = nil
		}
	}

	updateSearchMatches := func() {
		searchMatches = nil
		activeSearchIdx = 0
		if searchQuery == "" {
			return
		}
		lowerQuery := strings.ToLower(searchQuery)
		activeTerm := tabs[activeTabIdx].Term
		for y := 0; y < activeTerm.Rows(); y++ {
			rowStr := strings.ToLower(activeTerm.GetRowString(y))
			idx := 0
			for {
				pos := strings.Index(rowStr[idx:], lowerQuery)
				if pos == -1 {
					break
				}
				startCol := idx + pos
				endCol := startCol + len(searchQuery) - 1
				searchMatches = append(searchMatches, render.SearchMatch{
					Row:      y,
					StartCol: startCol,
					EndCol:   endCol,
				})
				idx = startCol + 1
				if idx >= len(rowStr) {
					break
				}
			}
		}
	}

	applyZoom := func(action platform.ActionType) {
		var changed bool
		switch action {
		case platform.ActionZoomIn:
			changed = fontEngine.ZoomIn()
		case platform.ActionZoomOut:
			changed = fontEngine.ZoomOut()
		case platform.ActionZoomReset:
			changed = fontEngine.ZoomReset()
		}
		if !changed {
			return
		}

		charW = fontEngine.CharWidth()
		charH = fontEngine.CharHeight()

		canvas.Resize(int(currentWidth), int(currentHeight))
		newCols := canvas.Cols()
		newRows := canvas.Rows()

		for _, t := range tabs {
			t.Term.Resize(newCols, newRows)
			_ = t.PTY.Resize(uint16(newCols), uint16(newRows), currentWidth, currentHeight)
			t.Term.MarkAllDirty()
		}

		currentCols = newCols
		currentRows = newRows

		appConfig.FontSize = fontEngine.FontSize()
		_ = config.Save(appConfig)

		if isSearchOpen {
			updateSearchMatches()
		}

		triggerRedraw()
	}

	var doWritePastedText func(text string)
	var handlePastedData func(pastedBytes []byte)

	doWritePastedText = func(text string) {
		if len(tabs) == 0 || activeTabIdx >= len(tabs) {
			return
		}
		activeTerm := tabs[activeTabIdx].Term
		activePTY := tabs[activeTabIdx].PTY
		var toSend []byte
		if appConfig.BracketedPaste && activeTerm.BracketedPaste() {
			toSend = []byte("\x1b[200~" + text + "\x1b[201~")
		} else {
			toSend = []byte(text)
		}
		_, _ = activePTY.Write(toSend)
		activeTerm.ResetScroll()
		triggerRedraw()
	}

	handlePastedData = func(pastedBytes []byte) {
		if len(pastedBytes) == 0 || len(tabs) == 0 || activeTabIdx >= len(tabs) {
			return
		}
		raw := string(pastedBytes)
		var text string
		var warnings []string

		if appConfig.SanitizePaste {
			res := paste.Sanitize(raw)
			text = res.Sanitized
			warnings = res.Warnings
		} else {
			text = raw
		}

		if text == "" {
			return
		}

		activeTerm := tabs[activeTabIdx].Term

		// In alternate screen mode (e.g. vim, nano, htop, less), don't disrupt full-screen apps
		if activeTerm.IsAlt() {
			doWritePastedText(text)
			return
		}

		isURL := paste.IsURL(text)
		isLarge := paste.IsLargePayload(text)
		isMulti := paste.IsMultiline(text)
		hasDangerousURL := isURL && paste.HasDangerousShellChars(text)

		needConfirmation := appConfig.ConfirmMultilinePaste && (isMulti || isLarge || hasDangerousURL || len(warnings) > 0)
		if needConfirmation {
			isPasteModalOpen = true
			pendingPasteText = text
			pendingPasteWarnings = warnings
			pendingPasteIsURL = isURL
			pendingPasteIsLarge = isLarge
			triggerRedraw()
			return
		}

		doWritePastedText(text)
	}

	renderScreen := func() {
		if len(tabs) == 0 {
			return
		}
		tabInfos := make([]render.TabInfo, len(tabs))
		for i, t := range tabs {
			tabInfos[i] = render.TabInfo{
				ID:      t.ID,
				Title:   t.Title,
				Active:  i == activeTabIdx,
				HasBell: t.HasBell,
			}
		}

		activeTerm := tabs[activeTabIdx].Term
		hud := fmt.Sprintf("%dx%d", canvas.Cols(), canvas.Rows())
		canvas.Render(activeTerm, cursorBlink, currentTitle, hud, tabInfos, searchMatches, activeSearchIdx, hoveredURL, activeGhostText, activeDiag)

		if isSearchOpen {
			matchCount := 0
			if len(searchMatches) > 0 {
				matchCount = activeSearchIdx + 1
			}
			canvas.RenderSearchBar(activeTerm.Theme(), searchQuery, matchCount, len(searchMatches))
		}

		if isPrefOpen {
			canvas.RenderPreferencesModal(activeTerm.Theme(), prefIndex, prefOptions, savedThemeID)
		}

		if isPasteModalOpen {
			canvas.RenderPasteConfirmModal(activeTerm.Theme(), pendingPasteText, pendingPasteWarnings, pendingPasteIsURL, pendingPasteIsLarge)
		}
		win.Blit(canvas.Pixels, currentWidth, currentHeight)
	}

	renderScreen()

	toCellCoords := func(pixelX, pixelY int) (int, int) {
		cx := (pixelX - gridStartX) / charW
		cy := (pixelY - gridStartY) / charH
		if cx < 0 {
			cx = 0
		}
		if cx >= canvas.Cols() {
			cx = canvas.Cols() - 1
		}
		if cy < 0 {
			cy = 0
		}
		if cy >= canvas.Rows() {
			cy = canvas.Rows() - 1
		}
		return cx, cy
	}

	var (
		lastClickTime time.Time
		lastClickX    int
		lastClickY    int
		clickCount    int
	)

	autoScrollTicker := time.NewTicker(40 * time.Millisecond)
	defer autoScrollTicker.Stop()

	var pendingEvent xgb.Event

	for {
		var ev xgb.Event
		var ok bool

		if pendingEvent != nil {
			ev = pendingEvent
			pendingEvent = nil
			ok = true
		} else {
			select {
			case tabID := <-tabExitNotifyCh:
				for i, t := range tabs {
					if t.ID == tabID {
						closeTab(i)
						break
					}
				}
				continue

			case tabID := <-ptyDataNotifyCh:
				if activeTabIdx < len(tabs) && tabs[activeTabIdx].ID == tabID {
					if isSearchOpen {
						updateSearchMatches()
					}
					triggerRedraw()
				} else {
					for i := range tabs {
						if tabs[i].ID == tabID {
							tabs[i].HasBell = true
							triggerRedraw()
							break
						}
					}
				}
				continue

			case tm := <-tabTitleNotifyCh:
				for i := range tabs {
					if tabs[i].ID == tm.tabID {
						tabs[i].Title = tm.title
						if i == activeTabIdx {
							currentTitle = tm.title
							win.SetTitle(AppName + " - " + tm.title)
							triggerRedraw()
						}
						break
					}
				}
				continue

			case <-cursorTicker.C:
				cursorBlink = !cursorBlink
				triggerRedraw()
				continue

			case <-redrawCh:
				drainChannel(redrawCh)
				renderScreen()
				continue

			case <-autoScrollTicker.C:
				if isSelecting && dragScrollDelta != 0 {
					activeTerm := tabs[activeTabIdx].Term
					activeTerm.ScrollKeepSelection(dragScrollDelta)
					if dragScrollDelta > 0 {
						activeTerm.UpdateSelection(lastDragCol, 0)
					} else {
						activeTerm.UpdateSelection(lastDragCol, canvas.Rows()-1)
					}
					triggerRedraw()
				}
				continue

			case ev, ok = <-xEventCh:
				if !ok {
					return
				}
			}
		}

		activeTerm := tabs[activeTabIdx].Term
		activePTY := tabs[activeTabIdx].PTY

		switch e := ev.(type) {
		case xproto.MotionNotifyEvent:
			lastMotion := e
		drainMotion:
			for {
				select {
				case nextEv, ok := <-xEventCh:
					if !ok {
						return
					}
					if m, isMotion := nextEv.(xproto.MotionNotifyEvent); isMotion {
						lastMotion = m
						continue
					}
					pendingEvent = nextEv
					break drainMotion
				default:
					break drainMotion
				}
			}
			e = lastMotion

			if (e.State & xproto.ButtonMask1) == 0 {
				if isSelecting {
					isSelecting = false
					dragScrollDelta = 0
				}
				if isDraggingScrollbar {
					isDraggingScrollbar = false
				}
			}

			if isDraggingScrollbar {
				maxScroll := activeTerm.ScrollbackLen()
				if maxScroll > 0 {
					trackY := render.HeaderHeight + 4
					trackH := int(currentHeight) - render.HeaderHeight - 8
					if trackH > 24 {
						totalLines := maxScroll + canvas.Rows()
						thumbH := (canvas.Rows() * trackH) / totalLines
						if thumbH < 20 {
							thumbH = 20
						}
						if thumbH > trackH {
							thumbH = trackH
						}
						availH := trackH - thumbH
						if availH > 0 {
							clickY := int(e.EventY) - trackY - (thumbH / 2)
							if clickY < 0 {
								clickY = 0
							}
							if clickY > availH {
								clickY = availH
							}
							progress := float64(clickY) / float64(availH)
							targetOff := int(float64(maxScroll)*(1.0-progress) + 0.5)
							activeTerm.SetScrollOff(targetOff)
							triggerRedraw()
						}
					}
				}
				continue
			}

			if (e.State&xproto.ButtonMask1) != 0 && isSelecting {
				rawY := int(e.EventY)
				rawX := int(e.EventX)
				gridEndY := gridStartY + (canvas.Rows() * charH)
				cx, cy := toCellCoords(rawX, rawY)
				lastDragCol = cx

				if rawY < gridStartY {
					dist := (gridStartY-rawY)/charH + 1
					if dist > 5 {
						dist = 5
					}
					dragScrollDelta = dist
					activeTerm.ScrollKeepSelection(dragScrollDelta)
					activeTerm.UpdateSelection(cx, 0)
				} else if rawY >= gridEndY {
					dist := (rawY-gridEndY)/charH + 1
					if dist > 5 {
						dist = 5
					}
					dragScrollDelta = -dist
					activeTerm.ScrollKeepSelection(dragScrollDelta)
					activeTerm.UpdateSelection(cx, canvas.Rows()-1)
				} else {
					dragScrollDelta = 0
					activeTerm.UpdateSelection(cx, cy)
				}
				triggerRedraw()
			} else {
				isCtrl := (e.State & platform.ModCtrl) != 0
				if isCtrl && int(e.EventY) >= render.HeaderHeight {
					cx, cy := toCellCoords(int(e.EventX), int(e.EventY))
					rowStr := activeTerm.GetRowString(cy)
					matches := urlRegex.FindAllStringIndex(rowStr, -1)
					var foundURL *render.URLRange
					for _, m := range matches {
						startCol, endCol := m[0], m[1]
						if cx >= startCol && cx < endCol {
							rawURL := rowStr[startCol:endCol]
							urlStr := strings.TrimRight(rawURL, ".,;:!?)'\"")
							foundURL = &render.URLRange{
								Row:      cy,
								StartCol: startCol,
								EndCol:   startCol + len(urlStr) - 1,
								URL:      urlStr,
							}
							break
						}
					}
					if foundURL != nil {
						if hoveredURL == nil || hoveredURL.URL != foundURL.URL || hoveredURL.Row != foundURL.Row || hoveredURL.StartCol != foundURL.StartCol {
							hoveredURL = foundURL
							win.SetCursorType(platform.CursorPointer)
							triggerRedraw()
						}
					} else {
						if hoveredURL != nil {
							hoveredURL = nil
							win.SetCursorType(platform.CursorText)
							triggerRedraw()
						}
					}
				} else {
					if hoveredURL != nil {
						hoveredURL = nil
						triggerRedraw()
					}

					dir := platform.GetResizeDirection(int(e.EventX), int(e.EventY), int(currentWidth), int(currentHeight), 8)
					switch dir {
					case platform.ResizeTopLeft:
						win.SetCursorType(platform.CursorResizeTopLeft)
					case platform.ResizeTop:
						win.SetCursorType(platform.CursorResizeTop)
					case platform.ResizeTopRight:
						win.SetCursorType(platform.CursorResizeTopRight)
					case platform.ResizeRight:
						win.SetCursorType(platform.CursorResizeRight)
					case platform.ResizeBottomRight:
						win.SetCursorType(platform.CursorResizeBottomRight)
					case platform.ResizeBottom:
						win.SetCursorType(platform.CursorResizeBottom)
					case platform.ResizeBottomLeft:
						win.SetCursorType(platform.CursorResizeBottomLeft)
					case platform.ResizeLeft:
						win.SetCursorType(platform.CursorResizeLeft)
					default:
						if int(e.EventX) >= int(currentWidth)-16 && int(e.EventY) >= render.HeaderHeight && activeTerm.ScrollbackLen() > 0 {
							win.SetCursorType(platform.CursorDefault)
						} else if int(e.EventY) < render.HeaderHeight {
							mx := int(e.EventX)
							isPointer := false
							if canvas.NewTabHitBox[1] > 0 && mx >= canvas.NewTabHitBox[0] && mx <= canvas.NewTabHitBox[1] {
								isPointer = true
							} else if mx <= 65 {
								isPointer = true
							} else {
								for _, hb := range canvas.TabHitBoxes {
									if mx >= hb.CloseX && mx <= hb.CloseEndX {
										isPointer = true
										break
									}
								}
							}
							if isPointer {
								win.SetCursorType(platform.CursorPointer)
							} else {
								win.SetCursorType(platform.CursorDefault)
							}
						} else {
							win.SetCursorType(platform.CursorText)
						}
					}
				}
			}

		case xproto.ButtonPressEvent:
			isCtrl := (e.State & platform.ModCtrl) != 0

			if isCtrl {
				if e.Detail == 4 {
					applyZoom(platform.ActionZoomIn)
					continue
				} else if e.Detail == 5 {
					applyZoom(platform.ActionZoomOut)
					continue
				}
			}

			if e.Detail == 1 {
				if isPasteModalOpen {
					isPasteModalOpen = false
					pendingPasteText = ""
					pendingPasteWarnings = nil
					triggerRedraw()
					continue
				}

				if isCtrl && hoveredURL != nil {
					_ = exec.Command("xdg-open", hoveredURL.URL).Start()
					continue
				}

				if isPrefOpen {
					modalW := 440
					rowH := 28
					headerH := 36
					footerH := 32
					modalH := headerH + len(prefOptions)*rowH + footerH
					if modalW > int(currentWidth)-40 {
						modalW = int(currentWidth) - 40
					}
					if modalH > int(currentHeight)-40 {
						modalH = int(currentHeight) - 40
					}
					modalX := (int(currentWidth) - modalW) / 2
					modalY := (int(currentHeight) - modalH) / 2

					px := int(e.EventX)
					py := int(e.EventY)

					if px >= modalX && px < modalX+modalW && py >= modalY && py < modalY+modalH {
						optStartY := modalY + headerH + 4
						if py >= optStartY && py < optStartY+len(prefOptions)*rowH {
							clickedIdx := (py - optStartY) / rowH
							if clickedIdx >= 0 && clickedIdx < len(prefOptions) {
								prefIndex = clickedIdx
								appConfig.Theme = prefOptions[prefIndex].ID
								savedThemeID = appConfig.Theme
								_ = config.Save(appConfig)
								activeTheme = terminal.ResolveTheme(savedThemeID, systemIsDark)
								for _, t := range tabs {
									t.Term.SetTheme(activeTheme)
									t.Term.MarkAllDirty()
								}
								isPrefOpen = false
								triggerRedraw()
							}
						}
					} else {
						activeTheme = terminal.ResolveTheme(savedThemeID, systemIsDark)
						for _, t := range tabs {
							t.Term.SetTheme(activeTheme)
							t.Term.MarkAllDirty()
						}
						isPrefOpen = false
						triggerRedraw()
					}
					continue
				}

				dir := platform.GetResizeDirection(int(e.EventX), int(e.EventY), int(currentWidth), int(currentHeight), 8)
				if dir != platform.ResizeNone {
					win.StartResize(dir, e.RootX, e.RootY)
					continue
				}

				if int(e.EventY) < render.HeaderHeight {
					if e.EventX >= 12 && e.EventX <= 26 {
						closeTab(activeTabIdx)
						continue
					} else if e.EventX >= 28 && e.EventX <= 42 {
						win.Minimize()
						continue
					} else if e.EventX >= 44 && e.EventX <= 58 {
						win.ToggleMaximize()
						continue
					} else if int(e.EventX) >= int(currentWidth)-90 {
						isPrefOpen = !isPrefOpen
						if isPrefOpen {
							for i, opt := range prefOptions {
								if opt.ID == savedThemeID {
									prefIndex = i
									break
								}
							}
						}
						activeTerm.MarkAllDirty()
						triggerRedraw()
						continue
					}

					if canvas.NewTabHitBox[1] > 0 && int(e.EventX) >= canvas.NewTabHitBox[0] && int(e.EventX) <= canvas.NewTabHitBox[1] {
						newTab, err := createTab(canvas.Cols(), canvas.Rows(), currentWidth, currentHeight, activeTheme)
						if err == nil {
							tabs = append(tabs, newTab)
							switchTab(len(tabs) - 1)
						}
						continue
					}

					tabHandled := false
					for _, thb := range canvas.TabHitBoxes {
						if int(e.EventX) >= thb.StartX && int(e.EventX) <= thb.EndX {
							tabHandled = true
							if int(e.EventX) >= thb.CloseX && int(e.EventX) <= thb.CloseEndX {
								for tIdx, t := range tabs {
									if t.ID == thb.ID {
										closeTab(tIdx)
										break
									}
								}
							} else {
								for tIdx, t := range tabs {
									if t.ID == thb.ID {
										switchTab(tIdx)
										break
									}
								}
							}
							break
						}
					}
					if tabHandled {
						continue
					}

					win.StartDrag(e.RootX, e.RootY)
					continue
				}

				if int(e.EventX) >= int(currentWidth)-16 && int(e.EventY) >= render.HeaderHeight {
					maxScroll := activeTerm.ScrollbackLen()
					if maxScroll > 0 {
						isDraggingScrollbar = true
						trackY := render.HeaderHeight + 4
						trackH := int(currentHeight) - render.HeaderHeight - 8
						if trackH > 24 {
							totalLines := maxScroll + canvas.Rows()
							thumbH := (canvas.Rows() * trackH) / totalLines
							if thumbH < 20 {
								thumbH = 20
							}
							if thumbH > trackH {
								thumbH = trackH
							}
							availH := trackH - thumbH
							if availH > 0 {
								clickY := int(e.EventY) - trackY - (thumbH / 2)
								if clickY < 0 {
									clickY = 0
								}
								if clickY > availH {
									clickY = availH
								}
								progress := float64(clickY) / float64(availH)
								targetOff := int(float64(maxScroll)*(1.0-progress) + 0.5)
								activeTerm.SetScrollOff(targetOff)
								triggerRedraw()
							}
						}
						continue
					}
				}

				cx, cy := toCellCoords(int(e.EventX), int(e.EventY))
				now := time.Now()
				if now.Sub(lastClickTime) < 350*time.Millisecond && cx == lastClickX && cy == lastClickY {
					clickCount++
				} else {
					clickCount = 1
				}
				lastClickTime = now
				lastClickX = cx
				lastClickY = cy

				if clickCount == 1 {
					isSelecting = true
					activeTerm.StartSelection(cx, cy)
					triggerRedraw()
				} else if clickCount == 2 {
					isSelecting = true
					activeTerm.SelectWord(cx, cy)
					if txt := activeTerm.GetSelectedText(); txt != "" {
						win.SetPrimary(txt)
					}
					triggerRedraw()
				} else if clickCount >= 3 {
					isSelecting = true
					activeTerm.SelectLine(cy)
					if txt := activeTerm.GetSelectedText(); txt != "" {
						win.SetPrimary(txt)
					}
					triggerRedraw()
				}
			} else if e.Detail == 2 {
				win.RequestPaste(xproto.AtomPrimary)
			} else if e.Detail == 4 {
				if isPrefOpen {
					if prefIndex > 0 {
						prefIndex--
					} else {
						prefIndex = len(prefOptions) - 1
					}
					activeTheme = terminal.ResolveTheme(prefOptions[prefIndex].ID, systemIsDark)
					for _, t := range tabs {
						t.Term.SetTheme(activeTheme)
						t.Term.MarkAllDirty()
					}
					triggerRedraw()
					continue
				}
				if (e.State & platform.ModCtrl) != 0 {
					applyZoom(platform.ActionZoomIn)
					continue
				}
				if activeTerm.IsAlt() {
					_, _ = activePTY.Write([]byte("\x1b[A\x1b[A\x1b[A"))
				} else {
					activeTerm.Scroll(3)
					triggerRedraw()
				}
			} else if e.Detail == 5 {
				if isPrefOpen {
					if prefIndex < len(prefOptions)-1 {
						prefIndex++
					} else {
						prefIndex = 0
					}
					activeTheme = terminal.ResolveTheme(prefOptions[prefIndex].ID, systemIsDark)
					for _, t := range tabs {
						t.Term.SetTheme(activeTheme)
						t.Term.MarkAllDirty()
					}
					triggerRedraw()
					continue
				}
				if (e.State & platform.ModCtrl) != 0 {
					applyZoom(platform.ActionZoomOut)
					continue
				}
				if activeTerm.IsAlt() {
					_, _ = activePTY.Write([]byte("\x1b[B\x1b[B\x1b[B"))
				} else {
					activeTerm.Scroll(-3)
					triggerRedraw()
				}
			}

		case xproto.ButtonReleaseEvent:
			if e.Detail == 1 {
				if isDraggingScrollbar {
					isDraggingScrollbar = false
				}
				if isSelecting {
					isSelecting = false
					dragScrollDelta = 0
					if activeTerm.HasSelection() {
						txt := activeTerm.GetSelectedText()
						if txt != "" {
							win.SetPrimary(txt)
						}
					} else {
						activeTerm.ClearSelection()
						triggerRedraw()
					}
				}
			}

		case xproto.KeyPressEvent:
			if keyHandler != nil {
				keysym := keyHandler.KeySym(e)
				data, action := keyHandler.Translate(e)

				if action == platform.ActionPreferences {
					isPrefOpen = !isPrefOpen
					if isPrefOpen {
						for i, opt := range prefOptions {
							if opt.ID == savedThemeID {
								prefIndex = i
								break
							}
						}
					} else {
						activeTheme = terminal.ResolveTheme(savedThemeID, systemIsDark)
						for _, t := range tabs {
							t.Term.SetTheme(activeTheme)
							t.Term.MarkAllDirty()
						}
					}
					triggerRedraw()
					continue
				}

				if isPasteModalOpen {
					switch keysym {
					case 0xff1b, 'c', 'C', 'n', 'N': // Escape, 'c', 'n' -> Cancel
						isPasteModalOpen = false
						pendingPasteText = ""
						pendingPasteWarnings = nil
						triggerRedraw()
					case 0xff0d, 'p', 'P', 'y', 'Y': // Enter, 'p', 'y' -> Paste as-is
						textToPaste := pendingPasteText
						isPasteModalOpen = false
						pendingPasteText = ""
						pendingPasteWarnings = nil
						doWritePastedText(textToPaste)
					case 's', 'S': // 's' -> Flatten to single line
						textToPaste := paste.FlattenToSingleLine(pendingPasteText)
						isPasteModalOpen = false
						pendingPasteText = ""
						pendingPasteWarnings = nil
						doWritePastedText(textToPaste)
					case 'q', 'Q': // 'q' -> Quote URL / argument
						textToPaste := paste.QuoteURL(pendingPasteText)
						isPasteModalOpen = false
						pendingPasteText = ""
						pendingPasteWarnings = nil
						doWritePastedText(textToPaste)
					}
					continue
				}

				if isPrefOpen {
					switch keysym {
					case 0xff52, 'k', 'K':
						if prefIndex > 0 {
							prefIndex--
						} else {
							prefIndex = len(prefOptions) - 1
						}
						activeTheme = terminal.ResolveTheme(prefOptions[prefIndex].ID, systemIsDark)
						for _, t := range tabs {
							t.Term.SetTheme(activeTheme)
							t.Term.MarkAllDirty()
						}
						triggerRedraw()
					case 0xff54, 'j', 'J':
						if prefIndex < len(prefOptions)-1 {
							prefIndex++
						} else {
							prefIndex = 0
						}
						activeTheme = terminal.ResolveTheme(prefOptions[prefIndex].ID, systemIsDark)
						for _, t := range tabs {
							t.Term.SetTheme(activeTheme)
							t.Term.MarkAllDirty()
						}
						triggerRedraw()
					case 0xff0d:
						appConfig.Theme = prefOptions[prefIndex].ID
						savedThemeID = appConfig.Theme
						_ = config.Save(appConfig)
						activeTheme = terminal.ResolveTheme(savedThemeID, systemIsDark)
						for _, t := range tabs {
							t.Term.SetTheme(activeTheme)
							t.Term.MarkAllDirty()
						}
						isPrefOpen = false
						triggerRedraw()
					case 0xff1b, 'q', 'Q':
						activeTheme = terminal.ResolveTheme(savedThemeID, systemIsDark)
						for _, t := range tabs {
							t.Term.SetTheme(activeTheme)
							t.Term.MarkAllDirty()
						}
						isPrefOpen = false
						triggerRedraw()
					case '1', '2', '3', '4', '5':
						idx := int(keysym - '1')
						if idx >= 0 && idx < len(prefOptions) {
							prefIndex = idx
							appConfig.Theme = prefOptions[prefIndex].ID
							savedThemeID = appConfig.Theme
							_ = config.Save(appConfig)
							activeTheme = terminal.ResolveTheme(savedThemeID, systemIsDark)
							for _, t := range tabs {
								t.Term.SetTheme(activeTheme)
								t.Term.MarkAllDirty()
							}
							isPrefOpen = false
							triggerRedraw()
						}
					}
					continue
				}

				if action == platform.ActionSearch {
					isSearchOpen = !isSearchOpen
					if !isSearchOpen {
						searchMatches = nil
					} else {
						updateSearchMatches()
					}
					triggerRedraw()
					continue
				}

				if isSearchOpen {
					isShift := (e.State & platform.ModShift) != 0
					isCtrl := (e.State & platform.ModCtrl) != 0
					isAlt := (e.State & platform.ModAlt) != 0

					switch keysym {
					case 0xff1b:
						isSearchOpen = false
						searchMatches = nil
						triggerRedraw()
					case 0xff0d:
						if len(searchMatches) > 0 {
							if isShift {
								activeSearchIdx = (activeSearchIdx - 1 + len(searchMatches)) % len(searchMatches)
							} else {
								activeSearchIdx = (activeSearchIdx + 1) % len(searchMatches)
							}
							triggerRedraw()
						}
					case 0xff08:
						if len(searchQuery) > 0 {
							searchQuery = searchQuery[:len(searchQuery)-1]
							updateSearchMatches()
							triggerRedraw()
						}
					case 0xff52:
						if len(searchMatches) > 0 {
							activeSearchIdx = (activeSearchIdx - 1 + len(searchMatches)) % len(searchMatches)
							triggerRedraw()
						}
					case 0xff54:
						if len(searchMatches) > 0 {
							activeSearchIdx = (activeSearchIdx + 1) % len(searchMatches)
							triggerRedraw()
						}
					default:
						if !isCtrl && !isAlt && keysym >= 32 && keysym <= 126 {
							searchQuery += string(rune(keysym))
							updateSearchMatches()
							triggerRedraw()
						}
					}
					continue
				}

				if action == platform.ActionZoomIn || action == platform.ActionZoomOut || action == platform.ActionZoomReset {
					applyZoom(action)
					continue
				}

				// Shortcut to toggle Live Diagnostics on/off: Ctrl + Shift + D
				if (e.State&platform.ModCtrl) != 0 && (e.State&platform.ModShift) != 0 && (keysym == 'D' || keysym == 'd') {
					appConfig.Diagnostics = !appConfig.Diagnostics
					_ = config.Save(appConfig)
					if !appConfig.Diagnostics {
						activeDiag = nil
					} else {
						updateDiagnostics()
					}
					triggerRedraw()
					continue
				}

				if action == platform.ActionNewTab {
					newTab, err := createTab(canvas.Cols(), canvas.Rows(), currentWidth, currentHeight, activeTheme)
					if err == nil {
						tabs = append(tabs, newTab)
						switchTab(len(tabs) - 1)
					}
					continue
				} else if action == platform.ActionCloseTab {
					closeTab(activeTabIdx)
					continue
				} else if action == platform.ActionNextTab {
					if len(tabs) > 1 {
						switchTab((activeTabIdx + 1) % len(tabs))
					}
					continue
				} else if action == platform.ActionPrevTab {
					if len(tabs) > 1 {
						switchTab((activeTabIdx - 1 + len(tabs)) % len(tabs))
					}
					continue
				} else if action >= platform.ActionSwitchTab1 && action <= platform.ActionSwitchTab9 {
					tIdx := int(action - platform.ActionSwitchTab1)
					if tIdx < len(tabs) {
						switchTab(tIdx)
					}
					continue
				}

				if action == platform.ActionCopy {
					if activeTerm.HasSelection() {
						win.SetClipboard(activeTerm.GetSelectedText())
						activeTerm.ClearSelection()
						triggerRedraw()
					}
				} else if action == platform.ActionPaste {
					win.RequestPaste(win.AtomClipboard)
				} else if action == platform.ActionSelectAll {
					activeTerm.SelectAll()
					triggerRedraw()
					continue
				} else if action == platform.ActionScrollUp {
					if activeTerm.IsAlt() {
						_, _ = activePTY.Write([]byte("\x1b[5~"))
					} else {
						activeTerm.Scroll(canvas.Rows() / 2)
						triggerRedraw()
					}
				} else if action == platform.ActionScrollDown {
					if activeTerm.IsAlt() {
						_, _ = activePTY.Write([]byte("\x1b[6~"))
					} else {
						activeTerm.Scroll(-canvas.Rows() / 2)
						triggerRedraw()
					}
				} else if action == platform.ActionScrollTop {
					activeTerm.ScrollToTop()
					triggerRedraw()
				} else if action == platform.ActionScrollBottom {
					activeTerm.ResetScroll()
					triggerRedraw()
				} else if len(data) > 0 {
					if len(data) == 1 && data[0] == 0x03 && activeTerm.HasSelection() {
						win.SetClipboard(activeTerm.GetSelectedText())
						activeTerm.ClearSelection()
						triggerRedraw()
					} else if len(data) == 1 && data[0] == 0x16 && !activeTerm.IsAlt() {
						win.RequestPaste(win.AtomClipboard)
					} else {
						// Ghost text completion: when Right Arrow (0xff53) or Tab (0xff09) is pressed with an active suggestion
						if activeGhostText != "" && !activeTerm.IsAlt() {
							isAcceptKey := (keysym == 0xff53) || (keysym == 0xff09 && (e.State&platform.ModShift) == 0)
							if isAcceptKey {
								toWrite := []byte(activeGhostText)
								_, _ = activePTY.Write(toWrite)
								currentInputBuffer += activeGhostText
								activeGhostText = ""
								activeTerm.ClearSelection()
								activeTerm.ResetScroll()
								triggerRedraw()
								continue
							}
						}

						// Track input buffer for ghost text matching and live diagnostics
						if !activeTerm.IsAlt() {
							isAlt := (e.State & platform.ModAlt) != 0
							if isAlt && (keysym == 0xff0d || keysym == 0xff8d) && activeDiag != nil {
								// Alt+Enter applies quickfix if available!
								diag := diagnostics.Analyze(strings.TrimSpace(currentInputBuffer))
								if diag != nil && diag.QuickFix != "" {
									// Erase current line in terminal: send Ctrl+U (\x15), then type quickfix
									_, _ = activePTY.Write([]byte{0x15})
									_, _ = activePTY.Write([]byte(diag.QuickFix))
									currentInputBuffer = diag.QuickFix
									updateGhostText()
									updateDiagnostics()
									activeTerm.ClearSelection()
									activeTerm.ResetScroll()
									triggerRedraw()
									continue
								}
							}

							if keysym == 0xff0d || keysym == 0xff8d { // Enter
								trimmedCmd := strings.TrimSpace(currentInputBuffer)
								if len(trimmedCmd) >= 2 {
									suggestEngine.Add(trimmedCmd)
									if strings.HasPrefix(trimmedCmd, "alias ") || strings.HasPrefix(trimmedCmd, "abbr ") {
										diagnostics.RegisterAliasFromLine(trimmedCmd)
									}
								}
								currentInputBuffer = ""
								activeGhostText = ""
								activeDiag = nil
							} else if keysym == 0xff08 { // Backspace
								if len(currentInputBuffer) > 0 {
									currentInputBuffer = currentInputBuffer[:len(currentInputBuffer)-1]
									updateGhostText()
									updateDiagnostics()
								}
							} else if keysym == 0xff1b || (len(data) == 1 && (data[0] == 0x03 || data[0] == 0x15)) { // Escape, Ctrl+C, Ctrl+U
								currentInputBuffer = ""
								activeGhostText = ""
								activeDiag = nil
							} else if len(data) == 1 && data[0] >= 32 && data[0] <= 126 { // Normal printable ASCII
								currentInputBuffer += string(data[0])
								updateGhostText()
								updateDiagnostics()
							}
						}

						activeTerm.ClearSelection()
						_, _ = activePTY.Write(data)
						activeTerm.ResetScroll()
						triggerRedraw()
					}
				}
			}

		case xproto.SelectionRequestEvent:
			win.HandleSelectionRequest(e)

		case xproto.SelectionNotifyEvent:
			pastedBytes := win.HandleSelectionNotify(e)
			if len(pastedBytes) > 0 {
				handlePastedData(pastedBytes)
			}

		case xproto.SelectionClearEvent:

		case xproto.ConfigureNotifyEvent:
			lastCfg := e
		drainCfg:
			for {
				select {
				case nextEv, ok := <-xEventCh:
					if !ok {
						return
					}
					if cfg, isCfg := nextEv.(xproto.ConfigureNotifyEvent); isCfg {
						lastCfg = cfg
						continue
					}
					pendingEvent = nextEv
					break drainCfg
				default:
					break drainCfg
				}
			}
			e = lastCfg

			if e.Width != currentWidth || e.Height != currentHeight {
				currentWidth = e.Width
				currentHeight = e.Height

				canvas.Resize(int(currentWidth), int(currentHeight))
				newCols := canvas.Cols()
				newRows := canvas.Rows()

				if newCols > 0 && newRows > 0 && (newCols != currentCols || newRows != currentRows) {
					currentCols = newCols
					currentRows = newRows
					for _, t := range tabs {
						t.Term.Resize(newCols, newRows)
						_ = t.PTY.Resize(uint16(newCols), uint16(newRows), currentWidth, currentHeight)
						t.Term.MarkAllDirty()
					}
				}

				if isSearchOpen {
					updateSearchMatches()
				}

				activeTerm.MarkAllDirty()
				renderScreen()
			}

		case xproto.ExposeEvent:
			activeTerm.MarkAllDirty()
			renderScreen()

		case xproto.MappingNotifyEvent:
			if keyHandler != nil {
				_ = keyHandler.RefreshMapping(win.X)
			}

		case xproto.FocusOutEvent:
			isSelecting = false
			if hoveredURL != nil {
				hoveredURL = nil
				triggerRedraw()
			}

		case xproto.ClientMessageEvent:
			if e.Format == 32 && len(e.Data.Data32) > 0 && xproto.Atom(e.Data.Data32[0]) == win.AtomWmDeleteWindow {
				return
			}
		}

		runtime.Gosched()
	}
}

func drainChannel(ch chan struct{}) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
