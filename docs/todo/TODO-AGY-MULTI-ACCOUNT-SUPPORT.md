# agy multi-account support and period-aware usage

This is the execution dossier for [EPIC-012](../roadmap/README.md#epic-012).
The [roadmap](../roadmap/README.md) owns its task membership, order and status.
Registration defines future work; it does not qualify the installed agy CLI.

## Outcome and authority

Query at least two independently authenticated agy accounts through their CLI,
and show each reported quota in the weekly, five-hour or monthly section.
The same result model and output will serve the existing providers and Cursor.

| Decision | Canonical owner |
|---|---|
| Required agy metadata interface | [REQ-060](../specs/README.md#req-060), [ARCH-18](../architecture/README.md#arch-18) |
| Quota identity and period semantics | [REQ-061](../specs/README.md#req-061), [ARCH-17](../architecture/README.md#arch-17) |
| Section layout and query progress | [REQ-062](../specs/README.md#req-062), [ARCH-17](../architecture/README.md#arch-17) |
| JSON v2 and migration | [REQ-063](../specs/README.md#req-063), [ADR-0002](../architecture-decision-records/ADR-0002-period-aware-provider-usage.md) |
| Account selection and isolation | [REQ-064](../specs/README.md#req-064), [ARCH-18](../architecture/README.md#arch-18) |
| Qualification gaps | [agy compatibility](../verification/agy-compatibility.md) |

## Work map

| Work unit | Input and required handoff |
|---|---|
| [TASK-034](../roadmap/README.md#task-034) | Resolve the installed CLI's metadata entry point, authentication namespace, quota schema and two-account isolation. Deliver the qualified contract and sanitized fixtures before production work. |
| [TASK-035](../roadmap/README.md#task-035) | Use that contract to introduce account results containing multiple quota results; migrate existing provider normalization without inventing missing periods. |
| [TASK-036](../roadmap/README.md#task-036) | Render the new model in period sections and JSON v2, with executable tests for ordering, failures and terminal behavior. |
| [TASK-037](../roadmap/README.md#task-037) | Wire the qualified agy account selector, environment policy and adapter into configuration and execution. |
| [TASK-038](../roadmap/README.md#task-038) | Qualify the integrated executable, two distinct agy accounts and existing-provider regressions; assess the complete epic acceptance. |

Follow the roadmap sequence. Each implementation task includes its focused tests
and canonical documentation updates; the final task does not absorb missing
component verification.

## Entry gate and stop conditions

The installed agy 1.3.5 help does not establish a usable metadata-only quota
interface. Neither a directory variable in the binary nor a configured alias
proves that Google login state is isolated. These remain qualification gaps.

TASK-034 passes only when both the metadata interface and two-account isolation
are demonstrated. If either is unavailable, record the exact missing capability,
mark the task and epic blocked in the roadmap, and stop before TASK-035.
A blocker report does not satisfy the positive acceptance criteria.
The canonical REQ-060/TASK-034 gate also requires a measured known-period quota
for each account, CLI-reported identity isolation and a frozen selector/update
policy. Unknown-only results do not pass. If the selector is not path-shaped,
resolve the configuration contract before TASK-035; do not invent a home variable.

The account homes must already be authenticated through agy outside the query.
Any live probe needs its own bounded authorization. Do not install or upgrade a
CLI, log in, switch a shared active account, copy credentials, or repair provider
state as part of qualification or the product query.

## Implementation boundaries

- Keep the CLI credential owner, subprocess, deadline and cleanup rules of
  [ARCH-09](../architecture/README.md#arch-09) and
  [ARCH-05](../architecture/README.md#arch-05).
- Do not use PTY or UI scraping, direct provider HTTP requests, application
  Keychain access, token parsing, custom OAuth, or model prompts.
- Do not treat `AGY_ACCOUNT`, `ANTIGRAVITY_APP_DATA_DIR`,
  `JETSKI_APP_DATA_DIR`, or an invented `AGY_HOME` as an authenticated account
  selector without the qualification required by ARCH-18.
- Do not ship an `UNSUPPORTED` placeholder as completed agy support.
- Keep weekly, five-hour and monthly quotas separate, including model pools;
  never average their percentages or select a maximum as the account's usage.
- Version and document the JSON change. Update the public README, configuration
  guide and test-stage mapping when their corresponding implementation lands.
  Keep final live acceptance pending in provider-support copy until TASK-038.

## Acceptance and downstream handoff

The epic needs a qualified CLI contract, all component tests, a full offline
gate and a positive prompt-free run of the final candidate against two distinct
agy logins. Verification must cover concurrent isolation, one-account failure,
unchanged default login selection, cancellation and owned-child cleanup.
Preserve the results of unaffected existing providers.
The canonical local epic gate requires a measured known-period quota from each
account. Other periods may remain fixture-only, with that limit disclosed.
TASK-035/036 own their complete offline REQ-061 through REQ-063 scope; TASK-038
rechecks it rather than postponing missing component work. ARCH-17 owns the
existing-provider mappings, status matrix and app-owned meter ID rules.

Handoff to EPIC-013 consists of the canonical multi-quota model, period output,
JSON v2 contract, extension points and verified safety behavior. Cursor consumes
those canonical outcomes; it does not depend on retaining this dossier.
Installation, release staging, tagging and publication are separate work.

At closeout, promote durable decisions to their owners and replace this epic's
`Detailed SOT` link with canonical outcome links. Inventory references before
proposing removal of this dossier and its TODO index entry.
