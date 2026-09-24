# Go 1.27 tooling qualification

Observed 2026-09-24 on macOS/arm64 against `bc51fbf` plus the Go 1.27
tooling changes in the working tree. Historical TASK evidence remains bound to
its original compiler and source. This record does not establish CI execution
or live-provider compatibility.

## Versions and scope

- `go version`: `go1.27.1 darwin/arm64`; module language version `go 1.27`.
- `golangci-lint`: v2.13.2, built with Go 1.27.1. Its standard linters are
  `errcheck`, `govet`, `ineffassign`, `staticcheck`, and `unused`.
- `govulncheck`: v1.8.0, built with Go 1.27.1. It was already installed at
  this version when implementation began; no reinstall was needed.
- `go list -m all` lists only `gaebokchi`; there are no third-party module
  dependencies to upgrade.

## Local results

- Before remediation, unrestricted `golangci-lint run` reported 158 issues:
  156 `errcheck`, one `staticcheck`, and one `unused`. After remediation,
  `golangci-lint config verify` and `golangci-lint run` pass with no issues.
- `gofmt -l .` reports no files. `go vet ./...`, `go mod tidy -diff`,
  `go run scripts/check-docs.go` (48 Markdown files), `git diff --check`,
  `go test ./... -count=1`, and
  `go test -race ./internal/... ./test/... -count=1` pass.
- `govulncheck ./...` reports no vulnerabilities against the public Go
  vulnerability database observed on this date. It does not query provider
  accounts.
- In a disposable source copy, `sh scripts/build-release.sh v0.1.0` built
  darwin/arm64, darwin/amd64, linux/arm64, and linux/amd64 archives;
  `GAEBOKCHI_REQUIRE_STAGED_RELEASE=1 go test ./scripts/ -run
  'TestVerify|TestStagedReleaseArchives' -count=1` and
  `shasum -a 256 -c SHA256SUMS` passed for all four. The extracted darwin/arm64
  binary printed `gaebokchi v0.1.0` and the expected JSON version object.
  The existing repository `dist/` archives were preserved.

CI, native Linux execution, and live account queries were not run locally.
The other three archives are cross-build evidence only. CI will exercise the
new pinned Actions and linter when this tree is submitted.
