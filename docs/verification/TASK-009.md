# TASK-009: Integrate bounded snapshot coordination

Task: TASK-009
Status: COMPLETE
Goal and implementation scope: internal/app runner composing both providers behind the one-target Querier boundary: fixed result slots in configured order, at most three active targets, per-target deadline starting at admission, independent target failures, queued-target cancellation without launch, and post-collection Codex label disambiguation.
Requirements and architecture sections covered: REQ-004, REQ-009, REQ-022, REQ-023 (primary); ARCH-03, ARCH-04, ARCH-05.
Changed files: internal/app/runner.go, internal/app/runner_test.go, docs/verification/TASK-009.md.
Verification (environment: macOS 26.6.2 arm64, go1.26.6):
  - V-RUNNER: `go test ./internal/app` — default three targets admitted simultaneously (wall time ~ one query, observed max concurrency exactly 3); deterministic order under deliberately reversed completion; five custom homes + grok honor the ≤3 bound with exactly one query attempt per target; a hanging home times out while both siblings keep valid results; mid-run cancellation leaves the three active queries cancelled and marks the three queued targets CANCELLED without ever launching them (including the slot-freed-after-cancel race, fixed by re-checking ctx after admission); deadline measured from admission (sequential bound-1 run with short budget succeeds); options-derived default targets feed the runner in codex:0, codex:1, grok:0 order.
  - Label disambiguation: duplicate emails across different homes become `<email> [home-basename]` and, when that is still ambiguous (duplicate basename labels), `#N`; unique labels untouched; rows never merged or hidden.
  - `go vet ./...`, `go test ./...`, `go test -race ./internal/app -count=3` — pass.
Fixtures: none (injected fake Queriers with counters and controlled delays).
Findings and bounded remediation: admission/cancellation race identified by the queued-cancellation test and fixed in the runner (post-acquisition ctx re-check); a late clean write after the per-target deadline is discarded as TIMEOUT.
Compatibility/platform limitations: none new.
Task commit subject: `feat(TASK-009): coordinate bounded account snapshots`
Handoff to next task: TASK-010 wires options, signals, providers, runner, and renderer into the final CLI table and exit contract.
