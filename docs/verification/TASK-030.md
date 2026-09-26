# TASK-030: Live terminal usage view

Date: 2026-09-26. Checks ran on the TASK-030 implementation tree based on
`6d856ac`; the task-scoped commit records the same implementation and lifecycle
closure. This is an offline feature qualification, not a release or a live
provider-account acceptance campaign.

Requirements: [REQ-053](../specs/README.md#req-053),
[REQ-054](../specs/README.md#req-054), and
[REQ-055](../specs/README.md#req-055). Architecture:
[ARCH-05](../architecture/README.md#arch-05) and
[ARCH-10](../architecture/README.md#arch-10-terminal).

Changed scope: terminal selection and window-size query, live and final frame
rendering, optional per-result observation, focused tests, a correction to the
fake-provider environment-isolation test's child-header parsing, help, README,
canonical contracts, and the document checker's runtime-directory exclusion.
`golang.org/x/sys` and `golang.org/x/text` became
direct module dependencies; no new module was downloaded.

Environment: macOS darwin/arm64, Go 1.27.1, golangci-lint 2.13.2. Automated
tests use synthetic responses and private temporary provider homes. The
terminal check used an isolated tmux server, the locally built executable,
and a fake provider program; it launched no real provider CLI.

Checks on the working tree:

- `go test ./internal/render ./internal/app ./cmd/gaebokchi -count=1` — passed.
- `ginkgo --race --focus 'TestEnvironmentIsolationAndArgv' ./test/integration`
  — passed after correcting the test's distinction between child argv and
  inherited fake-script environment values.
- `make test` — passed: gofmt, `go vet ./...`, golangci-lint (0 issues),
  `go build ./...`, document audit (74 Markdown files in this local checkout),
  four unit race suites, eight integration race suites, and all 29 executable
  E2E specs. One previously existing Main suite spec was skipped by its own
  condition; no E2E specs were skipped. The first correction-tree run stopped
  at lint because a test did not check `Close`; the test was fixed before this
  passing run.
- `go run scripts/check-docs.go` — passed before the review: 51 Markdown files,
  links and IDs OK, including tracked `AGENTS.md` and `CLAUDE.md`. The later
  `make test` audit counted 74 because the local, untracked `.mulgae/` runtime
  held review reports; that count is not the source-tree file count.
- The Low-settlement staged source passed isolated `make test`: vet, lint
  (0 issues), build, document audit (50 Markdown files), four unit race suites,
  eight integration race suites, and 29/29 E2E specs. The pre-existing Main
  suite condition skipped one spec; no E2E spec was skipped.
- After correcting the document scan, `make test` passed directly in the
  repository with `.mulgae/` review reports present: vet, lint (0 issues),
  build, document audit (51 Markdown files), four unit race suites, eight
  integration race suites, and 29/29 E2E specs. The pre-existing Main suite
  condition skipped one spec; no E2E spec was skipped.
- After the third review's bounded Low corrections, direct `make test` passed
  again with the same gate stages and 29/29 E2E specs. The pinned
  `./scripts/ginkgo --race` focused check passed the new document-audit fixture:
  ignored runtime reports containing broken links are excluded, while a broken
  link in a canonical source document still fails the checker.

After the second review, the Low-settlement candidate was exported from the
Git index into a private temporary directory and checked there with
`make test`. The in-place attempt reached the document checker after vet,
lint, and build, then stopped on a link inside a local `.mulgae/` review report.
Those report-relative links are not repository documentation. The checker now
skips local Mulgae and other derived runtime directories while continuing to
scan the canonical documents and tracked root Markdown. The direct full gate
above verifies this correction in the affected working tree.

Focused rendering tests cover fixed-time colorless golden frames with and
without a countdown, including grouped providers and a sanitized wide alias;
numeric thresholds at 49.9/50/79.9/80 and colorless/styled status classes;
eighth-cell bars at 0, 0.1,
37, 99.99, and 100; reset provenance and countdown units; provider grouping;
control-character sanitization and East Asian wide labels; width tiers and
intermediate clipping; valid colorless bold SGR; concurrent observations;
short-terminal fallback; the first-write-error rule; and a modeled final screen with visible cursor
after cancellation. Runner tests cover one stored-result observation per
target, including normal completion and a queued cancellation. CLI tests drive
the selected live branch through `execute` with an injected terminal size and
environment, including styled, colorless, `TERM=dumb`, and broken-stdout cases.
An execute-level JSON case with a terminal-eligible stdout confirms that JSON
remains the only output. Existing table, JSON, help, version, diagnostic, and
exit tests passed in the full gate.

The real-terminal fixture showed a pending `querying…` row followed by the
final frame at 100, 60, and 40 columns. At 60 columns, `NO_COLOR=1` kept the
frame and removed color. `TERM=dumb` selected the plain table. Piped stdout
with `TERM=xterm-256color` was byte-identical to the plain-table control.
Sending Ctrl-C while the fake queries were pending produced `CANCELLED` rows,
exit 130, and a visible cursor. These checks used fake provider failures and
cancelled requests; they do not establish live account compatibility.

The first full `make test` reached E2E and failed only
`TestEnvironmentIsolationAndArgv`: it searched entire environment blocks for
`app-server`, which also appeared in the Grok child's inherited fake Codex
script value. The test now identifies the child using its argv header and
checks exact environment rows. Its focused rerun and the full gate passed.
The corrected test also requires both Codex and Grok child blocks so an empty
environment log cannot pass vacuously.

The selected static review recovered one rate-limited testing role through an
exact rerun and combined all six roles. Its accepted reports identified a
malformed `NO_COLOR` percentage SGR, missing automated coverage of the CLI
terminal branch, narrower width and runner observation coverage, and several
Low documentation and maintainability gaps. The corrections use typed styles,
exercise the CLI branch and writer failure, cover the middle width tiers and
normal runner completions, and reconcile help and package documentation. The
first review predated those corrected bytes. The second six-role Mulgae review,
`r_01a0d95c-41b4-7174-8ce8-1b78fbd34a0a`, completed with committed
publication, full role coverage, passing backend checks, and a successful
findings query on staged digest
`aac2ff11bac5cb31bfdb1a52c95c4318499d818a51fb67c5da0e1260b64469da`.
Its accepted reports found no Medium-or-higher issue and identified six Low
issues: JSON-terminal and colorless-status test gaps, duplicated frame-height
and row-layout metrics, inconsistent scope-note dates, and an overstated
golden-frame claim. Those six items were corrected in the staged candidate;
focused tests and a direct full gate verify the changed bytes. The second
review predates this Low-only delta and the later document-checker correction.

The third six-role Mulgae review,
`r_01a0d9c5-d22b-7086-b4c0-1bc061c4de05`, completed with committed
publication, full coverage, passing backend checks, and a successful findings
query on staged digest
`670da4eda55fa457b5e0d1e5531ca55bd235cb3bc8640838e060298710d87f56`.
The structured query returned zero findings; the accepted free-form reports
contained five Low suggestions and no Medium-or-higher issue. Three were
applicable and corrected locally: restore nested `.git`/`dist` exclusion,
add a synthetic document-audit regression test, and assert the numeric
percentage survives narrow final-frame tiers. Focused checks and direct
`make test` passed after these changes. The suggested policy abstraction and
an additional automated live-signal test were not needed for current
correctness or acceptance: the policy remains small and explicit, while the
approved real-terminal fixture covers signal composition. The third review
predates the three bounded Low corrections.

No real provider account query, installation, release staging, or publication
was performed. REQ-048 remains a separate owner-authorized live gate. The
task-scoped commit subject is `feat(TASK-030): add live terminal usage view`.
