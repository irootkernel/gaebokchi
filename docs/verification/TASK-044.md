# TASK-044: Provider target selection

Status: COMPLETE — independently reviewed and accepted by the Coordinator
under the approved delegated Task authority.

## Candidate and scope

Base: `74ec93d986687750109e0b641707090f34166bad` on `main`, plus the
TASK-044 working changes. The completion commit is identified by subject
`[FEAT] Select configured providers by target name` and its EPIC-015 / TASK-044
attribution trailers.

Implements [REQ-070](../specs/README.md#req-070) with the alias configuration
and matching delivered by [TASK-043](TASK-043.md), under
[ARCH-21](../architecture/README.md#arch-21). TASK-045 owns final integrated
acceptance; REQ-069 and REQ-070 remain unchecked here.

`Options.TargetName` keeps the raw nonblank argument. Target construction skips
unselected providers before appending their accounts, preserving provider-local
IDs and dense indices. `Parse` checks the actual final target count, including
the legacy tagged-provider-key case where a matching provider produces no
targets. The builder signature and default query path remain unchanged.

The runner and all renderers receive the same selected target slice. Provider,
runner, renderer and domain implementations are unchanged. Help, public usage,
configuration instructions and affected canonical sections describe the new
option; the temporary dossier remains until TASK-045.

## Verification

Environment: native `darwin/arm64`, Go `go1.27.1`, golangci-lint `2.13.2`.
Execution used `GOPROXY=off`, `GOSUMDB=off` and `TMPDIR=/private/tmp`.

| Check | Command | Result |
|---|---|---|
| Focused gate | `go test ./internal/options ./cmd/gaebokchi ./internal/render ./test/integration -count=1` | PASS, exit 0 |
| Static analysis | `go vet ./internal/options ./cmd/gaebokchi ./internal/render ./test/integration` | PASS, exit 0 |
| Lint | `golangci-lint run ./internal/options ./cmd/gaebokchi ./internal/render ./test/integration` | PASS, exit 0; 0 issues |
| Formatting | `gofmt -l internal/options cmd/gaebokchi test/integration` | PASS, no output |
| Documentation | `go run scripts/check-docs.go` | PASS, exit 0; 65 Markdown files |
| Diff integrity | `git diff --check` and new-file whitespace checks | PASS, exit 0 |
| Release notes | Release-note inspector | PASS, exit 0; one approved entry under v0.1.2 Unreleased |

The options tests cover all four flag forms, shared short/long duplicate
rejection, missing/blank values, exact lowercase matching without trimming or
prefix/comma expansion, shared aliases and disabled providers. They verify
multiple Codex accounts, configured order, stable IDs, dense indices, both agy
pools and whole-configuration validation before selection. The tagged-key
regression checks the final empty-target error while preserving the existing
behavior without a target flag.

Command tests use recorded fake-provider calls to verify isolation, selected
failure rows and diagnostics, and existing exit semantics. A shared name
selecting Grok and ZCode produces the same two targets in the table,
deterministic terminal view and flat JSON, with each provider's own percentage.
Table and terminal retain weekly-above-5h grouping and order within each section.
Help and version tests preserve their configuration-free command forms and
reject unsupported target/version combinations and malformed target flags.

Executable tests run the real binary against fake CLIs and temporary homes.
They verify multiple selected accounts, shared Grok/Cursor and Grok/ZCode
aliases, configured JSON order, a ZCode-only 5h table, and a missing unselected
executable. Argument logs establish that selected authentication failures start
only the two Codex children, and invalid target/configuration cases start none.
Code inspection confirms that binary resolution and preflight occur inside
provider queries, which unselected providers never reach. A missing-bin result
alone is not used as proof that lookup did not occur.

The original signal-test helper remains byte-for-byte intact. All existing
tests run with the additions; no test was removed or disabled.

## Limits and handoff

- NOT RUN here: `make test`, the full all-package gate and platform/release
  qualification. TASK-045 owns complete offline acceptance on the final candidate.
- The earlier helper-environment ZCode SEA fixture failure and successful
  Coordinator reruns remain recorded in [TASK-043](TASK-043.md). This task's
  focused gate does not rerun that provider package or waive TASK-045's gate.
- Real account queries, provider-interface requalification, installation,
  tagging and publication are outside this filtering extension.
- TASK-045 receives the integrated option, isolation/output evidence and public
  instructions, and must retain durable contracts before retiring the dossier.
