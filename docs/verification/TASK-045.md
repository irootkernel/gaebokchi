# TASK-045: Integrated provider-selection acceptance

Date: 2026-10-23
Status: COMPLETE — independently reviewed and accepted by the Coordinator
under the approved delegated Task authority.

## Candidate and scope

Tested implementation: `d1868895aa630615afb7c01e54e73e770ad87230` on `main`,
following the alias implementation in `74ec93d`. Both full gates ran before
these documentation changes. `make test` left all tracked files unchanged.
This task adds this record and its index entry, connects requirement evidence,
updates the documentation, roadmap and TODO indexes, and removes the temporary
provider-selection dossier. No Go source, tests, provider protocols, dependencies,
public configuration contract or release-note entry changes.

The completion commit is identified by subject
`[DOC] Accept provider selection and retire the dossier` and its
EPIC-015 / TASK-045 attribution trailers. [REQ-069](../specs/README.md#req-069)
and [REQ-070](../specs/README.md#req-070) are assessed under
[ARCH-21](../architecture/README.md#arch-21). Final assembled Epic review and
Master's overall acceptance remain separate from delegated Task acceptance.

## Actual checks

Coordinator ran the successful checks on native `darwin/arm64`, Go `go1.27.1`,
golangci-lint `2.13.2`. Each command used `GOPROXY=off GOSUMDB=off
TMPDIR=/private/tmp`; no real provider login or usage was queried.

| Command | Result |
|---|---|
| `make test` | PASS, exit 0: prepare (gofmt, vet, lint with 0 issues, build, docs), 4 unit suites, 11 integration suites and 1 executable suite; Ginkgo race and randomized ordering enabled |
| `go test ./... -count=1` | PASS, exit 0: all 16 packages with tests; 2 packages have no test files |
| `go run scripts/check-docs.go` after retirement | PASS, exit 0; 65 Markdown files, links and identifiers valid |
| `git diff --check` and new-file whitespace/EOF checks | PASS, exit 0 |
| Retired-path reference search with `rg` | PASS, zero remaining references |

The successful race gate includes ZCode 7/7 specs and executable integration
44/44 specs, with no skipped executable specs. The succinct gate prints existing
conditional skip markers in the options and scripts suites; this is not a claim
that every platform-dependent or release-artifact check ran. No skip or pending
test was added, and the configured gate passed without a waiver.

Before that success, the Executor's helper environment ran `make test` and
failed with exit 2 in
`TestPreloadScriptReadOnlyNoOAuthPostNoCorruptBackupAndSeaIsolation`
(`internal/provider/zcode/provider_test.go:611`): the expected SEA builtin asset
was absent from the temporary preload directory. Its prepare and unit stages
passed; make stopped in test-int, so its E2E stage and separate all-package
command were not run. No source or documentation changed. The Writer stopped
and handed off before Coordinator ran both complete gates on the same source.
This reproduces the earlier [TASK-043](TASK-043.md) helper failure; the cause
remains unconfirmed. Neither the failed helper run nor an omitted stage is
reported as passing, and no gate was waived.

## Requirement and regression evidence

The fixtures use synthetic YAML, temporary homes and scripted fake CLIs. Earlier
focused results remain in [TASK-043](TASK-043.md) and [TASK-044](TASK-044.md);
the full gates above reran their tests with the existing provider regressions.

| Contract | Existing checks retained and rerun |
|---|---|
| REQ-069 strict aliases, all defaults, shared/colliding aliases, disabled providers and metadata order | `aliases_test.go`: `matches all six display names without implicit config-key aliases`, `unions shared aliases and default-name collisions once per enabled provider`, `rejects malformed provider aliases before constructing any targets`, metadata/order and tagged-root-key regressions |
| REQ-070 flag forms, errors, whole config validation, actual target count and dense IDs | `target_test.go`: four flag forms; blank/missing/repeated values; stable IDs/dense indices; whole-configuration validation; actual zero-target and legacy tagged-provider-key regressions |
| Unselected providers perform no work; selected failures retain rows and exit status | command tests `target selection queries only matching providers in table and JSON`, `selected target failures keep their row, diagnostic, and exit status`; executable tests assert actual child argument logs, no children for invalid input/config and no unselected children |
| Three output modes, shared weekly/5h selection and configured ordering | command test `a shared alias selects a weekly provider and ZCode in every output mode`; executable shared Grok/ZCode, ZCode-only 5h and Cursor-before-Grok JSON cases; agy retains both pool targets |
| Unchanged default behavior, help/version and provider regressions | existing options configured-order/default tests, command help/version/error/signal tests, complete provider/process/RPC suites and executable default/partial-failure/cancellation cases |

Code inspection confirms the no-lookup boundary: only selected targets
reach the runner, provider constructors perform no executable resolution, and
resolution/preflight occur inside their queries. Missing-bin output alone is
not treated as proof of lookup absence. Table and terminal preserve period
grouping and configured order within a section; JSON preserves flat target
order and each provider's percentage.

## Durable documentation and handoff

The specifications and ARCH-21 retain strict configuration, exact lowercase
matching, shared aliases, stable IDs and execution boundaries. Public README
and configuration guidance retain the example, default names, optional `codex`
alias, all four flag forms and error behavior. The three Task records retain
actual evidence and limitations. After this content was confirmed and successful
gate evidence recorded, the provider-selection dossier and its three incoming
links were retired. The unrelated agy and Cursor dossiers are unchanged.

Not performed: real account queries, provider-interface requalification,
installation, Linux or additional CPU/platform qualification, release staging,
tagging or publication. They are outside this filtering extension's acceptance
scope. Existing qualification and historical Task records keep their original
boundaries. TASK-045 adds no release note; the approved v0.1.2 entry is unchanged.
The assembled Epic still requires its independent final review and Master's
actual overall acceptance.
