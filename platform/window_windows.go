//go:build windows

package platform

import (
	"fmt"
	"runtime"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modUser32 = windows.NewLazySystemDLL("user32.dll")
	modGdi32  = windows.NewLazySystemDLL("gdi32.dll")
	modKernel = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterClassExW        = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW         = modUser32.NewProc("CreateWindowExW")
	procDefWindowProcW          = modUser32.NewProc("DefWindowProcW")
	procDestroyWindow           = modUser32.NewProc("DestroyWindow")
	procShowWindow              = modUser32.NewProc("ShowWindow")
	procUpdateWindow            = modUser32.NewProc("UpdateWindow")
	procSetWindowTextW          = modUser32.NewProc("SetWindowTextW")
	procGetMessageW             = modUser32.NewProc("GetMessageW")
	procTranslateMessage        = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW        = modUser32.NewProc("DispatchMessageW")
	procPostQuitMessage         = modUser32.NewProc("PostQuitMessage")
	procPostMessageW            = modUser32.NewProc("PostMessageW")
	procGetCursorPos            = modUser32.NewProc("GetCursorPos")
	procLoadCursorW             = modUser32.NewProc("LoadCursorW")
	procSetCursor               = modUser32.NewProc("SetCursor")
	procGetDC                   = modUser32.NewProc("GetDC")
	procReleaseDC               = modUser32.NewProc("ReleaseDC")
	procBeginPaint              = modUser32.NewProc("BeginPaint")
	procEndPaint                = modUser32.NewProc("EndPaint")
	procInvalidateRect          = modUser32.NewProc("InvalidateRect")
	procSetLayeredWindowAttrs   = modUser32.NewProc("SetLayeredWindowAttributes")
	procSetWindowLongW          = modUser32.NewProc("SetWindowLongW")
	procGetWindowLongW          = modUser32.NewProc("GetWindowLongW")
	procSendMessageW            = modUser32.NewProc("SendMessageW")
	procReleaseCapture          = modUser32.NewProc("ReleaseCapture")
	procIsZoomed                = modUser32.NewProc("IsZoomed")
	procGetKeyState             = modUser32.NewProc("GetKeyState")
	procOpenClipboard           = modUser32.NewProc("OpenClipboard")
	procCloseClipboard          = modUser32.NewProc("CloseClipboard")
	procEmptyClipboard          = modUser32.NewProc("EmptyClipboard")
	procGetClipboardData        = modUser32.NewProc("GetClipboardData")
	procSetClipboardData        = modUser32.NewProc("SetClipboardData")
	procIsClipboardFormatAvail  = modUser32.NewProc("IsClipboardFormatAvailable")

	procGetModuleHandleW        = modKernel.NewProc("GetModuleHandleW")
	procGlobalAlloc             = modKernel.NewProc("GlobalAlloc")
	procGlobalLock              = modKernel.NewProc("GlobalLock")
	procGlobalUnlock            = modKernel.NewProc("GlobalUnlock")
	procRtlMoveMemory           = modKernel.NewProc("RtlMoveMemory")

	procStretchDIBits           = modGdi32.NewProc("StretchDIBits")
)

