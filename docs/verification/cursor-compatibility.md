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

With owner authorization, a single read-only probe ran
`cursor-agent status --format json` with `NODE_OPTIONS=--require=<preload.cjs>`
intercepting `./src/commands/status.ts` (`handleStatus`). The hook checked
`credentialManager.getAccessToken()`, `getRefreshToken()`, and `getApiKey()`,
called `dashboardClient.getPlanInfo` and `dashboardClient.getCurrentPeriodUsage`,
omitted all PII (`email`, `userId`, `firstName`, `lastName`, `teamId`,
`teamName`, tokens, and keys), and wrote a single compact JSON line via
`fs.writeSync(1, ...)`.

Sanitized captured response (`cursor-agent 2026.10.16-e3cb207`, `darwin/arm64`):

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
   (`1792591073284`) matches `currentPeriodUsage.billingCycleEndMs`
   (`1792591073284`), whereas `planInfo.billingCycleEndMs` (`1794664361314`) is
   the subscription renewal timestamp. Weekly `ResetAt` must use
   `includedUsageResetsAtMs` when positive, falling back to
   `currentPeriodUsage.billingCycleEndMs`.
3. **Single-row `Included` usage percentage:** Weekly plans report percentage
   consumption directly in `planUsage.totalPercentUsed` (labeled `Included` in
   Cursor's `/usage` view) while `includedSpend` and `limit` remain `0`. Match
   `./src/usage/usage-data.ts`: use `planUsage.totalPercentUsed` when present,
   or `includedSpend / limit * 100` when `totalPercentUsed` is absent and
   `limit > 0`.
4. **Synchronous stdout flush:** Because `cursor-agent` exits inside
   `handleStatus`'s `finally { process.exit(exitCode) }` block, the preload hook
   must write compact JSON synchronously via `fs.writeSync(1, ...)` rather than
   buffered console helpers.

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
- **[TASK-041](../roadmap/README.md#task-041):** Runs the complete offline and
  race gates plus final-executable verification and closes
  [REQ-066](../specs/README.md#req-066) and [EPIC-013](../roadmap/README.md#epic-013).
