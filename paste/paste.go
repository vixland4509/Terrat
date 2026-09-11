package paste

import (
	"regexp"
	"strings"
)

var ansiRegex = regexp.MustCompile(`\x1b(\[[0-9;?]*[a-zA-Z~]|\][^\x07\x1b]*(\x07|\x1b\\)|[PX^_][^\x1b]*\x1b\\|.)`)

type SanitizeResult struct {
	Sanitized string
	Warnings  []string
}

func Sanitize(raw string) SanitizeResult {
	var warnings []string

	text := strings.ReplaceAll(raw, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	if ansiRegex.MatchString(text) {
		text = ansiRegex.ReplaceAllString(text, "")
		warnings = append(warnings, "Stripped raw ANSI escape sequences (potential pastejacking attack)")
	}

	var sb strings.Builder
	hasInvisible := false
	for _, r := range text {
		if isDangerousUnicode(r) {
			hasInvisible = true
			continue
		}

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

func isDangerousUnicode(r rune) bool {
	switch r {
	case '\u200B',
		'\u200C',
		'\u200D',
		'\u2060',
		'\uFEFF',
		'\u00AD',
		'\u200E',
		'\u200F',
		'\u202A',
		'\u202B',
		'\u202C',
		'\u202D',
		'\u202E',
		'\u2066',
		'\u2067',
		'\u2068',
		'\u2069':
		return true
	}
	return false
}

func IsMultiline(text string) bool {
	return strings.Contains(text, "\n")
}

func LineCount(text string) int {
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}

func SplitLines(text string) []string {
	return strings.Split(text, "\n")
}

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

func IsURL(text string) bool {
	t := strings.TrimSpace(text)
	if strings.Contains(t, "\n") || strings.Contains(t, " ") {
		return false
	}
	lower := strings.ToLower(t)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "ftp://")
}

func HasDangerousShellChars(text string) bool {
	for _, r := range text {
		switch r {
		case '&', '?', '*', '~', '|', ';', '$', '<', '>', '(', ')', '`', '{', '}':
			return true
		}
	}
	return false
}

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

func IsLargePayload(text string) bool {
	return len(text) > 16384 || LineCount(text) > 100
}

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
		l = strings.ReplaceAll(l, "\t", "    ")
		runes := []rune(l)
		if len(runes) > maxCols {
			if maxCols > 3 {
				lines[i] = string(runes[:maxCols-3]) + "..."
			} else if maxCols > 0 {
				lines[i] = string(runes[:maxCols])
			} else {
				lines[i] = ""
			}
		} else {
			lines[i] = l
		}
	}

	return lines, totalLines
}
