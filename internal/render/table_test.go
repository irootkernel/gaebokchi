package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"gaebokchi/internal/domain"
)

func pct(v float64) *float64 { return &v }

func TestTableGoldenMixedStatuses(t *testing.T) {
	results := []domain.Result{
		{Provider: domain.ProviderCodex, Account: "account1@example.com", Status: domain.StatusOK, UsedPercent: pct(37)},
		{Provider: domain.ProviderCodex, Account: ".codex-hsy", Status: domain.StatusAuthRequired},
		{Provider: domain.ProviderGrok, Account: "default", Status: domain.StatusNotApplicableMonthly},
	}
	got := Table(results)
	want := "AI PROVIDER  ACCOUNT               WEEKLY USAGE\n" +
		"ChatGPT      account1@example.com  37.0%\n" +
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
}

func TestTableAllNumeric(t *testing.T) {
	results := []domain.Result{
		{Provider: domain.ProviderCodex, Account: "a@example.com", Status: domain.StatusOK, UsedPercent: pct(0)},
		{Provider: domain.ProviderCodex, Account: "b@example.com", Status: domain.StatusOK, UsedPercent: pct(12.25)},
		{Provider: domain.ProviderGrok, Account: "default", Status: domain.StatusOK, UsedPercent: pct(54)},
	}
	got := Table(results)
	if !strings.Contains(got, "0.0%") || !strings.Contains(got, "12.2%") || !strings.Contains(got, "54.0%") {
		t.Fatalf("percentage formatting broken:\n%s", got)
	}
	if strings.Contains(got, "54.00%") {
		t.Fatal("not one decimal place")
	}
}

func TestUsageCellOneDecimal(t *testing.T) {
	cases := map[float64]string{0: "0.0%", 100: "100.0%", 37.06: "37.1%", 99.94: "99.9%", 5.5: "5.5%"}
	for v, want := range cases {
		got := UsageCell(domain.Result{Status: domain.StatusOK, UsedPercent: pct(v)})
		if got != want {
			t.Errorf("UsageCell(%v) = %q, want %q", v, got, want)
		}
	}
}

func TestSanitizeLabel(t *testing.T) {
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
		if got := SanitizeLabel(in); got != want {
			t.Errorf("SanitizeLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSanitizedLabelCannotAddColumnsOrRows(t *testing.T) {
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
}

func TestDiagnosticsOnlyForNonCleanRows(t *testing.T) {
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
}

func TestResetAndProvenanceNeverRendered(t *testing.T) {
	reset := time.Unix(1790411774, 0)
	results := []domain.Result{{
		Provider: domain.ProviderCodex, Account: "a@example.com", Status: domain.StatusOK, UsedPercent: pct(7),
		QuotaID: "codex", ResetAt: &reset, Source: domain.SourceCodexAppServer, Period: domain.PeriodWeekly,
	}}
	got := Table(results)
	if strings.Contains(got, "codex") || strings.Contains(got, "app_server") || strings.Contains(got, "2026") {
		t.Fatalf("window provenance leaked into the table:\n%s", got)
	}
}
