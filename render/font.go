package render

import (
	"image"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

type GlyphMask struct {
	Width  int
	Height int
	Alpha  []byte
}

type fallbackFont struct {
	face font.Face
	otf  *opentype.Font
}

type FontEngine struct {
	charWidth  int
	charHeight int
	baseline   int
	face       font.Face
	otf        *opentype.Font
	fontSize   float64

	fallbacks []fallbackFont

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

var candidateFallbackFonts = []string{
	// Arabic / Persian
	"/usr/share/fonts/truetype/noto/NotoSansArabic-Regular.ttf",
	"/usr/share/fonts/truetype/noto/NotoSansArabicUI-Regular.ttf",
	"/usr/share/fonts/truetype/noto/NotoSansArabicUI-Bold.ttf",
	"/usr/share/fonts/truetype/ibm-plex/IBMPlexSansArabic-Regular.ttf",
	"/usr/share/fonts/truetype/ibm-plex/IBMPlexSansArabic-Medium.ttf",
	// Symbols & Math
	"/usr/share/fonts/truetype/noto/NotoSansSymbols-Regular.ttf",
	"/usr/share/fonts/truetype/noto/NotoSansSymbols2-Regular.ttf",
	"/usr/share/fonts/truetype/noto/NotoSansMath-Regular.ttf",
	// General Unicode & CJK
	"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
	"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/truetype/freefont/FreeSans.ttf",
	// Windows fallbacks
	`C:\Windows\Fonts\segoeui.ttf`,
	`C:\Windows\Fonts\seguiemj.ttf`,
	`C:\Windows\Fonts\seguisym.ttf`,
	`C:\Windows\Fonts\arial.ttf`,
}

func init() {
	if windir := os.Getenv("WINDIR"); windir != "" {
		candidateTTFFonts = append([]string{
			filepath.Join(windir, "Fonts", "CascadiaMono.ttf"),
			filepath.Join(windir, "Fonts", "CascadiaCode.ttf"),
			filepath.Join(windir, "Fonts", "consola.ttf"),
			filepath.Join(windir, "Fonts", "lucon.ttf"),
		}, candidateTTFFonts...)
		candidateFallbackFonts = append([]string{
			filepath.Join(windir, "Fonts", "segoeui.ttf"),
			filepath.Join(windir, "Fonts", "seguiemj.ttf"),
			filepath.Join(windir, "Fonts", "seguisym.ttf"),
			filepath.Join(windir, "Fonts", "arial.ttf"),
		}, candidateFallbackFonts...)
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

	fe.loadFallbacks(fontSize)
	fe.precacheASCII()

	return fe, nil
}

func (fe *FontEngine) loadFallbacks(fontSize float64) {
	for _, fb := range fe.fallbacks {
		if fb.face != nil && fb.face != basicfont.Face7x13 {
			_ = fb.face.Close()
		}
	}
	fe.fallbacks = nil

	for _, path := range candidateFallbackFonts {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		otf, err := opentype.Parse(data)
		if err != nil {
			continue
		}
		face, err := opentype.NewFace(otf, &opentype.FaceOptions{
			Size:    fontSize,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err != nil {
			continue
		}
		fe.fallbacks = append(fe.fallbacks, fallbackFont{
			face: face,
			otf:  otf,
		})
	}
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

	fe.loadFallbacks(size)
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
	// Programmatic Box Drawing & Block Elements
	if box := renderBoxOrBlock(r, fe.charWidth, fe.charHeight, fe.baseline); box != nil {
		return box
	}

	var buf sfnt.Buffer
	useFace := fe.face
	found := true

	if fe.otf != nil {
		idx, err := fe.otf.GlyphIndex(&buf, r)
		if err != nil || idx == 0 {
			// Not in primary font, search fallbacks
			found = false
			for _, fb := range fe.fallbacks {
				if fb.otf != nil {
					fidx, ferr := fb.otf.GlyphIndex(&buf, r)
					if ferr == nil && fidx != 0 {
						useFace = fb.face
						found = true
						break
					}
				}
			}
		}
	}

	if !found {
		// Missing from all fonts: return blank/transparent glyph instead of .notdef box
		return &GlyphMask{
			Width:  fe.charWidth,
			Height: fe.charHeight,
			Alpha:  make([]byte, fe.charWidth*fe.charHeight),
		}
	}

	dst := image.NewAlpha(image.Rect(0, 0, fe.charWidth, fe.charHeight))

	dot := fixed.P(0, fe.baseline)
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.Opaque,
		Face: useFace,
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
	if fe.face != nil && fe.face != basicfont.Face7x13 {
		_ = fe.face.Close()
	}
	for _, fb := range fe.fallbacks {
		if fb.face != nil && fb.face != basicfont.Face7x13 {
			_ = fb.face.Close()
		}
	}
}

func renderBoxOrBlock(r rune, w, h, baseline int) *GlyphMask {
	if w <= 0 || h <= 0 {
		return nil
	}

	mask := &GlyphMask{
		Width:  w,
		Height: h,
		Alpha:  make([]byte, w*h),
	}

	setPixel := func(x, y int, alpha byte) {
		if x >= 0 && x < w && y >= 0 && y < h {
			mask.Alpha[y*w+x] = alpha
		}
	}
	fillRect := func(x0, y0, x1, y1 int, alpha byte) {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				setPixel(x, y, alpha)
			}
		}
	}

	// 1. Block Elements (U+2580 - U+259F)
	if r >= 0x2580 && r <= 0x259f {
		switch r {
		case 0x2588: // Full block █
			fillRect(0, 0, w, h, 0xff)
			return mask
		case 0x2580: // Upper half block ▀
			fillRect(0, 0, w, h/2, 0xff)
			return mask
		case 0x2584: // Lower half block ▄
			fillRect(0, h/2, w, h, 0xff)
			return mask
		case 0x258c: // Left half block ▌
			fillRect(0, 0, w/2, h, 0xff)
			return mask
		case 0x2590: // Right half block ▐
			fillRect(w/2, 0, w, h, 0xff)
			return mask
		case 0x2594: // Upper 1/8 block
			fillRect(0, 0, w, (h+7)/8, 0xff)
			return mask
		case 0x2595: // Right 1/8 block
			fillRect(w-(w+7)/8, 0, w, h, 0xff)
			return mask
		case 0x2591: // Light shade ░
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					if (x+y)%4 == 0 {
						setPixel(x, y, 0xff)
					}
				}
			}
			return mask
		case 0x2592: // Medium shade ▒
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					if (x+y)%2 == 0 {
						setPixel(x, y, 0xff)
					}
				}
			}
			return mask
		case 0x2593: // Dark shade ▓
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					if (x+y)%4 != 0 {
						setPixel(x, y, 0xff)
					}
				}
			}
			return mask
		}
		// Lower 1/8 to 7/8 (0x2581 - 0x2587)
		if r >= 0x2581 && r <= 0x2587 {
			eighths := int(r - 0x2580)
			blockH := (h * eighths) / 8
			if blockH < 1 {
				blockH = 1
			}
			fillRect(0, h-blockH, w, h, 0xff)
			return mask
		}
		// Left 7/8 to 1/8 (0x2589 - 0x258f)
		if r >= 0x2589 && r <= 0x258f {
			eighths := 8 - int(r-0x2588)
			blockW := (w * eighths) / 8
			if blockW < 1 {
				blockW = 1
			}
			fillRect(0, 0, blockW, h, 0xff)
			return mask
		}
	}

	// 2. Box Drawing (U+2500 - U+257F)
	if r >= 0x2500 && r <= 0x257f {
		var up, down, left, right int
		switch r {
		case 0x2500: left, right = 1, 1 // ─
		case 0x2501: left, right = 2, 2 // ━
		case 0x2502: up, down = 1, 1    // │
		case 0x2503: up, down = 2, 2    // ┃
		case 0x2508: left, right = 1, 1 // ┄
		case 0x2509: left, right = 2, 2 // ┅
		case 0x250A: up, down = 1, 1    // ┆
		case 0x250B: up, down = 2, 2    // ┇
		case 0x250C: down, right = 1, 1 // ┌
		case 0x250D: down, right = 1, 2 // ┍
		case 0x250E: down, right = 2, 1 // ┎
		case 0x250F: down, right = 2, 2 // ┏
		case 0x2510: down, left = 1, 1  // ┐
		case 0x2511: down, left = 1, 2  // ┑
		case 0x2512: down, left = 2, 1  // ┒
		case 0x2513: down, left = 2, 2  // ┓
		case 0x2514: up, right = 1, 1   // └
		case 0x2515: up, right = 1, 2   // ┕
		case 0x2516: up, right = 2, 1   // ┖
		case 0x2517: up, right = 2, 2   // ┗
		case 0x2518: up, left = 1, 1    // ┘
		case 0x2519: up, left = 1, 2    // ┙
		case 0x251A: up, left = 2, 1    // ┚
		case 0x251B: up, left = 2, 2    // ┛
		case 0x251C: up, down, right = 1, 1, 1 // ├
		case 0x251D: up, down, right = 1, 1, 2 // ┝
		case 0x251E: up, down, right = 2, 1, 1 // ┞
		case 0x251F: up, down, right = 1, 2, 1 // ┟
		case 0x2520: up, down, right = 2, 2, 1 // ┠
		case 0x2523: up, down, right = 2, 2, 2 // ┣
		case 0x2524: up, down, left = 1, 1, 1  // ┤
		case 0x2525: up, down, left = 1, 1, 2  // ┥
		case 0x2528: up, down, left = 2, 2, 1  // ┨
		case 0x252B: up, down, left = 2, 2, 2  // ┫
		case 0x252C: down, left, right = 1, 1, 1 // ┬
		case 0x252F: down, left, right = 1, 2, 2 // ┯
		case 0x2530: down, left, right = 2, 1, 1 // ┰
		case 0x2533: down, left, right = 2, 2, 2 // ┳
		case 0x2534: up, left, right = 1, 1, 1   // ┴
		case 0x2537: up, left, right = 1, 2, 2   // ┷
		case 0x2538: up, left, right = 2, 1, 1   // ┸
		case 0x253B: up, left, right = 2, 2, 2   // ┻
		case 0x253C: up, down, left, right = 1, 1, 1, 1 // ┼
		case 0x253F: up, down, left, right = 1, 1, 2, 2 // ┿
		case 0x2542: up, down, left, right = 2, 2, 1, 1 // ╂
		case 0x254B: up, down, left, right = 2, 2, 2, 2 // ╋
		case 0x254C: left, right = 1, 1 // ╌
		case 0x254D: left, right = 2, 2 // ╍
		case 0x254E: up, down = 1, 1    // ╎
		case 0x254F: up, down = 2, 2    // ╏
		case 0x2550: left, right = 3, 3 // ═
		case 0x2551: up, down = 3, 3    // ║
		case 0x2552: down, right = 1, 3 // ╒
		case 0x2553: down, right = 3, 1 // ╓
		case 0x2554: down, right = 3, 3 // ╔
		case 0x2555: down, left = 1, 3  // ╕
		case 0x2556: down, left = 3, 1  // ╖
		case 0x2557: down, left = 3, 3  // ╗
		case 0x2558: up, right = 1, 3   // ╘
		case 0x2559: up, right = 3, 1   // ╙
		case 0x255A: up, right = 3, 3   // ╚
		case 0x255B: up, left = 1, 3    // ╛
		case 0x255C: up, left = 3, 1    // ╜
		case 0x255D: up, left = 3, 3    // ╝
		case 0x255E: up, down, right = 1, 1, 3 // ╞
		case 0x255F: up, down, right = 3, 3, 1 // ╟
		case 0x2560: up, down, right = 3, 3, 3 // ╠
		case 0x2561: up, down, left = 1, 1, 3  // ╡
		case 0x2562: up, down, left = 3, 3, 1  // ╢
		case 0x2563: up, down, left = 3, 3, 3  // ╣
		case 0x2564: down, left, right = 1, 3, 3 // ╤
		case 0x2565: down, left, right = 3, 1, 1 // ╥
		case 0x2566: down, left, right = 3, 3, 3 // ╦
		case 0x2567: up, left, right = 1, 3, 3   // ╧
		case 0x2568: up, left, right = 3, 1, 1   // ╨
		case 0x2569: up, left, right = 3, 3, 3   // ╩
		case 0x256A: up, down, left, right = 1, 1, 3, 3 // ╪
		case 0x256B: up, down, left, right = 3, 3, 1, 1 // ╫
		case 0x256C: up, down, left, right = 3, 3, 3, 3 // ╬
		case 0x256D: down, right = 1, 1 // ╭
		case 0x256E: down, left = 1, 1  // ╮
		case 0x256F: up, left = 1, 1    // ╯
		case 0x2570: up, right = 1, 1   // ╰
		case 0x2574: left = 1          // ╴
		case 0x2575: up = 1            // ╵
		case 0x2576: right = 1         // ╶
		case 0x2577: down = 1          // ╷
		case 0x2578: left = 2          // ╸
		case 0x2579: up = 2            // ╹
		case 0x257A: right = 2         // ╺
		case 0x257B: down = 2          // ╻
		default:
			return nil
		}

		midX := w / 2
		midY := h / 2

		drawLine := func(mode int, x0, y0, x1, y1 int, isHoriz bool) {
			if mode == 0 {
				return
			}
			if mode == 1 { // Light line (1px)
				if isHoriz {
					fillRect(x0, midY, x1, midY+1, 0xff)
				} else {
					fillRect(midX, y0, midX+1, y1, 0xff)
				}
			} else if mode == 2 { // Heavy line (2px)
				if isHoriz {
					fillRect(x0, midY-1, x1, midY+1, 0xff)
				} else {
					fillRect(midX-1, y0, midX+1, y1, 0xff)
				}
			} else if mode == 3 { // Double line
				if isHoriz {
					fillRect(x0, midY-2, x1, midY-1, 0xff)
					fillRect(x0, midY+1, x1, midY+2, 0xff)
				} else {
					fillRect(midX-2, y0, midX-1, y1, 0xff)
					fillRect(midX+1, y0, midX+2, y1, 0xff)
				}
			}
		}

		drawLine(up, midX, 0, midX+1, midY+1, false)
		drawLine(down, midX, midY, midX+1, h, false)
		drawLine(left, 0, midY, midX+1, midY+1, true)
		drawLine(right, midX, midY, w, midY+1, true)
		return mask
	}

	return nil
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
