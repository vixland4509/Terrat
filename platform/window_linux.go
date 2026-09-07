//go:build !windows

package platform

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/bigreq"
	"github.com/jezek/xgb/xproto"
)

type Window struct {
	X      *xgb.Conn
	Win    xproto.Window
	GC     xproto.Gcontext
	Screen *xproto.ScreenInfo
	Depth  byte

	Width  uint16
	Height uint16

	AtomWmProtocols     xproto.Atom
	AtomWmDeleteWindow  xproto.Atom
	AtomNetWmName       xproto.Atom
	AtomUtf8String      xproto.Atom
	AtomNetWmMoveresize xproto.Atom
	AtomNetWmState      xproto.Atom
	AtomNetWmMaxV       xproto.Atom
	AtomNetWmMaxH       xproto.Atom
	AtomChangeState     xproto.Atom
	AtomClipboard       xproto.Atom
	AtomTargets         xproto.Atom
	AtomTerratPaste     xproto.Atom
	AtomNetWmOpacity    xproto.Atom

	clipMu        sync.RWMutex
	clipboardText string
	primaryText   string

	pixmap        xproto.Pixmap
	pixmapW       uint16
	pixmapH       uint16
	cursors       map[CursorType]xproto.Cursor
	currentCursor CursorType

	keyHandler *KeyHandler
	eventCh    chan Event
}

