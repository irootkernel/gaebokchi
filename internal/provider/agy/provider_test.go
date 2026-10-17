package agy

import (
	"context"
	ginkgo "github.com/onsi/ginkgo/v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gaebokchi/internal/domain"
)

const capturedAgyUsageJSON = `{
  "status": "SUCCESS",
  "response": "",
  "num_turns": 0,
  "conversation_id": "",
  "duration_seconds": 5.1,
  "usage": {
    "input_tokens": 0,
    "output_tokens": 0,
    "total_tokens": 0,
    "cache_read_tokens": 0,
    "thinking_tokens": 0
  },
  "command": {
    "name": "usage",
    "data": {
      "description": "Within each group, models share a weekly limit and a 5-hour limit.",
      "groups": [
        {
          "name": "Gemini Models",
          "description": "Models within this group: Gemini Flash, Gemini Pro",
          "buckets": [
            {
              "id": "gemini-weekly",
              "name": "Weekly Limit Remaining",
              "window": "weekly",
              "remaining_fraction": 0.2584281265735626,
              "reset_time": "2026-10-21T15:42:40Z"
            },
            {
              "id": "gemini-5h",
              "name": "Five Hour Limit Remaining",
              "window": "5h",
              "remaining_fraction": 0.8503220677375793,
              "reset_time": "2026-10-16T18:01:19Z"
            }
          ]
        },
        {
          "name": "Claude and GPT models",
          "description": "Models within this group: Claude Opus, Claude Sonnet, GPT-OSS",
          "buckets": [
            {
              "id": "3p-weekly",
              "name": "Weekly Limit Remaining",
              "window": "weekly",
              "remaining_fraction": 0.8723080158233643,
              "reset_time": "2026-10-23T15:13:44Z"
            },
            {
              "id": "3p-5h",
              "name": "Five Hour Limit Remaining",
              "window": "5h",
              "remaining_fraction": 0.7446160912513733,
              "reset_time": "2026-10-16T20:13:44Z"
            }
          ]
        }
      ]
    }
  }
}`

