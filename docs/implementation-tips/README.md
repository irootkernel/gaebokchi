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
| Cursor behavior | `internal/provider/cursor` | ARCH-19, REQ-065, REQ-066, the Cursor compatibility record |
| ZCode behavior | `internal/provider/zcode` | ARCH-20, REQ-067, REQ-068, the ZCode compatibility record |
| Child environment filtering | `internal/provider/envfilter` | ARCH-09 and the provider's own section |
| Table, JSON, terminal view, or diagnostics | `internal/render`, `cmd/gaebokchi` | ARCH-10, REQ-011, REQ-012, REQ-052 to REQ-055, REQ-068, the README |
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
[Cursor](../todo/TODO-CURSOR-AGENT-SUPPORT.md) dossiers map the provider
extensions to requirements and architecture. `EPIC-012` (`agy`) is complete.
On 2026-10-18, the owner retired the former "pure public CLI flags only"
restriction for `cursor-agent` and locked in the `EPIC-013` contract in
[ADR-0002](../architecture-decision-records/ADR-0002-period-aware-provider-usage.md),
[cursor-compatibility.md](../verification/cursor-compatibility.md), and the
[preflight record](../verification/provider-extension-preflight-2026-10-16.md).

For `EPIC-013` ([ARCH-19](../architecture/README.md#arch-19),
[REQ-065](../specs/README.md#req-065), [REQ-066](../specs/README.md#req-066)):
- Reuse the qualified `cursor-agent 2026.10.16-e3cb207` evidence in TASK-039:
  `<cursor-bin> status --format json` (default `cursor-agent`) executed with a
  read-only `NODE_OPTIONS=--require=<preloadPath>` hook on
  `./src/commands/status.ts` that calls `cursor-agent`'s bundled
  `dashboardClient.getPlanInfo` and `dashboardClient.getCurrentPeriodUsage` and
  writes one compact JSON line synchronously (`fs.writeSync(1, ...)`) with all
  PII omitted.
- In TASK-040, add optional single-entry `cursor` configuration in
  `~/.config/gaebokchi/config.yaml` (`- name: <alias>`, no `path` field, target
  ID `cursor:0`), `--cursor-bin` (default `cursor-agent`), `envfilter.Cursor()`
  (stripping `CURSOR_*` and ambient parent `NODE_OPTIONS` while setting the
  controlled `NODE_OPTIONS` preload path), and `internal/provider/cursor`.
- Render one weekly row (`QuotaID: "cursor_included"`,
  `Source: domain.SourceCursorCLIUsage`) using `planUsage.totalPercentUsed` (or
  `includedSpend / limit * 100` when `totalPercentUsed` is absent and
  `limit > 0`) and `includedUsageResetsAtMs` (falling back to
  `currentPeriodUsage.billingCycleEndMs`) when `includedUsagePeriod` is weekly
  (`2` / `INCLUDED_USAGE_PERIOD_WEEKLY` or a 7-day cycle window), and return
  `StatusNotApplicableMonthly` (`N/A (monthly)`) when monthly (`1` /
  `INCLUDED_USAGE_PERIOD_MONTHLY` or a monthly cycle window). Degrade cleanly to
  `StatusUnsupported` if a future CLI build does not trigger the hook.
- Add `internal/provider/cursor` to the explicit Makefile stage lists and root
  `TESTING.md` mapping, wire `Runner.querierFor(domain.ProviderCursor)`, and
  never allow the offline E2E harness (`test/integration/main_test.go`) to fall
  through to a real `agy` or `cursor-agent` executable. Release staging and
  publication remain outside these epics.

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
