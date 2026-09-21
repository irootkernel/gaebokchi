# TASK-012: Add portable build delivery and user documentation

Task: TASK-012
Status: COMPLETE
Goal and implementation scope: Reproducible four-target release builds with version metadata and checksums, native macOS+Linux test evidence, cross-document link checker, CI configuration, and the user-facing README.
Requirements and architecture sections covered: REQ-001, REQ-046, REQ-047, REQ-049 (primary); ARCH-02, ARCH-11, ARCH-13, ARCH-14.
Changed files: README.md, scripts/build-release.sh, scripts/check-docs.go, .github/workflows/ci.yml, test/integration/e2e_test.go + internal/rpc/client_test.go (portability fixes found by the Linux run), docs/verification/TASK-012.md.
Verification (environments: macOS 26.6.2 arm64 with go1.26.6; Linux arm64 via golang:1.26.6-bookworm container):
  - V-BUILD: `sh scripts/build-release.sh 0.1.0` — all four archives (darwin/arm64, darwin/amd64, linux/arm64, linux/amd64) built with CGO_ENABLED=0, -trimpath, and `-X main.version=0.1.0`; `--version` on the extracted binary prints `gaebokchi 0.1.0`; SHA256SUMS verified OK for every archive; archive contents are the binary + README + the three specification documents only (no provider binaries, no secrets, no license invented). Also run: `go vet ./...` and `go test ./...` green on both platforms, and `go test -race ./test/integration` green on both.
  - V-PLATFORM matrix (recorded truthfully): darwin/arm64 NATIVE-TESTED (full suite + race); linux/arm64 NATIVE-TESTED (full suite + race + docs check + linux/amd64 cross-build, containerized); darwin/amd64 and linux/amd64 CROSS-BUILT ONLY (no native execution evidence). Live provider acceptance is darwin/arm64 only (TASK-013).
  - V-DOCS: `go run scripts/check-docs.go` — 18 markdown files, all local links/anchors and REQ/ARCH/EPIC/TASK/V/S identifiers resolve, on both platforms. README smoke: help/version/table examples match the actual binary output; statuses, exit codes, flags, prerequisites, privacy/side-effect statements reviewed against implementation.
  - CI: .github/workflows/ci.yml pins go1.26.6, runs vet/test/race/docs-check on macos-latest and ubuntu-latest, and the release build on Linux. Not executed here (no CI runner configured for this repository); the containerized linux run above is the recorded equivalent native evidence.
Fixtures: none new.
Findings and bounded remediation: the first Linux run exposed two test-portability defects, both fixed: the e2e suite depended on the developer machine having ~/.codex (now uses private temp homes by default) and a stderr-flood script exceeded the Linux single-argument limit (reduced; still above the 64 KiB retention cap); a timing-dependent rpc test was made deterministic (held reply).
Compatibility/platform limitations: darwin/amd64 and linux/amd64 remain build-qualified only; CI is configured but not yet exercised by an actual runner.
Task commit subject: `build(TASK-012): add macOS Linux delivery and usage guide`
Handoff to next task: TASK-013 runs the delivered binary against the owner's real accounts.
