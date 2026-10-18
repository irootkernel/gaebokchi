# ZCode compatibility and qualification record

## Current finding

On 2026-10-18, `zcode 0.16.9` (`darwin/arm64`) was qualified through static
bundle inspection and an owner-requested read-only live probe. Because `zcode`
exposes no standalone public quota subcommand and ZCode Coding Plan
subscriptions enforce a **5-hour rolling token/credit quota window** (`unit == 3`,
`number == 5`) rather than a weekly subscription quota, the owner approved:

1. **Subprocess (`<zcode-bin> doctor --json`) + read-only `NODE_OPTIONS` preload
   hook** ([REQ-067](../specs/README.md#req-067), [ARCH-20](../architecture/README.md#arch-20))
   that resolves the preferred/active ZCode Coding Plan account first, reads
   only an already-stored Coding Plan API key (`:api-key`) for that selected
   account from `zcode`'s own credential store (`createSharedZCodeCredentialStore`),
   and performs a single read-only `GET /api/monitor/usage/quota/limit` request
   through `zcode`'s HTTP client (`createDefaultHttpClient`) without invoking
   `ProjectAccessTokenClient`, falling back to another family's key when the
   selected account is OAuth-only, or mutating disk state.
2. **Two-section usage presentation** ([REQ-068](../specs/README.md#req-068),
   [ARCH-10](../architecture/README.md#arch-10), [ARCH-20](../architecture/README.md#arch-20)):
   weekly providers (`codex`, `grok`, `claude`, `agy`, `cursor`) remain in the
   top **Weekly usage** section (`🐡 gaebokchi · weekly usage` / `WEEKLY USAGE`),
   while `zcode` is rendered in a bottom **5h usage** section
   (`🐡 gaebokchi · 5h usage` / `5H USAGE`) when configured.

## Static package inspection (`zcode 0.16.9`)

- **Wrapper and runtime:** `~/.local/bin/zcode` executes
  `exec node /Applications/ZCode.app/Contents/Resources/glm/zcode.cjs "$@"`.
  Bundled provider metadata lives at
  `/Applications/ZCode.app/Contents/Resources/config/provider/zcode-builtin.json`
  (`../config/provider/zcode-builtin.json` relative to `zcode.cjs`).
- **Command surface:** `zcode --help` exposes `tui`, `app-server`, `agent-server`,
  `login`, `logout`, `doctor`, and `mcp`. Plain `zcode doctor --json` outputs
  CLI/runtime/ripgrep diagnostics without account quota metadata.
- **Internal quota monitor (`zcode.cjs` -> `fetchQuotaLimit`):**
  - Sends `GET <origin>/api/monitor/usage/quota/limit` (`https://api.z.ai` for
    `zai` or `https://bigmodel.cn` for `bigmodel`) with header
    `Authorization: <codingPlanApiKey>` and `Accept: application/json`.
  - Parses `data.limits[]`, where each item contains `type`
    (`"TOKENS_LIMIT"`, `"CREDIT_LIMIT"`, `"TIME_LIMIT"`), `unit` (`3` = hours,
    `5` = months), `number` (`5` for the 5-hour window), `percentage`, `usage`,
    `currentValue`, `remaining`, and `nextResetTime` (epoch milliseconds).
- **Credential boundary (`zcode.cjs` -> `createSharedZCodeCredentialStore`):**
  - `~/.zcode/v2/credentials.json` stores AES-256-GCM encrypted entries (`enc:v1:...`),
    including `oauth:active_provider` (`"zai"` or `"bigmodel"`),
    `oauth:<family>:user_info` (`{"id":"<uuid>",...}`), and the provisioned
    Coding Plan API key under
    `account-provider:coding-plan:account:<family>-individual-coding-plan:account:<encodeURIComponent(id)>:api-key`.
  - Calling `createStandaloneProviderRuntimeHeadersPort.refreshBeforeModelRequest`
    when `auth-source` is `"oauth"` invokes `ProjectAccessTokenClient`, which
    sends `POST /api/auth/z/login`, `POST .../api_keys` (creating a server-side
    `zcode-api-key` if absent), and `POST .../access_tokens`. To remain strictly
    read-only, `preload.cjs` never calls `createStandaloneProviderRuntimeHeadersPort`
    or `ProjectAccessTokenClient`; it resolves the active/preferred account first,
    uses only an already-stored Coding Plan `:api-key` for that selected account
    (returning `unauthenticated` -> `AUTH REQUIRED` if the selected account only
    has OAuth tokens without a stored `:api-key`), and blocks any non-`GET`
    request or path other than `/api/monitor/usage/quota/limit`.
  - `readRawCredentialRecord` (`K0n`) backs up corrupt `credentials.json` files
    via `backupCorruptFile` (`MJt`), and SEA builds can materialize
    `zcode-builtin.json` under `dataBaseDir`. `preload.cjs` pre-validates
    `credentials.json` before `rawStore.load`, materializes any SEA asset only
    inside its own private temporary `preloadDir`, and wraps `node:fs` (sync and
    callback APIs) and `node:fs/promises` to reject any filesystem mutation
    outside `preloadDir` with `EROFS`.

## 2026-10-18 bounded live qualification (`darwin/arm64`)

- **Provider:** `zcode` (`domain.ProviderZCode`, `DisplayName() == "ZCode"`)
- **Observed at:** `2026-10-18`
- **OS / architecture:** `darwin/arm64` (macOS, Go `1.27.0`)
- **Executable identity and version:** `~/.local/bin/zcode` ->
  `/Applications/ZCode.app/Contents/Resources/glm/zcode.cjs` (`zcode --version`
  -> `0.16.9`)
- **Invocation and preload hook contract:** `<zcode-bin> doctor --json` (default
  `zcode`) with stdin closed (`os.DevNull`), a private mode-`0700` temporary
  working directory (`internal/process`), and `NODE_OPTIONS=--require=<preloadPath>`
  pointing to a read-only `preload.cjs` inside a private mode-`0700` temporary
  directory (`os.MkdirTemp("", "gaebokchi-zcode-")`) removed on `defer` (folding
  any removal error into `Result.CleanupFailed`).
- **Child environment policy (`envfilter.ZCode()`):** Preserve `HOME` so `zcode`
  locates `~/.zcode/v2/credentials.json`; strip all `ZCODE_*`, `ZAI_*`, and
  `BIGMODEL_*` variables and any ambient parent `NODE_OPTIONS`, then set
  `NODE_OPTIONS=--require=<preloadPath>`.
- **Read-only and privacy boundary (`ARCH-09`, `ARCH-20`):** Zero model turns,
  zero agent sessions, zero `POST` requests or server-side key creation, zero
  writes outside `preloadDir`, and no PII or raw credentials in stdout or
  stderr.

Sanitized captured response (`zcode 0.16.9`, `darwin/arm64`, provenance: `captured`):

```json
{
  "status": "authenticated",
  "isAuthenticated": true,
  "family": "zai",
  "quota": {
    "httpStatus": 200,
    "code": 200,
    "success": true,
    "limits": [
      {
        "type": "TIME_LIMIT",
        "unit": 5,
        "number": 1,
        "percentage": 1,
        "usage": 100,
        "currentValue": 1,
        "remaining": 99,
        "nextResetTime": null
      },
      {
        "type": "TOKENS_LIMIT",
        "unit": 3,
        "number": 5,
        "percentage": 75,
        "usage": 40000000,
        "currentValue": 30016792,
        "remaining": 9983208,
        "nextResetTime": "1792341573056"
      }
    ]
  },
  "quotaError": null
}
```

Key verified contract points:
1. **Strict 5-hour quota selection:** Select the entry in `quota.limits[]` where
   `type` is `"TOKENS_LIMIT"` or `"CREDIT_LIMIT"`, `unit == 3` (hours), and
   `number == 5` (5-hour window). Non-5h hourly windows (such as `unit == 3,
   number == 1`) and monthly tool limits (`type == "TIME_LIMIT", unit == 5`) are
   never treated as 5-hour usage.
2. **Percentage and reset mapping:** Use `percentage` when present in `[0, 100]`,
   falling back to `currentValue / usage * 100` or
   `(usage - remaining) / usage * 100` when `usage > 0`. Parse optional positive
   epoch milliseconds in `nextResetTime` into `ResetAt` (truncated to whole
   seconds UTC).
3. **Two-section rendering:** When both weekly targets and `zcode` (`PeriodFiveHour`)
   targets are present, the interactive terminal view renders
   `🐡 gaebokchi · weekly usage` followed by a blank line and
   `🐡 gaebokchi · 5h usage`, and the plain table renders the `WEEKLY USAGE`
   table followed by a blank line and the `5H USAGE` table. `--json` preserves
   the flat `{"results":[...]}` array with `target_id: "zcode:0"` and
   `provider: "zcode"`.
4. **Live final-executable verification (`darwin/arm64`, 2026-10-18):**
   - Plain table and interactive PTY runs with `codex`, `claude`, `agy`,
     `cursor`, and `zcode` configured exited `0` with `0` stderr bytes and
     rendered `ZCode  default  75%` (`↻ 42m` in interactive mode) under the
     `5H USAGE` / `🐡 gaebokchi · 5h usage` section.
   - Offline and race suites (`make test` across `test-prepare`, `test-unit`,
     `test-int`, and `test-e2e`) passed completely.
