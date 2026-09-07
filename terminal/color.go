package terminal

type Color uint32

const (
	ColorDefaultFG Color = 0x01000000
	ColorDefaultBG Color = 0x02000000
)

var (
	ColorCursor      = Color(0x7aa2f7)
	ColorSelectionBG = Color(0x364a82)
	ColorSelectionFG = Color(0xffffff)
	ColorHeaderBG    = Color(0x13141c)
	ColorHeaderLine  = Color(0x292e42)
	ColorBorder      = Color(0x3b4261)
	ColorCloseDot    = Color(0xf7768e)
	ColorMinDot      = Color(0xe0af68)
	ColorMaxDot      = Color(0x9ece6a)
	ColorBadgeText   = Color(0x7dcfff)
	ColorMutedText   = Color(0x565f89)
)

var ansi16 = [16]Color{
	0x15161e,
	0xf7768e,
	0x9ece6a,
	0xe0af68,
	0x7aa2f7,
	0xbb9af7,
	0x7dcfff,
	0xa9b1d6,
	0x414868,
	0xf7768e,
	0x73daca,
	0xe0af68,
	0x7aa2f7,
	0xbb9af7,
	0x2ac3de,
	0xc0caf5,
}

func RGB(r, g, b byte) Color {
	return Color(uint32(r)<<16 | uint32(g)<<8 | uint32(b))
}

func (c Color) ToRGBA() (r, g, b, a byte) {
	return byte((c >> 16) & 0xff), byte((c >> 8) & 0xff), byte(c & 0xff), 0xff
}

func (c Color) ToPixel() uint32 {
	r, g, b, a := c.ToRGBA()
	return uint32(b) | (uint32(g) << 8) | (uint32(r) << 16) | (uint32(a) << 24)
}

func ANSI256(index int) Color {
	if index < 0 || index > 255 {
		return ColorDefaultFG
	}
	if index < 16 {
		return ansi16[index]
	}
	if index < 232 {
		idx := index - 16
		r := byte((idx / 36) * 51)
		g := byte(((idx % 36) / 6) * 51)
		b := byte((idx % 6) * 51)
		return RGB(r, g, b)
	}
	gray := byte(8 + (index-232)*10)
	return RGB(gray, gray, gray)
}
