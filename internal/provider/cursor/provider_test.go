package cursor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"

	"gaebokchi/internal/domain"
	"gaebokchi/internal/process"
)

// Case 1a (captured — 2026-10-18 from cursor-agent 2026.10.16-e3cb207 on darwin/arm64).
const capturedCursorWeeklyJSON = `{
  "status": "authenticated",
  "isAuthenticated": true,
  "hasAccessToken": true,
  "hasRefreshToken": true,
  "hasApiKey": false,
  "planInfo": {
    "planName": "Ultra",
    "includedUsagePeriod": 2,
    "includedUsageResetsAtMs": "1792591073284",
    "billingCycleEndMs": "1794664361314"
  },
  "planError": null,
  "currentPeriodUsage": {
    "billingCycleStartMs": "1791986273284",
    "billingCycleEndMs": "1792591073284",
    "enabled": true,
    "planUsage": {
      "totalSpend": 0,
      "includedSpend": 0,
      "bonusSpend": 0,
      "remaining": 0,
      "limit": 0,
      "autoPercentUsed": 100,
      "apiPercentUsed": 0,
      "totalPercentUsed": 25
    }
  },
  "usageError": null
}`

// Case 1b (synthetic — weekly plan with includedSpend / limit * 100 fallback and spendLimitUsage exclusion).
const syntheticCursorSpendFallbackJSON = `{
  "status": "authenticated",
  "isAuthenticated": true,
  "hasAccessToken": true,
  "hasRefreshToken": true,
  "hasApiKey": false,
  "planInfo": {
    "planName": "Pro",
    "includedUsagePeriod": 2,
    "includedUsageResetsAtMs": null,
    "billingCycleEndMs": "1794664361314"
  },
  "planError": null,
  "currentPeriodUsage": {
    "billingCycleStartMs": "1791986273284",
    "billingCycleEndMs": "1792591073284",
    "enabled": true,
    "planUsage": {
      "totalSpend": 3500,
      "includedSpend": 1500,
      "bonusSpend": 0,
      "remaining": 3500,
      "limit": 5000,
      "autoPercentUsed": null,
      "apiPercentUsed": null,
      "totalPercentUsed": null
    },
    "spendLimitUsage": {
      "individualSpend": 2000,
      "individualLimit": 10000
    }
  },
  "usageError": null
}`

// Case 2 (synthetic — monthly plan -> StatusNotApplicableMonthly).
const syntheticCursorMonthlyJSON = `{
  "status": "authenticated",
  "isAuthenticated": true,
  "hasAccessToken": true,
  "hasRefreshToken": true,
  "hasApiKey": false,
  "planInfo": {
    "planName": "Pro",
    "includedUsagePeriod": 1,
    "includedUsageResetsAtMs": "1794664361314",
    "billingCycleEndMs": "1794664361314"
  },
  "planError": null,
  "currentPeriodUsage": {
    "billingCycleStartMs": "1791985961314",
    "billingCycleEndMs": "1794664361314",
    "enabled": true,
    "planUsage": {
      "totalSpend": 1000,
      "includedSpend": 1000,
      "bonusSpend": 0,
      "remaining": 1000,
      "limit": 2000,
      "totalPercentUsed": 50
    }
  },
  "usageError": null
}`

// Case 3 (synthetic — unauthenticated / missing token -> StatusAuthRequired).
const syntheticCursorUnauthenticatedJSON = `{
  "status": "unauthenticated",
  "isAuthenticated": false,
  "hasAccessToken": false,
  "hasRefreshToken": false,
  "hasApiKey": false
}`

// Case 4 (synthetic — API-key auth -> StatusNotApplicableAPIKey).
const syntheticCursorAPIKeyJSON = `{
  "status": "api_key",
  "isAuthenticated": true,
  "hasAccessToken": true,
  "hasRefreshToken": false,
  "hasApiKey": true
}`

