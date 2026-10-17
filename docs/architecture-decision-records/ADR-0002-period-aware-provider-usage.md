# ADR-0002: Weekly model-pool rows and single-account CLI credential ownership for agy and Cursor

- **Status:** Accepted on 2026-10-15; revised on 2026-10-17 after live `agy 1.4.0`
  qualification and owner scope decision (weekly-only continuity, single default
  `agy` login, and two weekly rows for `gemini` and `claude/gpt`).
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

## Decision

1. **Preserve the weekly-only output and existing JSON contract.** Keep the
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
5. **Enforce runtime version, environment, and auth fast-fail guards.** Verify
   `agy --version` (`>= 1.2.15`) within the target timeout budget before running
   `/usage`, strip ambient `ANTIGRAVITY_*`, `JETSKI_*`, `AGY_*`, and API-key/ADC
   overrides while setting `AGY_CLI_DISABLE_AUTO_UPDATE=true`, and terminate the
   child immediately if `Authentication required` appears on stderr. Use explicit
   provider dispatch in `Runner.querierFor` with no implicit Codex fallback.

## Alternatives considered

- **Multi-period sections (`WEEKLY`, `5 HOURS`, `MONTHLY`) and JSON v2:**
  Rejected by the owner on 2026-10-17 because `gaebokchi`'s purpose is a concise
  weekly usage view, and all target providers (`ChatGPT`, `Grok`, `Claude`,
  `agy`) provide weekly usage.
- **Multi-account `agy` via synthetic `HOME` directories:** Rejected by the owner
  on 2026-10-17; only the single main `agy` account is needed.
- **Combining Gemini and Claude/GPT pools into one row:** Rejected because `agy`
  enforces separate weekly quotas and reset times for `gemini-weekly` and
  `3p-weekly`.

## Consequences

- `internal/render/table.go`, `internal/render/terminal.go`,
  `internal/render/live.go`, and `internal/render/json.go` retain their existing
  public contracts without breaking changes.
- `TASK-034`'s metadata qualification for the single main `agy` login is satisfied
  by the 2026-10-17 `agy 1.4.0` evidence recorded in
  [agy-compatibility.md](../verification/agy-compatibility.md).
- `EPIC-012` implementation tasks (`TASK-035` through `TASK-038`) focus on exact
  remaining-fraction conversion, two-row `agy` target expansion (`gemini` and
  `claude/gpt`), the `internal/provider/agy` shared-query adapter, explicit
  provider dispatch, and final offline/live verification.
