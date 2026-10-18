# Verification Evidence

This directory contains snapshot- and artifact-bound evidence supporting the
requirements, architecture, roadmap, and release gates. Evidence does not become
current merely because a later tree retains the record.

- [compatibility.md](compatibility.md) records the frozen external protocol
  qualification and sanitized fixture provenance.
- [claude-compatibility.md](claude-compatibility.md) records the later Claude
  local-command and weekly reset-text qualification.
- [agy-compatibility.md](agy-compatibility.md) records the qualified `agy 1.4.0`
  single-account `/usage` JSON metadata contract, version guard (`>= 1.2.15`),
  child environment filter, absent-login fast-fail behavior, sanitized captured
  fixture for the `gemini` and `claude/gpt` weekly pools, and the completed
  `EPIC-012` (`TASK-034` through `TASK-038`) offline and live acceptance evidence.
- [cursor-compatibility.md](cursor-compatibility.md) records the qualified
  `cursor-agent 2026.10.16-e3cb207` subprocess (`status --format json`) +
  read-only `NODE_OPTIONS` preload hook contract, weekly/monthly period
  semantics, `CURSOR_*` environment filter, and sanitized live fixture for
  `EPIC-013` (`TASK-039` through `TASK-041`).
- [provider-extension-preflight-2026-10-16.md](provider-extension-preflight-2026-10-16.md)
  records the 2026-10-16 discovery pass, the 2026-10-17 live `agy 1.4.0`
  metadata qualification, and the 2026-10-18 live `cursor-agent
  2026.10.16-e3cb207` preload-hook qualification and scope alignment.
- [live-acceptance.md](live-acceptance.md) records the owner-account campaign and
  its explicit corrected-candidate limitation.
- `TASK-001.md` through `TASK-028.md` record historical task-scoped checks,
  findings, supersessions, and platform limitations.
- [TASK-029.md](TASK-029.md) records the JSON output checks and task-scoped
  commit subject.
- [TASK-030.md](TASK-030.md) records the interactive terminal view checks,
  fake-provider terminal exercise, and task-scoped commit subject.
- [TASK-031.md](TASK-031.md) records public release documentation and archive
  membership checks, including the pre-commit candidate limitation.
- [TASK-032.md](TASK-032.md) records current YAML and Claude acceptance, including
  live positive-path evidence and the distinct-login qualification limit.
- [TASK-033.md](TASK-033.md) records the current release-candidate review, native
  gates, artifact staging, and the final clean-binary live gate.
- [go-1.27-tooling.md](go-1.27-tooling.md) records the later compiler, lint,
  and local scanner upgrade against its own working-tree snapshot.

The canonical lifecycle state remains in [the roadmap](../roadmap/README.md),
and required behavior remains in [the specifications](../specs/README.md).
