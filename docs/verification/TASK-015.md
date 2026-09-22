# TASK-015: Correct the acceptance-state baseline and register this remediation

Task: TASK-015
Status: COMPLETE
Goal and implementation scope: Make the recorded acceptance state truthful before code changes: supersede the COMPLETE claims in the root documents, uncheck exactly the requirements whose full acceptance scope is affected by verified audit findings, and register EPIC-007 (TASK-015 through TASK-021) with the next available IDs.
Requirements and architecture sections covered: REQ-050 (traceability correction; re-qualification deferred to TASK-021); ARCH-14.
Changed files: docs/requirement.md (status header; 17 checkboxes unchecked with supersession notes; ownership index extended), docs/plan.md (status header; roadmap reconciliation and EPIC-007 row; supersession notes on TASK-013/TASK-014; EPIC-007 section with TASK-015..TASK-021; final release gate re-marked), docs/verification/TASK-015.md (this record).
Requirements unchecked, with the verified defect behind each (all confirmed against commit `f8b180d` by direct code inspection before unchecking):
  - REQ-006 — explicit relative binary overrides fail from the neutral child cwd (`internal/provider/*/provider.go` ResolveBin performs no absolutization; `internal/process/child.go` sets a private temp cwd); whitespace-only `--grok-account` accepted (`internal/options/options.go` `setScalar` compares against `""` only).
  - REQ-009 — `internal/app/runner.go` Run converts an `ExitClean` result to TIMEOUT when `qctx.Err()` is `DeadlineExceeded` after `Query` returns, but provider `Query` performs deferred cleanup inside the same budget (worst case ~2x CleanupBudget because `finish()` grants a fresh 2s after `shutdown()`).
  - REQ-010 — `isUsableEmail` accepts whitespace-only values (only control runes <0x20/0x7f rejected); `DisambiguateCodexAccounts` counts raw labels while `render.SanitizeLabel` normalizes later, so distinct raw labels can collapse to one visible label.
  - REQ-015 — Grok monthly branch (`internal/provider/grok/normalize.go`) returns `billingOutcome{Status}` without `PeriodMonthly`.
  - REQ-016/REQ-018 — `internal/rpc/client.go` `deliverResponse` accepts `error:null` with no `result` as success; Codex initialize and Grok authenticate results are never shape-validated; inbound jsonrpc version markers unchecked on both dialects.
  - REQ-020 — double cleanup budget (shutdown + fresh finish); direct-child exit short-circuits descendant handling.
  - REQ-021 — every `*rpcCallError` from Grok `authenticate` maps to AUTH REQUIRED "cached token rejected", including internal errors.
  - REQ-023 — a blocked context-less `Notify` write cannot be unblocked by cancellation, defeating signal cleanup.
  - REQ-024 — no SIGPIPE policy (closed stdout kills the process); output-failure `return 1` precedes the signal exit-code check in `cmd/gaebokchi/main.go`.
  - REQ-032 — `{api_key, grok.com}` without `cached_token` classified `N/A (API key)`; the interactive path is invisible when api_key is advertised.
  - REQ-039 — relative executable identity not resolved against the invocation environment before the child cwd changes.
  - REQ-044 — malformed-input cases (error:null, mistyped members, version markers, normalization collisions) uncovered and actually accepted.
  - REQ-045 — output-failure coverage used in-process `io.Writer` stubs; real broken-pipe/signal combinations untested.
  - REQ-048 — provider-display comparison never performed; evidence recorded the omission as a limitation while the box stayed checked.
  - REQ-049 — README conflates Linux fake-provider offline testing with live provider qualification.
  - REQ-050 — the prior "all 50 checked with valid evidence" claim was itself not evidence-backed.
Requirements left checked: 33 requirements not affected by any verified finding keep their original evidence and links (including REQ-005 and REQ-047, whose recorded live/native evidence remains valid for the artifact tested at that time).
Verification (environment: macOS 26.6.2 arm64):
  - V-AUDIT: `go run scripts/check-docs.go` — links, anchors, and the new EPIC-007/TASK-015..021 identifiers resolve; run after all edits in this task.
  - Baseline preserved: `git status` shows only docs changes in this commit; `.omc/` and `.zcode/` untracked directories untouched; no code changed in this task.
Fixtures: none touched.
Findings and bounded remediation: the audit's claim of a possible reentrant-close deadlock was refuted during verification (`die` is `sync.Once`-guarded; repeated `Close` is a safe no-op), so no requirement was reopened on that basis; the related `sendRejection` writer-leak concern is real and is tracked under TASK-016. Unknown/duplicate response-ID handling was verified already-safe (pinning tests only, TASK-018).
Compatibility/platform limitations: none introduced; documentation-only change.
Task commit subject: `docs(TASK-015): correct acceptance baseline and register remediation`
Handoff: EPIC-007 registered; proceed to TASK-016 (query deadlines, transport cancellation, owned-process cleanup) with no code changes made yet.
