package render

import (
	"bytes"
	ginkgo "github.com/onsi/ginkgo/v2"
	"math"
	"strings"
	"time"

	"gaebokchi/internal/domain"
)

func pct(v float64) *float64 { return &v }

var _ = ginkgo.It("TestTableGoldenMixedStatuses", func() {
	t := testT{ginkgo.GinkgoT()}
	results := []domain.Result{
		{Provider: domain.ProviderCodex, Account: "account1@example.com", Status: domain.StatusOK, UsedPercent: pct(37)},
		{Provider: domain.ProviderCodex, Account: ".codex-hsy", Status: domain.StatusAuthRequired},
		{Provider: domain.ProviderGrok, Account: "default", Status: domain.StatusNotApplicableMonthly},
	}
	got := Table(results)
	want := "AI PROVIDER  ACCOUNT               WEEKLY USAGE\n" +
		"ChatGPT      account1@example.com  37%\n" +
		"ChatGPT      .codex-hsy            AUTH REQUIRED\n" +
		"Grok         default               N/A (monthly)\n"
	if got != want {
		t.Fatalf("table mismatch:\n got:\n%s\nwant:\n%s", got, want)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatal("missing final newline")
	}
	if strings.Contains(got, "\x1b") || strings.Contains(got, "\r") {
		t.Fatal("control sequences in output")
	}
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("logical lines = %d, want 4", len(lines))
	}
	for _, line := range lines {
		if line == "" {
			t.Fatal("blank line emitted")
		}
		if strings.Contains(line, "|") || strings.HasPrefix(line, "-") {
			t.Fatal("separator or border emitted")
		}
	}
})

var _ = ginkgo.It("TestTableAllNumeric", func() {
	t := testT{ginkgo.GinkgoT()}
	results := []domain.Result{
		{Provider: domain.ProviderCodex, Account: "a@example.com", Status: domain.StatusOK, UsedPercent: pct(0)},
		{Provider: domain.ProviderCodex, Account: "b@example.com", Status: domain.StatusOK, UsedPercent: pct(12.25)},
		{Provider: domain.ProviderGrok, Account: "default", Status: domain.StatusOK, UsedPercent: pct(54)},
	}
	got := Table(results)
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != 4 || !strings.HasSuffix(lines[1], "  0%") ||
		!strings.HasSuffix(lines[2], "  12.2%") || !strings.HasSuffix(lines[3], "  54%") {
		t.Fatalf("percentage formatting broken:\n%s", got)
	}
})

var _ = ginkgo.Describe("TestUsageCellAdaptivePrecision", func() {
	cases := map[float64]string{
		0: "0%", 37: "37%", 100: "100%", 37.06: "37.1%", 37.04: "37.0%",
		99.94: "99.9%", 99.96: "100.0%", 5.5: "5.5%",
	}
	for v, want := range cases {
		ginkgo.It(want, func() {
			t := testT{ginkgo.GinkgoT()}
			got := UsageCell(domain.Result{Status: domain.StatusOK, UsedPercent: pct(v)})
			if got != want {
				t.Fatalf("UsageCell(%v) = %q, want %q", v, got, want)
			}
		})
	}
	ginkgo.It("normalizes signed zero", func() {
		t := testT{ginkgo.GinkgoT()}
		got := UsageCell(domain.Result{Status: domain.StatusOK, UsedPercent: pct(math.Copysign(0, -1))})
		if got != "0%" {
			t.Fatalf("signed zero rendered as %q", got)
		}
	})
})

var _ = ginkgo.Describe("TestSanitizeLabel", func() {
	cases := map[string]string{
		"plain":                  "plain",
		"injected\tcol":          "injected col",
		"line1\nline2":           "line1 line2",
		"esc\x1b[31mred":         "esc [31mred",
		"bidi\u202ahidden\u202c": "bidi hidden",
		"nbsp a b":               "nbsp a b",
		"collapse    spaces":     "collapse spaces",
		"  trimmed  ":            "trimmed",
		"unicode 개복치 🐡 stays":    "unicode 개복치 🐡 stays",
		"del\x7fchar":            "del char",
		"c1\x9bcontrol":          "c1 control",
	}
	for in, want := range cases {
		ginkgo.It(in, func() {
			t := testT{ginkgo.GinkgoT()}
			if got := SanitizeLabel(in); got != want {
				t.Fatalf("SanitizeLabel(%q) = %q, want %q", in, got, want)
			}
		})
	}
})

var _ = ginkgo.It("TestSanitizedLabelCannotAddColumnsOrRows", func() {
	t := testT{ginkgo.GinkgoT()}
	results := []domain.Result{
		{Provider: domain.ProviderGrok, Account: "a\tb\nc\x1b[2J", Status: domain.StatusOK, UsedPercent: pct(1)},
	}
	got := Table(results)
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("line injection: %d lines", len(lines))
	}
	if strings.Count(lines[1], "\t") != 0 {
		t.Fatalf("tab survived sanitization: %q", lines[1])
	}
})