const (
	WS_OVERLAPPED       = 0x00000000
	WS_POPUP            = 0x80000000
	WS_CAPTION          = 0x00C00000
	WS_SYSMENU          = 0x00080000
	WS_THICKFRAME       = 0x00040000
	WS_MINIMIZEBOX      = 0x00020000
	WS_MAXIMIZEBOX      = 0x00010000
	WS_VISIBLE          = 0x10000000

	WS_EX_LAYERED       = 0x00080000
	WS_EX_APPWINDOW     = 0x00040000

	LWA_ALPHA           = 0x00000002

	SW_SHOW             = 5
	SW_MINIMIZE         = 6
	SW_RESTORE          = 9
	SW_MAXIMIZE         = 3

	WM_DESTROY          = 0x0002
	WM_SIZE             = 0x0005
	WM_KILLFOCUS        = 0x0008
	WM_PAINT            = 0x000F
	WM_CLOSE            = 0x0010
	WM_KEYDOWN          = 0x0100
	WM_KEYUP            = 0x0101
	WM_CHAR             = 0x0102
	WM_SYSKEYDOWN       = 0x0104
	WM_MOUSEMOVE        = 0x0200
	WM_LBUTTONDOWN      = 0x0201
	WM_LBUTTONUP        = 0x0202
	WM_RBUTTONDOWN      = 0x0204
	WM_RBUTTONUP        = 0x0205
	WM_MBUTTONDOWN      = 0x0207
	WM_MBUTTONUP        = 0x0208
	WM_MOUSEWHEEL       = 0x020A
	WM_NCLBUTTONDOWN    = 0x00A1

	HTCAPTION           = 2
	HTLEFT              = 10
	HTRIGHT             = 11
	HTTOP               = 12
	HTTOPLEFT           = 13
	HTTOPRIGHT          = 14
	HTBOTTOM            = 15
	HTBOTTOMLEFT        = 16
	HTBOTTOMRIGHT       = 17

	MK_LBUTTON          = 0x0001
	MK_RBUTTON          = 0x0002
	MK_SHIFT            = 0x0004
	MK_CONTROL          = 0x0008
	MK_MBUTTON          = 0x0010

	CF_UNICODETEXT      = 13
	GMEM_MOVEABLE       = 0x0002

	BI_RGB              = 0
	DIB_RGB_COLORS      = 0
	SRCCOPY             = 0x00CC0020
)

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type PAINTSTRUCT struct {
	Hdc         windows.Handle
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type POINT struct {
	X int32
	Y int32
}

type MSG struct {
	Hwnd     windows.Handle
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       POINT
	LPrivate uint32
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]uint32
}

type Window struct {
	hwnd       windows.Handle
	Width      uint16
	Height     uint16
	eventCh    chan Event
	keyHandler *KeyHandler
	mu         sync.Mutex

	// Backbuffer for WM_PAINT
	backbuffer   []byte
	backbufferW  int
	backbufferH  int

	// Cached cursors
	cursors       map[CursorType]windows.Handle
	currentCursor CursorType
}

var (
	activeWindows = make(map[windows.Handle]*Window)
	activeWinMu   sync.RWMutex
	wndClassOnce  sync.Once
	wndClassName  = "TerraTerminalWindow"
)

func registerClass() error {
	h, _, err := procGetModuleHandleW.Call(0)
	if h == 0 {
		return err
	}
	hInstance := windows.Handle(h)

	cursor, _, _ := procLoadCursorW.Call(0, uintptr(32512)) // IDC_ARROW

	classNamePtr, _ := windows.UTF16PtrFromString(wndClassName)

	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:         0x0003, // CS_HREDRAW | CS_VREDRAW
		LpfnWndProc:   windows.NewCallback(wndProc),
		HInstance:     hInstance,
		HCursor:       windows.Handle(cursor),
		LpszClassName: classNamePtr,
	}

	ret, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 {
		return fmt.Errorf("failed to register window class: %w", err)
	}
	return nil
}

type windowCreateResult struct {
	w   *Window
	err error
}

func NewWindow(title string, width, height uint16, iconPath string, iconData []byte) (*Window, error) {
	resCh := make(chan windowCreateResult, 1)
	go windowThread(title, width, height, iconPath, iconData, resCh)
	res := <-resCh
	return res.w, res.err
}

