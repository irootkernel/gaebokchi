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

	"gaebokchi/internal/domain"
)

// Headings are contractual.
const (
	HeadingProvider = "AI PROVIDER"
	HeadingAccount  = "ACCOUNT"
	HeadingUsage    = "WEEKLY USAGE"
)

// SanitizeLabel applies the domain display-label normalization policy
// to an externally supplied label. The policy itself lives in
// domain.NormalizeLabel so validity checks and disambiguation share the
// exact semantics; this wrapper keeps the renderer's call sites stable.
func SanitizeLabel(s string) string {
	return domain.NormalizeLabel(s)
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

// WriteDiagnostics emits at most one bounded line per result, in result
// order, followed by nothing else. Only safe, fixed messages from the
// domain diagnostics are used. A numeric row whose cleanup failed also
// emits its single safe line — the measurement stays, the incomplete
// operation is reported.
func WriteDiagnostics(w io.Writer, results []domain.Result) {
	for _, r := range results {
		if r.Status == domain.StatusOK && !r.CleanupFailed {
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
