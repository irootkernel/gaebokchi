# Cursor Agent usage support

This is the execution dossier for [EPIC-013](../roadmap/README.md#epic-013).
The [roadmap](../roadmap/README.md) owns its task membership, order and status.

## Outcome and authority

Read the installed Cursor Agent CLI's subscription quotas for one named default
login and present them through the period sections and JSON v2 delivered by
EPIC-012. Cursor multi-account management is outside this epic.

| Decision | Canonical owner |
|---|---|
| Required metadata interface and qualification | [REQ-065](../specs/README.md#req-065), [ARCH-19](../architecture/README.md#arch-19) |
| Configuration, adapter and positive support | [REQ-066](../specs/README.md#req-066), [ARCH-19](../architecture/README.md#arch-19) |
| Shared quota, display and JSON contracts | [REQ-061](../specs/README.md#req-061) through [REQ-063](../specs/README.md#req-063), [ARCH-17](../architecture/README.md#arch-17) |
| Qualification gaps | [Cursor compatibility](../verification/cursor-compatibility.md) |

## Work map

| Work unit | Input and required handoff |
|---|---|
| [TASK-039](../roadmap/README.md#task-039) | Inspect the installed CLI and qualify a prompt-free metadata entry point, response periods, quota pools and default-login behavior. Freeze the actual contract and fixtures. |
| [TASK-040](../roadmap/README.md#task-040) | Add configuration, executable selection, the provider adapter and focused fake-process tests against that contract. |
| [TASK-041](../roadmap/README.md#task-041) | Verify the real executable, native period display, failures and regressions; assess the complete epic acceptance. |

## Entry gate and stop conditions

EPIC-012 must be complete before TASK-039 begins. The repository runs one epic
at a time. A blocked agy gate therefore also prevents Cursor execution until
the owner approves a different roadmap dependency or scope.

Cursor Agent 2026.10.14-0f9edde has interactive usage code, but its inspected
command surface does not expose that data through a qualified noninteractive
JSON command. `status` and `about` JSON do not supply quota data.
TASK-039 must qualify an actual installed metadata path before TASK-040 starts.
If it cannot, mark the task and epic blocked; do not substitute a direct API
client, a package-internals wrapper, or a model prompt.

## Implementation boundaries

- Use `cursor-agent` as the default executable; `agent` can name another tool.
- Query only the existing default CLI login. `CURSOR_CONFIG_DIR` selects
  configuration and does not establish a separate macOS credential namespace.
  TASK-039 freezes ambient override filtering, update suppression and the
  credential boundary on each claimed platform under ARCH-19.
- Keep authentication and refresh inside the CLI. Do not read its auth files,
  Keychain entries or tokens, or accept account tokens as configuration.
- Keep the shared concurrency, per-target deadline, cancellation and cleanup
  contracts. Do not terminate provider-owned or unrelated processes.
- Use explicit response period metadata. The installed pager's monthly title
  does not prove that every account has a monthly quota.
- Exclude on-demand spend from quota results. Do not turn a currency
  total, activity statistic or session token count into a subscription percentage.
- Probe only the bounded, authorized metadata path. Login, account switching,
  upgrades, model work and provider configuration writes are excluded.

## Acceptance and downstream handoff

The epic needs a qualified CLI contract, a positive prompt-free final-candidate
query, complete offline checks and existing-provider regression coverage.
Test weekly and monthly fixtures, absent period metadata, missing percentages,
auth-required, unsupported CLI, timeout, cancellation and cleanup failure.
Do not claim a live period or plan variant exercised only by synthetic fixtures.

Update user-facing usage and configuration documentation when the implementation
lands, marked final live acceptance pending until TASK-041. Delivery does not
include installation or publication.
At closeout, replace this epic's `Detailed SOT` with canonical outcome links and
inventory references before proposing removal of the dossier and index entry.
