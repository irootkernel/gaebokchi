package grok

import (
	"context"
	"encoding/json"
	ginkgo "github.com/onsi/ginkgo/v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gaebokchi/internal/domain"
	"gaebokchi/internal/process"
	"gaebokchi/internal/render"
	"gaebokchi/internal/testutil"
)

func billingConfig(periodType any, credit any, extra map[string]any) string {
	cfg := map[string]any{}
	if periodType != nil {
		period := map[string]any{"type": periodType}
		if end, ok := extra["periodEnd"]; ok {
			period["end"] = end
		}
		cfg["currentPeriod"] = period
	}
	if credit != nil {
		cfg["creditUsagePercent"] = credit
	}
	for k, v := range extra {
		if k == "periodEnd" {
			continue
		}
		cfg[k] = v
	}
	b, _ := json.Marshal(map[string]any{"config": cfg})
	return string(b)
}

var _ = ginkgo.Describe("TestNormalizeBillingMatrix", func() {
	cases := []struct {
		name      string
		payload   string
		want      domain.Status
		wantPct   *float64
		wantReset bool
	}{
		{"captured weekly fixture shape", billingConfig("USAGE_PERIOD_TYPE_WEEKLY", 28.0, map[string]any{"periodEnd": "2026-09-26T15:52:30Z", "prepaidBalance": map[string]any{"val": 0}, "isUnifiedBillingUser": true}), domain.StatusOK, fp(28), true},
		{"valid zero", billingConfig("USAGE_PERIOD_TYPE_WEEKLY", 0, nil), domain.StatusOK, fp(0), false},
		{"valid integer encoding", billingConfig("USAGE_PERIOD_TYPE_WEEKLY", 42, nil), domain.StatusOK, fp(42), false},
		{"boundary 100", billingConfig("USAGE_PERIOD_TYPE_WEEKLY", 100, nil), domain.StatusOK, fp(100), false},
		{"monthly", billingConfig("USAGE_PERIOD_TYPE_MONTHLY", 88, nil), domain.StatusNotApplicableMonthly, nil, false},
		{"monthly credit absent", billingConfig("USAGE_PERIOD_TYPE_MONTHLY", nil, nil), domain.StatusNotApplicableMonthly, nil, false},
		{"period absent legacy only", billingConfig(nil, 55, map[string]any{"monthlyLimit": 100, "used": 55}), domain.StatusUnavailablePeriod, nil, false},
		{"period absent plain", billingConfig(nil, 10, nil), domain.StatusUnavailablePeriod, nil, false},
		{"unknown enum", billingConfig("USAGE_PERIOD_TYPE_QUARTERLY", 10, nil), domain.StatusUnavailablePeriod, nil, false},
		{"type null", billingConfig(nil, 10, nil), domain.StatusUnavailablePeriod, nil, false},
		{"weekly credit absent", billingConfig("USAGE_PERIOD_TYPE_WEEKLY", nil, nil), domain.StatusOK, fp(0), false},
		{"weekly credit absent ignores legacy amounts", billingConfig("USAGE_PERIOD_TYPE_WEEKLY", nil, map[string]any{"monthlyLimit": map[string]any{"val": 100}, "used": map[string]any{"val": 50}}), domain.StatusOK, fp(0), false},
		{"weekly credit null", `{"config":{"currentPeriod":{"type":"USAGE_PERIOD_TYPE_WEEKLY"},"creditUsagePercent":null}}`, domain.StatusUnavailableNoData, nil, false},
		{"weekly credit string", billingConfig("USAGE_PERIOD_TYPE_WEEKLY", "28", nil), domain.StatusProtocolError, nil, false},
		{"weekly credit negative", billingConfig("USAGE_PERIOD_TYPE_WEEKLY", -1, nil), domain.StatusProtocolError, nil, false},
		{"weekly credit over 100", billingConfig("USAGE_PERIOD_TYPE_WEEKLY", 100.5, nil), domain.StatusProtocolError, nil, false},
		{"config null", `{"config":null}`, domain.StatusUnavailableNoData, nil, false},
		{"config absent", `{"subscription_tier":"SuperGrok Heavy"}`, domain.StatusUnavailableNoData, nil, false},
		{"config wrong type", `{"config":42}`, domain.StatusProtocolError, nil, false},
		{"result null", `null`, domain.StatusUnavailableNoData, nil, false},
		{"result not object", `"billing"`, domain.StatusProtocolError, nil, false},
		{"period wrong type", `{"config":{"currentPeriod":17}}`, domain.StatusProtocolError, nil, false},
		{"type wrong type", `{"config":{"currentPeriod":{"type":3}}}`, domain.StatusProtocolError, nil, false},
		{"malformed optional end discarded", billingConfig("USAGE_PERIOD_TYPE_WEEKLY", 30, map[string]any{"periodEnd": "not-a-date"}), domain.StatusOK, fp(30), false},
	}
	for _, tc := range cases {
		ginkgo.It(tc.name, func() {
			t := testT{ginkgo.GinkgoT()}
			out := normalizeBilling(json.RawMessage(tc.payload))
			if out.Status != tc.want {
				t.Fatalf("status = %v, want %v", out.Status, tc.want)
			}
			if tc.wantPct != nil {
				if out.Percent == nil || *out.Percent != *tc.wantPct {
					t.Fatalf("percent = %v, want %v", out.Percent, *tc.wantPct)
				}
				if out.Period != domain.PeriodWeekly {
					t.Fatalf("period = %v, want weekly", out.Period)
				}
			} else if out.Percent != nil {
				t.Fatalf("unexpected percent %v", *out.Percent)
			}
			if tc.wantReset && out.ResetAt == nil {
				t.Fatal("valid optional reset lost")
			}
			if !tc.wantReset && out.ResetAt != nil {
				t.Fatalf("unexpected reset %v", out.ResetAt)
			}
		})
	}
})

