# agy single-account weekly usage support

This is the execution dossier for [EPIC-012](../roadmap/README.md#epic-012).
The [roadmap](../roadmap/README.md) owns its task membership, order and status.

## Outcome and authority

Query the single default `agy` CLI login through its qualified metadata-only
interface (`agy --print /usage --output-format json --print-timeout 15s`) and
present its two weekly model pools (`gemini` and `claude/gpt`) as two ordered
weekly rows in `gaebokchi`'s existing 3-column plain table, interactive terminal
view, and `--json` `results` snapshot. Multi-account `agy` isolation and
non-weekly output sections are out of scope.

| Decision | Canonical owner |
|---|---|
| Qualified `agy` metadata interface and version guard | [REQ-060](../specs/README.md#req-060), [ARCH-18](../architecture/README.md#arch-18) |
| Weekly pool extraction and exact remaining-fraction conversion | [REQ-061](../specs/README.md#req-061), [ARCH-17](../architecture/README.md#arch-17) |
| Two-row weekly table and terminal presentation | [REQ-062](../specs/README.md#req-062), [ARCH-17](../architecture/README.md#arch-17) |
| JSON weekly `results` continuity | [REQ-063](../specs/README.md#req-063), [ADR-0002](../architecture-decision-records/ADR-0002-period-aware-provider-usage.md) |
| Single-account configuration and shared one-shot query | [REQ-064](../specs/README.md#req-064), [ARCH-18](../architecture/README.md#arch-18) |
| Qualified `agy 1.4.0` contract and fixture | [agy compatibility](../verification/agy-compatibility.md) |

## Work map

| Work unit | Input and required handoff |
|---|---|
| [TASK-034](../roadmap/README.md#task-034) | Freeze the verified `agy 1.4.0` single-account metadata contract, version guard (`>= 1.2.15`), child environment policy, stderr `Authentication required` fast-fail rule, and sanitized fixture. |
| [TASK-035](../roadmap/README.md#task-035) | Implement bounded exact-decimal `remaining_fraction` conversion (`100 * (1 - r)`) and weekly pool extraction (`gemini-weekly` -> `gemini`, `3p-weekly` -> `claude/gpt`), plus `ProviderAgy` domain types. |
| [TASK-036](../roadmap/README.md#task-036) | Verify two-row `agy` weekly presentation (`agy:0` = `gemini`, `agy:1` = `claude/gpt`) across the plain table, interactive terminal view, and `--json` output. |
| [TASK-037](../roadmap/README.md#task-037) | Wire the single-entry `agy` configuration, `--agy-bin`, explicit provider dispatch, and the shared one-shot `internal/provider/agy` adapter (`sync.Once`). |
| [TASK-038](../roadmap/README.md#task-038) | Run the complete offline gate (`make test`) and qualify the final executable live against the configured providers and default `agy` login. |

Follow the roadmap sequence. Each implementation task includes its focused tests
and canonical documentation updates.

## Entry gate and verified contract

On 2026-10-17, bounded live probes against `agy 1.4.0` on `darwin/arm64`
([preflight](../verification/provider-extension-preflight-2026-10-16.md),
[agy compatibility](../verification/agy-compatibility.md)) confirmed:
- `agy --print /usage --output-format json --print-timeout 15s` exits `0` in
  ~5.2s with `status: "SUCCESS"`, `num_turns: 0`, `usage.total_tokens: 0`,
  `conversation_id: ""`, and no created files.
- `command.data.groups[]` exposes two weekly model pools (`id: "gemini-weekly"`
  and `id: "3p-weekly"`, both with `window: "weekly"`, `remaining_fraction`, and
  RFC 3339 UTC `reset_time`) alongside 5-hour pools (`window: "5h"`).
- `agy` changelog confirms v1.1.11 added turn-free JSON `/usage` responses in
  print mode and v1.2.15 normalized piped/JSON reset timestamps to UTC (minimum
  supported version `>= 1.2.15`).
- Unauthenticated `agy` prints `Authentication required` on stderr and waits up
  to 60s; the adapter must terminate the child immediately upon seeing
  `Authentication required` on stderr.

## Implementation boundaries

- Keep the CLI credential owner, subprocess, deadline and cleanup rules of
  [ARCH-09](../architecture/README.md#arch-09) and
  [ARCH-05](../architecture/README.md#arch-05).
- Query only the single default `agy` login (`HOME` preserved); do not relocate
  `HOME` or use `ANTIGRAVITY_APP_DATA_DIR` / `JETSKI_APP_DATA_DIR` as account
  selectors.
- Strip `ANTIGRAVITY_*`, `JETSKI_*`, `AGY_*`, `GEMINI_API_KEY`,
  `GOOGLE_GEMINI_BASE_URL`, `AGY_ADC_AUTH`, `GOOGLE_APPLICATION_CREDENTIALS`,
  and `GOOGLE_CLOUD_QUOTA_PROJECT` in the child environment, and inject
  `AGY_CLI_DISABLE_AUTO_UPDATE=true`.
- Execute the `agy` CLI at most once per `gaebokchi` invocation (`sync.Once`) to
  populate both `agy:0` (`gemini`) and `agy:1` (`claude/gpt`) rows within one
  target timeout budget.
- Convert `remaining_fraction` using bounded exact decimal token math before
  storing/formatting; ignore `window: "5h"` entries.
- Replace the implicit Codex fallback in `Runner.querierFor` with explicit
  provider dispatch.

## Acceptance and downstream handoff

The epic closes when `make test` passes and a bounded live run of the final
executable returns ordered weekly rows for the configured providers plus both
`agy` rows (`gemini` and `claude/gpt`) with valid percentages, reset countdowns,
zero model turns, and no cleanup failure.

Handoff to [EPIC-013](../roadmap/README.md#epic-013) consists of explicit
provider dispatch and the shared runtime guard conventions. Installation,
release staging, tagging and publication remain separate work.