// Case 5 (synthetic — ConnectRPC error -> StatusProviderError).
const syntheticCursorRPCErrorJSON = `{
  "status": "authenticated",
  "isAuthenticated": true,
  "hasAccessToken": true,
  "hasRefreshToken": true,
  "hasApiKey": false,
  "planInfo": null,
  "planError": {
    "name": "ConnectError",
    "code": "unavailable"
  },
  "currentPeriodUsage": null,
  "usageError": null
}`

// Case 6 (synthetic — unhooked status --format json fallback -> StatusUnsupported).
const syntheticCursorUnhookedJSON = `{
  "isAuthenticated": true,
  "email": "user@example.com",
  "loginMethod": "cursor"
}`

var _ = ginkgo.It("TestParseStatusCase1aWeeklyTotalPercentUsed", func() {
	t := testT{ginkgo.GinkgoT()}
	out := parseStatus([]byte(capturedCursorWeeklyJSON))
	wantReset := time.Date(2026, 10, 21, 13, 57, 53, 0, time.UTC)
	if out.status != domain.StatusOK || out.period != domain.PeriodWeekly ||
		out.percent == nil || *out.percent != 25 ||
		out.resetAt == nil || !out.resetAt.Equal(wantReset) {
		t.Fatalf("Case 1a outcome = %+v", out)
	}
})

var _ = ginkgo.It("TestParseStatusCase1bSpendFallbackAndUnspecifiedCycleWindow", func() {
	t := testT{ginkgo.GinkgoT()}
	out := parseStatus([]byte(syntheticCursorSpendFallbackJSON))
	wantReset := time.Date(2026, 10, 21, 13, 57, 53, 0, time.UTC)
	if out.status != domain.StatusOK || out.period != domain.PeriodWeekly ||
		out.percent == nil || *out.percent != 30 ||
		out.resetAt == nil || !out.resetAt.Equal(wantReset) {
		t.Fatalf("Case 1b outcome = %+v", out)
	}

	unspecifiedWeekly := strings.Replace(syntheticCursorSpendFallbackJSON, `"includedUsagePeriod": 2`, `"includedUsagePeriod": 0`, 1)
	out2 := parseStatus([]byte(unspecifiedWeekly))
	if out2.status != domain.StatusOK || out2.period != domain.PeriodWeekly ||
		out2.percent == nil || *out2.percent != 30 {
		t.Fatalf("Case 1b unspecified weekly outcome = %+v", out2)
	}
})

var _ = ginkgo.It("TestParseStatusCase2MonthlyAndUnspecifiedMonthlyWindow", func() {
	t := testT{ginkgo.GinkgoT()}
	out := parseStatus([]byte(syntheticCursorMonthlyJSON))
	if out.status != domain.StatusNotApplicableMonthly || out.period != domain.PeriodMonthly ||
		out.percent != nil || out.resetAt != nil {
		t.Fatalf("Case 2 monthly outcome = %+v", out)
	}

	unspecifiedMonthly := strings.Replace(syntheticCursorMonthlyJSON, `"includedUsagePeriod": 1`, `"includedUsagePeriod": 0`, 1)
	out2 := parseStatus([]byte(unspecifiedMonthly))
	if out2.status != domain.StatusNotApplicableMonthly || out2.period != domain.PeriodMonthly || out2.percent != nil {
		t.Fatalf("Case 2 unspecified monthly outcome = %+v", out2)
	}
})

