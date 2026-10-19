package zcode

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

const syntheticZCodeFiveHourJSON = `{
  "status": "authenticated",
  "isAuthenticated": true,
  "family": "zai",
  "quota": {
    "httpStatus": 200,
    "code": 0,
    "success": true,
    "limits": [
      {
        "type": "TOKENS_LIMIT",
        "unit": 3,
        "number": 5,
        "percentage": 42,
        "usage": 1000000,
        "currentValue": 420000,
        "remaining": 580000,
        "nextResetTime": "1792591073284"
      },
      {
        "type": "TIME_LIMIT",
        "unit": 5,
        "number": 1,
        "percentage": 10,
        "usage": 100,
        "currentValue": 10,
        "remaining": 90,
        "nextResetTime": null
      }
    ]
  },
  "quotaError": null
}`

const syntheticZCodeFallbackCurrentValueJSON = `{
  "status": "authenticated",
  "isAuthenticated": true,
  "family": "bigmodel",
  "quota": {
    "httpStatus": 200,
    "code": 200,
    "success": true,
    "limits": [
      {
        "type": "CREDIT_LIMIT",
        "unit": 3,
        "number": 5,
        "percentage": null,
        "usage": 200,
        "currentValue": 50,
        "remaining": 150,
        "nextResetTime": "1792591073284"
      }
    ]
  },
  "quotaError": null
}`

const syntheticZCodeUnauthenticatedJSON = `{
  "status": "unauthenticated",
  "isAuthenticated": false
}`

const syntheticZCodeUnhookedDoctorJSON = `{
  "cli": { "name": "zcode", "version": "0.16.9" },
  "runtime": { "platform": "darwin", "arch": "arm64" }
}`

var _ = ginkgo.It("TestParseStatusFiveHourPercentageAndFallback", func() {
	t := testT{ginkgo.GinkgoT()}
	out := parseStatus([]byte(syntheticZCodeFiveHourJSON))
	wantReset := time.Date(2026, 10, 21, 13, 57, 53, 0, time.UTC)
	if out.status != domain.StatusOK || out.period != domain.PeriodFiveHour ||
		out.percent == nil || *out.percent != 42 ||
		out.resetAt == nil || !out.resetAt.Equal(wantReset) {
		t.Fatalf("5h outcome = %+v", out)
	}

	outFallback := parseStatus([]byte(syntheticZCodeFallbackCurrentValueJSON))
	if outFallback.status != domain.StatusOK || outFallback.period != domain.PeriodFiveHour ||
		outFallback.percent == nil || *outFallback.percent != 25 ||
		outFallback.resetAt == nil || !outFallback.resetAt.Equal(wantReset) {
		t.Fatalf("5h fallback outcome = %+v", outFallback)
	}

	remainingFallback := strings.Replace(syntheticZCodeFallbackCurrentValueJSON, `"currentValue": 50`, `"currentValue": null`, 1)
	outRem := parseStatus([]byte(remainingFallback))
	if outRem.status != domain.StatusOK || outRem.percent == nil || *outRem.percent != 25 {
		t.Fatalf("5h remaining fallback outcome = %+v", outRem)
	}
})

