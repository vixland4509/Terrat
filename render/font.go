package render

import (
	"image"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type GlyphMask struct {
	Width  int
	Height int
	Alpha  []byte
}

type FontEngine struct {
	charWidth  int
	charHeight int
	baseline   int
	face       font.Face
	otf        *opentype.Font
	fontSize   float64

	asciiCache [96][2]*GlyphMask
	otherCache map[rune]*GlyphMask
}

var candidateTTFFonts = []string{
	// Windows standard fonts
	`C:\Windows\Fonts\CascadiaMono.ttf`,
	`C:\Windows\Fonts\CascadiaCode.ttf`,
	`C:\Windows\Fonts\consola.ttf`,
	`C:\Windows\Fonts\lucon.ttf`,
	// Linux standard fonts
	"/usr/share/fonts/truetype/hack/Hack-Regular.ttf",
	"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
	"/usr/share/fonts/truetype/liberation/LiberationMono-Regular.ttf",
	"/usr/share/fonts/truetype/ubuntu/UbuntuMono-R.ttf",
	"/usr/share/fonts/truetype/freefont/FreeMono.ttf",
}

func init() {
	if windir := os.Getenv("WINDIR"); windir != "" {
		candidateTTFFonts = append([]string{
			filepath.Join(windir, "Fonts", "CascadiaMono.ttf"),
			filepath.Join(windir, "Fonts", "CascadiaCode.ttf"),
			filepath.Join(windir, "Fonts", "consola.ttf"),
			filepath.Join(windir, "Fonts", "lucon.ttf"),
		}, candidateTTFFonts...)
	}
}

func NewFontEngine(fontSize float64) (*FontEngine, error) {
	fe := &FontEngine{
		otherCache: make(map[rune]*GlyphMask),
		fontSize:   fontSize,
	}

	var loadedFace font.Face
	var loadedOTF *opentype.Font
	for _, path := range candidateTTFFonts {
		if data, err := os.ReadFile(path); err == nil {
			if otf, err := opentype.Parse(data); err == nil {
				face, err := opentype.NewFace(otf, &opentype.FaceOptions{
					Size:    fontSize,
					DPI:     72,
					Hinting: font.HintingFull,
				})
				if err == nil {
					loadedFace = face
					loadedOTF = otf
					break
				}
			}
		}
	}

	if loadedFace != nil {
		fe.face = loadedFace
		fe.otf = loadedOTF
		metrics := loadedFace.Metrics()
		fe.charHeight = metrics.Height.Ceil()
		fe.baseline = metrics.Ascent.Ceil()

		adv, ok := loadedFace.GlyphAdvance('M')
		if ok && adv > 0 {
			fe.charWidth = adv.Ceil()
		} else {
			fe.charWidth = 8
		}
	} else {
		fe.face = basicfont.Face7x13
		fe.charWidth = 7
		fe.charHeight = 13
		fe.baseline = 11
	}

	if fe.charWidth < 6 {
		fe.charWidth = 6
	}
	if fe.charHeight < 10 {
		fe.charHeight = 10
	}

	fe.precacheASCII()

	return fe, nil
}

func (fe *FontEngine) precacheASCII() {
	for r := rune(32); r < 127; r++ {
		idx := int(r - 32)
		fe.asciiCache[idx][0] = fe.rasterizeRune(r, false)
		fe.asciiCache[idx][1] = fe.rasterizeRune(r, true)
	}
}

func (fe *FontEngine) CharWidth() int {
	return fe.charWidth
}

func (fe *FontEngine) CharHeight() int {
	return fe.charHeight
}

func (fe *FontEngine) Baseline() int {
	return fe.baseline
}

func (fe *FontEngine) FontSize() float64 {
	return fe.fontSize
}

func (fe *FontEngine) SetFontSize(size float64) error {
	if size < 7.0 {
		size = 7.0
	}
	if size > 36.0 {
		size = 36.0
	}
	if fe.fontSize == size {
		return nil
	}

	fe.fontSize = size
	if fe.otf != nil {
		face, err := opentype.NewFace(fe.otf, &opentype.FaceOptions{
			Size:    size,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err != nil {
			return err
		}
		if fe.face != nil && fe.face != basicfont.Face7x13 {
			_ = fe.face.Close()
		}
		fe.face = face

		metrics := face.Metrics()
		fe.charHeight = metrics.Height.Ceil()
		fe.baseline = metrics.Ascent.Ceil()

		adv, ok := face.GlyphAdvance('M')
		if ok && adv > 0 {
			fe.charWidth = adv.Ceil()
		} else {
			fe.charWidth = 8
		}
	} else {
		fe.charWidth = 7
		fe.charHeight = 13
		fe.baseline = 11
	}

	if fe.charWidth < 6 {
		fe.charWidth = 6
	}
	if fe.charHeight < 10 {
		fe.charHeight = 10
	}

	fe.otherCache = make(map[rune]*GlyphMask)
	fe.precacheASCII()
	return nil
}

func (fe *FontEngine) ZoomIn() bool {
	if fe.fontSize >= 32.0 {
		return false
	}
	_ = fe.SetFontSize(fe.fontSize + 1.0)
	return true
}

func (fe *FontEngine) ZoomOut() bool {
	if fe.fontSize <= 8.0 {
		return false
	}
	_ = fe.SetFontSize(fe.fontSize - 1.0)
	return true
}

func (fe *FontEngine) ZoomReset() bool {
	if fe.fontSize == 13.0 {
		return false
	}
	_ = fe.SetFontSize(13.0)
	return true
}

func (fe *FontEngine) GetGlyph(r rune, bold bool) *GlyphMask {
	boldIdx := 0
	if bold {
		boldIdx = 1
	}

	if r >= 32 && r < 127 {
		idx := int(r - 32)
		mask := fe.asciiCache[idx][boldIdx]
		if mask != nil {
			return mask
		}
	}

	if mask, ok := fe.otherCache[r]; ok {
		return mask
	}

	mask := fe.rasterizeRune(r, bold)
	fe.otherCache[r] = mask
	return mask
}

func (fe *FontEngine) rasterizeRune(r rune, bold bool) *GlyphMask {
	dst := image.NewAlpha(image.Rect(0, 0, fe.charWidth, fe.charHeight))

	dot := fixed.P(0, fe.baseline)
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.Opaque,
		Face: fe.face,
		Dot:  dot,
	}
	d.DrawString(string(r))

	if bold {
		d.Dot = fixed.P(1, fe.baseline)
		d.DrawString(string(r))
	}

	mask := &GlyphMask{
		Width:  fe.charWidth,
		Height: fe.charHeight,
		Alpha:  make([]byte, fe.charWidth*fe.charHeight),
	}

	for y := 0; y < fe.charHeight; y++ {
		for x := 0; x < fe.charWidth; x++ {
			c := dst.AlphaAt(x, y)
			mask.Alpha[y*fe.charWidth+x] = c.A
		}
	}

	return mask
}

func (fe *FontEngine) Close() {
	if fe.face != nil {
		_ = fe.face.Close()
	}
}

func DrawGlyphBlit(buf []byte, stride int, cellX, cellY int, mask *GlyphMask, fgPixel, bgPixel uint32, underline bool, baseline int) {
	if mask == nil {
		return
	}
	width := mask.Width
	height := mask.Height

	fgR := byte(fgPixel >> 16)
	fgG := byte(fgPixel >> 8)
	fgB := byte(fgPixel)

	bgR := byte(bgPixel >> 16)
	bgG := byte(bgPixel >> 8)
	bgB := byte(bgPixel)

	for y := 0; y < height; y++ {
		py := cellY + y
		if py < 0 || py*stride >= len(buf) {
			continue
		}
		bufRowOffset := py*stride + cellX*4
		maskRowOffset := y * width

		for x := 0; x < width; x++ {
			pixelOffset := bufRowOffset + x*4
			if pixelOffset < 0 || pixelOffset+3 >= len(buf) {
				continue
			}

			alpha := int(mask.Alpha[maskRowOffset+x])
			if alpha == 0 {
				buf[pixelOffset+0] = bgB
				buf[pixelOffset+1] = bgG
				buf[pixelOffset+2] = bgR
				buf[pixelOffset+3] = 0xff
			} else if alpha >= 255 {
				buf[pixelOffset+0] = fgB
				buf[pixelOffset+1] = fgG
				buf[pixelOffset+2] = fgR
				buf[pixelOffset+3] = 0xff
			} else {
				inv := 255 - alpha
				outB := byte((int(fgB)*alpha + int(bgB)*inv) / 255)
				outG := byte((int(fgG)*alpha + int(bgG)*inv) / 255)
				outR := byte((int(fgR)*alpha + int(bgR)*inv) / 255)

				buf[pixelOffset+0] = outB
				buf[pixelOffset+1] = outG
				buf[pixelOffset+2] = outR
				buf[pixelOffset+3] = 0xff
			}
		}
	}

	if underline && baseline+1 < height {
		py := cellY + baseline + 1
		if py >= 0 && py*stride < len(buf) {
			bufRowOffset := py*stride + cellX*4
			for x := 0; x < width; x++ {
				pixelOffset := bufRowOffset + x*4
				if pixelOffset >= 0 && pixelOffset+3 < len(buf) {
					buf[pixelOffset+0] = fgB
					buf[pixelOffset+1] = fgG
					buf[pixelOffset+2] = fgR
					buf[pixelOffset+3] = 0xff
				}
			}
		}
	}
}

func FillRect(buf []byte, stride int, x, y, w, h int, pixel uint32) {
	if w <= 0 || h <= 0 {
		return
	}
	b := byte(pixel)
	g := byte(pixel >> 8)
	r := byte(pixel >> 16)

	firstRowOffset := y*stride + x*4
	if firstRowOffset < 0 || firstRowOffset+w*4 > len(buf) {
		for row := 0; row < h; row++ {
			py := y + row
			if py < 0 || py*stride >= len(buf) {
				continue
			}
			rowOffset := py * stride
			for col := 0; col < w; col++ {
				p := rowOffset + (x+col)*4
				if p >= 0 && p+3 < len(buf) {
					buf[p+0] = b
					buf[p+1] = g
					buf[p+2] = r
					buf[p+3] = 0xff
				}
			}
		}
		return
	}

	for col := 0; col < w; col++ {
		p := firstRowOffset + col*4
		buf[p+0] = b
		buf[p+1] = g
		buf[p+2] = r
		buf[p+3] = 0xff
	}
	firstRow := buf[firstRowOffset : firstRowOffset+w*4]

	for row := 1; row < h; row++ {
		py := y + row
		rowOffset := py*stride + x*4
		if rowOffset+w*4 <= len(buf) {
			copy(buf[rowOffset:rowOffset+w*4], firstRow)
		}
	}
}

func DrawCircle(buf []byte, stride int, cx, cy, radius int, pixel uint32) {
	b := byte(pixel)
	g := byte(pixel >> 8)
	r := byte(pixel >> 16)
	rSquared := radius * radius

	for dy := -radius; dy <= radius; dy++ {
		py := cy + dy
		if py < 0 || py*stride >= len(buf) {
			continue
		}
		rowOffset := py * stride
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= rSquared {
				px := cx + dx
				p := rowOffset + px*4
				if p >= 0 && p+3 < len(buf) {
					buf[p+0] = b
					buf[p+1] = g
					buf[p+2] = r
					buf[p+3] = 0xff
				}
			}
		}
	}
}

func (fe *FontEngine) DrawString(buf []byte, stride int, startX, startY int, text string, fgPixel, bgPixel uint32, bold bool) {
	curX := startX
	for _, ch := range text {
		mask := fe.GetGlyph(ch, bold)
		DrawGlyphBlit(buf, stride, curX, startY, mask, fgPixel, bgPixel, false, fe.baseline)
		curX += fe.charWidth
	}
}

func DrawGlyphBlitTransparent(buf []byte, stride int, cellX, cellY int, mask *GlyphMask, fgPixel uint32, underline bool, baseline int) {
	if mask == nil {
		return
	}
	width := mask.Width
	height := mask.Height

	fgR := byte(fgPixel >> 16)
	fgG := byte(fgPixel >> 8)
	fgB := byte(fgPixel)

	for y := 0; y < height; y++ {
		py := cellY + y
		if py < 0 || py*stride >= len(buf) {
			continue
		}
		bufRowOffset := py*stride + cellX*4
		maskRowOffset := y * width

		for x := 0; x < width; x++ {
			alpha := int(mask.Alpha[maskRowOffset+x])
			if alpha == 0 {
				continue
			}
			pixelOffset := bufRowOffset + x*4
			if pixelOffset < 0 || pixelOffset+3 >= len(buf) {
				continue
			}

			if alpha >= 255 {
				buf[pixelOffset+0] = fgB
				buf[pixelOffset+1] = fgG
				buf[pixelOffset+2] = fgR
				buf[pixelOffset+3] = 0xff
			} else {
				inv := 255 - alpha
				buf[pixelOffset+0] = byte((int(fgB)*alpha + int(buf[pixelOffset+0])*inv) / 255)
				buf[pixelOffset+1] = byte((int(fgG)*alpha + int(buf[pixelOffset+1])*inv) / 255)
				buf[pixelOffset+2] = byte((int(fgR)*alpha + int(buf[pixelOffset+2])*inv) / 255)
				buf[pixelOffset+3] = 0xff
			}
		}
	}

	if underline && baseline+1 < height {
		py := cellY + baseline + 1
		if py >= 0 && py*stride < len(buf) {
			bufRowOffset := py*stride + cellX*4
			for x := 0; x < width; x++ {
				pixelOffset := bufRowOffset + x*4
				if pixelOffset >= 0 && pixelOffset+3 < len(buf) {
					buf[pixelOffset+0] = fgB
					buf[pixelOffset+1] = fgG
					buf[pixelOffset+2] = fgR
					buf[pixelOffset+3] = 0xff
				}
			}
		}
	}
}

func MinecraftShadow(pixel uint32) uint32 {
	r := ((pixel >> 16) & 0xff) / 4
	g := ((pixel >> 8) & 0xff) / 4
	b := (pixel & 0xff) / 4
	return (r << 16) | (g << 8) | b
}

func (fe *FontEngine) DrawStringShadow(buf []byte, stride int, startX, startY int, text string, fgPixel, shadowPixel uint32, bold bool) {
	curX := startX + 1
	curY := startY + 1
	for _, ch := range text {
		mask := fe.GetGlyph(ch, bold)
		DrawGlyphBlitTransparent(buf, stride, curX, curY, mask, shadowPixel, false, fe.baseline)
		curX += fe.charWidth
	}
	curX = startX
	for _, ch := range text {
		mask := fe.GetGlyph(ch, bold)
		DrawGlyphBlitTransparent(buf, stride, curX, startY, mask, fgPixel, false, fe.baseline)
		curX += fe.charWidth
	}
}

func DrawHLine(buf []byte, stride int, x, y, width int, pixel uint32) {
	if y < 0 || y*stride >= len(buf) {
		return
	}
	b := byte(pixel)
	g := byte(pixel >> 8)
	r := byte(pixel >> 16)
	rowOffset := y * stride
	for col := 0; col < width; col++ {
		p := rowOffset + (x+col)*4
		if p >= 0 && p+3 < len(buf) {
			buf[p+0] = b
			buf[p+1] = g
			buf[p+2] = r
			buf[p+3] = 0xff
		}
	}
}

func DrawVLine(buf []byte, stride int, x, y, height int, pixel uint32) {
	b := byte(pixel)
	g := byte(pixel >> 8)
	r := byte(pixel >> 16)
	for row := 0; row < height; row++ {
		py := y + row
		p := py*stride + x*4
		if p >= 0 && p+3 < len(buf) {
			buf[p+0] = b
			buf[p+1] = g
			buf[p+2] = r
			buf[p+3] = 0xff
		}
	}
}

func DrawRectBorder(buf []byte, stride int, x, y, w, h int, pixel uint32) {
	DrawHLine(buf, stride, x, y, w, pixel)
	DrawHLine(buf, stride, x, y+h-1, w, pixel)
	DrawVLine(buf, stride, x, y, h, pixel)
	DrawVLine(buf, stride, x+w-1, y, h, pixel)
}
