package paste

import (
	"strings"
	"testing"
)

func TestSanitizeNormal(t *testing.T) {
	input := "echo hello world\nls -la"
	res := Sanitize(input)
	if res.Sanitized != input {
		t.Fatalf("expected %q, got %q", input, res.Sanitized)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("expected 0 warnings, got %v", res.Warnings)
	}
}

func TestSanitizeCRLF(t *testing.T) {
	input := "line1\r\nline2\rline3\n"
	res := Sanitize(input)
	expected := "line1\nline2\nline3\n"
	if res.Sanitized != expected {
		t.Fatalf("expected %q, got %q", expected, res.Sanitized)
	}
}

func TestSanitizeANSIEscapeSequences(t *testing.T) {
	// Pastejacking attempt: visually hiding commands using cursor movement / erase sequences
	malicious := "git clone https://example.com\x1b[2K\rrm -rf /\n"
	res := Sanitize(malicious)

	if strings.Contains(res.Sanitized, "\x1b") {
		t.Fatalf("expected ANSI escape to be stripped, got %q", res.Sanitized)
	}
	if len(res.Warnings) == 0 {
		t.Fatalf("expected warning about ANSI escapes, got none")
	}
}

func TestSanitizeInvisibleUnicode(t *testing.T) {
	// Trojan source / zero-width space attack: s[ZWSP]udo
	malicious := "s\u200Budo rm -rf /\u202Ereversed"
	res := Sanitize(malicious)

	if strings.Contains(res.Sanitized, "\u200B") || strings.Contains(res.Sanitized, "\u202E") {
		t.Fatalf("expected invisible / bidi chars to be stripped, got %q", res.Sanitized)
	}
	if len(res.Warnings) == 0 {
		t.Fatalf("expected warning about invisible chars, got none")
	}
}

func TestSanitizeControlCharacters(t *testing.T) {
	// NUL and BEL and BS characters
	raw := "echo \x00hello\x07 \x08world"
	res := Sanitize(raw)
	expected := "echo hello world"
	if res.Sanitized != expected {
		t.Fatalf("expected %q, got %q", expected, res.Sanitized)
	}
}

func TestIsMultiline(t *testing.T) {
	if IsMultiline("single line command") {
		t.Fatalf("expected false for single line")
	}
	if !IsMultiline("line1\nline2") {
		t.Fatalf("expected true for multiline")
	}
}

func TestFlattenToSingleLine(t *testing.T) {
	multiline := "mkdir -p mydir\ncd mydir\ngit init"
	flat := FlattenToSingleLine(multiline)
	expected := "mkdir -p mydir; cd mydir; git init"
	if flat != expected {
		t.Fatalf("expected %q, got %q", expected, flat)
	}

	// Line ending with backslash
	multilineWithSlash := "git commit \\\n  -m \"initial commit\""
	flatSlash := FlattenToSingleLine(multilineWithSlash)
	expectedSlash := "git commit \\ -m \"initial commit\""
	if flatSlash != expectedSlash {
		t.Fatalf("expected %q, got %q", expectedSlash, flatSlash)
	}
}

func TestURLDetectionAndQuoting(t *testing.T) {
	url := "https://example.com/api?user=1&token=secret"
	if !IsURL(url) {
		t.Fatalf("expected IsURL to be true")
	}
	if !HasDangerousShellChars(url) {
		t.Fatalf("expected HasDangerousShellChars to be true for '&' and '?'")
	}

	quoted := QuoteURL(url)
	expected := "\"https://example.com/api?user=1&token=secret\""
	if quoted != expected {
		t.Fatalf("expected %q, got %q", expected, quoted)
	}

	// Non-URL
	if IsURL("curl -v https://example.com") {
		t.Fatalf("expected IsURL to be false for multi-word command")
	}
}

func TestTruncatePreview(t *testing.T) {
	input := "line1\nline2\nline3\nline4\nline5"
	preview, total := TruncatePreview(input, 3, 50)
	if total != 5 {
		t.Fatalf("expected total 5, got %d", total)
	}
	if len(preview) != 3 {
		t.Fatalf("expected preview length 3, got %d", len(preview))
	}
	if preview[0] != "line1" || preview[2] != "line3" {
		t.Fatalf("unexpected preview content: %v", preview)
	}
}

func TestTruncatePreviewUTF8(t *testing.T) {
	input := "سلام دنیا و جهان بسیار زیبا\nHello World"
	preview, total := TruncatePreview(input, 2, 10)
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	expectedFirst := string([]rune("سلام دنیا و جهان بسیار زیبا")[:7]) + "..."
	if preview[0] != expectedFirst {
		t.Fatalf("expected %q, got %q", expectedFirst, preview[0])
	}
	if preview[1] != "Hello W..." {
		t.Fatalf("expected 'Hello W...', got %q", preview[1])
	}
}
