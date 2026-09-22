# TASK-028: Implement the stable version command and release identity

Task: TASK-028
Status: COMPLETE — implementation and offline verification are complete; the
task-scoped commit is identified by the unique subject recorded below.
Goal and implementation scope: Replace the legacy `--version` flag with the
project-independent `version [--json]` command; expose only product name and
version; resolve release metadata from linker override, Go build information,
then `(devel)`; use a v-prefixed release identity and staging path.
Requirements and architecture sections covered: REQ-051; ARCH-03, ARCH-13,
ARCH-14.
Changed files: `cmd/gaebokchi` version dispatch/resolution and tests;
`internal/options` legacy-flag removal and tests; executable broken-pipe test;
release build/archive/CI paths; README, canonical architecture/specification/
roadmap, implementation tips, and verification index. Generated local artifacts
are staged under ignored `dist/gaebokchi-v0.1.0/`.
Verification (2026-09-23, macOS 26.6.2 darwin/arm64, Go 1.26.6):

- `go vet ./...` — pass.
- `go test ./... -count=1` — pass, including command, options, scripts, and
  executable integration coverage.
- `go test -race ./internal/... ./test/... -count=1` — pass.
- `test -z "$(gofmt -l .)"`, `go run scripts/check-docs.go`, and
  `git diff --check` — pass.
- `go run ./cmd/gaebokchi version` and `version --json` — source-build outputs
  `gaebokchi (devel)` and exactly
  `{"name":"gaebokchi","version":"(devel)"}`.
- `sh scripts/build-release.sh v0.1.0` — four CGO-disabled darwin/linux,
  amd64/arm64 archives staged in `dist/gaebokchi-v0.1.0/`; unprefixed input
  `0.1.0` rejects with exit 2.
- Mandatory staged archive tests and independent `shasum -a 256 -c` — pass for
  all four archives; packaged documents match the finalized working tree.
- Extracted darwin/arm64 release smoke — `gaebokchi v0.1.0` and exactly
  `{"name":"gaebokchi","version":"v0.1.0"}`.

Checksums:

- darwin/amd64: `0df8669db41751c200f38cab5f2250c7f7de1b832ba39964a78adaea6a48b30f`
- darwin/arm64: `7681b1b1d51f0223e6ba04dac8b0d6430ff4251b958b6adc66a1990b94ce56f4`
- linux/amd64: `580fc633fa0c06477e0752777e3079e0ac6b9ee6992f0ba534911dfff792ddb7`
- linux/arm64: `1eb9ae4db6abfbce0a71650f57599363428f30e081e1d21d47c42b34647b0014`

Fixtures: none; version behavior is covered by deterministic unit and executable
tests.
Findings and bounded remediation: the first focused executable test showed that
moving version dispatch before the existing signal setup exposed SIGPIPE on a
closed stdout pipe. `run` now installs the write-error policy before early
dispatch, and the real-pipe regression exits 1. No finding remains in scope.
Compatibility/platform limitations: native execution in this task covered
darwin/arm64; darwin/amd64 and both Linux artifacts are cross-build-only for
this working tree. Live provider acceptance is outside this task; REQ-048
remains open.
Task commit subject: `feat(TASK-028): align version command and release identity`.
Handoff: TASK-028 and EPIC-008 are closed. REQ-048 remains open and requires a
separate owner-authorized live campaign; no live provider work is implied here.