func windowThread(title string, width, height uint16, iconPath string, iconData []byte, resCh chan<- windowCreateResult) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var initErr error
	wndClassOnce.Do(func() {
		initErr = registerClass()
	})
	if initErr != nil {
		resCh <- windowCreateResult{err: initErr}
		return
	}

	h, _, err := procGetModuleHandleW.Call(0)
	if h == 0 {
		resCh <- windowCreateResult{err: err}
		return
	}
	hInstance := windows.Handle(h)

	classNamePtr, _ := windows.UTF16PtrFromString(wndClassName)
	titlePtr, _ := windows.UTF16PtrFromString(title)

	// Modern frameless window style that supports Windows snap layouts & resizing
	style := uint32(WS_POPUP | WS_THICKFRAME | WS_MINIMIZEBOX | WS_MAXIMIZEBOX | WS_VISIBLE)
	exStyle := uint32(WS_EX_APPWINDOW | WS_EX_LAYERED)

	ret, _, err := procCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(classNamePtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(style),
		0x80000000, // CW_USEDEFAULT
		0x80000000,
		uintptr(width),
		uintptr(height),
		0,
		0,
		uintptr(hInstance),
		0,
	)
	if ret == 0 {
		resCh <- windowCreateResult{err: fmt.Errorf("CreateWindowExW failed: %w", err)}
		return
	}

	hwnd := windows.Handle(ret)

	// Load standard cursors
	loadCur := func(id uintptr) windows.Handle {
		h, _, _ := procLoadCursorW.Call(0, id)
		return windows.Handle(h)
	}

	cursors := map[CursorType]windows.Handle{
		CursorDefault:           loadCur(32512), // IDC_ARROW
		CursorText:              loadCur(32513), // IDC_IBEAM
		CursorPointer:           loadCur(32649), // IDC_HAND
		CursorResizeTopLeft:     loadCur(32642), // IDC_SIZENWSE
		CursorResizeTop:         loadCur(32645), // IDC_SIZENS
		CursorResizeTopRight:    loadCur(32643), // IDC_SIZENESW
		CursorResizeRight:       loadCur(32644), // IDC_SIZEWE
		CursorResizeBottomRight: loadCur(32642), // IDC_SIZENWSE
		CursorResizeBottom:      loadCur(32645), // IDC_SIZENS
		CursorResizeBottomLeft:  loadCur(32643), // IDC_SIZENESW
		CursorResizeLeft:        loadCur(32644), // IDC_SIZEWE
	}

	w := &Window{
		hwnd:          hwnd,
		Width:         width,
		Height:        height,
		eventCh:       make(chan Event, 1024),
		keyHandler:    NewKeyHandler(),
		cursors:       cursors,
		currentCursor: CursorDefault,
	}

	activeWinMu.Lock()
	activeWindows[hwnd] = w
	activeWinMu.Unlock()

	// Default opacity 100% (255)
	procSetLayeredWindowAttrs.Call(uintptr(hwnd), 0, 255, LWA_ALPHA)
	procShowWindow.Call(uintptr(hwnd), SW_SHOW)
	procUpdateWindow.Call(uintptr(hwnd))

	// Signal window creation successful
	resCh <- windowCreateResult{w: w}

	// Message loop running on the dedicated window-owning thread
	var msg MSG
	for {
		ret, _, _ := procGetMessageW.Call(
			uintptr(unsafe.Pointer(&msg)),
			0,
			0,
			0,
		)
		if int32(ret) <= 0 {
			close(w.eventCh)
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func getModifiers() uint16 {
	var state uint16
	isDown := func(vk int) bool {
		ret, _, _ := procGetKeyState.Call(uintptr(vk))
		return (int16(ret) & -0x8000) != 0
	}
	if isDown(VK_SHIFT) {
		state |= ModShift
	}
	if isDown(VK_CONTROL) {
		state |= ModCtrl
	}
	if isDown(VK_MENU) {
		state |= ModAlt
	}
	if isDown(1) { // VK_LBUTTON
		state |= ButtonMask1
	}
	return state
}

func wndProc(hwnd windows.Handle, msg, wParam, lParam uintptr) uintptr {
	activeWinMu.RLock()
	w, exists := activeWindows[hwnd]
	activeWinMu.RUnlock()

	if !exists {
		ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
		return ret
	}

	switch msg {
	case WM_MOUSEMOVE:
		x := int16(int32(lParam & 0xFFFF))
		y := int16(int32((lParam >> 16) & 0xFFFF))
		state := getModifiers()
		w.eventCh <- MotionNotifyEvent{
			EventX: x,
			EventY: y,
			RootX:  x,
			RootY:  y,
			State:  state,
		}
		return 0

	case WM_LBUTTONDOWN:
		x := int16(int32(lParam & 0xFFFF))
		y := int16(int32((lParam >> 16) & 0xFFFF))
		w.eventCh <- ButtonPressEvent{
			Detail: 1,
			State:  getModifiers(),
			EventX: x,
			EventY: y,
			RootX:  x,
			RootY:  y,
		}
		return 0

	case WM_LBUTTONUP:
		x := int16(int32(lParam & 0xFFFF))
		y := int16(int32((lParam >> 16) & 0xFFFF))
		w.eventCh <- ButtonReleaseEvent{
			Detail: 1,
			State:  getModifiers(),
			EventX: x,
			EventY: y,
			RootX:  x,
			RootY:  y,
		}
		return 0

	case WM_MBUTTONDOWN:
		x := int16(int32(lParam & 0xFFFF))
		y := int16(int32((lParam >> 16) & 0xFFFF))
		w.eventCh <- ButtonPressEvent{
			Detail: 2,
			State:  getModifiers(),
			EventX: x,
			EventY: y,
			RootX:  x,
			RootY:  y,
		}
		return 0

	case WM_MBUTTONUP:
		x := int16(int32(lParam & 0xFFFF))
		y := int16(int32((lParam >> 16) & 0xFFFF))
		w.eventCh <- ButtonReleaseEvent{
			Detail: 2,
			State:  getModifiers(),
			EventX: x,
			EventY: y,
			RootX:  x,
			RootY:  y,
		}
		return 0

	case WM_RBUTTONDOWN:
		x := int16(int32(lParam & 0xFFFF))
		y := int16(int32((lParam >> 16) & 0xFFFF))
		w.eventCh <- ButtonPressEvent{
			Detail: 3,
			State:  getModifiers(),
			EventX: x,
			EventY: y,
			RootX:  x,
			RootY:  y,
		}
		return 0

	case WM_RBUTTONUP:
		x := int16(int32(lParam & 0xFFFF))
		y := int16(int32((lParam >> 16) & 0xFFFF))
		w.eventCh <- ButtonReleaseEvent{
			Detail: 3,
			State:  getModifiers(),
			EventX: x,
			EventY: y,
			RootX:  x,
			RootY:  y,
		}
		return 0

	case WM_MOUSEWHEEL:
		delta := int16(wParam >> 16)
		detail := uint8(4) // Wheel Up
		if delta < 0 {
			detail = 5 // Wheel Down
		}
		x := int16(int32(lParam & 0xFFFF))
		y := int16(int32((lParam >> 16) & 0xFFFF))
		w.eventCh <- ButtonPressEvent{
			Detail: detail,
			State:  getModifiers(),
			EventX: x,
			EventY: y,
			RootX:  x,
			RootY:  y,
		}
		return 0

	case WM_KEYDOWN, WM_SYSKEYDOWN:
		vk := uint32(wParam)
		state := getModifiers()
		action := w.keyHandler.LookupAction(vk, state)

		var ksym uint32
		switch vk {
		case VK_ESCAPE:
			ksym = 0xff1b
		case VK_RETURN:
			ksym = 0xff0d
		case VK_BACK:
			ksym = 0xff08
		case VK_TAB:
			ksym = 0xff09
		case VK_UP:
			ksym = 0xff52
		case VK_DOWN:
			ksym = 0xff54
		case VK_LEFT:
			ksym = 0xff51
		case VK_RIGHT:
			ksym = 0xff53
		default:
			if vk >= '0' && vk <= '9' {
				ksym = vk
			} else if vk >= 'A' && vk <= 'Z' {
				if (state & ModShift) != 0 {
					ksym = vk
				} else {
					ksym = vk + 32 // lowercase
				}
			}
		}

		if action != ActionNone {
			w.eventCh <- KeyPressEvent{
				Detail: byte(vk),
				State:  state,
				KeySym: ksym,
				Action: action,
			}
			return 0
		}

		if bytes, ok := w.keyHandler.LookupSpecialKey(vk, state); ok {
			w.eventCh <- KeyPressEvent{
				Detail: byte(vk),
				State:  state,
				KeySym: ksym,
				Bytes:  bytes,
				Action: ActionNone,
			}
			return 0
		}

	case WM_CHAR:
		r := rune(wParam)
		if r >= 32 && r != 127 { // Printable character
			state := getModifiers()
			w.eventCh <- KeyPressEvent{
				Detail: 0,
				State:  state,
				KeySym: uint32(r),
				Bytes:  CharToBytes(r),
				Action: ActionNone,
			}
			return 0
		}

	case WM_SIZE:
		if wParam == 1 /* SIZE_MINIMIZED */ {
			return 0
		}
		width := uint16(lParam & 0xFFFF)
		height := uint16((lParam >> 16) & 0xFFFF)
		if width == 0 || height == 0 {
			return 0
		}
		w.Width = width
		w.Height = height
		w.eventCh <- ConfigureNotifyEvent{
			Width:  width,
			Height: height,
		}
		return 0

	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		w.mu.Lock()
		if len(w.backbuffer) > 0 && w.backbufferW > 0 && w.backbufferH > 0 {
			w.drawDIB(windows.Handle(hdc), w.backbuffer, w.backbufferW, w.backbufferH)
		}
		w.mu.Unlock()
		procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		w.eventCh <- ExposeEvent{}
		return 0

	case WM_KILLFOCUS:
		w.eventCh <- FocusOutEvent{}
		return 0

	case WM_CLOSE:
		w.eventCh <- CloseRequestEvent{}
		return 0

	case WM_DESTROY:
		activeWinMu.Lock()
		delete(activeWindows, hwnd)
		activeWinMu.Unlock()
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func (w *Window) drawDIB(hdc windows.Handle, pixels []byte, width, height int) {
	bmi := BITMAPINFO{
		BmiHeader: BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			BiWidth:       int32(width),
			BiHeight:      -int32(height), // Negative for top-down bitmap
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: BI_RGB,
		},
	}

	procStretchDIBits.Call(
		uintptr(hdc),
		0, 0, uintptr(width), uintptr(height),
		0, 0, uintptr(width), uintptr(height),
		uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&bmi)),
		DIB_RGB_COLORS,
		SRCCOPY,
	)
}

func (w *Window) Events() <-chan Event {
	return w.eventCh
}

func (w *Window) Blit(pixels []byte, width, height uint16) {
	if len(pixels) == 0 || width == 0 || height == 0 {
		return
	}

	w.mu.Lock()
	if len(w.backbuffer) < len(pixels) {
		w.backbuffer = make([]byte, len(pixels))
	}
	copy(w.backbuffer, pixels)
	w.backbufferW = int(width)
	w.backbufferH = int(height)
	w.mu.Unlock()

	// Direct blit to DC for maximum speed (sub-millisecond DMA)
	hdc, _, _ := procGetDC.Call(uintptr(w.hwnd))
	if hdc != 0 {
		w.drawDIB(windows.Handle(hdc), pixels, int(width), int(height))
		procReleaseDC.Call(uintptr(w.hwnd), hdc)
	}
}

func (w *Window) SetTitle(title string) {
	ptr, err := windows.UTF16PtrFromString(title)
	if err == nil {
		procSetWindowTextW.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(ptr)))
	}
}

