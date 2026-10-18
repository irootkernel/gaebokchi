// Package render turns domain results into the interactive terminal view,
// the plain table, the JSON snapshot, and safe stderr diagnostics. The table,
// JSON, and diagnostics contain no color, terminal controls, or raw provider
// payloads; only the selected terminal view emits its documented ANSI controls.
package render

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"
	"text/tabwriter"

	"gaebokchi/internal/domain"
)

// Headings are contractual.
const (
	HeadingProvider      = "AI PROVIDER"
	HeadingAccount       = "ACCOUNT"
	HeadingUsage         = "WEEKLY USAGE"
	HeadingFiveHourUsage = "5H USAGE"
)

// SanitizeLabel applies the domain display-label normalization policy
// to an externally supplied label. The policy itself lives in
// domain.NormalizeLabel so validity checks and disambiguation share the
// exact semantics; this wrapper keeps the renderer's call sites stable.
func SanitizeLabel(s string) string {
	return domain.NormalizeLabel(s)
}

// UsageCell renders the usage cell for one result.
func UsageCell(r domain.Result) string {
	if r.Status == domain.StatusOK {
		if r.UsedPercent == nil {
			// Defensive: ok without a percentage is a provider error.
			return domain.StatusProviderError.Cell()
		}
		percent := domain.NormalizePercentSignedZero(*r.UsedPercent)
		if percent == math.Trunc(percent) {
			return fmt.Sprintf("%.0f%%", percent)
		}
		return fmt.Sprintf("%.1f%%", percent)
	}
	return r.Status.Cell()
}

// Table renders the weekly section and, when 5-hour targets are present,
// the 5-hour section below it.
func Table(results []domain.Result) string {
	var weekly, fiveHour []domain.Result
	for _, r := range results {
		if r.Provider.IsFiveHour() {
			fiveHour = append(fiveHour, r)
		} else {
			weekly = append(weekly, r)
		}
	}
	renderSection := func(heading string, rows []domain.Result) string {
		var buf bytes.Buffer
		w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		// The tabwriter writes only to an in-memory bytes.Buffer, which cannot fail.
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", HeadingProvider, HeadingAccount, heading)
		for _, r := range rows {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
				SanitizeLabel(r.Provider.DisplayName()),
				SanitizeLabel(r.Account),
				UsageCell(r),
			)
		}
		_ = w.Flush()
		return buf.String()
	}
	if len(fiveHour) == 0 {
		return renderSection(HeadingUsage, weekly)
	}
	if len(weekly) == 0 {
		return renderSection(HeadingFiveHourUsage, fiveHour)
	}
	return renderSection(HeadingUsage, weekly) + "\n" + renderSection(HeadingFiveHourUsage, fiveHour)
}

// WriteDiagnostics emits at most one bounded line per result, in result
// order, followed by nothing else. Only safe, fixed messages from the
// domain diagnostics are used. A numeric row whose cleanup failed also
// emits its single safe line — the measurement stays, the incomplete
// operation is reported.
func WriteDiagnostics(w io.Writer, results []domain.Result) {
	// Diagnostics are best effort; callers have no alternative channel if
	// stderr itself is unwritable.
	for _, r := range results {
		if r.Status == domain.StatusOK && !r.CleanupFailed {
			continue
		}
		cell := UsageCell(r)
		msg := r.Diagnostic.Message
		if msg == "" {
			msg = r.TargetID + ": " + strings.ToLower(cell)
		}
		_, _ = fmt.Fprintf(w, "gaebokchi: %s: %s: %s\n", r.TargetID, cell, SanitizeLabel(msg))
	}
}
