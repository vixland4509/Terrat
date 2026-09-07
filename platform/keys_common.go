package platform

const (
	ModShift    = 1 << 0
	ModLock     = 1 << 1
	ModCtrl     = 1 << 2
	ModAlt      = 1 << 3
	ButtonMask1 = 1 << 8
)

type ActionType int

const (
	ActionNone ActionType = iota
	ActionScrollUp
	ActionScrollDown
	ActionScrollTop
	ActionScrollBottom
	ActionCopy
	ActionPaste
	ActionSelectAll
	ActionPreferences
	ActionZoomIn
	ActionZoomOut
	ActionZoomReset
	ActionNewTab
	ActionCloseTab
	ActionNextTab
	ActionPrevTab
	ActionSearch
	ActionSwitchTab1
	ActionSwitchTab2
	ActionSwitchTab3
	ActionSwitchTab4
	ActionSwitchTab5
	ActionSwitchTab6
	ActionSwitchTab7
	ActionSwitchTab8
	ActionSwitchTab9
	ActionToggleDiagnostics
)
