# Gaebokchi (개복치)

A small, read-only command-line tool in the Aquarium ecosystem that shows
**weekly subscription usage** for your Codex (ChatGPT), Grok, and Claude
accounts in one shot, reusing the logins managed by their installed CLIs.

On an interactive terminal, rows appear as queries finish and the command
leaves this kind of final frame:

```text

  🐡 gaebokchi · weekly usage

  ChatGPT  account1@example.com  ███████░░░░░░░░░░░░░     37%
    ↻ 2d 4h
           account2@example.com  ██░░░░░░░░░░░░░░░░░░     12%
    ↻ 5d 1h
  Grok     default               ███████████░░░░░░░░░     54%
    ↻ 3d 11h
  Claude   default               █████████░░░░░░░░░░░   43.2%

```

Pipes and redirects keep the plain table:

```text
AI PROVIDER  ACCOUNT                WEEKLY USAGE
ChatGPT      account1@example.com   37%
ChatGPT      account2@example.com   12%
Grok         default                54%
Claude       default                43.2%
```

(Illustrative output; stdout can contain your real account identifiers.)

## What it does (and does not do)

- Queries four targets by default: the Codex homes `~/.codex` and
  `~/.codex-hsy`, the existing Grok login, and the active Claude CLI login.
- Shows the percentage already consumed in the provider-reported
  **weekly subscription quota**. It never shows remaining quota, local
  token totals, API spend, or a calendar-week guess.
- When Grok reports a weekly period but omits `creditUsagePercent`, displays
  `0%` under a Grok compatibility policy. This is a display fallback,
  not a measured zero; an explicit `null` percentage remains `N/A (no data)`.
- One snapshot, then exit. Terminal progress does not re-query. No daemon,
  watch mode, polling, retries, history, or cache.
- Read-only on the business level: no model turns, tool actions,
  login/logout, or billing mutations are issued. Provider
  startup itself may refresh tokens, update provider logs/caches, or use
  a provider-managed shared process — those are provider internals
  gaebokchi deliberately does not touch.
- Claude uses the CLI's `/usage` local command in print mode. Its weekly
  all-models line is parsed only after the CLI confirms zero model turns
  and zero cost. This human-readable line can change across CLI versions.

## Prerequisites

- The `codex` CLI installed and logged in (ChatGPT subscription) for
  each home you query.
- The `grok` CLI installed and logged in (`grok` manages its own cached
  token in `~/.grok`).
- The `claude` CLI installed and logged in through a Claude subscription.
- Go 1.27 or newer is needed only to build from source; release binaries
  are self-contained (`CGO_ENABLED=0`).

Protocol compatibility was qualified with live, prompt-free queries
against `codex-cli 0.155.1` and `grok 1.0.40 (stable)` on macOS/arm64.
The Claude local-command path was checked against Claude Code 2.1.280 on
macOS/arm64; see its [compatibility record](docs/verification/claude-compatibility.md).
In a source checkout, see the
[compatibility record](docs/verification/compatibility.md) for the frozen
protocol evidence. Separately, the offline application
and fake-provider test suites also ran natively on Linux/arm64
(containerized) — no real provider CLI was exercised on Linux.
Compatibility is capability-driven:
newer CLI versions may change wire contracts, in which case rows will
report `UNSUPPORTED` or `PROTOCOL ERROR` rather than guess.

## Install / build

From a locally produced release archive: unpack it and put `gaebokchi` on
your `PATH`.

From source:

```sh
go build -trimpath -ldflags "-X main.buildVersion=v0.1.0" -o gaebokchi ./cmd/gaebokchi
```

This repository does not yet declare a remote download URL, package-manager
channel, or project license. The documented distribution paths are therefore
local archives and source builds only; do not infer publication from the
presence of build scripts or verification records.

## Usage

```sh
gaebokchi                                            # default four targets
gaebokchi --timeout 40s                              # longer query budget
gaebokchi --codex-home ~/.codex-work                 # one custom home (replaces defaults)
gaebokchi --codex-home ~/.codex-work --codex-home ~/.codex-personal
gaebokchi --grok-account personal --grok-bin /opt/homebrew/bin/grok
gaebokchi --claude-bin /opt/homebrew/bin/claude
gaebokchi --json                                    # machine-readable usage snapshot
NO_COLOR=1 gaebokchi                                # terminal layout without color
gaebokchi version
gaebokchi version --json
```

Release builds print `gaebokchi v0.1.0`. Source builds without an explicit
version override print `gaebokchi (devel)`.

### Flags

| Flag | Default | Meaning |
|---|---|---|
| `--codex-home PATH` | `~/.codex` and `~/.codex-hsy` | Repeatable; supplying any replaces both defaults |
| `--codex-bin PATH` | `codex` | codex executable path or name |
| `--grok-bin PATH` | `grok` | grok executable path or name |
| `--claude-bin PATH` | `claude` | claude executable path or name |
| `--grok-account LABEL` | `default` | Display alias for the Grok row (not an email) |
| `--timeout DURATION` | `20s` | Per-target query budget, 1s–120s |
| `--json` | off | Emit the usage snapshot as JSON |
| `-h`, `--help` | — | Print help without launching providers |

