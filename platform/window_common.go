package platform

type CursorType int

const (
	CursorDefault CursorType = iota
	CursorText
	CursorPointer
	CursorResizeTopLeft
	CursorResizeTop
	CursorResizeTopRight
	CursorResizeRight
	CursorResizeBottomRight
	CursorResizeBottom
	CursorResizeBottomLeft
	CursorResizeLeft
)

const (
	ResizeTopLeft     = 0
	ResizeTop         = 1
	ResizeTopRight    = 2
	ResizeRight       = 3
	ResizeBottomRight = 4
	ResizeBottom      = 5
	ResizeBottomLeft  = 6
	ResizeLeft        = 7
	ResizeMove        = 8
	ResizeNone        = -1
)

func GetResizeDirection(x, y int, width, height int, borderThreshold int) int {
	onLeft := x < borderThreshold
	onRight := x >= width-borderThreshold
	onTop := y < borderThreshold
	onBottom := y >= height-borderThreshold

	switch {
	case onTop && onLeft:
		return ResizeTopLeft
	case onTop && onRight:
		return ResizeTopRight
	case onBottom && onLeft:
		return ResizeBottomLeft
	case onBottom && onRight:
		return ResizeBottomRight
	case onTop:
		return ResizeTop
	case onBottom:
		return ResizeBottom
	case onLeft:
		return ResizeLeft
	case onRight:
		return ResizeRight
	default:
		return ResizeNone
	}
}
