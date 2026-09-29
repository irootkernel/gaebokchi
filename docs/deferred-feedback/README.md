# Deferred Feedback

This directory owns small actionable findings intentionally postponed from
current work. It is not a roadmap or release-status authority. REQ-048's
candidate-binary live run remains a release condition in the specifications and
roadmap; the provider-display comparison below is separate.

## 2026-09-29: Compare provider-owned usage displays

- **Context:** The owner removed provider-display comparison from the v0.1.0
  release gate after reviewing a four-row live Gaebokchi snapshot. The
  configured candidate also returned four weekly `ok` rows. The available
  Grok browser session did not represent the configured CLI account, and only
  one Codex row could be compared with the available ChatGPT display.
- **Proposed action:** when matching read-only displays are available, compare
  both Codex homes and the Grok login within their quota periods without a
  model turn. Keep this separate from the v0.1.0 acceptance decision.
- **Affected authorities:** compatibility evidence and ARCH-12.

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