var _ = ginkgo.It("TestParseStatusCases3To6AuthAPIKeyRPCErrorAndUnhooked", func() {
	t := testT{ginkgo.GinkgoT()}
	if out := parseStatus([]byte(syntheticCursorUnauthenticatedJSON)); out.status != domain.StatusAuthRequired {
		t.Fatalf("Case 3 unauthenticated = %+v, want StatusAuthRequired", out)
	}
	partialToken := `{"status":"authenticated","isAuthenticated":true,"hasAccessToken":true,"hasRefreshToken":false,"hasApiKey":false}`
	if out := parseStatus([]byte(partialToken)); out.status != domain.StatusAuthRequired {
		t.Fatalf("Case 3 missing refresh token = %+v, want StatusAuthRequired", out)
	}
	if out := parseStatus([]byte(syntheticCursorAPIKeyJSON)); out.status != domain.StatusNotApplicableAPIKey {
		t.Fatalf("Case 4 API key = %+v, want StatusNotApplicableAPIKey", out)
	}
	if out := parseStatus([]byte(syntheticCursorRPCErrorJSON)); out.status != domain.StatusProviderError {
		t.Fatalf("Case 5 RPC unavailable = %+v, want StatusProviderError", out)
	}
	rpcUnauth := `{"status":"authenticated","isAuthenticated":true,"hasAccessToken":true,"hasRefreshToken":true,"hasApiKey":false,"planInfo":null,"planError":{"name":"ConnectError","code":"unauthenticated"},"currentPeriodUsage":null,"usageError":null}`
	if out := parseStatus([]byte(rpcUnauth)); out.status != domain.StatusAuthRequired {
		t.Fatalf("Case 5 RPC unauthenticated = %+v, want StatusAuthRequired", out)
	}
	if out := parseStatus([]byte(syntheticCursorUnhookedJSON)); out.status != domain.StatusUnsupported {
		t.Fatalf("Case 6 unhooked = %+v, want StatusUnsupported", out)
	}
})

var _ = ginkgo.It("TestParseStatusCase7MalformedOutOfRangeAndUnavailable", func() {
	t := testT{ginkgo.GinkgoT()}
	for _, raw := range []string{
		`not-json`,
		capturedCursorWeeklyJSON + `{"extra":1}`,
		strings.Replace(capturedCursorWeeklyJSON, `"totalPercentUsed": 25`, `"totalPercentUsed": -0.1`, 1),
		strings.Replace(capturedCursorWeeklyJSON, `"totalPercentUsed": 25`, `"totalPercentUsed": 100.1`, 1),
		strings.Replace(syntheticCursorSpendFallbackJSON, `"includedSpend": 1500`, `"includedSpend": 6000`, 1),
		strings.Replace(syntheticCursorSpendFallbackJSON, `"includedSpend": 1500`, `"includedSpend": -1`, 1),
		strings.Replace(syntheticCursorSpendFallbackJSON, `"limit": 5000`, `"limit": -10`, 1),
	} {
		if out := parseStatus([]byte(raw)); out.status != domain.StatusProtocolError {
			t.Fatalf("expected StatusProtocolError for %q, got %+v", raw, out)
		}
	}

	for _, raw := range []string{
		strings.Replace(capturedCursorWeeklyJSON, `"includedUsagePeriod": 2`, `"includedUsagePeriod": 99`, 1),
		strings.Replace(capturedCursorWeeklyJSON, `"includedUsagePeriod": 2`, `"includedUsagePeriod": 0`, 1) + "",
		`{"status":"authenticated","isAuthenticated":true,"hasAccessToken":true,"hasRefreshToken":true,"hasApiKey":false,"planInfo":null,"planError":null,"currentPeriodUsage":null,"usageError":null}`,
	} {
		if strings.Contains(raw, `"includedUsagePeriod": 0`) {
			// Replace cycle end with 15 days so it is neither weekly nor monthly.
			raw = strings.Replace(raw, `"billingCycleEndMs": "1792591073284"`, `"billingCycleEndMs": "1793282273284"`, 1)
		}
		if out := parseStatus([]byte(raw)); out.status != domain.StatusUnavailablePeriod {
			t.Fatalf("expected StatusUnavailablePeriod for %q, got %+v", raw, out)
		}
	}

	for _, raw := range []string{
		strings.Replace(capturedCursorWeeklyJSON, `"enabled": true`, `"enabled": false`, 1),
		strings.Replace(capturedCursorWeeklyJSON, `"totalPercentUsed": 25`, `"totalPercentUsed": null`, 1),
		`{"status":"authenticated","isAuthenticated":true,"hasAccessToken":true,"hasRefreshToken":true,"hasApiKey":false,"planInfo":{"planName":"Ultra","includedUsagePeriod":2},"planError":null,"currentPeriodUsage":null,"usageError":null}`,
	} {
		out := parseStatus([]byte(raw))
		if out.status != domain.StatusUnavailableNoData || out.period != domain.PeriodWeekly {
			t.Fatalf("expected weekly StatusUnavailableNoData for %q, got %+v", raw, out)
		}
	}
})

