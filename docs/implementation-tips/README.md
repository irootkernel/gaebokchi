# Implementation Tips

This directory owns non-normative development, testing, and release-engineering
guidance. Product behavior remains owned by the specifications.

## Local checks

```sh
gofmt -l .
go vet ./...
golangci-lint run ./... # v2.13.2
go test ./... -count=1
go test -race ./internal/... ./test/... -count=1
go run scripts/check-docs.go
govulncheck ./... # v1.8.0; queries the public Go vulnerability database
```

The normal test suite is offline and uses fake providers. It does not authorize
or perform real-account queries.

CI pins `golangci-lint` v2.13.2; use the same binary version locally. For the
optional vulnerability scan, install the scanner with the active Go 1.27
toolchain using `go install golang.org/x/vuln/cmd/govulncheck@v1.8.0`.

## Local release staging

```sh
sh scripts/build-release.sh v0.1.0
GAEBOKCHI_REQUIRE_STAGED_RELEASE=1 go test ./scripts/ -count=1
```

The build script replaces `dist/gaebokchi-v0.1.0` and stages four platform
archives plus checksums. Inspect any existing local staging before running it.
These commands do not publish, sign, tag, choose a license, or grant authority
to perform those operations.
