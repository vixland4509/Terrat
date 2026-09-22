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
