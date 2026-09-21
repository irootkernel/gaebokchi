package grok

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"gaebokchi/internal/domain"
	"gaebokchi/internal/process"
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

func TestNormalizeBillingMatrix(t *testing.T) {
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
		{"period absent legacy only", billingConfig(nil, 55, map[string]any{"monthlyLimit": 100, "used": 55}), domain.StatusUnavailablePeriod, nil, false},
		{"period absent plain", billingConfig(nil, 10, nil), domain.StatusUnavailablePeriod, nil, false},
		{"unknown enum", billingConfig("USAGE_PERIOD_TYPE_QUARTERLY", 10, nil), domain.StatusUnavailablePeriod, nil, false},
		{"type null", billingConfig(nil, 10, nil), domain.StatusUnavailablePeriod, nil, false},
		{"weekly credit absent", billingConfig("USAGE_PERIOD_TYPE_WEEKLY", nil, nil), domain.StatusUnavailableNoData, nil, false},
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
		t.Run(tc.name, func(t *testing.T) {
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
}

func TestMonthlyNeverBecomesWeeklyValue(t *testing.T) {
	out := normalizeBilling(json.RawMessage(billingConfig("USAGE_PERIOD_TYPE_MONTHLY", 88, nil)))
	if out.Percent != nil || out.Period == domain.PeriodWeekly {
		t.Fatalf("monthly payload leaked into weekly semantics: %+v", out)
	}
	if out.Status != domain.StatusNotApplicableMonthly {
		t.Fatalf("status = %v", out.Status)
	}
}

func TestLegacyFieldsNeverConverted(t *testing.T) {
	out := normalizeBilling(json.RawMessage(`{"config":{"monthlyLimit":100,"used":55,"prepaidBalance":{"val":10},"onDemandUsed":{"val":3}}}`))
	if out.Percent != nil || out.Status != domain.StatusUnavailablePeriod {
		t.Fatalf("legacy credits converted: %+v", out)
	}
}

func fp(v float64) *float64 { return &v }

// ---- full Query lifecycle ----

func grokTarget(alias string) domain.Target {
	return domain.Target{
		ID: "grok:0", Index: 0, Provider: domain.ProviderGrok,
		FallbackAccount: alias, Bin: os.Args[0],
	}
}

func providerOver(child *process.Child) *Provider {
	return &Provider{Start: func(process.Config) (*process.Child, error) { return child, nil }}
}

func TestQueryWeeklyFromCapturedFixture(t *testing.T) {
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
}

func TestQueryMonthlyNonApplicable(t *testing.T) {
	monthly := billingConfig("USAGE_PERIOD_TYPE_MONTHLY", 88, nil)
	steps := []testutil.Step{
		{Wait: 1},
		{Emit: initReplyWith(1, "cached_token") + "\n"},
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 1, Result: map[string]any{"_meta": map[string]any{"ok": true}}}}},
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 2, RawResult: monthly}}},
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
}

func TestQueryAuthAndUnsupportedPaths(t *testing.T) {
	code := -32601
	t.Run("api key only", func(t *testing.T) {
		steps := []testutil.Step{{Wait: 1}, {Emit: initReplyWith(1, "api_key") + "\n"}}
		child := testutil.StartFake(t, steps)
		res := providerOver(child).Query(context.Background(), grokTarget("default"))
		if res.Status != domain.StatusNotApplicableAPIKey {
			t.Fatalf("status = %v", res.Status)
		}
	})
	t.Run("interactive only", func(t *testing.T) {
		steps := []testutil.Step{{Wait: 1}, {Emit: initReplyWith(1, "grok.com") + "\n"}}
		child := testutil.StartFake(t, steps)
		res := providerOver(child).Query(context.Background(), grokTarget("default"))
		if res.Status != domain.StatusAuthRequired {
			t.Fatalf("status = %v", res.Status)
		}
	})
	t.Run("authenticate rejected", func(t *testing.T) {
		steps := []testutil.Step{
			{Wait: 1}, {Emit: initReplyWith(1, "cached_token") + "\n"},
			{Wait: 1},
			{Reply: []testutil.ReplySpec{{Slot: 1, ErrCode: &code, ErrMsg: "no"}}},
		}
		child := testutil.StartFake(t, steps)
		res := providerOver(child).Query(context.Background(), grokTarget("default"))
		if res.Status != domain.StatusAuthRequired {
			t.Fatalf("status = %v", res.Status)
		}
	})
	t.Run("billing method not found", func(t *testing.T) {
		steps := []testutil.Step{
			{Wait: 1}, {Emit: initReplyWith(1, "cached_token") + "\n"},
			{Wait: 1},
			{Reply: []testutil.ReplySpec{{Slot: 1, Result: map[string]any{"_meta": map[string]any{}}}}},
			{Wait: 1},
			{Reply: []testutil.ReplySpec{{Slot: 2, ErrCode: &code, ErrMsg: "method not found"}}},
		}
		child := testutil.StartFake(t, steps)
		res := providerOver(child).Query(context.Background(), grokTarget("default"))
		if res.Status != domain.StatusUnsupported {
			t.Fatalf("status = %v", res.Status)
		}
	})
	t.Run("version mismatch", func(t *testing.T) {
		steps := []testutil.Step{{Wait: 1}, {Emit: initReplyWith(2, "cached_token") + "\n"}}
		child := testutil.StartFake(t, steps)
		res := providerOver(child).Query(context.Background(), grokTarget("default"))
		if res.Status != domain.StatusUnsupported {
			t.Fatalf("status = %v", res.Status)
		}
	})
}

func TestQueryAliasOnEveryOutcome(t *testing.T) {
	alias := "personal"
	cases := [][]testutil.Step{
		{{Wait: 1}, {Emit: initReplyWith(1, "cached_token") + "\n"}, {Wait: 1}, {Reply: []testutil.ReplySpec{{Slot: 1, Result: map[string]any{}}}}, {Wait: 1}, {Reply: []testutil.ReplySpec{{Slot: 2, RawResult: billingConfig("USAGE_PERIOD_TYPE_WEEKLY", 5, nil)}}}},
		{{Wait: 1}, {Emit: initReplyWith(1, "grok.com") + "\n"}},
		{{Wait: 1}, {Emit: initReplyWith(1, "cached_token") + "\n"}, {Wait: 1}, {Reply: []testutil.ReplySpec{{Slot: 1, Result: map[string]any{}}}}, {Wait: 1}, {Reply: []testutil.ReplySpec{{Slot: 2, RawResult: billingConfig(nil, 5, map[string]any{"monthlyLimit": 9})}}}},
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
}

func TestQueryMissingExecutable(t *testing.T) {
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
}

func TestQueryTimeout(t *testing.T) {
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
}
