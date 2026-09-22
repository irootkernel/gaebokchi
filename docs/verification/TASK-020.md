# TASK-020: Fix test hygiene, formatting, and documentation inconsistencies

> Follow-up note (2026-09-22, TASK-024): the confirmation review found later state drift (roadmap row, acceptance checkboxes, and packaged documents inconsistent at `c340245`) despite this task's own scope being correct at its tree state; reconciled by TASK-024 with packaging freshness re-verified in TASK-025.

Task: TASK-020
Status: COMPLETE
Goal and implementation scope: Test-owned cleanup removes only directories the current invocation created; formatting is applied and enforced by the verification workflow rather than assumed; documentation states exactly what was verified, distinguishing the four evidence kinds.
Requirements and architecture sections covered: REQ-049 (re-qualification); ARCH-11, ARCH-13, ARCH-14.
Changed files: test/integration/main_test.go (owned-directory registry replaces the prefix glob; cleanupOwnedDirs removes only registered directories), test/integration/e2e_test.go (the build directory registers itself), test/integration/cleanup_test.go (new regression), .github/workflows/ci.yml (gofmt gate added ahead of the test steps), docs/verification/compatibility.md (fixture filenames corrected to the actual files: ratelimits_weekly_home1/home2.json, account_chatgpt_home1/home2.json), README.md (live qualification is darwin/arm64 only; the Linux/arm64 run is explicitly offline fake-provider evidence), docs/plan.md (TASK-016 through TASK-019 marked complete with their evidence records), plus `gofmt -w` applied to 16 Go files (formatting-only diff: 54 lines changed, 54 lines changed, verified with `git diff --stat`).
Verification (environment: macOS 26.6.2 arm64, go1.26.6):
  - V-E2E: `go test -count=1 ./test/integration` — pass, including the new TestCleanupRemovesOnlyOwnedDirs (an owned dir is removed; a similarly named foreign dir survives, including across a second drained-cleanup call).
  - Formatting: `gofmt -l internal cmd test scripts` — empty. The CI workflow now fails on formatting drift (gofmt gate step added before the test steps), so formatting is checked rather than assumed.
  - V-DOCS: `go run scripts/check-docs.go` — 28 markdown files, links and identifiers OK, including the corrected fixture references.
  - Documentation consistency: README now distinguishes (1) live provider qualification on darwin/arm64, (2) native offline suites on darwin/arm64 and linux/arm64, (3) cross-build-only targets, and (4) the provider-display comparison gap (owned by TASK-021); compatibility.md fixture references match the repository's actual testdata files; plan.md task statuses match the completed corrective tasks.
  - Full regression: `go vet ./...` pass; `go test -count=1 ./...` all 12 packages pass.
Fixtures: unchanged in content; only their documented filenames were corrected.
Findings and bounded remediation: none outstanding in this scope.
Compatibility/platform limitations: none introduced.
Task commit subject: `test(TASK-020): fix test hygiene formatting and documentation consistency`
Handoff: proceed to TASK-021 (final qualification and the truthful release decision).
