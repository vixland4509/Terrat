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
	"sync"
	"time"

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
	ID        int
	Title     string
	Term      *terminal.Terminal
	PTY       *pty.TerminalPTY
	ExitCh    chan struct{}
	closeOnce sync.Once
	HasBell   bool
}

func (t *Tab) Close() {
	t.closeOnce.Do(func() {
		close(t.ExitCh)
	})
	if t.PTY != nil {
		_ = t.PTY.Close()
	}
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
	themeFlag := fs.String("theme", "", "Color theme override (tokyo-night, catppuccin-mocha, minecraft, tokyo-day, solarized-light)")
	_ = fs.Parse(remainingArgs)

	if *verFlag || *versionFlag {
		fmt.Println(AppBanner)
		return
	}

	appConfig := config.Load()
	if *fontSizeFlag > 0 {
		appConfig.FontSize = *fontSizeFlag
	}
	if *themeFlag != "" {
		appConfig.Theme = *themeFlag
	}

	fontEngine, err := render.NewFontEngine(appConfig.FontSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing font engine: %v\n", err)
		os.Exit(1)
	}
	defer fontEngine.Close()

	initCols := 80
	initRows := 24
	if runtime.GOOS == "windows" {
		initCols = 128
		initRows = 33
	}
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

	isPrefOpen := false
	prefIndex := 0
	savedThemeID := appConfig.Theme

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

		go func(id int, tab *Tab) {
			_, _ = tab.PTY.Wait()
			tab.Close()
			tabExitNotifyCh <- id
		}(tab.ID, tab)

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

		go func(id int, titleChan chan string, exitCh chan struct{}) {
			for {
				select {
				case <-exitCh:
					return
				case title, ok := <-titleChan:
					if !ok {
						return
					}
					select {
					case tabTitleNotifyCh <- tabTitleMsg{tabID: id, title: title}:
					case <-exitCh:
						return
					}
				}
			}
		}(tab.ID, t.TitleChan, tab.ExitCh)

		go func(pm *pty.TerminalPTY, respCh chan []byte, exitCh chan struct{}) {
			for {
				select {
				case <-exitCh:
					return
				case resp, ok := <-respCh:
					if !ok {
						return
					}
					if len(resp) > 0 {
						_, _ = pm.Write(resp)
					}
				}
			}
		}(pMaster, t.ResponseChan, tab.ExitCh)

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
		triggerRedraw()
	}

	closeTab := func(idx int) {
		if idx < 0 || idx >= len(tabs) {
			return
		}
		closingTab := tabs[idx]
		closingTab.Close()

		tabs = append(tabs[:idx], tabs[idx+1:]...)
		if len(tabs) == 0 {
			os.Exit(0)
		}

		targetIdx := activeTabIdx
		if targetIdx >= len(tabs) {
			targetIdx = len(tabs) - 1
		} else if idx < targetIdx {
			targetIdx--
		}

		switchTab(targetIdx)
	}

	xEventCh := win.Events()

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

	updateScrollbar := func(eventY int) {
		activeTerm := tabs[activeTabIdx].Term
		maxScroll := activeTerm.ScrollbackLen()
		if maxScroll <= 0 {
			return
		}
		trackY := render.HeaderHeight + 4
		trackH := int(currentHeight) - render.HeaderHeight - 8
		if trackH <= 24 {
			return
		}
		totalLines := maxScroll + canvas.Rows()
		thumbH := (canvas.Rows() * trackH) / totalLines
		if thumbH < 20 {
			thumbH = 20
		}
		if thumbH > trackH {
			thumbH = trackH
		}
		availH := trackH - thumbH
		if availH <= 0 {
			return
		}
		clickY := eventY - trackY - (thumbH / 2)
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

	isPasswordPrompt := func(term *terminal.Terminal) bool {
		if term == nil {
			return false
		}
		_, cy, _ := term.Cursor()
		rowStr := strings.ToLower(term.GetRowString(cy))
		lower := strings.TrimSpace(rowStr)
		if strings.Contains(lower, "[sudo]") {
			return true
		}
		if strings.HasSuffix(lower, ":") || strings.HasSuffix(lower, ": ") {
			if strings.Contains(lower, "password") || strings.Contains(lower, "passphrase") || strings.Contains(lower, "pin") {
				return true
			}
		}
		return false
	}

	suggestEngine := autosuggest.NewEngine()
	updateGhostText = func() {
		if !appConfig.GhostText || tabs[activeTabIdx].Term.IsAlt() || isSearchOpen || isPrefOpen || isPasteModalOpen {
			activeGhostText = ""
			return
		}
		curPTY := tabs[activeTabIdx].PTY
		if !curPTY.IsForegroundShell() || isPasswordPrompt(tabs[activeTabIdx].Term) {
			activeGhostText = ""
			return
		}
		trimmed := strings.TrimLeft(currentInputBuffer, " ")
		if len(trimmed) >= 1 {
			activeGhostText = suggestEngine.Suggest(trimmed, curPTY.GetCwd())
		} else {
			activeGhostText = ""
		}
	}

	updateDiagnostics = func() {
		if !appConfig.Diagnostics || tabs[activeTabIdx].Term.IsAlt() || isSearchOpen || isPrefOpen || isPasteModalOpen {
			activeDiag = nil
			return
		}
		curPTY := tabs[activeTabIdx].PTY
		if !curPTY.IsForegroundShell() || isPasswordPrompt(tabs[activeTabIdx].Term) {
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

	getSettingsItems := func() []render.PrefOption {
		themeDisplay := appConfig.Theme
		switch appConfig.Theme {
		case "auto":
			themeDisplay = "Auto"
		case "tokyo-night":
			themeDisplay = "Tokyo Night"
		case "catppuccin-mocha":
			themeDisplay = "Catppuccin"
		case "minecraft":
			themeDisplay = "Minecraft"
		case "tokyo-day":
			themeDisplay = "Tokyo Day"
		case "solarized-light":
			themeDisplay = "Solarized"
		}

		return []render.PrefOption{
			{
				ID:       "diagnostics",
				Label:    "Live Diagnostics",
				IsToggle: true,
				Enabled:  appConfig.Diagnostics,
			},
			{
				ID:       "ghost_text",
				Label:    "Ghost Autocomplete",
				IsToggle: true,
				Enabled:  appConfig.GhostText,
			},
			{
				ID:       "paste_guard",
				Label:    "Multiline Paste Guard",
				IsToggle: true,
				Enabled:  appConfig.ConfirmMultilinePaste,
			},
			{
				ID:       "sanitize_paste",
				Label:    "Sanitize Pasted Text",
				IsToggle: true,
				Enabled:  appConfig.SanitizePaste,
			},
			{
				ID:       "search",
				Label:    "Find in Buffer",
				IsToggle: true,
				Enabled:  isSearchOpen,
			},
			{
				ID:       "theme",
				Label:    "Color Theme",
				Value:    themeDisplay,
			},
			{
				ID:       "font_size",
				Label:    "Font Size",
				Value:    fmt.Sprintf("%.0fpt", appConfig.FontSize),
			},
			{
				ID:       "opacity",
				Label:    "Window Opacity",
				Value:    fmt.Sprintf("%d%%", int(appConfig.Opacity*100)),
			},
		}
	}

	executeSettingsAction := func(item render.PrefOption, isSecondaryAction bool) {
		switch item.ID {
		case "diagnostics":
			appConfig.Diagnostics = !appConfig.Diagnostics
			_ = config.Save(appConfig)
			if !appConfig.Diagnostics {
				activeDiag = nil
			} else {
				updateDiagnostics()
			}
		case "ghost_text":
			appConfig.GhostText = !appConfig.GhostText
			_ = config.Save(appConfig)
			if !appConfig.GhostText {
				activeGhostText = ""
			} else {
				updateGhostText()
			}
		case "paste_guard":
			appConfig.ConfirmMultilinePaste = !appConfig.ConfirmMultilinePaste
			_ = config.Save(appConfig)
		case "sanitize_paste":
			appConfig.SanitizePaste = !appConfig.SanitizePaste
			_ = config.Save(appConfig)
		case "search":
			isSearchOpen = !isSearchOpen
			if isSearchOpen {
				updateSearchMatches()
			}
		case "theme":
			themesList := []string{"tokyo-night", "catppuccin-mocha", "minecraft", "tokyo-day", "solarized-light", "auto"}
			curIdx := 0
			for i, thID := range themesList {
				if thID == appConfig.Theme {
					curIdx = i
					break
				}
			}
			nextIdx := (curIdx + 1) % len(themesList)
			if isSecondaryAction {
				nextIdx = (curIdx - 1 + len(themesList)) % len(themesList)
			}
			appConfig.Theme = themesList[nextIdx]
			savedThemeID = appConfig.Theme
			_ = config.Save(appConfig)
			activeTheme = terminal.ResolveTheme(savedThemeID, systemIsDark)
			for _, t := range tabs {
				t.Term.SetTheme(activeTheme)
			}
		case "font_size":
			if isSecondaryAction {
				applyZoom(platform.ActionZoomOut)
			} else {
				applyZoom(platform.ActionZoomIn)
			}
		case "opacity":
			opacities := []float64{1.0, 0.95, 0.90, 0.85, 0.80, 0.75}
			curIdx := 0
			for i, op := range opacities {
				diff := op - appConfig.Opacity
				if diff < 0 {
					diff = -diff
				}
				if diff < 0.02 {
					curIdx = i
					break
				}
			}
			nextIdx := (curIdx + 1) % len(opacities)
			if isSecondaryAction {
				nextIdx = (curIdx - 1 + len(opacities)) % len(opacities)
			}
			appConfig.Opacity = opacities[nextIdx]
			_ = config.Save(appConfig)
			win.SetOpacity(appConfig.Opacity)
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
			canvas.RenderPreferencesModal(activeTerm.Theme(), prefIndex, getSettingsItems(), savedThemeID)
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

	sendMouseEvent := func(btn int, press bool, pixelX, pixelY int, state uint16) bool {
		if activeTabIdx < 0 || activeTabIdx >= len(tabs) {
			return false
		}
		curTerm := tabs[activeTabIdx].Term
		curPTY := tabs[activeTabIdx].PTY
		if !curTerm.MouseTrackingLocked() {
			return false
		}
		cx, cy := toCellCoords(pixelX, pixelY)
		col := cx + 1
		row := cy + 1

		cb := btn
		if (state & platform.ModShift) != 0 {
			cb |= 4
		}
		if (state & platform.ModAlt) != 0 {
			cb |= 8
		}
		if (state & platform.ModCtrl) != 0 {
			cb |= 16
		}

		if curTerm.MouseSGRLocked() {
			terminator := 'M'
			if !press {
				terminator = 'm'
			}
			seq := fmt.Sprintf("\x1b[<%d;%d;%d%c", cb, col, row, terminator)
			_, _ = curPTY.Write([]byte(seq))
			return true
		} else if press {
			if col > 223 {
				col = 223
			}
			if row > 223 {
				row = 223
			}
			b := []byte{0x1b, '[', 'M', byte(cb + 32), byte(col + 32), byte(row + 32)}
			_, _ = curPTY.Write(b)
			return true
		}
		return false
	}

	var (
		lastClickTime time.Time
		lastClickX    int
		lastClickY    int
		clickCount    int

		lastHeaderClickTime time.Time
		lastHeaderClickX    int
		lastHeaderClickY    int
	)

	autoScrollTicker := time.NewTicker(40 * time.Millisecond)
	defer autoScrollTicker.Stop()

	var pendingEvent platform.Event

	for {
		var ev platform.Event
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
		case platform.MotionNotifyEvent:
			lastMotion := e
		drainMotion:
			for {
				select {
				case nextEv, ok := <-xEventCh:
					if !ok {
						return
					}
					if m, isMotion := nextEv.(platform.MotionNotifyEvent); isMotion {
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

			if (e.State & platform.ButtonMask1) == 0 {
				if isSelecting {
					isSelecting = false
					dragScrollDelta = 0
				}
				if isDraggingScrollbar {
					isDraggingScrollbar = false
				}
			}

			if isDraggingScrollbar {
				updateScrollbar(int(e.EventY))
				continue
			}

			if (e.State&platform.ButtonMask1) != 0 && isSelecting {
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

		case platform.ButtonPressEvent:
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
					openURL(hoveredURL.URL)
					continue
				}

				if isPrefOpen {
					items := getSettingsItems()
					modalX := canvas.PrefModalBox[0]
					modalY := canvas.PrefModalBox[1]
					modalW := canvas.PrefModalBox[2]
					modalH := canvas.PrefModalBox[3]
					rowH := canvas.PrefModalBox[4]
					headerH := 36
					if activeTheme.ID == "minecraft" {
						headerH = 40
					}

					px := int(e.EventX)
					py := int(e.EventY)

					if px >= modalX && px < modalX+modalW && py >= modalY && py < modalY+modalH {
						optStartY := modalY + headerH + 4
						if py >= optStartY && py < optStartY+len(items)*rowH {
							clickedIdx := (py - optStartY) / rowH
							if clickedIdx >= 0 && clickedIdx < len(items) {
								prefIndex = clickedIdx
								executeSettingsAction(items[clickedIdx], false)
							}
						}
					} else {
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
					if int(e.EventX) >= int(currentWidth)-36 {
						for _, t := range tabs {
							t.Close()
						}
						os.Exit(0)
					} else if int(e.EventX) >= int(currentWidth)-72 && int(e.EventX) < int(currentWidth)-36 {
						win.ToggleMaximize()
						continue
					} else if int(e.EventX) >= int(currentWidth)-108 && int(e.EventX) < int(currentWidth)-72 {
						win.Minimize()
						continue
					} else if int(e.EventX) >= int(currentWidth)-144 && int(e.EventX) < int(currentWidth)-108 {
						isPrefOpen = !isPrefOpen
						prefIndex = 0
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

					isTabCloseBtn := false
					for _, thb := range canvas.TabHitBoxes {
						if int(e.EventX) >= thb.CloseX && int(e.EventX) <= thb.CloseEndX {
							isTabCloseBtn = true
							break
						}
					}

					isControlBtn := int(e.EventX) >= int(currentWidth)-144

					now := time.Now()
					dx := int(e.EventX) - lastHeaderClickX
					dy := int(e.EventY) - lastHeaderClickY
					if dx < 0 {
						dx = -dx
					}
					if dy < 0 {
						dy = -dy
					}

					if !isTabCloseBtn && !isControlBtn && now.Sub(lastHeaderClickTime) < 350*time.Millisecond && dx < 10 && dy < 10 {
						lastHeaderClickTime = time.Time{}
						win.ToggleMaximize()
						continue
					}
					if !isTabCloseBtn && !isControlBtn {
						lastHeaderClickTime = now
						lastHeaderClickX = int(e.EventX)
						lastHeaderClickY = int(e.EventY)
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
					if activeTerm.ScrollbackLen() > 0 {
						isDraggingScrollbar = true
						updateScrollbar(int(e.EventY))
						continue
					}
				}

				if activeTerm.MouseTrackingLocked() && (e.State&platform.ModShift) == 0 {
					if sendMouseEvent(0, true, int(e.EventX), int(e.EventY), e.State) {
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
				if activeTerm.MouseTrackingLocked() && (e.State&platform.ModShift) == 0 {
					if sendMouseEvent(1, true, int(e.EventX), int(e.EventY), e.State) {
						continue
					}
				}
				win.PastePrimary()
			} else if e.Detail == 3 {
				if activeTerm.MouseTrackingLocked() && (e.State&platform.ModShift) == 0 {
					if sendMouseEvent(2, true, int(e.EventX), int(e.EventY), e.State) {
						continue
					}
				}
			} else if e.Detail == 4 {
				if isPrefOpen {
					items := getSettingsItems()
					if prefIndex > 0 {
						prefIndex--
					} else {
						prefIndex = len(items) - 1
					}
					triggerRedraw()
					continue
				}
				if (e.State & platform.ModCtrl) != 0 {
					applyZoom(platform.ActionZoomIn)
					continue
				}
				if activeTerm.MouseTrackingLocked() && (e.State&platform.ModShift) == 0 {
					if sendMouseEvent(64, true, int(e.EventX), int(e.EventY), e.State) {
						continue
					}
				}
				if activeTerm.IsAlt() {
					_, _ = activePTY.Write([]byte("\x1b[A\x1b[A\x1b[A"))
				} else {
					activeTerm.Scroll(3)
					triggerRedraw()
				}
			} else if e.Detail == 5 {
				if isPrefOpen {
					items := getSettingsItems()
					if prefIndex < len(items)-1 {
						prefIndex++
					} else {
						prefIndex = 0
					}
					triggerRedraw()
					continue
				}
				if (e.State & platform.ModCtrl) != 0 {
					applyZoom(platform.ActionZoomOut)
					continue
				}
				if activeTerm.MouseTrackingLocked() && (e.State&platform.ModShift) == 0 {
					if sendMouseEvent(65, true, int(e.EventX), int(e.EventY), e.State) {
						continue
					}
				}
				if activeTerm.IsAlt() {
					_, _ = activePTY.Write([]byte("\x1b[B\x1b[B\x1b[B"))
				} else {
					activeTerm.Scroll(-3)
					triggerRedraw()
				}
			}

		case platform.ButtonReleaseEvent:
			if activeTerm.MouseTrackingLocked() && (e.State&platform.ModShift) == 0 {
				btn := -1
				if e.Detail == 1 {
					btn = 0
				} else if e.Detail == 2 {
					btn = 1
				} else if e.Detail == 3 {
					btn = 2
				}
				if btn >= 0 && sendMouseEvent(btn, false, int(e.EventX), int(e.EventY), e.State) {
					continue
				}
			}
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

		case platform.KeyPressEvent:
			keysym := e.KeySym
			data := string(e.Bytes)
			action := e.Action

			if activeTerm.AppCursorLocked() && (e.State&(platform.ModCtrl|platform.ModAlt|platform.ModShift)) == 0 {
				switch keysym {
				case 0xff52:
					data = "\x1bOA"
				case 0xff54:
					data = "\x1bOB"
				case 0xff53:
					data = "\x1bOC"
				case 0xff51:
					data = "\x1bOD"
				case 0xff50:
					data = "\x1bOH"
				case 0xff57:
					data = "\x1bOF"
				}
			}

			if action == platform.ActionPreferences {
				isPrefOpen = !isPrefOpen
				if isPrefOpen {
					prefIndex = 0
				}
				triggerRedraw()
				continue
			}

			if isPasteModalOpen {
				switch keysym {
				case 0xff1b, 'c', 'C', 'n', 'N':
					isPasteModalOpen = false
					pendingPasteText = ""
					pendingPasteWarnings = nil
					triggerRedraw()
				case 0xff0d, 'p', 'P', 'y', 'Y':
					textToPaste := pendingPasteText
					isPasteModalOpen = false
					pendingPasteText = ""
					pendingPasteWarnings = nil
					doWritePastedText(textToPaste)
				case 's', 'S':
					textToPaste := paste.FlattenToSingleLine(pendingPasteText)
					isPasteModalOpen = false
					pendingPasteText = ""
					pendingPasteWarnings = nil
					doWritePastedText(textToPaste)
				case 'q', 'Q':
					textToPaste := paste.QuoteURL(pendingPasteText)
					isPasteModalOpen = false
					pendingPasteText = ""
					pendingPasteWarnings = nil
					doWritePastedText(textToPaste)
				}
				continue
			}

			if isPrefOpen {
				items := getSettingsItems()
				switch keysym {
				case 0xff52, 'k', 'K':
					if prefIndex > 0 {
						prefIndex--
					} else {
						prefIndex = len(items) - 1
					}
					triggerRedraw()
				case 0xff54, 'j', 'J':
					if prefIndex < len(items)-1 {
						prefIndex++
					} else {
						prefIndex = 0
					}
					triggerRedraw()
				case 0xff51, 'h', 'H', '-':
					if prefIndex >= 0 && prefIndex < len(items) {
						executeSettingsAction(items[prefIndex], true)
					}
				case 0xff53, 'l', 'L', '+', '=':
					if prefIndex >= 0 && prefIndex < len(items) {
						executeSettingsAction(items[prefIndex], false)
					}
				case 0xff0d, ' ':
					if prefIndex >= 0 && prefIndex < len(items) {
						executeSettingsAction(items[prefIndex], false)
					}
				case 0xff1b, 'q', 'Q':
					isPrefOpen = false
					triggerRedraw()
				case '1', '2', '3', '4', '5', '6', '7', '8':
					idx := int(keysym - '1')
					if idx >= 0 && idx < len(items) {
						prefIndex = idx
						executeSettingsAction(items[prefIndex], false)
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

				if action == platform.ActionToggleDiagnostics {
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
						text := activeTerm.GetSelectedText()
						if text != "" {
							win.SetClipboard(text)
						}
						activeTerm.ClearSelection()
						triggerRedraw()
					} else {
						text := activeTerm.GetAllText()
						if text != "" {
							win.SetClipboard(text)
						}
					}
				} else if action == platform.ActionPaste {
					win.Paste()
				} else if action == platform.ActionSelectAll {
					activeTerm.SelectAll()
					if txt := activeTerm.GetSelectedText(); txt != "" {
						win.SetPrimary(txt)
					}
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
						win.Paste()
					} else {
						if activeGhostText != "" && !activeTerm.IsAlt() {
							isAcceptKey := (keysym == 0xff53) || (keysym == 0xff09 && (e.State&platform.ModShift) == 0)
							if isAcceptKey {
								if len(currentInputBuffer) > 0 && currentInputBuffer[len(currentInputBuffer)-1] == ' ' && !strings.HasSuffix(currentInputBuffer, "\\ ") {
									_, _ = activePTY.Write([]byte{0x08, '\\', ' '})
									currentInputBuffer = currentInputBuffer[:len(currentInputBuffer)-1] + "\\ "
								}
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

						if !activeTerm.IsAlt() {
							if !activePTY.IsForegroundShell() || isPasswordPrompt(activeTerm) {
								currentInputBuffer = ""
								activeGhostText = ""
								activeDiag = nil
							} else {
								isAlt := (e.State & platform.ModAlt) != 0
								if isAlt && (keysym == 0xff0d || keysym == 0xff8d) && activeDiag != nil {
									diag := diagnostics.Analyze(strings.TrimSpace(currentInputBuffer))
									if diag != nil && diag.QuickFix != "" {
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

								if keysym == 0xff0d || keysym == 0xff8d {
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
								} else if keysym == 0xff08 {
									if len(currentInputBuffer) > 0 {
										currentInputBuffer = currentInputBuffer[:len(currentInputBuffer)-1]
										updateGhostText()
										updateDiagnostics()
									}
								} else if keysym == 0xff1b || (len(data) == 1 && (data[0] == 0x03 || data[0] == 0x15)) {
									currentInputBuffer = ""
									activeGhostText = ""
									activeDiag = nil
								} else if len(data) > 0 && data[0] >= 32 && data[0] != 127 {
									currentInputBuffer += data
									updateGhostText()
									updateDiagnostics()
								}
							}
						}

						activeTerm.ClearSelection()
						_, _ = activePTY.Write([]byte(data))
						activeTerm.ResetScroll()
						triggerRedraw()
					}
				}

		case platform.PasteNotifyEvent:
			if len(e.Text) > 0 {
				handlePastedData([]byte(e.Text))
			}

		case platform.ConfigureNotifyEvent:
			lastCfg := e
		drainCfg:
			for {
				select {
				case nextEv, ok := <-xEventCh:
					if !ok {
						return
					}
					if cfg, isCfg := nextEv.(platform.ConfigureNotifyEvent); isCfg {
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
					}
				}

				if isSearchOpen {
					updateSearchMatches()
				}

				renderScreen()
			}

		case platform.ExposeEvent:
			renderScreen()

		case platform.MappingNotifyEvent:

		case platform.FocusOutEvent:
			isSelecting = false
			if hoveredURL != nil {
				hoveredURL = nil
				triggerRedraw()
			}

		case platform.CloseRequestEvent:
			return
		}

		runtime.Gosched()
	}
}

func openURL(urlStr string) {
	if runtime.GOOS == "windows" {
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", urlStr).Start()
	} else if runtime.GOOS == "darwin" {
		_ = exec.Command("open", urlStr).Start()
	} else {
		_ = exec.Command("xdg-open", urlStr).Start()
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