var _ = ginkgo.It("TestMonthlyNeverBecomesWeeklyValue", func() {
	t := testT{ginkgo.GinkgoT()}
	out := normalizeBilling(json.RawMessage(billingConfig("USAGE_PERIOD_TYPE_MONTHLY", 88, nil)))
	if out.Percent != nil || out.Period == domain.PeriodWeekly {
		t.Fatalf("monthly payload leaked into weekly semantics: %+v", out)
	}
	if out.Status != domain.StatusNotApplicableMonthly {
		t.Fatalf("status = %v", out.Status)
	}
})

var _ = ginkgo.It("TestLegacyFieldsNeverConverted", func() {
	t := testT{ginkgo.GinkgoT()}
	out := normalizeBilling(json.RawMessage(`{"config":{"monthlyLimit":100,"used":55,"prepaidBalance":{"val":10},"onDemandUsed":{"val":3}}}`))
	if out.Percent != nil || out.Status != domain.StatusUnavailablePeriod {
		t.Fatalf("legacy credits converted: %+v", out)
	}
})

func fp(v float64) *float64 { return &v }

// ---- full Query lifecycle ----

func grokTarget(alias string) domain.Target {
	return domain.Target{
		ID: "grok:0", Index: 0, Provider: domain.ProviderGrok,
		FallbackAccount: alias, Home: os.TempDir(), Bin: os.Args[0],
	}
}

var _ = ginkgo.It("rejects an inaccessible Grok home before starting the CLI", func() {
	t := testT{ginkgo.GinkgoT()}
	path := filepath.Join(t.TempDir(), "restricted")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(path, 0o700) }()
	if dir, err := os.Open(path); err == nil {
		_ = dir.Close()
		t.Skip("directory permission checks are bypassed by this user")
	}
	target := grokTarget("restricted")
	target.Home = path
	target.Bin = filepath.Join(path, "missing-cli")
	result := New().Query(context.Background(), target)
	if result.Status != domain.StatusHomeError || result.Diagnostic.Code != "home_error" {
		t.Fatalf("inaccessible Grok home = %+v", result)
	}
})

