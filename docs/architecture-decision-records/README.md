# Architecture Decision Records

This directory owns durable records of accepted, superseded, deprecated, and rejected architecture decisions. Current decisions that need no separate rationale stay in the [architecture](../architecture/README.md), starting with ARCH-01.

| ADR | Decision | Status |
|---|---|---|
| [ADR-0001](ADR-0001-claude-usage-local-command.md) | Read Claude usage through the CLI's `/usage` local command | Accepted |
| [ADR-0002](ADR-0002-period-aware-provider-usage.md) | Preserve native quota periods, use JSON v2 and retain CLI credential ownership | Accepted design; implementation pending in EPIC-012 and EPIC-013 |

Create the next numbered ADR, named `ADR-NNNN-short-title.md`, when a decision needs alternatives, rationale, or supersession history that should outlive an architecture rewrite.
