# Gaebokchi (개복치): Requirements

Project: **Gaebokchi (개복치)** · Ecosystem: Aquarium tool · Repository and executable: `gaebokchi`

## Document contract

This document owns **what the product must do**: its required behavior and its durable user-visible contracts. The [architecture](../architecture/README.md) owns implementation structure, data semantics, and protocol boundaries. The [roadmap](../roadmap/README.md) owns work identity, execution order, lifecycle status, and delivery history. Verification records under `docs/verification/` hold the evidence behind each checked requirement.

Release archives carry this document, the architecture, the roadmap, and the root README. Links to verification records and other maintainer documents resolve only in a source checkout. Requirement anchors and identifiers are stable and are never reused. Resolve a contradiction between the three documents in the same scoped change, and never weaken a requirement merely because an installed provider CLI does not support it. Read the roadmap's [execution rules](../roadmap/README.md#plan-execution) before starting work.

A checked box means that the requirement's complete acceptance scope is verified; the [completion rules](#completion-rules) define what that takes. The [acceptance history](#acceptance-history) summarizes how the checklist reached its current state.

The implemented baseline below remains weekly-only for Codex, Grok and Claude.
[Planned provider extensions](#planned-provider-extensions) define adopted,
unchecked requirements for EPIC-012 and EPIC-013. Their registration does not
change current behavior or reuse earlier acceptance as evidence for the new scope.

## Product outcome

**Gaebokchi (개복치)** is a small, read-only Go CLI in the **Aquarium tool ecosystem**. It shows how much of each provider-reported **weekly subscription allowance** has been used for configured Codex (ChatGPT), Grok, and Claude accounts. It reuses the logins that the installed provider CLIs already manage instead of implementing account linking.

The project name is **Gaebokchi (개복치)**; the repository basename and executable are `gaebokchi`, as defined in [ARCH-01](../architecture/README.md#arch-01). A usage run requires `~/.config/gaebokchi/config.yaml` and queries only its listed accounts in Codex, Grok, Claude order ([REQ-002](#req-002)); each provider list preserves its file order. Each row displays its configured alias, including when two different providers use the same alias.

One ordered result set has three presentations, described under [output modes](#output-modes): an interactive terminal view, the plain table for every other stdout, and a JSON snapshot selected with `--json`. Illustrative plain-table output, not measured account data:

```text
AI PROVIDER  ACCOUNT               WEEKLY USAGE
ChatGPT      work  37%
ChatGPT      personal              12%
Grok         personal               54%
Claude       work                  43.2%
```

## Scope boundaries

The implemented checklist and the explicitly marked planned extensions define the product scope. UI scraping, a PTY, an API-cost dashboard, browser session extraction, custom OAuth or credential handling, model prompting, background refresh, watch mode, and persistent history or caches are out of scope. A runtime configuration file is required for account selection. No application-owned credential or usage cache is created.

Provider compatibility is an implementation gate. Published documentation or source can establish a candidate interface, but only a real prompt-free probe establishes that an installed binary exposes it. An internal Grok billing handler is not proof of a usable stdio endpoint, and Claude's human-readable `/usage` text is not a published structured quota schema. See [ARCH-12](../architecture/README.md#arch-12) and [TASK-001](../roadmap/README.md#task-001).

## Acceptance checklist

The 2026-09-29 configuration extension changes REQ-002, REQ-006 through REQ-008, REQ-010 through REQ-011, and REQ-039. [TASK-032](../verification/TASK-032.md) re-qualified their current contract with parser, environment, and executable tests plus a positive live run of two Codex, one Grok, and one Claude target. Distinct real Grok and Claude logins still need separate compatibility qualification; see [deferred feedback](../deferred-feedback/README.md).

### Product scope

<a id="req-001"></a>
- [x] **REQ-001: Native Go CLI.** Provide an executable named `gaebokchi` for macOS and Linux. Use a small Go codebase and the installed provider CLIs; a Go installation is not required to run a distributed binary.
  - Architecture: [ARCH-02](../architecture/README.md#arch-02). Owner: [TASK-012](../roadmap/README.md#task-012). Verification: [V-BUILD](../architecture/README.md#v-build). Evidence: [TASK-012](../verification/TASK-012.md) — self-contained CGO-free binaries for macOS and Linux; `gaebokchi` executable with version metadata.
  - Changed 2026-09-23 (Claude extension): "the installed `codex` and `grok` executables" became "the installed provider CLIs"; [REQ-056](#req-056) covers the Claude CLI.

<a id="req-002"></a>
- [x] **REQ-002: Configured account set.** Require `~/.config/gaebokchi/config.yaml` for usage. Query only configured accounts, ordered Codex, Grok, Claude, `agy` and then by each list. Omitted or empty provider lists disable that provider. Require at least one account. Preserve every configured row when another target fails. Missing or invalid configuration exits 2 before any provider starts and prints its path and the GitHub configuration guide link to stderr. Help and version work without the file.
  - Architecture: [ARCH-03](../architecture/README.md#arch-03), [ARCH-18](../architecture/README.md#arch-18). Owner: [TASK-010](../roadmap/README.md#task-010). Verification: [V-OUTPUT](../architecture/README.md#v-output). Evidence: [TASK-010](../verification/TASK-010.md) — the default run queried `codex:0`, `codex:1`, and `grok:0` and rendered the header plus three rows; per-target failures kept their rows.
  - Changed 2026-09-23 (Claude extension, historical): the default set gained the Claude target, so that release queried four targets. The then-current CLI tests asserted the ordered four-row default; they ran in the [TASK-030](../verification/TASK-030.md) `make test` gate.
  - Re-qualified 2026-09-29: [TASK-032](../verification/TASK-032.md) covers current YAML selection, no-launch configuration errors, ordered rows and partial failure; the live run returned four configured rows.
  - Changed 2026-10-17 (agy extension): added optional single-entry `agy` configuration after Claude, expanding into two ordered weekly rows (`gemini` and `claude/gpt`); verified in [agy compatibility](../verification/agy-compatibility.md) ([REQ-064](#req-064)).

<a id="req-003"></a>
- [x] **REQ-003: Meaning of weekly usage.** Display the percentage already consumed in a provider-reported weekly subscription quota, with the Grok omission fallback in REQ-034 explicitly documented as a display convention rather than a measured zero. Do not present remaining quota, local token totals, API spending, or an inferred calendar-week calculation as this value. `ChatGPT` identifies the Codex-backed row, not an aggregation of every ChatGPT feature.
  - Architecture: [ARCH-01](../architecture/README.md#arch-01). Owner: [TASK-006](../roadmap/README.md#task-006). Verification: [V-QUOTA](../architecture/README.md#v-quota). Evidence: [TASK-006](../verification/TASK-006.md) — the only metric surfaced is the provider-reported weekly `usedPercent`; credits, token summaries, and reset times never feed the value; `ChatGPT` labels the Codex row via `ProviderCodex.DisplayName`.

<a id="req-004"></a>
- [x] **REQ-004: One-shot operation.** Perform one finite snapshot operation and exit. Do not introduce an application daemon, watch mode, polling, scheduled refresh, retry loop, usage history, or persistent usage cache. Blocking reads of protocol messages are permitted.
  - Architecture: [ARCH-05](../architecture/README.md#arch-05). Owner: [TASK-009](../roadmap/README.md#task-009). Verification: [V-RUNNER](../architecture/README.md#v-runner). Evidence: [TASK-009](../verification/TASK-009.md) — one bounded run per invocation, exactly one query attempt per target, no retry, daemon, or cache; the CLI exits after rendering.

<a id="req-005"></a>
- [x] **REQ-005: Real provider support.** Qualify the Codex and Grok integrations through real, prompt-free CLI queries before claiming them complete. An `UNSUPPORTED` fallback, a synthetic fixture, or a handler found only in source is not evidence of a working integration. [REQ-056](#req-056) applies the same gate to Claude.
  - Architecture: [ARCH-12](../architecture/README.md#arch-12). Owner: [TASK-013](../roadmap/README.md#task-013). Verification: [V-LIVE](../architecture/README.md#v-live). Evidence: [live acceptance](../verification/live-acceptance.md) — live prompt-free queries through both installed CLIs via the delivered binary; no `UNSUPPORTED` or fixture fallback used.
  - Changed 2026-09-29: this earlier live evidence does not qualify the new `GROK_HOME` or `CLAUDE_CONFIG_DIR` multi-account selection. That follow-up is recorded in [deferred feedback](../deferred-feedback/README.md).

### CLI and output

<a id="req-006"></a>
- [x] **REQ-006: Small documented command surface.** Implement only the command surface in ARCH-03: help; the `version [--json]` command; executable overrides; the per-target timeout; and `--json` ([REQ-052](#req-052)). Reject removed `--codex-home` and `--grok-account` flags. Help and version must not launch a provider process. Reject unknown flags and unsupported positional arguments, including the removed `--version` form. Resolve an explicit relative executable override against the invocation directory before any child starts ([REQ-039](#req-039)). Normalize configured aliases with the display-label policy of [REQ-012](#req-012); a blank alias is a configuration error before any provider starts.
  - Architecture: [ARCH-03](../architecture/README.md#arch-03), [ARCH-18](../architecture/README.md#arch-18). Owner: [TASK-010](../roadmap/README.md#task-010). Re-qualified: [TASK-017](../roadmap/README.md#task-017), [TASK-019](../roadmap/README.md#task-019), [TASK-023](../roadmap/README.md#task-023). Verification: [V-OPTIONS](../architecture/README.md#v-options). Evidence: [TASK-002](../verification/TASK-002.md) parser matrix plus [TASK-010](../verification/TASK-010.md) help, version, no-launch, and exit-code tests; [TASK-017](../verification/TASK-017.md) relative executable overrides; [TASK-019](../verification/TASK-019.md) and [TASK-023](../verification/TASK-023.md) alias normalization.
  - Changed 2026-09-23 (Claude extension) and 2026-09-25 (EPIC-009): the surface also includes `--claude-bin` ([REQ-056](#req-056)) and `--json` ([REQ-052](#req-052)).
  - Re-qualified 2026-09-29: [TASK-032](../verification/TASK-032.md) covers the YAML-era command and error matrix, relative executable overrides, alias validation, and no-launch help/version paths.
  - Changed 2026-10-17 (agy extension): added `--agy-bin` ([REQ-064](#req-064)); verified in [agy compatibility](../verification/agy-compatibility.md).

<a id="req-007"></a>
- [x] **REQ-007: Configured provider homes.** Each configured Codex, Grok, or Claude account has a required `name` and `path`. Set `CODEX_HOME` or `GROK_HOME` on that target's child process. For a non-default Claude home, set `CLAUDE_CONFIG_DIR`; for the current user's `~/.claude`, remove ambient `CLAUDE_CONFIG_DIR` and use the CLI's implicit default credential namespace. An enabled `agy` entry uses the single default CLI login (`HOME` preserved) and does not require `path` ([REQ-064](#req-064)). Never discover accounts implicitly or let ambient values select an account. Account names are display aliases, not login selectors or verified identities.
  - Architecture: [ARCH-03](../architecture/README.md#arch-03), [ARCH-18](../architecture/README.md#arch-18). Owner: [TASK-002](../roadmap/README.md#task-002). Verification: [V-OPTIONS](../architecture/README.md#v-options). Evidence: [TASK-002](../verification/TASK-002.md) — replacement order, ambient `CODEX_HOME` ignored, Grok after the Codex homes.
  - Changed 2026-09-23 (Claude extension): Grok is no longer the last target.
  - Changed 2026-09-29 (default Claude home): keep the CLI's implicit credential namespace when `path` is the current user's `~/.claude`; custom Claude homes still set `CLAUDE_CONFIG_DIR`.
  - Re-qualified 2026-09-29: [TASK-032](../verification/TASK-032.md) covers target-home environment isolation and the default Claude namespace. Distinct live Grok and non-default Claude logins remain an explicit compatibility limit.
  - Changed 2026-10-17 (agy extension): `agy` queries the single default CLI login (`HOME` preserved) with `path` optional and not used as a credential selector; verified in [agy compatibility](../verification/agy-compatibility.md).

<a id="req-008"></a>
- [x] **REQ-008: Safe path handling.** Accept absolute account paths and current-user `~/` paths; expand `~/` before use without collapsing `.` or `..` ahead of filesystem traversal. Reject relative paths, empty paths, and duplicate paths within a provider, including aliases through existing symlink ancestors of missing paths. For missing paths with the same deepest existing parent, reject suffixes equal after Unicode canonical decomposition and case folding on every platform; distinct existing directories retain their filesystem identities. Do not create missing provider directories; report them on their rows. Reject duplicate normalized names, unknown configuration keys, YAML merge keys, and null provider lists.
  - Architecture: [ARCH-03](../architecture/README.md#arch-03). Owner: [TASK-002](../roadmap/README.md#task-002). Verification: [V-OPTIONS](../architecture/README.md#v-options). Evidence: [TASK-002](../verification/TASK-002.md) — tilde-only expansion, `~user` rejected, relative path rejection, duplicate and symlink rejection, missing homes never created.
  - Re-qualified 2026-09-29: [TASK-032](../verification/TASK-032.md) covers strict YAML validation and the current path-identity matrix, including missing-path Unicode/case aliases and symlink traversal.

<a id="req-009"></a>
- [x] **REQ-009: Configurable finite deadline.** Default to a 20-second query budget per admitted target; support `--timeout` as a Go duration from 1 second through 120 seconds. Include startup and every provider request in that single budget. Bound cleanup separately with one absolute allowance of at most 2 seconds. A result established within the query budget survives cleanup that runs past the deadline.
  - Architecture: [ARCH-05](../architecture/README.md#arch-05). Owner: [TASK-009](../roadmap/README.md#task-009). Re-qualified: [TASK-016](../roadmap/README.md#task-016). Verification: [V-LIFECYCLE](../architecture/README.md#v-lifecycle). Evidence: [TASK-002](../verification/TASK-002.md) (1s–120s validation, 20s default), [TASK-009](../verification/TASK-009.md) (admission-time budget covering startup and every RPC), [TASK-003](../verification/TASK-003.md) (separate 2s cleanup bound), [TASK-016](../verification/TASK-016.md) (established results survive cleanup; one absolute cleanup budget).

<a id="req-010"></a>
- [x] **REQ-010: Account labels.** Display each configured `name` on every status, including when Codex returns a protocol email. Normalize the alias with [REQ-012](#req-012), and reject duplicate visible names within a provider before querying. The selected path chooses the CLI account; the alias does not prove its real identity.
  - Architecture: [ARCH-04](../architecture/README.md#arch-04). Owner: [TASK-010](../roadmap/README.md#task-010). Re-qualified: [TASK-019](../roadmap/README.md#task-019), [TASK-023](../roadmap/README.md#task-023). Verification: [V-IDENTITY](../architecture/README.md#v-identity). Evidence: [TASK-010](../verification/TASK-010.md) and the TASK-006 and TASK-008 identity suites — protocol email preferred, home-basename fallback, Grok alias, duplicate disambiguation; [TASK-019](../verification/TASK-019.md) and [TASK-023](../verification/TASK-023.md) — normalized labels and the target-index suffix rule.
  - Re-qualified 2026-09-29: the prior email/fallback rule is historical. [TASK-032](../verification/TASK-032.md) covers configured aliases on every current row and duplicate visible-name rejection before launch.

<a id="req-011"></a>
- [x] **REQ-011: Fixed logical table.** For non-interactive, non-JSON usage, use exactly the three headings `AI PROVIDER`, `ACCOUNT`, and `WEEKLY USAGE`. Emit no title, separator, progress row, reset-time column, totals, or blank lines. Emit the header and exactly one newline-terminated logical line per configured target (N+1 lines for N accounts).
  - Architecture: [ARCH-10](../architecture/README.md#arch-10). Owner: [TASK-010](../roadmap/README.md#task-010). Verification: [V-OUTPUT](../architecture/README.md#v-output). Evidence: [TASK-010](../verification/TASK-010.md) golden tests — three headings, no extras, four lines by default, the N+2 rule, final newline.
  - Changed 2026-09-23 (Claude extension, historical): the Claude row made that default table five lines. The then-current CLI tests asserted five default lines and N+3 lines with three custom homes; they ran in the [TASK-030](../verification/TASK-030.md) `make test` gate.
  - Changed 2026-09-26 ([EPIC-010](../roadmap/README.md#epic-010)): this table governs non-interactive, non-JSON usage stdout ([REQ-053](#req-053)); interactive terminals use the live view.
  - Re-qualified 2026-09-29: the historical default and N+2 counts above are superseded. [TASK-032](../verification/TASK-032.md) covers exactly N+1 lines for the current configured target count.

<a id="req-012"></a>
- [x] **REQ-012: Safe output.** Write usage output only to stdout and concise diagnostics only to stderr. Do not forward raw provider output. Non-interactive stdout has no color or terminal control sequences. Sanitize account text so embedded tabs, newlines, escape sequences, and control or format characters cannot add columns or rows. The shared display-label policy treats every Unicode whitespace character as whitespace, so no whitespace-only label renders as a blank cell, and it preserves ordinary printable Unicode such as Korean and other CJK text.
  - Architecture: [ARCH-10](../architecture/README.md#arch-10). Owner: [TASK-010](../roadmap/README.md#task-010). Re-qualified: [TASK-023](../roadmap/README.md#task-023). Verification: [V-OUTPUT](../architecture/README.md#v-output). Evidence: [TASK-010](../verification/TASK-010.md) — stdout table only, stderr diagnostics only, no raw provider output, control-character sanitization; [TASK-023](../verification/TASK-023.md) — complete Unicode whitespace policy.
  - Changed 2026-09-26 ([EPIC-010](../roadmap/README.md#epic-010)): the ban on color and terminal control sequences governs non-interactive stdout. The terminal view keeps stdout/stderr separation, excludes raw provider output, and sanitizes labels ([REQ-054](#req-054)).

<a id="req-051"></a>
- [x] **REQ-051: Stable executable version contract.** Provide the project-independent commands `gaebokchi version` and `gaebokchi version --json`. Human output is `gaebokchi <version>`; JSON contains exactly `name` and `version`. Release versions and versioned staging paths use a `v` prefix. The version comes from the release linker override, otherwise from Go build information, and a build that supplies neither reports `(devel)`. `--version` is rejected. Version lookup must not launch providers or depend on HOME, the working directory, or project state.
  - Architecture: [ARCH-03](../architecture/README.md#arch-03), [ARCH-13](../architecture/README.md#arch-13). Owner: [TASK-028](../roadmap/README.md#task-028). Verification: [V-OPTIONS](../architecture/README.md#v-options), [V-BUILD](../architecture/README.md#v-build). Evidence: [TASK-028](../verification/TASK-028.md) records exact human and JSON behavior, development and release metadata, invalid syntax, output failure, full offline and race gates, four target builds, document-fresh archives, checksums, and a native darwin/arm64 smoke.

### Usage semantics

<a id="req-013"></a>
- [x] **REQ-013: Presence-aware percentages.** Emit `%` for a finite, explicitly present numeric value from 0 through 100. Omit the decimal when the original value is an integer; otherwise show one decimal place. Preserve a genuine zero as `0%`. Reject negative, over-100, non-numeric, or non-finite values; do not clamp them or multiply ratios by 100. Do not substitute zero for missing fields except for the narrowly qualified Grok weekly omission in REQ-034.
  - Architecture: [ARCH-04](../architecture/README.md#arch-04). Owner: [TASK-008](../roadmap/README.md#task-008). Verification: [V-QUOTA](../architecture/README.md#v-quota). Evidence: [TASK-008](../verification/TASK-008.md) and the TASK-006 suites — explicit zero preserved; out-of-range and non-numeric values rejected without clamping or ratio conversion; the current renderer tests cover adaptive precision.
  - Changed 2026-09-27: whole values omit the decimal. TASK-008 verified the earlier format, which always showed one decimal place.

<a id="req-014"></a>
- [x] **REQ-014: No invented weekly value.** Show a defined unavailable or non-applicable status when the response cannot establish a weekly value, except for the explicit Grok display convention in REQ-034. Do not average multiple quotas, choose the highest percentage, extrapolate monthly values, or guess a window from reset time.
  - Architecture: [ARCH-04](../architecture/README.md#arch-04). Owner: [TASK-008](../roadmap/README.md#task-008). Verification: [V-QUOTA](../architecture/README.md#v-quota). Evidence: [TASK-008](../verification/TASK-008.md) — defined unavailable and non-applicable statuses for every weekly value that cannot be established; no averaging or extrapolation anywhere.

<a id="req-015"></a>
- [x] **REQ-015: Window provenance.** Keep the selected quota identifier, period kind, source, and optional reset time in the in-memory result, on monthly and weekly branches alike and including a weekly window whose usage is missing. Never invent a quota identifier. Reset metadata is not a table column or JSON field and must not be required when the source makes it optional.
  - Architecture: [ARCH-04](../architecture/README.md#arch-04). Owner: [TASK-008](../roadmap/README.md#task-008). Re-qualified: [TASK-019](../roadmap/README.md#task-019). Verification: [V-QUOTA](../architecture/README.md#v-quota). Evidence: [TASK-006](../verification/TASK-006.md) and TASK-008 — quota identifier, period kind, source, and optional reset preserved in domain results; [TASK-019](../verification/TASK-019.md) — period provenance on the monthly branch.
  - Changed 2026-09-26 ([EPIC-010](../roadmap/README.md#epic-010)): the interactive terminal view may show a countdown from optional reset metadata ([REQ-054](#req-054)). The plain table and JSON still add no reset column or field.

### Protocol and execution

<a id="req-016"></a>
- [x] **REQ-016: Separate wire dialects.** Use stdio JSON messages with the correct provider-specific envelopes and initialization rules. Codex App Server and Grok ACP must not be treated as one interchangeable protocol. Freeze the Grok extension wire encoding from an actual installed-binary probe. Validate inbound responses per dialect, including the ACP `"jsonrpc": "2.0"` marker, and enforce each handshake's qualified result shape while tolerating additive members. Claude uses local CLI commands rather than an RPC dialect ([REQ-056](#req-056)).
  - Architecture: [ARCH-06](../architecture/README.md#arch-06). Owner: [TASK-004](../roadmap/README.md#task-004). Re-qualified: [TASK-018](../roadmap/README.md#task-018). Verification: [V-RPC](../architecture/README.md#v-rpc). Evidence: [TASK-004](../verification/TASK-004.md) plus the frozen live encoding `_x.ai/billing` from the [compatibility record](../verification/compatibility.md); [TASK-018](../verification/TASK-018.md) — per-dialect inbound validation and handshake shapes.

<a id="req-017"></a>
- [x] **REQ-017: Bounded framing.** Support fragmented and coalesced newline-delimited UTF-8 JSON, with a 4 MiB maximum frame. Reject malformed or oversized frames and unsupported batches; bound buffered stderr to 64 KiB while continuing to drain excess bytes.
  - Architecture: [ARCH-06](../architecture/README.md#arch-06). Owner: [TASK-004](../roadmap/README.md#task-004). Verification: [V-RPC](../architecture/README.md#v-rpc). Evidence: [TASK-004](../verification/TASK-004.md) — framing matrix including boundary sizes; 64 KiB stderr drain in TASK-003.

<a id="req-018"></a>
- [x] **REQ-018: Response correlation.** Correlate responses by request ID, tolerate unrelated well-formed notifications, and never use the first received JSON object as the response. Unknown or duplicate response IDs cannot satisfy a pending request. A matching response must carry exactly one structurally valid `result` or `error`; `error: null` without a result, an incomplete error object, or a shapeless response is malformed. Fail a pending call on EOF or a malformed matching response.
  - Architecture: [ARCH-06](../architecture/README.md#arch-06). Owner: [TASK-004](../roadmap/README.md#task-004). Re-qualified: [TASK-018](../roadmap/README.md#task-018). Verification: [V-RPC](../architecture/README.md#v-rpc). Evidence: [TASK-004](../verification/TASK-004.md); [TASK-018](../verification/TASK-018.md) — malformed matching responses fail the call.

<a id="req-019"></a>
- [x] **REQ-019: No host capability execution.** Advertise no filesystem, terminal, or tool execution capability to Grok. Never execute server-initiated requests, approve tools, answer elicitation with secrets, or service token-refresh callbacks requiring application-held credentials. Reject unexpected requests and close the affected query safely.
  - Architecture: [ARCH-09](../architecture/README.md#arch-09). Owner: [TASK-004](../roadmap/README.md#task-004). Verification: [V-SAFETY](../architecture/README.md#v-safety). Evidence: [TASK-004](../verification/TASK-004.md) — server requests rejected with the exact ID and the connection aborted; Grok's false capabilities are declared in the TASK-007 wire tests.

<a id="req-020"></a>
- [x] **REQ-020: Owned-process cleanup.** Close stdin and reap each owned child exactly once. Apply bounded EOF, termination, and kill cleanup on success, error, timeout, and cancellation within one absolute cleanup deadline. Signal only the direct child that gaebokchi started, on both the graceful and the forced path; never signal its private process group, unrelated CLI sessions, or provider-managed shared leaders. Report incomplete cleanup as [REQ-040](#req-040) requires.
  - Architecture: [ARCH-05](../architecture/README.md#arch-05). Owner: [TASK-003](../roadmap/README.md#task-003). Re-qualified: [TASK-016](../roadmap/README.md#task-016), [TASK-022](../roadmap/README.md#task-022). Verification: [V-LIFECYCLE](../architecture/README.md#v-lifecycle). Evidence: [TASK-003](../verification/TASK-003.md) — stdin close, exactly-once wait, staged EOF, SIGTERM, and SIGKILL within budget, unrelated-process survival; [TASK-016](../verification/TASK-016.md) — one absolute cleanup deadline; [TASK-022](../verification/TASK-022.md) — direct-child-only cleanup with reported incompleteness.

<a id="req-021"></a>
- [x] **REQ-021: Safe diagnostics.** Classify errors using local failure types and verified structured protocol errors. Do not print auth files, bearer tokens, environment values, raw RPC bodies, or raw child stderr. A generic sanitized error is preferable to unsafe or guessed detail. For example, a Grok `authenticate` method-not-found error is `UNSUPPORTED` and any other remote authentication error is `ERROR`; `AUTH REQUIRED` comes only from the advertised authentication methods ([REQ-032](#req-032)).
  - Architecture: [ARCH-09](../architecture/README.md#arch-09). Owner: [TASK-011](../roadmap/README.md#task-011). Re-qualified: [TASK-019](../roadmap/README.md#task-019). Verification: [V-SAFETY](../architecture/README.md#v-safety). Evidence: [TASK-011](../verification/TASK-011.md) — fixed application-owned messages only; secret markers and raw provider output never reach stdout or stderr; [TASK-019](../verification/TASK-019.md) — structure-based Grok authentication errors.

<a id="req-022"></a>
- [x] **REQ-022: Independent bounded queries.** Allow at most three target queries concurrently, using isolated environment slices and result slots. A target beyond the first three waits for a free slot. A failure in one target must not cancel healthy siblings. Preserve configured row order regardless of completion order.
  - Architecture: [ARCH-05](../architecture/README.md#arch-05). Owner: [TASK-009](../roadmap/README.md#task-009). Verification: [V-RUNNER](../architecture/README.md#v-runner). Evidence: [TASK-009](../verification/TASK-009.md) — at most three active, isolated environments and slots, sibling preservation under failure or hang, configured order.
  - Changed 2026-09-23 (Claude extension, historical): one of that release's four default targets waited for a slot.

<a id="req-023"></a>
- [x] **REQ-023: Signal handling.** On SIGINT or SIGTERM, cancel pending work, clean up owned processes, preserve already obtained results, and render unresolved rows as `CANCELLED` when stdout is writable. Cancellation must also unblock a provider write that is blocked on a wedged child, and results established before the signal survive it. Do not start targets still queued after cancellation.
  - Architecture: [ARCH-05](../architecture/README.md#arch-05). Owner: [TASK-009](../roadmap/README.md#task-009). Re-qualified: [TASK-016](../roadmap/README.md#task-016). Verification: [V-LIFECYCLE](../architecture/README.md#v-lifecycle). Evidence: [TASK-009](../verification/TASK-009.md) (queued cancellation, no late launch), TASK-010 (130/143 and the `CANCELLED` table), TASK-011 (SIGINT end to end), [TASK-016](../verification/TASK-016.md) (blocked writes released, established results kept).

<a id="req-024"></a>
- [x] **REQ-024: Exit contract.** Implement exit codes 0, 1, 2, 130, and 143 exactly as ARCH-10 defines. Keep partial results visible even when exit code 1 is returned. Explicit monthly or API-key non-applicability is not an operational failure; missing data and unsupported methods are incomplete results. A closed stdout pipe yields exit 1 rather than death by `SIGPIPE`, and 130 or 143 take precedence over an output failure.
  - Architecture: [ARCH-10](../architecture/README.md#arch-10). Owner: [TASK-010](../roadmap/README.md#task-010). Re-qualified: [TASK-017](../roadmap/README.md#task-017). Verification: [V-EXIT](../architecture/README.md#v-exit). Evidence: [TASK-010](../verification/TASK-010.md) — codes 0, 1, 2, and 130 exercised end to end; 143 shares the SIGTERM path; non-applicability is clean; [TASK-017](../verification/TASK-017.md) — broken pipes and signal precedence against the actual executable.

### Codex provider

<a id="req-025"></a>
- [x] **REQ-025: Per-target App Server.** Start a separate `codex app-server` process for each Codex home, with that target’s absolute `CODEX_HOME`. Do not switch accounts inside a shared App Server or mutate the parent environment.
  - Architecture: [ARCH-07](../architecture/README.md#arch-07). Owner: [TASK-005](../roadmap/README.md#task-005). Verification: [V-CODEX](../architecture/README.md#v-codex). Evidence: [TASK-005](../verification/TASK-005.md) — per-target child with a filtered environment replacing `CODEX_HOME`; parent environment unchanged; two simultaneous isolated sessions.

<a id="req-026"></a>
- [x] **REQ-026: Metadata-only handshake.** Complete `initialize`, wait for its response, then send `initialized` before the account calls. Use only the allowlisted account-read and rate-limit-read operations; do not start or resume threads or turns.
  - Architecture: [ARCH-07](../architecture/README.md#arch-07). Owner: [TASK-005](../roadmap/README.md#task-005). Verification: [V-CODEX](../architecture/README.md#v-codex). Evidence: [TASK-005](../verification/TASK-005.md) — transcript-asserted ordering and outbound surface.

<a id="req-027"></a>
- [x] **REQ-027: Existing subscription auth.** Use provider-managed ChatGPT login state without reading credentials directly. An absent or expired login becomes `AUTH REQUIRED`; API-key-only auth becomes `N/A (API key)`. Other unqualified auth variants must not be silently treated as ChatGPT subscriptions.
  - Architecture: [ARCH-07](../architecture/README.md#arch-07). Owner: [TASK-005](../roadmap/README.md#task-005). Verification: [V-CODEX](../architecture/README.md#v-codex). Evidence: [TASK-005](../verification/TASK-005.md) — `chatgpt`, `apiKey`, absent, and unknown account mapping without credential reads.

<a id="req-028"></a>
- [x] **REQ-028: Deterministic quota bucket selection.** Prefer `rateLimitsByLimitId["codex"]` when the multi-bucket map is non-empty. Do not fall through to an unrelated bucket or average map entries. Use legacy `rateLimits` only under the explicit compatibility rules in ARCH-07.
  - Architecture: [ARCH-07](../architecture/README.md#arch-07). Owner: [TASK-006](../roadmap/README.md#task-006). Verification: [V-CODEX](../architecture/README.md#v-codex). Evidence: [TASK-006](../verification/TASK-006.md) — bucket-selection table: a non-empty map without `codex` is `no_weekly`, never legacy or an average; a null `codex` entry is `no_data`; a wrong type is a protocol error; an empty or null map uses legacy data only with an explicit `codex` `limitId`.

<a id="req-029"></a>
- [x] **REQ-029: Duration-based weekly window.** Inspect both `primary` and `secondary` in the selected Codex bucket. Select by `windowDurationMins == 10080`, never by slot name. Handle absent, null, reversed, duplicated, and conflicting windows as defined in ARCH-07.
  - Architecture: [ARCH-07](../architecture/README.md#arch-07). Owner: [TASK-006](../roadmap/README.md#task-006). Verification: [V-CODEX](../architecture/README.md#v-codex). Evidence: [TASK-006](../verification/TASK-006.md) — slot matrix including reversed slots, five-hour-only rejection, agreeing duplicates (37 and 37.0), and conflicting weekly windows.

<a id="req-030"></a>
- [x] **REQ-030: Identity isolation.** Obtain Codex account information and quota through the same isolated child. Preserve the correct fallback identity on authentication or transport failure, and test simultaneous results from two different homes.
  - Architecture: [ARCH-07](../architecture/README.md#arch-07). Owner: [TASK-006](../roadmap/README.md#task-006). Verification: [V-IDENTITY](../architecture/README.md#v-identity). Evidence: [TASK-006](../verification/TASK-006.md) — identity and quota from the same child; simultaneous two-home lifecycle tests with distinct emails and percentages; home-alias fallback preserved on every failure path.

### Grok provider

<a id="req-031"></a>
- [x] **REQ-031: ACP-only query path.** Use the qualified headless stdio invocation `grok --no-auto-update agent stdio`. Do not use the interactive TUI, a PTY, `-p`, an undocumented REST fallback, or an application-implemented billing HTTP client.
  - Architecture: [ARCH-08](../architecture/README.md#arch-08). Owner: [TASK-007](../roadmap/README.md#task-007). Verification: [V-GROK](../architecture/README.md#v-grok). Evidence: [TASK-007](../verification/TASK-007.md) — exact `--no-auto-update agent stdio` invocation asserted from the child argv; no TUI, PTY, or REST path anywhere (TASK-011 static scan).

<a id="req-032"></a>
- [x] **REQ-032: Cached login without new auth.** Select `cached_token` only when advertised and authenticate with headless metadata. When `cached_token` is absent, an advertisement that still offers the interactive `grok.com` method means `AUTH REQUIRED`, even if `api_key` is also offered; only an API-key-only advertisement is `N/A (API key)`, and unrecognized combinations are never treated as clean. Do not choose an API key because it exists in the environment, launch a browser, run `grok login`, or initiate device-code or OIDC login.
  - Architecture: [ARCH-08](../architecture/README.md#arch-08). Owner: [TASK-007](../roadmap/README.md#task-007). Re-qualified: [TASK-019](../roadmap/README.md#task-019). Verification: [V-GROK](../architecture/README.md#v-grok). Evidence: [TASK-007](../verification/TASK-007.md) — `cached_token` selected only when advertised, headless `_meta`, API-key environment never selects auth, no browser, login, or device flows; [TASK-019](../verification/TASK-019.md) — advertisement decision table.

<a id="req-033"></a>
- [x] **REQ-033: Qualified billing extension.** Call the source-backed billing extension using the captured ACP wire contract. Distinguish the internal `x.ai/billing` name from its wire representation. A valid method-not-found response produces `UNSUPPORTED`, not fabricated usage or an automatic alternate transport.
  - Architecture: [ARCH-08](../architecture/README.md#arch-08). Owner: [TASK-007](../roadmap/README.md#task-007). Verification: [V-GROK](../architecture/README.md#v-grok). Evidence: [TASK-007](../verification/TASK-007.md) — frozen `_x.ai/billing` wire method from the live probe; method-not-found maps to `UNSUPPORTED` (TASK-008); no alternate transports.

<a id="req-034"></a>
- [x] **REQ-034: Explicit weekly billing semantics.** Use `config.creditUsagePercent` only when `config.currentPeriod.type` explicitly identifies the qualified weekly enum. For that weekly enum only, display `0%` when the percentage member is omitted; this is a chosen compatibility convention, not a verified zero-usage measurement. An explicit `null` percentage remains `N/A (no data)`. An explicit monthly enum produces `N/A (monthly)`; absent or unknown period metadata does not establish a week.
  - Architecture: [ARCH-08](../architecture/README.md#arch-08). Owner: [TASK-008](../roadmap/README.md#task-008). Verification: [V-GROK](../architecture/README.md#v-grok). Evidence: [TASK-008](../verification/TASK-008.md) — `creditUsagePercent` read only under the explicit weekly enum; monthly is `N/A (monthly)`; absent or unknown periods are `N/A (period)`.
  - Changed 2026-09-27: the [Grok 1.0.41 compatibility evidence](../verification/compatibility.md#grok-1041-omitted-weekly-percentage) and the current normalization and query tests qualify the omitted-member fallback. TASK-008 records the earlier policy, which mapped an omitted percentage to `N/A`.

<a id="req-035"></a>
- [x] **REQ-035: No legacy-credit estimation.** Do not compute weekly usage from `monthlyLimit`, `used`, purchased or prepaid balance, on-demand usage, or billing history. Treat legacy-only data as unavailable rather than converting it.
  - Architecture: [ARCH-08](../architecture/README.md#arch-08). Owner: [TASK-008](../roadmap/README.md#task-008). Verification: [V-GROK](../architecture/README.md#v-grok). Evidence: [TASK-008](../verification/TASK-008.md) — legacy `monthlyLimit`, `used`, prepaid, and on-demand payloads map to `N/A (period)` and are never converted.

<a id="req-036"></a>
- [x] **REQ-036: No speculative identity endpoint.** Keep the Grok alias available on success and failure. Do not read Grok auth JSON, decode JWTs, or invent a `whoami` or profile API to fill the ACCOUNT column.
  - Architecture: [ARCH-08](../architecture/README.md#arch-08). Owner: [TASK-008](../roadmap/README.md#task-008). Verification: [V-IDENTITY](../architecture/README.md#v-identity). Evidence: [TASK-008](../verification/TASK-008.md) — configured alias on success and failure; auth payloads are never identity sources; no `whoami`, JWT, or auth-file code (TASK-011 static scan).

<a id="current-claude-extension"></a>
### Claude provider

Claude support was added after the original v0.1 campaign; [ADR-0001](../architecture-decision-records/ADR-0001-claude-usage-local-command.md) records the decision and [ARCH-16](../architecture/README.md#arch-16) the adapter. [TASK-032](../verification/TASK-032.md) records the current candidate's positive live result and failure-path tests under the [completion rules](#completion-rules).

<a id="req-056"></a>
- [x] **REQ-056: Claude targets and local-command query.** Query each configured Claude target after the Grok targets, using its selected Claude home and alias. Let `--claude-bin` select the executable; a missing executable keeps the row as `NOT INSTALLED`. Use exactly two prompt-free commands within the target's single query budget: `claude --safe-mode auth status --json`, then `claude --safe-mode --tools "" -p /usage --max-turns 0 --output-format json --no-session-persistence`. Close each child's stdin immediately and cap its stdout at 256 KiB; larger output is `PROTOCOL ERROR`. Do not use an interactive session, a PTY, a model turn, a tool, the experimental `stream-json` usage report, or an application-owned HTTP client.
  - Architecture: [ARCH-16](../architecture/README.md#arch-16). Owner: [TASK-032](../roadmap/README.md#task-032). Verification: [V-CLAUDE](../architecture/README.md#v-claude). Evidence: [Claude compatibility record](../verification/claude-compatibility.md) (live, Claude Code 2.1.280, darwin/arm64); the Claude provider suite ran in the [TASK-030](../verification/TASK-030.md) `make test` gate.
  - Re-qualified 2026-09-29: [TASK-032](../verification/TASK-032.md) records live Claude Code 2.1.284 success, exact-command tests, and an oversized-output regression. Distinct custom-home login qualification remains limited as documented.

<a id="req-057"></a>
- [x] **REQ-057: Claude authentication and environment.** Classify the auth status before reading usage. `loggedIn: false` is `AUTH REQUIRED`, whatever the command's exit status. Continue only when the command exits 0 and reports `loggedIn: true` with `authMethod` `claude.ai` and `apiProvider` `firstParty`. An auth method whose name identifies an API key is `N/A (API key)`; only fixtures qualify this branch. Any other combination, or output that does not parse or lacks `loggedIn`, is `UNSUPPORTED`. An auth command that cannot run, or that exits nonzero while logged in, is `ERROR`; a timeout or cancellation is `TIMEOUT` or `CANCELLED`. Remove Anthropic credential and third-party routing overrides from the child environment while keeping `HOME`. Set `CLAUDE_CONFIG_DIR` for non-default homes and leave it unset for the default `~/.claude` home, as listed in ARCH-16. Never read Claude credentials or start a login.
  - Architecture: [ARCH-16](../architecture/README.md#arch-16), [ARCH-09](../architecture/README.md#arch-09). Owner: [TASK-032](../roadmap/README.md#task-032). Verification: [V-CLAUDE](../architecture/README.md#v-claude). Evidence: the live auth read in the [Claude compatibility record](../verification/claude-compatibility.md); logged-out, API-key, unknown-mode, and environment tests ran in the [TASK-030](../verification/TASK-030.md) `make test` gate.
  - Re-qualified 2026-09-29: [TASK-032](../verification/TASK-032.md) combines the current live first-party positive path with auth-mode and child-environment tests. API-key and third-party branches remain fixture-qualified.

<a id="req-058"></a>
- [x] **REQ-058: Claude usage envelope and weekly line.** Accept the usage command's output only when it exits 0 with a JSON envelope that reports `local_command` `usage`, zero turns, zero cost, `is_error: false`, and a text `result`; anything else is `UNSUPPORTED`. Select the one line of the form `Current week (all models): N% used`, where N is a non-negative decimal. A missing or duplicated line, or one in any other form, including a negative value, is `UNSUPPORTED`; a value above 100 is `PROTOCOL ERROR`. Never substitute the current-session meter or a model-specific weekly meter.
  - Architecture: [ARCH-16](../architecture/README.md#arch-16). Owner: [TASK-032](../roadmap/README.md#task-032). Verification: [V-CLAUDE](../architecture/README.md#v-claude). Evidence: the live envelope and line shape in the [Claude compatibility record](../verification/claude-compatibility.md); the parser tests ran in the [TASK-030](../verification/TASK-030.md) `make test` gate.
  - Re-qualified 2026-09-29: [TASK-032](../verification/TASK-032.md) records a live numeric weekly result on the current CLI and the complete envelope and line-shape parser matrix.

<a id="req-059"></a>
- [x] **REQ-059: Claude reset metadata.** Keep a reset time only from the ` · resets …` suffix of the selected weekly line, and only when the text names exactly one instant: an en-US month and day with an optional year, joined to a 12-hour time by ` at ` or a comma, followed by an IANA zone that the host can load. When the year is omitted, use the zone's current year, and discard the reset if that year changed while the command ran. Also discard it for an unrecognized suffix, an unknown zone, an impossible date, or a local time that a daylight-saving change skips or repeats. A discarded reset never changes the percentage, and no reset time is derived from other data.
  - Architecture: [ARCH-16](../architecture/README.md#arch-16). Owner: [TASK-032](../roadmap/README.md#task-032). Verification: [V-CLAUDE](../architecture/README.md#v-claude). Evidence: the [weekly reset text](../verification/claude-compatibility.md#weekly-reset-text) record (live, Claude Code 2.1.283); [TASK-032](../verification/TASK-032.md) records current reset parser regressions. The 2.1.284 live read retained no reset timestamp, so reset presence on that version is unclaimed.

### Security and boundaries

<a id="req-037"></a>
- [x] **REQ-037: Credential ownership.** Leave credential discovery, keychain access, refresh, and storage to the installed CLIs. The application must not parse or copy auth files, request tokens from the user, store secrets, or implement OAuth.
  - Architecture: [ARCH-09](../architecture/README.md#arch-09). Owner: [TASK-011](../roadmap/README.md#task-011). Verification: [V-SAFETY](../architecture/README.md#v-safety). Evidence: [TASK-011](../verification/TASK-011.md) static scan (no `auth.json`, keychain, or token parsing) and provider suites that use provider-managed login state only.

<a id="req-038"></a>
- [x] **REQ-038: Business-level read-only boundary.** Issue no prompts, agent sessions, tool actions, login or logout, billing mutations, quota-reset redemption, purchases, or configuration writes. Distinguish this from provider-owned token refresh, logs, caches, and shared-process lifecycle that can occur during CLI startup. Claude's `-p /usage` invocation runs a local command, not a model prompt, under the zero-turn constraints of [REQ-056](#req-056).
  - Architecture: [ARCH-09](../architecture/README.md#arch-09). Owner: [TASK-011](../roadmap/README.md#task-011). Verification: [V-SAFETY](../architecture/README.md#v-safety). Evidence: [TASK-011](../verification/TASK-011.md) — allowlisted metadata methods only; no prompts, sessions, or mutations; static capability scan.
  - Changed 2026-09-23 (Claude extension): the last sentence records how the Claude local command fits this boundary.

<a id="req-039"></a>
- [x] **REQ-039: Isolated child environments.** Use explicit executable paths and argument arrays without a shell. Resolve an explicit relative executable to an absolute path before the child's working directory changes, without bypassing Go's `ErrDot` protection. Copy and filter each child environment; replace the configured provider home per target and exclude provider API-key overrides and agent-hosted parent-session markers identified during qualification. Preserve the required home, keychain, and proxy context. [REQ-057](#req-057) adds Claude's credential and routing overrides to the filtered set.
  - Architecture: [ARCH-09](../architecture/README.md#arch-09). Owner: [TASK-005](../roadmap/README.md#task-005). Re-qualified: [TASK-017](../roadmap/README.md#task-017). Verification: [V-SAFETY](../architecture/README.md#v-safety). Evidence: [TASK-005](../verification/TASK-005.md) — environment filter with drops and per-target `CODEX_HOME` replacement, `HOME` and proxy preserved, parent untouched; argv-only child start from TASK-003; [TASK-017](../verification/TASK-017.md) — relative executable identity made absolute.
  - Re-qualified 2026-09-29: [TASK-032](../verification/TASK-032.md) covers YAML-selected child homes and Claude's current credential/routing filter alongside the prior process-isolation contract.

<a id="req-040"></a>
- [x] **REQ-040: Neutral working directory.** Launch probes and runtime children in private empty temporary directories outside the caller’s project tree. Do not pass repository prompts, MCP servers, skills, or project files. Delete application-owned temporary resources after child cleanup. If cleanup cannot complete, including failure to remove a temporary directory, mark the result's cleanup as failed, print a bounded safe diagnostic, and exit 1 while keeping any measured percentage.
  - Architecture: [ARCH-05](../architecture/README.md#arch-05). Owner: [TASK-003](../roadmap/README.md#task-003). Re-qualified: [TASK-022](../roadmap/README.md#task-022). Verification: [V-SAFETY](../architecture/README.md#v-safety). Evidence: [TASK-003](../verification/TASK-003.md) — private 0700 temporary working directory outside the project, removed after cleanup; TASK-011 confirms invocation from neutral directories; [TASK-022](../verification/TASK-022.md) — cleanup failures reach the diagnostic and the exit code.

<a id="req-041"></a>
- [x] **REQ-041: No application service or data store.** Create no listener, remote service, database, application-owned permanent runtime config, permanent cache, credentials store, or application telemetry. Read the required user-authored account configuration without modifying it. A provider-owned leader is not a Gaebokchi daemon and must not be controlled as one.
  - Architecture: [ARCH-02](../architecture/README.md#arch-02). Owner: [TASK-011](../roadmap/README.md#task-011). Verification: [V-SAFETY](../architecture/README.md#v-safety). Evidence: [TASK-011](../verification/TASK-011.md) static scan (no listeners or application-owned persistent files) and TASK-003 temporary-directory cleanup.
  - Changed 2026-09-29: the user-authored account configuration is the one permitted persistent input; gaebokchi does not create or modify it.

### Verification and delivery

<a id="req-042"></a>
- [x] **REQ-042: Version-qualified external contracts.** Record actual CLI executable and version evidence, protocol forms, source references, and sanitized observations. Never infer an installed version from a repository SHA or claim an untested version is supported. Detect contract mismatch safely without auto-upgrading a provider CLI.
  - Architecture: [ARCH-12](../architecture/README.md#arch-12). Owner: [TASK-001](../roadmap/README.md#task-001). Verification: [V-CONTRACT](../architecture/README.md#v-contract). Evidence: [compatibility record](../verification/compatibility.md) — codex-cli 0.155.1, grok 1.0.40, live two-home Codex reads, live weekly Grok billing, sanitized fixtures.

<a id="req-043"></a>
- [x] **REQ-043: Offline default tests.** Run normal tests entirely against fixtures and fake child executables, without live accounts, provider installations, credentials, or network access. Keep live probes explicit and opt-in.
  - Architecture: [ARCH-11](../architecture/README.md#arch-11). Owner: [TASK-004](../roadmap/README.md#task-004). Verification: [V-RPC](../architecture/README.md#v-rpc). Evidence: [TASK-004](../verification/TASK-004.md) — the entire suite runs against in-memory pipes and the compiled fake-child harness; the offline suite invokes no live CLI.

<a id="req-044"></a>
- [x] **REQ-044: Parser and edge-case coverage.** Cover the quota, authentication, dialect, account-isolation, malformed-input, and semantic edge cases listed in ARCH-11. Malformed-input coverage includes `error: null`, mistyped error members, incompatible version markers, and label-normalization collisions. Unknown additive fields must not break an otherwise valid response.
  - Architecture: [ARCH-11](../architecture/README.md#arch-11). Owner: [TASK-011](../roadmap/README.md#task-011). Re-qualified: [TASK-018](../roadmap/README.md#task-018), [TASK-019](../roadmap/README.md#task-019), [TASK-021](../roadmap/README.md#task-021). Verification: [V-QUOTA](../architecture/README.md#v-quota). Evidence: [TASK-006](../verification/TASK-006.md) (Codex matrix), TASK-008 (Grok matrix), TASK-004 (framing and dialects), with unknown additive fields ignored throughout; [TASK-018](../verification/TASK-018.md) and [TASK-019](../verification/TASK-019.md) — malformed-input and label cases; [TASK-021](../verification/TASK-021.md) — final offline gate on darwin/arm64 and linux/arm64.

<a id="req-045"></a>
- [x] **REQ-045: End-to-end failure isolation.** Verify stdout, stderr, exit status, row count and order, callback rejection, timeout, signal handling, and cleanup through the actual CLI executable with scripted fake providers. Use real OS pipes to cover a broken stdout pipe, each signal with unwritable stdout, and mixed completion under cancellation.
  - Architecture: [ARCH-11](../architecture/README.md#arch-11). Owner: [TASK-011](../roadmap/README.md#task-011). Re-qualified: [TASK-017](../roadmap/README.md#task-017). Verification: [V-E2E](../architecture/README.md#v-e2e). Evidence: [TASK-011](../verification/TASK-011.md) — executable-level suite with fake providers covering stdout, stderr, exit status, rows, callbacks, timeouts, signals, and cleanup; [TASK-017](../verification/TASK-017.md) — real-pipe output failures and signals.

<a id="req-046"></a>
- [x] **REQ-046: Build artifacts.** Produce binaries for darwin/arm64, darwin/amd64, linux/arm64, and linux/amd64, with version metadata and checksums. Do not bundle third-party provider binaries or automatically publish a release.
  - Architecture: [ARCH-13](../architecture/README.md#arch-13). Owner: [TASK-012](../roadmap/README.md#task-012). Verification: [V-BUILD](../architecture/README.md#v-build). Evidence: [TASK-012](../verification/TASK-012.md) — four target archives with SHA-256 checksums, specification documents included, no provider binaries, no publication.

<a id="req-047"></a>
- [x] **REQ-047: Native platform evidence.** Run the application and fake-provider integration tests natively on macOS and Linux. Record which CPU architectures were executed and which were only cross-built; a successful cross-build alone is not runtime validation.
  - Architecture: [ARCH-13](../architecture/README.md#arch-13). Owner: [TASK-012](../roadmap/README.md#task-012). Verification: [V-PLATFORM](../architecture/README.md#v-platform). Evidence: [TASK-012](../verification/TASK-012.md) — native darwin/arm64 and linux/arm64 (containerized) full suites; darwin/amd64 and linux/amd64 labelled cross-build-only. [TASK-021](../verification/TASK-021.md) re-ran natively on the corrected tree: darwin/arm64 full suite and race tests; linux/arm64 (golang:1.26.6-bookworm container) vet, full suite, race subset, and docs check.
  - Note: EPIC-008 through EPIC-010 and the initial Claude extension recorded native runs on darwin/arm64 only. [TASK-031](../verification/TASK-031.md) and [TASK-032](../verification/TASK-032.md) later ran full gates on darwin/arm64 and linux/arm64. The final release commit still needs its own native gates under [TASK-033](../roadmap/README.md#task-033).

<a id="req-048"></a>
- [x] **REQ-048: Owner-account live acceptance.** Run the v0.1.0 candidate binary on at least one supported host against the owner's two distinct Codex homes, existing Grok login, and existing Claude login. Require four ordered `ok` rows with provider-reported weekly values, no cleanup failure, and no stderr diagnostic. Do not generate a model turn. Record the binary's source revision and build state, installed provider CLI versions, command outcome, and platform without retaining account identifiers or raw provider responses.
  - Architecture: [ARCH-12](../architecture/README.md#arch-12). Owner: [TASK-013](../roadmap/README.md#task-013). Re-qualified: [TASK-033](../roadmap/README.md#task-033). Verification: [V-LIVE](../architecture/README.md#v-live). Evidence: [historical live acceptance](../verification/live-acceptance.md) and [current candidate acceptance](../verification/TASK-033.md).
  - Changed 2026-09-29 (owner decision): a comparison with each provider's own usage display is no longer a v0.1.0 release condition. The owner supplied a four-row live application snapshot, and a separate prompt-free run of the corrected staged binary returned four `ok` weekly rows, qualifying this requirement. That staged binary has `vcs.modified=true`; the exact clean release binary must repeat the run under the separate [current release gate](../roadmap/README.md#current-release-gate).

<a id="req-049"></a>
- [x] **REQ-049: Usage and troubleshooting guide.** Ship a README covering install and build, exact flags, the existing-login prerequisite, quota meaning, statuses, exit codes, supported and tested versions, platform evidence, and known limitations. Explain that a Grok alias is not a verified email and a monthly result is not weekly usage. Distinguish live provider qualification from native offline test runs and from cross-build-only targets.
  - Architecture: [ARCH-13](../architecture/README.md#arch-13). Owner: [TASK-012](../roadmap/README.md#task-012). Re-qualified: [TASK-020](../roadmap/README.md#task-020). Verification: [V-DOCS](../architecture/README.md#v-docs). Evidence: the root README, smoke-checked against the binary; [TASK-020](../verification/TASK-020.md) — live qualification separated from offline Linux evidence.
  - Changed 2026-09-28: the README was rewritten for the GitHub release with the same required topics.

<a id="req-050"></a>
- [x] **REQ-050: Traceable completion.** Maintain requirement-to-architecture-to-task links and record verification evidence before checking off a requirement. Keep roadmap rows, epic and task statuses, requirement checkboxes, and verification records consistent, and verify that packaged documents are fresh before staging a release candidate. Complete epics in plan order, one task-scoped commit at a time. Do not close a provider requirement on fallback-only evidence or run unbounded review cycles.
  - Architecture: [ARCH-14](../architecture/README.md#arch-14). Owner: [TASK-014](../roadmap/README.md#task-014). Re-qualified: [TASK-021](../roadmap/README.md#task-021), [TASK-024](../roadmap/README.md#task-024). Verification: [V-AUDIT](../architecture/README.md#v-audit). Evidence: the TASK-014 audit; [TASK-021](../verification/TASK-021.md) and [TASK-024](../verification/TASK-024.md) — reconciled acceptance state; [TASK-025](../verification/TASK-025.md) — packaged-document freshness before staging.

<a id="output-modes"></a>
### Output modes

The usage command presents one ordered result set in three ways. `--json` selects the JSON snapshot ([REQ-052](#req-052)). Otherwise an interactive terminal gets the live view ([REQ-053](#req-053) through [REQ-055](#req-055)), and every other stdout gets the plain table ([REQ-011](#req-011)). [REQ-012](#req-012) still requires label sanitization and stdout/stderr separation in every mode, and [REQ-015](#req-015) still requires in-memory window provenance.

<a id="current-terminal-view-extension"></a>
The synthetic frame below shows the terminal view while the last query is still running; [REQ-054](#req-054) defines its colors:

```text

  🐡 gaebokchi · weekly usage

  ChatGPT  work                  ███████░░░░░░░░░░░░░     37%
    ↻ 2d 4h
           personal              ██░░░░░░░░░░░░░░░░░░     12%
    ↻ 5d 1h
  Grok     default               ███████████░░░░░░░░░     54%
    ↻ 3d 11h
  Claude   default               ⠹ querying…

```

On a wider terminal, all qualified reset countdowns appear inline when every reset row fits with a full 20-cell bar.

<a id="req-052"></a>
- [x] **REQ-052: Machine-readable usage snapshot.** Accept `--json` on the usage command and emit one newline-terminated JSON object with a `results` array in target order. Each result contains exactly `target_id`, `provider`, normalized `account`, `status`, `used_percent` (a number for a measured weekly percentage or the [REQ-034](#req-034) Grok weekly omission fallback, otherwise `null`), and `cleanup_failed`. Preserve reported numeric precision and zero, partial results, stderr diagnostics, and existing exit codes. The Grok fallback is numeric zero in JSON, although it is not a measurement. Non-interactive usage follows the [REQ-011](#req-011) table structure and [REQ-013](#req-013) percentage formatting; text help and `version --json` remain unchanged. Usage errors print no stdout snapshot. Exclude raw provider data and internal metadata.
  - Architecture: [ARCH-03](../architecture/README.md#arch-03), [ARCH-10](../architecture/README.md#arch-10). Owner: [TASK-029](../roadmap/README.md#task-029). Verification: [V-OPTIONS](../architecture/README.md#v-options), [V-OUTPUT](../architecture/README.md#v-output), [V-EXIT](../architecture/README.md#v-exit). Evidence: [TASK-029](../verification/TASK-029.md) records offline checks and the task-scoped commit subject.
  - Changed 2026-09-27: the [Grok weekly omission compatibility record](../verification/compatibility.md#grok-1041-omitted-weekly-percentage) establishes the numeric-zero exception to TASK-029's measured-only JSON contract.

<a id="req-053"></a>
- [x] **REQ-053: Terminal view selection.** When `--json` is present, emit only the [REQ-052](#req-052) JSON snapshot on stdout, regardless of terminal state. Without `--json`, use the interactive terminal view only when stdout is a terminal whose window size can be read and `TERM` is neither empty nor `dumb`. In every other non-JSON case, including pipes, regular files, and `/dev/null`, stdout is the [REQ-011](#req-011) table with [REQ-013](#req-013) percentage formatting. A non-empty `NO_COLOR` suppresses color while keeping the layout, glyphs, bold, and faint text. Help flags, structure, and exit behavior, `version`, stderr diagnostics, and exit codes do not change, and the view adds no flag. Help usage wording describes the interactive terminal view and the non-interactive table.
  - Architecture: [ARCH-10](../architecture/README.md#arch-10-terminal). Owner: [TASK-030](../roadmap/README.md#task-030). Verification: [V-OUTPUT](../architecture/README.md#v-output), [V-E2E](../architecture/README.md#v-e2e). Evidence: [TASK-030](../verification/TASK-030.md).
  - Changed 2026-09-26: help retains its command and flag contract while its usage description names the terminal view. Help still performs no provider query.

<a id="req-054"></a>
- [x] **REQ-054: Terminal usage frame.** Render a title line and then one target row per target in target order. Each target row shows the provider display name, left blank when it repeats the previous row's provider; the normalized account label, disambiguated as in the table once the final frame is drawn; and either a usage bar of whole `█` and `░` cells followed by the [REQ-013](#req-013) percentage, or the existing status cell text. Round the bar to the nearest whole cell; the percentage retains its precision. Color numeric rows green below 50%, yellow from 50% to below 80%, and red at 80% and above. Show non-applicable and cancelled rows faint, `AUTH REQUIRED` and unavailable rows yellow, and every other failure red. Show a reset countdown only on numeric rows whose provider supplied reset metadata, in whole truncated units (`2d 4h`, `4h 12m`, `12m`, `<1m`, or `now`), and never derive a reset time. Put every qualified countdown inline only when every such row fits with a full 20-cell bar; otherwise put each on a four-space-indented line directly below its account. Fit numeric rows within the terminal width by shrinking the bar from 20 to no fewer than 10 cells, then dropping the bar if it still does not fit; never truncate an account label in the final frame. Count East Asian wide characters as two cells, keep the [REQ-012](#req-012) label sanitization, and emit only the control sequences that the architecture allows.
  - Architecture: [ARCH-10](../architecture/README.md#arch-10-terminal). Owner: [TASK-030](../roadmap/README.md#task-030). Verification: [V-OUTPUT](../architecture/README.md#v-output). Evidence: [TASK-030](../verification/TASK-030.md).
  - Changed 2026-09-27: whole-cell bars and width-dependent reset placement supersede the original eighth-cell, inline-reset layout recorded in TASK-030. The current rendering and live-redraw tests verify this layout.
  - Changed 2026-09-27: Claude rows qualify for a countdown through the reset text on the all-models weekly line ([REQ-059](#req-059)), as recorded in the [Claude compatibility record](../verification/claude-compatibility.md). Provider and rendering tests verify it.

<a id="req-055"></a>
- [x] **REQ-055: Live query progress.** While queries run, show every target row in place: unfinished rows with a spinner and `querying…`, and finished rows as soon as their results are known. Redraw without scrolling and keep intermediate frames within the terminal width. When the frame is not strictly shorter than the terminal height (leaving one cursor row), write only the final frame, once. After completion or SIGINT/SIGTERM cancellation, leave the final frame in the scrollback and restore cursor visibility. After an output failure, stop drawing and apply the existing output-failure exit policy. Progress rendering must not change target admission, deadlines, or query attempts.
  - Architecture: [ARCH-05](../architecture/README.md#arch-05), [ARCH-10](../architecture/README.md#arch-10-terminal). Owner: [TASK-030](../roadmap/README.md#task-030). Verification: [V-RUNNER](../architecture/README.md#v-runner), [V-OUTPUT](../architecture/README.md#v-output), [V-EXIT](../architecture/README.md#v-exit). Evidence: [TASK-030](../verification/TASK-030.md).

<a id="planned-provider-extensions"></a>
## Planned provider extensions

These requirements were adopted on 2026-10-15 and revised on 2026-10-17 after
live `agy 1.4.0` qualification and the owner's scope decision.
[EPIC-012](../roadmap/README.md#epic-012) owns `agy` single-account weekly
support (`gemini` and `claude/gpt` rows); [EPIC-013](../roadmap/README.md#epic-013)
owns Cursor.
[ADR-0002](../architecture-decision-records/ADR-0002-period-aware-provider-usage.md)
records the accepted design direction. Retain REQ-037 through REQ-043 throughout,
including no provider auto-upgrade and offline test isolation.

<a id="req-060"></a>
- [x] **REQ-060: Qualified agy metadata interface.** Establish a real installed-CLI path (`agy --print /usage --output-format json --print-timeout 15s`) for the single default `agy` login that returns structured weekly subscription quota metadata without model turns, agent tools, sessions, interactive auth, account switching or billing mutations. Enforce a pre-invocation runtime version guard (`agy --version` `>= 1.2.15`) within the target timeout budget and terminate the child immediately if `Authentication required` appears on stderr. Multi-account `agy` isolation is out of scope.
  - Architecture: [ARCH-18](../architecture/README.md#arch-18). Owner: [TASK-034](../roadmap/README.md#task-034). Verification: [V-AGY](../architecture/README.md#v-agy). Evidence: [agy compatibility](../verification/agy-compatibility.md).
  - Changed 2026-10-17: scoped to the single default CLI login after the owner confirmed single-account weekly usage; qualified live against `agy 1.4.0` on `darwin/arm64` ([preflight record](../verification/provider-extension-preflight-2026-10-16.md)).

<a id="req-061"></a>
- [x] **REQ-061: Weekly model-pool extraction and exact remaining-fraction conversion.** Extract `agy`'s two weekly model-group quotas (`id == "gemini-weekly"` with `window == "weekly"` and `id == "3p-weekly"` with `window == "weekly"`) from `command.data.groups[].buckets[]`, ignoring non-weekly (`"5h"`) entries. Never average pools or select their maximum as a single row. Convert a qualified remaining fraction $r \in [0,1]$ to `used_percent = 100 * (1 - r)` from the original JSON decimal token using bounded exact decimal arithmetic (at most 128 ASCII bytes, 64 coefficient digits, exponent in `[-128, 128]`) so exact integer percentages (such as `remaining = 0.9` -> `10%`) never expose binary floating-point truncation noise. Parse optional RFC 3339 UTC `reset_time` values into `ResetAt` and preserve sibling pool results when one pool is missing or malformed.
  - Architecture: [ARCH-17](../architecture/README.md#arch-17). Owner: [TASK-035](../roadmap/README.md#task-035). Verification: [V-QUOTA](../architecture/README.md#v-quota), [V-RUNNER](../architecture/README.md#v-runner). Evidence: [agy compatibility](../verification/agy-compatibility.md).
  - Changed 2026-10-17: focused on weekly model pools (`gemini-weekly` and `3p-weekly`) and exact remaining-fraction conversion while preserving existing Codex, Grok, and Claude weekly normalization.

<a id="req-062"></a>
- [x] **REQ-062: Two-row weekly presentation for agy.** Keep `gaebokchi` weekly-only in both the plain 3-column table (`AI PROVIDER`, `ACCOUNT`, `WEEKLY USAGE`) and the interactive terminal view (`🐡 gaebokchi · weekly usage`). Present the enabled default `agy` account as two ordered rows under provider `agy`: `gemini` (`agy:0`) followed by `claude/gpt` (`agy:1`), with the interactive terminal view blanking the repeated provider label on the second row and displaying each pool's weekly progress bar, percentage, and `↻` reset countdown. Preserve all existing statuses, exit codes, bounded stderr diagnostics, and signal/output-failure rules.
  - Architecture: [ARCH-17](../architecture/README.md#arch-17), [ARCH-10](../architecture/README.md#arch-10). Owner: [TASK-036](../roadmap/README.md#task-036). Verification: [V-OUTPUT](../architecture/README.md#v-output), [V-EXIT](../architecture/README.md#v-exit), [V-E2E](../architecture/README.md#v-e2e). Evidence: [agy compatibility](../verification/agy-compatibility.md).
  - Changed 2026-10-17: replaced multi-period (`WEEKLY` / `5 HOURS` / `MONTHLY`) section headers with two weekly rows (`gemini` and `claude/gpt`) inside the existing weekly-only table and terminal view.

<a id="req-063"></a>
- [x] **REQ-063: JSON weekly snapshot continuity for agy pools.** Keep the existing `--json` output shape (`{"results":[...]}`) without a breaking schema change. Include `agy:0` (`provider: "agy"`, `account: "gemini"`) and `agy:1` (`provider: "agy"`, `account: "claude/gpt"`) in configured display order with their weekly `status`, `used_percent`, and `cleanup_failed` fields.
  - Architecture: [ARCH-17](../architecture/README.md#arch-17), [ARCH-10](../architecture/README.md#arch-10). Owner: [TASK-036](../roadmap/README.md#task-036). Verification: [V-OUTPUT](../architecture/README.md#v-output), [V-E2E](../architecture/README.md#v-e2e). Evidence: [agy compatibility](../verification/agy-compatibility.md).
  - Changed 2026-10-17: retained the existing JSON v1 `results` array instead of introducing `schema_version: 2` multi-period objects.

<a id="req-064"></a>
- [x] **REQ-064: Single-account agy configuration and shared query execution.** Add an optional `agy` list in `~/.config/gaebokchi/config.yaml` accepting at most one entry (such as `- name: default`, with `path` optional and not used as a credential selector) and `--agy-bin` with default `agy`. Omission or `[]` disables `agy`. Execute at most one `agy` query per `gaebokchi` invocation to populate both `agy:0` (`gemini`) and `agy:1` (`claude/gpt`) rows within a single target timeout budget, using explicit provider dispatch in `Runner.querierFor` with no implicit Codex fallback. Isolate child environment variables (`AGY_CLI_DISABLE_AUTO_UPDATE=true`, stripping `ANTIGRAVITY_*`, `JETSKI_*`, `AGY_*`, and API-key/ADC overrides while keeping `HOME`) and satisfy the complete offline gate and final-executable positive live acceptance.
  - Architecture: [ARCH-18](../architecture/README.md#arch-18). Implementation: [TASK-037](../roadmap/README.md#task-037). Acceptance owner: [TASK-038](../roadmap/README.md#task-038). Verification: [V-AGY](../architecture/README.md#v-agy), [V-LIFECYCLE](../architecture/README.md#v-lifecycle), [V-SAFETY](../architecture/README.md#v-safety). Evidence: [agy compatibility](../verification/agy-compatibility.md).
  - Changed 2026-10-17: scoped to a single default-login `agy` entry producing two weekly pool rows (`gemini` and `claude/gpt`) from one shared CLI invocation.

<a id="req-065"></a>
- [ ] **REQ-065: Qualified Cursor metadata interface.** Establish a real installed-CLI path for subscription quotas using the existing default CLI login without model turns, agent tools, session creation, interactive auth or billing mutations. Freeze its actual invocation, response schema, auth errors, explicit periods, model/pool identities, reset semantics and process boundaries before adapter implementation. Identity JSON, an interactive pager or an internal RPC function alone cannot pass this gate.
  - Architecture: [ARCH-19](../architecture/README.md#arch-19). Owner: [TASK-039](../roadmap/README.md#task-039). Verification: [V-CURSOR](../architecture/README.md#v-cursor). Qualification gaps: [Cursor compatibility](../verification/cursor-compatibility.md).
  - Planned clarification 2026-10-16: apply the [shared qualification and runtime guards](../architecture/README.md#provider-extension-qualification), including the applicability check for version-bound period mappings. ACP transport support alone does not qualify a session-free quota method. Earlier discovery can inform this task without depending on agy's outcome.

<a id="req-066"></a>
- [ ] **REQ-066: Positive Cursor default-login support.** Add an optional single named Cursor target and `--cursor-bin` with default `cursor-agent`. Query its existing default CLI login and present its qualified weekly usage in the shared output and JSON snapshot. Omission disables the provider. Keep on-demand spend outside quota percentages, preserve missing values and sibling outcomes, and verify the complete offline gate and a positive prompt-free final-executable live query. Multi-account management is outside this requirement, and a configuration directory is not an authenticated account selector. Cursor is the query source, not necessarily a Grok-only meter: preserve qualified shared-pool scope and do not add source-specific views as independent allowances. Cross-provider account merging or automatic deduplication is outside scope.
  - Architecture: [ARCH-19](../architecture/README.md#arch-19). Implementation: [TASK-040](../roadmap/README.md#task-040). Acceptance owner: [TASK-041](../roadmap/README.md#task-041). Verification: [V-CURSOR](../architecture/README.md#v-cursor), [V-LIFECYCLE](../architecture/README.md#v-lifecycle), [V-SAFETY](../architecture/README.md#v-safety).

| Delivery | Required reconciliation |
|---|---|
| TASK-035 | Implement exact decimal `remaining_fraction` conversion (`100 * (1 - r)`) with token/exponent bounds and `agy` weekly pool normalization (`gemini-weekly` -> `gemini`, `3p-weekly` -> `claude/gpt`), keeping all existing Codex, Grok, and Claude weekly contracts intact. |
| TASK-036 | Verify two-row `agy` (`gemini` and `claude/gpt`) rendering in the existing 3-column plain table, interactive terminal view (with blanked second-row provider cell and `↻` reset countdowns), and `--json` `results` array. |
| TASK-037 / TASK-040 | Update REQ-002, REQ-006, REQ-007 and provider-specific sections; ARCH-02, ARCH-03, ARCH-09, ARCH-11 and the relevant new provider section. Remove the implicit Codex fallback in `Runner.querierFor` and update public guidance and test-stage mapping. |

REQ-060 through REQ-064 are verified and closed by EPIC-012 ([TASK-034](../roadmap/README.md#task-034)
through [TASK-038](../roadmap/README.md#task-038)). REQ-065 closes at TASK-039 and
TASK-041 owns REQ-066. Preserve completed task/evidence bodies and the
historical v0.1.0 release gate.

<a id="completion-rules"></a>
## Completion rules

A percentage is not required when the provider explicitly reports a monthly allowance or the account uses an out-of-scope API-key mode. Those are supported, truthful **non-applicable** outcomes. An unsupported method, missing weekly metadata, absent login, timeout, or missing executable is a supported **failure or unavailable presentation**, not evidence that the positive provider integration is complete.

REQ-005, REQ-048, and REQ-056 require live successful queries. A live valid monthly Grok billing response and `N/A (monthly)` may qualify REQ-005's provider behavior, while weekly parsing is additionally verified with source-backed fixtures. REQ-048's v0.1.0 owner-account gate separately requires a live weekly Grok row. The evidence must explicitly state when a live weekly Grok percentage was not exercised; it must not claim otherwise. A method-not-found response cannot satisfy either live gate.

Checklists are not a replacement for evidence. Check a requirement only when its complete acceptance scope is verified, the evidence is recorded in the [task evidence format](../roadmap/README.md#plan-evidence), and a roadmap task owns it ([REQ-050](#req-050)). Preserve failures and unrun checks. A dated **Changed** note records a contract change that took effect after the original evidence, together with the evidence that covers it.

For the planned extensions, REQ-060 and REQ-065 require actual CLI qualification;
REQ-064 and REQ-066 additionally require positive integrated-executable live
acceptance. Static inspection or a recorded blocker cannot close these gates.

## Requirement ownership index

This index is a navigation aid. The individual checkbox remains authoritative.

| Task | Requirements primarily closed by the task |
|---|---|
| [TASK-001](../roadmap/README.md#task-001) | [REQ-042](#req-042) |
| [TASK-002](../roadmap/README.md#task-002) | [REQ-007](#req-007), [REQ-008](#req-008) |
| [TASK-003](../roadmap/README.md#task-003) | [REQ-020](#req-020), [REQ-040](#req-040) |
| [TASK-004](../roadmap/README.md#task-004) | [REQ-016](#req-016), [REQ-017](#req-017), [REQ-018](#req-018), [REQ-019](#req-019), [REQ-043](#req-043) |
| [TASK-005](../roadmap/README.md#task-005) | [REQ-025](#req-025), [REQ-026](#req-026), [REQ-027](#req-027), [REQ-039](#req-039) |
| [TASK-006](../roadmap/README.md#task-006) | [REQ-003](#req-003), [REQ-028](#req-028), [REQ-029](#req-029), [REQ-030](#req-030) |
| [TASK-007](../roadmap/README.md#task-007) | [REQ-031](#req-031), [REQ-032](#req-032), [REQ-033](#req-033) |
| [TASK-008](../roadmap/README.md#task-008) | [REQ-013](#req-013), [REQ-014](#req-014), [REQ-015](#req-015), [REQ-034](#req-034), [REQ-035](#req-035), [REQ-036](#req-036) |
| [TASK-009](../roadmap/README.md#task-009) | [REQ-004](#req-004), [REQ-009](#req-009), [REQ-022](#req-022), [REQ-023](#req-023) |
| [TASK-010](../roadmap/README.md#task-010) | [REQ-002](#req-002), [REQ-006](#req-006), [REQ-010](#req-010), [REQ-011](#req-011), [REQ-012](#req-012), [REQ-024](#req-024) |
| [TASK-011](../roadmap/README.md#task-011) | [REQ-021](#req-021), [REQ-037](#req-037), [REQ-038](#req-038), [REQ-041](#req-041), [REQ-044](#req-044), [REQ-045](#req-045) |
| [TASK-012](../roadmap/README.md#task-012) | [REQ-001](#req-001), [REQ-046](#req-046), [REQ-047](#req-047), [REQ-049](#req-049) |
| [TASK-013](../roadmap/README.md#task-013) | [REQ-005](#req-005), [REQ-048](#req-048) |
| [TASK-014](../roadmap/README.md#task-014) | [REQ-050](#req-050) |
| [TASK-015](../roadmap/README.md#task-015) | Acceptance-baseline correction only; reopens the checkboxes listed below, closes none |
| [TASK-016](../roadmap/README.md#task-016) | Re-qualifies [REQ-009](#req-009), [REQ-020](#req-020), [REQ-023](#req-023) |
| [TASK-017](../roadmap/README.md#task-017) | Re-qualifies [REQ-006](#req-006) (binary overrides), [REQ-024](#req-024), [REQ-039](#req-039), [REQ-045](#req-045) |
| [TASK-018](../roadmap/README.md#task-018) | Re-qualifies [REQ-016](#req-016), [REQ-018](#req-018) |
| [TASK-019](../roadmap/README.md#task-019) | Re-qualifies [REQ-006](#req-006) (alias validation), [REQ-010](#req-010), [REQ-015](#req-015), [REQ-021](#req-021), [REQ-032](#req-032) |
| [TASK-020](../roadmap/README.md#task-020) | Re-qualifies [REQ-049](#req-049) |
| [TASK-021](../roadmap/README.md#task-021) | Re-qualifies [REQ-044](#req-044), [REQ-048](#req-048), [REQ-050](#req-050) |
| [TASK-022](../roadmap/README.md#task-022) | Re-qualifies [REQ-020](#req-020), [REQ-040](#req-040) |
| [TASK-023](../roadmap/README.md#task-023) | Re-qualifies [REQ-006](#req-006), [REQ-010](#req-010), [REQ-012](#req-012) |
| [TASK-024](../roadmap/README.md#task-024) | Re-qualifies [REQ-050](#req-050) |
| [TASK-025](../roadmap/README.md#task-025) | Final gate and candidate packaging evidence |
| [TASK-028](../roadmap/README.md#task-028) | [REQ-051](#req-051) |
| [TASK-029](../roadmap/README.md#task-029) | [REQ-052](#req-052) |
| [TASK-030](../roadmap/README.md#task-030) | [REQ-053](#req-053), [REQ-054](#req-054), [REQ-055](#req-055) |
| [TASK-032](../roadmap/README.md#task-032) | Re-qualifies [REQ-002](#req-002), [REQ-006](#req-006) through [REQ-008](#req-008), [REQ-010](#req-010) through [REQ-011](#req-011), [REQ-039](#req-039), and [REQ-056](#req-056) through [REQ-059](#req-059) for the current configuration and Claude contracts |
| [TASK-033](../roadmap/README.md#task-033) | Re-qualifies [REQ-048](#req-048) for the final v0.1.0 candidate |
| [TASK-034](../roadmap/README.md#task-034) | [REQ-060](#req-060) |
| [TASK-035](../roadmap/README.md#task-035) | [REQ-061](#req-061) |
| [TASK-036](../roadmap/README.md#task-036) | [REQ-062](#req-062), [REQ-063](#req-063) |
| [TASK-038](../roadmap/README.md#task-038) | [REQ-064](#req-064); TASK-037 supplies implementation coverage |
| [TASK-039](../roadmap/README.md#task-039) | Planned: [REQ-065](#req-065) |
| [TASK-041](../roadmap/README.md#task-041) | Planned: [REQ-066](#req-066); TASK-040 supplies implementation coverage |

Component tasks also contribute to cross-cutting requirements owned by integration or release tasks. See the dependency and verification mappings in [the roadmap](../roadmap/README.md#plan-roadmap).

<a id="acceptance-history"></a>
## Acceptance history

- The original campaign checked 50 requirements, and the TASK-014 release audit rechecked them against the delivered tree at commit `f8b180d`.
- On 2026-09-22 an adversarial audit of executable-level behavior superseded that result. TASK-015 unchecked 17 requirements, and [EPIC-007](../roadmap/README.md#epic-007) fixed the defects; TASK-021 re-verified each full acceptance scope against the corrected tree and restored 16 of them. A confirmation review of commit `c340245` then reopened further requirements, which TASK-022 through TASK-024 re-qualified. The roadmap's EPIC-007 section and the verification records [TASK-015](../verification/TASK-015.md) through [TASK-024](../verification/TASK-024.md) keep each finding. The requirement text above states the corrected rules, and each trace line names its re-qualifying tasks.
- REQ-048 stayed open after the earlier campaign because the original contract required provider-display comparison; its historical records retain that reason.
- Later extensions changed the contract without reopening the original v0.1 evidence: the version command (EPIC-008, which added REQ-051), JSON output (EPIC-009), the interactive terminal view (EPIC-010), and Claude support. EPIC-011 owns their public-release reconciliation. Dated Changed notes mark every affected requirement.
- On 2026-09-29 TASK-032 re-qualified the YAML-era account and Claude requirements against the current product code, focused offline checks, and a prompt-free positive live run. Distinct Grok and Claude multi-login behavior remains a documented compatibility limit. Later that day the owner removed provider-display comparison from REQ-048's v0.1.0 acceptance; the current release gate owns the final clean-binary live run after TASK-033's candidate acceptance.
- The roadmap keeps the original v0.1 [historical gate](../roadmap/README.md#plan-release-gate) and a separate [current release gate](../roadmap/README.md#current-release-gate). Claude support and YAML configuration are current behavior; a PTY remains out of scope.