func providerOver(child *process.Child) *Provider {
	return &Provider{Start: func(process.Config) (*process.Child, error) { return child, nil }}
}

var _ = ginkgo.It("TestQueryWeeklyFromCapturedFixture", func() {
	t := testT{ginkgo.GinkgoT()}
	steps := []testutil.Step{
		{Wait: 1},
		{Emit: compactFixture(t, "initialize_response.json") + "\n"},
		{Wait: 1},
		{Emit: compactFixture(t, "authenticate_cached_token_response.json") + "\n"},
		{Wait: 1},
		{Emit: compactFixture(t, "billing_weekly_response.json") + "\n"},
	}
	child := testutil.StartFake(t, steps)
	p := providerOver(child)
	res := p.Query(context.Background(), grokTarget("lab"))
	if res.Status != domain.StatusOK {
		t.Fatalf("status = %v (%s)", res.Status, res.Diagnostic.Message)
	}
	if res.UsedPercent == nil || *res.UsedPercent != 28 {
		t.Fatalf("percent = %v", res.UsedPercent)
	}
	if res.Account != "lab" || res.AccountSource != domain.AccountSourceConfiguredAlias {
		t.Fatalf("account = %q (%v): the billing/auth payloads are not email sources", res.Account, res.AccountSource)
	}
	if res.Period != domain.PeriodWeekly || res.Source != domain.SourceGrokACPBilling {
		t.Fatalf("meta = %v/%q", res.Period, res.Source)
	}
	if res.ResetAt == nil {
		t.Fatal("period end not preserved as reset metadata")
	}
})

var _ = ginkgo.It("TestQueryMonthlyNonApplicable", func() {
	t := testT{ginkgo.GinkgoT()}
	monthly := billingConfig("USAGE_PERIOD_TYPE_MONTHLY", 88, map[string]any{"periodEnd": "2026-10-19T15:52:30Z"})
	steps := []testutil.Step{
		{Wait: 1},
		{Emit: initReplyWith(1, "cached_token") + "\n"},
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 1, JSONRPC: true, Result: map[string]any{"_meta": map[string]any{"ok": true}}}}},
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 2, JSONRPC: true, RawResult: monthly}}},
	}
	child := testutil.StartFake(t, steps)
	p := providerOver(child)
	res := p.Query(context.Background(), grokTarget("default"))
	if res.Status != domain.StatusNotApplicableMonthly {
		t.Fatalf("status = %v", res.Status)
	}
	if res.UsedPercent != nil {
		t.Fatal("monthly percentage must not appear as weekly usage")
	}
	if res.Status.ExitClass() != domain.ExitClean {
		t.Fatalf("monthly non-applicability must be exit-clean")
	}
	// The audit's provenance finding: the established monthly period
	// must survive in the in-memory result even without a percentage.
	if res.Period != domain.PeriodMonthly {
		t.Fatalf("period = %v, want monthly", res.Period)
	}
	if res.Source != domain.SourceGrokACPBilling {
		t.Fatalf("source = %v", res.Source)
	}
	if res.ResetAt == nil {
		t.Fatal("monthly period end must be preserved as reset metadata")
	}
})