func (w *Window) SetOpacity(opacity float64) {
	if opacity < 0.1 {
		opacity = 0.1
	}
	if opacity > 1.0 {
		opacity = 1.0
	}
	alpha := byte(opacity * 255.0)
	procSetLayeredWindowAttrs.Call(uintptr(w.hwnd), 0, uintptr(alpha), LWA_ALPHA)
}

func (w *Window) SetCursorType(ct CursorType) {
	if ct == w.currentCursor {
		return
	}
	w.currentCursor = ct
	if cur, ok := w.cursors[ct]; ok && cur != 0 {
		procSetCursor.Call(uintptr(cur))
	}
}

func (w *Window) StartDrag(rootX, rootY int16) {
	procReleaseCapture.Call()
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	lParam := uintptr(uint32(uint16(pt.X)) | (uint32(uint16(pt.Y)) << 16))
	procSendMessageW.Call(uintptr(w.hwnd), WM_NCLBUTTONDOWN, HTCAPTION, lParam)
}

func (w *Window) StartResize(direction int, rootX, rootY int16) {
	procReleaseCapture.Call()
	var ht uintptr
	switch direction {
	case ResizeTopLeft:
		ht = HTTOPLEFT
	case ResizeTop:
		ht = HTTOP
	case ResizeTopRight:
		ht = HTTOPRIGHT
	case ResizeRight:
		ht = HTRIGHT
	case ResizeBottomRight:
		ht = HTBOTTOMRIGHT
	case ResizeBottom:
		ht = HTBOTTOM
	case ResizeBottomLeft:
		ht = HTBOTTOMLEFT
	case ResizeLeft:
		ht = HTLEFT
	default:
		return
	}
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	lParam := uintptr(uint32(uint16(pt.X)) | (uint32(uint16(pt.Y)) << 16))
	procSendMessageW.Call(uintptr(w.hwnd), WM_NCLBUTTONDOWN, ht, lParam)
}

