# Implementation Tips

This directory owns non-normative development, testing, and release-engineering
guidance. Product behavior remains owned by the [specifications](../specs/README.md),
and structure by the [architecture](../architecture/README.md).

## Local checks

```sh
make test          # prepare, unit, integration, and E2E, serially
make test-prepare  # gofmt -w, go vet, golangci-lint, go build, check-docs
make test-unit
make test-int
make test-e2e
```

The root `TESTING.md` owns the stage mapping, pinned Ginkgo runner, local
fake-provider E2E environment, and approved legacy E2E waiver. The complete
`make test` gate formats Go source before running vet, lint, build, and the
behavioral stages. It requires Go 1.27 and `golangci-lint` v2.13.2.

The normal test suite is offline and uses fake providers. It does not authorize
or perform real-account queries.

CI pins `golangci-lint` v2.13.2; use the same binary version locally. For the
optional vulnerability scan, install the scanner with the active Go 1.27
toolchain using `go install golang.org/x/vuln/cmd/govulncheck@v1.8.0`.

## Where to change what

| Change | Start in | Also update |
|---|---|---|
| Flags, defaults, or target order | `internal/options` | ARCH-03, REQ-006 to REQ-008, the help text in `cmd/gaebokchi/main.go`, the README |
| Statuses, labels, or percentage policy | `internal/domain` | ARCH-04, REQ-010 to REQ-015, the README status tables |
| Concurrency, deadlines, or cancellation | `internal/app`, `internal/process` | ARCH-05, REQ-009, REQ-020, REQ-022, REQ-023 |
| Codex or Grok wire behavior | `internal/rpc`, `internal/provider/codex`, `internal/provider/grok` | ARCH-06 to ARCH-08, the `testdata/` fixtures, the compatibility record |
| Claude behavior | `internal/provider/claude` | ARCH-16, REQ-056 to REQ-059, the Claude compatibility record |
| `agy` behavior | `internal/provider/agy` | ARCH-17, ARCH-18, REQ-060 to REQ-064, the agy compatibility record |
| Child environment filtering | `internal/provider/envfilter` | ARCH-09 and the provider's own section |
| Table, JSON, terminal view, or diagnostics | `internal/render`, `cmd/gaebokchi` | ARCH-10, REQ-011, REQ-012, REQ-052 to REQ-055, the README |
| Exit codes or signals | `cmd/gaebokchi` | ARCH-10, REQ-024 |
| Release packaging | `scripts/build-release.sh`, `scripts/archivecheck.go` | ARCH-13, REQ-046, the release procedure below |

## Changing a provider interface

