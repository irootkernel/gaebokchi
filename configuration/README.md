# Configure Gaebokchi accounts

Gaebokchi reads `~/.config/gaebokchi/config.yaml` on each usage run. Create it
before running `gaebokchi`:

```sh
mkdir -p ~/.config/gaebokchi
```

Add only the accounts you want to query. For example:

```yaml
codex:
  - name: work
    path: ~/.codex-work
  - name: personal
    path: ~/.codex-personal
grok:
  - name: personal
    path: ~/.grok-personal
claude:
  - name: work
    path: ~/.claude-work
  - name: personal
    path: ~/.claude-personal
agy:
  - name: default
cursor:
  - name: default
zcode:
  - name: default
```

Save that YAML as `~/.config/gaebokchi/config.yaml`. The example names and paths
are placeholders; use the provider homes where you have logged in. The file
contains display aliases and paths, not passwords or tokens.

## Account paths and logins

Each `codex`, `grok`, and `claude` entry needs a `name` and a `path`. `name` is
the label shown in the table and JSON output. It does not select or verify a
provider login. `path` selects the CLI-managed account state for that row.
`agy` accepts at most one entry (such as `- name: default`, with no `path`
required) and queries the single default `agy` login once to display two weekly
model-pool rows (`gemini` and `claude/gpt`). An optional `agy.path` is validated
as a path but does not select a login or change the default credential home.
`cursor` accepts at most one entry
(such as `- name: default`, with no `path` field allowed) and queries the single
default `cursor-agent` login once to display its weekly `Included` subscription
usage row. `zcode` accepts at most one entry (such as `- name: default`, with no
`path` field allowed) and queries the single default `zcode` login once to
display its 5-hour Coding Plan subscription usage row in the `5h usage` section:

| List | Child environment variable | Login state |
|---|---|---|
| `codex` | `CODEX_HOME` | Codex CLI home |
| `grok` | `GROK_HOME` | Grok CLI home |
| `claude` | `CLAUDE_CONFIG_DIR` for other homes | Claude Code configuration directory |
| `agy` | Preserves `HOME` (`AGY_CLI_DISABLE_AUTO_UPDATE=true`) | Single default Google Antigravity CLI login (`gemini` and `claude/gpt` weekly pools) |
| `cursor` | Preserves `HOME` (controlled `NODE_OPTIONS` preload hook) | Single default Cursor CLI login (`cursor-agent`) |
| `zcode` | Preserves `HOME` (controlled `NODE_OPTIONS` preload hook) | Single default ZCode CLI login (`zcode`) with a stored Coding Plan API key |

Log in separately through each provider CLI with the matching variable. For
example:

```sh
CODEX_HOME="$HOME/.codex-work" codex login
CLAUDE_CONFIG_DIR="$HOME/.claude-work" claude auth login
```

For Grok, set `GROK_HOME` to the configured path while using the installed
Grok CLI's login flow. For `agy` and `cursor`, log in once through the
installed `agy` or `cursor-agent` CLI. For `zcode`, ensure the active Coding
Plan account has a stored Coding Plan API key (`:api-key`) in
`~/.zcode/v2/credentials.json` (by configuring a Coding Plan API key through the
`zcode` CLI outside `gaebokchi`; an OAuth-only `zcode login` without a stored
Coding Plan key reports `AUTH REQUIRED` because `gaebokchi` never issues
token-exchange or key-provisioning `POST` requests).
Gaebokchi does not create provider homes, read credentials, or perform logins.
For Claude, it removes an inherited `CLAUDE_CODE_OAUTH_TOKEN` from the child
environment so the configured login state is used. The default `~/.claude` home
uses the CLI's implicit credential namespace; other Claude homes set
`CLAUDE_CONFIG_DIR` and need their own login. For Cursor and ZCode, it removes
inherited `CURSOR_*`, `ZCODE_*`, `ZAI_*`, and `BIGMODEL_*` overrides and any
ambient `NODE_OPTIONS` so the default stored login is used.

## Choose which providers run

Omit a provider or set it to `[]` to disable it. A present key with a null
value, such as `grok:`, is an error. For example, this file queries
only two Claude accounts:

```yaml
claude:
  - name: work
    path: ~/.claude-work
  - name: personal
    path: ~/.claude-personal
```

At least one account is required. Rows follow the provider and account order
written in `config.yaml` within each displayed usage period, with `agy`
expanding to `gemini` and `claude/gpt` at its configured position. JSON retains
one flat array in configured target order. A missing CLI for a queried account
produces a `NOT INSTALLED` row; an omitted provider launches no CLI.

## Provider aliases and target selection

An optional top-level `aliases` mapping adds extra selection names for
providers. Configured accounts, their display labels and provider order stay
unchanged:

```yaml
aliases:
  codex: [codex]
  cursor: [grok]
```

The always-valid default selection names are `chatgpt` (the Codex-backed
provider), `grok`, `claude`, `agy`, `cursor`, and `zcode`. There is no implicit
`codex` alias; add `aliases: {codex: [codex]}` to select the Codex-backed
provider with that spelling. Names and aliases match case-insensitively as
whole strings; gaebokchi never trims spaces or expands prefixes, substrings, or
comma lists. The defaults and the configured aliases together select every
matching enabled provider once, and a default name has no priority over an
alias. Omit `aliases`, use `{}`, or use an empty per-provider list to add
nothing. Alias entries for disabled providers are allowed but never enable
them, and an alias-only configuration still fails the at-least-one-account
requirement. Unknown provider keys, non-string or blank aliases, and invalid
mapping or list shapes fail validation before any provider starts.

Gaebokchi validates the entire configuration before applying a selection:
invalid data in an unselected provider still fails the run with exit 2. Use
`-t NAME`, `-t=NAME`, `--target NAME`, or `--target=NAME` to query only the
providers whose default name or alias matches NAME, through every configured
account of each (both `agy` pools included). The flag may be supplied once per
invocation. All output modes keep their existing schema, and unselected
providers launch nothing and add no rows. Without the flag, every configured
account is queried as before. In the example above, `-t ChatGPT` and `-t CODEX`
both select the Codex accounts, and `-t grok` selects Grok and Cursor. An
empty, whitespace-only, repeated, or unmatched value exits 2 before any
provider starts.

## Check the file

Run `gaebokchi --json` for one machine-readable usage snapshot or `gaebokchi`
for the terminal view or plain table. The reported value is the percentage of
the provider's weekly (or 5-hour, for `zcode`) subscription allowance already
used, not a raw token count.

Paths must be absolute or start with `~/`. Shell variables and other `~user`
forms are not expanded. `~/` is expanded before use; `.` and `..` retain their
filesystem meaning, including when a path traverses a symlink. Names and paths
must be unique within each provider; existing symlink aliases count as the same
path, including missing components beneath an existing symlink. Missing paths
under the same existing parent also reserve case-folded, canonically equivalent
names on every platform. On a case-sensitive filesystem, two such paths can be
configured separately after both directories exist. Empty names or paths,
unknown YAML keys, YAML merge keys (`<<`), and multiple YAML documents are
rejected. Missing or inaccessible provider directories produce error rows
instead of being created.

If the file is missing or invalid, Gaebokchi exits with code 2 before starting
any provider. The diagnostic identifies the configuration path. Help and
version work without the file (`gaebokchi --help`, `gaebokchi version`).
