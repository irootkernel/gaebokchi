# TASK-033: Qualify the final release candidate

Task: TASK-033
Status: COMPLETE
Candidate baseline: clean commit `3716d0e8cf099896939bd0f63a539e9edecdac65` on 2026-09-29. The process fix and documentation updates described below are later uncommitted changes; the final release commit does not yet exist.

## Baseline checks

- `make test` on native darwin/arm64 with Go 1.27.1: exit 0. Vet, golangci-lint v2.13.2 (zero issues), build, documentation check, race-enabled unit and integration suites, and E2E 32/32 passed. The worktree stayed clean.
- `docker run --rm --platform linux/arm64 -v "$PWD":/src -v "$(go env GOMODCACHE)":/go/pkg/mod -w /src golang:1.27.1-bookworm sh -c 'go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 && make test'`: exit 0 on native linux/arm64. The same stages passed, including E2E 32/32. The gate itself used the repository's offline Go module settings. darwin/amd64 and linux/amd64 were cross-built, not runtime-tested.
- `sh scripts/build-release.sh v0.1.0` from the clean baseline: exit 0. `GAEBOKCHI_REQUIRE_STAGED_RELEASE=1 go run github.com/onsi/ginkgo/v2/ginkgo --fail-on-empty --label-filter=staged-release ./scripts`: exit 0. Four archives contained the executable, current packaged documents, LICENSE, and THIRD_PARTY_NOTICES.txt with the expected permissions and valid `SHA256SUMS`. Each extracted binary reported `vcs.revision=3716d0e8cf099896939bd0f63a539e9edecdac65` and `vcs.modified=false`; the packaged darwin/arm64 binary reported v0.1.0. These archives are provisional because the release review found a code defect and the documentation has since changed.
- The packaged darwin/arm64 binary made one prompt-free owner-configuration usage query: two Codex, one Grok, and one Claude target returned four ordered `ok` rows with weekly values, no cleanup failure, and no stderr diagnostic. No account identifier, credential, raw provider response, or percentage was retained in this record. The installed CLIs were codex-cli 0.158.0, grok 1.0.44, and Claude Code 2.1.284. This is positive-path compatibility evidence for this baseline, not acceptance of the final binary or REQ-048.

## Review and correction

A separate read-only static review of the baseline found one release-blocking issue: `internal/process/child.go` allowed temporary-directory removal to overrun the shared two-second cleanup budget and still report success if removal eventually succeeded. This violates REQ-009 and REQ-040. The implementation now runs removal against the remaining absolute deadline, returns `ErrCleanupIncomplete` on expiry, and lets the filesystem operation finish best-effort in the background. A deterministic delayed-removal regression was added.

On the corrected, uncommitted worktree based on `3716d0e8cf099896939bd0f63a539e9edecdac65`:

- `go test -race ./internal/process -count=1`: exit 0, including the new delayed-removal case.
- A bounded independent read-only re-review of the correction delta reported no remaining blocking finding. The reviewer did not run tests; its verdict is static only.
- Full `make test` on native darwin/arm64 with Go 1.27.1: exit 0, including zero lint issues and E2E 32/32.
- The same full gate in the native linux/arm64 Go 1.27.1 container: exit 0, including zero lint issues and E2E 32/32.
- `go test ./... -count=1` on darwin/arm64: exit 0 across all packages, including `test/integration`.
- `sh scripts/build-release.sh v0.1.0` and the required staged-release check: exit 0, with all four archives and valid checksums. The extracted darwin/arm64 binary reports `vcs.revision=3716d0e8cf099896939bd0f63a539e9edecdac65` and `vcs.modified=true`, as expected for uncommitted code. These archives validate package layout and document freshness only; rebuild from the eventual clean release commit for distribution.
- A prompt-free `--json` query with that extracted, corrected darwin/arm64 binary: exit 0; two Codex, one Grok, and one Claude target returned four ordered `ok` rows with weekly values, `cleanup_failed=false`, and no stderr diagnostic. Only these fields were retained; no alias, identifier, credential, percentage, or raw provider response was recorded.

## REQ-048 live acceptance

The owner-facing ChatGPT usage screen was observed in the browser, then the packaged baseline binary was queried in the same quota period. One Codex row's used percentage matched the complement of the screen's remaining percentage twice; a later change in the displayed quota was reflected by the same row. This supports one comparison but does not independently prove which configured home maps to that browser login. The other Codex home has no matching provider display observation. The available Grok browser login showed a Free subscription and did not match the configured CLI account. No model turn was generated for these comparisons. The owner removed provider-display comparison from the v0.1.0 release gate on 2026-09-29; these observations remain supplemental evidence.

The owner also showed a four-row interactive `gaebokchi` snapshot with weekly values for two Codex, one Grok, and one Claude target. The `gaebokchi` binary on this workspace's `PATH` is an earlier dirty development build from `a28d551bc156451ecbe753e50bea4c5dad60a3a6`, not the corrected staging binary above. The owner's shell environment was not independently identified, so the snapshot is not attributed to that exact installed file. The corrected staged binary's separate four-row `--json` success above qualifies the revised REQ-048 on this candidate. The exact clean release binary remains a separate release-gate check.

The owner-approved acceptance change affected documents only. The corrected
candidate's full macOS and Linux gates above ran before that document change;
the product code is unchanged. After the change, `go run scripts/check-docs.go`
passed, and rebuilding all four provisional archives plus the staged-release
check passed again. The final clean release commit still needs its own gates.

The bounded independent review of that document change found three current
inconsistencies: the closeout procedure did not say when post-commit evidence
closes the gate, the monthly Grok exception could be read as satisfying the
new weekly-only REQ-048, and the architecture overstated the macOS-only
platform history. The focused confirmation review found the first two
resolved and identified one remaining stale sentence in ARCH-13. That sentence
was corrected to include TASK-031, TASK-032, and this task's corrected
uncommitted native runs. The reviewer did not execute tests or provider calls.

## Release decision

TASK-033 is **complete** on the reviewed, corrected candidate. The current
release gate remains **open** for exact-commit QA. Run both native full gates,
archive staging, and the four-account live check on this task commit. After
they pass, close EPIC-011 in a documentation commit and repeat the gates on
that exact release commit. Record its SHA and archive checksums in a
post-commit local receipt and owner handoff; binaries embed the commit
revision, so those values cannot appear in the same commit's tracked
document. Do not tag if the final receipt does not pass. No tag, push, or
hosted Release has been created.