func NewWindow(title string, width, height uint16, iconPath string, iconData []byte) (*Window, error) {
	X, err := xgb.NewConn()
	if err != nil {
		displayEnv := os.Getenv("DISPLAY")
		waylandEnv := os.Getenv("WAYLAND_DISPLAY")
		if waylandEnv != "" && displayEnv == "" {
			return nil, fmt.Errorf("cannot connect to display server: running Wayland (%s) but DISPLAY is not set (Xwayland is required). Please ensure xorg-xwayland is installed: %w", waylandEnv, err)
		}
		return nil, fmt.Errorf("cannot connect to X11 display (DISPLAY=%q): %w", displayEnv, err)
	}

	if err := bigreq.Init(X); err == nil {
		_, _ = bigreq.Enable(X).Reply()
	}

	setup := xproto.Setup(X)
	screen := setup.DefaultScreen(X)

	winId, err := xproto.NewWindowId(X)
	if err != nil {
		X.Close()
		return nil, fmt.Errorf("failed to allocate window id: %w", err)
	}

	eventMask := uint32(
		xproto.EventMaskExposure |
			xproto.EventMaskKeyPress |
			xproto.EventMaskStructureNotify |
			xproto.EventMaskFocusChange |
			xproto.EventMaskButtonPress |
			xproto.EventMaskButtonRelease |
			xproto.EventMaskPointerMotion,
	)

	createMask := uint32(
		xproto.CwBackPixmap |
			xproto.CwBitGravity |
			xproto.CwWinGravity |
			xproto.CwEventMask,
	)
	createVals := []uint32{
		xproto.BackPixmapNone,
		xproto.GravityNorthWest,
		xproto.GravityNorthWest,
		eventMask,
	}

	err = xproto.CreateWindowChecked(
		X,
		screen.RootDepth,
		winId,
		screen.Root,
		80, 80,
		width, height,
		0,
		xproto.WindowClassInputOutput,
		screen.RootVisual,
		createMask,
		createVals,
	).Check()
	if err != nil {
		X.Close()
		return nil, fmt.Errorf("failed to create X11 window: %w", err)
	}

	gcId, err := xproto.NewGcontextId(X)
	if err != nil {
		X.Close()
		return nil, fmt.Errorf("failed to allocate GC id: %w", err)
	}

	err = xproto.CreateGCChecked(X, gcId, xproto.Drawable(winId), xproto.GcGraphicsExposures, []uint32{0}).Check()
	if err != nil {
		X.Close()
		return nil, fmt.Errorf("failed to create GC: %w", err)
	}

	w := &Window{
		X:             X,
		Win:           winId,
		GC:            gcId,
		Screen:        screen,
		Depth:         screen.RootDepth,
		Width:         width,
		Height:        height,
		cursors:       make(map[CursorType]xproto.Cursor),
		currentCursor: CursorDefault,
	}

	w.AtomWmProtocols, _ = internAtom(X, "WM_PROTOCOLS")
	w.AtomWmDeleteWindow, _ = internAtom(X, "WM_DELETE_WINDOW")
	w.AtomNetWmName, _ = internAtom(X, "_NET_WM_NAME")
	w.AtomUtf8String, _ = internAtom(X, "UTF8_STRING")
	w.AtomNetWmMoveresize, _ = internAtom(X, "_NET_WM_MOVERESIZE")
	w.AtomNetWmState, _ = internAtom(X, "_NET_WM_STATE")
	w.AtomNetWmMaxV, _ = internAtom(X, "_NET_WM_STATE_MAXIMIZED_VERT")
	w.AtomNetWmMaxH, _ = internAtom(X, "_NET_WM_STATE_MAXIMIZED_HORZ")
	w.AtomChangeState, _ = internAtom(X, "WM_CHANGE_STATE")
	w.AtomClipboard, _ = internAtom(X, "CLIPBOARD")
	w.AtomTargets, _ = internAtom(X, "TARGETS")
	w.AtomTerratPaste, _ = internAtom(X, "TERRAT_SELECTION")
	w.AtomNetWmOpacity, _ = internAtom(X, "_NET_WM_WINDOW_OPACITY")

	motifAtom, err := internAtom(X, "_MOTIF_WM_HINTS")
	if err == nil && motifAtom != 0 {
		hints := []uint32{2, 0, 0, 0, 0}
		data := make([]byte, 20)
		for i, val := range hints {
			binary.LittleEndian.PutUint32(data[i*4:], val)
		}
		_ = xproto.ChangeProperty(
			X,
			xproto.PropModeReplace,
			winId,
			motifAtom,
			motifAtom,
			32,
			5,
			data,
		)
	}

	if w.AtomWmProtocols != 0 && w.AtomWmDeleteWindow != 0 {
		atomBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(atomBytes, uint32(w.AtomWmDeleteWindow))
		_ = xproto.ChangePropertyChecked(
			X,
			xproto.PropModeReplace,
			winId,
			w.AtomWmProtocols,
			xproto.AtomAtom,
			32,
			1,
			atomBytes,
		).Check()
	}

	wmClassAtom, err := internAtom(X, "WM_CLASS")
	if err == nil && wmClassAtom != 0 {
		stringAtom, _ := internAtom(X, "STRING")
		if stringAtom == 0 {
			stringAtom = xproto.AtomString
		}
		classData := []byte("terrat\x00TerraTerminal\x00")
		_ = xproto.ChangePropertyChecked(
			X,
			xproto.PropModeReplace,
			winId,
			wmClassAtom,
			stringAtom,
			8,
			uint32(len(classData)),
			classData,
		).Check()
	}

	pidAtom, err := internAtom(X, "_NET_WM_PID")
	cardinalAtom, _ := internAtom(X, "CARDINAL")
	if err == nil && pidAtom != 0 && cardinalAtom != 0 {
		pidBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(pidBytes, uint32(os.Getpid()))
		_ = xproto.ChangePropertyChecked(
			X,
			xproto.PropModeReplace,
			winId,
			pidAtom,
			cardinalAtom,
			32,
			1,
			pidBytes,
		).Check()
	}

	winTypeAtom, err := internAtom(X, "_NET_WM_WINDOW_TYPE")
	normalTypeAtom, err2 := internAtom(X, "_NET_WM_WINDOW_TYPE_NORMAL")
	atomAtom, _ := internAtom(X, "ATOM")
	if err == nil && err2 == nil && winTypeAtom != 0 && normalTypeAtom != 0 {
		typeBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(typeBytes, uint32(normalTypeAtom))
		_ = xproto.ChangePropertyChecked(
			X,
			xproto.PropModeReplace,
			winId,
			winTypeAtom,
			atomAtom,
			32,
			1,
			typeBytes,
		).Check()
	}

	cursorFontId, err := xproto.NewFontId(X)
	if err == nil {
		if err := xproto.OpenFontChecked(X, cursorFontId, uint16(len("cursor")), "cursor").Check(); err == nil {
			glyphMap := map[CursorType]uint16{
				CursorDefault:           68,
				CursorText:              152,
				CursorPointer:           58,
				CursorResizeTopLeft:     134,
				CursorResizeTop:         138,
				CursorResizeTopRight:    136,
				CursorResizeRight:       96,
				CursorResizeBottomRight: 14,
				CursorResizeBottom:      16,
				CursorResizeBottomLeft:  12,
				CursorResizeLeft:        70,
			}
			for ct, glyph := range glyphMap {
				if cid, err := xproto.NewCursorId(X); err == nil {
					if err := xproto.CreateGlyphCursorChecked(X, cid, cursorFontId, cursorFontId,
						glyph, glyph+1, 0, 0, 0, 65535, 65535, 65535).Check(); err == nil {
						w.cursors[ct] = cid
					}
				}
			}
		}
	}

	w.SetTitle(title)

	if len(iconData) > 0 {
		_ = SetWindowIconBytes(X, winId, iconData)
	} else if iconPath != "" {
		_ = SetWindowIcon(X, winId, iconPath)
	}

	w.SetMinSize(320, 180)

	_ = xproto.MapWindowChecked(X, winId).Check()
	X.Sync()

	w.keyHandler, _ = NewKeyHandler(X)
	w.eventCh = make(chan Event, 1024)
	go w.eventLoop()

	return w, nil
}