func (w *Window) Minimize() {
	procShowWindow.Call(uintptr(w.hwnd), SW_MINIMIZE)
}

func (w *Window) ToggleMaximize() {
	isMax, _, _ := procIsZoomed.Call(uintptr(w.hwnd))
	if isMax != 0 {
		procShowWindow.Call(uintptr(w.hwnd), SW_RESTORE)
	} else {
		procShowWindow.Call(uintptr(w.hwnd), SW_MAXIMIZE)
	}
}

func (w *Window) SetClipboard(text string) {
	if text == "" {
		return
	}
	ret, _, _ := procOpenClipboard.Call(uintptr(w.hwnd))
	if ret == 0 {
		return
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()

	utf16Units := utf16.Encode([]rune(text + "\x00"))
	size := len(utf16Units) * 2

	hMem, _, _ := procGlobalAlloc.Call(GMEM_MOVEABLE, uintptr(size))
	if hMem == 0 {
		return
	}

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return
	}

	procRtlMoveMemory.Call(ptr, uintptr(unsafe.Pointer(&utf16Units[0])), uintptr(size))
	procGlobalUnlock.Call(hMem)

	procSetClipboardData.Call(CF_UNICODETEXT, hMem)
}

func (w *Window) SetPrimary(text string) {
	w.SetClipboard(text)
}

func (w *Window) Paste() {
	ret, _, _ := procOpenClipboard.Call(uintptr(w.hwnd))
	if ret == 0 {
		return
	}
	defer procCloseClipboard.Call()

	hData, _, _ := procGetClipboardData.Call(CF_UNICODETEXT)
	if hData == 0 {
		return
	}

	ptr, _, _ := procGlobalLock.Call(hData)
	if ptr == 0 {
		return
	}
	defer procGlobalUnlock.Call(hData)

	text := windows.UTF16PtrToString((*uint16)(unsafe.Pointer(ptr)))
	if text != "" {
		w.eventCh <- PasteNotifyEvent{Text: text}
	}
}

func (w *Window) PastePrimary() {
	w.Paste()
}

func (w *Window) Close() {
	procPostMessageW.Call(uintptr(w.hwnd), WM_CLOSE, 0, 0)
}