// An omitted weekly percentage uses the chosen display convention while
// keeping the period and reset provenance for every output format.
var _ = ginkgo.It("TestWeeklyOmittedUsageDefaultsToZero", func() {
	t := testT{ginkgo.GinkgoT()}
	noUsage := map[string]any{"currentPeriod": map[string]any{
		"type": "USAGE_PERIOD_TYPE_WEEKLY",
		"end":  "2026-09-26T15:52:30Z",
	}}
	steps := []testutil.Step{
		{Wait: 1}, {Emit: initReplyWith(1, "cached_token") + "\n"},
		{Wait: 1}, {Reply: []testutil.ReplySpec{{Slot: 1, JSONRPC: true, Result: map[string]any{"_meta": map[string]any{}}}}},
		{Wait: 1}, {Reply: []testutil.ReplySpec{{Slot: 2, JSONRPC: true, Result: map[string]any{"config": noUsage}}}},
	}
	child := testutil.StartFake(t, steps)
	res := providerOver(child).Query(context.Background(), grokTarget("default"))
	if res.Status != domain.StatusOK || res.UsedPercent == nil || *res.UsedPercent != 0 {
		t.Fatalf("status/percent = %v/%v, want ok/0", res.Status, res.UsedPercent)
	}
	if res.Period != domain.PeriodWeekly {
		t.Fatalf("period = %v, want weekly provenance retained", res.Period)
	}
	if res.Source != domain.SourceGrokACPBilling {
		t.Fatalf("source = %v", res.Source)
	}
	if res.ResetAt == nil {
		t.Fatal("weekly reset lost")
	}
	if res.Diagnostic.Message != "" || res.Status.ExitClass() != domain.ExitClean {
		t.Fatalf("zero fallback should be clean: %+v", res)
	}
	if table := render.Table([]domain.Result{res}); !strings.HasSuffix(table, "  0%\n") {
		t.Fatalf("table missing zero percentage: %q", table)
	}
	jsonView, err := render.JSON([]domain.Result{res})
	if err != nil || !strings.Contains(jsonView, `"used_percent":0`) {
		t.Fatalf("JSON missing numeric zero: %q (%v)", jsonView, err)
	}
	frame, _ := render.TerminalFrame([]domain.Result{res}, 100, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC), false, false, "")
	if !strings.Contains(frame, "    0%") || !strings.Contains(frame, "↻") {
		t.Fatalf("terminal missing zero or reset: %q", frame)
	}
})

