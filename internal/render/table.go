// Package render turns domain results into the fixed logical table and
// safe stderr diagnostics. Output is plain text only: no color, no
// terminal control sequences, no provider payloads.
package render

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"unicode"

	"gaebokchi/internal/domain"
)

// Headings are contractual.
const (
	HeadingProvider = "AI PROVIDER"
	HeadingAccount  = "ACCOUNT"
	HeadingUsage    = "WEEKLY USAGE"
)

// SanitizeLabel makes an externally supplied label safe for the table:
// control and format characters (including bidi controls) and non-space
// whitespace become spaces, runs collapse to one space, and the result
// is trimmed. Ordinary printable Unicode is preserved.
func SanitizeLabel(s string) string {
	var b strings.Builder
	spacePending := false
	wroteAny := false
	for _, r := range s {
		switch {
		case isUnsafeRune(r):
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

func isUnsafeRune(r rune) bool {
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

// UsageCell renders the WEEKLY USAGE cell for one result.
func UsageCell(r domain.Result) string {
	if r.Status == domain.StatusOK {
		if r.UsedPercent == nil {
			// Defensive: ok without a percentage is a provider error.
			return domain.StatusProviderError.Cell()
		}
		return fmt.Sprintf("%.1f%%", *r.UsedPercent)
	}
	return r.Status.Cell()
}

// Table renders the header plus one row per result in the given order.
// The returned text ends with a newline and contains no blank lines,
// separators, titles, or trailing padding.
func Table(results []domain.Result) string {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\t%s\n", HeadingProvider, HeadingAccount, HeadingUsage)
	for _, r := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			SanitizeLabel(r.Provider.DisplayName()),
			SanitizeLabel(r.Account),
			UsageCell(r),
		)
	}
	_ = w.Flush()
	return buf.String()
}

// WriteDiagnostics emits at most one bounded line per non-clean result,
// in result order, followed by nothing else. Only safe, fixed messages
// from the domain diagnostics are used.
func WriteDiagnostics(w io.Writer, results []domain.Result) {
	for _, r := range results {
		if r.Status == domain.StatusOK {
			continue
		}
		cell := UsageCell(r)
		msg := r.Diagnostic.Message
		if msg == "" {
			msg = r.TargetID + ": " + strings.ToLower(cell)
		}
		fmt.Fprintf(w, "gaebokchi: %s: %s: %s\n", r.TargetID, cell, SanitizeLabel(msg))
	}
}