1. Qualify the installed CLI first with a bounded, prompt-free probe
   ([ARCH-12](../architecture/README.md#arch-12)). A source handler or a
   documentation page is not enough.
2. Sanitize captured transcripts before they enter the repository, and tag
   each fixture as captured or synthetic.
3. In the same change, update the requirement, the architecture section, the
   fixtures, the tests, and the compatibility record, and add a dated Changed
   note to each affected requirement.
4. Keep live account queries out of the normal test suite.

## Planned provider and output work

The adopted [agy](../todo/TODO-AGY-MULTI-ACCOUNT-SUPPORT.md) and
[Cursor](../todo/TODO-CURSOR-AGENT-SUPPORT.md) dossiers map the future work to
requirements and architecture. On 2026-10-17, the owner scoped `EPIC-012` to the
single default `agy` login with two weekly rows (`gemini` and `claude/gpt`) in
`gaebokchi`'s existing weekly-only output, and the `agy 1.4.0` metadata contract
was qualified in [agy-compatibility.md](../verification/agy-compatibility.md)
and the [preflight record](../verification/provider-extension-preflight-2026-10-16.md).
TASK-039 must still qualify the Cursor CLI metadata contract before TASK-040.

Use ARCH-17 and ARCH-18's runtime guards for `agy`: verify `<agy-bin> --version`
(`>= 1.2.15`) within the target timeout budget before running
`agy --print /usage --output-format json --print-timeout 15s`, inject
`AGY_CLI_DISABLE_AUTO_UPDATE=true`, strip `ANTIGRAVITY_*`, `JETSKI_*`, `AGY_*`,
and API-key/ADC overrides while preserving `HOME`, and terminate the child
immediately if `Authentication required` appears on stderr. Convert
`remaining_fraction` $r \in [0, 1]$ using bounded exact decimal token math
(`100 * (1 - r)`) in TASK-035 and ignore `window: "5h"` entries.

In TASK-036 and TASK-037, keep the existing 3-column plain table, interactive
terminal view, and `--json` `results` schema unchanged. Expand an enabled `agy`
entry in `config.yaml` into two weekly targets (`agy:0` with label `gemini` and
`agy:1` with label `claude/gpt`) populated by a single shared `agy` subprocess
invocation (`sync.Once`). Add each new provider's fake-process suite to the
explicit Makefile stage lists and the root `TESTING.md` mapping, and replace the
implicit Codex fallback in `Runner.querierFor` with explicit provider dispatch.
Never allow the offline E2E harness (`test/integration/main_test.go`) to fall
through to a real `agy` or `cursor-agent` executable or misroute `agy` argv to
Grok. Release staging and publication remain outside these epics.

## Release procedure

These steps stage and publish a release from a clean checkout. Building and
validation change only `dist/` and the Go caches. Tagging and publishing are
separate actions that need the owner's explicit approval each time; this guide
does not grant that authority. TASK-033's completion commit records the
corrected candidate and its live evidence. Run steps 1–6 on that clean commit;
after they pass, make the separate EPIC-011 closeout commit that checks the
remaining release-gate items and marks the epic complete. That closeout commit
is the release commit only after steps 1–6 pass again on its exact bytes. This
avoids putting a commit's own SHA or archive checksums in its tracked source
tree.

1. **Start clean.** On the commit to release, `git status --short` prints
   nothing.
2. **Run the offline gate.** Run `make test`. Its prepare stage runs
   `gofmt -w`, so confirm afterwards that `git status --short` still prints
   nothing.
3. **Run natively on Linux.** Run the same gate natively on Linux against the
   release commit, for example through the CI workflow or in a container.
   Record the exact commit and CPU architecture for each native gate. TASK-031
   and TASK-032 include native darwin/arm64 and linux/arm64 evidence; the final
   release commit needs its own runs.
4. **Stage the archives.**

   ```sh
   sh scripts/build-release.sh v0.1.0
   GAEBOKCHI_REQUIRE_STAGED_RELEASE=1 go run github.com/onsi/ginkgo/v2/ginkgo --fail-on-empty --label-filter=staged-release ./scripts
   ```

   The build script replaces `dist/gaebokchi-v0.1.0` with four platform
   archives plus `SHA256SUMS`; inspect any existing local staging before
   running it. The staged-release check verifies the archive members (including
   LICENSE and THIRD_PARTY_NOTICES.txt), executable permissions,
   byte-identical packaged text files, and checksums.
5. **Check build provenance.** Extract one archive and run `go version -m` on
   its binary: `vcs.revision` must be the release commit and `vcs.modified`
   must be `false`.
6. **Run live acceptance.** On the owner's supported account host, run the
   extracted candidate binary against the configured two Codex homes, Grok,
   and Claude without a model turn. Verify four ordered weekly `ok` rows,
   no cleanup failure, and no stderr diagnostic as [REQ-048](../specs/README.md#req-048)
   requires. Record the candidate revision, provider CLI versions, platform,
   and outcome; do not retain account identifiers or raw provider output.
   The provider-display comparison is optional follow-up evidence.
7. **Finalize the gate receipt.** After the closeout commit passes steps 1–6,
   record its full SHA, all four archive checksums, platform gate results,
   archive check, and sanitized four-row live outcome in a local receipt under
   ignored `dist/gaebokchi-v0.1.0/`. Verify that the tracked requirement,
   task, epic, and release-gate checkboxes agree with that evidence. If any
   check fails, stop before tagging and correct the repository state; do not
   treat the closeout commit's pre-commit evidence as a substitute for this
   receipt.
8. **Tag and publish, with owner approval only.**

   ```sh
   git tag -a v0.1.0 -m "gaebokchi v0.1.0" <release-commit>
   git push origin v0.1.0
   gh release create v0.1.0 --verify-tag --title "gaebokchi v0.1.0" --notes-file <release-notes.md> \
     dist/gaebokchi-v0.1.0/gaebokchi-*.tar.gz dist/gaebokchi-v0.1.0/SHA256SUMS
   ```

   Upload only the `dist/gaebokchi-v<version>/` directory you just staged;
   other directories under `dist/` can hold stale staging from earlier builds.
9. **Verify the download.** Follow the README's install steps against the
   published release, including checksum verification.

The archives include the project `LICENSE` and `THIRD_PARTY_NOTICES.txt`; see
[ARCH-13](../architecture/README.md#arch-13). Refresh third-party texts and
their stated versions whenever a bundled dependency or release Go toolchain
changes.

## Changing the release version

Update these together:

- the default version in `scripts/build-release.sh`;
- the release build job in `.github/workflows/ci.yml`;
- the required staging path in `scripts/archivecheck_test.go`;
- the examples in the README and in this guide;
- if the protocol client version should follow the release, the fixed
  `clientInfo` version `0.1.0` in `internal/provider/codex/provider.go` and
  `internal/provider/grok/provider.go`.
