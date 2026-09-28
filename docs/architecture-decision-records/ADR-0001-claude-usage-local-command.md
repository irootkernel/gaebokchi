# ADR-0001: Read Claude usage through the CLI's `/usage` local command

- **Status:** Accepted. Implemented on 2026-09-23 with the Claude extension; reset-text handling followed the 2026-09-27 qualification. Recorded retroactively on 2026-09-28.
- **Supersedes:** the v0.1 decision "Codex and Grok only; no Claude" that [ARCH-01](../architecture/README.md#arch-01) carried until 2026-09-28.
- **Related:** [ARCH-16](../architecture/README.md#arch-16), [REQ-056](../specs/README.md#req-056) through [REQ-059](../specs/README.md#req-059), and the [Claude compatibility record](../verification/claude-compatibility.md).

## Context

The v0.1 plan deliberately excluded Claude; its roadmap still shows that historical scope. Afterwards, Claude's weekly subscription usage was added beside the Codex and Grok rows. The existing boundaries still applied: no credential handling ([REQ-037](../specs/README.md#req-037)), no model prompts or other business-level mutation ([REQ-038](../specs/README.md#req-038)), no PTY or UI scraping, and no application-owned provider API client.

The installed Claude CLI exposes the weekly allowance in two places: the text of its built-in `/usage` command, and a structured usage report attached to `stream-json` output whose schema is marked experimental.

## Decision

Query the active Claude CLI login with two prompt-free commands. The first is `claude --safe-mode auth status --json`, which must establish a first-party `claude.ai` subscription login. The second is `claude --safe-mode --tools "" -p /usage --max-turns 0 --output-format json --no-session-persistence`. Accept its output only when the JSON envelope reports the `usage` local command with zero turns and zero cost, and read only the `Current week (all models)` line. Keep a reset time only when the line's reset text names exactly one instant.

## Alternatives not adopted

- **The experimental `stream-json` usage report.** It is structured, but its schema is marked experimental, so it is a weaker compatibility contract than the documented local command.
- **Reading Claude credentials or calling Anthropic APIs directly.** This would move credential handling into gaebokchi, which [REQ-037](../specs/README.md#req-037) forbids.
- **Driving the interactive TUI through a PTY.** Screen scraping is out of scope and would risk starting a session.

## Consequences

- The weekly line is human-readable CLI output. A wording or envelope change produces `UNSUPPORTED` rather than a guessed value, and each new CLI version needs its own qualification record ([ARCH-12](../architecture/README.md#arch-12)).
- The reset text has minute resolution and no UTC offset. Some texts cannot be resolved to one instant and show no countdown, and a countdown can reach `now` up to a minute early.
- The extension was implemented without a roadmap task, so REQ-056 through REQ-059 stay unchecked until a task adopts their acceptance ([deferred feedback](../deferred-feedback/README.md)).
