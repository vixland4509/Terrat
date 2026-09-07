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

func TestScrollKeepSelection(t *testing.T) {
	term := New(40, 10)

	for i := 0; i < 30; i++ {
		term.Write([]byte(fmt.Sprintf("Row_%02d\r\n", i)))
	}

	term.StartSelection(0, 5)
	term.UpdateSelection(10, 5)

	if !term.HasSelection() {
		t.Fatalf("expected selection to be active")
	}

	// Scroll up 2 lines keeping selection
	term.ScrollKeepSelection(2)

	if !term.HasSelection() {
		t.Fatalf("expected selection to remain active after ScrollKeepSelection")
	}

	// Selection start row should have shifted from 5 to 7
	term.mu.RLock()
	startY := term.sel.StartY
	origStartY := term.sel.OrigStartY
	term.mu.RUnlock()

	if startY != 7 || origStartY != 7 {
		t.Fatalf("expected startY to be 7, got startY=%d origStartY=%d", startY, origStartY)
	}

	// Scroll down 1 line
	term.ScrollKeepSelection(-1)
	term.mu.RLock()
	startY = term.sel.StartY
	term.mu.RUnlock()

	if startY != 6 {
		t.Fatalf("expected startY to be 6 after scrolling down, got %d", startY)
	}
}

func TestSetScrollOff(t *testing.T) {
	term := New(40, 10)

	for i := 0; i < 30; i++ {
		term.Write([]byte(fmt.Sprintf("Row_%02d\r\n", i)))
	}

	maxScroll := term.ScrollbackLen()
	if maxScroll == 0 {
		t.Fatalf("expected scrollback > 0")
	}

	term.SetScrollOff(5)
	if term.ScrollOff() != 5 {
		t.Fatalf("expected scrollOff to be 5, got %d", term.ScrollOff())
	}

	term.SetScrollOff(maxScroll + 50)
	if term.ScrollOff() != maxScroll {
		t.Fatalf("expected scrollOff to clamp to %d, got %d", maxScroll, term.ScrollOff())
	}

	term.SetScrollOff(-10)
	if term.ScrollOff() != 0 {
		t.Fatalf("expected scrollOff to clamp to 0, got %d", term.ScrollOff())
	}
}

func TestEditorAlternateScreenAndMargins(t *testing.T) {
	term := New(80, 24)

	// Shell output before launching editor
	term.Write([]byte("user@box:~$ nano myfile.txt\r\n"))
	initialScrollback := term.ScrollbackLen()

	// Nano enters alternate screen and restricts scroll margin
	term.Write([]byte("\x1b[?1049h"))
	if !term.IsAlt() {
		t.Fatalf("expected alt screen to be active")
	}

	// Nano sets top/bottom margins (e.g. lines 2 to 22)
	term.Write([]byte("\x1b[2;22r"))
	// Nano writes content and scrolls within its window
	for i := 0; i < 50; i++ {
		term.Write([]byte(fmt.Sprintf("editor content line %d\r\n", i)))
	}

	// In alternate screen, scrollback must NOT grow!
	if term.ScrollbackLen() != initialScrollback {
		t.Fatalf("scrollback increased inside alternate screen: before %d, now %d", initialScrollback, term.ScrollbackLen())
	}

	// Nano exits: leaves alternate screen
	term.Write([]byte("\x1b[?1049l"))
	if term.IsAlt() {
		t.Fatalf("expected alt screen to be inactive after exit")
	}

	// Margins must be restored to full terminal (0 to 23), not trapped in nano's 2..22
	if term.scrollTop != 0 || term.scrollBottom != 23 {
		t.Fatalf("scroll margins were not reset upon exiting alt screen: top=%d, bottom=%d", term.scrollTop, term.scrollBottom)
	}

	// Subsequent shell outputs should not push unexpected garbage or be clipped
	term.Write([]byte("user@box:~$ echo hello\r\n"))
}

func TestEditorCSICommands(t *testing.T) {
	term := New(80, 24)

	// Test VPA (CSI d)
	term.Write([]byte("\x1b[10d"))
	_, y, _ := term.Cursor()
	if y != 9 {
		t.Fatalf("expected cursor Y to be 9 after CSI 10d, got %d", y)
	}

	// Test ICH (CSI @) and ECH (CSI X)
	term.Write([]byte("\x1b[1;1HABCDEF"))
	term.Write([]byte("\x1b[1;3H\x1b[2@")) // Insert 2 spaces at col 3 (0-indexed 2)
	rowStr := term.GetRowString(0)
	if !strings.HasPrefix(rowStr, "AB  CD") {
		t.Fatalf("expected row to start with 'AB  CD' after ICH, got %q", rowStr[:10])
	}

	term.Write([]byte("\x1b[1;1H\x1b[2X")) // Erase 2 characters at col 1
	rowStr = term.GetRowString(0)
	if !strings.HasPrefix(rowStr, "    CD") {
		t.Fatalf("expected row to start with '    CD' after ECH, got %q", rowStr[:10])
	}

	// Test SU (CSI S) and SD (CSI T)
	term.Write([]byte("\x1b[1;1HTOP_LINE\r\nSECOND_LINE"))
	term.Write([]byte("\x1b[1S")) // Scroll up 1
	r0 := term.GetRowString(0)
	if !strings.HasPrefix(r0, "SECOND_LINE") {
		t.Fatalf("expected row 0 to be 'SECOND_LINE' after SU, got %q", r0[:15])
	}

	term.Write([]byte("\x1b[1T")) // Scroll down 1
	r1 := term.GetRowString(1)
	if !strings.HasPrefix(r1, "SECOND_LINE") {
		t.Fatalf("expected row 1 to be 'SECOND_LINE' after SD, got %q", r1[:15])
	}
}