func (w *Window) SetTitle(title string) {
	if w.AtomNetWmName != 0 && w.AtomUtf8String != 0 {
		_ = xproto.ChangeProperty(
			w.X,
			xproto.PropModeReplace,
			w.Win,
			w.AtomNetWmName,
			w.AtomUtf8String,
			8,
			uint32(len(title)),
			[]byte(title),
		)
	}
	_ = xproto.ChangeProperty(
		w.X,
		xproto.PropModeReplace,
		w.Win,
		xproto.AtomWmName,
		xproto.AtomString,
		8,
		uint32(len(title)),
		[]byte(title),
	)
}

func (w *Window) SetCursorType(ct CursorType) {
	if ct == w.currentCursor {
		return
	}
	cid, ok := w.cursors[ct]
	if !ok {
		return
	}
	w.currentCursor = ct
	_ = xproto.ChangeWindowAttributes(w.X, w.Win, xproto.CwCursor, []uint32{uint32(cid)})
}

func (w *Window) SetOpacity(opacity float64) {
	if w.AtomNetWmOpacity == 0 {
		return
	}
	if opacity < 0.1 {
		opacity = 0.1
	}
	if opacity > 1.0 {
		opacity = 1.0
	}
	val := uint32(opacity * 4294967295.0)
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, val)

	_ = xproto.ChangePropertyChecked(
		w.X,
		xproto.PropModeReplace,
		w.Win,
		w.AtomNetWmOpacity,
		xproto.AtomCardinal,
		32,
		1,
		buf,
	).Check()
}



func (w *Window) StartResize(direction int, rootX, rootY int16) {
	if w.AtomNetWmMoveresize == 0 || direction < 0 || direction > 8 {
		return
	}
	cm := xproto.ClientMessageEvent{
		Format: 32,
		Window: w.Win,
		Type:   w.AtomNetWmMoveresize,
		Data: xproto.ClientMessageDataUnionData32New([]uint32{
			uint32(rootX),
			uint32(rootY),
			uint32(direction),
			1,
			1,
		}),
	}
	_ = xproto.SendEvent(
		w.X,
		false,
		w.Screen.Root,
		xproto.EventMaskSubstructureNotify|xproto.EventMaskSubstructureRedirect,
		string(cm.Bytes()),
	)
	w.X.Sync()
}

func (w *Window) StartDrag(rootX, rootY int16) {
	w.StartResize(ResizeMove, rootX, rootY)
}

func (w *Window) Minimize() {
	if w.AtomChangeState == 0 {
		return
	}
	cm := xproto.ClientMessageEvent{
		Format: 32,
		Window: w.Win,
		Type:   w.AtomChangeState,
		Data: xproto.ClientMessageDataUnionData32New([]uint32{
			3,
			0, 0, 0, 0,
		}),
	}
	_ = xproto.SendEvent(
		w.X,
		false,
		w.Screen.Root,
		xproto.EventMaskSubstructureNotify|xproto.EventMaskSubstructureRedirect,
		string(cm.Bytes()),
	)
	w.X.Sync()
}

func (w *Window) ToggleMaximize() {
	if w.AtomNetWmState == 0 || w.AtomNetWmMaxV == 0 || w.AtomNetWmMaxH == 0 {
		return
	}
	cm := xproto.ClientMessageEvent{
		Format: 32,
		Window: w.Win,
		Type:   w.AtomNetWmState,
		Data: xproto.ClientMessageDataUnionData32New([]uint32{
			2,
			uint32(w.AtomNetWmMaxV),
			uint32(w.AtomNetWmMaxH),
			1,
			0,
		}),
	}
	_ = xproto.SendEvent(
		w.X,
		false,
		w.Screen.Root,
		xproto.EventMaskSubstructureNotify|xproto.EventMaskSubstructureRedirect,
		string(cm.Bytes()),
	)
	w.X.Sync()
}

