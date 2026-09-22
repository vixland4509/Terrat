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
	raw := "echo \x00hello\x07 \x08world"
	res := Sanitize(raw)
	expected := "echo hello world"
	if res.Sanitized != expected {
		t.Fatalf("expected %q, got %q", expected, res.Sanitized)
	}
}
