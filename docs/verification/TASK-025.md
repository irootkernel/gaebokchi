# TASK-025: Final verification and candidate packaging

Task: TASK-025
Status: COMPLETE
Goal and implementation scope: Re-qualify the corrected tree end to end and package the candidate with packaged-document freshness verified, correcting the stale-document packaging found by the confirmation review.
Requirements and architecture sections covered: final-gate evidence for every re-qualified requirement; ARCH-13, ARCH-14.
Changed files: docs/plan.md and docs/requirement.md (final reconciled state, applied before packaging), internal/rpc/fakechild_test.go and internal/rpc/client_deadline_test.go and internal/rpc/client_test.go (test-stability fixes found during the gate — see findings), test/integration/cleanup_test.go (registry isolation fix), dist/gaebokchi-0.1.0/ (rebuilt archives).
Final gate — exact commands, outcomes, environments (source revision `bdde926d095ec7a722988dff15d8ea0799ed4d70` plus this task's finalization; environments: macOS 26.6.2 arm64 with go1.26.6; linux/arm64 via the golang:1.26.6-bookworm container):
  - `gofmt -l cmd internal scripts test` — empty, both platforms.
  - `go vet ./...` — pass, both platforms.
  - `go test -count=1 ./...` — all 12 packages pass, both platforms.
  - `go test -race -count=1 ./...` — all packages pass on darwin/arm64; on linux/arm64 the full race suite passed in the gate run, and the previously flaky combination was additionally verified at `-race -count=3` across all five subprocess-heavy packages (green).
  - `go run scripts/check-docs.go` — 33 markdown files OK, both platforms.
  - Four CGO-disabled builds via `sh scripts/build-release.sh 0.1.0` (darwin/arm64, darwin/amd64, linux/arm64, linux/amd64; -trimpath, `-X main.version=0.1.0`).
  - Archive inspection: every tar.gz contains exactly the binary, README.md, and the three specification documents (`tar -tzf`).
  - Document freshness (the confirmation review's packaging finding): all four archives extracted and every packaged document byte-compared with the finalized working tree — 16/16 IDENTICAL (`cmp -s`): README.md, docs/requirement.md, docs/architecture.md, docs/plan.md per archive. Documents were finalized before the build; the only file added after staging is this unpackaged verification record.
  - Version output: the extracted darwin/arm64 binary prints `gaebokchi 0.1.0`; `--help` prints the usage contract without launching a provider.
  - Checksums (`shasum -a 256 -c SHA256SUMS` in dist/gaebokchi-0.1.0 — all OK):
    - 34ce4c6ceb5021494e9edc1601eddc69d5f3308e78d9c121b98aced4aa1f1b23  gaebokchi-darwin-amd64.tar.gz
    - ac240e6d6d7509c9ac92aa7ed6ff8c7a91368207523312d9e3c87041cb228f08  gaebokchi-darwin-arm64.tar.gz
    - 0a10690af312eb8690deab5c48b3f5a9ff7e289e8d1ce5ba3205710dc9b17e6c  gaebokchi-linux-amd64.tar.gz
    - 0f29732b73a90934b20d5ef67853f13ef43b8b1052e02f743abfba538563a296  gaebokchi-linux-arm64.tar.gz
  - Previously fixed scenarios re-verified in the same runs: timely-quota survival across cleanup deadlines (runner/provider/rpc suites), late-response rejection, relative binary overrides, broken-pipe exit 1 and 130/143 precedence, error:null and ACP-marker rejection, mixed-advertisement classification, non-auth-error generic mapping, ASCII and Unicode whitespace rejection (all green in the gate above).
Findings fixed during this gate (all test-stability, no production change):
  - TestFakeChildStderrFloodDoesNotBreakProtocol asserted stderr retention before joining the drain goroutine — a pre-existing test race that failed intermittently under combined race stress; the child is now closed (drain joined) before the assertion.
  - TestCleanupRemovesOnlyOwnedDirs operated on the suite's shared registry and, under -count=N re-runs, could delete the still-in-use build directory; it now runs against a private registry slice.
  - Three timing ceilings in the rpc tests were widened from 2–3s to 5s: they guard against permanent wedges, not precise latency, and were flake-prone under full-load containers.
NOT RUN (recorded truthfully): native darwin/amd64 and linux/amd64 execution (cross-build only, as before); every live check — the corrected candidate has not been live-tested and the provider-display comparison was never performed (REQ-048, owner-authorized).
Platform matrix (corrected tree): darwin/arm64 NATIVE-TESTED (full suite + race); linux/arm64 NATIVE-TESTED (containerized: vet + full suite + race, plus -count=3 stress on the subprocess-heavy packages); darwin/amd64, linux/amd64 CROSS-BUILT ONLY.
Fixtures: unchanged.
Task commit subject: `test(TASK-025): requalify candidate and repackage archives`
Handoff: stop. All five residual findings are addressed with evidence; v0.1 release acceptance remains pending REQ-048 alone (owner-authorized live campaign: candidate live test plus the provider-display comparison). Do not start another feature automatically.
