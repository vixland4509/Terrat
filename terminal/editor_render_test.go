package terminal

import (
	"strings"
	"testing"
	"time"

	"terrat/pty"
)

func TestLeadingSemicolonCSI(t *testing.T) {
	term := New(80, 24)
	term.Write([]byte("\x1b[;10H"))
	x, y, _ := term.Cursor()
	if y != 0 || x != 9 {
		t.Fatalf("expected cursor at (9, 0), got (%d, %d)", x, y)
	}
}

func TestNanoRendering(t *testing.T) {
	cols, rows := 80, 24
	pm, err := pty.Start(uint16(cols), uint16(rows), 800, 600, "nano")
	if err != nil {
		t.Skipf("nano not available: %v", err)
	}
	defer pm.Close()

	term := New(cols, rows)
	go func() {
		for resp := range term.ResponseChan {
			_, _ = pm.Write(resp)
		}
	}()

	buf := make([]byte, 4096)
	go func() {
		for {
			n, err := pm.Read(buf)
			if n > 0 {
				_, _ = term.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	time.Sleep(400 * time.Millisecond)

	row0 := ""
	for x := 0; x < cols; x++ {
		c := term.GetCell(x, 0).Char
		if c == 0 {
			row0 += " "
		} else {
			row0 += string(c)
		}
	}

	row23 := ""
	for x := 0; x < cols; x++ {
		c := term.GetCell(x, 23).Char
		if c == 0 {
			row23 += " "
		} else {
			row23 += string(c)
		}
	}

	if !strings.Contains(row0, "nano") {
		t.Errorf("Row 0 must contain nano header, got: %q", row0)
	}
	if strings.Contains(row0, "Exit") {
		t.Errorf("Row 0 must NOT contain Exit shortcut, got: %q", row0)
	}
	if !strings.Contains(row23, "Exit") {
		t.Errorf("Row 23 must contain Exit shortcut, got: %q", row23)
	}

	_, _ = pm.Write([]byte("\x18"))
	time.Sleep(100 * time.Millisecond)
}

func TestMicroRendering(t *testing.T) {
	cols, rows := 80, 24
	pm, err := pty.Start(uint16(cols), uint16(rows), 800, 600, "micro")
	if err != nil {
		t.Skipf("micro not available: %v", err)
	}
	defer pm.Close()

	term := New(cols, rows)
	go func() {
		for resp := range term.ResponseChan {
			_, _ = pm.Write(resp)
		}
	}()

	buf := make([]byte, 4096)
	go func() {
		for {
			n, err := pm.Read(buf)
			if n > 0 {
				_, _ = term.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	time.Sleep(400 * time.Millisecond)

	row0 := ""
	for x := 0; x < cols; x++ {
		c := term.GetCell(x, 0).Char
		if c == 0 {
			row0 += " "
		} else {
			row0 += string(c)
		}
	}

	row22 := ""
	for x := 0; x < cols; x++ {
		c := term.GetCell(x, 22).Char
		if c == 0 {
			row22 += " "
		} else {
			row22 += string(c)
		}
	}

	if !strings.Contains(row0, "1") {
		t.Errorf("Micro row 0 must contain line number 1, got: %q", row0)
	}
	if !strings.Contains(row22, "Alt-g") {
		t.Errorf("Micro row 22 must contain status bar, got: %q", row22)
	}

	_, _ = pm.Write([]byte("\x11"))
	time.Sleep(100 * time.Millisecond)
}
