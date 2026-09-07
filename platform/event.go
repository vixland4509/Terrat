package platform

// Event is the common interface implemented by all platform window events.
type Event interface {
	isPlatformEvent()
}

// MotionNotifyEvent represents mouse movement.
type MotionNotifyEvent struct {
	EventX int16
	EventY int16
	RootX  int16
	RootY  int16
	State  uint16
}

func (MotionNotifyEvent) isPlatformEvent() {}

// ButtonPressEvent represents a mouse button press or scroll wheel step.
// Detail: 1=Left, 2=Middle, 3=Right, 4=ScrollUp, 5=ScrollDown.
type ButtonPressEvent struct {
	Detail uint8
	State  uint16
	EventX int16
	EventY int16
	RootX  int16
	RootY  int16
}

func (ButtonPressEvent) isPlatformEvent() {}

// ButtonReleaseEvent represents a mouse button release.
type ButtonReleaseEvent struct {
	Detail uint8
	State  uint16
	EventX int16
	EventY int16
	RootX  int16
	RootY  int16
}

func (ButtonReleaseEvent) isPlatformEvent() {}

// KeyPressEvent represents a key press with pre-resolved string and action.
type KeyPressEvent struct {
	Detail uint8
	State  uint16
	KeySym uint32
	Bytes  []byte
	Action ActionType
}

func (KeyPressEvent) isPlatformEvent() {}

// ConfigureNotifyEvent represents a window resize or reconfiguration.
type ConfigureNotifyEvent struct {
	Width  uint16
	Height uint16
}

func (ConfigureNotifyEvent) isPlatformEvent() {}

// ExposeEvent requests a redraw of the window area.
type ExposeEvent struct{}

func (ExposeEvent) isPlatformEvent() {}

// FocusOutEvent is fired when the window loses focus.
type FocusOutEvent struct{}

func (FocusOutEvent) isPlatformEvent() {}

// PasteNotifyEvent is fired when clipboard text has arrived.
type PasteNotifyEvent struct {
	Text string
}

func (PasteNotifyEvent) isPlatformEvent() {}

// CloseRequestEvent is fired when the user clicks the window close button or sends WM_DELETE_WINDOW.
type CloseRequestEvent struct{}

func (CloseRequestEvent) isPlatformEvent() {}

// MappingNotifyEvent is fired when the keyboard mapping changes.
type MappingNotifyEvent struct{}

func (MappingNotifyEvent) isPlatformEvent() {}
