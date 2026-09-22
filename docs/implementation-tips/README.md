# Implementation Tips

This directory owns non-normative development, testing, and release-engineering
guidance. Product behavior remains owned by the specifications.

## Local checks

```sh
gofmt -l .
go vet ./...
go test ./... -count=1
go test -race ./internal/... ./test/... -count=1
go run scripts/check-docs.go
```

The normal test suite is offline and uses fake providers. It does not authorize
or perform real-account queries.

## Local release staging

```sh
sh scripts/build-release.sh 0.1.0
GAEBOKCHI_REQUIRE_STAGED_RELEASE=1 go test ./scripts/ -count=1
```

The build script replaces `dist/gaebokchi-0.1.0` and stages four platform
archives plus checksums. Inspect any existing local staging before running it.
These commands do not publish, sign, tag, choose a license, or grant authority
to perform those operations.
