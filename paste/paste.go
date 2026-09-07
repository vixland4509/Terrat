package paste

import (
	"regexp"
	"strings"
	"unicode"
)

// ansiRegex matches ANSI escape sequences (CSI, OSC, etc.)
var ansiRegex = regexp.MustCompile(`\x1b(\[[0-9;?]*[a-zA-Z~]|\][^\x07\x1b]*(\x07|\x1b\\)|[PX^_][^\x1b]*\x1b\\|.)`)

// SanitizeResult contains the sanitized text and any safety warnings triggered.
type SanitizeResult struct {
	Sanitized string
	Warnings  []string
}

// Sanitize cleans pasted content from terminal injection vectors (pastejacking),
// dangerous ANSI escape sequences, raw control codes, and invisible Unicode overrides.
func Sanitize(raw string) SanitizeResult {
	var warnings []string

	// 1. Normalize line endings (\r\n -> \n, lone \r -> \n)
	text := strings.ReplaceAll(raw, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 2. Strip ANSI escape sequences (Pastejacking mitigation)
	if ansiRegex.MatchString(text) {
		text = ansiRegex.ReplaceAllString(text, "")
		warnings = append(warnings, "Stripped raw ANSI escape sequences (potential pastejacking attack)")
	}

	// 3. Strip invisible unicode characters and Bidi overrides (Trojan Source mitigation)
	var sb strings.Builder
	hasInvisible := false
	for _, r := range text {
		// Detect invisible zero-width characters and bidi overrides
		if isDangerousUnicode(r) {
			hasInvisible = true
			continue
		}

		// Detect dangerous ASCII control characters (< 0x20 except \t and \n, plus 0x7F DEL)
		if (r < 0x20 && r != '\t' && r != '\n') || r == 0x7f {
			continue
		}

		sb.WriteRune(r)
	}

	if hasInvisible {
		warnings = append(warnings, "Stripped invisible zero-width and bidirectional override characters")
	}

	return SanitizeResult{
		Sanitized: sb.String(),
		Warnings:  warnings,
	}
}

// isDangerousUnicode returns true for zero-width spaces, joiners, formatting marks,
// and bidi directional overrides that can visually disguise malicious commands.
func isDangerousUnicode(r rune) bool {
	switch r {
	case '\u200B', // Zero Width Space
		'\u200C', // Zero Width Non-Joiner
		'\u200D', // Zero Width Joiner
		'\u2060', // Word Joiner
		'\uFEFF', // Zero Width No-Break Space (BOM)
		'\u00AD', // Soft Hyphen
		'\u200E', // Left-to-Right Mark
		'\u200F', // Right-to-Left Mark
		'\u202A', // Left-to-Right Embedding
		'\u202B', // Right-to-Left Embedding
		'\u202C', // Pop Directional Formatting
		'\u202D', // Left-to-Right Override
		'\u202E', // Right-to-Left Override
		'\u2066', // Left-to-Right Isolate
		'\u2067', // Right-to-Left Isolate
		'\u2068', // First Strong Isolate
		'\u2069': // Pop Directional Isolate
		return true
	}
	return false
}

// IsMultiline reports whether text contains one or more newline characters.
func IsMultiline(text string) bool {
	return strings.Contains(text, "\n")
}

// LineCount returns the number of lines in text.
func LineCount(text string) int {
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}

// SplitLines returns individual lines from text.
func SplitLines(text string) []string {
	return strings.Split(text, "\n")
}

// FlattenToSingleLine converts multi-line text into a single-line command
// using semicolons or logical connectors where appropriate.
func FlattenToSingleLine(text string) string {
	rawLines := SplitLines(text)
	var cleaned []string
	for _, l := range rawLines {
		t := strings.TrimSpace(l)
		if t != "" {
			cleaned = append(cleaned, t)
		}
	}

	if len(cleaned) == 0 {
		return ""
	}

	var b strings.Builder
	for i, line := range cleaned {
		if i > 0 {
			prev := cleaned[i-1]
			if strings.HasSuffix(prev, "\\") || strings.HasSuffix(prev, "&&") || strings.HasSuffix(prev, "||") || strings.HasSuffix(prev, "|") || strings.HasSuffix(prev, ";") {
				b.WriteString(" ")
			} else {
				b.WriteString("; ")
			}
		}
		b.WriteString(line)
	}
	return b.String()
}

// IsURL reports whether trimmed text appears to be a web URL without newlines.
func IsURL(text string) bool {
	t := strings.TrimSpace(text)
	if strings.Contains(t, "\n") || strings.Contains(t, " ") {
		return false
	}
	lower := strings.ToLower(t)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "ftp://")
}

// HasDangerousShellChars reports whether text contains unquoted shell metacharacters
// like &, ?, *, ~, |, ;, $, <, >, (, ).
func HasDangerousShellChars(text string) bool {
	for _, r := range text {
		switch r {
		case '&', '?', '*', '~', '|', ';', '$', '<', '>', '(', ')', '`', '{', '}':
			return true
		}
	}
	return false
}

// QuoteURL encloses text in double quotes if it is a URL with dangerous shell characters.
func QuoteURL(text string) string {
	t := strings.TrimSpace(text)
	if strings.HasPrefix(t, "\"") && strings.HasSuffix(t, "\"") {
		return t
	}
	if strings.HasPrefix(t, "'") && strings.HasSuffix(t, "'") {
		return t
	}
	return "\"" + t + "\""
}

// IsLargePayload reports whether text exceeds standard safe thresholds for interactive paste.
func IsLargePayload(text string) bool {
	return len(text) > 16384 || LineCount(text) > 100
}

// TruncatePreview returns up to maxLines lines of text, truncating long individual lines to maxCols.
func TruncatePreview(text string, maxLines, maxCols int) (lines []string, totalLines int) {
	all := SplitLines(text)
	totalLines = len(all)

	limit := maxLines
	if limit > totalLines {
		limit = totalLines
	}

	lines = make([]string, limit)
	for i := 0; i < limit; i++ {
		l := all[i]
		// Replace tabs with 4 spaces for display
		l = strings.ReplaceAll(l, "\t", "    ")
		if len(l) > maxCols {
			lines[i] = l[:maxCols-3] + "..."
		} else {
			lines[i] = l
		}
	}

	return lines, totalLines
}

// ContainsNonPrintable checks if string contains control characters.
func ContainsNonPrintable(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}
	return false
}
