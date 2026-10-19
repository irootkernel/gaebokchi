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

At least one account is required. Rows appear in the provider and account order
written in `config.yaml` (with `agy` expanding to `gemini` then `claude/gpt` at
its configured position). A missing CLI for a configured account
produces a `NOT INSTALLED` row; an omitted provider launches no CLI.

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
