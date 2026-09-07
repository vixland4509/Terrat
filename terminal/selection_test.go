package terminal

import (
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

