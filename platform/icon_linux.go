//go:build !windows

package platform

import (
	"bytes"
	"encoding/binary"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func SetWindowIcon(X *xgb.Conn, win xproto.Window, iconPath string) error {
	f, err := os.Open(iconPath)
	if err != nil {
		return err
	}
	defer f.Close()

	srcImg, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	return SetWindowIconImage(X, win, srcImg)
}

func SetWindowIconBytes(X *xgb.Conn, win xproto.Window, data []byte) error {
	srcImg, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	return SetWindowIconImage(X, win, srcImg)
}

func SetWindowIconImage(X *xgb.Conn, win xproto.Window, srcImg image.Image) error {
	targetSizes := []int{128, 64, 48, 32}
	buf := new(bytes.Buffer)

	for _, size := range targetSizes {
		resized := resizeImage(srcImg, size, size)
		_ = binary.Write(buf, binary.LittleEndian, uint32(size))
		_ = binary.Write(buf, binary.LittleEndian, uint32(size))

		cx := float64(size-1) / 2.0
		cy := float64(size-1) / 2.0
		radius := float64(size) / 2.0

		bounds := resized.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			dy := float64(y) - cy
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				dx := float64(x) - cx
				dist := math.Sqrt(dx*dx + dy*dy)

				var maskAlpha float64
				if dist <= radius-1.0 {
					maskAlpha = 1.0
				} else if dist >= radius {
					maskAlpha = 0.0
				} else {
					maskAlpha = radius - dist
				}

				if maskAlpha <= 0 {
					_ = binary.Write(buf, binary.LittleEndian, uint32(0))
					continue
				}

				r, g, b, a := resized.At(x, y).RGBA()
				origA := float64(a>>8) / 255.0
				finalA := origA * maskAlpha

				r8 := uint32(byte(float64(r>>8) * maskAlpha))
				g8 := uint32(byte(float64(g>>8) * maskAlpha))
				b8 := uint32(byte(float64(b>>8) * maskAlpha))
				a8 := uint32(byte(finalA * 255.0))

				pixel := (a8 << 24) | (r8 << 16) | (g8 << 8) | b8
				_ = binary.Write(buf, binary.LittleEndian, pixel)
			}
		}
	}

	netWmIconAtom, err := internAtom(X, "_NET_WM_ICON")
	if err != nil {
		return err
	}

	cardinalAtom, err := internAtom(X, "CARDINAL")
	if err != nil {
		return err
	}

	data := buf.Bytes()
	itemCount := uint32(len(data) / 4)

	return xproto.ChangePropertyChecked(
		X,
		xproto.PropModeReplace,
		win,
		netWmIconAtom,
		cardinalAtom,
		32,
		itemCount,
		data,
	).Check()
}

func resizeImage(src image.Image, dstW, dstH int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	for y := 0; y < dstH; y++ {
		srcY := srcBounds.Min.Y + (y * srcH / dstH)
		for x := 0; x < dstW; x++ {
			srcX := srcBounds.Min.X + (x * srcW / dstW)
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}

	return dst
}

func internAtom(X *xgb.Conn, name string) (xproto.Atom, error) {
	reply, err := xproto.InternAtom(X, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, err
	}
	return reply.Atom, nil
}