var _ = ginkgo.Describe("TestQueryAuthAndUnsupportedPaths", func() {
	code := -32601
	ginkgo.It("api key only", func() {
		t := testT{ginkgo.GinkgoT()}
		steps := []testutil.Step{{Wait: 1}, {Emit: initReplyWith(1, "api_key") + "\n"}}
		child := testutil.StartFake(t, steps)
		res := providerOver(child).Query(context.Background(), grokTarget("default"))
		if res.Status != domain.StatusNotApplicableAPIKey {
			t.Fatalf("status = %v", res.Status)
		}
	})
	ginkgo.It("interactive only", func() {
		t := testT{ginkgo.GinkgoT()}
		steps := []testutil.Step{{Wait: 1}, {Emit: initReplyWith(1, "grok.com") + "\n"}}
		child := testutil.StartFake(t, steps)
		res := providerOver(child).Query(context.Background(), grokTarget("default"))
		if res.Status != domain.StatusAuthRequired {
			t.Fatalf("status = %v", res.Status)
		}
	})
	ginkgo.It("authenticate server error is generic", func() {
		t := testT{ginkgo.GinkgoT()}
		code := -32000
		steps := []testutil.Step{
			{Wait: 1}, {Emit: initReplyWith(1, "cached_token") + "\n"},
			{Wait: 1},
			{Reply: []testutil.ReplySpec{{Slot: 1, JSONRPC: true, ErrCode: &code, ErrMsg: "internal error"}}},
		}
		child := testutil.StartFake(t, steps)
		res := providerOver(child).Query(context.Background(), grokTarget("default"))
		// The audit's finding: an internal error is not a verified
		// authentication rejection and must not claim the cached token
		// was rejected.
		if res.Status != domain.StatusProviderError {
			t.Fatalf("status = %v (%s), want provider_error", res.Status, res.Diagnostic.Message)
		}
		if res.Diagnostic.Code != "remote_error" {
			t.Fatalf("diagnostic code = %q", res.Diagnostic.Code)
		}
		if strings.Contains(res.Diagnostic.Message, "cached token") {
			t.Fatalf("guessed authentication detail: %q", res.Diagnostic.Message)
		}
	})
	ginkgo.It("authenticate method not found is unsupported", func() {
		t := testT{ginkgo.GinkgoT()}
		steps := []testutil.Step{
			{Wait: 1}, {Emit: initReplyWith(1, "cached_token") + "\n"},
			{Wait: 1},
			{Reply: []testutil.ReplySpec{{Slot: 1, JSONRPC: true, ErrCode: &code, ErrMsg: "method not found"}}},
		}
		child := testutil.StartFake(t, steps)
		res := providerOver(child).Query(context.Background(), grokTarget("default"))
		if res.Status != domain.StatusUnsupported {
			t.Fatalf("status = %v (%s), want unsupported", res.Status, res.Diagnostic.Message)
		}
	})
	ginkgo.It("billing method not found", func() {
		t := testT{ginkgo.GinkgoT()}
		steps := []testutil.Step{
			{Wait: 1}, {Emit: initReplyWith(1, "cached_token") + "\n"},
			{Wait: 1},
			{Reply: []testutil.ReplySpec{{Slot: 1, JSONRPC: true, Result: map[string]any{"_meta": map[string]any{}}}}},
			{Wait: 1},
			{Reply: []testutil.ReplySpec{{Slot: 2, JSONRPC: true, ErrCode: &code, ErrMsg: "method not found"}}},
		}
		child := testutil.StartFake(t, steps)
		res := providerOver(child).Query(context.Background(), grokTarget("default"))
		if res.Status != domain.StatusUnsupported {
			t.Fatalf("status = %v", res.Status)
		}
	})
	ginkgo.It("version mismatch", func() {
		t := testT{ginkgo.GinkgoT()}
		steps := []testutil.Step{{Wait: 1}, {Emit: initReplyWith(2, "cached_token") + "\n"}}
		child := testutil.StartFake(t, steps)
		res := providerOver(child).Query(context.Background(), grokTarget("default"))
		if res.Status != domain.StatusUnsupported {
			t.Fatalf("status = %v", res.Status)
		}
	})
})

var _ = ginkgo.It("TestQueryAliasOnEveryOutcome", func() {
	t := testT{ginkgo.GinkgoT()}
	alias := "personal"
	authOK := testutil.Step{Reply: []testutil.ReplySpec{{Slot: 1, JSONRPC: true, Result: map[string]any{"_meta": map[string]any{}}}}}
	weekly := testutil.Step{Reply: []testutil.ReplySpec{{Slot: 2, JSONRPC: true, RawResult: billingConfig("USAGE_PERIOD_TYPE_WEEKLY", 5, nil)}}}
	legacyOnly := testutil.Step{Reply: []testutil.ReplySpec{{Slot: 2, JSONRPC: true, RawResult: billingConfig(nil, 5, map[string]any{"monthlyLimit": 9})}}}
	cases := [][]testutil.Step{
		withCapturedInit(t, testutil.Step{Wait: 1}, authOK, testutil.Step{Wait: 1}, weekly),
		{{Wait: 1}, {Emit: initReplyWith(1, "grok.com") + "\n"}},
		withCapturedInit(t, testutil.Step{Wait: 1}, authOK, testutil.Step{Wait: 1}, legacyOnly),
	}
	for i, steps := range cases {
		child := testutil.StartFake(t, steps)
		res := providerOver(child).Query(context.Background(), grokTarget(alias))
		if res.Account != alias {
			t.Fatalf("case %d: account = %q, want alias %q", i, res.Account, alias)
		}
		if res.AccountSource != domain.AccountSourceConfiguredAlias {
			t.Fatalf("case %d: account source = %v", i, res.AccountSource)
		}
		if strings.Contains(res.Account, "@") {
			t.Fatalf("case %d: email leaked into account column", i)
		}
	}
})