`version [--json]` is a project-independent command and never launches a
provider. The former `--version` flag is not supported.

For a usage query, `--json` writes one newline-terminated object with a
`results` array in display order. Each result has `target_id`, `provider`
(`codex`, `grok`, or `claude`), normalized `account`, `status`, numeric
`used_percent` (or `null` when unavailable), and boolean `cleanup_failed`.
Measured numeric values retain provider precision and have no `%` suffix.
The Grok weekly omission fallback is numeric zero. For example, a successful
row may look like:

```json
{"results":[{"target_id":"codex:0","provider":"codex","account":"account1@example.com","status":"ok","used_percent":37.125,"cleanup_failed":false}]}
```

The example shows one result for brevity; actual output retains every target,
including failures. Diagnostics remain on stderr, and exit codes do not change.
Help remains text. Invalid arguments produce no stdout JSON.

Without `--json`, an interactive terminal with a readable size and a usable
`TERM` shows one live row per target, then a final usage bar or status. A reset
countdown appears only when the provider supplied a weekly reset time.
`NO_COLOR` suppresses color but keeps the layout. Pipes, redirects,
`TERM=dumb`, and terminals without a readable size use the plain table shown
above, with the current percentage formatting. The view adds no flag, JSON
reset field, or table column.

JSON `status` values are `ok`, `not_applicable/monthly`,
`not_applicable/api_key`, `unavailable/no_weekly`, `unavailable/period`,
`unavailable/no_data`, `auth_required`, `unsupported`, `cli_not_found`,
`home_not_found`, `home_error`, `timeout`, `protocol_error`, `provider_error`,
and `cancelled`.

The ambient `CODEX_HOME` environment variable does not change the default
account list. Unknown flags, positional arguments, empty values, invalid
durations, and duplicate homes (including through symlinks) are usage
errors.

## Statuses and exit codes

| WEEKLY USAGE cell | Meaning |
|---|---|
| `37%` or `43.2%` | Weekly subscription usage consumed; integers omit the decimal, fractional values show one decimal |
| `0%` for Grok with an omitted weekly percentage | Grok-compatible display fallback; zero usage is not independently verified |
| `N/A (monthly)` | The account reports a monthly period — not weekly usage |
| `N/A (API key)` | API-key auth has no provider subscription quota |
| `N/A (no weekly)` | No weekly window in the provider's quota data |
| `N/A (period)` / `N/A (no data)` | Period metadata or usage missing |
| `AUTH REQUIRED` | No/failed login; authenticate with the provider CLI |
| `UNSUPPORTED` | Installed CLI lacks the qualified usage interface or its output changed |
| `NOT INSTALLED` / `HOME NOT FOUND` / `HOME ERROR` | Local preflight failures |
| `TIMEOUT` / `PROTOCOL ERROR` / `ERROR` | Query failures (row preserved) |
| `CANCELLED` | Interrupted by SIGINT/SIGTERM |

| Exit code | Meaning |
|---|---|
| `0` | All rows numeric or explicitly non-applicable (monthly/API key) |
| `1` | One or more incomplete/error rows, or stdout write failure |
| `2` | Invalid arguments; no usage snapshot is printed |
| `130` / `143` | Cancelled by SIGINT / SIGTERM |

Grok and Claude use the display label `default`; it is not a verified email.
A monthly result is truthful non-applicability, not weekly usage.

## Privacy and side effects

- Credentials stay entirely with the provider CLIs: gaebokchi never
  reads auth files, tokens, or keychains, and never implements OAuth.
- Child processes get a filtered environment: API-key overrides
  (`OPENAI_API_KEY`, `CODEX_API_KEY`, `XAI_API_KEY`, `GROK_API_KEY`,
  `ANTHROPIC_API_KEY`) are removed, and each Codex child gets its own
  `CODEX_HOME`. Claude also drops external-provider routing overrides
  while keeping `HOME` and `CLAUDE_CONFIG_DIR`.
- Children run in private temporary directories outside your project;
  the directories are removed after each run.
- stdout contains the terminal view, table, or JSON snapshot (with account identifiers); diagnostics go
  to stderr as fixed, application-owned messages only.

## Troubleshooting

- **AUTH REQUIRED** — run `codex login` / `grok` / `claude auth login` outside
  gaebokchi.
- **NOT INSTALLED** — install the CLI or pass `--codex-bin`/`--grok-bin`/`--claude-bin`
  with an explicit path.
- **UNSUPPORTED / PROTOCOL ERROR** — the installed CLI version does not
  expose the qualified metadata interface; check the compatibility record
  above before upgrading anything.
- **TIMEOUT** — raise `--timeout` (network-bound quota reads can be slow).

## Documentation

The root [TESTING.md](TESTING.md) documents the offline test gate and its
local fake-provider E2E environment.

The packaged product contract is split across the
[requirements](docs/specs/README.md),
[architecture](docs/architecture/README.md), and
[roadmap](docs/roadmap/README.md). In a source checkout, maintainer and
contributor documentation starts at [docs/README.md](docs/README.md), with
verification evidence under [docs/verification/](docs/verification/).
