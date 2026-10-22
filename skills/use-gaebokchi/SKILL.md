---
name: use-gaebokchi
description: Query subscription usage through the Gaebokchi CLI, configure its accounts and provider aliases, and diagnose setup or snapshot failures. Use for operating Gaebokchi; developing its source alone does not activate this skill.
---

# Use Gaebokchi

Gaebokchi takes one subscription-usage snapshot through installed provider CLIs.
It has no daemon, watch mode, polling, history, or cache. Provider CLIs own
credentials and network access; they may refresh their own tokens, logs, or
caches during a query.

## Establish the executable and requested scope

Resolve `command -v gaebokchi`. Use that executable's `version --json` and
`--help` to establish its available commands when entering the workflow or
diagnosing a mismatch. Both work without configuration and launch no providers.
Use `version`, not `--version`.

Distinguish the installed executable from a source checkout or temporary build.
Source documentation does not prove that an older installation supports a flag
or configuration field. For configuration changes, consult documentation or
source matching the executable as well as its help. If the needed capability is
absent or unconfirmed, report it before changing the working configuration;
use an already authorized suitable build or obtain direction for an upgrade.
Never silently replace the installed binary or downgrade the requested behavior.

A usage request authorizes the requested snapshot; proceed without asking again.
A setup inspection authorizes offline inspection, not a live account query.
Configuration edits, installation, provider login changes, and publication have
their own scopes. Carry existing authorization forward without adding those
effects to a query.

## Take one snapshot

Use `--json` for agent inspection and retain stdout, stderr, and exit status
separately. The ordinary command queries every enabled configured account:

```sh
gaebokchi --json
```

When the executable supports target selection, select the requested provider
through its default name or an existing alias:

```sh
gaebokchi -t chatgpt --json
gaebokchi --target zcode --json
```

Default selection names are `chatgpt`, `grok`, `claude`, `agy`, `cursor`, and
`zcode`. The configuration key and JSON provider ID for ChatGPT are `codex`;
`codex` is a selection name only when configured as an alias. Match whole strings
case-insensitively. Do not trim, expand prefixes, or turn comma lists into targets.
The shared `-t` / `--target` option may appear only once.

Selection queries every account of every matching enabled provider, including
both `agy` pools. A shared alias, including one colliding with a default name,
selects the union once. Check aliases before assuming a name selects only one
provider. Without target support, do not broaden a provider-only request into
an all-provider query or rewrite the configuration to simulate selection.

Use the observed help for binary overrides and timeout options. The current
`--timeout` accepts 1s through 120s, default 20s. A snapshot finishes once;
do not start a watch loop or automatically retry failed providers.

## Interpret and report the result

The current JSON contract is an object with `results`, an ordered array whose
rows contain `target_id`, `provider`, `account`, `status`, `used_percent`, and
`cleanup_failed`. Do not invent JSON period, reset, or observation-time fields.
Use documentation matching the executable if its contract differs.

Report the percentage **already used**: weekly for Codex, Grok, Claude, `agy`,
and Cursor; rolling 5-hour for ZCode. Keep accounts and periods separate; these
percentages cannot be summed into a shared allowance. A null percentage is
unavailable, never zero. Grok may report zero by convention when its weekly
percentage is omitted, so its zero alone does not prove measured zero usage.
Account labels come from configuration and do not verify a login identity.

| Exit | Interpretation |
| --- | --- |
| `0` | Rows are successful or not applicable (monthly / API key); inspect each status. |
| `1` | Incomplete rows, process or temporary-directory cleanup failure, or output failure. Preserve any successful rows. |
| `2` | Invalid arguments or configuration; no usage output. |
| `130` / `143` | Interrupted by SIGINT / SIGTERM. |

Check `cleanup_failed` even when a row has a percentage. Keep safe stderr
diagnostics with failures. Show only account labels needed for the requested
result; redact real account identifiers from setup reports and shared artifacts.
Do not paste raw configuration, credentials, environment dumps, or provider logs.

## Configure accounts and aliases

The configuration is `~/.config/gaebokchi/config.yaml`; there is no native
`--config`, `doctor`, or validation-only command in the current CLI. Running
`gaebokchi --json` performs real queries and is not an offline validation check.

Read only the configuration needed for the requested edit. Preserve existing
accounts, provider/account order, aliases, comments, and permissions. Use a
private temporary backup for a real configuration edit, check for concurrent
changes, and replace atomically after applicable offline checks. A YAML parse
alone does not establish Gaebokchi's semantic validity. Report the verification
limit when no authorized live check or matching source-level check is available.

- `codex`, `grok`, and `claude` accept account lists with `name` and `path`.
  Paths must be absolute or start with `~/`; shell variables are not expanded.
  Names and paths must be unique within a provider, including symlink aliases.
- `agy`, `cursor`, and `zcode` each support one default login. `agy` expands to
  two weekly pool rows. `cursor` and `zcode` reject `path`; an optional `agy.path`
  does not select another login. Do not invent account-isolation variables.
- Omitted providers or `[]` disable them. Null lists fail; at least one account
  is required. The entire configuration is validated before target selection,
  so invalid unselected data still blocks a query.
- On versions supporting aliases, merge into the optional top-level `aliases`
  map using provider keys and nonblank string lists. For example, adding
  `codex: [codex]` under `aliases` enables that selection spelling. Aliases do
  not enable disabled providers or change account labels. Unknown keys, duplicate
  YAML keys, merge keys, and multiple YAML documents fail validation.

For the complete schema, use the matching distribution's `configuration/README.md`
or the source checkout's guide. The
[upstream configuration guide](https://github.com/irootkernel/gaebokchi/blob/main/configuration/README.md)
tracks main and may be newer than the installed executable.

## Diagnose within the provider boundary

For `cli_not_found`, inspect the executable path. For `home_not_found` or
`home_error`, inspect only the configured directory's existence and access.
For `auth_required`, refer the user to the owning provider CLI's login flow.
For `unsupported` or protocol failures, compare installed provider versions with
Gaebokchi's matching compatibility documentation; do not claim qualification
from a version string or fixture alone.

Do not read token files or Keychain entries, perform token exchange, call quota
HTTP endpoints directly, scrape a provider TUI, or send a model prompt as a
fallback. In particular, prompt-mode `/usage` is not a substitute for a qualified
metadata interface. Report the observed failure and the bounded next action.
