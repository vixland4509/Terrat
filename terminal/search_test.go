package terminal

import "testing"

func TestFindRowMatchesASCII(t *testing.T) {
	matches := FindRowMatches("hello world hello", "hello")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if matches[0].StartCol != 0 || matches[0].EndCol != 4 {
		t.Fatalf("unexpected first match: %+v", matches[0])
	}
	if matches[1].StartCol != 12 || matches[1].EndCol != 16 {
		t.Fatalf("unexpected second match: %+v", matches[1])
	}
}

func TestFindRowMatchesCaseInsensitive(t *testing.T) {
	matches := FindRowMatches("Hello WORLD", "world")
	if len(matches) != 1 || matches[0].StartCol != 6 || matches[0].EndCol != 10 {
		t.Fatalf("unexpected matches: %+v", matches)
	}
}

func TestFindRowMatchesUnicode(t *testing.T) {
	// Multi-byte Persian text; rune indices must match grid columns.
	row := "سلام دنیا سلام"
	matches := FindRowMatches(row, "سلام")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d: %+v", len(matches), matches)
	}
	if matches[0].StartCol != 0 || matches[0].EndCol != 3 {
		t.Fatalf("unexpected first match: %+v", matches[0])
	}
	if matches[1].StartCol != 10 || matches[1].EndCol != 13 {
		t.Fatalf("unexpected second match: %+v", matches[1])
	}
}

func TestFindRowMatchesUnicodeQuery(t *testing.T) {
	row := "mv file-файл.txt ~"
	matches := FindRowMatches(row, "файл")
	if len(matches) != 1 || matches[0].StartCol != 8 || matches[0].EndCol != 11 {
		t.Fatalf("unexpected matches: %+v", matches)
	}
}

func TestFindRowMatchesEmptyQuery(t *testing.T) {
	if matches := FindRowMatches("anything", ""); matches != nil {
		t.Fatalf("expected nil for empty query, got %+v", matches)
	}
}

func TestFindRowMatchesInTerminal(t *testing.T) {
	term := New(80, 24)
	_, err := term.Write([]byte("echo سلام دنیا\r\n"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	matches := FindRowMatches(term.GetRowString(0), "دنیا")
	if len(matches) != 1 || matches[0].StartCol != 10 || matches[0].EndCol != 13 {
		t.Fatalf("unexpected matches: %+v", matches)
	}
}
