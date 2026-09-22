package codex

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gaebokchi/internal/domain"
	"gaebokchi/internal/process"
	"gaebokchi/internal/testutil"
)

// fullScript builds a fake transcript answering the complete metadata
// sequence: initialize, account/read (chatgpt+email), rateLimits/read.
func fullScript(t *testing.T, accountFixture, quotaFixture string) []testutil.Step {
	t.Helper()
	return []testutil.Step{
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"platformOs": "macos"}}}},
		{Wait: 1},
		{Emit: compactFixture(t, accountFixture) + "\n"},
		{Wait: 1},
		{Emit: compactFixture(t, quotaFixture) + "\n"},
	}
}

func compactFixture(t *testing.T, name string) string {
	t.Helper()
	var envelope map[string]json.RawMessage
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	if err := json.Unmarshal(b, &envelope); err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	// Renumber ids to the per-connection sequence 1,2,3.
	var compact map[string]any
	if err := json.Unmarshal(b, &compact); err != nil {
		t.Fatal(err)
	}
	delete(compact, "id")
	switch name {
	case "account_chatgpt_home1.json":
		compact["id"] = 2
	case "account_chatgpt_home2.json":
		compact["id"] = 2
	case "ratelimits_weekly_home1.json", "ratelimits_weekly_home2.json":
		compact["id"] = 3
	}
	out, err := json.Marshal(compact)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func codexTarget(home string) domain.Target {
	return domain.Target{
		ID: "codex:0", Index: 0, Provider: domain.ProviderCodex,
		Home: home, FallbackAccount: filepath.Base(home), Bin: os.Args[0],
	}
}

// providerOver returns a Provider whose process factory always yields
// the pre-started fake child, so Query's lifecycle runs against it.
func providerOver(child *process.Child) *Provider {
	return &Provider{Start: func(process.Config) (*process.Child, error) { return child, nil }}
}

func TestQueryFullLifecycleFromFixtures(t *testing.T) {
	home := t.TempDir()
	child := testutil.StartFake(t, fullScript(t, "account_chatgpt_home1.json", "ratelimits_weekly_home1.json"))
	p := providerOver(child)
	res := p.Query(context.Background(), codexTarget(home))
	if res.Status != domain.StatusOK {
		t.Fatalf("status = %v (%s)", res.Status, res.Diagnostic.Message)
	}
	if res.UsedPercent == nil || *res.UsedPercent != 37 {
		t.Fatalf("percent = %v", res.UsedPercent)
	}
	if res.Account != "account1@example.com" || res.AccountSource != domain.AccountSourceProtocolEmail {
		t.Fatalf("account = %q (%v)", res.Account, res.AccountSource)
	}
	if res.Period != domain.PeriodWeekly || res.QuotaID != "codex" || res.Source != domain.SourceCodexAppServer {
		t.Fatalf("meta = %v/%q/%q", res.Period, res.QuotaID, res.Source)
	}
	if res.ResetAt == nil {
		t.Fatal("reset metadata not preserved")
	}
	child.Close()
	kept := string(child.StderrKept())
	if !containsStr(kept, `"method":"account/rateLimits/read"`) {
		t.Fatalf("quota read missing from transcript: %q", kept)
	}
}

func TestQueryTwoHomesDifferentValues(t *testing.T) {
	homeA := t.TempDir()
	homeB := t.TempDir()
	childA := testutil.StartFake(t, fullScript(t, "account_chatgpt_home1.json", "ratelimits_weekly_home1.json"))
	childB := testutil.StartFake(t, fullScript(t, "account_chatgpt_home2.json", "ratelimits_weekly_home2.json"))
	pa := providerOver(childA)
	pb := providerOver(childB)
	tgtA := codexTarget(homeA)
	tgtB := codexTarget(homeB)
	tgtB.ID = "codex:1"
	tgtB.Index = 1

	resB := pb.Query(context.Background(), tgtB)
	resA := pa.Query(context.Background(), tgtA)
	if resA.UsedPercent == nil || resB.UsedPercent == nil || *resA.UsedPercent == *resB.UsedPercent {
		t.Fatalf("values must differ per home: %v / %v", resA.UsedPercent, resB.UsedPercent)
	}
	if resA.Account == resB.Account {
		t.Fatalf("accounts must differ per home: %q", resA.Account)
	}
}

func TestQueryHomePreflight(t *testing.T) {
	p := New()
	missing := p.Query(context.Background(), codexTarget("/definitely/not/here"))
	if missing.Status != domain.StatusHomeNotFound {
		t.Fatalf("missing home status = %v", missing.Status)
	}
	file := filepath.Join(t.TempDir(), "plainfile")
	os.WriteFile(file, []byte("x"), 0o644)
	invalid := p.Query(context.Background(), codexTarget(file))
	if invalid.Status != domain.StatusHomeError {
		t.Fatalf("invalid home status = %v", invalid.Status)
	}
}

func TestQueryMissingExecutable(t *testing.T) {
	p := New()
	tgt := codexTarget(t.TempDir())
	tgt.Bin = "/nonexistent/gaebokchi-codex-missing"
	res := p.Query(context.Background(), tgt)
	if res.Status != domain.StatusCLINotFound {
		t.Fatalf("status = %v", res.Status)
	}
	if res.Account == "" {
		t.Fatal("fallback alias must be preserved")
	}
}

func TestQueryAuthOutcomesShortCircuit(t *testing.T) {
	accountPayload := func(account string) []testutil.Step {
		return []testutil.Step{
			{Wait: 1},
			{Reply: []testutil.ReplySpec{{Slot: 0, Result: map[string]any{"ok": true}}}},
			{Wait: 1},
			{Emit: `{"id":2,"result":{"account":` + account + `}}` + "\n"},
		}
	}
	cases := []struct {
		name    string
		account string
		want    domain.Status
	}{
		{"absent", "null", domain.StatusAuthRequired},
		{"api key", `{"type":"apiKey"}`, domain.StatusNotApplicableAPIKey},
		{"unknown", `{"type":"device"}`, domain.StatusUnsupported},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			child := testutil.StartFake(t, accountPayload(tc.account))
			p := providerOver(child)
			res := p.Query(context.Background(), codexTarget(t.TempDir()))
			if res.Status != tc.want {
				t.Fatalf("status = %v, want %v", res.Status, tc.want)
			}
			if res.AccountSource != domain.AccountSourceConfiguredAlias {
				t.Fatalf("identity source = %v", res.AccountSource)
			}
			// No quota read occurs after these outcomes.
			if containsStr(string(child.StderrKept()), "rateLimits") {
				t.Fatal("quota read must not run without a chatgpt account")
			}
		})
	}
}

