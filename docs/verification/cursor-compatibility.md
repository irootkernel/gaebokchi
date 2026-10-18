# Cursor Agent compatibility and qualification record

## Current finding

On 2026-10-18, `cursor-agent 2026.10.16-e3cb207` (`darwin/arm64`) was qualified
through static bundle analysis and a bounded, owner-authorized read-only live
probe. Because `cursor-agent` still lacks a standalone public usage flag
(`cursor-agent -p /usage` sends `"/usage"` as an LLM prompt, `acp` exposes only
`cursor/list_available_models`, and plain `status`/`about` JSON omit quota
fields), the owner retired the former "pure public CLI flags only" restriction
for Cursor and approved the **subprocess (`<cursor-bin> status --format json`) +
read-only `NODE_OPTIONS` preload hook** contract ([ADR-0002](../architecture-decision-records/ADR-0002-period-aware-provider-usage.md),
[REQ-065](../specs/README.md#req-065), [ARCH-19](../architecture/README.md#arch-19)).

Use the unambiguous default executable `cursor-agent`; an executable named
`agent` can belong to a different provider.

## Static package inspection (`2026.10.16-e3cb207`)

- **Wrapper and runtime:** `~/.local/bin/cursor-agent` resolves to
  `~/.local/share/cursor-agent/versions/2026.10.16-e3cb207/cursor-agent`, which
  executes `$SCRIPT_DIR/node --use-system-ca "$SCRIPT_DIR/index.js" "$@"`.
- **Headless print mode (`7956.index.js` -> `./src/headless.ts`):** Passing
  `-p /usage --output-format json` does not intercept `/usage` as a local slash
  command; it wraps `"/usage"` in `UserMessageAction` and submits a normal model
  turn. `cursor-agent -p /usage` must never be invoked by `gaebokchi`.
- **ACP mode (`5300.index.js` -> `./src/acp/cursor-acp-agent.ts`):** `extMethod`
  supports only `cursor/list_available_models`; all other extension methods
  return `RequestError.methodNotFound`.
- **Plain `status` and `about` commands (`684.index.js` and `5233.index.js`):**
  - `cursor-agent status --format json` initializes `credentialManager` and
    `dashboardClient` (`aiserver.v1.DashboardService` ConnectRPC client) in
    `index.js` and passes both to `./src/commands/status.ts` (`handleStatus`),
    but the default handler calls only `dashboardClient.getMe({})`.
  - `cursor-agent about --format json` calls `dashboardClient.getMe({})` and
    `dashboardClient.getPlanInfo({})`, but outputs only `subscriptionTier:
    planInfo?.planName ?? null` and discards period, reset, and usage fields.
- **Internal `/usage` data loader (`7956.index.js` -> `./src/usage/usage-data.ts`):**
  Calls `dashboardClient.getCurrentPeriodUsage(new GetCurrentPeriodUsageRequest({}))`
  and `dashboardClient.getPlanInfo(new GetPlanInfoRequest({}))`:
  - `GetPlanInfoResponse.PlanInfo` (`WS` in `index.js`):
    - `planName` (`string`, e.g., `"Pro"`, `"Ultra"`)
    - `includedUsagePeriod` (`IncludedUsagePeriod` enum: `0 = UNSPECIFIED`,
      `1 = MONTHLY`, `2 = WEEKLY`)
    - `includedUsageResetsAt` (optional int64 epoch ms)
    - `billingCycleEnd` (optional int64 epoch ms)
  - `GetCurrentPeriodUsageResponse` (`bO3` in `index.js`):
    - `billingCycleStart`, `billingCycleEnd` (int64 epoch ms)
    - `enabled` (`bool`)
    - `planUsage` (`PlanUsage` / `jA`): `totalSpend`, `includedSpend`,
      `bonusSpend`, `remaining`, `limit` (int32 cents), `autoPercentUsed`,
      `apiPercentUsed`, `totalPercentUsed` (optional double)
    - `spendLimitUsage` (`SpendLimitUsage` / `DA` — on-demand spend, excluded
      from subscription quota percentages)
- **Credential boundary (`index.js` -> `./src/utils/credential-store.ts` and
  `./src/api-key-auth.ts`):** `CURSOR_CONFIG_DIR` selects configuration files
  only; on macOS, credentials are stored in Keychain under the default Cursor
  service names regardless of `CURSOR_CONFIG_DIR`. Ambient `CURSOR_API_KEY` and
  `CURSOR_AUTH_TOKEN` can override or replace stored authentication, and
  `credentialManager.getApiKey()` reports whether the stored authentication
  used an API key.

## 2026-10-18 bounded live qualification (`darwin/arm64`)

- **Provider:** `cursor` (`domain.ProviderCursor`, `DisplayName() == "Cursor"`)
- **Observed at:** `2026-10-18`
- **OS / architecture:** `darwin/arm64` (macOS, Go `1.27.0`)
- **Executable identity and version:** `~/.local/bin/cursor-agent` -> `~/.local/share/cursor-agent/versions/2026.10.16-e3cb207/cursor-agent` (`cursor-agent --version` -> `2026.10.16-e3cb207`)
- **Public source / reference:** [S-20](../architecture/README.md#s-20), [S-21](../architecture/README.md#s-21), [S-24](../architecture/README.md#s-24), and static inspection of the installed `2026.10.16-e3cb207` bundle (`index.js`, `684.index.js`, `5233.index.js`, `5300.index.js`, `7956.index.js`)
- **Invocation and preload hook contract:** `<cursor-bin> status --format json` (default `cursor-agent`) with stdin closed (`os.DevNull`), a private mode-`0700` temporary working directory (`internal/process`), and `NODE_OPTIONS=--require=<preloadPath>` pointing to a read-only `preload.cjs` written inside a private mode-`0700` temporary directory (`os.MkdirTemp("", "gaebokchi-cursor-")`) that is removed on `defer` (folding any removal error into `Result.CleanupFailed`).
- **Child environment policy (`envfilter.Cursor()`):** Preserve `HOME` and platform keyring variables so `cursor-agent` accesses its default stored login; strip all `CURSOR_*` variables (`CURSOR_API_KEY`, `CURSOR_AUTH_TOKEN`, `CURSOR_CONFIG_DIR`, `CURSOR_STATSIG_OVERRIDES`) and any ambient parent `NODE_OPTIONS`, then set `NODE_OPTIONS=--require=<preloadPath>`.
- **Direct-child lifecycle and synchronous output:** `cursor-agent status --format json` runs as a single short-lived direct child with no shared background daemon or leader process. Because `./src/commands/status.ts` (`handleStatus`) calls `process.exit(exitCode)` inside its `finally` block while `index.js` retains open HTTP/2 and timer handles, `preload.cjs` writes one compact newline-terminated JSON line synchronously via a `fs.writeSync(1, ...)` loop (`writeStdoutSync`, handling partial pipe writes above 512 bytes) before calling `process.exit(0)`.
- **Read-only and privacy boundary (`ARCH-09`):** Zero model turns, zero agent sessions, zero tool calls, no login/logout or billing mutation, and no direct Keychain or credential-file reads by `gaebokchi` outside `cursor-agent`'s own `credentialManager`. All PII (`email`, `userId`, `firstName`, `lastName`, `teamId`, `teamName`) and raw secrets (`accessToken`, `refreshToken`, `apiKey`) are excluded inside `preload.cjs` before serialization.

With owner authorization, a single read-only probe ran
`cursor-agent status --format json` with `NODE_OPTIONS=--require=<preload.cjs>`
intercepting `./src/commands/status.ts` (`handleStatus`). The hook checked
`credentialManager.getAccessToken()`, `getRefreshToken()`, and `getApiKey()`,
called `dashboardClient.getPlanInfo` and `dashboardClient.getCurrentPeriodUsage`,
omitted all PII (`email`, `userId`, `firstName`, `lastName`, `teamId`,
`teamName`, tokens, and keys), and wrote a single compact JSON line via
`fs.writeSync(1, ...)`.

Sanitized captured response (`cursor-agent 2026.10.16-e3cb207`, `darwin/arm64`, provenance: `captured`):

```json
{
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
}
```

Key verified contract points:
1. **Explicit weekly period:** `planInfo.includedUsagePeriod` is `2`
   (`INCLUDED_USAGE_PERIOD_WEEKLY`), and
   `currentPeriodUsage.billingCycleEndMs - currentPeriodUsage.billingCycleStartMs`
   is `1792591073284 - 1791986273284 = 604,800,000 ms` (exactly 7 days).
2. **Weekly reset timestamp:** `planInfo.includedUsageResetsAtMs`
   (`1792591073284`, `2026-10-21T13:57:53Z`) matches
   `currentPeriodUsage.billingCycleEndMs` (`1792591073284`), whereas
   `planInfo.billingCycleEndMs` (`1794664361314`) is the subscription renewal
   timestamp. Weekly `ResetAt` must use `includedUsageResetsAtMs` when positive,
   falling back to `currentPeriodUsage.billingCycleEndMs`.
3. **Single-row `Included` usage percentage:** Weekly plans report percentage
   consumption directly in `planUsage.totalPercentUsed` (labeled `Included` in
   Cursor's `/usage` view) while `includedSpend` and `limit` remain `0`. Match
   `./src/usage/usage-data.ts`: use `planUsage.totalPercentUsed` when present,
   or `includedSpend / limit * 100` when `totalPercentUsed` is absent and
   `limit > 0`.
4. **Synchronous stdout flush and process exit:** Because `./src/commands/status.ts`
   (`handleStatus`) owns the `finally { process.exit(exitCode) }` termination in
   `cursor-agent` while `index.js` keeps open HTTP/2 and timer handles, the
   preload hook must loop `fs.writeSync(1, ...)` to completion across partial
   pipe writes (`> 512` bytes on macOS) and call `process.exit(0)` rather than
   relying on buffered console helpers or natural event-loop drain.

## Sanitized fixture catalog for `TASK-040` offline tests

Each fixture below is tagged `captured` or `synthetic` per [ARCH-12](../architecture/README.md#arch-12)
and defines the exact normalization contract for `internal/provider/cursor` in
[TASK-040](../roadmap/README.md#task-040):

### Case 1a (`captured` — Weekly active plan with `totalPercentUsed` -> `StatusOK`)

- **Provenance:** `captured` on 2026-10-18 from `cursor-agent 2026.10.16-e3cb207` (`darwin/arm64`), field-allowlisted with no PII or secrets (payload shown in the section above).
- **Expected normalization:**
  - `Status`: `domain.StatusOK` (`ExitClean`)
  - `UsedPercent`: `25` (rendered `25%`)
  - `Period`: `domain.PeriodWeekly`
  - `QuotaID`: `"cursor_included"`
  - `Source`: `domain.SourceCursorCLIUsage` (`"cursor_cli_usage"`)
  - `ResetAt`: `2026-10-21T13:57:53Z` (from `planInfo.includedUsageResetsAtMs: "1792591073284"`)

### Case 1b (`synthetic` — Weekly active plan with `includedSpend / limit * 100` fallback and `spendLimitUsage` exclusion -> `StatusOK`)

- **Provenance:** `synthetic`, derived from `./src/usage/usage-data.ts` (`7956.index.js`) and `aiserver.v1.DashboardService` protobuf definitions (`jA` / `PlanUsage`, `DA` / `SpendLimitUsage`). Also covers `includedUsagePeriod: 0` (`UNSPECIFIED`) when `currentPeriodUsage.billingCycleEndMs - currentPeriodUsage.billingCycleStartMs` is within 6–8 days (`[6*24h, 8*24h]`).
- **Payload:**
  ```json
  {
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
  }
  ```
- **Expected normalization:**
  - `Status`: `domain.StatusOK` (`ExitClean`)
  - `UsedPercent`: `30` (`1500 / 5000 * 100`, rendered `30%`; `spendLimitUsage` on-demand spend is ignored)
  - `Period`: `domain.PeriodWeekly`
  - `QuotaID`: `"cursor_included"`
  - `Source`: `domain.SourceCursorCLIUsage` (`"cursor_cli_usage"`)
  - `ResetAt`: `2026-10-21T13:57:53Z` (fallback to `currentPeriodUsage.billingCycleEndMs` when `includedUsageResetsAtMs` is null/absent)

### Case 2 (`synthetic` — Monthly plan -> `StatusNotApplicableMonthly` / `N/A (monthly)`)

- **Provenance:** `synthetic`, derived from `IncludedUsagePeriod` enum `1 = INCLUDED_USAGE_PERIOD_MONTHLY` in `index.js` (also applies when `includedUsagePeriod == 0` and `currentPeriodUsage.billingCycleEndMs - currentPeriodUsage.billingCycleStartMs` is within 27–32 days).
- **Payload:**
  ```json
  {
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
  }
  ```
- **Expected normalization:**
  - `Status`: `domain.StatusNotApplicableMonthly` (`N/A (monthly)`, `ExitClean`)
  - `UsedPercent`: `nil` (monthly percentage is never displayed as weekly usage)
  - `Period`: `domain.PeriodMonthly`
  - `QuotaID`: `"cursor_included"`
  - `ResetAt`: `nil`

### Case 3 (`synthetic` — Unauthenticated / missing access or refresh token -> `StatusAuthRequired` / `AUTH REQUIRED`)

- **Provenance:** `synthetic`, derived from `./src/utils/credential-store.ts` (`getAccessToken()`, `getRefreshToken()`). Triggered whenever `!isAuthenticated || !hasAccessToken || !hasRefreshToken`.
- **Payload:**
  ```json
  {
    "status": "unauthenticated",
    "isAuthenticated": false,
    "hasAccessToken": false,
    "hasRefreshToken": false,
    "hasApiKey": false
  }
  ```
- **Expected normalization:**
  - `Status`: `domain.StatusAuthRequired` (`AUTH REQUIRED`, `ExitIncomplete`)

### Case 4 (`synthetic` — API-key authentication -> `StatusNotApplicableAPIKey` / `N/A (API key)`)

- **Provenance:** `synthetic`, derived from `./src/api-key-auth.ts` and `credentialManager.getApiKey()`.
- **Payload:**
  ```json
  {
    "status": "api_key",
    "isAuthenticated": true,
    "hasAccessToken": true,
    "hasRefreshToken": false,
    "hasApiKey": true
  }
  ```
- **Expected normalization:**
  - `Status`: `domain.StatusNotApplicableAPIKey` (`N/A (API key)`, `ExitClean`)

### Case 5 (`synthetic` — ConnectRPC error (`planError` or `usageError`) -> `StatusProviderError` / `ERROR`)

- **Provenance:** `synthetic`, derived from `preload.cjs` error capture around `dashboardClient.getPlanInfo` and `dashboardClient.getCurrentPeriodUsage`.
- **Payload:**
  ```json
  {
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
  }
  ```
- **Expected normalization:**
  - `Status`: `domain.StatusProviderError` (`ERROR`, `ExitIncomplete`), or `domain.StatusAuthRequired` only when the structured RPC error `code` is explicitly `"unauthenticated"`, with an application-owned diagnostic and no raw provider error text echoed to stderr.

### Case 6 (`synthetic` — Unhooked plain `status --format json` fallback -> `StatusUnsupported` / `UNSUPPORTED`)

- **Provenance:** `synthetic`, derived from `684.index.js` (`./src/commands/status.ts` default `format: "json"` output when the preload hook does not intercept `handleStatus`).
- **Payload:**
  ```json
  {
    "isAuthenticated": true,
    "email": "user@example.com",
    "loginMethod": "cursor"
  }
  ```
- **Expected normalization:**
  - `Status`: `domain.StatusUnsupported` (`UNSUPPORTED`, `ExitIncomplete`) because the preload-hook discriminator fields (`status`, `hasAccessToken`, `hasRefreshToken`, `hasApiKey`) are absent. Any PII in unhooked output is discarded and never rendered.

### Case 7 (`synthetic` — Malformed, out-of-range, oversized, and unavailable edge cases)

- **Provenance:** `synthetic`, covering [ARCH-04](../architecture/README.md#arch-04) and [ARCH-19](../architecture/README.md#arch-19) defensive normalization branches:
  - **Non-JSON stdout, stdout > 256 KiB, `totalPercentUsed < 0` or `> 100`, or invalid spend/limit (`limit > 0` with `includedSpend < 0` or `includedSpend > limit`):** `domain.StatusProtocolError` (`PROTOCOL ERROR`).
  - **Unknown or unresolved period (`planInfo == null`, unknown `includedUsagePeriod` enum, or `includedUsagePeriod == 0` without a 6–8 day or 27–32 day billing cycle window):** `domain.StatusUnavailablePeriod` (`N/A (period)`).
  - **Weekly period with missing usage data (`currentPeriodUsage == null`, `currentPeriodUsage.enabled == false`, `planUsage == null`, or `totalPercentUsed == null` with `limit <= 0`):** `domain.StatusUnavailableNoData` (`N/A (no data)`, `Period: domain.PeriodWeekly`, `QuotaID: "cursor_included"`).

## Frozen EPIC-013 implementation contract (`TASK-039` -> `TASK-041`)

- **[TASK-039](../roadmap/README.md#task-039):** Reuses this qualified
  `2026.10.16-e3cb207` evidence to close [REQ-065](../specs/README.md#req-065)
  and hand the frozen contract and fixtures to `TASK-040`.
- **[TASK-040](../roadmap/README.md#task-040):** Implements
  `internal/provider/cursor`, `envfilter.Cursor()` (stripping `CURSOR_*` and
  ambient `NODE_OPTIONS` while setting `NODE_OPTIONS=--require=<preloadPath>`),
  single-entry `cursor` configuration in `~/.config/gaebokchi/config.yaml`
  (`cursor:0`), `--cursor-bin` (default `cursor-agent`), explicit
  `Runner.querierFor(domain.ProviderCursor)` dispatch, and offline fake-process
  unit/integration tests.
- **[TASK-041](../roadmap/README.md#task-041):** Completed the full offline and
  race verification gates (`make test` and `go test -race ./... -count=1`) plus
  bounded prompt-free live final-executable acceptance (`table` and `--json`)
  against `cursor-agent 2026.10.16-e3cb207` on `darwin/arm64`, closing
  [REQ-066](../specs/README.md#req-066) and [EPIC-013](../roadmap/README.md#epic-013).

## TASK-039 verification evidence

- **Task:** [TASK-039](../roadmap/README.md#task-039) (`Qualify Cursor usage metadata and period semantics`)
- **Status:** `COMPLETE`
- **Goal and implementation scope:** Freeze the qualified `cursor-agent 2026.10.16-e3cb207` subprocess (`<cursor-bin> status --format json`) + read-only `NODE_OPTIONS=--require=<preloadPath>` preload hook contract on `./src/commands/status.ts` (`handleStatus(credentialManager, dashboardClient)`), weekly vs. monthly period semantics, single-row `Included` (`totalPercentUsed`, `QuotaID: "cursor_included"`) normalization, `CURSOR_*` and ambient `NODE_OPTIONS` environment filtering, PII/secret exclusion, and sanitized `captured`/`synthetic` fixture catalog before adapter implementation in `TASK-040`. No production Go source files modified.
- **Requirements and architecture sections covered:** [REQ-065](../specs/README.md#req-065); [ARCH-19](../architecture/README.md#arch-19), [ARCH-12](../architecture/README.md#arch-12), [ARCH-09](../architecture/README.md#arch-09).
- **Changed files:**
  - `docs/verification/cursor-compatibility.md`
  - `docs/specs/README.md`
  - `docs/architecture/README.md`
  - `docs/roadmap/README.md`
- **Verification:**
  - `V-CURSOR`, `V-CONTRACT`, `V-SAFETY`: Reused the owner-authorized 2026-10-18 static bundle inspection and bounded read-only live probe against `cursor-agent 2026.10.16-e3cb207` on `darwin/arm64` (`status: "authenticated"`, `planInfo.planName: "Ultra"`, `includedUsagePeriod: 2`, `totalPercentUsed: 25`, `includedUsageResetsAtMs: "1792591073284"`, zero model turns, zero sessions, zero tool calls, no direct Keychain/file reads outside `cursor-agent`, and all PII/tokens omitted).
  - `V-AUDIT`, `V-DOCS`: `go run scripts/check-docs.go` -> `check-docs: 60 markdown files, links and identifiers OK` (`darwin/arm64`, Go `1.27.0`).
  - **Explicitly NOT RUN / SKIPPED:** Additional live `cursor-agent` account probes were not re-run during `TASK-039` execution per the roadmap preflight reuse rule ([provider-discovery-preflight](../roadmap/README.md#provider-discovery-preflight)); Go package tests (`make test`) were not required because `TASK-039` is a documentation/contract qualification task with no Go code changes (`TASK-040` and `TASK-041` own offline and live executable gates).
- **Fixtures:**
  - Case 1a (`captured`): Sanitized live response from `cursor-agent 2026.10.16-e3cb207` on `darwin/arm64` (2026-10-18); field-allowlisted in `preload.cjs` before emission so `email`, `userId`, `firstName`, `lastName`, `teamId`, `teamName`, `accessToken`, `refreshToken`, and `apiKey` never leave the child process.
  - Cases 1b, 2, 3, 4, 5, 6, 7 (`synthetic`): Source-backed fixtures derived from `2026.10.16-e3cb207` bundle modules (`./src/usage/usage-data.ts`, `./src/commands/status.ts`, `./src/utils/credential-store.ts`, `./src/api-key-auth.ts`) covering `includedSpend / limit * 100` fallback with `spendLimitUsage` exclusion, monthly non-applicability (`StatusNotApplicableMonthly`), unauthenticated/missing-token (`StatusAuthRequired`), API-key non-applicability (`StatusNotApplicableAPIKey`), ConnectRPC errors (`StatusProviderError`), unhooked `status --format json` degradation (`StatusUnsupported`), and malformed/unavailable edge cases (`StatusProtocolError`, `StatusUnavailablePeriod`, `StatusUnavailableNoData`).
- **Findings and bounded remediation:** None blocking. The retired public-flag assumption (`cursor-agent -p /usage`) and the read-only `status --format json` preload hook contract are reconciled across `docs/specs/README.md`, `docs/architecture/README.md`, `docs/roadmap/README.md`, and `docs/verification/cursor-compatibility.md`.
- **Compatibility/platform limitations:** Live qualification was performed on `darwin/arm64` with `cursor-agent 2026.10.16-e3cb207` against an active weekly (`Ultra`) subscription; monthly (`includedUsagePeriod == 1`), `includedSpend / limit` fallback, API-key, unauthenticated, RPC error, and unhooked fallback branches are qualified via source inspection and synthetic fixtures for `TASK-040`. Multi-account Cursor selection is out of scope because `cursor-agent` uses a global OS Keychain service name regardless of `CURSOR_CONFIG_DIR`.
- **Task commit subject:** `[DOC] Qualify Cursor usage metadata and period semantics` (with trailers `Aquarium-Workflow: aquarium:task-commit`, `Aquarium-Epic: EPIC-013`, `Aquarium-Task: TASK-039`).
- **Handoff to next task:** [TASK-040](../roadmap/README.md#task-040) receives the frozen `cursor-agent status --format json` + `NODE_OPTIONS` preload hook contract, environment filter rules, and sanitized fixture catalog (Cases 1a–7) to implement `internal/provider/cursor`, config/CLI wiring, and offline fake-process tests.

## TASK-040 verification evidence

- **Task:** [TASK-040](../roadmap/README.md#task-040) (`Integrate the Cursor default-login provider`)
- **Status:** `COMPLETE`
- **Goal and implementation scope:** Integrate the single-account Cursor provider (`domain.ProviderCursor`, `DisplayName() == "Cursor"`, `SourceCursorCLIUsage == "cursor_cli_usage"`) into `gaebokchi`'s configuration (`cursor:` list with at most one `- name: <alias>` entry and no `path` field, target ID `cursor:0` after `agy`), CLI flag (`--cursor-bin`, default `cursor-agent`), environment filter (`envfilter.Cursor()`, stripping `CURSOR_*` and ambient parent `NODE_OPTIONS` while injecting `NODE_OPTIONS=--require=<preloadPath>`), explicit runner dispatch (`Runner.querierFor(domain.ProviderCursor)`), and adapter (`internal/provider/cursor` with embedded read-only `preload.cjs`). Normalize weekly plans (`PeriodWeekly`, `QuotaID: "cursor_included"`, `UsedPercent` from `planUsage.totalPercentUsed` or `includedSpend / limit * 100` fallback, `ResetAt` from `planInfo.includedUsageResetsAtMs` or `currentPeriodUsage.billingCycleEndMs` fallback), monthly plans (`StatusNotApplicableMonthly`, `N/A (monthly)`), API-key logins (`StatusNotApplicableAPIKey`), unauthenticated/partial credentials (`StatusAuthRequired`), ConnectRPC errors (`StatusProviderError`, or `StatusAuthRequired` on structured `"unauthenticated"` code), unhooked `status --format json` fallback (`StatusUnsupported`), and malformed/oversized/unavailable edge cases (`StatusProtocolError`, `StatusUnavailablePeriod`, `StatusUnavailableNoData`) while excluding on-demand spend (`spendLimitUsage`).
- **Requirements and architecture sections covered:** [REQ-002](../specs/README.md#req-002), [REQ-006](../specs/README.md#req-006), [REQ-007](../specs/README.md#req-007), [REQ-066](../specs/README.md#req-066) (implementation and offline verification; final live executable acceptance in [TASK-041](../roadmap/README.md#task-041)); [ARCH-01](../architecture/README.md#arch-01) through [ARCH-04](../architecture/README.md#arch-04), [ARCH-09](../architecture/README.md#arch-09) through [ARCH-11](../architecture/README.md#arch-11), [ARCH-19](../architecture/README.md#arch-19).
- **Changed files:**
  - `internal/domain/target.go`
  - `internal/domain/result.go`
  - `internal/domain/domain_test.go`
  - `internal/provider/envfilter/envfilter.go`
  - `internal/provider/envfilter/envfilter_test.go`
  - `internal/options/config.go`
  - `internal/options/options.go`
  - `internal/options/options_test.go`
  - `internal/provider/cursor/preload.cjs`
  - `internal/provider/cursor/provider.go`
  - `internal/provider/cursor/cursor_suite_test.go`
  - `internal/provider/cursor/provider_test.go`
  - `internal/app/runner.go`
  - `internal/app/runner_test.go`
  - `cmd/gaebokchi/main.go`
  - `cmd/gaebokchi/main_test.go`
  - `internal/render/table_test.go`
  - `internal/render/terminal_test.go`
  - `internal/render/json_test.go`
  - `test/integration/main_test.go`
  - `test/integration/e2e_test.go`
  - `Makefile`
  - `README.md`
  - `configuration/README.md`
  - `TESTING.md`
  - `docs/specs/README.md`
  - `docs/architecture/README.md`
  - `docs/roadmap/README.md`
  - `docs/verification/cursor-compatibility.md`
- **Verification:**
  - `V-OPTIONS`, `V-CURSOR`, `V-QUOTA`, `V-RUNNER`, `V-OUTPUT`, `V-EXIT`, `V-SAFETY`, `V-E2E`:
    - `go test ./internal/domain ./internal/provider/envfilter ./internal/options ./internal/provider/cursor ./internal/app ./internal/render ./cmd/gaebokchi ./test/integration -count=1` -> PASS (`darwin/arm64`, Go `1.27.0`).
    - `go test ./... -count=1` -> PASS across all 13 packages (`cmd/gaebokchi`, `internal/app`, `internal/domain`, `internal/options`, `internal/process`, `internal/provider/agy`, `internal/provider/claude`, `internal/provider/codex`, `internal/provider/cursor`, `internal/provider/envfilter`, `internal/provider/grok`, `internal/provider/home`, `internal/render`, `internal/rpc`, `internal/testutil`, `test/integration`).
    - `make test` -> PASS (`prepare`, `test-unit`, `test-int` including `./internal/provider/cursor` with `-race`, and `test-e2e` including `./test/integration` with `-race` and `archivecheck.go`).
  - `V-AUDIT`, `V-DOCS`: `go run scripts/check-docs.go` -> `check-docs: 60 markdown files, links and identifiers OK`.
  - **Explicitly NOT RUN / SKIPPED:** Live `cursor-agent` account query through the compiled `gaebokchi` binary was not run in `TASK-040`; [TASK-041](../roadmap/README.md#task-041) owns final live executable acceptance and closes [REQ-066](../specs/README.md#req-066) and [EPIC-013](../roadmap/README.md#epic-013).
- **Fixtures:**
  - Offline fake-process tests in `internal/provider/cursor/provider_test.go`, `cmd/gaebokchi/main_test.go`, and `test/integration/e2e_test.go` exercise all 7 fixture classes from `docs/verification/cursor-compatibility.md` (Case 1a `captured` weekly `totalPercentUsed`, Case 1b `synthetic` weekly `includedSpend / limit * 100` fallback and `spendLimitUsage` exclusion, Case 2 `synthetic` monthly `StatusNotApplicableMonthly`, Case 3 `synthetic` unauthenticated `StatusAuthRequired`, Case 4 `synthetic` API-key `StatusNotApplicableAPIKey`, Case 5 `synthetic` ConnectRPC error `StatusProviderError` and structured `"unauthenticated"` -> `StatusAuthRequired`, Case 6 `synthetic` unhooked plain `status --format json` -> `StatusUnsupported`, and Case 7 `synthetic` malformed/oversized/out-of-range/unavailable edge cases), plus real Node execution of `preload.cjs` verifying synchronous JSON output and PII/token omission.
- **Findings and bounded remediation:** None blocking. `WriteDiagnostics` emits one bounded stderr line (`gaebokchi: cursor:0: N/A (monthly): cursor:0: cursor account reports a monthly period, not weekly`) for `StatusNotApplicableMonthly` while exiting `0`, matching Grok's monthly non-applicability behavior across `cmd/gaebokchi` and `test/integration`.
- **Compatibility/platform limitations:** Offline fake-process and race-enabled integration/E2E gates verified on `darwin/arm64` (`Go 1.27.0`). Live final-executable acceptance against the installed `cursor-agent` login is deferred to `TASK-041`.
- **Task commit subject:** `[FEAT] Integrate the Cursor default-login provider` (with trailers `Aquarium-Workflow: aquarium:task-commit`, `Aquarium-Epic: EPIC-013`, `Aquarium-Task: TASK-040`).
- **Handoff to next task:** [TASK-041](../roadmap/README.md#task-041) receives the integrated, offline-tested Cursor candidate to run the final live executable query and complete `EPIC-013`.

## TASK-041 verification evidence

- **Task:** [TASK-041](../roadmap/README.md#task-041) (`Accept Cursor support and provider regressions`)
- **Status:** `COMPLETE`
- **Goal and implementation scope:** Run the complete offline and race verification gates (`make test` and `go test -race ./... -count=1`), verify a bounded prompt-free live Cursor usage query through the final compiled `gaebokchi` executable (`table` and `--json` modes) against the default `cursor-agent 2026.10.16-e3cb207` login on `darwin/arm64`, remediate partial non-blocking pipe writes and child event-loop exit in `internal/provider/cursor/preload.cjs`, and reconcile canonical and supporting documentation to close [REQ-066](../specs/README.md#req-066), [TASK-041](../roadmap/README.md#task-041), and [EPIC-013](../roadmap/README.md#epic-013).
- **Requirements and architecture sections covered:** [REQ-066](../specs/README.md#req-066), [REQ-065](../specs/README.md#req-065), [REQ-002](../specs/README.md#req-002), [REQ-006](../specs/README.md#req-006), [REQ-007](../specs/README.md#req-007), [REQ-050](../specs/README.md#req-050); [ARCH-11](../architecture/README.md#arch-11), [ARCH-12](../architecture/README.md#arch-12), [ARCH-14](../architecture/README.md#arch-14), [ARCH-19](../architecture/README.md#arch-19).
- **Changed files:**
  - `internal/provider/cursor/preload.cjs`
  - `internal/provider/cursor/provider_test.go`
  - `docs/verification/cursor-compatibility.md`
  - `docs/specs/README.md`
  - `docs/architecture/README.md`
  - `docs/roadmap/README.md`
  - `docs/todo/TODO-CURSOR-AGENT-SUPPORT.md`
  - `docs/todo/README.md`
  - `docs/README.md`
  - `docs/verification/README.md`
- **Verification:**
  - `V-CURSOR`, `V-OPTIONS`, `V-QUOTA`, `V-RUNNER`, `V-OUTPUT`, `V-EXIT`, `V-LIFECYCLE`, `V-SAFETY`, `V-E2E`:
    - `make test` (`test-prepare`, `test-unit`, `test-int`, `test-e2e`) -> **PASS** (`darwin/arm64`, Go `1.27.0`, `golangci-lint` `v2.13.2`; 4 unit suites, 10 integration suites including `Cursor Suite - 8/8 specs`, and `Integration Suite - 34/34 specs` passed with `-race`).
    - `go test -race ./... -count=1` -> **PASS** across all tested packages (`cmd/gaebokchi`, `internal/app`, `internal/domain`, `internal/options`, `internal/process`, `internal/provider/agy`, `internal/provider/claude`, `internal/provider/codex`, `internal/provider/cursor`, `internal/provider/envfilter`, `internal/provider/grok`, `internal/render`, `internal/rpc`, `scripts`, `test/integration`).
  - `V-CURSOR`, `V-LIVE`, `V-SAFETY` (bounded prompt-free live final-executable acceptance on `darwin/arm64`, 2026-10-18):
    - Built the candidate binary (`go build -o "$TMP_DIR/gaebokchi" ./cmd/gaebokchi`) and executed both plain-table and `--json` modes against `~/.local/bin/cursor-agent` (`2026.10.16-e3cb207`) using an isolated temporary `HOME` (`$TMP_DIR/home/.config/gaebokchi/config.yaml` containing `cursor:\n  - name: default\n`, with `$TMP_DIR/home/Library` symlinked to `$HOME/Library` so macOS Keychain resolved the default login without touching `~/.config/gaebokchi/config.yaml`, then removed `$TMP_DIR`).
    - Plain table (`HOME="$TMP_DIR/home" "$TMP_DIR/gaebokchi" --cursor-bin "$HOME/.local/bin/cursor-agent"`): exited `0` with `0` stderr bytes:
      ```text
      AI PROVIDER  ACCOUNT  WEEKLY USAGE
      Cursor       default  100%
      ```
    - JSON (`HOME="$TMP_DIR/home" "$TMP_DIR/gaebokchi" --cursor-bin "$HOME/.local/bin/cursor-agent" --json`): exited `0` with `0` stderr bytes:
      ```json
      {"results":[{"target_id":"cursor:0","provider":"cursor","account":"default","status":"ok","used_percent":100,"cleanup_failed":false}]}
      ```
    - Confirmed zero model turns, zero agent sessions, zero tool calls, no direct Keychain/credential reads outside `cursor-agent`, zero PII/secrets in stdout or stderr, and complete removal of `$TMP_DIR` and `gaebokchi-cursor-*` preload temporary directories.
  - `V-AUDIT`, `V-DOCS`: `go run scripts/check-docs.go` -> `check-docs: 60 markdown files, links and identifiers OK`.
  - **Explicitly NOT RUN / SKIPPED:** Native Linux execution of the `EPIC-013` test suite was not run on this macOS (`darwin/arm64`) host; live monthly (`StatusNotApplicableMonthly`) and live API-key (`StatusNotApplicableAPIKey`) Cursor accounts were not available on the host (those branches are verified via source inspection and offline fake-process tests in `internal/provider/cursor/provider_test.go` and `test/integration/e2e_test.go`).
- **Fixtures:**
  - Retained the 7 fixture classes (Case 1a `captured` and Cases 1b–7 `synthetic`) in `internal/provider/cursor/provider_test.go` and `test/integration/e2e_test.go`; updated `TestPreloadScriptExecutesAndStripsPIIInNode` so the Node harness keeps an active event-loop timer (`setInterval`) and relies on `preload.cjs` calling `process.exit(0)` after `writeStdoutSync`.
- **Findings and bounded remediation:** During the live final-executable run against `cursor-agent 2026.10.16-e3cb207` over an `os.Pipe()` stdout on `darwin/arm64`, two runtime behaviors were identified and fixed in `internal/provider/cursor/preload.cjs`: (1) `./src/commands/status.ts` (`handleStatus`) owns the `finally { process.exit(c) }` call while `index.js` keeps active ConnectRPC HTTP/2 and timer handles open, so `wrapHandleStatus` must call `process.exit(0)` after writing its JSON payload; and (2) `cursor-agent` places fd 1 in non-blocking mode, causing a single `fs.writeSync(1, str)` of a 544-byte JSON payload (`totalPercentUsed: 100`) to write only the first 512 bytes (`PIPE_BUF`). Adding `writeStdoutSync(str)` (looping `fs.writeSync(1, buf, off, buf.length - off)` across partial writes and `EAGAIN`/`EWOULDBLOCK` before `process.exit(0)`) resolved both issues cleanly.
- **Compatibility/platform limitations:** Live positive final-executable acceptance was verified on `darwin/arm64` against `cursor-agent 2026.10.16-e3cb207` with an active weekly (`Ultra`) subscription. If a future `cursor-agent` release alters the `./src/commands/status.ts` module boundary so the preload hook does not intercept `handleStatus`, `gaebokchi` degrades deterministically to `StatusUnsupported` (`UNSUPPORTED`). Multi-account Cursor selection remains out of scope because `cursor-agent` uses a single global OS Keychain credential namespace.
- **Task commit subject:** `[FIX] Accept Cursor support and provider regressions` (with trailers `Aquarium-Workflow: aquarium:task-commit`, `Aquarium-Epic: EPIC-013`, `Aquarium-Task: TASK-041`).
- **Handoff to next task/epic:** [EPIC-013](../roadmap/README.md#epic-013) (`TASK-039` through `TASK-041`) is `COMPLETE`. All planned provider extensions (`EPIC-012` `agy` and `EPIC-013` `Cursor`) are verified and closed.