var _ = ginkgo.Describe("TestIsSupportedVersion", func() {
	cases := []struct {
		raw  string
		want bool
	}{
		{"1.2.15\n", true},
		{"1.4.0\n", true},
		{"v1.3.0", true},
		{"agy version 2.0.0", true},
		{"1.2.14\n", false},
		{"1.1.11\n", false},
		{"0.9.9\n", false},
		{"unknown", false},
	}
	for _, tc := range cases {
		ginkgo.It(tc.raw, func() {
			t := testT{ginkgo.GinkgoT()}
			if got := isSupportedVersion(tc.raw); got != tc.want {
				t.Fatalf("isSupportedVersion(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
})

var _ = ginkgo.It("TestParseUsageExtractsBothWeeklyPoolsAndIgnores5h", func() {
	t := testT{ginkgo.GinkgoT()}
	pools, topStatus, _, _ := parseUsage([]byte(capturedAgyUsageJSON))
	if topStatus != "" {
		t.Fatalf("unexpected top status: %v", topStatus)
	}
	gem := pools[QuotaGemini]
	if gem.status != domain.StatusOK || gem.percent == nil || *gem.percent != 74.15718734264374 ||
		gem.resetAt == nil || !gem.resetAt.Equal(time.Date(2026, 10, 21, 15, 42, 40, 0, time.UTC)) {
		t.Fatalf("gemini-weekly outcome = %+v", gem)
	}
	claude := pools[QuotaClaudeGPT]
	if claude.status != domain.StatusOK || claude.percent == nil || *claude.percent != 12.76919841766357 ||
		claude.resetAt == nil || !claude.resetAt.Equal(time.Date(2026, 10, 23, 15, 13, 44, 0, time.UTC)) {
		t.Fatalf("3p-weekly outcome = %+v", claude)
	}
})

var _ = ginkgo.It("TestParseUsagePreservesSiblingWhenOnePoolMissingOrMalformed", func() {
	t := testT{ginkgo.GinkgoT()}
	missing3P := `{
	  "type":"result","status":"SUCCESS","num_turns":0,
	  "command":{"name":"usage","data":{"groups":[{"quotas":[
	    {"id":"gemini-weekly","window":"weekly","remaining_fraction":0.9,"reset_time":"not-a-timestamp"}
	  ]}]}}
	}`
	pools, topStatus, _, _ := parseUsage([]byte(missing3P))
	if topStatus != "" {
		t.Fatalf("unexpected top status: %v", topStatus)
	}
	if pools[QuotaGemini].status != domain.StatusOK || pools[QuotaGemini].percent == nil || *pools[QuotaGemini].percent != 10 || pools[QuotaGemini].resetAt != nil {
		t.Fatalf("gemini-weekly outcome = %+v", pools[QuotaGemini])
	}
	if pools[QuotaClaudeGPT].status != domain.StatusUnavailableNoWeekly {
		t.Fatalf("missing 3p-weekly status = %v, want %v", pools[QuotaClaudeGPT].status, domain.StatusUnavailableNoWeekly)
	}

	malformedGemini := `{
	  "type":"result","status":"SUCCESS","num_turns":0,
	  "command":{"name":"usage","data":{"groups":[{"quotas":[
	    {"id":"gemini-weekly","window":"weekly","remaining_fraction":1e-1000000000},
	    {"id":"3p-weekly","window":"weekly","remaining_fraction":0.9}
	  ]}]}}
	}`
	pools2, topStatus2, _, _ := parseUsage([]byte(malformedGemini))
	if topStatus2 != "" {
		t.Fatalf("unexpected top status: %v", topStatus2)
	}
	if pools2[QuotaGemini].status != domain.StatusProtocolError {
		t.Fatalf("malformed gemini-weekly status = %v, want protocol_error", pools2[QuotaGemini].status)
	}
	if pools2[QuotaClaudeGPT].status != domain.StatusOK || pools2[QuotaClaudeGPT].percent == nil || *pools2[QuotaClaudeGPT].percent != 10 {
		t.Fatalf("sibling 3p-weekly was not preserved: %+v", pools2[QuotaClaudeGPT])
	}
})

var _ = ginkgo.It("TestParseUsageRejectsInvalidEnvelopeOrTurns", func() {
	t := testT{ginkgo.GinkgoT()}
	if _, status, _, _ := parseUsage([]byte(`not-json`)); status != domain.StatusProtocolError {
		t.Fatalf("invalid JSON status = %v, want protocol_error", status)
	}
	withTurn := `{"type":"result","status":"SUCCESS","num_turns":1,"command":{"name":"usage","data":{"groups":[]}}}`
	if _, status, _, _ := parseUsage([]byte(withTurn)); status != domain.StatusUnsupported {
		t.Fatalf("non-zero turn status = %v, want unsupported", status)
	}
})

func fakeAgy(t testT, version, usage string) (bin, log string) {
	t.Helper()
	dir := t.TempDir()
	bin = filepath.Join(dir, "agy")
	log = filepath.Join(dir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + log + "\n" +
		"case \" $* \" in\n" +
		"  *' --version '*) printf '%s\\n' '" + version + "' ;;\n" +
		"  *) printf '%s' '" + usage + "' ;;\n" +
		"esac\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return bin, log
}

func agyTargets(bin string) (domain.Target, domain.Target) {
	return domain.Target{
		ID: "agy:0", Provider: domain.ProviderAgy, FallbackAccount: AccountGemini, Bin: bin,
	}, domain.Target{
		ID: "agy:1", Provider: domain.ProviderAgy, FallbackAccount: AccountClaudeGP, Bin: bin,
	}
}

var _ = ginkgo.It("TestQueryExecutesCLIOnceForBothWeeklyPools", func() {
	t := testT{ginkgo.GinkgoT()}
	bin, log := fakeAgy(t, "1.4.0", capturedAgyUsageJSON)
	p := New()
	t0, t1 := agyTargets(bin)
	r0 := p.Query(context.Background(), t0)
	r1 := p.Query(context.Background(), t1)
	if r0.Status != domain.StatusOK || r0.UsedPercent == nil || *r0.UsedPercent != 74.15718734264374 ||
		r0.Account != "gemini" || r0.QuotaID != QuotaGemini || r0.Source != domain.SourceAgyCLIUsage {
		t.Fatalf("unexpected agy:0 result: %+v", r0)
	}
	if r1.Status != domain.StatusOK || r1.UsedPercent == nil || *r1.UsedPercent != 12.76919841766357 ||
		r1.Account != "claude/gpt" || r1.QuotaID != QuotaClaudeGPT || r1.Source != domain.SourceAgyCLIUsage {
		t.Fatalf("unexpected agy:1 result: %+v", r1)
	}
	args, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(args)), "\n")
	if len(lines) != 2 || lines[0] != "--version" ||
		lines[1] != "--print /usage --output-format json --print-timeout 15s" {
		t.Fatalf("expected exactly one --version and one /usage call, got: %q", args)
	}
})

var _ = ginkgo.It("TestQueryVersionGuardBlocksOldCLIWithoutRunningUsage", func() {
	t := testT{ginkgo.GinkgoT()}
	bin, log := fakeAgy(t, "1.2.14", capturedAgyUsageJSON)
	p := New()
	t0, t1 := agyTargets(bin)
	r0 := p.Query(context.Background(), t0)
	r1 := p.Query(context.Background(), t1)
	if r0.Status != domain.StatusUnsupported || r1.Status != domain.StatusUnsupported {
		t.Fatalf("expected StatusUnsupported for both rows, got %v and %v", r0.Status, r1.Status)
	}
	args, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(args)) != "--version" {
		t.Fatalf("old CLI must not reach /usage, got log: %q", args)
	}
})

var _ = ginkgo.It("TestQueryFastFailsWhenStderrReportsAuthenticationRequired", func() {
	t := testT{ginkgo.GinkgoT()}
	dir := t.TempDir()
	bin := filepath.Join(dir, "agy")
	script := `#!/bin/sh
case " $* " in
  *' --version '*) printf '1.4.0\n' ;;
  *)
    printf 'Authentication required. Please visit the URL to log in.\nWaiting for authentication (timeout 60s)...\n' >&2
    exec sleep 10
    ;;
esac
`
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	p := New()
	t0, t1 := agyTargets(bin)
	r0 := p.Query(context.Background(), t0)
	r1 := p.Query(context.Background(), t1)
	elapsed := time.Since(start)
	if r0.Status != domain.StatusAuthRequired || r1.Status != domain.StatusAuthRequired {
		t.Fatalf("expected StatusAuthRequired, got %v and %v", r0.Status, r1.Status)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("auth fast-fail took too long: %v", elapsed)
	}
})

var _ = ginkgo.It("TestChildEnvironmentFiltersOverridesAndKeepsHome", func() {
	t := testT{ginkgo.GinkgoT()}
	t.Setenv("HOME", "/Users/tester")
	t.Setenv("ANTIGRAVITY_LS_ADDRESS", "127.0.0.1:9999")
	t.Setenv("JETSKI_APP_DATA_DIR", "/tmp/jetski")
	t.Setenv("AGY_ADC_AUTH", "1")
	t.Setenv("GEMINI_API_KEY", "secret")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/tmp/creds.json")
	env := childEnv()
	var sawHome, sawDisableUpdate bool
	for _, kv := range env {
		if kv == "HOME=/Users/tester" {
			sawHome = true
		}
		if kv == "AGY_CLI_DISABLE_AUTO_UPDATE=true" {
			sawDisableUpdate = true
		}
		if strings.HasPrefix(kv, "ANTIGRAVITY_") || strings.HasPrefix(kv, "JETSKI_") ||
			strings.HasPrefix(kv, "GEMINI_API_KEY=") || strings.HasPrefix(kv, "GOOGLE_APPLICATION_CREDENTIALS=") ||
			strings.HasPrefix(kv, "AGY_ADC_AUTH=") {
			t.Fatalf("override leaked to agy child env: %q", kv)
		}
	}
	if !sawHome || !sawDisableUpdate {
		t.Fatalf("missing HOME or AGY_CLI_DISABLE_AUTO_UPDATE in env: %v", env)
	}
})