var _ = ginkgo.It("TestQueryMissingExecutable", func() {
	t := testT{ginkgo.GinkgoT()}
	p := New()
	tgt := grokTarget("default")
	tgt.Bin = "/nonexistent/gaebokchi-grok-missing"
	res := p.Query(context.Background(), tgt)
	if res.Status != domain.StatusCLINotFound {
		t.Fatalf("status = %v", res.Status)
	}
	if res.Account != "default" {
		t.Fatalf("alias lost: %q", res.Account)
	}
})

var _ = ginkgo.It("TestQueryTimeout", func() {
	t := testT{ginkgo.GinkgoT()}
	steps := []testutil.Step{{Wait: 1}, {SleepMs: 30000}}
	child := testutil.StartFake(t, steps)
	p := providerOver(child)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	res := p.Query(ctx, grokTarget("default"))
	if res.Status != domain.StatusTimeout {
		t.Fatalf("status = %v", res.Status)
	}
	if err := child.Close(); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
})

// TestAuthenticateResultShapeValidation pins the audit's finding that a
// malformed authenticate response (error:null, result:null, a shapeless
// object, or an incompatible envelope) used to count as an
// authenticated session: each must be a protocol failure, never a
// numeric success and never an unsupported authentication claim.
var _ = ginkgo.Describe("TestAuthenticateResultShapeValidation", func() {
	cases := []struct {
		name string
		auth string
	}{
		{"error null without result", `{"jsonrpc":"2.0","id":2,"error":null}`},
		{"result null", `{"jsonrpc":"2.0","id":2,"result":null}`},
		{"empty object without meta", `{"jsonrpc":"2.0","id":2,"result":{}}`},
		{"bare scalar", `{"jsonrpc":"2.0","id":2,"result":"ok"}`},
		{"incompatible jsonrpc marker", `{"jsonrpc":"1.0","id":2,"result":{"_meta":{}}}`},
		{"missing jsonrpc marker", `{"id":2,"result":{"_meta":{}}}`},
	}
	for _, tc := range cases {
		ginkgo.It(tc.name, func() {
			t := testT{ginkgo.GinkgoT()}
			steps := []testutil.Step{
				{Wait: 1}, {Emit: initReplyWith(1, "cached_token") + "\n"},
				{Wait: 1}, {Emit: tc.auth + "\n"},
			}
			child := testutil.StartFake(t, steps)
			res := providerOver(child).Query(context.Background(), grokTarget("default"))
			if res.Status != domain.StatusProtocolError {
				t.Fatalf("status = %v (%s), want protocol_error", res.Status, res.Diagnostic.Message)
			}
		})
	}
	ginkgo.It("qualified shape proceeds to billing", func() {
		t := testT{ginkgo.GinkgoT()}
		steps := []testutil.Step{
			{Wait: 1}, {Emit: initReplyWith(1, "cached_token") + "\n"},
			{Wait: 1}, {Reply: []testutil.ReplySpec{{Slot: 1, JSONRPC: true, Result: map[string]any{"_meta": map[string]any{"auth_mode": "Oidc"}, "additive": true}}}},
			{Wait: 1}, {Reply: []testutil.ReplySpec{{Slot: 2, JSONRPC: true, RawResult: billingConfig("USAGE_PERIOD_TYPE_WEEKLY", 5, nil)}}},
		}
		child := testutil.StartFake(t, steps)
		res := providerOver(child).Query(context.Background(), grokTarget("default"))
		if res.Status != domain.StatusOK || res.UsedPercent == nil || *res.UsedPercent != 5 {
			t.Fatalf("status = %v percent = %v; qualified shape must authenticate", res.Status, res.UsedPercent)
		}
	})
})
