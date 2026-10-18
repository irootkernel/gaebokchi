# Provider extension preflight: 2026-10-16 (with 2026-10-17 agy and 2026-10-18 Cursor supplements)

## Result and scope

**Result: `agy 1.4.0` single-account metadata contract qualified on 2026-10-17
and `cursor-agent 2026.10.16-e3cb207` subprocess + read-only preload hook
contract qualified on 2026-10-18.** On 2026-10-17 and 2026-10-18, the owner
authorized bounded live `agy` and `cursor-agent` probes and locked in the
`EPIC-012` and `EPIC-013` scopes:
1. `gaebokchi` remains **weekly-only** (keeping the existing 3-column plain
   table, interactive terminal view, and JSON `results` snapshot).
2. `agy` and `Cursor` use **only the single main (default) CLI login** (no
   multi-account `HOME` or config-dir credential isolation).
3. A single `agy --print /usage --output-format json --print-timeout 15s` query
   populates **two weekly rows** under `agy`: `gemini` (`id: "gemini-weekly"`)
   and `claude/gpt` (`id: "3p-weekly"`).
4. For `Cursor`, the former "pure public CLI flags only" restriction is retired
   in favor of `<cursor-bin> status --format json` with a read-only
   `NODE_OPTIONS` preload hook on `./src/commands/status.ts` that calls
   `cursor-agent`'s bundled `dashboardClient.getPlanInfo` and
   `dashboardClient.getCurrentPeriodUsage`, populating **one weekly row**
   (`cursor:0`, `QuotaID: "cursor_included"`) from `planUsage.totalPercentUsed`
   when the plan period is weekly (`INCLUDED_USAGE_PERIOD_WEEKLY` or a 7-day
   window) and reporting `N/A (monthly)` when monthly.

No credentials, CLI installations, or release states were modified during these
preflights.

## Observations and limitations

| Check | Evidence and result | Scope and limitations |
|---|---|---|
| Repository baseline | HEAD `751c067` (2026-10-17) / `edc1eca` (2026-10-18 after `EPIC-012` completion). | Documentation and planning preflight only. |
| `agy` version/help (2026-10-16 & 2026-10-17) | `agy --version` returned `1.4.0` on `darwin/arm64`. `agy changelog` confirmed v1.1.11 added turn-free JSON `/usage` responses in print mode and v1.2.15 normalized piped/JSON reset timestamps to UTC. | Requires runtime version guard `>= 1.2.15` before invoking `--print /usage`. |
| `agy` live metadata probe (2026-10-17) | Executed `agy --print /usage --output-format json --print-timeout 15s` with `AGY_CLI_DISABLE_AUTO_UPDATE=true`, stdin closed, empty private cwd, and parent `ANTIGRAVITY_*`/`JETSKI_*`/`AGY_*` stripped. Exited `0` in ~5.2s with `status: "SUCCESS"`, `num_turns: 0`, `usage.total_tokens: 0`, `conversation_id: ""`, and zero cwd files. | Returned two weekly groups (`gemini-weekly` and `3p-weekly`) and two 5-hour groups (`gemini-5h` and `3p-5h`) with `remaining_fraction` and RFC 3339 UTC `reset_time`. Sanitized fixture recorded in [agy-compatibility.md](agy-compatibility.md). |
| `agy` absent-login & env probe (2026-10-17) | Unauthenticated `agy` prints `Authentication required. Please visit the URL to log in: ... Waiting for authentication (timeout 60s)...` on stderr and waits up to 60s even with closed stdin, `CI=true`, and `AGY_CLI_NONINTERACTIVE_HEADLESS=1`. `ANTIGRAVITY_APP_DATA_DIR` and `JETSKI_APP_DATA_DIR` are ignored as input selectors. | Adapter must watch bounded stderr and terminate the child immediately when `Authentication required` appears, returning `StatusAuthRequired` (`AUTH REQUIRED`). |
| Cursor static & live preload-hook probe (2026-10-18) | `cursor-agent` symlink points to `2026.10.16-e3cb207` on `darwin/arm64`. Static inspection of `index.js`, `684.index.js`, `5233.index.js`, `5300.index.js`, and `7956.index.js` confirmed `-p /usage` sends a model prompt and plain `status`/`about`/`acp` omit usage, while `status --format json` passes `credentialManager` and `dashboardClient` to `./src/commands/status.ts`. A bounded read-only probe with `NODE_OPTIONS=--require=<preload.cjs>` returned `includedUsagePeriod: 2` (`WEEKLY`), 7-day `billingCycleStartMs`–`billingCycleEndMs` (`604,800,000 ms`), `includedUsageResetsAtMs`, and `planUsage.totalPercentUsed` with zero model turns and all PII omitted. | Sanitized fixture and contract recorded in [cursor-compatibility.md](cursor-compatibility.md); reused by [TASK-039](../roadmap/README.md#task-039) in [EPIC-013](../roadmap/README.md#epic-013). |

## Plan alignment from the 2026-10-17 and 2026-10-18 passes

- **Weekly-only continuity:** `gaebokchi` keeps its existing weekly-only table,
  interactive terminal frame, and JSON `results` schema. Five-hour quotas
  (`window: "5h"`) in `agy` responses are ignored.
- **Single default `agy` login with two weekly rows:** `EPIC-012` no longer
  requires two `agy` accounts or `HOME` relocation. Enabling `agy` in
  `~/.config/gaebokchi/config.yaml` emits two weekly targets (`agy:0` with label
  `gemini` for `gemini-weekly`, and `agy:1` with label `claude/gpt` for
  `3p-weekly`) backed by one shared `agy` subprocess execution per invocation.
- **Single default `Cursor` login with subprocess + read-only preload hook:**
  `EPIC-013` retires the pure public-flag restriction for `cursor-agent` and
  uses `<cursor-bin> status --format json` with a read-only `NODE_OPTIONS`
  preload hook on `./src/commands/status.ts` to read `getPlanInfo` and
  `getCurrentPeriodUsage`, emitting one weekly target (`cursor:0`, configured
  `name`, `QuotaID: "cursor_included"`) when weekly and `N/A (monthly)` when
  monthly ([ADR-0002](../architecture-decision-records/ADR-0002-period-aware-provider-usage.md)).
- **Exact decimal conversion:** `remaining_fraction` $r \in [0, 1]$ is converted
  to `used_percent = 100 * (1 - r)` from the JSON number token using exact
  decimal arithmetic before storing/formatting so integer percentages (such as
  `r = 0.9` -> `10%`) never suffer binary floating-point truncation noise.
- **Explicit provider dispatch:** `Runner.querierFor` removes the implicit Codex
  fallback and dispatches `codex`, `grok`, `claude`, `agy`, and `cursor`
  explicitly.

Sources: [S-17](../architecture/README.md#s-17) through
[S-24](../architecture/README.md#s-24). Canonical records:
[agy compatibility](agy-compatibility.md),
[Cursor compatibility](cursor-compatibility.md), and
[shared runtime guards](../architecture/README.md#provider-extension-qualification).
