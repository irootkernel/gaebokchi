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

## Weekly reset text

Observed on 2026-09-27 on macOS/arm64 with Claude Code 2.1.283, through the
same prompt-free usage read. When the server reports a reset time, each meter
line, including the all-models weekly line, ends with a reset suffix:

```text
Current week (all models): N% used · resets Oct 3 at 10am (America/Los_Angeles)
```

The installed CLI builds this suffix from the usage endpoint's `resets_at`
with a fixed en-US formatter. It gives the short month and day, then a comma
and a four-digit year only when that year differs from the current year in
the named zone. The time is 12-hour with lowercase `am`/`pm`, and the minutes
are omitted on the hour. The runtime's IANA time zone follows in parentheses.
Seconds are dropped, so the text has minute resolution.

The live read joined the date and time with ` at `. Bun 1.4.2 and
JavaScriptCore reproduce that form, including `Jan 2, 2027 at 5pm`. The same
formatter under Node 26 joins them with a comma instead (`Jan 2, 2027, 5pm`).
Gaebokchi accepts either join. The observed percentages and times were masked
and are not stored here.

Gaebokchi reads only the suffix of the all-models weekly line and keeps the
reset only when the text names exactly one instant. The text carries no UTC
offset. So the reset stays unset, without affecting the percentage, when the
suffix is missing or unrecognized, when the host cannot load the zone, when
the date does not exist, or when a daylight-saving change skips or repeats the
printed local time. It also stays unset when the text omits the year and the
zone's year changed while the usage command ran. The CLI also attaches a
structured `usage_report` to `stream-json` output. Its schema is marked
experimental, so gaebokchi does not use it, and the qualified command line is
unchanged.
