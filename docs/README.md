# Gaebokchi Documentation

This directory is the maintainer and contributor documentation for Gaebokchi. If you only want to install and run the tool, start with the root [README](../README.md).

## Start here

1. The root [README](../README.md) describes the product as users see it: installation, usage, statuses, privacy, and limitations.
2. The [architecture](architecture/README.md) maps the packages and how a query flows through them; start with [ARCH-02](architecture/README.md#arch-02).
3. The [specifications](specs/README.md) define the required behavior, as `REQ-###` requirements that every change must keep.
4. [`TESTING.md`](../TESTING.md) and the [implementation tips](implementation-tips/README.md) explain how to verify a change and how to stage and publish a release.
5. The [roadmap](roadmap/README.md) records how the work was planned and delivered, and how new work is admitted.

## Profile and ownership

- Profile: `single-scope`
- Delivery scope: `gaebokchi`
- Documentation language: English; the human-readable product name may include `Gaebokchi (개복치)`.

| Role | Canonical owner | What it holds |
|---|---|---|
| Specifications | [specs/README.md](specs/README.md) | Required behavior and user-visible contracts, with acceptance checkboxes |
| Architecture | [architecture/README.md](architecture/README.md) | Components, data semantics, protocol and process rules, verification design |
| Architecture decision records | [architecture-decision-records/README.md](architecture-decision-records/README.md) | Decisions whose rationale should outlive an architecture rewrite |
| Implementation tips | [implementation-tips/README.md](implementation-tips/README.md) | Non-normative guidance for changing, testing, and releasing |
| Operations | [ops/README.md](ops/README.md) | Operated environments; this local CLI has none |
| Roadmap | [roadmap/README.md](roadmap/README.md) | Epic and task identity, order, status, and delivery history |
| TODO and temporary dossiers | [todo/README.md](todo/README.md) | Future epic-sized candidates and adopted execution dossiers |
| Deferred feedback | [deferred-feedback/README.md](deferred-feedback/README.md) | Small findings postponed from current work |

Supporting documents outside the eight roles:

- [configuration/README.md](../configuration/README.md) is the public account setup guide. The root README remains the public product entrypoint.
- [verification/README.md](verification/README.md) indexes snapshot-bound validation evidence. Evidence supports claims but is not a ninth normative role and does not replace current specifications, architecture, or roadmap state.
- [`TESTING.md`](../TESTING.md) at the repository root is the test contract: stages, tools, and the E2E environment.
- [`AGENTS.md`](../AGENTS.md) holds the guidance for AI coding agents; `CLAUDE.md` defers to it.

## Audience boundary

The root README is the public product entrypoint and must stay useful to someone who has never seen this repository. It summarizes public behavior and is not a second product contract. This directory is for maintainers and contributors.

Release archives carry the root README, LICENSE, THIRD_PARTY_NOTICES.txt, configuration guide, specifications, architecture, and roadmap (`scripts/build-release.sh`, enforced by `scripts/archivecheck.go`). Links from those documents to other files, such as verification records, resolve only in the repository.

## Provider extensions

[EPIC-012](roadmap/README.md#epic-012) (`TASK-034` through `TASK-038`, `REQ-060`
through `REQ-064`, `ARCH-17`, and `ARCH-18`) is **COMPLETE**: `gaebokchi` supports
single-account `agy` weekly usage (`gemini` and `claude/gpt` rows) from a single
shared `agy --print /usage --output-format json --print-timeout 15s` invocation,
verified in [agy-compatibility.md](verification/agy-compatibility.md) and the
[preflight record](verification/provider-extension-preflight-2026-10-16.md).

[EPIC-013](roadmap/README.md#epic-013) (`TASK-039` through `TASK-041`, `REQ-065`,
`REQ-066`, and `ARCH-19`) is **COMPLETE**: `gaebokchi` supports single-account
Cursor default-login weekly usage (`cursor:0`) using `<cursor-bin> status --format json`
with a read-only `NODE_OPTIONS` preload hook, qualified and verified against
`cursor-agent 2026.10.16-e3cb207` in
[cursor-compatibility.md](verification/cursor-compatibility.md) and the
[preflight record](verification/provider-extension-preflight-2026-10-16.md).
Each extension has a linked [execution dossier](todo/README.md). The roadmap
owns lifecycle state; the dossiers map tasks to canonical requirements and
architecture.

The ZCode (`zcode`) 5h usage extension (`REQ-067`, `REQ-068`, and `ARCH-20`) adds
single-account ZCode Coding Plan 5-hour usage (`zcode:0`) via `<zcode-bin> doctor --json`
with a read-only `NODE_OPTIONS` preload hook and splits the UI into top
**Weekly usage** and bottom **5h usage** sections when configured, verified
against `zcode 0.16.9` in [zcode-compatibility.md](verification/zcode-compatibility.md).

## Precedence

When documents disagree, resolve the conflict in the same scoped change. Until that happens, use this precedence: specifications, accepted ADRs, current architecture, roadmap, then implementation or operations guidance.

## Identifiers

- **Roadmap.** The sole roadmap namespace is `docs/roadmap/README.md`. Existing `EPIC-NNN` and `TASK-NNN` identifiers, lifecycle terms, ordering, and completion history are preserved. Epic and task sequences are independent and monotonic; numbers are never reused. New allocations increment the current epic maximum of 13 and task maximum of 41. Cross-scope qualification is unnecessary while this repository has one delivery scope.
- **Specification.** Requirements (`REQ-###`), architecture sections (`ARCH-##`), verification catalog entries (such as V-CODEX, in ARCH-11), and primary sources (`S-##`, in ARCH-15) are stable identifiers with explicit `<a id>` anchors. They are never renumbered or reused. The current maxima are REQ-068, ARCH-20, and S-24.
- **Decision records.** ADR file names are numbered from `ADR-0001` upward.

## Checks

- `go run scripts/check-docs.go` resolves every relative Markdown link and cross-file fragment and requires the ten canonical documents. It also requires every REQ, ARCH, EPIC, TASK, V, and S identifier mentioned under `docs/` to appear in the specifications, architecture, or roadmap. It does not check fragments that point within the same file. `make test-prepare` runs it.
- The staged-release check in the [implementation tips](implementation-tips/README.md) verifies that packaged documents are byte-identical to the repository.

## History

The 2026-09-23 path migration changed only documentation ownership paths:

| Previous path | Canonical path |
|---|---|
| `docs/requirement.md` | `docs/specs/README.md` |
| `docs/architecture.md` | `docs/architecture/README.md` |
| `docs/plan.md` | `docs/roadmap/README.md` |

On 2026-09-28 the specifications and architecture were consolidated into current-state documents for the first public release:

- The former "Current Claude extension", "Current JSON output extension", and "Current terminal view extension" preambles were folded into the body.
- The EPIC-007 supersession notes moved out of the specifications, whose trace lines now name the re-qualifying tasks.
- Claude support gained REQ-056 through REQ-059, ARCH-16, V-CLAUDE, and ADR-0001.
- Every earlier identifier and inbound anchor was kept.

Completed verification records and roadmap history are snapshots and are not rewritten. They can mention the previous paths, the former preamble sections, or superseded display details such as `0.0%` for a whole percentage. Read such mentions as descriptions of the tree they verified, not as current authorities or redirects.

## Release metadata

- License: MIT ([`LICENSE`](../LICENSE)).
- Distribution: GitHub Releases for `github.com/irootkernel/gaebokchi`, plus source builds. The [implementation tips](implementation-tips/README.md) hold the staging and publication procedure; documentation never grants authority to tag or publish.
