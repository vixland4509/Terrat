package terminal

import (
	"regexp"
	"strings"
	"testing"
)

func TestGetRowStringAndURLDetection(t *testing.T) {
	term := New(80, 24)
	output := "Visit https://github.com/vixland4509/Terrat for source code.\r\n"
	_, _ = term.Write([]byte(output))

	rowStr := term.GetRowString(0)
	if !strings.Contains(rowStr, "https://github.com/vixland4509/Terrat") {
		t.Fatalf("row string does not contain expected URL, got: %q", rowStr)
	}

	urlRegex := regexp.MustCompile(`https?://[^\s<>"'()]+`)
	matches := urlRegex.FindAllStringIndex(rowStr, -1)
	if len(matches) != 1 {
		t.Fatalf("expected 1 URL match, got %d", len(matches))
	}

	matchedURL := rowStr[matches[0][0]:matches[0][1]]
	if matchedURL != "https://github.com/vixland4509/Terrat" {
		t.Fatalf("unexpected matched URL: %s", matchedURL)
	}
}

func TestSearchMatching(t *testing.T) {
	term := New(80, 24)
	_, _ = term.Write([]byte("Hello World\r\nSearch test searching\r\n"))

	query := "search"
	lowerQuery := strings.ToLower(query)

	type match struct {
		row int
		col int
	}
	var found []match

	for y := 0; y < term.Rows(); y++ {
		rowStr := strings.ToLower(term.GetRowString(y))
		idx := 0
		for {
			pos := strings.Index(rowStr[idx:], lowerQuery)
			if pos == -1 {
				break
			}
			startCol := idx + pos
			found = append(found, match{row: y, col: startCol})
			idx = startCol + 1
			if idx >= len(rowStr) {
				break
			}
		}
	}

	if len(found) != 2 {
		t.Fatalf("expected 2 matches for 'search', got %d: %+v", len(found), found)
	}
	if found[0].row != 1 || found[0].col != 0 {
		t.Fatalf("first match expected at (1, 0), got (%d, %d)", found[0].row, found[0].col)
	}
	if found[1].row != 1 || found[1].col != 12 {
		t.Fatalf("second match expected at (1, 12), got (%d, %d)", found[1].row, found[1].col)
	}
}
