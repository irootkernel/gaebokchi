package codex

import (
	"encoding/json"
	ginkgo "github.com/onsi/ginkgo/v2"
	"os"

	"gaebokchi/internal/domain"
)

func mustStatus(t testT, raw string) domain.Status {
	t.Helper()
	return normalizeQuota(json.RawMessage(raw)).Status
}

func weeklyBucket(primary, secondary string) string {
	return `{"rateLimitsByLimitId":{"codex":{"limitId":"codex","primary":` + primary + `,"secondary":` + secondary + `}}}`
}

func window(pct, dur, reset any) string {
	m := map[string]any{}
	if pct != nil {
		m["usedPercent"] = pct
	}
	if dur != nil {
		m["windowDurationMins"] = dur
	}
	if reset != nil {
		m["resetsAt"] = reset
	}
	b, _ := json.Marshal(m)
	return string(b)
}

var _ = ginkgo.It("TestNormalizeCapturedFixtureShapes", func() {
	t := testT{ginkgo.GinkgoT()}
	for _, name := range []string{"ratelimits_weekly_home1.json", "ratelimits_weekly_home2.json"} {
		var envelope struct {
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(readFixtureRaw(t, name), &envelope); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		out := normalizeQuota(envelope.Result)
		if out.Status != domain.StatusOK {
			t.Fatalf("%s: status = %v", name, out.Status)
		}
		if out.Percent == nil || *out.Percent != 37 && *out.Percent != 38 {
			t.Fatalf("%s: percent = %v", name, out.Percent)
		}
		if out.QuotaID != "codex" {
			t.Fatalf("%s: quota id = %q", name, out.QuotaID)
		}
		if out.ResetAt == nil || out.ResetAt.Unix() < 1700000000 {
			t.Fatalf("%s: reset = %v", name, out.ResetAt)
		}
	}
})

var _ = ginkgo.Describe("TestNormalizeWindowSlotSelection", func() {
	cases := []struct {
		name      string
		primary   string
		secondary string
		want      domain.Status
		wantPct   *float64
	}{
		{"primary weekly", window(37, 10080, 1790411774), "null", domain.StatusOK, fp(37)},
		{"secondary weekly", "null", window(12.5, 10080, nil), domain.StatusOK, fp(12.5)},
		{"swapped slots secondary only", window(80, 300, nil), window(44, 10080, 1790000000), domain.StatusOK, fp(44)},
		{"five hour primary only", window(80, 300, nil), "null", domain.StatusUnavailableNoWeekly, nil},
		{"no duration", window(50, nil, nil), "null", domain.StatusUnavailableNoWeekly, nil},
		{"null secondary", window(37, 10080, 1790411774), "null", domain.StatusOK, fp(37)},
		{"absent secondary", window(0, 10080, nil), "null", domain.StatusOK, fp(0)},
		{"explicit zero preserved", window(0, 10080, nil), "null", domain.StatusOK, fp(0)},
		{"both weekly agree", window(37, 10080, 1790411774), window(37, 10080, 1790411774), domain.StatusOK, fp(37)},
		{"both weekly numeric agreement 37 vs 37.0", window(37, 10080, nil), window(37.0, 10080, nil), domain.StatusOK, fp(37)},
		{"both weekly conflict", window(37, 10080, nil), window(38, 10080, nil), domain.StatusProtocolError, nil},
		{"both weekly reset mismatch", window(37, 10080, 1790411774), window(37, 10080, 1790000000), domain.StatusProtocolError, nil},
		{"both weekly one-sided reset", window(37, 10080, 1790411774), window(37, 10080, nil), domain.StatusProtocolError, nil},
		{"weekly missing usage", window(nil, 10080, nil), "null", domain.StatusUnavailableNoData, nil},
		{"weekly usage null", `{"windowDurationMins":10080,"usedPercent":null}`, "null", domain.StatusUnavailableNoData, nil},
		{"weekly usage string type", window("37", 10080, nil), "null", domain.StatusProtocolError, nil},
		{"weekly usage negative", window(-1, 10080, nil), "null", domain.StatusProtocolError, nil},
		{"weekly usage over 100", window(100.5, 10080, nil), "null", domain.StatusProtocolError, nil},
		{"weekly boundary 100 ok", window(100, 10080, nil), "null", domain.StatusOK, fp(100)},
		{"duration non-integer", window(37, 10080.5, nil), "null", domain.StatusProtocolError, nil},
		{"duration string type", window(37, "10080", nil), "null", domain.StatusProtocolError, nil},
	}
	for _, tc := range cases {
		ginkgo.It(tc.name, func() {
			t := testT{ginkgo.GinkgoT()}
			raw := weeklyBucket(tc.primary, tc.secondary)
			out := normalizeQuota(json.RawMessage(raw))
			if out.Status != tc.want {
				t.Fatalf("status = %v, want %v", out.Status, tc.want)
			}
			if tc.wantPct != nil {
				if out.Percent == nil || *out.Percent != *tc.wantPct {
					t.Fatalf("percent = %v, want %v", out.Percent, *tc.wantPct)
				}
			} else if out.Percent != nil {
				t.Fatalf("unexpected percent %v", *out.Percent)
			}
		})
	}
})

var _ = ginkgo.Describe("TestNormalizeBucketSelection", func() {
	weekly := window(37, 10080, 1790411774)
	cases := []struct {
		name string
		raw  string
		want domain.Status
	}{
		{"non-empty map without codex", `{"rateLimitsByLimitId":{"other":` + weekly + `}}`, domain.StatusUnavailableNoWeekly},
		{"non-empty map codex null", `{"rateLimitsByLimitId":{"codex":null}}`, domain.StatusUnavailableNoData},
		{"codex wrong structural type", `{"rateLimitsByLimitId":{"codex":42}}`, domain.StatusProtocolError},
		{"map not an object", `{"rateLimitsByLimitId":"nope"}`, domain.StatusProtocolError},
		{"empty map falls to legacy with codex", `{"rateLimitsByLimitId":{},"rateLimits":{"limitId":"codex","primary":` + weekly + `}}`, domain.StatusOK},
		{"map null falls to legacy", `{"rateLimitsByLimitId":null,"rateLimits":{"limitId":"codex","primary":` + weekly + `}}`, domain.StatusOK},
		{"legacy unrelated limitId", `{"rateLimits":{"limitId":"gpt5e","primary":` + weekly + `}}`, domain.StatusUnavailableNoWeekly},
		{"legacy absent limitId", `{"rateLimits":{"primary":` + weekly + `}}`, domain.StatusUnavailableNoWeekly},
		{"legacy null limitId", `{"rateLimits":{"limitId":null,"primary":` + weekly + `}}`, domain.StatusUnavailableNoWeekly},
		{"no bucket data at all", `{"ordinaryUsageAllowed":true}`, domain.StatusUnavailableNoWeekly},
		{"result null", `null`, domain.StatusUnavailableNoData},
		{"unrelated malformed bucket ignored", `{"rateLimitsByLimitId":{"broken":{"primary":"not-a-window"},"codex":{"limitId":"codex","primary":` + weekly + `}}}`, domain.StatusOK},
	}
	for _, tc := range cases {
		ginkgo.It(tc.name, func() {
			t := testT{ginkgo.GinkgoT()}
			if got := mustStatus(t, tc.raw); got != tc.want {
				t.Fatalf("status = %v, want %v", got, tc.want)
			}
		})
	}
})

var _ = ginkgo.It("TestNormalizeUnknownAdditiveFieldsIgnored", func() {
	t := testT{ginkgo.GinkgoT()}
	raw := `{"rateLimitsByLimitId":{"codex":{"limitId":"codex","primary":{"usedPercent":37,"windowDurationMins":10080,"resetsAt":1790411774,"futureField":{"x":1}},"secondary":null,"credits":{"balance":"0"}}},"brandNewTopLevel":{"a":[1,2,3]}}`
	out := normalizeQuota(json.RawMessage(raw))
	if out.Status != domain.StatusOK || out.Percent == nil || *out.Percent != 37 {
		t.Fatalf("additive fields broke parsing: %+v", out)
	}
})

var _ = ginkgo.It("TestNormalizeMalformedOptionalResetDiscarded", func() {
	t := testT{ginkgo.GinkgoT()}
	raw := weeklyBucket(`{"usedPercent":37,"windowDurationMins":10080,"resetsAt":"not-a-number"}`, "null")
	out := normalizeQuota(json.RawMessage(raw))
	if out.Status != domain.StatusOK || out.Percent == nil || *out.Percent != 37 {
		t.Fatalf("malformed optional reset poisoned result: %+v", out)
	}
	if out.ResetAt != nil {
		t.Fatalf("malformed reset retained: %v", out.ResetAt)
	}
})

var _ = ginkgo.It("TestNormalizeSignedZero", func() {
	t := testT{ginkgo.GinkgoT()}
	raw := weeklyBucket(`{"usedPercent":0,"windowDurationMins":10080}`, "null")
	out := normalizeQuota(json.RawMessage(raw))
	if out.Status != domain.StatusOK || out.Percent == nil || *out.Percent != 0 {
		t.Fatalf("genuine zero lost: %+v", out)
	}
})

func fp(v float64) *float64 { return &v }

func readFixtureRaw(t testT, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return b
}
