package domain

import (
	"strings"
	"unicode"
)

// NormalizeLabel is the single display-label normalization policy.
// All Unicode whitespace (ASCII spaces and tabs, NBSP, the ideographic
// space U+3000, and every other White_Space rune), control and format
// characters (including bidi controls), and invalid-UTF-8 replacement
// runes become spaces, runs collapse to one space, and the result is
// trimmed. Ordinary printable Unicode — including Korean and other CJK
// text — is preserved. Validity checks (blank-alias rejection,
// usable-email decisions) and duplicate-label disambiguation must apply
// this policy BEFORE deciding, so that what is validated and compared
// is the label the table will actually show.
func NormalizeLabel(s string) string {
	var b strings.Builder
	spacePending := false
	wroteAny := false
	for _, r := range s {
		if isUnsafeLabelRune(r) {
			spacePending = wroteAny
			continue
		}
		if spacePending {
			b.WriteByte(' ')
			spacePending = false
		}
		b.WriteRune(r)
		wroteAny = true
	}
	return b.String()
}

func isUnsafeLabelRune(r rune) bool {
	if r == unicode.ReplacementChar {
		// Invalid UTF-8 bytes surface as U+FFFD; they cannot stay in a
		// rendered cell.
		return true
	}
	if unicode.IsSpace(r) {
		// The complete Unicode whitespace predicate: covers ASCII
		// whitespace, NBSP, U+1680, U+2000-U+200A, U+2028-U+202F,
		// U+205F, U+3000, and the rest of the White_Space property.
		return true
	}
	if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
		return true
	}
	return unicode.IsControl(r) || unicode.Is(unicode.Cf, r)
}
