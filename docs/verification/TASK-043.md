# TASK-043: Provider alias configuration and matching

Status: COMPLETE — independently reviewed and accepted by the Coordinator
under the approved delegated Task authority.

## Candidate and scope

Base: `b429739ae2fd961ee8a8f9b76f6e33aed093f580` on `main`, plus the
TASK-043 working changes. The completion commit is identified by subject
`[FEAT] Add provider aliases and matching` and its EPIC-015 / TASK-043
attribution trailers, avoiding a self-referential commit hash.

Implements the configuration and matching portion of
[REQ-069](../specs/README.md#req-069) under [ARCH-21](../architecture/README.md#arch-21).
The target flag, execution filtering and output/no-launch evidence belong to
TASK-044. TASK-045 owns integrated acceptance; REQ-069 and REQ-070 stay unchecked.

Changed areas: `internal/options/config.go`, new alias validation/matching and
focused tests, the canonical partial-delivery annotations, this record and its
verification-index entry. Provider protocols, rendering and default queries
are unchanged.

## Verification

Environment: native `darwin/arm64`, Go `go1.27.1`, golangci-lint `2.13.2`.
Commands run with `GOPROXY=off`, `GOSUMDB=off` and a physical temporary root
outside the repository.

| Check | Command | Result |
|---|---|---|
| V-OPTIONS | `go test ./internal/options -count=1` | PASS, exit 0 |
| Static analysis | `go vet ./...` | PASS, exit 0 |
| Lint | `golangci-lint run ./...` | PASS, exit 0; 0 issues |
| Offline regression | `go test ./... -count=1` | PASS, exit 0 in the Coordinator environment; earlier helper failure described below |
| Formatting | `gofmt -l internal/options` | PASS, no output |
| Documentation | `go run scripts/check-docs.go` | PASS, exit 0; 64 Markdown files |
| Diff integrity | `git diff --check` | PASS, exit 0 |

The focused tests verify all six default names, mixed case, the absence of an
implicit `codex` alias, configured aliases, shared/default-colliding and duplicate
matches, exact non-trimmed spelling, and invocation-local isolation. They check
that absent/empty alias metadata and metadata at every top-level position leave
all default targets and provider order unchanged, including multiple Codex
accounts and agy's two pools.

Negative cases cover unknown or incorrectly cased provider keys, invalid mapping
and list shapes, nulls, non-string scalars and nested values, blank aliases,
duplicate keys including YAML alias keys, merge keys and extra documents.
Positive YAML anchors and aliases remain supported. Disabled providers remain
disabled, including when they share another provider's default name; metadata
alone still fails the existing account requirement.

Root metadata keys are recognized using the same decoded string as the typed
YAML decoder. Regression tests reject numeric aliases under a binary-tagged
`aliases` key, accept valid tagged metadata without changing provider order or
default targets, and preserve anchored root-key behavior. A temporary diagnostic
probe reproduced the validation bypass before the correction and passed afterward;
the permanent tests retain this coverage.

The first all-package run in the helper environment failed the existing ZCode
SEA asset fixture at `internal/provider/zcode/provider_test.go:611`; its expected
`zcode-builtin.json` was absent. A package rerun there reproduced the failure.
The same package passed in the Coordinator environment, both directly and under
the helper's OS Git-directory write restriction. The complete all-package run
also passed there on the corrected source. The environmental cause was not
established; no ZCode provider or fixture source was changed.

Fixtures are synthetic YAML in test-owned temporary homes. The suite does not
query provider CLIs or real accounts.

## Limits and handoff

- NOT RUN here: `make test`; TASK-045 owns the complete offline gate.
- NOT RUN here: targeted executable no-launch and output scenarios; TASK-044
  owns the flag and execution integration.
- Real account queries, provider requalification, installation and release work
  are outside this filtering extension.
- TASK-044 receives strict alias metadata and `accountConfig.matchProviders`,
  which returns every matching enabled provider once. Default execution still
  queries all configured accounts.