var _ = ginkgo.It("TestParseStatusAuthErrorsUnsupportedAndMalformed", func() {
	t := testT{ginkgo.GinkgoT()}
	if out := parseStatus([]byte(syntheticZCodeUnauthenticatedJSON)); out.status != domain.StatusAuthRequired {
		t.Fatalf("unauthenticated = %+v, want StatusAuthRequired", out)
	}
	if out := parseStatus([]byte(`{"status":"authenticated","isAuthenticated":true,"quota":null,"quotaError":{"name":"Error","code":"auth_required"}}`)); out.status != domain.StatusAuthRequired {
		t.Fatalf("quotaError auth_required = %+v, want StatusAuthRequired", out)
	}
	if out := parseStatus([]byte(`{"status":"authenticated","isAuthenticated":true,"quota":{"httpStatus":401,"code":401,"limits":null},"quotaError":null}`)); out.status != domain.StatusAuthRequired {
		t.Fatalf("HTTP 401 = %+v, want StatusAuthRequired", out)
	}
	if out := parseStatus([]byte(`{"status":"authenticated","isAuthenticated":true,"quota":null,"quotaError":{"name":"Error","code":"timeout"}}`)); out.status != domain.StatusTimeout {
		t.Fatalf("quotaError timeout = %+v, want StatusTimeout", out)
	}
	if out := parseStatus([]byte(`{"status":"authenticated","isAuthenticated":true,"quota":null,"quotaError":{"name":"Error","code":"http_500"}}`)); out.status != domain.StatusProviderError {
		t.Fatalf("quotaError http_500 = %+v, want StatusProviderError", out)
	}
	if out := parseStatus([]byte(syntheticZCodeUnhookedDoctorJSON)); out.status != domain.StatusUnsupported {
		t.Fatalf("unhooked doctor JSON = %+v, want StatusUnsupported", out)
	}
	if out := parseStatus([]byte(`{"status":"authenticated","isAuthenticated":true,"quota":{"httpStatus":200,"code":0,"limits":[]},"quotaError":null}`)); out.status != domain.StatusUnavailableNoData {
		t.Fatalf("empty limits = %+v, want StatusUnavailableNoData", out)
	}
	oneHourLimit := strings.Replace(syntheticZCodeFiveHourJSON, `"number": 5`, `"number": 1`, 1)
	if out := parseStatus([]byte(oneHourLimit)); out.status != domain.StatusUnavailableNoData {
		t.Fatalf("unit=3 number=1 (1h limit) = %+v, want StatusUnavailableNoData", out)
	}
	for _, bad := range []string{
		`not-json`,
		syntheticZCodeFiveHourJSON + `{"extra":1}`,
		strings.Replace(syntheticZCodeFiveHourJSON, `"percentage": 42`, `"percentage": -1`, 1),
		strings.Replace(syntheticZCodeFiveHourJSON, `"percentage": 42`, `"percentage": 100.5`, 1),
	} {
		if out := parseStatus([]byte(bad)); out.status != domain.StatusProtocolError {
			t.Fatalf("expected StatusProtocolError for %q, got %+v", bad, out)
		}
	}
})