func fakeCursor(t testT, payload string) (bin, argLog, envLog, preloadCheckLog string) {
	t.Helper()
	dir := t.TempDir()
	bin = filepath.Join(dir, "cursor-agent")
	argLog = filepath.Join(dir, "args.log")
	envLog = filepath.Join(dir, "env.log")
	preloadCheckLog = filepath.Join(dir, "preload.log")
	payloadFile := filepath.Join(dir, "payload.json")
	if err := os.WriteFile(payloadFile, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> " + argLog + "\n" +
		"env >> " + envLog + "\n" +
		"req=${NODE_OPTIONS#--require=}\n" +
		"req=${req#\\\"}\n" +
		"req=${req%\\\"}\n" +
		"if [ -f \"$req\" ]; then printf 'exists:%s\\n' \"$req\" > " + preloadCheckLog + "; fi\n" +
		"cat " + payloadFile + "\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return bin, argLog, envLog, preloadCheckLog
}

var _ = ginkgo.It("TestQueryExecutesStatusWithPreloadHookFiltersEnvAndCleansTempDir", func() {
	t := testT{ginkgo.GinkgoT()}
	t.Setenv("HOME", "/Users/tester")
	t.Setenv("CURSOR_API_KEY", "crsr-secret-marker")
	t.Setenv("CURSOR_AUTH_TOKEN", "crsr-token-marker")
	t.Setenv("CURSOR_CONFIG_DIR", "/tmp/cursor-override")
	t.Setenv("NODE_OPTIONS", "--inspect")

	bin, argLog, envLog, preloadCheckLog := fakeCursor(t, capturedCursorWeeklyJSON)
	p := New()
	p.TempParent = filepath.Join(t.TempDir(), "tmp with space")
	if err := os.MkdirAll(p.TempParent, 0o700); err != nil {
		t.Fatal(err)
	}
	tgt := domain.Target{
		ID: "cursor:0", Provider: domain.ProviderCursor, FallbackAccount: "default", Bin: bin,
	}
	res := p.Query(context.Background(), tgt)
	if res.Status != domain.StatusOK || res.UsedPercent == nil || *res.UsedPercent != 25 ||
		res.Account != "default" || res.Period != domain.PeriodWeekly ||
		res.QuotaID != QuotaIncluded || res.Source != domain.SourceCursorCLIUsage || res.CleanupFailed {
		t.Fatalf("unexpected cursor:0 result: %+v", res)
	}
	args, err := os.ReadFile(argLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(args)) != "status --format json" {
		t.Fatalf("unexpected cursor argv: %q", args)
	}
	preloadCheck, err := os.ReadFile(preloadCheckLog)
	if err != nil || !strings.HasPrefix(strings.TrimSpace(string(preloadCheck)), "exists:") {
		t.Fatalf("preload file was not present during child execution: %v / %q", err, preloadCheck)
	}
	preloadPath := strings.TrimPrefix(strings.TrimSpace(string(preloadCheck)), "exists:")
	if _, statErr := os.Stat(filepath.Dir(preloadPath)); !os.IsNotExist(statErr) {
		t.Fatalf("preload directory %q survived after Query returned: %v", filepath.Dir(preloadPath), statErr)
	}
	envBytes, err := os.ReadFile(envLog)
	if err != nil {
		t.Fatal(err)
	}
	envStr := "\n" + string(envBytes)
	if strings.Contains(envStr, "\nCURSOR_API_KEY=") ||
		strings.Contains(envStr, "\nCURSOR_AUTH_TOKEN=") ||
		strings.Contains(envStr, "\nCURSOR_CONFIG_DIR=") ||
		strings.Contains(envStr, "--inspect") ||
		!strings.Contains(envStr, "\nHOME=/Users/tester\n") ||
		!strings.Contains(envStr, "\nNODE_OPTIONS=--require="+strconv.Quote(preloadPath)+"\n") {
		t.Fatalf("child environment was not filtered properly:\n%s", envStr)
	}
})

var _ = ginkgo.It("TestQueryHandlesMissingCLITimeoutOversizedOutputAndCleanupFailure", func() {
	t := testT{ginkgo.GinkgoT()}
	p := New()
	missing := p.Query(context.Background(), domain.Target{
		ID: "cursor:0", Provider: domain.ProviderCursor, FallbackAccount: "default",
		Bin: filepath.Join(t.TempDir(), "no-such-cursor-agent"),
	})
	if missing.Status != domain.StatusCLINotFound {
		t.Fatalf("missing CLI status = %v, want StatusCLINotFound", missing.Status)
	}

	dir := t.TempDir()
	sleepBin := filepath.Join(dir, "cursor-agent-sleep")
	if err := os.WriteFile(sleepBin, []byte("#!/bin/sh\nexec sleep 10\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	timedOut := p.Query(ctx, domain.Target{
		ID: "cursor:0", Provider: domain.ProviderCursor, FallbackAccount: "default", Bin: sleepBin,
	})
	if timedOut.Status != domain.StatusTimeout {
		t.Fatalf("timeout status = %v, want StatusTimeout", timedOut.Status)
	}

	hugeBin := filepath.Join(dir, "cursor-agent-huge")
	if err := os.WriteFile(hugeBin, []byte("#!/bin/sh\nhead -c 300000 /dev/zero | tr '\\0' 'a'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	oversized := p.Query(context.Background(), domain.Target{
		ID: "cursor:0", Provider: domain.ProviderCursor, FallbackAccount: "default", Bin: hugeBin,
	})
	if oversized.Status != domain.StatusProtocolError {
		t.Fatalf("oversized status = %v, want StatusProtocolError", oversized.Status)
	}

	bin, _, _, _ := fakeCursor(t, capturedCursorWeeklyJSON)
	pFail := &Provider{
		Start: func(cfg process.Config) (*process.Child, error) {
			if os.Getuid() != 0 {
				parent := t.TempDir()
				cfg.TempParent = parent
				c, err := process.Start(cfg)
				if err == nil {
					_ = os.Chmod(parent, 0o500)
					t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })
				}
				return c, err
			}
			return process.Start(cfg)
		},
	}
	if os.Getuid() != 0 {
		res := pFail.Query(context.Background(), domain.Target{
			ID: "cursor:0", Provider: domain.ProviderCursor, FallbackAccount: "default", Bin: bin,
		})
		if res.Status != domain.StatusOK || !res.CleanupFailed || !strings.Contains(res.Diagnostic.Message, "provider cleanup incomplete") {
			t.Fatalf("cleanup failure was not recorded on result: %+v", res)
		}
	}
})

