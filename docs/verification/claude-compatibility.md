# Claude Code usage compatibility

Observed on 2026-09-23 on macOS/arm64 with Claude Code 2.1.280. This is a
separate extension to the historical v0.1 Codex/Grok qualification.

The installed CLI accepted the following prompt-free reads from a private
temporary working directory with API-key overrides removed:

```text
claude --safe-mode auth status --json
claude --safe-mode --tools '' -p /usage --max-turns 0 --output-format json --no-session-persistence
```

The auth read returned `loggedIn: true`, `authMethod: "claude.ai"`, and
`apiProvider: "firstParty"`. The usage read exited 0 with a JSON result
containing `local_command: "usage"`, `num_turns: 0`, `total_cost_usd: 0`, and
`is_error: false`. Its text result contained separate current-session,
current-week-all-models, and model-specific weekly lines. Only the
all-models weekly percentage is selected. No account identifier, credential,
or measured percentage is stored in this record.

The [Claude command reference](https://code.claude.com/docs/en/commands)
documents `/usage` as an in-session command, while the
[CLI reference](https://code.claude.com/docs/en/cli-reference) documents
print mode, JSON output, safe mode, and zero-turn limits. The weekly line is
human-readable CLI output, not a published structured quota schema. If its
wording or JSON envelope changes, gaebokchi reports `UNSUPPORTED` rather
than interpreting another meter as weekly usage. The
[environment reference](https://code.claude.com/docs/en/env-vars)
documents that `ANTHROPIC_API_KEY` overrides a subscription in print mode.

Offline tests use a fake Claude executable. No real Claude CLI acceptance
was run on Linux in this qualification.