func fakeZCode(t testT, payload string) (bin, argLog, envLog, preloadCheckLog string) {
	t.Helper()
	dir := t.TempDir()
	bin = filepath.Join(dir, "zcode")
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

var _ = ginkgo.It("TestQueryExecutesDoctorWithPreloadHookFiltersEnvAndCleansTempDir", func() {
	t := testT{ginkgo.GinkgoT()}
	t.Setenv("HOME", "/Users/tester")
	t.Setenv("ZCODE_DATA_BASE_DIR", "/tmp/zcode-override")
	t.Setenv("ZCODE_CREDENTIAL_SECRET", "zcode-secret")
	t.Setenv("ZAI_BUSINESS_BASE_URL", "https://override.invalid")
	t.Setenv("BIGMODEL_API_ORIGIN", "https://override.invalid")
	t.Setenv("NODE_OPTIONS", "--inspect")

	bin, argLog, envLog, preloadCheckLog := fakeZCode(t, syntheticZCodeFiveHourJSON)
	p := New()
	p.TempParent = filepath.Join(t.TempDir(), "tmp with space")
	if err := os.MkdirAll(p.TempParent, 0o700); err != nil {
		t.Fatal(err)
	}
	tgt := domain.Target{
		ID: "zcode:0", Provider: domain.ProviderZCode, FallbackAccount: "default", Bin: bin,
	}
	res := p.Query(context.Background(), tgt)
	if res.Status != domain.StatusOK || res.UsedPercent == nil || *res.UsedPercent != 42 ||
		res.Account != "default" || res.Period != domain.PeriodFiveHour ||
		res.QuotaID != QuotaFiveHour || res.Source != domain.SourceZCodeCLIUsage || res.CleanupFailed {
		t.Fatalf("unexpected zcode:0 result: %+v", res)
	}
	args, err := os.ReadFile(argLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(args)) != "doctor --json" {
		t.Fatalf("unexpected zcode argv: %q", args)
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
	if strings.Contains(envStr, "\nZCODE_DATA_BASE_DIR=") ||
		strings.Contains(envStr, "\nZCODE_CREDENTIAL_SECRET=") ||
		strings.Contains(envStr, "\nZAI_BUSINESS_BASE_URL=") ||
		strings.Contains(envStr, "\nBIGMODEL_API_ORIGIN=") ||
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
		ID: "zcode:0", Provider: domain.ProviderZCode, FallbackAccount: "default",
		Bin: filepath.Join(t.TempDir(), "no-such-zcode"),
	})
	if missing.Status != domain.StatusCLINotFound {
		t.Fatalf("missing CLI status = %v, want StatusCLINotFound", missing.Status)
	}

	dir := t.TempDir()
	sleepBin := filepath.Join(dir, "zcode-sleep")
	if err := os.WriteFile(sleepBin, []byte("#!/bin/sh\nexec sleep 10\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	timedOut := p.Query(ctx, domain.Target{
		ID: "zcode:0", Provider: domain.ProviderZCode, FallbackAccount: "default", Bin: sleepBin,
	})
	if timedOut.Status != domain.StatusTimeout {
		t.Fatalf("timeout status = %v, want StatusTimeout", timedOut.Status)
	}

	hugeBin := filepath.Join(dir, "zcode-huge")
	if err := os.WriteFile(hugeBin, []byte("#!/bin/sh\nhead -c 300000 /dev/zero\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	limitCtx, limitCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer limitCancel()
	oversized := p.Query(limitCtx, domain.Target{ID: "zcode:0", Provider: domain.ProviderZCode, FallbackAccount: "default", Bin: hugeBin})
	if oversized.Status != domain.StatusProtocolError || oversized.UsedPercent != nil || oversized.CleanupFailed {
		t.Fatalf("oversized child output = %+v", oversized)
	}

	bin, _, _, _ := fakeZCode(t, syntheticZCodeFiveHourJSON)
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
			ID: "zcode:0", Provider: domain.ProviderZCode, FallbackAccount: "default", Bin: bin,
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
	dir := t.TempDir()
	preloadDir := filepath.Join(dir, "preload with space")
	if err := os.MkdirAll(preloadDir, 0o700); err != nil {
		t.Fatal(err)
	}
	preloadPath := filepath.Join(preloadDir, "preload.cjs")
	if err := os.WriteFile(preloadPath, preloadScript, 0o600); err != nil {
		t.Fatal(err)
	}
	driverPath := filepath.Join(dir, "driver.cjs")
	driverCode := `"use strict";
const defProp = Object.defineProperty;
const r = (fn, name) => defProp(fn, "name", { value: name, configurable: true });

let diskMutated = false;
const createStore = r(() => ({
  filePath: "/tmp/nonexistent-gaebokchi-zcode-credentials.json",
  load: async (k) => k === "oauth:active_provider" ? "zai" : "secret-coding-plan-token",
  loadMany: async (keys) => Object.fromEntries(keys.map((k) => [k, "secret-coding-plan-token"])),
  save: async () => { diskMutated = true; },
  saveMany: async () => { diskMutated = true; },
}), "createSharedZCodeCredentialStore");

const readProviders = r(async () => [
  { family: "zai", providerId: "account:zai-individual-coding-plan", modelId: "glm-5.1" }
], "readStandaloneCodingPlanProviders");

const readSnapshot = r(async () => {
  if (diskMutated) {
    throw new Error("credential store mutated on disk");
  }
  return {
    providers: new Map([
      ["account:zai-individual-coding-plan", { access: { entitled: true } }]
    ])
  };
}, "readStandaloneAccountProviderConfigSnapshot");

const resolveEndpoints = r(() => ({
  zaiBusinessBaseUrl: "https://api.z.ai",
  bigModelApiOrigin: "https://bigmodel.cn"
}), "resolveRuntimeProductEndpointConfig");

const createHttpClient = r(() => ({
  request: async (req) => {
    if (req.method !== "GET" || !req.url.endsWith("/api/monitor/usage/quota/limit")) {
      throw new Error("unexpected request: " + req.method + " " + req.url);
    }
    return {
      status: 200,
      body: Buffer.from(JSON.stringify({
        code: 0,
        success: true,
        data: {
          email: "pii-secret@example.com",
          userId: "user-pii-123",
          limits: [
            {
              type: "TOKENS_LIMIT",
              unit: 3,
              number: 5,
              percentage: 37.5,
              usage: 1000,
              currentValue: 375,
              remaining: 625,
              nextResetTime: 1792591073284,
              secretToken: "must-not-leak"
            }
          ]
        }
      }), "utf8")
    };
  }
}), "createDefaultHttpClient");

const loadBootstrap = r(async () => ({
  createStore, readProviders, readSnapshot, resolveEndpoints, createHttpClient
}), "loadBootstrapModule");

const runDoctor = r((io, opts) => {
  io.stdout.write('{"unhooked":true}\n');
  return 0;
}, "runDoctor");

setInterval(() => {}, 60000);
runDoctor({ stdout: process.stdout, stderr: process.stderr }, { json: true });
`
	if err := os.WriteFile(driverPath, []byte(driverCode), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(nodeBin, driverPath)
	cmd.Env = childEnv(preloadPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node execution failed: %v: %s", err, out)
	}
	if strings.Contains(string(out), "pii-secret") || strings.Contains(string(out), "secret-coding-plan-token") ||
		strings.Contains(string(out), "must-not-leak") {
		t.Fatalf("PII or secret leaked from preload.cjs: %s", out)
	}
	parsed := parseStatus(out)
	if parsed.status != domain.StatusOK || parsed.period != domain.PeriodFiveHour ||
		parsed.percent == nil || *parsed.percent != 37.5 {
		t.Fatalf("preload output did not parse as 37.5%% 5h StatusOK: %+v (%s)", parsed, out)
	}
})

var _ = ginkgo.It("TestPreloadScriptReadOnlyNoOAuthPostNoCorruptBackupAndSeaIsolation", func() {
	t := testT{ginkgo.GinkgoT()}
	nodeBin, err := exec.LookPath("node")
	if err != nil {
		ginkgo.Skip("node not on PATH")
	}

	// Case A: When the active/preferred account is Zai OAuth (without a stored
	// Coding Plan :api-key) even if BigModel has a stored :api-key, preload.cjs
	// must NOT call headersPort / ProjectAccessTokenClient, must NOT fall back
	// to BigModel's stored key, and must return StatusAuthRequired.
	{
		dir := t.TempDir()
		preloadDir := filepath.Join(dir, "preload")
		if err := os.MkdirAll(preloadDir, 0o700); err != nil {
			t.Fatal(err)
		}
		preloadPath := filepath.Join(preloadDir, "preload.cjs")
		if err := os.WriteFile(preloadPath, preloadScript, 0o600); err != nil {
			t.Fatal(err)
		}
		personalConfig := filepath.Join(dir, "provider_config.json")
		if err := os.WriteFile(personalConfig, []byte(`{"config":{"defaultModelSelection":{"providerId":"account:zai-individual-coding-plan","modelId":"glm-5.1"}}}`), 0o600); err != nil {
			t.Fatal(err)
		}
		driverPath := filepath.Join(dir, "driver-oauth-only.cjs")
		driverCode := `"use strict";
const defProp = Object.defineProperty;
const r = (fn, name) => defProp(fn, "name", { value: name, configurable: true });

const storeData = {
  "oauth:active_provider": "zai",
  "oauth:zai:access_token": "oauth-token-without-coding-plan-key",
  "oauth:zai:user_info": JSON.stringify({ id: "zai-user-1" }),
  "account-provider:account:bigmodel-individual-coding-plan:identity": "bm-user-1",
  "account-provider:account:bigmodel-individual-coding-plan:auth-source": "manual",
  "account-provider:coding-plan:account:bigmodel-individual-coding-plan:account:bm-user-1:api-key": "bm-stored-key"
};

const createStore = r(() => ({
  filePath: "/tmp/nonexistent-gaebokchi-oauth-only.json",
  load: async (k) => (k in storeData ? storeData[k] : null),
  loadMany: async (keys) => Object.fromEntries(keys.map((k) => [k, k in storeData ? storeData[k] : null])),
}), "createSharedZCodeCredentialStore");

const readProviders = r(async () => [
  { family: "zai", providerId: "account:zai-individual-coding-plan", modelId: "glm-5.1" },
  { family: "bigmodel", providerId: "account:bigmodel-individual-coding-plan", modelId: "glm-5.1" }
], "readStandaloneCodingPlanProviders");

const readSnapshot = r(async (store) => {
  const zaiId = await store.load("account-provider:account:zai-individual-coding-plan:identity");
  const zaiToken = await store.load("oauth:zai:access_token");
  const bmKey = await store.load("account-provider:coding-plan:account:bigmodel-individual-coding-plan:account:bm-user-1:api-key");
  return {
    providers: new Map([
      ["account:zai-individual-coding-plan", { access: { entitled: Boolean(zaiId && zaiToken) } }],
      ["account:bigmodel-individual-coding-plan", { access: { entitled: Boolean(bmKey) } }]
    ])
  };
}, "readStandaloneAccountProviderConfigSnapshot");

const createHeadersPort = r(() => ({
  refreshBeforeModelRequest: async () => {
    throw new Error("MUST NOT call refreshBeforeModelRequest or mint API keys");
  }
}), "createStandaloneProviderRuntimeHeadersPort");

const resolveEndpoints = r(() => ({
  zaiBusinessBaseUrl: "https://api.z.ai",
  bigModelApiOrigin: "https://bigmodel.cn"
}), "resolveRuntimeProductEndpointConfig");

const createHttpClient = r(() => ({
  request: async (req) => {
    throw new Error("MUST NOT make any HTTP request when selected account lacks stored API key: " + req.url);
  }
}), "createDefaultHttpClient");

const loadBootstrap = r(async () => ({
  createStore, readProviders, readSnapshot, createHeadersPort, resolveEndpoints, createHttpClient
}), "loadBootstrapModule");

const runDoctor = r(() => 0, "runDoctor");
runDoctor({ stdout: process.stdout, stderr: process.stderr }, { json: true });
`
		if err := os.WriteFile(driverPath, []byte(driverCode), 0o600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(nodeBin, driverPath)
		cmd.Env = append(
			os.Environ(),
			"NODE_OPTIONS=--require="+preloadPath,
			"ZCODE_PERSONAL_PROVIDER_CONFIG_FILE="+personalConfig,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("oauth-only node execution failed: %v: %s", err, out)
		}
		parsed := parseStatus(out)
		if parsed.status != domain.StatusAuthRequired {
			t.Fatalf("oauth-only without stored api-key = %+v (%s), want StatusAuthRequired", parsed, out)
		}
	}

	// Case B: Corrupt credentials.json on disk, callback/promise/sync fs write
	// guards, and SEA builtin config asset must NEVER create a .corrupt-*.bak
	// file or write anything under dataBaseDir.
	{
		dir := t.TempDir()
		preloadDir := filepath.Join(dir, "preload")
		dataBaseDir := filepath.Join(dir, "zcode-home")
		credDir := filepath.Join(dataBaseDir, ".zcode", "v2")
		if err := os.MkdirAll(preloadDir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(credDir, 0o700); err != nil {
			t.Fatal(err)
		}
		preloadPath := filepath.Join(preloadDir, "preload.cjs")
		if err := os.WriteFile(preloadPath, preloadScript, 0o600); err != nil {
			t.Fatal(err)
		}
		credFile := filepath.Join(credDir, "credentials.json")
		if err := os.WriteFile(credFile, []byte("{corrupt-json"), 0o600); err != nil {
			t.Fatal(err)
		}
		driverPath := filepath.Join(dir, "driver-corrupt.cjs")
		driverCode := `"use strict";
const fs = require("node:fs");
const fsp = require("node:fs/promises");
const defProp = Object.defineProperty;
const r = (fn, name) => defProp(fn, "name", { value: name, configurable: true });
const credFile = process.env.TEST_CRED_FILE;

const getSeaAssets = r(() => ({
  isSea: () => true,
  getAsset: () => '{"providers":[]}'
}), "getSeaProviderConfigAssets");

const createStore = r(() => ({
  filePath: credFile,
  load: async () => {
    await fsp.writeFile(credFile + ".corrupt-123.bak", "backup");
    return null;
  },
  loadMany: async () => ({}),
}), "createSharedZCodeCredentialStore");

const readProviders = r(async () => {
  // Verify sync, promise, and callback fs write APIs outside preloadDir all fail with EROFS.
  const externalTarget = credFile + ".probe";
  let syncCode = "";
  try {
    fs.writeFileSync(externalTarget, "x");
  } catch (e) {
    syncCode = e && e.code;
  }
  const promiseCode = await fsp.writeFile(externalTarget, "x").then(() => "OK", (e) => e && e.code);
  const cbWriteCode = await new Promise((resolve) => {
    fs.writeFile(externalTarget, "x", (e) => resolve(e && e.code));
  });
  const cbOpenCode = await new Promise((resolve) => {
    fs.open(externalTarget, "w", (e) => resolve(e && e.code));
  });
  if (syncCode !== "EROFS" || promiseCode !== "EROFS" || cbWriteCode !== "EROFS" || cbOpenCode !== "EROFS") {
    throw new Error("fs write guard bypass: " + JSON.stringify({ syncCode, promiseCode, cbWriteCode, cbOpenCode }));
  }
  return [
    { family: "zai", providerId: "account:zai-individual-coding-plan", modelId: "glm-5.1" }
  ];
}, "readStandaloneCodingPlanProviders");

const readSnapshot = r(async () => ({ providers: new Map() }), "readStandaloneAccountProviderConfigSnapshot");
const resolveEndpoints = r(() => ({ zaiBusinessBaseUrl: "https://api.z.ai", bigModelApiOrigin: "https://bigmodel.cn" }), "resolveRuntimeProductEndpointConfig");
const createHttpClient = r(() => ({ request: async () => ({ status: 200, body: "{}" }) }), "createDefaultHttpClient");
const loadBootstrap = r(async () => ({ getSeaAssets, createStore, readProviders, readSnapshot, resolveEndpoints, createHttpClient }), "loadBootstrapModule");

const runDoctor = r(() => 0, "runDoctor");
runDoctor({ stdout: process.stdout, stderr: process.stderr }, { json: true });
`
		if err := os.WriteFile(driverPath, []byte(driverCode), 0o600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(nodeBin, driverPath)
		cmd.Env = append(
			os.Environ(),
			"NODE_OPTIONS=--require="+preloadPath,
			"ZCODE_DATA_BASE_DIR="+dataBaseDir,
			"TEST_CRED_FILE="+credFile,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("corrupt-creds node execution failed: %v: %s", err, out)
		}
		parsed := parseStatus(out)
		if parsed.status != domain.StatusProviderError {
			t.Fatalf("corrupt credentials = %+v (%s), want StatusProviderError", parsed, out)
		}
		entries, err := os.ReadDir(credDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != "credentials.json" {
			t.Fatalf("unexpected files created in %s: %+v", credDir, entries)
		}
		if _, statErr := os.Stat(filepath.Join(preloadDir, "zcode-builtin.json")); statErr != nil {
			t.Fatalf("expected SEA builtin asset in preloadDir, got: %v", statErr)
		}
	}
})

var _ = ginkgo.It("reports a nonzero zcode CLI exit without fabricating usage", func() {
	t := testT{ginkgo.GinkgoT()}
	bin := filepath.Join(t.TempDir(), "zcode-fail")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result := New().Query(ctx, domain.Target{ID: "zcode:0", Provider: domain.ProviderZCode, FallbackAccount: "default", Bin: bin})
	if result.Status != domain.StatusProviderError || result.Diagnostic.Code != "status_exit_error" || result.UsedPercent != nil || result.CleanupFailed {
		t.Fatalf("nonzero CLI exit = %+v", result)
	}
})
