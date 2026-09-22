# Gaebokchi (개복치)

A small, read-only command-line tool in the Aquarium ecosystem that shows
**weekly subscription usage** for your Codex (ChatGPT) and Grok accounts
in one shot, reusing the logins already managed by the installed `codex`
and `grok` CLIs.

```text
AI PROVIDER  ACCOUNT                WEEKLY USAGE
ChatGPT      account1@example.com   37.0%
ChatGPT      account2@example.com   12.0%
Grok         default                54.0%
```

(Illustrative output; the ACCOUNT column can contain your real account
identifiers.)

## What it does (and does not do)

- Queries exactly three targets by default: the Codex homes `~/.codex`
  and `~/.codex-hsy`, plus the existing Grok login.
- Shows the percentage already consumed in the provider-reported
  **weekly subscription quota**. It never shows remaining quota, local
  token totals, API spend, or a calendar-week guess.
- One snapshot, then exit. No daemon, watch mode, polling, retries,
  history, or cache.
- Read-only on the business level: no prompts, agent sessions, tool
  actions, login/logout, or billing mutations are ever issued. Provider
  startup itself may refresh tokens, update provider logs/caches, or use
  a provider-managed shared process — those are provider internals
  gaebokchi deliberately does not touch.
- Claude is out of scope for v0.1.

## Prerequisites

- The `codex` CLI installed and logged in (ChatGPT subscription) for
  each home you query.
- The `grok` CLI installed and logged in (`grok` manages its own cached
  token in `~/.grok`).
- A Go toolchain is needed only to build from source; release binaries
  are self-contained (`CGO_ENABLED=0`).

Protocol compatibility was qualified with live, prompt-free queries
against `codex-cli 0.155.1` and `grok 1.0.40 (stable)` on macOS/arm64.
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
gaebokchi                                            # default three targets
gaebokchi --timeout 40s                              # longer query budget
gaebokchi --codex-home ~/.codex-work                 # one custom home (replaces defaults)
gaebokchi --codex-home ~/.codex-work --codex-home ~/.codex-personal
gaebokchi --grok-account personal --grok-bin /opt/homebrew/bin/grok
gaebokchi version
gaebokchi version --json
```

Release builds print `gaebokchi v0.1.0`; ordinary source builds without an
explicit version override print `gaebokchi (devel)`. The JSON form is exactly
`{"name":"gaebokchi","version":"v0.1.0"}` for the v0.1.0 release build.

### Flags

| Flag | Default | Meaning |
|---|---|---|
| `--codex-home PATH` | `~/.codex` and `~/.codex-hsy` | Repeatable; supplying any replaces both defaults |
| `--codex-bin PATH` | `codex` | codex executable path or name |
| `--grok-bin PATH` | `grok` | grok executable path or name |
| `--grok-account LABEL` | `default` | Display alias for the Grok row (not an email) |
| `--timeout DURATION` | `20s` | Per-target query budget, 1s–120s |
| `-h`, `--help` | — | Print help without launching providers |

`version [--json]` is a project-independent command and never launches a
provider. The former `--version` flag is not supported.

The ambient `CODEX_HOME` environment variable does not change the default
account list. Unknown flags, positional arguments, empty values, invalid
durations, and duplicate homes (including through symlinks) are usage
errors.

## Statuses and exit codes

| WEEKLY USAGE cell | Meaning |
|---|---|
| `37.0%` | Weekly subscription usage consumed (one decimal) |
| `N/A (monthly)` | The account reports a monthly period — not weekly usage |
| `N/A (API key)` | API-key auth has no ChatGPT/Grok subscription quota |
| `N/A (no weekly)` | No weekly window in the provider's quota data |
| `N/A (period)` / `N/A (no data)` | Period metadata or usage missing |
| `AUTH REQUIRED` | No/failed login; authenticate with the provider CLI |
| `UNSUPPORTED` | Installed CLI lacks the metadata interface |
| `NOT INSTALLED` / `HOME NOT FOUND` / `HOME ERROR` | Local preflight failures |
| `TIMEOUT` / `PROTOCOL ERROR` / `ERROR` | Query failures (row preserved) |
| `CANCELLED` | Interrupted by SIGINT/SIGTERM |

| Exit code | Meaning |
|---|---|
| `0` | All rows numeric or explicitly non-applicable (monthly/API key) |
| `1` | One or more incomplete/error rows, or stdout write failure |
| `2` | Invalid arguments; no table is printed |
| `130` / `143` | Cancelled by SIGINT / SIGTERM |

A Grok alias is a label you chose, **not** a verified email. A monthly
result is truthful non-applicability, not weekly usage.

## Privacy and side effects

- Credentials stay entirely with the provider CLIs: gaebokchi never
  reads auth files, tokens, or keychains, and never implements OAuth.
- Child processes get a filtered environment: API-key overrides
  (`OPENAI_API_KEY`, `CODEX_API_KEY`, `XAI_API_KEY`, `GROK_API_KEY`)
  are removed, and each Codex child gets its own `CODEX_HOME`.
- Children run in private temporary directories outside your project;
  the directories are removed after each run.
- stdout contains the table (with account identifiers); diagnostics go
  to stderr as fixed, application-owned messages only.

## Troubleshooting

- **AUTH REQUIRED** — run `codex login` / `grok` login flow outside
  gaebokchi.
- **NOT INSTALLED** — install the CLI or pass `--codex-bin`/`--grok-bin`
  with an explicit path.
- **UNSUPPORTED / PROTOCOL ERROR** — the installed CLI version does not
  expose the qualified metadata interface; check the compatibility record
  above before upgrading anything.
- **TIMEOUT** — raise `--timeout` (network-bound quota reads can be slow).

## Documentation

The packaged product contract is split across the
[requirements](docs/specs/README.md),
[architecture](docs/architecture/README.md), and
[roadmap](docs/roadmap/README.md). In a source checkout, maintainer and
contributor documentation starts at [docs/README.md](docs/README.md), with
verification evidence under [docs/verification/](docs/verification/).