var _ = ginkgo.It("TestDiagnosticsOnlyForNonCleanRows", func() {
	t := testT{ginkgo.GinkgoT()}
	results := []domain.Result{
		{TargetID: "codex:0", Provider: domain.ProviderCodex, Account: "a@example.com", Status: domain.StatusOK, UsedPercent: pct(1)},
		{TargetID: "codex:1", Provider: domain.ProviderCodex, Account: ".codex-hsy", Status: domain.StatusAuthRequired,
			Diagnostic: domain.Diagnostic{Code: "auth_required", Message: "codex:1: authenticate using the provider CLI outside gaebokchi"}},
		{TargetID: "grok:0", Provider: domain.ProviderGrok, Account: "default", Status: domain.StatusNotApplicableMonthly,
			Diagnostic: domain.Diagnostic{Code: "not_applicable/monthly", Message: "grok:0: grok account reports a monthly period, not weekly"}},
	}
	var buf bytes.Buffer
	WriteDiagnostics(&buf, results)
	out := buf.String()
	if strings.Count(out, "\n") != 2 {
		t.Fatalf("diagnostic lines = %d, want 2:\n%s", strings.Count(out, "\n"), out)
	}
	if !strings.HasPrefix(out, "gaebokchi: codex:1: AUTH REQUIRED:") {
		t.Fatalf("unexpected diagnostic format: %q", out)
	}
	if !strings.Contains(out, "grok:0: N/A (monthly):") {
		t.Fatalf("monthly diagnostic missing: %q", out)
	}
	if strings.Contains(out, "codex:0") {
		t.Fatal("successful row produced a diagnostic")
	}
})

var _ = ginkgo.It("TestResetAndProvenanceNeverRendered", func() {
	t := testT{ginkgo.GinkgoT()}
	reset := time.Unix(1790411774, 0)
	results := []domain.Result{{
		Provider: domain.ProviderCodex, Account: "a@example.com", Status: domain.StatusOK, UsedPercent: pct(7),
		QuotaID: "codex", ResetAt: &reset, Source: domain.SourceCodexAppServer, Period: domain.PeriodWeekly,
	}}
	got := Table(results)
	if strings.Contains(got, "codex") || strings.Contains(got, "app_server") || strings.Contains(got, "2026") {
		t.Fatalf("window provenance leaked into the table:\n%s", got)
	}
})

var _ = ginkgo.It("TestTableAgyTwoRows", func() {
	t := testT{ginkgo.GinkgoT()}
	results := []domain.Result{
		{TargetID: "claude:0", Provider: domain.ProviderClaude, Account: "default", Status: domain.StatusOK, UsedPercent: pct(42)},
		{TargetID: "agy:0", Provider: domain.ProviderAgy, Account: "gemini", Status: domain.StatusOK, UsedPercent: pct(74.15718734264374)},
		{TargetID: "agy:1", Provider: domain.ProviderAgy, Account: "claude/gpt", Status: domain.StatusOK, UsedPercent: pct(12.76919841766357)},
	}
	got := Table(results)
	want := "AI PROVIDER  ACCOUNT     WEEKLY USAGE\n" +
		"Claude       default     42%\n" +
		"agy          gemini      74.2%\n" +
		"agy          claude/gpt  12.8%\n"
	if got != want {
		t.Fatalf("agy two-row table mismatch:\n got:\n%s\nwant:\n%s", got, want)
	}
})

var _ = ginkgo.It("TestTableCursorRow", func() {
	t := testT{ginkgo.GinkgoT()}
	results := []domain.Result{
		{TargetID: "agy:0", Provider: domain.ProviderAgy, Account: "gemini", Status: domain.StatusOK, UsedPercent: pct(74.15718734264374)},
		{TargetID: "agy:1", Provider: domain.ProviderAgy, Account: "claude/gpt", Status: domain.StatusOK, UsedPercent: pct(12.76919841766357)},
		{TargetID: "cursor:0", Provider: domain.ProviderCursor, Account: "default", Status: domain.StatusOK, UsedPercent: pct(25)},
	}
	got := Table(results)
	want := "AI PROVIDER  ACCOUNT     WEEKLY USAGE\n" +
		"agy          gemini      74.2%\n" +
		"agy          claude/gpt  12.8%\n" +
		"Cursor       default     25%\n"
	if got != want {
		t.Fatalf("cursor row table mismatch:\n got:\n%s\nwant:\n%s", got, want)
	}

	monthly := Table([]domain.Result{
		{TargetID: "cursor:0", Provider: domain.ProviderCursor, Account: "default", Status: domain.StatusNotApplicableMonthly},
	})
	wantMonthly := "AI PROVIDER  ACCOUNT  WEEKLY USAGE\n" +
		"Cursor       default  N/A (monthly)\n"
	if monthly != wantMonthly {
		t.Fatalf("cursor monthly row table mismatch:\n got:\n%s\nwant:\n%s", monthly, wantMonthly)
	}
})

var _ = ginkgo.It("TestTableZCodeFiveHourSection", func() {
	t := testT{ginkgo.GinkgoT()}
	both := Table([]domain.Result{
		{TargetID: "cursor:0", Provider: domain.ProviderCursor, Account: "default", Status: domain.StatusOK, UsedPercent: pct(25)},
		{TargetID: "zcode:0", Provider: domain.ProviderZCode, Account: "default", Status: domain.StatusOK, UsedPercent: pct(42)},
	})
	wantBoth := "AI PROVIDER  ACCOUNT  WEEKLY USAGE\n" +
		"Cursor       default  25%\n\n" +
		"AI PROVIDER  ACCOUNT  5H USAGE\n" +
		"ZCode        default  42%\n"
	if both != wantBoth {
		t.Fatalf("weekly+5h table mismatch:\n got:\n%s\nwant:\n%s", both, wantBoth)
	}

	onlyFiveHour := Table([]domain.Result{
		{TargetID: "zcode:0", Provider: domain.ProviderZCode, Account: "default", Status: domain.StatusOK, UsedPercent: pct(42)},
	})
	wantOnlyFiveHour := "AI PROVIDER  ACCOUNT  5H USAGE\n" +
		"ZCode        default  42%\n"
	if onlyFiveHour != wantOnlyFiveHour {
		t.Fatalf("5h-only table mismatch:\n got:\n%s\nwant:\n%s", onlyFiveHour, wantOnlyFiveHour)
	}
})
