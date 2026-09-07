package terminal

import (
	"fmt"
	"strings"
	"testing"
)

func TestScrollOffsetClamping(t *testing.T) {
	term := New(80, 24)

	if term.ScrollOff() != 0 {
		t.Fatalf("expected initial scrollOff to be 0, got %d", term.ScrollOff())
	}

	term.Scroll(5)
	if term.ScrollOff() != 0 {
		t.Fatalf("expected scrollOff to remain 0 when scrollback is empty, got %d", term.ScrollOff())
	}

	for i := 0; i < 50; i++ {
		term.Write([]byte(fmt.Sprintf("Line %02d\r\n", i)))
	}

	sbLen := term.ScrollbackLen()
	if sbLen == 0 {
		t.Fatalf("expected scrollback to have entries after 50 lines, got 0")
	}

	term.Scroll(5)
	if term.ScrollOff() != 5 {
		t.Fatalf("expected scrollOff to be 5, got %d", term.ScrollOff())
	}

	term.Scroll(sbLen + 100)
	if term.ScrollOff() != sbLen {
		t.Fatalf("expected scrollOff to clamp at %d, got %d", sbLen, term.ScrollOff())
	}

	term.ScrollToTop()
	if term.ScrollOff() != sbLen {
		t.Fatalf("expected ScrollToTop to set scrollOff to %d, got %d", sbLen, term.ScrollOff())
	}

	term.Scroll(-10)
	if term.ScrollOff() != sbLen-10 {
		t.Fatalf("expected scrollOff to be %d, got %d", sbLen-10, term.ScrollOff())
	}

	term.Scroll(-sbLen * 2)
	if term.ScrollOff() != 0 {
		t.Fatalf("expected scrollOff to clamp at 0, got %d", term.ScrollOff())
	}

	term.ScrollToTop()
	term.ResetScroll()
	if term.ScrollOff() != 0 {
		t.Fatalf("expected ResetScroll to set scrollOff to 0, got %d", term.ScrollOff())
	}
}

func TestSeamlessScrollbackGridBoundary(t *testing.T) {
	term := New(20, 10)

	for i := 0; i < 20; i++ {
		term.Write([]byte(fmt.Sprintf("L%02d\r\n", i)))
	}

	term.ResetScroll()
	term.Scroll(1)

	row0 := strings.TrimSpace(term.GetRowString(0))
	row1 := strings.TrimSpace(term.GetRowString(1))

	var num0, num1 int
	if n, _ := fmt.Sscanf(row0, "L%02d", &num0); n == 1 {
		if n1, _ := fmt.Sscanf(row1, "L%02d", &num1); n1 == 1 {
			if num1 != num0+1 {
				t.Fatalf("seam broken: row 0 is %q (%d), but row 1 is %q (%d). Expected consecutive lines without skipping!", row0, num0, row1, num1)
			}
		}
	}

	term.Scroll(2)
	for y := 0; y < 9; y++ {
		ra := strings.TrimSpace(term.GetRowString(y))
		rb := strings.TrimSpace(term.GetRowString(y + 1))
		var na, nb int
		if n, _ := fmt.Sscanf(ra, "L%02d", &na); n == 1 {
			if n2, _ := fmt.Sscanf(rb, "L%02d", &nb); n2 == 1 {
				if nb != na+1 {
					t.Fatalf("gap detected between visible row %d (%q) and row %d (%q)", y, ra, y+1, rb)
				}
			}
		}
	}
}

func TestAltScreenScrollLock(t *testing.T) {
	term := New(80, 24)

	for i := 0; i < 30; i++ {
		term.Write([]byte(fmt.Sprintf("Line %d\r\n", i)))
	}

	if term.IsAlt() {
		t.Fatalf("expected IsAlt to be false initially")
	}

	term.Write([]byte("\x1b[?1049h"))
	if !term.IsAlt() {
		t.Fatalf("expected IsAlt to be true after DECSET 1049")
	}

	term.Scroll(5)
	if term.ScrollOff() != 0 {
		t.Fatalf("expected scrollOff to remain 0 in alt screen, got %d", term.ScrollOff())
	}

	term.Write([]byte("\x1b[?1049l"))
	if term.IsAlt() {
		t.Fatalf("expected IsAlt to be false after DECRST 1049")
	}

	term.Scroll(3)
	if term.ScrollOff() != 3 {
		t.Fatalf("expected scrollOff to be 3 after exiting alt screen, got %d", term.ScrollOff())
	}
}

func TestScrollbackSelection(t *testing.T) {
	term := New(40, 10)

	for i := 0; i < 20; i++ {
		term.Write([]byte(fmt.Sprintf("HIST_%02d_VALUE\r\n", i)))
	}

	term.Scroll(5)

	term.SelectLine(0)
	txt := term.GetSelectedText()
	if !strings.Contains(txt, "HIST_") {
		t.Fatalf("expected selected text to contain HIST_, got %q", txt)
	}
}