// TestQueryValidResultSurvivesSlowCleanup reproduces the audit's
// executable-level finding at the provider boundary: every response is
// routed well inside the query budget, but the provider lingers after
// its final reply, so deferred cleanup crosses the deadline inside
// Query. The numeric result must survive; only a response established
// after the deadline may time out.
func TestQueryValidResultSurvivesSlowCleanup(t *testing.T) {
	steps := fullScript(t, "account_chatgpt_home1.json", "ratelimits_weekly_home1.json")
	steps = append(steps, testutil.Step{SleepMs: 600}) // linger like a slow provider
	child := testutil.StartFake(t, steps)
	p := providerOver(child)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	res := p.Query(ctx, codexTarget(t.TempDir()))
	if res.Status != domain.StatusOK {
		t.Fatalf("status = %v (%s); timely result discarded for slow cleanup", res.Status, res.Diagnostic.Message)
	}
	if res.UsedPercent == nil || *res.UsedPercent != 37 {
		t.Fatalf("percent = %v", res.UsedPercent)
	}
}

func TestQueryTimeoutClassification(t *testing.T) {
	steps := []testutil.Step{
		{Wait: 1},
		{SleepMs: 30000}, // never answers initialize
	}
	child := testutil.StartFake(t, steps)
	p := providerOver(child)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	res := p.Query(ctx, codexTarget(t.TempDir()))
	if res.Status != domain.StatusTimeout {
		t.Fatalf("status = %v (%s)", res.Status, res.Diagnostic.Message)
	}
	if res.Account == "" {
		t.Fatal("fallback alias lost on timeout")
	}
	if err := child.Close(); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
}

func TestQueryProtocolErrorClassification(t *testing.T) {
	steps := []testutil.Step{
		{Wait: 1},
		{Emit: "banner-not-json\n"},
	}
	child := testutil.StartFake(t, steps)
	p := providerOver(child)
	res := p.Query(context.Background(), codexTarget(t.TempDir()))
	if res.Status != domain.StatusProtocolError {
		t.Fatalf("status = %v", res.Status)
	}
	if containsStr(res.Diagnostic.Message, "banner") {
		t.Fatalf("raw provider data leaked into diagnostic: %q", res.Diagnostic.Message)
	}
}

func containsStr(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