var _ = ginkgo.It("TestPreloadScriptExecutesAndStripsPIIInNode", func() {
	t := testT{ginkgo.GinkgoT()}
	nodeBin, err := exec.LookPath("node")
	if err != nil {
		ginkgo.Skip("node not on PATH")
	}
	dir := filepath.Join(t.TempDir(), "tmp with space")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	preloadPath := filepath.Join(dir, "preload.cjs")
	if err := os.WriteFile(preloadPath, preloadScript, 0o600); err != nil {
		t.Fatal(err)
	}
	chunkPath := filepath.Join(dir, "chunk.cjs")
	chunkCode := `"use strict";
exports.modules = {
  "./src/commands/status.ts": (e, t, a) => {
    a.d(t, { handleStatus: () => origHandleStatus });
    async function origHandleStatus() {
      process.stdout.write('{"unhooked":true}\n');
      return 0;
    }
  }
};
`
	if err := os.WriteFile(chunkPath, []byte(chunkCode), 0o600); err != nil {
		t.Fatal(err)
	}
	driverPath := filepath.Join(dir, "driver.cjs")
	driverCode := `"use strict";
const chunk = require("./chunk.cjs");
const exp = {};
const webpackRequire = {
  d(target, defs) {
    for (const k of Object.keys(defs)) {
      Object.defineProperty(target, k, { enumerable: true, get: defs[k] });
    }
  }
};
chunk.modules["./src/commands/status.ts"]({}, exp, webpackRequire);
const credMgr = {
  getAccessToken: async () => "secret-access-token",
  getRefreshToken: async () => "secret-refresh-token",
  getApiKey: async () => null,
};
const dashClient = {
  getPlanInfo: async () => ({
    planInfo: {
      planName: "Ultra",
      includedUsagePeriod: 2,
      includedUsageResetsAt: BigInt("1792591073284"),
      billingCycleEnd: BigInt("1794664361314"),
      email: "pii-secret@example.com",
      userId: "user-pii-123"
    }
  }),
  getCurrentPeriodUsage: async () => ({
    billingCycleStart: BigInt("1791986273284"),
    billingCycleEnd: BigInt("1792591073284"),
    enabled: true,
    planUsage: JSON.parse(process.argv[2]),
    spendLimitUsage: {
      individualSpend: 9999
    }
  })
};
setInterval(() => {}, 60000);
exp.handleStatus(credMgr, dashClient, { format: "json" });
`
	if err := os.WriteFile(driverPath, []byte(driverCode), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name      string
		planUsage string
		status    domain.Status
		percent   float64
	}{
		{"reported percentage", "{\"totalPercentUsed\":25,\"includedSpend\":0,\"limit\":0}", domain.StatusOK, 25},
		{"spend fallback", "{\"includedSpend\":3,\"limit\":10}", domain.StatusOK, 30},
		{"measured zero", "{\"includedSpend\":0,\"limit\":10}", domain.StatusOK, 0},
		{"missing spend", "{\"limit\":10}", domain.StatusUnavailableNoData, 0},
		{"nonnumeric spend", "{\"includedSpend\":\"not-numeric\",\"limit\":10}", domain.StatusUnavailableNoData, 0},
		{"reported percentage without spend", "{\"totalPercentUsed\":25,\"limit\":10}", domain.StatusOK, 25},
	} {
		cmd := exec.Command(nodeBin, driverPath, tc.planUsage)
		cmd.Env = childEnv(preloadPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: node execution failed: %v: %s", tc.name, err, out)
		}
		if strings.Contains(string(out), "pii-secret") || strings.Contains(string(out), "secret-access-token") ||
			strings.Contains(string(out), "spendLimitUsage") {
			t.Fatalf("%s: PII, token, or spendLimitUsage leaked from preload.cjs: %s", tc.name, out)
		}
		parsed := parseStatus(out)
		if parsed.status != tc.status {
			t.Fatalf("%s: parsed outcome = %+v, want %s (%s)", tc.name, parsed, tc.status, out)
		}
		if tc.status == domain.StatusOK {
			if parsed.percent == nil || *parsed.percent != tc.percent {
				t.Fatalf("%s: percentage = %v, want %v", tc.name, parsed.percent, tc.percent)
			}
		} else if parsed.percent != nil {
			t.Fatalf("%s: unavailable usage has a measured percentage: %v", tc.name, *parsed.percent)
		}
	}
})

var _ = ginkgo.It("reports a nonzero cursor CLI exit without fabricating usage", func() {
	t := testT{ginkgo.GinkgoT()}
	bin := filepath.Join(t.TempDir(), "cursor-fail")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result := New().Query(ctx, domain.Target{ID: "cursor:0", Provider: domain.ProviderCursor, FallbackAccount: "default", Bin: bin})
	if result.Status != domain.StatusProviderError || result.Diagnostic.Code != "status_exit_error" || result.UsedPercent != nil || result.CleanupFailed {
		t.Fatalf("nonzero CLI exit = %+v", result)
	}
})
