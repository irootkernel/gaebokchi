# Gaebokchi (개복치): Requirements

Version: 0.1 specification  
Status: CORRECTIVE (residual findings open) — the confirmation review of `c340245` found five residual defects (discarded cleanup failures, unqualified forced group termination, incomplete Unicode whitespace normalization, a suffix-rule mismatch, stale roadmap/acceptance/packaging state). Affected requirements are reopened below pending TASK-022 through TASK-025; REQ-048 remains the separate open live gate  
Prepared: 2026-09-22  
Project: Gaebokchi (개복치)  
Ecosystem: Aquarium tool  
Repository and executable: `gaebokchi`

## Document contract

This document owns **what must be delivered**. [architecture.md](architecture.md) owns implementation structure, data semantics, and protocol boundaries. [plan.md](plan.md) owns execution order, task scope, and completion evidence. Read the [execution rules](plan.md#plan-execution) before starting work.

The three documents belong together at the repository root. Explicit anchors and IDs are stable. An implementation AI must not weaken a requirement merely because an upstream CLI is unsupported. Resolve contradictions across the three documents in the same task-scoped change.

The original campaign checked all 50 requirements, and the TASK-014 release audit rechecked them against the delivered tree at commit `f8b180d`. A later adversarial audit of executable-level behavior superseded that result: 17 requirements were unchecked in TASK-015 with supersession notes naming the defect and the re-qualifying task under [EPIC-007](plan.md#epic-007). TASK-021 re-verified each full acceptance scope against the corrected tree and restored 16 of them with re-qualification notes; REQ-048 remains open because the provider-display comparison requires owner authorization that was not available. Unaffected requirements kept their original evidence throughout.

## Product outcome

Build **Gaebokchi (개복치)**, a small, read-only Go CLI in the **Aquarium tool ecosystem**, for **Codex multi-account plus Grok single-account**. Reuse existing CLI login state instead of implementing account linking. Claude is explicitly outside v0.1.

The project name is **Gaebokchi (개복치)**; the repository basename and executable are `gaebokchi`, as defined in [ARCH-01](architecture.md#arch-01). The default invocation is `gaebokchi`. Illustrative output, not measured account data:

```text
AI PROVIDER  ACCOUNT                WEEKLY USAGE
ChatGPT      account1@example.com   37.0%
ChatGPT      account2@example.com   12.0%
Grok         default                54.0%
```

The requested final scope is **header plus three account rows**, not the five-line table discussed before Claude was removed. Accounts retain separate rows even if they belong to the same provider or resolve to the same email.

## Scope boundaries

The checklist below defines the final v0.1 product, not speculative future-provider support. No Claude adapter, UI scraping, PTY, API-cost dashboard, browser session extraction, custom OAuth, model prompting, or background refresh belongs to this release. No runtime configuration file is needed; explicit flags provide the limited customization.

Provider compatibility is an implementation gate. Published documentation/source can establish a candidate interface, but only a real prompt-free probe establishes that the installed binary exposes it. In particular, an internal Grok billing handler is not proof of a usable stdio endpoint. See [ARCH-12](architecture.md#arch-12) and [TASK-001](plan.md#task-001).

## Acceptance checklist

### Product scope

<a id="req-001"></a>
- [x] **REQ-001: Native Go CLI.** Provide an executable named `gaebokchi` for macOS and Linux. Use a small Go codebase and the installed `codex` and `grok` executables; a Go installation is not required to run a distributed binary.
  - - Architecture: [ARCH-02](architecture.md#arch-02). Owner: [TASK-012](plan.md#task-012). Verification: [V-BUILD](architecture.md#v-build). Evidence: [docs/verification/TASK-012.md](verification/TASK-012.md) — self-contained CGO-free binaries for macOS/Linux; `gaebokchi` executable with version metadata.

<a id="req-002"></a>
- [x] **REQ-002: Default account set.** With no account flags, query exactly three targets, in this order: Codex at `~/.codex`, Codex at `~/.codex-hsy`, and the existing default Grok login. Render the header and those three rows, even when individual targets fail.
  - - Architecture: [ARCH-03](architecture.md#arch-03). Owner: [TASK-010](plan.md#task-010). Verification: [V-OUTPUT](architecture.md#v-output). Evidence: [docs/verification/TASK-010.md](verification/TASK-010.md) — default run queries exactly codex:0/codex:1/grok:0 and renders header+3 rows; per-target failures keep their rows.

<a id="req-003"></a>
- [x] **REQ-003: Meaning of weekly usage.** Display the percentage already consumed in a provider-reported weekly subscription quota. Do not present remaining quota, local token totals, API spending, or an inferred calendar-week calculation as this value. `ChatGPT` identifies the Codex-backed row, not an aggregation of every ChatGPT feature.
  - Architecture: [ARCH-01](architecture.md#arch-01). Owner: [TASK-006](plan.md#task-006). Verification: [V-QUOTA](architecture.md#v-quota). Evidence: [docs/verification/TASK-006.md](verification/TASK-006.md) — the only metric surfaced is the provider-reported weekly `usedPercent`; credits, token summaries, and reset times never feed the value; `ChatGPT` labels the Codex row via `ProviderCodex.DisplayName`.

<a id="req-004"></a>
- [x] **REQ-004: One-shot operation.** Perform one finite snapshot operation and exit. Do not introduce an application daemon, watch mode, polling, scheduled refresh, retry loop, usage history, or persistent usage cache. Blocking reads of protocol messages are permitted.
  - - Architecture: [ARCH-05](architecture.md#arch-05). Owner: [TASK-009](plan.md#task-009). Verification: [V-RUNNER](architecture.md#v-runner). Evidence: [docs/verification/TASK-009.md](verification/TASK-009.md) — one bounded Run per invocation, exactly one query attempt per target, no retry/daemon/cache; the CLI exits after rendering.

<a id="req-005"></a>
- [x] **REQ-005: Real provider support.** Qualify both providers through real, prompt-free CLI queries before claiming v0.1 complete. An `UNSUPPORTED` fallback, a synthetic fixture, or a handler found only in source is not evidence of working Grok integration.
  - - Architecture: [ARCH-12](architecture.md#arch-12). Owner: [TASK-013](plan.md#task-013). Verification: [V-LIVE](architecture.md#v-live). Evidence: [docs/verification/live-acceptance.md](verification/live-acceptance.md) — live prompt-free queries through both installed CLIs via the delivered binary; no UNSUPPORTED/fixture fallback used.

### CLI and output

<a id="req-006"></a>
- [ ] **REQ-006: Small documented command surface.** Implement only the v0.1 flags in ARCH-03, including help, version, repeatable Codex home overrides, binary overrides, Grok display alias, and per-target timeout. Help/version must not launch a provider process. Reject unknown flags and positional arguments.
  - - Architecture: [ARCH-03](architecture.md#arch-03). Owner: [TASK-010](plan.md#task-010). Verification: [V-OPTIONS](architecture.md#v-options). Evidence: [docs/verification/TASK-002.md](verification/TASK-002.md) parser matrix plus TASK-010 help/version/no-launch and exit-code tests.
  - Superseded 2026-09-22 by the corrective audit: binary overrides given as explicit relative paths fail from the neutral child cwd, and a whitespace-only `--grok-account` alias is accepted. Re-qualification: [TASK-017](plan.md#task-017) (path resolution) and [TASK-019](plan.md#task-019) (alias validation).
  - Re-qualified 2026-09-22 under EPIC-007: explicit relative binary overrides resolve against the invocation cwd (unit + end-to-end wrapper-script test), and aliases normalize before validation with blank-after-normalization rejected as a usage error before launch. Checkbox restored.

  - Reopened 2026-09-22 by the confirmation review of `c340245`: whitespace built from Unicode spaces (U+3000, U+205F, U+1680) bypassed alias validation. Re-qualification: [TASK-023](plan.md#task-023).
<a id="req-007"></a>
- [x] **REQ-007: Codex home overrides.** When one or more `--codex-home` flags are supplied, replace the two default Codex targets with those homes in flag order. Keep one Grok target last. Do not silently discover additional accounts or use the parent `CODEX_HOME` as an implicit override.
  - Architecture: [ARCH-03](architecture.md#arch-03). Owner: [TASK-002](plan.md#task-002). Verification: [V-OPTIONS](architecture.md#v-options). Evidence: [docs/verification/TASK-002.md](verification/TASK-002.md) (replacement order, ambient CODEX_HOME ignored, Grok last).

<a id="req-008"></a>
- [x] **REQ-008: Safe path handling.** Expand only the current-user `~` or `~/` prefix, resolve relative paths against the invocation directory, and normalize paths before use. Reject empty paths and duplicate normalized homes, including existing symlink aliases. Do not create missing provider home directories.
  - Architecture: [ARCH-03](architecture.md#arch-03). Owner: [TASK-002](plan.md#task-002). Verification: [V-OPTIONS](architecture.md#v-options). Evidence: [docs/verification/TASK-002.md](verification/TASK-002.md) (tilde-only expansion, `~user` rejected, relative resolution, normalization, duplicate/symlink rejection, missing homes never created).

<a id="req-009"></a>
- [x] **REQ-009: Configurable finite deadline.** Default to a 20-second query budget per admitted target; support `--timeout` as a Go duration from 1 second through 120 seconds. Include startup and every query RPC in that single budget. Bound cleanup separately to at most 2 seconds.
  - - Architecture: [ARCH-05](architecture.md#arch-05). Owner: [TASK-009](plan.md#task-009). Verification: [V-LIFECYCLE](architecture.md#v-lifecycle). Evidence: [docs/verification/TASK-002.md](verification/TASK-002.md) (1s–120s validation, 20s default), TASK-009 (admission-time budget covering startup and every RPC), TASK-003 (separate 2s cleanup bound).
  - Superseded 2026-09-22 by the corrective audit: a valid result established within the query budget is discarded when deferred cleanup crosses the deadline, and cleanup consumes up to twice the documented 2s bound (separate shutdown/finish budgets). Re-qualification: [TASK-016](plan.md#task-016).
  - Re-qualified 2026-09-22 under EPIC-007: results established before the deadline survive cleanup crossing it (runner, provider, and rpc establishment-timestamp regressions), and one absolute cleanup budget is enforced end to end.

<a id="req-010"></a>
- [ ] **REQ-010: Account identification.** Prefer a Codex email returned by the same App Server instance that supplies the quota; otherwise use the configured home basename. Use the explicit Grok alias, default `default`, rather than guessing an email. Disambiguate duplicate visible Codex account names without merging rows.
  - - Architecture: [ARCH-04](architecture.md#arch-04). Owner: [TASK-010](plan.md#task-010). Verification: [V-IDENTITY](architecture.md#v-identity). Evidence: [docs/verification/TASK-010.md](verification/TASK-010.md) and TASK-006/008 identity suites — protocol email preferred, home-basename fallback, Grok alias, duplicate disambiguation.
  - Superseded 2026-09-22 by the corrective audit: whitespace-only protocol emails are adopted as labels instead of falling back to the home basename, and duplicate disambiguation runs on raw labels before sanitization. Re-qualification: [TASK-019](plan.md#task-019).
  - Re-qualified 2026-09-22 under EPIC-007: emails and aliases pass through the single display-label policy before validity decisions, and disambiguation operates on normalized final labels with suffix-lookalike collision handling.

  - Reopened 2026-09-22 by the confirmation review of `c340245`: Unicode-whitespace labels bypassed validity checks and duplicate suffixes used a collision counter instead of the documented target-index rule. Re-qualification: [TASK-023](plan.md#task-023).
<a id="req-011"></a>
- [x] **REQ-011: Fixed logical table.** Use exactly the three headings `AI PROVIDER`, `ACCOUNT`, and `WEEKLY USAGE`. Emit no title, separator, progress row, reset-time column, totals, or blank lines. With N Codex homes, emit N+2 newline-terminated logical lines, including the header and one Grok row.
  - - Architecture: [ARCH-10](architecture.md#arch-10). Owner: [TASK-010](plan.md#task-010). Verification: [V-OUTPUT](architecture.md#v-output). Evidence: [docs/verification/TASK-010.md](verification/TASK-010.md) golden tests — three headings, no extras, 4 lines default, N+2 rule, final newline.

<a id="req-012"></a>
- [ ] **REQ-012: Safe, plain output.** Write the table only to stdout and concise diagnostics only to stderr. Do not forward raw provider output. Emit no color or terminal control sequences. Sanitize account text so embedded tabs, newlines, escape sequences, and control/format characters cannot add columns or rows.
  - - Architecture: [ARCH-10](architecture.md#arch-10). Owner: [TASK-010](plan.md#task-010). Verification: [V-OUTPUT](architecture.md#v-output). Evidence: [docs/verification/TASK-010.md](verification/TASK-010.md) — stdout table only, stderr diagnostics only, no raw provider output, control-character sanitization.

### Usage semantics

  - Reopened 2026-09-22 by the confirmation review of `c340245`: the shared normalization policy enumerated only some Unicode whitespace, so whitespace-only labels could render as blank cells. Re-qualification: [TASK-023](plan.md#task-023).
<a id="req-013"></a>
- [x] **REQ-013: Presence-aware percentages.** Emit one decimal place and `%` for a finite, explicitly present numeric value from 0 through 100. Preserve a genuine zero as `0.0%`. Reject negative, over-100, non-numeric, or non-finite values; do not clamp them, multiply ratios by 100, or substitute zero for missing fields.
  - - Architecture: [ARCH-04](architecture.md#arch-04). Owner: [TASK-008](plan.md#task-008). Verification: [V-QUOTA](architecture.md#v-quota). Evidence: [docs/verification/TASK-008.md](verification/TASK-008.md) and TASK-006 suites — one-decimal formatting, explicit zero preserved, out-of-range/non-numeric rejected without clamping or ratio conversion.

<a id="req-014"></a>
- [x] **REQ-014: No invented weekly value.** Show a defined unavailable/non-applicable status when the response cannot establish a weekly value. Do not average multiple quotas, choose the highest percentage, extrapolate monthly values, or guess a window from reset time.
  - - Architecture: [ARCH-04](architecture.md#arch-04). Owner: [TASK-008](plan.md#task-008). Verification: [V-QUOTA](architecture.md#v-quota). Evidence: [docs/verification/TASK-008.md](verification/TASK-008.md) — defined unavailable/non-applicable statuses for every unestablishable weekly value; no averaging/extrapolation anywhere.

<a id="req-015"></a>
- [x] **REQ-015: Window provenance.** Keep the selected quota identifier, period kind, source, and optional reset time in the in-memory result. Reset metadata is not an additional output column and must not be required when the source makes it optional.
  - - Architecture: [ARCH-04](architecture.md#arch-04). Owner: [TASK-008](plan.md#task-008). Verification: [V-QUOTA](architecture.md#v-quota). Evidence: [docs/verification/TASK-006.md](verification/TASK-006.md) and TASK-008 — quota id, period kind, source, and optional reset preserved in domain results and never rendered as columns.
  - Superseded 2026-09-22 by the corrective audit: the Grok monthly billing branch drops the established period kind from the in-memory result. Re-qualification: [TASK-019](plan.md#task-019).
  - Re-qualified 2026-09-22 under EPIC-007: monthly and weekly branches retain Period (and optional reset/source) provenance, including weekly windows with missing usage; no quota identifier is invented.

### Protocol and execution

<a id="req-016"></a>
- [x] **REQ-016: Separate wire dialects.** Use stdio JSON messages with the correct provider-specific envelopes and initialization rules. Codex App Server and Grok ACP must not be treated as one interchangeable protocol. Freeze Grok extension wire encoding from an actual installed-binary probe.
  - Architecture: [ARCH-06](architecture.md#arch-06). Owner: [TASK-004](plan.md#task-004). Verification: [V-RPC](architecture.md#v-rpc). Evidence: [docs/verification/TASK-004.md](verification/TASK-004.md) plus frozen live encoding `_x.ai/billing` from [compatibility.md](verification/compatibility.md).
  - Superseded 2026-09-22 by the corrective audit: inbound responses are not validated per dialect — `error:null` without `result` satisfies a pending call, handshake results are accepted without shape validation, and inbound jsonrpc version markers are unchecked. Re-qualification: [TASK-018](plan.md#task-018).
  - Re-qualified 2026-09-22 under EPIC-007: responses require exactly one structurally valid alternative, the ACP envelope marker is validated per dialect, and both handshakes enforce their qualified result shapes while additive members stay tolerated.

<a id="req-017"></a>
- [x] **REQ-017: Bounded framing.** Support fragmented and coalesced newline-delimited UTF-8 JSON, with a 4 MiB maximum frame. Reject malformed/oversized frames and unsupported batches; bound buffered stderr to 64 KiB while continuing to drain excess bytes.
  - Architecture: [ARCH-06](architecture.md#arch-06). Owner: [TASK-004](plan.md#task-004). Verification: [V-RPC](architecture.md#v-rpc). Evidence: [docs/verification/TASK-004.md](verification/TASK-004.md) (framing matrix incl. boundary sizes; 64 KiB stderr drain in TASK-003).

<a id="req-018"></a>
- [x] **REQ-018: Response correlation.** Correlate responses by request ID, tolerate unrelated well-formed notifications, and never use the first received JSON object as the response. Unknown/duplicate response IDs cannot satisfy a pending request. Fail a pending call on EOF or malformed matching responses.
  - Architecture: [ARCH-06](architecture.md#arch-06). Owner: [TASK-004](plan.md#task-004). Verification: [V-RPC](architecture.md#v-rpc). Evidence: [docs/verification/TASK-004.md](verification/TASK-004.md).
  - Superseded 2026-09-22 by the corrective audit: a matching response with `error:null` and no `result` is accepted as successful instead of failing the pending call as malformed. Re-qualification: [TASK-018](plan.md#task-018).
  - Re-qualified 2026-09-22 under EPIC-007: error:null, incomplete error objects, and shapeless matching responses fail the pending call as malformed instead of satisfying it.

<a id="req-019"></a>
- [x] **REQ-019: No host capability execution.** Advertise no filesystem, terminal, or tool execution capability to Grok. Never execute server-initiated requests, approve tools, answer elicitation with secrets, or service token-refresh callbacks requiring application-held credentials. Reject unexpected requests and close the affected query safely.
  - Architecture: [ARCH-09](architecture.md#arch-09). Owner: [TASK-004](plan.md#task-004). Verification: [V-SAFETY](architecture.md#v-safety). Evidence: [docs/verification/TASK-004.md](verification/TASK-004.md) (server requests rejected with exact id and connection aborted; Grok false capabilities declared in TASK-007 wire tests).

<a id="req-020"></a>
- [ ] **REQ-020: Owned-process cleanup.** Close stdin and reap each owned child exactly once. Apply bounded EOF/termination/kill cleanup on success, error, timeout, and cancellation. Do not kill unrelated CLI sessions or provider-managed shared leaders; private process-group cleanup is allowed only with proven ownership.
  - - Architecture: [ARCH-05](architecture.md#arch-05). Owner: [TASK-003](plan.md#task-003). Verification: [V-LIFECYCLE](architecture.md#v-lifecycle). Evidence: [docs/verification/TASK-003.md](verification/TASK-003.md) — stdin close, exactly-once wait, EOF/SIGTERM/SIGKILL stages within budget, unrelated-process survival, owned-group-only kills.
  - Superseded 2026-09-22 by the corrective audit: cleanup grants `finish` a fresh full budget after `shutdown` (violating the single bounded allowance), and the direct child exiting early can leave application-owned descendants unsignalled. Re-qualification: [TASK-016](plan.md#task-016).
  - Re-qualified 2026-09-22 under EPIC-007: one absolute cleanup deadline covers shutdown and finish, incomplete cleanup surfaces ErrCleanupIncomplete, exactly-once reaping and the descendant/shared-leader boundary are tested on both sides.

  - Reopened 2026-09-22 by the confirmation review of `c340245`: forced cleanup still SIGKILLed the whole private group when the direct child was unresponsive, without proven ownership. Re-qualification: [TASK-022](plan.md#task-022).
<a id="req-021"></a>
- [x] **REQ-021: Safe diagnostics.** Classify errors using local failure types and verified structured protocol errors. Do not print auth files, bearer tokens, environment values, raw RPC bodies, or raw child stderr. A generic sanitized error is preferable to unsafe or guessed detail.
  - - Architecture: [ARCH-09](architecture.md#arch-09). Owner: [TASK-011](plan.md#task-011). Verification: [V-SAFETY](architecture.md#v-safety). Evidence: [docs/verification/TASK-011.md](verification/TASK-011.md) — fixed application-owned messages only; secret markers and raw provider output never reach stdout/stderr.
  - Superseded 2026-09-22 by the corrective audit: every remote error from Grok `authenticate`, including internal errors, is reported as "cached token rejected" — a guessed detail, not a verified structured authentication failure. Re-qualification: [TASK-019](plan.md#task-019).
  - Re-qualified 2026-09-22 under EPIC-007: authenticate remote errors map only from verified structure (method-not-found to UNSUPPORTED, others to a generic provider error); AUTH REQUIRED is established by the advertisement alone.

<a id="req-022"></a>
- [x] **REQ-022: Independent bounded queries.** Allow at most three target queries concurrently, using isolated environment slices and result slots. A failure in one target must not cancel healthy siblings. Preserve configured row order regardless of completion order.
  - - Architecture: [ARCH-05](architecture.md#arch-05). Owner: [TASK-009](plan.md#task-009). Verification: [V-RUNNER](architecture.md#v-runner). Evidence: [docs/verification/TASK-009.md](verification/TASK-009.md) — ≤3 active, isolated envs/slots, sibling preservation under failure/hang, configured order.

<a id="req-023"></a>
- [x] **REQ-023: Signal handling.** On SIGINT or SIGTERM, cancel pending work, clean up owned processes, preserve already obtained results, and render unresolved rows as `CANCELLED` when stdout is writable. Do not start targets still queued after cancellation.
  - - Architecture: [ARCH-05](architecture.md#arch-05). Owner: [TASK-009](plan.md#task-009). Verification: [V-LIFECYCLE](architecture.md#v-lifecycle). Evidence: [docs/verification/TASK-009.md](verification/TASK-009.md) (queued cancellation, no late launch), TASK-010 (130/143 + CANCELLED table), TASK-011 (SIGINT end-to-end).
  - Superseded 2026-09-22 by the corrective audit: signal cancellation cannot unblock a blocked notification write, so a wedged provider can defeat cancellation and owned-process cleanup. Re-qualification: [TASK-016](plan.md#task-016).
  - Re-qualified 2026-09-22 under EPIC-007: cancellation now unblocks blocked notification and rejection writes, established results survive root cancellation, and the executable-level signal suite passed with real pipes.

<a id="req-024"></a>
- [x] **REQ-024: Exit contract.** Implement exit codes 0, 1, 2, 130, and 143 exactly as ARCH-10 defines. Keep partial results visible even when exit code 1 is returned. Explicit monthly/API-key non-applicability is not an operational failure; missing data and unsupported methods are incomplete results.
  - - Architecture: [ARCH-10](architecture.md#arch-10). Owner: [TASK-010](plan.md#task-010). Verification: [V-EXIT](architecture.md#v-exit). Evidence: [docs/verification/TASK-010.md](verification/TASK-010.md) — codes 0/1/2/130 exercised end-to-end; 143 shares the SIGTERM path; non-applicability is clean.
  - Superseded 2026-09-22 by the corrective audit: a closed stdout pipe terminates the executable via SIGPIPE instead of exit 1, and an output failure after a handled signal returns 1 instead of 130/143. Re-qualification: [TASK-017](plan.md#task-017).
  - Re-qualified 2026-09-22 under EPIC-007: a closed stdout pipe exits 1 (never SIGPIPE death), and 130/143 outrank output failure — verified against the actual executable with real OS pipes.

### Codex provider

<a id="req-025"></a>
- [x] **REQ-025: Per-target App Server.** Start a separate `codex app-server` process for each Codex home, with that target’s absolute `CODEX_HOME`. Do not switch accounts inside a shared App Server or mutate the parent environment.
  - Architecture: [ARCH-07](architecture.md#arch-07). Owner: [TASK-005](plan.md#task-005). Verification: [V-CODEX](architecture.md#v-codex). Evidence: [docs/verification/TASK-005.md](verification/TASK-005.md) (per-target child with filtered env replacing CODEX_HOME; parent env immutable; two simultaneous isolated sessions).

<a id="req-026"></a>
- [x] **REQ-026: Metadata-only handshake.** Complete `initialize`, wait for its response, then send `initialized` before the account calls. Use only the allowlisted account-read and rate-limit-read operations; do not start/resume threads or turns.
  - Architecture: [ARCH-07](architecture.md#arch-07). Owner: [TASK-005](plan.md#task-005). Verification: [V-CODEX](architecture.md#v-codex). Evidence: [docs/verification/TASK-005.md](verification/TASK-005.md) (transcript-asserted ordering and outbound surface).

<a id="req-027"></a>
- [x] **REQ-027: Existing subscription auth.** Use provider-managed ChatGPT login state without reading credentials directly. An absent/expired login becomes `AUTH REQUIRED`; API-key-only auth becomes `N/A (API key)`. Other unqualified auth variants must not be silently treated as ChatGPT subscriptions.
  - Architecture: [ARCH-07](architecture.md#arch-07). Owner: [TASK-005](plan.md#task-005). Verification: [V-CODEX](architecture.md#v-codex). Evidence: [docs/verification/TASK-005.md](verification/TASK-005.md) (chatgpt/apiKey/absent/unknown mapping without credential reads).

<a id="req-028"></a>
- [x] **REQ-028: Deterministic quota bucket selection.** Prefer `rateLimitsByLimitId["codex"]` when the multi-bucket map is non-empty. Do not fall through to an unrelated bucket or average map entries. Use legacy `rateLimits` only under the explicit compatibility rules in ARCH-07.
  - Architecture: [ARCH-07](architecture.md#arch-07). Owner: [TASK-006](plan.md#task-006). Verification: [V-CODEX](architecture.md#v-codex). Evidence: [docs/verification/TASK-006.md](verification/TASK-006.md) — bucket-selection table (non-empty map without codex → no_weekly, never legacy or averaging; codex null → no_data; wrong type → protocol error; empty/null map → legacy with explicit codex limitId only).

<a id="req-029"></a>
- [x] **REQ-029: Duration-based weekly window.** Inspect both `primary` and `secondary` in the selected Codex bucket. Select by `windowDurationMins == 10080`, never by slot name. Handle absent, null, reversed, duplicated, and conflicting windows as defined in ARCH-07.
  - Architecture: [ARCH-07](architecture.md#arch-07). Owner: [TASK-006](plan.md#task-006). Verification: [V-CODEX](architecture.md#v-codex). Evidence: [docs/verification/TASK-006.md](verification/TASK-006.md) — slot matrix incl. reversed slots, five-hour-only rejection, duplicate-agreeing (37 vs 37.0) and conflicting weekly windows.

<a id="req-030"></a>
- [x] **REQ-030: Identity isolation.** Obtain Codex account information and quota through the same isolated child. Preserve the correct fallback identity on authentication or transport failure, and test simultaneous results from two different homes.
  - Architecture: [ARCH-07](architecture.md#arch-07). Owner: [TASK-006](plan.md#task-006). Verification: [V-IDENTITY](architecture.md#v-identity). Evidence: [docs/verification/TASK-006.md](verification/TASK-006.md) — same-child identity+quota; simultaneous two-home lifecycle tests with distinct emails and percentages; home-alias fallback preserved on every failure path.

### Grok provider

<a id="req-031"></a>
- [x] **REQ-031: ACP-only query path.** Use the qualified headless stdio invocation, initially `grok --no-auto-update agent stdio`. Do not use the interactive TUI, PTY, `-p`, an undocumented REST fallback, or an application-implemented billing HTTP client.
  - - Architecture: [ARCH-08](architecture.md#arch-08). Owner: [TASK-007](plan.md#task-007). Verification: [V-GROK](architecture.md#v-grok). Evidence: [docs/verification/TASK-007.md](verification/TASK-007.md) — exact `--no-auto-update agent stdio` invocation asserted from the child argv; no TUI/PTY/REST anywhere (TASK-011 static scan).

<a id="req-032"></a>
- [x] **REQ-032: Cached login without new auth.** Select `cached_token` only when advertised and authenticate with headless metadata. Do not choose an API key because it exists in the environment, launch a browser, run `grok login`, or initiate device-code/OIDC login.
  - - Architecture: [ARCH-08](architecture.md#arch-08). Owner: [TASK-007](plan.md#task-007). Verification: [V-GROK](architecture.md#v-grok). Evidence: [docs/verification/TASK-007.md](verification/TASK-007.md) — cached_token selected only when advertised, headless _meta, API-key env never selects auth, no browser/login/device flows.
  - Superseded 2026-09-22 by the corrective audit: an advertisement offering both `api_key` and `grok.com` without `cached_token` is classified `N/A (API key)` instead of requiring authentication for the interactive subscription path. Re-qualification: [TASK-019](plan.md#task-019).
  - Re-qualified 2026-09-22 under EPIC-007: the advertisement decision table requires authentication for {api_key, grok.com}, keeps cached_token precedence, never declares unknown combinations clean, and never initiates interactive authentication.

<a id="req-033"></a>
- [x] **REQ-033: Qualified billing extension.** Call the source-backed billing extension using the captured ACP wire contract. Distinguish the internal `x.ai/billing` name from its wire representation. A valid method-not-found response produces `UNSUPPORTED`, not fabricated usage or an automatic alternate transport.
  - - Architecture: [ARCH-08](architecture.md#arch-08). Owner: [TASK-007](plan.md#task-007). Verification: [V-GROK](architecture.md#v-grok). Evidence: [docs/verification/TASK-007.md](verification/TASK-007.md) — frozen `_x.ai/billing` wire method from the live probe; method-not-found maps to UNSUPPORTED (TASK-008), no alternate transports.

<a id="req-034"></a>
- [x] **REQ-034: Explicit weekly billing semantics.** Use `config.creditUsagePercent` only when `config.currentPeriod.type` explicitly identifies the qualified weekly enum. An explicit monthly enum produces `N/A (monthly)`; absent/unknown period metadata does not establish a week.
  - - Architecture: [ARCH-08](architecture.md#arch-08). Owner: [TASK-008](plan.md#task-008). Verification: [V-GROK](architecture.md#v-grok). Evidence: [docs/verification/TASK-008.md](verification/TASK-008.md) — creditUsagePercent read only under the explicit weekly enum; monthly → N/A (monthly); absent/unknown → N/A (period).

<a id="req-035"></a>
- [x] **REQ-035: No legacy-credit estimation.** Do not compute weekly usage from `monthlyLimit`, `used`, purchased/prepaid balance, on-demand usage, or billing history. Treat legacy-only data as unavailable rather than converting it.
  - - Architecture: [ARCH-08](architecture.md#arch-08). Owner: [TASK-008](plan.md#task-008). Verification: [V-GROK](architecture.md#v-grok). Evidence: [docs/verification/TASK-008.md](verification/TASK-008.md) — legacy monthlyLimit/used/prepaid/on-demand payloads map to N/A (period), never converted.

<a id="req-036"></a>
- [x] **REQ-036: No speculative identity endpoint.** Keep the Grok alias available on success and failure. Do not read Grok auth JSON, decode JWTs, or invent a `whoami`/profile API to fill the ACCOUNT column.
  - - Architecture: [ARCH-08](architecture.md#arch-08). Owner: [TASK-008](plan.md#task-008). Verification: [V-IDENTITY](architecture.md#v-identity). Evidence: [docs/verification/TASK-008.md](verification/TASK-008.md) — configured alias on success/failure; auth payloads are never identity sources; no whoami/JWT/auth-file code (TASK-011 static scan).

### Security and boundaries

<a id="req-037"></a>
- [x] **REQ-037: Credential ownership.** Leave credential discovery, keychain access, refresh, and storage to the installed CLIs. The application must not parse/copy auth files, request tokens from the user, store secrets, or implement OAuth.
  - - Architecture: [ARCH-09](architecture.md#arch-09). Owner: [TASK-011](plan.md#task-011). Verification: [V-SAFETY](architecture.md#v-safety). Evidence: [docs/verification/TASK-011.md](verification/TASK-011.md) static scan (no auth.json/keychain/token parsing) and provider suites using provider-managed login state only.

<a id="req-038"></a>
- [x] **REQ-038: Business-level read-only boundary.** Issue no prompts, agent sessions, tool actions, login/logout, billing mutations, quota-reset redemption, purchases, or configuration writes. Distinguish this from provider-owned token refresh, logs, caches, and shared-process lifecycle that can occur during CLI startup.
  - - Architecture: [ARCH-09](architecture.md#arch-09). Owner: [TASK-011](plan.md#task-011). Verification: [V-SAFETY](architecture.md#v-safety). Evidence: [docs/verification/TASK-011.md](verification/TASK-011.md) — allowlisted metadata methods only; no prompts/sessions/mutations; static capability scan.

<a id="req-039"></a>
- [x] **REQ-039: Isolated child environments.** Use explicit executable paths/argument arrays without a shell. Copy and filter each child environment; replace Codex home per target and exclude provider API-key overrides and agent-hosted parent-session markers identified during qualification. Preserve required home/keychain/proxy context.
  - Architecture: [ARCH-09](architecture.md#arch-09). Owner: [TASK-005](plan.md#task-005). Verification: [V-SAFETY](architecture.md#v-safety). Evidence: [docs/verification/TASK-005.md](verification/TASK-005.md) (envfilter policy: drops + per-target CODEX_HOME replacement, HOME/proxy preserved, parent untouched; argv-only child start from TASK-003).
  - Superseded 2026-09-22 by the corrective audit: explicit relative executable paths are not resolved against the invocation environment before the child working directory changes, so they fail from the neutral child cwd. Re-qualification: [TASK-017](plan.md#task-017).
  - Re-qualified 2026-09-22 under EPIC-007: explicit relative executable identity is absolutized before the neutral child cwd while ErrDot protection is preserved.

<a id="req-040"></a>
- [ ] **REQ-040: Neutral working directory.** Launch probes and runtime children in private empty temporary directories outside the caller’s project tree. Do not pass repository prompts, MCP servers, skills, or project files. Delete application-owned temporary resources after child cleanup.
  - - Architecture: [ARCH-05](architecture.md#arch-05). Owner: [TASK-003](plan.md#task-003). Verification: [V-SAFETY](architecture.md#v-safety). Evidence: [docs/verification/TASK-003.md](verification/TASK-003.md) — private 0700 temp cwd outside the project, removed after cleanup; TASK-011 confirms invocation from neutral dirs.

  - Reopened 2026-09-22 by the confirmation review of `c340245`: cleanup failures (including temporary-directory removal failures) were discarded before reaching the CLI, so an invocation with unremovable application-owned directories still exited 0. Re-qualification: [TASK-022](plan.md#task-022).
<a id="req-041"></a>
- [x] **REQ-041: No application service or data store.** Create no listener, remote service, database, permanent runtime config, permanent cache, credentials store, or application telemetry. A provider-owned leader is not a Gaebokchi daemon and must not be controlled as one.
  - - Architecture: [ARCH-02](architecture.md#arch-02). Owner: [TASK-011](plan.md#task-011). Verification: [V-SAFETY](architecture.md#v-safety). Evidence: [docs/verification/TASK-011.md](verification/TASK-011.md) static scan (no listeners, no persistent files) and TASK-003 temp-dir cleanup.

### Verification and delivery

<a id="req-042"></a>
- [x] **REQ-042: Version-qualified external contracts.** Record actual CLI executable/version evidence, protocol forms, source references, and sanitized observations. Never infer an installed version from a repository SHA or claim an untested version is supported. Detect contract mismatch safely without auto-upgrading either CLI.
  - Architecture: [ARCH-12](architecture.md#arch-12). Owner: [TASK-001](plan.md#task-001). Verification: [V-CONTRACT](architecture.md#v-contract). Evidence: [docs/verification/compatibility.md](verification/compatibility.md) (codex-cli 0.155.1, grok 1.0.40, live two-home Codex reads, live weekly Grok billing, sanitized fixtures).

<a id="req-043"></a>
- [x] **REQ-043: Offline default tests.** Run normal tests entirely against fixtures and fake child executables, without live accounts, provider installations, credentials, or network access. Keep live probes explicit and opt-in.
  - Architecture: [ARCH-11](architecture.md#arch-11). Owner: [TASK-004](plan.md#task-004). Verification: [V-RPC](architecture.md#v-rpc). Evidence: [docs/verification/TASK-004.md](verification/TASK-004.md) — the entire suite runs against in-memory pipes and the compiled fake-child harness; no live CLI is invoked by `go test ./...`.

<a id="req-044"></a>
- [x] **REQ-044: Parser and edge-case coverage.** Cover the quota, authentication, dialect, account-isolation, malformed-input, and semantic edge cases listed in ARCH-11. Unknown additive fields must not break an otherwise valid response.
  - - Architecture: [ARCH-11](architecture.md#arch-11). Owner: [TASK-011](plan.md#task-011). Verification: [V-QUOTA](architecture.md#v-quota). Evidence: [docs/verification/TASK-006.md](verification/TASK-006.md) (Codex matrix), TASK-008 (Grok matrix), TASK-004 (framing/dialects) — unknown additive fields ignored throughout.
  - Superseded 2026-09-22 by the corrective audit: malformed-input edge cases (error:null, mistyped error members, incompatible version markers, label-normalization collisions) were not covered and were in fact accepted by the implementation. Re-qualification: [TASK-018](plan.md#task-018) and [TASK-019](plan.md#task-019); final re-check at [TASK-021](plan.md#task-021).
  - Re-qualified 2026-09-22 under EPIC-007: the malformed-input and label edge cases named by the audit are covered and rejected by the rpc/provider validation suites; the final offline gate (vet, tests, race, docs, gofmt, builds) ran green on darwin/arm64 and linux/arm64 against the corrected tree.

<a id="req-045"></a>
- [x] **REQ-045: End-to-end failure isolation.** Verify stdout, stderr, exit status, row count/order, callback rejection, timeout, signal handling, and cleanup through the actual CLI executable with scripted fake providers.
  - - Architecture: [ARCH-11](architecture.md#arch-11). Owner: [TASK-011](plan.md#task-011). Verification: [V-E2E](architecture.md#v-e2e). Evidence: [docs/verification/TASK-011.md](verification/TASK-011.md) — executable-level suite with fake providers covering stdout/stderr/exit/rows/callbacks/timeouts/signals/cleanup.
  - Superseded 2026-09-22 by the corrective audit: the executable-level coverage used in-process writer stubs for output failure; real broken-pipe and signal-plus-output-failure behavior was untested and violates the contract. Re-qualification: [TASK-017](plan.md#task-017).
  - Re-qualified 2026-09-22 under EPIC-007: the executable-level suite now covers broken pipes, each signal with unwritable stdout, and mixed completion under cancellation using the actual binary and real OS pipes.

<a id="req-046"></a>
- [x] **REQ-046: Build artifacts.** Produce binaries for darwin/arm64, darwin/amd64, linux/arm64, and linux/amd64, with version metadata and checksums. Do not bundle third-party provider binaries or automatically publish a release.
  - - Architecture: [ARCH-13](architecture.md#arch-13). Owner: [TASK-012](plan.md#task-012). Verification: [V-BUILD](architecture.md#v-build). Evidence: [docs/verification/TASK-012.md](verification/TASK-012.md) — four target archives with SHA-256 checksums, spec docs included, no provider binaries, no publication.

<a id="req-047"></a>
- [x] **REQ-047: Native platform evidence.** Run the application and fake-provider integration tests natively on macOS and Linux. Record which CPU architectures were executed and which were only cross-built; a successful cross-build alone is not runtime validation.
  - - Architecture: [ARCH-13](architecture.md#arch-13). Owner: [TASK-012](plan.md#task-012). Verification: [V-PLATFORM](architecture.md#v-platform). Evidence: [docs/verification/TASK-012.md](verification/TASK-012.md) — native darwin/arm64 and linux/arm64 (containerized) full suites; darwin/amd64 and linux/amd64 labelled cross-build-only. Re-run natively on the corrected tree under [TASK-021](plan.md#task-021) (EPIC-007): darwin/arm64 full suite + race, linux/arm64 (golang:1.26.6-bookworm container) vet + full suite + race subset + docs check, all green; cross-build-only labels unchanged.

<a id="req-048"></a>
- [ ] **REQ-048: Owner-account live acceptance.** Perform an explicit live campaign against the owner’s two distinct Codex homes and existing Grok account on at least one supported host. Compare with the respective provider’s own usage display within the same quota period, without generating a model turn.
  - - Architecture: [ARCH-12](architecture.md#arch-12). Owner: [TASK-013](plan.md#task-013). Verification: [V-LIVE](architecture.md#v-live). Evidence: [docs/verification/live-acceptance.md](verification/live-acceptance.md) — both Codex homes plus the existing Grok login on darwin/arm64, live weekly values on all rows, window-boundary comparison with the headless-display limitation recorded.
  - Superseded 2026-09-22 by the corrective audit: the provider's own usage display was never compared; the recorded campaign substituted a second metadata probe while the evidence itself recorded the omission as a limitation. The live metadata-query evidence for the tested artifact is preserved; only the provider-display comparison portion is reopened. Re-qualification: owner-authorized comparison recorded at [TASK-021](plan.md#task-021); blocked while authorization or display access is unavailable — not waivable on the owner's behalf.

<a id="req-049"></a>
- [x] **REQ-049: Usage and troubleshooting guide.** Ship a README covering install/build, exact flags, existing-login prerequisite, quota meaning, statuses, exit codes, supported/tested versions, platform evidence, and known limitations. Explain that a Grok alias is not a verified email and a monthly result is not weekly usage.
  - - Architecture: [ARCH-13](architecture.md#arch-13). Owner: [TASK-012](plan.md#task-012). Verification: [V-DOCS](architecture.md#v-docs). Evidence: README.md — install/build, exact flags, login prerequisite, quota meaning, all statuses, exit codes, tested versions/platforms, limitations incl. alias-not-email and monthly-not-weekly; smoke-checked against the binary.
  - Superseded 2026-09-22 by the corrective audit: README wording conflates Linux fake-provider offline testing with live provider qualification. Re-qualification: [TASK-020](plan.md#task-020).
  - Re-qualified 2026-09-22 under EPIC-007: README distinguishes live provider qualification (darwin/arm64) from the native offline Linux run and from cross-build-only targets.

<a id="req-050"></a>
- [ ] **REQ-050: Traceable completion.** Maintain requirement-to-architecture-to-task links and record verification evidence before checking off a requirement. Complete epics in plan order, one task-scoped commit at a time. Do not close a provider requirement on fallback-only evidence or run unbounded review cycles.
  - - Architecture: [ARCH-14](architecture.md#arch-14). Owner: [TASK-014](plan.md#task-014). Verification: [V-AUDIT](architecture.md#v-audit). Evidence: this audit — every REQ carries architecture/owner/verification/evidence links, fourteen task-scoped commits in plan order (git log), live evidence distinguished from fixtures throughout.
  - Superseded 2026-09-22 by the corrective audit: 17 requirements were checked without their full acceptance scope being verified, and plan/requirement statuses contradicted each other. Re-qualification: [TASK-021](plan.md#task-021).
  - Re-qualified 2026-09-22 under EPIC-007: TASK-015 through TASK-021 each landed one coherent commit with evidence; the audit trail records the supersession and re-qualification of every affected requirement, and REQ-048 remains visibly open rather than silently closed.

  - Reopened 2026-09-22 by the confirmation review of `c340245`: roadmap/acceptance-checkbox/packaged-document states were inconsistent with the recorded completion claims. Re-qualification: [TASK-024](plan.md#task-024).
## Completion rules

A percentage is not required when the provider explicitly reports a monthly allowance or the account uses an out-of-scope API-key mode. Those are supported, truthful **non-applicable** outcomes. An unsupported method, missing weekly metadata, absent login, timeout, or missing executable is a supported **failure/unavailable presentation**, not evidence that the positive provider integration is complete.

REQ-005 and REQ-048 require live successful queries. When the owner's Grok account is monthly, a live valid billing response and `N/A (monthly)` may qualify that account's behavior, while weekly parsing is additionally verified with source-backed fixtures. The evidence must explicitly state that a live weekly Grok percentage was not exercised; it must not claim otherwise. A method-not-found response cannot satisfy that gate.

Checklists are not a replacement for evidence. Use the [task evidence format](plan.md#plan-evidence), preserve failures and unrun checks, and finish with [TASK-014](plan.md#task-014).

## Requirement ownership index

This index is a navigation aid. The individual checkbox remains authoritative.

| Task | Requirements primarily closed by the task |
|---|---|
| [TASK-001](plan.md#task-001) | [REQ-042](#req-042) |
| [TASK-002](plan.md#task-002) | [REQ-007](#req-007), [REQ-008](#req-008) |
| [TASK-003](plan.md#task-003) | [REQ-020](#req-020), [REQ-040](#req-040) |
| [TASK-004](plan.md#task-004) | [REQ-016](#req-016), [REQ-017](#req-017), [REQ-018](#req-018), [REQ-019](#req-019), [REQ-043](#req-043) |
| [TASK-005](plan.md#task-005) | [REQ-025](#req-025), [REQ-026](#req-026), [REQ-027](#req-027), [REQ-039](#req-039) |
| [TASK-006](plan.md#task-006) | [REQ-003](#req-003), [REQ-028](#req-028), [REQ-029](#req-029), [REQ-030](#req-030) |
| [TASK-007](plan.md#task-007) | [REQ-031](#req-031), [REQ-032](#req-032), [REQ-033](#req-033) |
| [TASK-008](plan.md#task-008) | [REQ-013](#req-013), [REQ-014](#req-014), [REQ-015](#req-015), [REQ-034](#req-034), [REQ-035](#req-035), [REQ-036](#req-036) |
| [TASK-009](plan.md#task-009) | [REQ-004](#req-004), [REQ-009](#req-009), [REQ-022](#req-022), [REQ-023](#req-023) |
| [TASK-010](plan.md#task-010) | [REQ-002](#req-002), [REQ-006](#req-006), [REQ-010](#req-010), [REQ-011](#req-011), [REQ-012](#req-012), [REQ-024](#req-024) |
| [TASK-011](plan.md#task-011) | [REQ-021](#req-021), [REQ-037](#req-037), [REQ-038](#req-038), [REQ-041](#req-041), [REQ-044](#req-044), [REQ-045](#req-045) |
| [TASK-012](plan.md#task-012) | [REQ-001](#req-001), [REQ-046](#req-046), [REQ-047](#req-047), [REQ-049](#req-049) |
| [TASK-013](plan.md#task-013) | [REQ-005](#req-005), [REQ-048](#req-048) |
| [TASK-014](plan.md#task-014) | [REQ-050](#req-050) |
| [TASK-015](plan.md#task-015) | Acceptance-baseline correction only; reopens the checkboxes listed below, closes none |
| [TASK-016](plan.md#task-016) | Re-qualifies [REQ-009](#req-009), [REQ-020](#req-020), [REQ-023](#req-023) |
| [TASK-017](plan.md#task-017) | Re-qualifies [REQ-006](#req-006) (binary overrides), [REQ-024](#req-024), [REQ-039](#req-039), [REQ-045](#req-045) |
| [TASK-018](plan.md#task-018) | Re-qualifies [REQ-016](#req-016), [REQ-018](#req-018) |
| [TASK-019](plan.md#task-019) | Re-qualifies [REQ-006](#req-006) (alias validation), [REQ-010](#req-010), [REQ-015](#req-015), [REQ-021](#req-021), [REQ-032](#req-032) |
| [TASK-020](plan.md#task-020) | Re-qualifies [REQ-049](#req-049) |
| [TASK-021](plan.md#task-021) | Re-qualifies [REQ-044](#req-044), [REQ-048](#req-048), [REQ-050](#req-050) |

Component tasks also contribute to cross-cutting requirements owned by integration or release tasks. See the dependency and verification mappings in [plan.md](plan.md#plan-roadmap).
