# Deferred Feedback

This directory owns small actionable findings intentionally postponed from
current work. It is not a roadmap or release-status authority. The open REQ-048
live acceptance gate stays in the specifications and roadmap because it is a
current release condition, not deferred feedback.

## 2026-09-29: Qualify provider home selectors with distinct live logins

- **Context:** The configuration extension sets `GROK_HOME` and
  `CLAUDE_CONFIG_DIR` per target. The offline fake-provider gate verifies the
  selected child environments and output rows, but it cannot establish that
  each installed CLI reads a distinct real subscription login from those
  directories on every supported platform and version.
- **Proposed action:** with separately authorized real-account probes, verify
  two pre-authenticated directories for Grok and Claude, record CLI versions
  and masked account/usage evidence, and update compatibility qualification.
  Do not read or copy credentials in gaebokchi.
- **Affected authorities:** compatibility records, ARCH-12, REQ-005, and
  REQ-056 through REQ-057.

## 2026-09-28: The Claude extension has no owning roadmap task

- **Context:** Claude support (REQ-056 through REQ-059, ARCH-16, and
  [ADR-0001](../architecture-decision-records/ADR-0001-claude-usage-local-command.md))
  was implemented outside the roadmap. The completion rules require an owning
  task before a requirement is checked, so the four requirements stay unchecked
  despite live and offline evidence.
- **Proposed action:** have a work-definition workflow adopt an acceptance task
  for the Claude extension that runs the V-CLAUDE coverage, records the
  evidence, and checks REQ-056 through REQ-059.
- **Affected authorities:** specifications, roadmap, verification records.

## 2026-09-28: The module path prevents `go install`

- **Context:** the Go module path is the local-only `gaebokchi` (ARCH-01), so
  `go install github.com/irootkernel/gaebokchi/cmd/gaebokchi@<version>` cannot
  work. Users install from release archives or build from a clone.
- **Proposed action:** if `go install` support is wanted, change the module path
  to `github.com/irootkernel/gaebokchi` and update every import in one change.
- **Affected authorities:** `go.mod`, the Go imports, ARCH-01, and the README
  install section.

## 2026-09-28: Diagnostic and comment polish

- **Context:**
  - Each stderr diagnostic repeats the target ID, because provider messages
    already begin with it (`gaebokchi: codex:1: AUTH REQUIRED: codex:1: …`).
  - A Claude cleanup failure sets no message, so its diagnostic repeats the
    percentage cell instead of naming the cleanup failure.
  - Some code comments predate the Claude provider or the current duplicate
    handling: `internal/app/runner.go` (the `Querier` comment),
    `internal/domain/result.go` (`AccountSource`), `internal/domain/target.go`
    (`Home`), and `internal/provider/envfilter/envfilter.go` (`Apply`).
- **Proposed action:** drop the duplicated prefix, give a Claude cleanup
  failure a fixed message, refresh the comments, and update the ARCH-10 stderr
  example.
- **Affected authorities:** ARCH-10, `internal/render`,
  `internal/provider/claude`, and the files listed above.
