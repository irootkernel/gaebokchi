package domain

import (
	"strings"
	"unicode"
)

// NormalizeLabel is the single display-label normalization policy.
// Control and format characters (including bidi controls) and non-space
// whitespace become spaces, runs collapse to one space, and the result
// is trimmed. Ordinary printable Unicode is preserved. Validity checks
// (blank-alias rejection, usable-email decisions) and duplicate-label
// disambiguation must apply this policy BEFORE deciding, so that what
// is validated and compared is the label the table will actually show.
func NormalizeLabel(s string) string {
	var b strings.Builder
	spacePending := false
	wroteAny := false
	for _, r := range s {
		switch {
		case isUnsafeLabelRune(r):
			spacePending = wroteAny
		case r == ' ':
			spacePending = wroteAny
		default:
			if spacePending {
				b.WriteByte(' ')
				spacePending = false
			}
			b.WriteRune(r)
			wroteAny = true
		}
	}
	return b.String()
}

func isUnsafeLabelRune(r rune) bool {
	if r == unicode.ReplacementChar {
		// Invalid UTF-8 bytes surface as U+FFFD; they cannot stay in a
		// rendered cell.
		return true
	}
	if r == '\t' || r == '\n' || r == '\r' || r == 0x7f {
		return true
	}
	if r < 0x20 || (r >= 0x80 && r <= 0x9f) {
		return true
	}
	if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
		return true
	}
	// Non-space whitespace (NBSP, line separators, bidi spaces).
	return r == 0xa0 || (r >= 0x2000 && r <= 0x200f) || (r >= 0x2028 && r <= 0x202f)
}
