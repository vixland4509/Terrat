package terminal

import (
	"fmt"
	"strings"
	"testing"
)

func TestSelectionNormalizedAndIsSelected(t *testing.T) {
	term := New(80, 24)

	if term.HasSelection() {
		t.Errorf("expected no selection initially")
	}

	term.StartSelection(10, 2)
	term.UpdateSelection(20, 4)

	if !term.HasSelection() {
		t.Errorf("expected active selection after drag")
	}

	if term.IsSelected(9, 2) {
		t.Errorf("expected (9, 2) to NOT be selected")
	}
	if !term.IsSelected(10, 2) {
		t.Errorf("expected (10, 2) to be selected")
	}
	if !term.IsSelected(50, 3) {
		t.Errorf("expected (50, 3) to be selected (middle row)")
	}
	if !term.IsSelected(20, 4) {
		t.Errorf("expected (20, 4) to be selected (end cell)")
	}
	if term.IsSelected(21, 4) {
		t.Errorf("expected (21, 4) to NOT be selected")
	}

	term.StartSelection(20, 4)
	term.UpdateSelection(10, 2)

	if !term.IsSelected(10, 2) {
		t.Errorf("expected (10, 2) to be selected on reverse drag")
	}
	if !term.IsSelected(20, 4) {
		t.Errorf("expected (20, 4) to be selected on reverse drag")
	}
}

func TestWordSelectionAndExtraction(t *testing.T) {
	term := New(80, 24)

	sample := "hello world 12345"
	for i, r := range sample {
		term.lines[0][i] = Cell{Char: r}
	}

	term.SelectWord(8, 0)

	if !term.HasSelection() {
		t.Errorf("expected word selection to be active")
	}

	selected := term.GetSelectedText()
	if selected != "world" {
		t.Errorf("expected selected text to be 'world', got %q", selected)
	}

	term.SelectLine(0)
	lineSelected := term.GetSelectedText()
	if lineSelected != sample {
		t.Errorf("expected line selection to be %q, got %q", sample, lineSelected)
	}

	term.ClearSelection()
	if term.HasSelection() {
		t.Errorf("expected selection to be cleared")
	}
	if term.GetSelectedText() != "" {
		t.Errorf("expected empty string after clearing selection")
	}
}

func TestSelectAll(t *testing.T) {
	term := New(80, 24)

	sampleLine := "test all selection line"
	for i, r := range sampleLine {
		term.lines[0][i] = Cell{Char: r}
	}

	term.SelectAll()

	if !term.HasSelection() {
		t.Fatalf("expected HasSelection to be true after SelectAll")
	}

	// Verify boundary cells
	if !term.IsSelected(0, 0) {
		t.Errorf("expected (0, 0) to be selected")
	}
	if !term.IsSelected(79, 23) {
		t.Errorf("expected (79, 23) to be selected")
	}
	if !term.IsSelected(40, 12) {
		t.Errorf("expected (40, 12) to be selected")
	}

	text := term.GetSelectedText()
	if text == "" {
		t.Fatalf("expected non-empty selected text")
	}
	if text[:len(sampleLine)] != sampleLine {
		t.Errorf("expected first line to start with %q, got %q", sampleLine, text[:len(sampleLine)])
	}
}

func TestWordSelectionReverseDrag(t *testing.T) {
	term := New(80, 24)

	sample := "apple banana cherry"
	for i, r := range sample {
		term.lines[0][i] = Cell{Char: r}
	}

	// Double-click on 'banana' (starts at 6, ends at 11)
	term.SelectWord(8, 0)
	// Drag backwards to 'apple' (col 2, row 0)
	term.UpdateSelection(2, 0)

	selected := term.GetSelectedText()
	expected := "apple banana"
	if selected != expected {
		t.Fatalf("expected backwards word selection to be %q, got %q", expected, selected)
	}
}

func TestSelectAllWithScrollbackAndNoTrailingBlanks(t *testing.T) {
	term := New(80, 10)

	// Write 50 lines to create 40 lines of scrollback history and 10 lines in active grid
	for i := 0; i < 50; i++ {
		term.Write([]byte(fmt.Sprintf("Line_%02d\r\n", i)))
	}

	if term.ScrollbackLen() < 40 {
		t.Fatalf("expected at least 40 lines in scrollback, got %d", term.ScrollbackLen())
	}

	term.SelectAll()

	if !term.HasSelection() {
		t.Fatalf("expected HasSelection to be true after SelectAll")
	}

	text := term.GetSelectedText()
	lines := strings.Split(text, "\n")

	// Verify all 50 lines from scrollback to active grid are included
	if len(lines) < 50 {
		t.Fatalf("expected at least 50 lines selected, got %d lines", len(lines))
	}

	if lines[0] != "Line_00" {
		t.Fatalf("expected first line to be 'Line_00', got %q", lines[0])
	}
	if lines[49] != "Line_49" {
		t.Fatalf("expected line 49 to be 'Line_49', got %q", lines[49])
	}

	// Verify GetAllText produces the exact same complete output
	allText := term.GetAllText()
	if allText != text {
		t.Fatalf("expected GetAllText to match GetSelectedText on SelectAll")
	}

	// Verify scrolling does not cancel SelectAll mode
	term.Scroll(5)
	if !term.HasSelection() {
		t.Fatalf("expected selection to remain active after scrolling in SelectAll mode")
	}
}

func TestNoTrailingEmptyLines(t *testing.T) {
	term := New(80, 24)

	term.Write([]byte("Hello World\r\nSecond Line\r\n"))

	term.SelectAll()
	text := term.GetSelectedText()
	lines := strings.Split(text, "\n")

	if len(lines) != 2 {
		t.Fatalf("expected exactly 2 lines without 22 trailing blank lines, got %d lines: %q", len(lines), text)
	}
	if lines[0] != "Hello World" || lines[1] != "Second Line" {
		t.Fatalf("unexpected content: %v", lines)
	}

	allText := term.GetAllText()
	if allText != text {
		t.Fatalf("expected GetAllText to equal %q, got %q", text, allText)
	}
}