func (w *Window) SetMinSize(minW, minH uint32) {
	atomNormalHints, _ := internAtom(w.X, "WM_NORMAL_HINTS")
	atomSizeHints, _ := internAtom(w.X, "WM_SIZE_HINTS")
	if atomNormalHints == 0 || atomSizeHints == 0 {
		return
	}
	hints := make([]uint32, 18)
	hints[0] = 0x10
	hints[5] = minW
	hints[6] = minH

	data := make([]byte, 18*4)
	for i, v := range hints {
		binary.LittleEndian.PutUint32(data[i*4:], v)
	}
	_ = xproto.ChangeProperty(
		w.X,
		xproto.PropModeReplace,
		w.Win,
		atomNormalHints,
		atomSizeHints,
		32,
		18,
		data,
	)
}

func (w *Window) Blit(pixels []byte, width, height uint16) {
	if len(pixels) == 0 || width == 0 || height == 0 {
		return
	}

	if w.pixmap == 0 || w.pixmapW != width || w.pixmapH != height {
		if w.pixmap != 0 {
			_ = xproto.FreePixmap(w.X, w.pixmap)
			w.pixmap = 0
		}
		pid, err := xproto.NewPixmapId(w.X)
		if err == nil {
			err = xproto.CreatePixmapChecked(w.X, w.Depth, pid, xproto.Drawable(w.Win), width, height).Check()
			if err == nil {
				w.pixmap = pid
				w.pixmapW = width
				w.pixmapH = height
			}
		}
	}

	targetDrawable := xproto.Drawable(w.Win)
	if w.pixmap != 0 {
		targetDrawable = xproto.Drawable(w.pixmap)
	}

	stride := int(width) * 4
	maxRowsPerChunk := 65536 / stride
	if maxRowsPerChunk < 1 {
		maxRowsPerChunk = 1
	}

	h := int(height)
	for y := 0; y < h; y += maxRowsPerChunk {
		chunkH := maxRowsPerChunk
		if y+chunkH > h {
			chunkH = h - y
		}
		start := y * stride
		end := start + (chunkH * stride)
		if end > len(pixels) {
			end = len(pixels)
		}

		_ = xproto.PutImage(
			w.X,
			xproto.ImageFormatZPixmap,
			targetDrawable,
			w.GC,
			width,
			uint16(chunkH),
			0, int16(y),
			0,
			w.Depth,
			pixels[start:end],
		)
	}

	if w.pixmap != 0 {
		_ = xproto.CopyArea(
			w.X,
			xproto.Drawable(w.pixmap),
			xproto.Drawable(w.Win),
			w.GC,
			0, 0,
			0, 0,
			width, height,
		)
	}

	w.X.Sync()
}

func (w *Window) Close() {
	if w.pixmap != 0 {
		_ = xproto.FreePixmap(w.X, w.pixmap)
		w.pixmap = 0
	}
	_ = xproto.FreeGC(w.X, w.GC)
	_ = xproto.DestroyWindow(w.X, w.Win)
	w.X.Close()
}

func (w *Window) SetClipboard(text string) {
	if text == "" {
		return
	}
	w.clipMu.Lock()
	w.clipboardText = text
	w.clipMu.Unlock()
	if w.AtomClipboard != 0 {
		_ = xproto.SetSelectionOwner(w.X, w.Win, w.AtomClipboard, xproto.TimeCurrentTime)
	}
}

func (w *Window) SetPrimary(text string) {
	if text == "" {
		return
	}
	w.clipMu.Lock()
	w.primaryText = text
	w.clipMu.Unlock()
	_ = xproto.SetSelectionOwner(w.X, w.Win, xproto.AtomPrimary, xproto.TimeCurrentTime)
}

func (w *Window) HandleSelectionRequest(e xproto.SelectionRequestEvent) {
	var data []byte
	var text string

	w.clipMu.RLock()
	if e.Selection == xproto.AtomPrimary {
		text = w.primaryText
	} else if e.Selection == w.AtomClipboard {
		text = w.clipboardText
	}
	w.clipMu.RUnlock()

	target := e.Target
	property := e.Property
	if property == 0 {
		property = target
	}

	success := false

	if target == w.AtomTargets && w.AtomTargets != 0 {
		targets := []uint32{
			uint32(w.AtomTargets),
			uint32(w.AtomUtf8String),
			uint32(xproto.AtomString),
		}
		data = make([]byte, len(targets)*4)
		for i, t := range targets {
			binary.LittleEndian.PutUint32(data[i*4:], t)
		}
		_ = xproto.ChangeProperty(
			w.X,
			xproto.PropModeReplace,
			e.Requestor,
			property,
			xproto.AtomAtom,
			32,
			uint32(len(targets)),
			data,
		)
		success = true
	} else if target == w.AtomUtf8String || target == xproto.AtomString {
		data = []byte(text)
		_ = xproto.ChangeProperty(
			w.X,
			xproto.PropModeReplace,
			e.Requestor,
			property,
			target,
			8,
			uint32(len(data)),
			data,
		)
		success = true
	}

	respProperty := property
	if !success {
		respProperty = 0
	}

	resp := xproto.SelectionNotifyEvent{
		Time:      e.Time,
		Requestor: e.Requestor,
		Selection: e.Selection,
		Target:    target,
		Property:  respProperty,
	}

	_ = xproto.SendEvent(
		w.X,
		false,
		e.Requestor,
		0,
		string(resp.Bytes()),
	)
	w.X.Sync()
}

