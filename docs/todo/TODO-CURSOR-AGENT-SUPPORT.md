# Cursor Agent usage support

This is the execution dossier for [EPIC-013](../roadmap/README.md#epic-013).
The [roadmap](../roadmap/README.md) owns its task membership, order and status.

## Outcome and authority

Query the single default `cursor-agent` CLI login through `<cursor-bin> status --format json`
with a read-only `NODE_OPTIONS` preload hook on `./src/commands/status.ts` and
present its weekly `Included` allowance (`totalPercentUsed`) as a single row in
`gaebokchi`'s output and JSON snapshot (`cursor:0`). Cursor multi-account
management is outside this epic.

| Decision | Canonical owner |
|---|---|
| Qualified subprocess + preload-hook metadata interface | [REQ-065](../specs/README.md#req-065), [ARCH-19](../architecture/README.md#arch-19), [ADR-0002](../architecture-decision-records/ADR-0002-period-aware-provider-usage.md) |
| Configuration, adapter and positive support | [REQ-066](../specs/README.md#req-066), [ARCH-19](../architecture/README.md#arch-19) |
| Shared display, JSON and dispatch contracts | [REQ-061](../specs/README.md#req-061) through [REQ-063](../specs/README.md#req-063), [ARCH-17](../architecture/README.md#arch-17), [ADR-0002](../architecture-decision-records/ADR-0002-period-aware-provider-usage.md) |
| Qualification evidence | [Cursor compatibility](../verification/cursor-compatibility.md), [preflight record](../verification/provider-extension-preflight-2026-10-16.md) |

## Work map

| Work unit | Input and required handoff |
|---|---|
| [TASK-039](../roadmap/README.md#task-039) | Reuse the qualified `cursor-agent 2026.10.16-e3cb207` subprocess + read-only `NODE_OPTIONS` preload hook evidence ([cursor-compatibility.md](../verification/cursor-compatibility.md)), freeze the `status --format json` hook schema (`planInfo` and `currentPeriodUsage`), weekly/monthly classification, single-row `cursor_included` mapping, and `CURSOR_*` environment filter, and close [REQ-065](../specs/README.md#req-065). |
| [TASK-040](../roadmap/README.md#task-040) | Implement `internal/provider/cursor` (with embedded `preload.cjs`), `envfilter.Cursor()`, `SourceCursorCLIUsage`, single-entry `cursor` configuration (`cursor:0`, `name` only, no `path`), `--cursor-bin` (default `cursor-agent`), explicit `Runner.querierFor(domain.ProviderCursor)` dispatch, and offline fake-process unit/integration tests. |
| [TASK-041](../roadmap/README.md#task-041) | Run the complete offline and race gates plus a positive prompt-free final-executable query, update user/developer documentation, and close [REQ-066](../specs/README.md#req-066) and [EPIC-013](../roadmap/README.md#epic-013). |

## Entry gate and locked architectural direction

EPIC-012 is complete. On 2026-10-18, the owner retired the former "pure public
CLI flags only" restriction for `cursor-agent` after verifying on
`2026.10.16-e3cb207` that `-p /usage` sends `"/usage"` as an LLM prompt and plain
`status`/`about`/`acp` omit quota data, while `cursor-agent status --format json`
already initializes `credentialManager` and `dashboardClient`
(`aiserver.v1.DashboardService`) and passes both into `./src/commands/status.ts`
(`handleStatus`).

The canonical Cursor metadata mechanism is therefore **subprocess
(`<cursor-bin> status --format json`) + read-only `NODE_OPTIONS` preload hook**
([ADR-0002](../architecture-decision-records/ADR-0002-period-aware-provider-usage.md)):
- `internal/provider/cursor` writes an embedded read-only `preload.cjs` into a
  private `0700` temporary directory (`os.MkdirTemp("", "gaebokchi-cursor-")`,
  removed on `defer`) and sets `NODE_OPTIONS=--require=<preloadPath>`.
- `preload.cjs` wraps `handleStatus(credentialManager, dashboardClient)`, checks
  `getAccessToken()`, `getRefreshToken()`, and `getApiKey()`, and—when
  authenticated without an API key—calls `dashboardClient.getPlanInfo` and
  `dashboardClient.getCurrentPeriodUsage` concurrently, writing one compact JSON
  line via `fs.writeSync(1, ...)` with all PII omitted.
- If a future `cursor-agent` build changes `./src/commands/status.ts` so the
  hook does not intercept `handleStatus`, the subprocess still returns standard
  `status --format json` output and `gaebokchi` degrades deterministically to
  `StatusUnsupported` (`UNSUPPORTED`).

## Implementation boundaries

- Use `cursor-agent` as the default executable (`--cursor-bin`); `agent` can
  name another tool.
- Query only the existing default CLI login (`cursor:0`, at most one entry in
  `config.yaml` with `name` only and no `path` field). `CURSOR_CONFIG_DIR`
  selects configuration files only and does not create a separate macOS Keychain
  credential namespace.
- Strip all `CURSOR_*` variables (`CURSOR_API_KEY`, `CURSOR_AUTH_TOKEN`,
  `CURSOR_CONFIG_DIR`, `CURSOR_STATSIG_OVERRIDES`) and ambient parent
  `NODE_OPTIONS` in `envfilter.Cursor()`, then inject only the controlled
  `NODE_OPTIONS` preload path.
- Keep authentication and token refresh inside `cursor-agent`'s own
  `credentialManager` and `dashboardClient`. Do not read Keychain entries or
  auth files from Go, and do not build a direct HTTP/ConnectRPC client in Go.
- Keep the shared concurrency, per-target deadline, cancellation and cleanup
  contracts. Do not terminate provider-owned or unrelated processes.
- Classify periods explicitly from `planInfo.includedUsagePeriod` and
  `currentPeriodUsage`:
  - `WEEKLY` (`2` or `"INCLUDED_USAGE_PERIOD_WEEKLY"`, or `0`/`UNSPECIFIED` with
    a 6–8 day `billingCycleEnd - billingCycleStart` window) -> `StatusOK`,
    `Period: domain.PeriodWeekly`, `QuotaID: "cursor_included"`,
    `Source: domain.SourceCursorCLIUsage`, `UsedPercent` from
    `planUsage.totalPercentUsed` (or `includedSpend / limit * 100` when
    `totalPercentUsed` is absent and `limit > 0`), and `ResetAt` from
    `planInfo.includedUsageResetsAtMs` (falling back to
    `currentPeriodUsage.billingCycleEndMs`).
  - `MONTHLY` (`1` or `"INCLUDED_USAGE_PERIOD_MONTHLY"`, or `0`/`UNSPECIFIED`
    with a 27–32 day cycle window) -> `StatusNotApplicableMonthly`
    (`N/A (monthly)`, `Period: domain.PeriodMonthly`).
- Exclude on-demand spend (`spendLimitUsage`) from quota percentages.
- Test explicit Cursor dispatch in `Runner.querierFor(domain.ProviderCursor)` and
  unknown-provider/missing-wiring no-launch behavior; never fall through to
  Codex or allow offline E2E tests to invoke a real `cursor-agent` binary.

## Acceptance and downstream handoff

Completed on 2026-10-18 ([cursor compatibility](../verification/cursor-compatibility.md)):
`TASK-039` qualified the `cursor-agent 2026.10.16-e3cb207` subprocess +
read-only `NODE_OPTIONS` preload hook contract ([REQ-065](../specs/README.md#req-065)),
`TASK-040` integrated `internal/provider/cursor` with offline fake-process tests
covering weekly, monthly, `includedSpend/limit` fallback, API-key, auth-required,
hook-fallback `StatusUnsupported`, timeout, cancellation, and cleanup failure,
and `TASK-041` passed the complete offline and race gates plus bounded
prompt-free live final-executable acceptance (`Cursor default 100%` and `--json`
`cursor:0` `status: "ok"`, `used_percent: 100`, `cleanup_failed: false`)
([REQ-066](../specs/README.md#req-066)). Delivery does not include installation,
release staging, tagging, or publication.

