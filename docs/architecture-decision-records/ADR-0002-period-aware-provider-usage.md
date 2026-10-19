# ADR-0002: Weekly model-pool rows and single-account CLI credential ownership for agy and Cursor

- **Status:** Accepted on 2026-10-15; revised on 2026-10-17 after live `agy 1.4.0`
  qualification and owner scope decision (weekly-only continuity, single default
  `agy` login, and two weekly rows for `gemini` and `claude/gpt`); revised on
  2026-10-18 after live `cursor-agent 2026.10.16-e3cb207` qualification and owner
  decision to retire the pure public-flag restriction for Cursor in favor of a
  subprocess (`cursor-agent status --format json`) + read-only `NODE_OPTIONS`
  preload hook contract.
- **ZCode amendment (2026-10-18):** [REQ-068](../specs/README.md#req-068)
  and [ARCH-20](../architecture/README.md#arch-20) add a separate bottom 5-hour
  section for ZCode. The weekly-only decision below remains the scope of the
  agy and Cursor deliveries; it does not prohibit that later ZCode extension.
  The flat JSON `results` contract remains unchanged.
- **Delivery:** [EPIC-012](../roadmap/README.md#epic-012) and
  [EPIC-013](../roadmap/README.md#epic-013).
- **Related:** [REQ-060](../specs/README.md#req-060) through
  [REQ-066](../specs/README.md#req-066),
  [ARCH-17](../architecture/README.md#arch-17) through
  [ARCH-19](../architecture/README.md#arch-19).

## Context

`gaebokchi` renders a weekly subscription usage snapshot across configured AI
coding CLIs using three columns in plain text (`AI PROVIDER`, `ACCOUNT`,
`WEEKLY USAGE`), an inline interactive terminal frame with progress bars and
reset countdowns (`↻`), and a machine-readable `--json` `results` array.

Live qualification of `agy 1.4.0` (`agy --print /usage --output-format json
--print-timeout 15s`) established that:
1. `agy` exposes two distinct weekly model-group quotas in one structured JSON
   response: **Gemini Models** (`id: "gemini-weekly"`, `window: "weekly"`) and
   **Claude and GPT models** (`id: "3p-weekly"`, `window: "weekly"`), alongside
   5-hour windows (`window: "5h"`).
2. `agy` stores credentials in the user's default CLI login context and ignores
   `ANTIGRAVITY_APP_DATA_DIR` / `JETSKI_APP_DATA_DIR` as input selectors.
3. The owner confirmed that `gaebokchi` must remain **weekly-only**, use **only
   the single main (default) `agy` login**, and display `agy`'s **Gemini** and
   **Claude/GPT** weekly usage as **two separate rows** (`gemini` and
   `claude/gpt`) in the existing weekly output format.

Static and bounded live qualification of `cursor-agent 2026.10.16-e3cb207` on
2026-10-18 established that:
1. Unlike `claude -p /usage` and `agy --print /usage`, `cursor-agent -p /usage`
   is not a local slash command and sends `"/usage"` as an ordinary user prompt
   to the model; `cursor-agent acp` exposes only `cursor/list_available_models`;
   and plain `cursor-agent status --format json` / `about --format json` omit
   subscription quota and period fields.
2. However, `cursor-agent` is distributed as a Node.js bundle (`node`,
   `index.js`, and chunk files including `684.index.js` and `7956.index.js`)
   whose `status --format json` command already initializes the CLI's own
   `credentialManager` and `dashboardClient` (`aiserver.v1.DashboardService`)
   and passes both to `./src/commands/status.ts` (`handleStatus`).
3. The owner decided on 2026-10-18 to **retire the "pure public CLI flags only"
   restriction for Cursor** and adopt a **subprocess (`cursor-agent status
   --format json`) + read-only `NODE_OPTIONS` preload hook** contract that
   invokes `dashboardClient.getPlanInfo` and `dashboardClient.getCurrentPeriodUsage`
   inside the `cursor-agent` child process, presenting **one row per configured
   Cursor target** (`cursor:0`) for the `Included` allowance (`totalPercentUsed`)
   when the plan period is weekly (`INCLUDED_USAGE_PERIOD_WEEKLY` or a 7-day
   billing cycle) and `N/A (monthly)` when the plan period is monthly.

## Decision

1. **Preserve weekly output for the weekly providers and the existing JSON
   contract.** For the agy and Cursor deliveries, keep the
   current 3-column plain table (`AI PROVIDER`, `ACCOUNT`, `WEEKLY USAGE`),
   the interactive terminal frame (`🐡 gaebokchi · weekly usage`), and the
   existing `--json` `results` snapshot without introducing multi-period
   (`5 HOURS`, `MONTHLY`) sections or a breaking JSON v2 schema. Ignore
   non-weekly (`5h`) windows in `agy` responses.
2. **Use the single default CLI login for `agy` and Cursor.** Neither `EPIC-012`
   (`agy`) nor `EPIC-013` (`Cursor`) manages multiple accounts or relocates
   `HOME`. Credentials, login, and refresh remain inside each provider CLI.
3. **Render `agy`'s two weekly model pools as two ordered weekly rows from one
   shared CLI query.** When `agy` is enabled in `~/.config/gaebokchi/config.yaml`
   (at most one entry, e.g. `- name: default`), target construction emits two
   weekly targets under provider `agy`:
   - `agy:0` with row label `gemini` (`QuotaID: "gemini-weekly"`)
   - `agy:1` with row label `claude/gpt` (`QuotaID: "3p-weekly"`)
   A single `agy --print /usage --output-format json --print-timeout 15s`
   invocation per `gaebokchi` run (`sync.Once`) populates both rows. Because the
   existing terminal renderer blanks repeated provider names on consecutive rows,
   `agy` naturally displays as two lines (`gemini` and `claude/gpt`) just like
   multi-row `ChatGPT`.
4. **Convert `remaining_fraction` using exact decimal token arithmetic.** Derive
   `used_percent = 100 * (1 - remaining_fraction)` from the JSON number token
   before storing or formatting so exact integers (such as `remaining = 0.9` ->
   `10%`) never suffer IEEE-754 floating-point truncation artifacts.
5. **Enforce runtime version, environment, and auth fast-fail guards for `agy`.**
   Verify `agy --version` (`>= 1.2.15`) within the target timeout budget before
   running `/usage`, strip ambient `ANTIGRAVITY_*`, `JETSKI_*`, `AGY_*`, and
   API-key/ADC overrides while setting `AGY_CLI_DISABLE_AUTO_UPDATE=true`, and
   terminate the child immediately if `Authentication required` appears on
   stderr. Use explicit provider dispatch in `Runner.querierFor` with no implicit
   Codex fallback.
6. **Use a subprocess + read-only `NODE_OPTIONS` preload hook for Cursor and
   retire the pure public-flag restriction.** For `EPIC-013`, `internal/provider/cursor`
   writes an embedded read-only preload module (`preload.cjs`) into a private
   `0700` temporary directory, strips ambient `CURSOR_*` and parent
   `NODE_OPTIONS` variables, sets `NODE_OPTIONS=--require=<preloadPath>`, and
   executes `<cursor-bin> status --format json` (default `cursor-agent`). The
   preload module intercepts `./src/commands/status.ts` (`handleStatus`),
   verifies authentication via `credentialManager`, and calls
   `dashboardClient.getPlanInfo` and `dashboardClient.getCurrentPeriodUsage`
   without model turns or PII output (`email`, `userId`, `teamId`, tokens are
   omitted). When `includedUsagePeriod` is weekly (`2` /
   `INCLUDED_USAGE_PERIOD_WEEKLY`, or `0` / `UNSPECIFIED` with a 7-day
   `billingCycleEnd - billingCycleStart` window), `gaebokchi` reports one weekly
   row (`cursor:0`, configured `name`, `QuotaID: "cursor_included"`,
   `Source: "cursor_cli_usage"`) using `planUsage.totalPercentUsed` (or
   `includedSpend / limit * 100` when `totalPercentUsed` is absent and
   `limit > 0`) and `includedUsageResetsAt` (falling back to
   `currentPeriodUsage.billingCycleEnd`). When the period is monthly (`1` /
   `INCLUDED_USAGE_PERIOD_MONTHLY` or a monthly cycle window), `gaebokchi`
   reports `StatusNotApplicableMonthly` (`N/A (monthly)`). If a future CLI
   version does not trigger the hook, plain `status --format json` output
   degrades cleanly to `StatusUnsupported`.

## Alternatives considered

- **Multi-period sections (`WEEKLY`, `5 HOURS`, `MONTHLY`) and JSON v2:**
  Rejected by the owner on 2026-10-17 because `gaebokchi`'s purpose is a concise
  weekly usage view, and all target providers (`ChatGPT`, `Grok`, `Claude`,
  `agy`, `Cursor`) provide weekly usage.
  The later ZCode amendment permits its separate `5H USAGE` section while
  retaining weekly rows for these providers and the flat JSON snapshot.
- **Multi-account `agy` or `Cursor` via synthetic `HOME` / config directories:**
  Rejected by the owner; only the single default CLI login is used for both
  providers.
- **Combining Gemini and Claude/GPT pools into one row for `agy`:** Rejected
  because `agy` enforces separate weekly quotas and reset times for
  `gemini-weekly` and `3p-weekly`.
- **Keeping `EPIC-013` blocked on a hypothetical pure public `cursor-agent`
  usage flag:** Rejected by the owner on 2026-10-18 because `cursor-agent`
  already ships the complete `credentialManager` and `dashboardClient` runtime
  inside its `status --format json` execution path, allowing a safe, read-only
  subprocess preload hook without `gaebokchi` touching credentials or network
  clients directly.
- **Splitting Cursor into multiple rows (`Included`, `Auto`, `API`):** Rejected
  by the owner on 2026-10-18 in favor of a single configured Cursor row
  displaying the combined `Included` (`totalPercentUsed`) allowance.

## Consequences

- `internal/render/table.go`, `internal/render/terminal.go`,
  `internal/render/live.go`, and `internal/render/json.go` retain their existing
  public contracts without breaking changes.
- `TASK-034`'s metadata qualification for the single main `agy` login is satisfied
  by the 2026-10-17 `agy 1.4.0` evidence recorded in
  [agy-compatibility.md](../verification/agy-compatibility.md), and `EPIC-012` is
  complete.
- `TASK-039` reuses the 2026-10-18 `cursor-agent 2026.10.16-e3cb207` subprocess +
  read-only preload hook qualification recorded in
  [cursor-compatibility.md](../verification/cursor-compatibility.md), and
  `EPIC-013` (`TASK-039` through `TASK-041`) proceeds directly to the
  `internal/provider/cursor` adapter, single-entry `cursor` configuration,
  `--cursor-bin` flag, and offline/live verification without waiting on an
  upstream CLI flag change.
