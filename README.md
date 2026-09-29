# Gaebokchi (개복치)

See how much of your weekly Codex (ChatGPT), Grok, and Claude subscription
allowance you have used, in one command, using the logins your installed CLIs
already manage.

```text

  🐡 gaebokchi · weekly usage

  ChatGPT  work                  ███████░░░░░░░░░░░░░     37%
    ↻ 2d 4h
           personal              ██░░░░░░░░░░░░░░░░░░     12%
    ↻ 5d 1h
  Grok     personal               ███████████░░░░░░░░░     54%
    ↻ 3d 11h
  Claude   work                  █████████░░░░░░░░░░░   43.2%
    ↻ 19h 4m

```

(Illustrative output. Account names are the aliases in your configuration.)

## Why

AI coding subscriptions enforce weekly limits, and each provider shows them
somewhere different: a TUI screen, a web page, a slash command. Gaebokchi asks
each installed CLI for its provider-reported weekly usage through a prompt-free
metadata interface and puts the results side by side. It never sends a model
prompt, never reads your credentials, and exits after one snapshot.

Gaebokchi is a small, read-only tool in the Aquarium tool family.

## What it shows, and what it does not

- One row per account in `~/.config/gaebokchi/config.yaml`. Omit a provider
  to disable it; list multiple homes for Codex, Grok, or Claude.
- The **percentage already used** of each provider-reported **weekly
  subscription allowance**. It never shows remaining quota, local token totals,
  API spend, or a calendar-week estimate.
- A reset countdown in the interactive view, when the provider reports a reset
  time.
- One snapshot, then exit. There is no daemon, watch mode, polling, retry,
  history, or cache.
- Read-only at the business level: no model turns, tool actions, login or
  logout, or billing changes. Provider CLIs may still refresh their own tokens,
  logs, or caches when they start; those are provider internals that gaebokchi
  does not touch.

## Requirements

- macOS or Linux on arm64 or amd64. Windows is not supported.
- The provider CLIs you want to see, installed and logged in:
  - `codex` with a ChatGPT subscription login, for each Codex home you query;
  - `grok`, logged in separately for every configured `GROK_HOME`;
  - `claude` (Claude Code), logged in for each configured Claude home.
- Go 1.27 or newer, only if you build from source.

A configured account whose CLI is missing gets a `NOT INSTALLED` row and makes
the exit status 1. Omitted providers are never queried.

## Install

### From GitHub Releases

