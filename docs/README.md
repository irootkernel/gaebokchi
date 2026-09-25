# Gaebokchi Documentation

This directory is the maintainer and contributor documentation authority for
Gaebokchi. The repository-root `README.md` is the public product entrypoint.

## Profile and ownership

- Profile: `single-scope`
- Delivery scope: `gaebokchi`
- Documentation language: English; the human-readable product name may include
  `Gaebokchi (개복치)`.

| Role | Canonical owner |
|---|---|
| Specifications | [specs/README.md](specs/README.md) |
| Architecture | [architecture/README.md](architecture/README.md) |
| Architecture decision records | [architecture-decision-records/README.md](architecture-decision-records/README.md) |
| Implementation tips | [implementation-tips/README.md](implementation-tips/README.md) |
| Operations | [ops/README.md](ops/README.md) |
| Roadmap | [roadmap/README.md](roadmap/README.md) |
| TODO and temporary dossiers | [todo/README.md](todo/README.md) |
| Deferred feedback | [deferred-feedback/README.md](deferred-feedback/README.md) |

[verification/README.md](verification/README.md) indexes validation evidence.
Evidence supports claims but is not a ninth normative role and does not replace
current specifications, architecture, or roadmap state.

## Precedence

When documents disagree, resolve the conflict in the same scoped change. Until
that happens, use this precedence: specifications, accepted ADRs, current
architecture, roadmap, then implementation or operations guidance. The root
README summarizes public behavior and is not a second product contract.

## Roadmap identity

The sole roadmap namespace is `docs/roadmap/README.md`. Existing `EPIC-NNN`
and `TASK-NNN` identifiers, lifecycle terms, ordering, and completion history
are preserved. Epic and task sequences are independent and monotonic; numbers
are never reused. New allocations increment the current epic maximum of 9 and
task maximum of 29.
Cross-scope qualification is unnecessary while this repository has one delivery
scope.

## Path migration

The 2026-09-23 path migration changed only documentation ownership paths:

| Previous path | Canonical path |
|---|---|
| `docs/requirement.md` | `docs/specs/README.md` |
| `docs/architecture.md` | `docs/architecture/README.md` |
| `docs/plan.md` | `docs/roadmap/README.md` |

Plain references to the previous paths inside completed verification records
remain historical descriptions of the commits they document; they are not
current authorities or redirects.

## Repository checks and unresolved metadata

Run `go run scripts/check-docs.go` for the repository-owned link, anchor,
identifier, and required-owner audit. Packaging checks live under `scripts/`.
The repository currently declares no remote download URL, package-manager
channel, tag history, or project license. Those values require a separate owner
decision and must not be inferred from local release artifacts.