func (w *Window) RequestPaste(selectionAtom xproto.Atom) {
	if selectionAtom == 0 || w.AtomTerratPaste == 0 || w.AtomUtf8String == 0 {
		return
	}
	_ = xproto.ConvertSelection(
		w.X,
		w.Win,
		selectionAtom,
		w.AtomUtf8String,
		w.AtomTerratPaste,
		xproto.TimeCurrentTime,
	)
	w.X.Sync()
}

func (w *Window) HandleSelectionNotify(e xproto.SelectionNotifyEvent) []byte {
	if e.Property == 0 {
		return nil
	}

	reply, err := xproto.GetProperty(
		w.X,
		true,
		w.Win,
		e.Property,
		xproto.AtomAny,
		0,
		1024*1024/4,
	).Reply()

	if err != nil || reply == nil || len(reply.Value) == 0 {
		return nil
	}

	return reply.Value
}

func (w *Window) Events() <-chan Event {
	return w.eventCh
}

func (w *Window) Paste() {
	w.RequestPaste(w.AtomClipboard)
}

func (w *Window) PastePrimary() {
	w.RequestPaste(xproto.AtomPrimary)
}

func (w *Window) eventLoop() {
	for {
		ev, err := w.X.WaitForEvent()
		if ev == nil && err == nil {
			close(w.eventCh)
			return
		}
		if err != nil {
			continue
		}
		switch e := ev.(type) {
		case xproto.MotionNotifyEvent:
			w.eventCh <- MotionNotifyEvent{
				EventX: e.EventX,
				EventY: e.EventY,
				RootX:  e.RootX,
				RootY:  e.RootY,
				State:  e.State,
			}
		case xproto.ButtonPressEvent:
			w.eventCh <- ButtonPressEvent{
				Detail: uint8(e.Detail),
				State:  e.State,
				EventX: e.EventX,
				EventY: e.EventY,
				RootX:  e.RootX,
				RootY:  e.RootY,
			}
		case xproto.ButtonReleaseEvent:
			w.eventCh <- ButtonReleaseEvent{
				Detail: uint8(e.Detail),
				State:  e.State,
				EventX: e.EventX,
				EventY: e.EventY,
				RootX:  e.RootX,
				RootY:  e.RootY,
			}
		case xproto.KeyPressEvent:
			var act ActionType
			var b []byte
			var ksym uint32
			if w.keyHandler != nil {
				ksym = uint32(w.keyHandler.KeySym(e))
				b, act = w.keyHandler.Translate(e)
			}
			w.eventCh <- KeyPressEvent{
				Detail: byte(e.Detail),
				State:  e.State,
				KeySym: ksym,
				Bytes:  b,
				Action: act,
			}
		case xproto.ConfigureNotifyEvent:
			w.eventCh <- ConfigureNotifyEvent{
				Width:  e.Width,
				Height: e.Height,
			}
		case xproto.ExposeEvent:
			w.eventCh <- ExposeEvent{}
		case xproto.FocusOutEvent:
			w.eventCh <- FocusOutEvent{}
		case xproto.MappingNotifyEvent:
			if w.keyHandler != nil {
				_ = w.keyHandler.RefreshMapping(w.X)
			}
			w.eventCh <- MappingNotifyEvent{}
		case xproto.SelectionRequestEvent:
			w.HandleSelectionRequest(e)
		case xproto.SelectionNotifyEvent:
			val := w.HandleSelectionNotify(e)
			if len(val) > 0 {
				w.eventCh <- PasteNotifyEvent{Text: string(val)}
			}
		case xproto.SelectionClearEvent:
			// selection clear - no-op
		case xproto.ClientMessageEvent:
			if e.Format == 32 && len(e.Data.Data32) > 0 && xproto.Atom(e.Data.Data32[0]) == w.AtomWmDeleteWindow {
				w.eventCh <- CloseRequestEvent{}
			}
		}
	}
}