Each [release](https://github.com/irootkernel/gaebokchi/releases) provides one
archive per platform and a `SHA256SUMS` file.

| System | `uname -sm` prints | Archive |
|---|---|---|
| macOS, Apple silicon | `Darwin arm64` | `gaebokchi-darwin-arm64.tar.gz` |
| macOS, Intel | `Darwin x86_64` | `gaebokchi-darwin-amd64.tar.gz` |
| Linux, 64-bit ARM | `Linux aarch64` | `gaebokchi-linux-arm64.tar.gz` |
| Linux, x86-64 | `Linux x86_64` | `gaebokchi-linux-amd64.tar.gz` |

```sh
VERSION=v0.1.0
TARGET=darwin-arm64    # pick yours from the table above
curl -fLO "https://github.com/irootkernel/gaebokchi/releases/download/$VERSION/gaebokchi-$TARGET.tar.gz"
curl -fLO "https://github.com/irootkernel/gaebokchi/releases/download/$VERSION/SHA256SUMS"
shasum -a 256 -c --ignore-missing SHA256SUMS    # on Linux: sha256sum -c --ignore-missing SHA256SUMS
tar -xzf "gaebokchi-$TARGET.tar.gz"
mkdir -p ~/.local/bin
install -m 0755 "gaebokchi-$TARGET/gaebokchi" ~/.local/bin/gaebokchi
```

The checksum command must print `gaebokchi-<target>.tar.gz: OK`. With
`--ignore-missing` it also succeeds when no listed file is present, so look for
that line. Make sure `~/.local/bin`, or whichever directory you chose, is on
your `PATH`.

Each archive unpacks into `gaebokchi-<os>-<arch>/`, which holds the
`gaebokchi` binary, this README, `LICENSE`, `THIRD_PARTY_NOTICES.txt`, the
account setup guide at `configuration/README.md`, and the maintainer
specification under `docs/`.

The macOS binaries are not notarized. Downloads with `curl` work as shown. If
you download with a browser and macOS blocks the binary, remove the quarantine
attribute with `xattr -d com.apple.quarantine ~/.local/bin/gaebokchi`.

### From source

```sh
git clone --branch v0.1.0 https://github.com/irootkernel/gaebokchi.git
cd gaebokchi
go build -trimpath -ldflags "-X main.buildVersion=v0.1.0" -o gaebokchi ./cmd/gaebokchi
```

Set `buildVersion` only when building the matching tag; otherwise another
commit would claim that version. Without the override, `gaebokchi version`
reports the version that Go records at build time: one derived from the
repository when that information is available, otherwise `(devel)`.

### Uninstall

Delete the `gaebokchi` binary and, if no longer needed,
`~/.config/gaebokchi/config.yaml`. Gaebokchi keeps no cache or usage history.

## Configuration

For a step-by-step setup and a Claude-only example, see the
[configuration guide](configuration/README.md).

Create `~/.config/gaebokchi/config.yaml` before running a usage query:

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
```

Each `name` is a display alias, not a login selector or a verified email. Each
`path` selects the CLI-managed account state for that row: `CODEX_HOME` for
Codex, `GROK_HOME` for Grok, and `CLAUDE_CONFIG_DIR` for non-default Claude
homes. The default Claude home (`~/.claude`) uses the CLI's implicit login
state. Log in through the matching provider CLI before querying.
For example, `CODEX_HOME="$HOME/.codex-work" codex login` and
`CLAUDE_CONFIG_DIR="$HOME/.claude-work" claude auth login` select those homes;
use the Grok CLI's own login flow with `GROK_HOME="$HOME/.grok-personal"`.
Gaebokchi never reads or creates provider credentials.

Use an absolute path or a path beginning with `~/`; shell variables and
relative paths are not expanded. `~/` is expanded without changing the filesystem
meaning of `.` or `..` across symlinks. Missing or inaccessible directories keep
an error row. Names and paths must be unique within each provider, including
existing symlink aliases, even when a trailing path component is missing. A
blank or whitespace-only name, unknown YAML key, YAML merge key, null provider list, or empty
configuration is an error. Providers omitted from the file (or listed as `[]`)
are disabled; at least one account must remain. Rows appear in Codex,
Grok, Claude order, then in each provider's list order.

If the file is missing or invalid, gaebokchi exits 2 before starting a
provider and prints the path plus a link to the configuration guide. `--help` and
`version` work without the file.

## Usage

```sh
gaebokchi                                            # configured accounts
gaebokchi --timeout 40s                              # longer query budget
gaebokchi --grok-bin /opt/homebrew/bin/grok
gaebokchi --claude-bin /opt/homebrew/bin/claude
gaebokchi --json                                     # machine-readable snapshot
NO_COLOR=1 gaebokchi                                 # terminal view without color
gaebokchi version
gaebokchi version --json
```

### Flags

| Flag | Default | Meaning |
|---|---|---|
| `--codex-bin PATH` | `codex` | codex executable path or name |
| `--grok-bin PATH` | `grok` | grok executable path or name |
| `--claude-bin PATH` | `claude` | claude executable path or name |
| `--timeout DURATION` | `20s` | Per-target query budget, from 1s to 120s |
| `--json` | off | Emit one JSON usage snapshot instead of the terminal view or table |
| `-h`, `--help` | — | Print help and exit without launching providers |

Flags accept `--flag value` or `--flag=value`. Ambient `CODEX_HOME`,
`GROK_HOME`, and `CLAUDE_CONFIG_DIR` do not change the configured account list.

The following are usage errors, which exit 2 and print nothing on stdout:

- unknown flags and positional arguments;
- repeating a flag;
- a missing or empty value;
- an invalid or out-of-range `--timeout`;
- a missing or invalid configuration file. The diagnostic links to the
  [configuration guide](configuration/README.md).

`gaebokchi version` prints `gaebokchi <version>`, and `gaebokchi version --json`
prints `{"name":"gaebokchi","version":"<version>"}`. Neither launches a
provider. The former `--version` flag is not supported.

## Output

### Interactive terminal view

This view is used when stdout is a terminal with a readable size and `TERM` is
set to something other than `dumb`. Every target appears at once with a
spinner and `querying…`, and each row fills in as its result arrives. The final
frame, like the one at the top of this page, stays in your scrollback.

- **Colors.** Bars are green below 50%, yellow from 50%, and red from 80%.
- **Countdowns.** A reset countdown appears only when the provider reported a
  reset time. Countdowns sit at the ends of the rows when the terminal is wide
  enough for every countdown next to a full 20-cell bar; otherwise each goes on
  its own line below its row.
- **Short terminals.** If the terminal is too short for the frame, only the
  final frame is drawn, once.
- **`NO_COLOR`** removes the colors but keeps the layout.

### Plain table

Pipes, redirects, `TERM=dumb`, and terminals without a readable size get a
plain table with one line per target:

```text
AI PROVIDER  ACCOUNT               WEEKLY USAGE
ChatGPT      work                  37%
ChatGPT      personal              12%
Grok         personal              54%
Claude       work                  43.2%
```

### JSON

`--json` writes one newline-terminated object with a `results` array in
display order, whatever stdout is. For brevity, this example shows a single
result; real output includes every target, failures too:

```json
{"results":[{"target_id":"codex:0","provider":"codex","account":"work","status":"ok","used_percent":37.125,"cleanup_failed":false}]}
```

Each result has exactly these fields:

- `target_id`;
- `provider` (`codex`, `grok`, or `claude`);
- `account`;
- `status`;
- `used_percent`, a number with the provider's precision, or `null` when no
  percentage exists. The Grok omitted-percentage convention is `0`;
- `cleanup_failed`.

`status` is one of `ok`, `not_applicable/monthly`, `not_applicable/api_key`,
`unavailable/no_weekly`, `unavailable/period`, `unavailable/no_data`,
`auth_required`, `unsupported`, `cli_not_found`, `home_not_found`,
`home_error`, `timeout`, `protocol_error`, `provider_error`, or `cancelled`.
Diagnostics stay on stderr, and the exit codes are the same as for the other
outputs. Help output stays text.

## Statuses and exit codes

| WEEKLY USAGE | Meaning |
|---|---|
| `37%`, `43.2%` | Weekly allowance used; whole values have no decimal, others show one decimal place |
| `0%` on a Grok row | Grok reported a weekly period but omitted the percentage; shown as zero by convention, not a measured zero |
| `N/A (monthly)` | The account has a monthly allowance, which is not weekly usage |
| `N/A (API key)` | API-key authentication has no subscription allowance |
| `N/A (no weekly)` | The provider's quota data has no weekly window |
| `N/A (period)` / `N/A (no data)` | The period or the usage value is missing |
| `AUTH REQUIRED` | The provider CLI reports no active login |
| `UNSUPPORTED` | The installed CLI lacks the qualified interface, its output changed, or its login mode is not supported, for example a Claude login routed through Bedrock or Vertex |
| `NOT INSTALLED` | The executable was not found |
| `HOME NOT FOUND` / `HOME ERROR` | A configured provider directory is missing or inaccessible |
| `TIMEOUT` | The query did not finish within `--timeout` |
| `PROTOCOL ERROR` | The provider answered in a shape gaebokchi does not accept |
| `ERROR` | Any other failure, including a failed authentication check |
| `CANCELLED` | Interrupted by SIGINT or SIGTERM |

| Exit code | Meaning |
|---|---|
| `0` | Every row is a percentage, `N/A (monthly)`, or `N/A (API key)`; also for help and version |
| `1` | Any other row status, a provider process or temporary directory that could not be cleaned up, or stdout that could not be written |
| `2` | Invalid arguments or configuration; no usage output |
| `130` / `143` | Cancelled by SIGINT / SIGTERM |

Rows are kept even when their query fails, so one broken provider never hides
the others. All account labels come from configuration and are not verified emails.

## Privacy and side effects

- **Credentials** stay with the provider CLIs. Gaebokchi never reads auth
  files, tokens, or keychains, and never implements OAuth.
- **Environment.** Each provider runs as a child process with a filtered
  environment:
  - The API-key overrides `OPENAI_API_KEY`, `CODEX_API_KEY`, `XAI_API_KEY`,
    and `GROK_API_KEY` are removed.
  - For Claude, Anthropic credential variables such as `ANTHROPIC_API_KEY` and
    `ANTHROPIC_AUTH_TOKEN` are removed, together with the switches that route
    Claude through Bedrock, Vertex, or another provider.
  - Codex and Grok children get their configured `CODEX_HOME` or `GROK_HOME`.
    Claude children get `CLAUDE_CONFIG_DIR` for non-default homes; the default
    `~/.claude` home uses the CLI's implicit credential namespace. `HOME` and
    proxy settings are kept.
- **Working directories.** Children run in private temporary directories
  outside your project, and those directories are removed afterwards.
- **Output.** stdout carries the view, table, or JSON, including your account
  identifiers. Diagnostics go to stderr as fixed, application-owned messages.

## Performance

Each target has its own `--timeout` budget, and at most three targets run at
once. Additional targets queue until a slot is free. A run against slow providers
can therefore take multiple timeout intervals, plus bounded cleanup.

## Compatibility

Gaebokchi drives provider interfaces that are qualified per CLI version:

| Provider | Tested CLI | Interface |
|---|---|---|
| Codex | `codex-cli 0.155.1` and `0.158.0` | `codex app-server` account and rate-limit reads |
| Grok | `grok 1.0.40` and `1.0.44`; plus a `1.0.41` metadata probe | The `grok --no-auto-update agent stdio` billing extension |
| Claude | Claude Code 2.1.280 and 2.1.284; reset text on 2.1.283 | The prompt-free `auth status` read and the `/usage` local command |

The later positive-path qualification used the current configured-account binary
with two Codex homes, one Grok home, and one Claude home on macOS/arm64;
see [TASK-032](docs/verification/TASK-032.md). Distinct real Grok or Claude
logins across multiple homes have not been qualified.

- **Offline test suites.** The current code's suites ran natively on
  macOS/arm64 and Linux/arm64 (containerized) during
  [TASK-032](docs/verification/TASK-032.md), including the fake-provider E2E suite.
- **Other targets** are cross-compiled.

Newer CLI versions may change these interfaces. When that happens, gaebokchi
reports `UNSUPPORTED` or `PROTOCOL ERROR` rather than guessing.

## Known limitations

- The Claude row depends on the human-readable `Current week (all models)` line
  of Claude Code's `/usage` output. If a CLI update changes that line, the row
  shows `UNSUPPORTED` until gaebokchi is qualified against the new version.
- A Grok weekly response without a percentage is shown as `0%`, which is a
  display convention, not a measured zero.
- Multi-account Grok and non-default Claude homes depend on the installed CLI
  honoring `GROK_HOME` and `CLAUDE_CONFIG_DIR` respectively. Different aliases alone
  do not establish different logged-in identities.
- Bars can misalign in terminals configured to draw East Asian Ambiguous
  characters at double width. Narrowing the window during a live redraw can
  leave reflowed lines above the final frame.
- Claude reports its reset time to the minute and without a UTC offset, so its
  countdown can reach `now` up to a minute early. Reset text that does not name
  exactly one instant, such as a time repeated by a daylight-saving change,
  shows no countdown.
- A monthly Grok allowance is reported as `N/A (monthly)`, never converted to
  weekly usage.

## Troubleshooting

- **`AUTH REQUIRED`**: log in with the provider CLI using the matching
  configured home (`codex login`, `grok`, or `claude auth login`), then rerun.
- **`NOT INSTALLED`**: install the CLI, or point `--codex-bin`, `--grok-bin`, or
  `--claude-bin` at it.
- **`HOME NOT FOUND` / `HOME ERROR`**: check the path, or pass the homes you
  configured in `config.yaml`. Gaebokchi never creates a home directory.
- **`UNSUPPORTED` / `PROTOCOL ERROR`**: gaebokchi does not accept the installed
  CLI's version, output, or login mode. Check the compatibility table above
  before upgrading or downgrading anything. For Claude, confirm that
  `claude auth status` reports a Claude subscription login.
- **`TIMEOUT`**: raise `--timeout`; quota reads go over the network and can be
  slow.
- **`ERROR`**: the provider CLI failed in a way gaebokchi does not classify.
  Run the provider CLI on its own to see its messages; gaebokchi deliberately
  does not print raw provider output.
- **Exit 1 although every row shows a value**: a provider process or temporary
  directory could not be cleaned up. The stderr line names the target.
- **`N/A` rows**: these report what the provider said, such as a monthly plan
  or an API-key login. Nothing needs fixing.

## Development

Maintainer and contributor documentation starts at
[docs/README.md](docs/README.md). [TESTING.md](TESTING.md) describes the
offline test gate (`make test`) and its fake-provider E2E environment. These
links work in the source repository; release archives include this README, the
[configuration guide](configuration/README.md), and the specification documents
under `docs/`.

## License

Gaebokchi is released under the MIT License. Copyright (c) 2026 Root Kernel.
See [LICENSE](LICENSE) in the source repository or the release archive.

Release binaries also contain the Go standard library and the
`go.yaml.in/yaml/v3`, `golang.org/x/sys`, and `golang.org/x/text` modules.
Those components retain their own licenses; the YAML module includes MIT and
Apache-licensed code, while the Go components use BSD-style licenses. Their
license and notice texts are in [THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt)
in the source repository and each release archive.
