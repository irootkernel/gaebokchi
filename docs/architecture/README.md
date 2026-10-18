# Gaebokchi (개복치): Architecture

Project: **Gaebokchi (개복치)** · Ecosystem: Aquarium tool · Repository and executable: `gaebokchi`

## Document contract and navigation

The [requirements](../specs/README.md) define required behavior and acceptance. The [roadmap](../roadmap/README.md) owns work identity, execution order, status, and delivery history. This document describes the current implementation and its boundaries: components, data semantics, protocol and process rules, and verification design. Release archives carry these three documents, while links to verification records and other maintainer documents resolve only in a source checkout. **External interface facts** have sources in [ARCH-15](#arch-15); **project decisions** are normative design choices, not claims that providers guarantee them. Decisions whose rationale should outlive a rewrite of this document are kept as [architecture decision records](../architecture-decision-records/README.md).

ARCH-01 through ARCH-18 describe the implemented baseline. ARCH-19 specifies the
planned EPIC-013 Cursor single-account adapter and read-only preload hook
contract qualified against `cursor-agent 2026.10.16-e3cb207`.

| Section | Responsibility |
|---|---|
| [ARCH-01](#arch-01) | Decisions and scope |
| [ARCH-02](#arch-02) | Components, dependency direction, repository layout |
| [ARCH-03](#arch-03) | CLI options and account targets |
| [ARCH-04](#arch-04) | Domain model and usage semantics |
| [ARCH-05](#arch-05) | Execution, deadlines, ownership and cleanup |
| [ARCH-06](#arch-06) | Shared transport with separate wire dialects |
| [ARCH-07](#arch-07) | Codex adapter |
| [ARCH-08](#arch-08) | Grok adapter |
| [ARCH-16](#arch-16) | Claude adapter |
| [ARCH-09](#arch-09) | Security and read-only policy |
| [ARCH-10](#arch-10) | Rendering, diagnostics and exits |
| [ARCH-11](#arch-11) | Verification catalog |
| [ARCH-12](#arch-12) | Compatibility discovery and live gates |
| [ARCH-13](#arch-13) | Build and distribution |
| [ARCH-14](#arch-14) | Change and completion discipline |
| [ARCH-15](#arch-15) | Primary sources |
| [ARCH-17](#arch-17) | Weekly model-pool rows and exact remaining-fraction conversion |
| [ARCH-18](#arch-18) | agy single-account adapter and shared two-row query |
| [ARCH-19](#arch-19) | Planned Cursor single-account adapter and read-only preload hook |

Section numbers are stable identifiers, not reading order: ARCH-16 was added after ARCH-15 and sits with the other provider adapters.

<a id="arch-01"></a>
## ARCH-01: Scope and settled decisions

Related: [REQ-003](../specs/README.md#req-003), [REQ-004](../specs/README.md#req-004), [REQ-005](../specs/README.md#req-005), [REQ-056](../specs/README.md#req-056). Implemented across [EPIC-001](../roadmap/README.md#epic-001) through [EPIC-010](../roadmap/README.md#epic-010), plus the Claude extension recorded in [ADR-0001](../architecture-decision-records/ADR-0001-claude-usage-local-command.md).

| Decision | Choice | Consequence |
|---|---|---|
| Product | **Gaebokchi (개복치)**, command-line only | No full-screen TUI, service, or account dashboard; the inline terminal view is drawn directly with ANSI sequences ([ARCH-10](#arch-10-terminal)) |
| Ecosystem | Aquarium tool | Standalone; no Aquarium runtime or build dependency |
| Repository and executable | `gaebokchi` | One consistent ASCII name for repository, binary and command directory |
| Language | Go, standard library first | Small build and deployment surface |
| Providers | Codex, Grok, Claude, and `agy`, each through its installed CLI | No PTY, UI scraping, or application-owned provider API client |
| Authentication | Existing provider CLI login | No token parsing or account-linking workflow |
| Account set | Required `~/.config/gaebokchi/config.yaml` | Only listed accounts are queried; omitted providers are disabled |
| Metric | Provider-defined weekly allowance consumed | No cross-provider arithmetic or cost comparison |
| Transport | Local stdio to provider executables | No network listener or app-owned HTTP client |
| Execution | One snapshot, up to three targets concurrently | No periodic refresh or automatic retry; excess targets wait for a free slot |
| Persistence | User-authored account configuration only | No application-owned usage cache or credential store |
| Output | Interactive terminal view, three-column table, or JSON; one line per configured target plus the header | Failures remain in their account row |

The v0.1 plan limited the providers to Codex and Grok. [ADR-0001](../architecture-decision-records/ADR-0001-claude-usage-local-command.md) records why and how Claude was added afterwards; the adapter is [ARCH-16](#arch-16).

A provider may define a rolling or anchored seven-day allowance rather than a Monday-to-Sunday week. Follow the provider's reported period. Do not promise a shared quota across all ChatGPT, Grok, or Claude products. A shared or unified Grok pool may be present in the billing response, but the application reports the selected allowance without broadening its interpretation. The Claude value is the CLI's all-models weekly line.

**Naming contract:** Use **Gaebokchi (개복치)** for human-readable project titles and `gaebokchi` for the repository basename, executable, `cmd/gaebokchi` entry point, protocol client names, diagnostic prefix and release-artifact prefix. Where a protocol has a separate human-readable client title, use `Gaebokchi`. The Go module path is the local-only `gaebokchi` that TASK-002 chose while the repository had no remote identity; the release repository is `github.com/irootkernel/gaebokchi`, and aligning the module path with it is recorded as [deferred feedback](../deferred-feedback/README.md). Aquarium membership is ecosystem identity, not an additional integration feature or runtime dependency.

Runtime target concurrency is unrelated to development parallelism: implementation tasks remain strictly sequential.

<a id="arch-02"></a>
## ARCH-02: Components and dependency direction

Related: [REQ-001](../specs/README.md#req-001), [REQ-041](../specs/README.md#req-041). Owners: [TASK-002](../roadmap/README.md#task-002), [TASK-012](../roadmap/README.md#task-012).

```text
cmd/gaebokchi   composition root: version dispatch, help, output mode, signals, exit code
 |
 +-> internal/options               argv parsing, YAML account configuration, ordered targets
 +-> internal/app                   bounded fan-out and fixed result slots
 +-> internal/render                terminal frame, live redraw, table, JSON, diagnostics
 +-> internal/provider/codex  --+-> internal/rpc                  framing, IDs, dialect codecs
 +-> internal/provider/grok   --+-> internal/process              owned child lifecycle
 |                              +-> internal/provider/envfilter   child environment policy
 +-> internal/provider/claude ----> internal/process, internal/provider/envfilter (no RPC)
 +-> internal/provider/agy    ----> internal/process, internal/provider/envfilter (no RPC)

cmd/gaebokchi builds the providers and injects them into internal/app through the
app.Querier interface; internal/app imports only internal/domain.
Every package above depends inward on internal/domain, which imports no internal package.
internal/rpc, internal/process, and internal/provider/envfilter import only the standard library.
internal/testutil is an offline fake-process harness that only tests import.
```

The application boundary is the small `app.Querier` interface, `Query(context.Context, domain.Target) domain.Result`. A provider returns exactly one terminal result for one target; it does not render, terminate the program, mutate the global environment, or coordinate other accounts. Expected provider failures belong in a typed result. Setup and argument failures before targets exist remain command-level errors.

Multiple targets of one provider share adapter code but not connection state. The shared RPC layer knows framing and message correlation, not quota fields or provider billing semantics. Dialect codecs in `internal/rpc` own each provider's envelope rules and outbound method allowlist, while the Codex and Grok packages own their handshake sequences. The Claude adapter runs two CLI commands through `internal/process` and parses their output itself ([ARCH-16](#arch-16)); the `agy` adapter runs `--version` and `--print /usage --output-format json --print-timeout 15s` once per invocation and projects its two weekly pools (`gemini` and `claude/gpt`) onto `agy:0` and `agy:1` ([ARCH-18](#arch-18)).

Repository layout (test suites are listed in `TESTING.md`):

```text
cmd/gaebokchi/{main,version,terminal_unix}.go
internal/app/runner.go
internal/domain/{target,result,status,label,percent}.go
internal/options/options.go
internal/options/config.go
internal/process/{child,process_unix}.go
internal/rpc/{client,frame,codec}.go
internal/provider/codex/{provider,normalize}.go
internal/provider/codex/testdata/            # sanitized captured fixtures
internal/provider/grok/{provider,normalize}.go
internal/provider/grok/testdata/             # sanitized captured fixtures
internal/provider/claude/provider.go
internal/provider/agy/provider.go
internal/provider/envfilter/envfilter.go
internal/provider/home/home.go
internal/render/{terminal,live,table,json}.go
internal/testutil/fake.go                    # offline fake-process harness, test-only
test/integration/                            # executable-level E2E suite
scripts/{check-docs,archivecheck}.go, scripts/build-release.sh, scripts/ginkgo
docs/                                        # maintainer documentation, indexed by docs/README.md
README.md, LICENSE, TESTING.md, Makefile, .github/workflows/ci.yml
```

Prefer constructors and ordinary functions over a service container, plugin registry, event bus, job framework, or generic agent SDK. The terminal path uses `golang.org/x/sys/unix` for window size and `golang.org/x/text/width` for display cells; it adds no TUI framework. Test process injection through a narrow factory, not global monkey patching. The module path is recorded in [ARCH-01](#arch-01). The toolchain is Go 1.27 (`go.mod`), pinned in CI; `TESTING.md` lists the required tools.

<a id="arch-03"></a>
## ARCH-03: Command contract and account targets

Related: [REQ-002](../specs/README.md#req-002), [REQ-006](../specs/README.md#req-006) through [REQ-008](../specs/README.md#req-008), [REQ-056](../specs/README.md#req-056), [REQ-064](../specs/README.md#req-064).

A usage run reads `~/.config/gaebokchi/config.yaml`; help and `version [--json]` do not. The file has optional `codex`, `grok`, `claude`, and `agy` lists. `codex`, `grok`, and `claude` entries contain required `name` and `path`; `agy` accepts at most one entry (such as `- name: default`) with `path` optional and not used as a credential selector ([ARCH-18](#arch-18)). At least one account is required. Omission or `[]` disables a provider; a present null list is invalid. Strict YAML decoding rejects unknown and duplicate keys, merge keys, or multiple documents. Names are normalized display aliases and must be nonblank and unique within the provider. Paths must be absolute or begin with the current user's `~/`; `~/` is expanded without cleaning `.` or `..` before filesystem traversal. Duplicate detection compares existing directories by filesystem identity and missing homes by the deepest existing directory plus Unicode-decomposed, case-folded unresolved suffix, following existing symlink ancestors within each provider. This conservative missing-path rule is the same on every platform; two case variants on a case-sensitive filesystem become independently configurable once both directories exist. Missing directories remain row-level `HOME NOT FOUND` errors; inaccessible or non-directory paths become `HOME ERROR`. No provider state is created.

Targets are built in Codex, Grok, Claude, `agy` order, preserving list order within each provider. IDs are `codex:0`, `grok:0`, `claude:0`, `agy:0` (`gemini`), `agy:1` (`claude/gpt`), etc., with dense result indices in final output order. The configured `path` becomes `CODEX_HOME`, `GROK_HOME`, or, for non-default Claude homes, `CLAUDE_CONFIG_DIR` in that target's child environment. For the current user's `~/.claude`, remove ambient `CLAUDE_CONFIG_DIR` and use the CLI's implicit default credential namespace. An enabled `agy` entry uses the CLI's single default login (`HOME` preserved) and expands into two weekly pool rows (`gemini` and `claude/gpt`). The configured `name` is the output account alias on every status (or `gemini` and `claude/gpt` for `agy`'s two model-pool rows). Ambient selector values do not choose a target. A name does not verify which login is stored in its path.

Missing or invalid configuration exits 2 without starting a provider or writing usage stdout. stderr names the config location and points to `https://github.com/irootkernel/gaebokchi/blob/main/configuration/README.md`. The old `--codex-home` and `--grok-account` flags are rejected. The remaining flags are `--codex-bin`, `--grok-bin`, `--claude-bin`, `--agy-bin`, `--timeout` (1s to 120s; default 20s), `--json`, and `--help`/`-h`. The only subcommand is `version [--json]`; it never reads configuration or starts providers. Executable overrides retain explicit-path resolution before the child's temporary working directory is selected.

<a id="arch-04"></a>
## ARCH-04: Domain and normalized result

Related: [REQ-010](../specs/README.md#req-010), [REQ-013](../specs/README.md#req-013), [REQ-014](../specs/README.md#req-014), [REQ-015](../specs/README.md#req-015). Owners: [TASK-002](../roadmap/README.md#task-002), [TASK-006](../roadmap/README.md#task-006), [TASK-008](../roadmap/README.md#task-008), [TASK-010](../roadmap/README.md#task-010).

Conceptual Go model, not a requirement to copy exact field names:

```go
type Result struct {
    TargetID      string
    Provider      ProviderKind
    Account       string
    AccountSource AccountSource // protocol_email or configured_alias
    Status        Status
    UsedPercent   *float64      // non-nil only for StatusOK
    Period        PeriodKind   // weekly, monthly, unknown
    QuotaID       string        // e.g. codex or claude_all_models; not a cross-provider unit
    ResetAt       *time.Time
    ObservedAt    time.Time
    Source        string        // codex_app_server, grok_acp_billing, or claude_cli_usage
    Diagnostic    Diagnostic    // safe code + application-owned message
    CleanupFailed bool          // bounded cleanup did not complete (ARCH-05)
}
```

Transport structs remain in the provider package. Preserve field presence with pointers or raw JSON decoding. Go zero values must not turn omitted numbers into a measured zero; the Grok weekly omission fallback in ARCH-08 is an explicit display convention. Decode the percentage as a number, not a numeric string. Ignore additive unknown object fields, but validate all fields needed to establish the metric.

Project percentage policy: accept finite values in `[0,100]`; in the renderer, omit the decimal for an original integer value and otherwise show one decimal place, normalizing signed zero. Do not round before selecting a period or comparing duplicate windows. JSON preserves provider precision. Out-of-range values are a protocol/semantic mismatch, not a reason to clamp. A future upstream definition allowing percentages over 100 requires an explicit contract change.

### Terminal statuses

| Internal status/reason | WEEKLY USAGE cell | Exit class |
|---|---|---|
| `ok` | e.g. `37%` or `43.2%` | Clean |
| `not_applicable/monthly` | `N/A (monthly)` | Clean |
| `not_applicable/api_key` | `N/A (API key)` | Clean |
| `unavailable/no_weekly` | `N/A (no weekly)` | Incomplete |
| `unavailable/period` | `N/A (period)` | Incomplete |
| `unavailable/no_data` | `N/A (no data)` | Incomplete |
| `auth_required` | `AUTH REQUIRED` | Incomplete |
| `unsupported` | `UNSUPPORTED` | Incomplete |
| `cli_not_found` | `NOT INSTALLED` | Incomplete |
| `home_not_found` | `HOME NOT FOUND` | Incomplete |
| `home_error` | `HOME ERROR` | Incomplete |
| `timeout` | `TIMEOUT` | Incomplete |
| `protocol_error` | `PROTOCOL ERROR` | Incomplete |
| `provider_error` | `ERROR` | Incomplete |
| `cancelled` | `CANCELLED` | Signal cancellation |

An unknown Codex authentication type maps to `UNSUPPORTED`, not the API-key status. For Grok, absence of `cached_token` with only an API-key method advertised establishes API-key-only non-applicability; interactive methods without cached authentication mean `AUTH REQUIRED`. Unrecognized auth advertisements mean `UNSUPPORTED`. For Claude, `loggedIn: false` means `AUTH REQUIRED`, and only a first-party `claude.ai` login proceeds to the usage read ([ARCH-16](#arch-16)).

For output, every result uses the configured `name`. The Codex adapter may read an email as part of its qualified account response, but it does not override the configured output alias. Duplicate normalized aliases within a provider are rejected during configuration loading, so no post-query label disambiguation is needed.

<a id="arch-05"></a>
## ARCH-05: Execution and process ownership

Related: [REQ-004](../specs/README.md#req-004), [REQ-009](../specs/README.md#req-009), [REQ-020](../specs/README.md#req-020), [REQ-022](../specs/README.md#req-022), [REQ-023](../specs/README.md#req-023), [REQ-040](../specs/README.md#req-040). Owners: [TASK-003](../roadmap/README.md#task-003), [TASK-009](../roadmap/README.md#task-009).

```text
parse/validate -> build ordered targets -> admit up to 3 targets
    each target:
        preflight -> private temporary cwd per child -> start child + pipes
        -> Codex, Grok: initialize -> account/auth -> quota request
           Claude:      auth status command -> /usage command
        -> normalize -> bounded cleanup -> terminal result
all terminal results -> selected output with configured aliases -> exit code
```

Use a fixed result slice and index-based assignment. A target's deadline starts when admitted, not when first queued. At most three are active. The first three configured targets are admitted immediately; any additional targets use the same bounded worker pool. A failed target does not cancel the invocation context.

The optional `Runner.Observe` callback receives each stored terminal result
once, including a target cancelled while queued. The CLI observer copies the
result into renderer-owned state; a separate writer serializes terminal
frames. This does not change target admission, query attempts, or deadlines.

The application issues no automatic retries or process restarts. A provider can internally refresh credentials or retry its own network requests, but the outer deadline still applies. Do not wait for a quota notification as a substitute for the explicit read response.

### Child lifecycle

The process component owns start, stdin/stdout/stderr, process wait, and cleanup. The Codex and Grok adapters borrow an RPC connection over the child's pipes and always close it; the Claude adapter closes stdin at once and reads bounded stdout ([ARCH-16](#arch-16)). Use `os/exec` with explicit arguments, not `sh -c`. [S-07](#s-07)

At termination, stop accepting new RPC work and close stdin. Allow up to 500 ms for EOF exit, then up to 500 ms after SIGTERM, then use SIGKILL if required and finish bounded reaping/pipe closure within the remaining cleanup budget. The total cleanup allowance is 2 seconds, separate from the query budget. Design cancellation and pipe ownership so a stuck pipe reader cannot extend this allowance indefinitely.

`Wait` has one owner and is called exactly once for every successfully started direct child. Drain stderr concurrently so backpressure cannot deadlock the child. Retain at most 64 KiB in memory and discard the rest; do not emit that raw buffer. Stop/join protocol goroutines and close descriptors on every path. A normal, planned termination after a valid query must not overwrite the collected result with a spurious failure. An exit before a required response is an error.

A private Unix process group is created for every child, but only for signal isolation: the child and anything it starts cannot deliver terminal-control signals to gaebokchi's own group. The group is never signalled during cleanup. All three production adapters run under the direct-child-only policy: gaebokchi terminates, reaps, and closes its own pipe ends for the direct child it started, on both the graceful path (EOF, SIGTERM) and the forced path (SIGKILL to the direct child alone). No qualified basis exists for treating a group member as disposable — group membership or ancestry does not distinguish an application-owned descendant from a provider-managed shared leader, and a direct child's unresponsiveness does not either. Descendants with uncertain ownership therefore survive both graceful and forced cleanup; closing gaebokchi's own pipe ends is what unblocks its readers. Grok can use a provider-managed shared leader; never send leader shutdown/control RPCs, never use `pkill grok`/`pkill codex`, and never kill a process merely because it appeared during a query. TASK-001 records the installed CLI's ownership behavior.

Cleanup incompleteness is reported, never hidden: when bounded cleanup cannot fully complete (for example, the private temporary directory cannot be removed), the target's result carries `CleanupFailed`, a bounded application-owned diagnostic is emitted, and the invocation exits 1 — while a measured quota stays numeric and the signal exit codes (130/143) keep their precedence.

Create one mode-0700 temporary cwd per started child outside the invocation project; a Claude query starts two children, and a target that fails preflight starts none. Preserve the real credential homes. Remove only gaebokchi-owned temporary paths, after process cleanup. This reduces accidental project-config discovery; it is not a claim that a trusted provider binary is sandboxed or incapable of reading user-level configuration.

SIGINT/SIGTERM cancel active contexts and mark unstarted targets cancelled. Render once with completed results retained when output remains writable, then return the signal-specific exit code. Do not forcibly clean up the caller's other agent sessions.

<a id="arch-06"></a>
## ARCH-06: Shared RPC transport, separate codecs

Related: [REQ-016](../specs/README.md#req-016) through [REQ-019](../specs/README.md#req-019), [REQ-043](../specs/README.md#req-043). Owner: [TASK-004](../roadmap/README.md#task-004).

A reusable connection provides newline framing, serialization, request-ID correlation, cancellation, bounded buffers, and terminal transport errors. It does **not** implement a universal Codex/ACP handshake. Prefer one outstanding client request per provider connection; different account processes can run concurrently. Only the Codex and Grok adapters use this transport; Claude has no RPC dialect ([ARCH-16](#arch-16)).

| Property | Codex App Server | Grok ACP |
|---|---|---|
| Framing | One JSON object per newline | One JSON object per newline |
| Outgoing `jsonrpc` | Omitted | `"2.0"` |
| Inbound response `jsonrpc` | Not required; an additive member is tolerated | Required and exactly `"2.0"` |
| Startup | `initialize` response, then `initialized` notification | `initialize` negotiation, then `authenticate` |
| Identity | `account/read` | Configured alias; no identity query |
| Quota method | `account/rateLimits/read` | Qualified billing extension `_x.ai/billing` |
| Agent work | Forbidden | Forbidden |

These external protocol properties come from [S-01](#s-01), [S-03](#s-03), and [S-04](#s-04). TASK-001 qualified the installed-version envelopes, which the [compatibility record](../verification/compatibility.md) keeps.

Use a line reader with an explicit 4 MiB ceiling, including delimiter accounting. Do not rely on an unmodified `bufio.Scanner` default. Accept LF and CRLF framing, partial reads, multiple messages in one read, and valid UTF-8. Reject invalid JSON/UTF-8, top-level arrays/batches, and unexpected top-level types. Ignore empty transport lines only; non-JSON banners on stdout are protocol errors, not content to scrape. [S-08](#s-08)

Allocate positive integer request IDs per connection. A response must contain the matching ID and exactly one structurally valid `result` or `error`; `error: null` without a result, an error object with missing or mistyped members, or a shapeless response is malformed and fails the call. Preserve explicit JSON null where a schema allows it, and let the adapter reject null when an object is required. Each adapter also validates its handshake result shape (Codex `initialize`, Grok `authenticate`) while tolerating additive members. Well-formed unknown/duplicate response IDs are ignored and cannot complete a call. A well-formed unrelated notification is ignored. EOF fails unresolved calls immediately.

A message with a method and ID is a server-initiated request, not a response. No capability in this product permits executing it. Return one dialect-correct unsupported-method error with the exact received ID, then abort the affected target with a safe diagnostic. Never provide filesystem contents, terminal access, approval, external tokens, or interactive authentication.

Timeouts unblock both reads and writes by closing/cancelling the owned transport. A blocking receive loop that processes notifications is not polling; do not add sleeps that repeatedly query server status.

### Provider method allowlists

Codex outbound: `initialize`, `initialized`, `account/read`, `account/rateLimits/read`.

Grok outbound: `initialize`, `authenticate`, and the single qualified billing wire method. No `initialized` notification is added to Grok merely because Codex uses it. No `session/new`, `session/load`, `session/prompt`, `turn/start`, or approval/credit-mutating methods are allowed.

Apply the allowlist at the provider codec boundary and assert actual outbound transcripts in tests. Do not ship a generic arbitrary-RPC command.

<a id="arch-07"></a>
## ARCH-07: Codex adapter

Related: [REQ-025](../specs/README.md#req-025) through [REQ-030](../specs/README.md#req-030), [REQ-039](../specs/README.md#req-039). Owners: [TASK-005](../roadmap/README.md#task-005), [TASK-006](../roadmap/README.md#task-006).

Provider CLI invocation: `codex app-server`, with a separately constructed `CODEX_HOME` for each target. Local state may live in the home directory or an OS credential store managed by Codex. The application reads neither. [S-02](#s-02)

Qualified protocol exchange, documented in [S-01](#s-01) and confirmed against codex-cli 0.155.1 in the [compatibility record](../verification/compatibility.md). The `clientInfo.version` value is the fixed string `0.1.0`, independent of the build version:

```json
{"id":1,"method":"initialize","params":{"clientInfo":{"name":"gaebokchi","title":"Gaebokchi","version":"0.1.0"}}}
```

Wait for response 1, which must be an object carrying `platformOs`. Then:

```json
{"method":"initialized","params":{}}
{"id":2,"method":"account/read","params":{"refreshToken":false}}
```

After validating the account response, send:

```json
{"id":3,"method":"account/rateLimits/read"}
```

`refreshToken:false` means this application does not force a refresh in that read; it does not guarantee that Codex will never perform provider-owned refresh elsewhere. Do not enable experimental capabilities unless a verified required interface actually needs one, and then record that dependency. No thread or turn is necessary for this design.

### Authentication mapping

Accept the qualified `account.type == "chatgpt"` shape and an optional email. `account:null` is `AUTH REQUIRED`. The known `apiKey` type is non-applicable, and unknown auth variants are `UNSUPPORTED`. A quota call's authentication failure maps to `AUTH REQUIRED` only when a qualified structured error identifies it. None has been qualified, so a remote error from `account/read` or the quota call is `ERROR`. Do not classify arbitrary English error text as proof of expired login. Preserve the home alias if identity was not read successfully.

### Quota selection algorithm

The following is the application's deterministic selection policy, not an undocumented provider promise:

1. Validate the outer rate-limit result object. When `rateLimitsByLimitId` is a **non-empty map**, select only its `codex` entry. If it is missing, return `N/A (no weekly)` and do not switch to another bucket or the legacy view. A present `codex` value with a wrong structural type is a protocol error; null yields `N/A (no data)`.
2. When the map is absent, null, or empty, consider the backward-compatible `rateLimits` object. Accept an explicit `limitId == "codex"`. TASK-001 captured no legacy fixture, so an absent, null, or empty `limitId` is not eligible and yields `N/A (no weekly)`. Explicit non-Codex identifiers are not eligible. No legacy fixture means no speculative legacy branch.
3. In the selected bucket, inspect both `primary` and `secondary`. Match an explicitly present numeric integer `windowDurationMins == 10080`. Null or absent windows are allowed. Never infer seven days from slot position or `resetsAt`.
4. If no eligible window exists, return `N/A (no weekly)`. If exactly one exists, require a valid `usedPercent`. Missing/null usage gives `N/A (no data)`; a wrong type/range gives `PROTOCOL ERROR`.
5. If both windows are weekly and their used percentage and optional reset values agree exactly, treat them as a duplicate representation of one window. If either candidate is malformed or they conflict, return `PROTOCOL ERROR`. Do not choose max/min/first.
6. Preserve valid reset metadata as a Unix-seconds timestamp when available. Missing reset metadata does not invalidate an otherwise valid percentage. A malformed optional reset can be discarded with a safe diagnostic; it must not be used to guess the period.

Inspecting only an unrelated malformed bucket must not poison a valid selected Codex bucket. Unknown additive fields, credit balances, and token-summary fields are ignored. All normalization functions are pure and table-tested independently of a child process.

<a id="arch-08"></a>
## ARCH-08: Grok adapter

Related: [REQ-031](../specs/README.md#req-031) through [REQ-036](../specs/README.md#req-036). Owners: [TASK-007](../roadmap/README.md#task-007), [TASK-008](../roadmap/README.md#task-008).

**Compatibility boundary:** the official source includes a billing extension, and TASK-001 qualified it live through the installed stdio binary (grok 1.0.40; [compatibility record](../verification/compatibility.md)). The public source revision inspected is recorded in [S-05](#s-05). Do not equate a source handler or in-process TUI call with exposed wire support.

Qualified invocation: `grok --no-auto-update agent stdio`. The documented flag suppresses automatic-update checks in automated use. Do not silently remove it and accept update side effects if a binary rejects it; qualify another supported invocation only through the contract gate. [S-03](#s-03)

Handshake:

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientInfo":{"name":"gaebokchi","version":"0.1.0"},"clientCapabilities":{"fs":{"readTextFile":false,"writeTextFile":false},"terminal":false}}}
```

Wait for the response, validate protocol version 1 (any other version is `UNSUPPORTED`), and inspect the advertised auth methods. Select `cached_token` only. Then send the following; its result must be an object carrying `_meta`:

```json
{"jsonrpc":"2.0","id":2,"method":"authenticate","params":{"methodId":"cached_token","_meta":{"headless":true}}}
```

No interactive method is a fallback. API-key environment variables must not silently select a different billing account. Preserve the provider's normal credential context but never pass a token in RPC parameters. The cached-token/headless path is documented in [S-03](#s-03).

### Internal method name versus wire name

The handler is named `x.ai/billing` internally. ACP reserves underscore-prefixed wire methods for extensions; the inspected Grok source also documents this prefix distinction. [S-04](#s-04), [S-05](#s-05), [S-06](#s-06)

The qualified wire form (grok 1.0.40) is:

```json
{"jsonrpc":"2.0","id":3,"method":"_x.ai/billing","params":{}}
```

TASK-001 captured the exact successful envelope and parameter shape from the installed version. An SDK's `ExtRequest("x.ai/billing", ...)` may serialize differently from handwritten JSON. Inspect the matching serializer/router and validate end to end. Do not label the endpoint unsupported merely because a bare internal method name was sent incorrectly, and do not add a runtime shotgun sequence of guessed names/wrappers. Freeze one observed contract in the adapter and fixture; version-specific branches require separate evidence.

Require billing retrieval without `session/new` or a prompt. If it needs an agent session, a browser, token extraction, a private REST call, or a mutation, stop at the gate and record the incompatibility. Do not silently broaden scope. A well-formed method-not-found error from the correctly encoded request is `UNSUPPORTED`.

### Result mapping

The inspected billing types contain `config.creditUsagePercent` and `config.currentPeriod` with a period `type`, optional `start`, and optional `end`. The source describes `USAGE_PERIOD_TYPE_WEEKLY` and weekly/monthly periods. Preserve the distinction between an omitted percentage, explicit `null`, and numeric zero. [S-05](#s-05)

Normalize in this order:

| Observation | Result |
|---|---|
| Outer response is not the qualified object shape | `PROTOCOL ERROR` |
| `config` absent/null | `N/A (no data)` |
| `config` wrong type | `PROTOCOL ERROR` |
| `currentPeriod.type == "USAGE_PERIOD_TYPE_MONTHLY"` | `N/A (monthly)`; never show its percentage as weekly |
| Period/type absent or unknown well-formed enum | `N/A (period)` |
| Required period/type has an invalid structural type | `PROTOCOL ERROR` |
| Explicit weekly enum, percentage member omitted | `0%` as a Grok display compatibility convention |
| Explicit weekly enum, percentage explicitly null | `N/A (no data)` |
| Explicit weekly enum, valid percentage | Numeric weekly result |
| Explicit weekly enum, invalid percentage | `PROTOCOL ERROR` |
| Legacy monthly credit fields only | `N/A (period)` |

The monthly enum spelling comes from the source and fixtures only: the qualified account was weekly, so no live monthly response has been observed. Confirm enum spellings, wrappers, numeric encodings, and auth or error shapes from source and fixtures rather than guessing them from a web UI. A numeric enum is not accepted merely because its value appears plausible; it needs a version-qualified mapping.

Grok 1.0.41 omitted `creditUsagePercent` in a live weekly response. Its pager can use 0.0 when the optional percentage is absent and no usable legacy monthly limit exists. Gaebokchi's chosen fallback applies only to an explicit weekly period; omission does not independently prove measured zero usage. See [the 1.0.41 compatibility record](../verification/compatibility.md#grok-1041-omitted-weekly-percentage). An unavailable `config` or explicit `null` percentage still remains unavailable.

Parse an optional period end as RFC 3339 when present. A malformed optional end can be discarded with a diagnostic without replacing the verified weekly percentage. Never infer a period from date subtraction. Ignore prepaid balance, on-demand usage/caps, history, and subscription tier for percentage calculation.

The ACCOUNT column uses the configured alias. Each Grok child receives its own `GROK_HOME`; the known billing payload is not treated as an email source.

<a id="arch-16"></a>
<a id="current-claude-extension"></a>
## ARCH-16: Claude adapter

Related: [REQ-056](../specs/README.md#req-056) through [REQ-059](../specs/README.md#req-059), [REQ-038](../specs/README.md#req-038), [REQ-039](../specs/README.md#req-039). Owner: none; no roadmap task owns the Claude extension ([REQ-056](../specs/README.md#req-056)). Decision: [ADR-0001](../architecture-decision-records/ADR-0001-claude-usage-local-command.md). Verification: [V-CLAUDE](#v-claude).

For each configured Claude home, the adapter reads that Claude CLI login's weekly all-models allowance through Claude Code's built-in `/usage` local command. It starts two short-lived children through `internal/process` with an environment from `internal/provider/envfilter`. It has no RPC dialect, credential parsing, or application-owned HTTP connection. The [Claude compatibility record](../verification/claude-compatibility.md) holds the live observations; [S-14](#s-14) through [S-16](#s-16) are the published references.

**Invocation.** Resolve `--claude-bin` (default `claude`) for the query; a missing executable is `NOT INSTALLED`, and one that cannot be used is `ERROR`. Build one filtered environment, then run two commands in sequence under the target's single query budget. Each command is its own owned child with its own private temporary working directory:

```text
claude --safe-mode auth status --json
claude --safe-mode --tools "" -p /usage --max-turns 0 --output-format json --no-session-persistence
```

Close each child's stdin at once and read its stdout with a 256 KiB ceiling; more output is `PROTOCOL ERROR`. A timeout or cancellation during either command is `TIMEOUT` or `CANCELLED`, and any other failure to run a command is `ERROR`. Cleanup follows [ARCH-05](#arch-05); a cleanup failure in either child sets the result's `CleanupFailed`.

**Authentication mapping.** The usage command runs only after the auth status establishes a first-party subscription login:

| Auth status observation | Result |
|---|---|
| Not JSON, or `loggedIn` missing | `UNSUPPORTED` |
| `loggedIn: false`, whatever the exit status | `AUTH REQUIRED` |
| Nonzero exit with `loggedIn: true` | `ERROR` |
| `authMethod` is `claude.ai` and `apiProvider` is `firstParty` | Continue to the usage read |
| Otherwise, `authMethod` contains both `api` and `key` (case-insensitive) | `N/A (API key)`; qualified by fixtures only |
| Any other combination, such as Bedrock routing | `UNSUPPORTED` |

**Usage result.** A nonzero exit is `UNSUPPORTED`. The JSON envelope must report `local_command: "usage"`, `num_turns: 0`, `total_cost_usd: 0`, `is_error: false`, and a string `result`; anything else is `UNSUPPORTED`. Scan the result text line by line, ignoring a trailing carriage return, for exactly one match of:

```text
^Current week \(all models\): ([0-9]+(?:\.[0-9]+)?)% used((?:[ \t].*)?)$
```

No match, or more than one, is `UNSUPPORTED`; a value above 100 is `PROTOCOL ERROR`. The current-session meter and model-specific weekly meters are never used. A successful result has period `weekly`, quota ID `claude_all_models`, source `claude_cli_usage`, and the configured account alias.

**Reset metadata.** Only the selected line's suffix can supply `ResetAt`. It must match:

```text
^ · resets ([A-Z][a-z]{2} [0-9]{1,2})(?:, ([0-9]{4}))?(?:,| at) ([0-9]{1,2})(?::([0-9]{2}))?([ap]m) \(([^()]+)\)$
```

The zone is loaded from the host's zone database with `time.LoadLocation`; the binary embeds no `time/tzdata`, so a zone the host cannot load discards the reset. An omitted year means the zone's current year, and the reset is discarded when that year changed while the usage command ran. An impossible date is discarded. The local time must map to exactly one instant, so a time that a daylight-saving change skips or repeats is discarded. A discarded reset never changes the percentage.

**Environment.** Remove `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_BASE_URL`, `ANTHROPIC_AWS_API_KEY`, `ANTHROPIC_AWS_BASE_URL`, `CLAUDE_CODE_OAUTH_TOKEN`, `CLAUDE_CODE_USE_ANTHROPIC_AWS`, `CLAUDE_CODE_USE_BEDROCK`, `CLAUDE_CODE_USE_FOUNDRY`, `CLAUDE_CODE_USE_MANTLE`, `CLAUDE_CODE_USE_VERTEX`, `CLAUDE_CODE_PROVIDER_MANAGED_BY_HOST`, `CLAUDE_CODE_SIMPLE`, and ambient `CLAUDE_CONFIG_DIR` from the inherited environment. Set the target's `CLAUDE_CONFIG_DIR` only for a non-default Claude home. The default `~/.claude` home leaves it unset so the CLI uses its existing credential namespace. `HOME` and proxy settings stay inherited so the CLI can use its own credential store.

**Statuses.** A Claude row can be `ok`, `not_applicable/api_key`, `auth_required`, `unsupported`, `home_not_found`, `home_error`, `cli_not_found`, `timeout`, `protocol_error`, `provider_error`, or `cancelled`.

**Limitations.** The weekly line is human-readable CLI output, not a published structured schema; a wording or envelope change yields `UNSUPPORTED` rather than a guess. The CLI's `stream-json` output carries a structured usage report whose schema is marked experimental, so the adapter does not use it. Reset text has minute resolution and no UTC offset, so a countdown can reach `now` up to a minute early. Each configured Claude directory is a separate query target; the installed CLI must honor `CLAUDE_CONFIG_DIR` for non-default homes to keep their logins distinct.

<a id="arch-09"></a>
## ARCH-09: Security and read-only boundary

Related: [REQ-019](../specs/README.md#req-019), [REQ-021](../specs/README.md#req-021), [REQ-037](../specs/README.md#req-037) through [REQ-041](../specs/README.md#req-041), [REQ-057](../specs/README.md#req-057). Owners: [TASK-003](../roadmap/README.md#task-003), [TASK-004](../roadmap/README.md#task-004), [TASK-005](../roadmap/README.md#task-005), [TASK-011](../roadmap/README.md#task-011).

The trust boundary is the user-selected, already installed provider executable. The application is not a sandbox for a malicious binary. It avoids adding credential handling or agent capabilities of its own.

| Boundary | Required behavior |
|---|---|
| Credentials | No auth-file parsing/copying, JWT decoding, token input, or application keychain access |
| Account changes | No login/logout, account switching, device flow, browser launch, or automatic repair |
| Agent capabilities | No thread/session creation, model prompts, host tools, approvals, or workspace data; Claude's `/usage` local command runs with `--tools ""`, `--max-turns 0`, and `--no-session-persistence`, and its result must report zero turns and zero cost |
| Billing | Read allowance only; no purchases, top-up changes, reset credits, or notifications to account owners |
| Configuration | No writes to Codex, Grok, or Claude configuration or auth; no changes to existing login; Claude commands run in `--safe-mode` |
| Networking | Only provider processes contact their backends; gaebokchi has no HTTP billing client or listener |
| Logs | Application-owned error codes/messages only; raw protocol/stderr remains undisclosed |
| Persistence | No application telemetry or permanent runtime files |
| Shared processes | Do not shut down/kill provider leaders or unrelated coding sessions |

Construct a new child environment for every target. Start from inherited environment only to retain necessary PATH, HOME, XDG/keychain, certificate, and proxy behavior; then filter known conflicting authentication/parent-session variables. Baseline Codex filtering includes `OPENAI_API_KEY`, `CODEX_API_KEY`, and replacement of every `CODEX_HOME` entry. Baseline Grok filtering includes `XAI_API_KEY` and `GROK_API_KEY`, followed by per-target `GROK_HOME`. Claude filtering removes the Anthropic credential and third-party routing overrides listed in [ARCH-16](#arch-16), preserves `HOME`, and sets `CLAUDE_CONFIG_DIR` only for non-default homes. TASK-001 found no further variable that changed Codex or Grok outcomes; add a newly identified variable to the provider's filter only with evidence. Do not dump environment values as evidence or clear all variables indiscriminately.

Shell aliases are not executables. Binary flags accept one program path/name, not arbitrary arguments. Do not bypass Go's `ErrDot` protection or run a binary accidentally found through an implicit current-directory PATH entry. A deliberately specified path containing spaces is passed as one executable value, not split into a command.

“Read-only” here describes application requests and business effects. Provider startup can refresh existing tokens, update provider logs/caches, or use a provider-managed leader. The application must neither promise a byte-for-byte unchanged provider home nor take over those provider internals. Document any material observed startup effect, and block an integration that requires model work or an account/billing mutation.

Treat error data as untrusted. Convert only verified structured codes to named statuses; otherwise use `ERROR` with a fixed safe message and target ID. Never print `error.Error()` blindly when it can include provider payloads, environment values, auth redirects, or local secrets. Tests inject synthetic secret markers to verify they never reach stdout/stderr or checked-in evidence.

<a id="arch-10"></a>
## ARCH-10: Table, diagnostics and exit behavior

Related: [REQ-010](../specs/README.md#req-010) through [REQ-012](../specs/README.md#req-012), [REQ-024](../specs/README.md#req-024), [REQ-052](../specs/README.md#req-052). Owners: [TASK-010](../roadmap/README.md#task-010), [TASK-029](../roadmap/README.md#task-029).

For the plain table, render only after targets have terminal results, in their original order. Use `text/tabwriter` or an equivalent small standard-library implementation to produce space-aligned columns. Padding is cosmetic; headings, cells, order, logical line count, and final newline are contractual. Narrow terminals may visually wrap a long line; the table renderer must not insert additional line breaks to accommodate terminal width. [S-09](#s-09)

Sanitize all externally supplied labels before rendering: replace control/format characters and non-space whitespace with safe spaces, trim, and collapse whitespace. Never emit ESC or bidi formatting controls. A blank sanitized configured alias is invalid input. Preserve ordinary printable Unicode characters. The README must warn that stdout can contain account identifiers supplied by the user.

Illustrative partial result (configured account aliases are `work`, `personal`, and `default`):

```text
AI PROVIDER  ACCOUNT               WEEKLY USAGE
ChatGPT      work                  37%
ChatGPT      personal              AUTH REQUIRED
Grok         default               N/A (monthly)
Claude       default               43.2%
```

The corresponding stderr lines are shown below. Each line has the form `gaebokchi: <target_id>: <CELL>: <message>`, and provider messages begin with their own target ID:

```text
gaebokchi: codex:1: AUTH REQUIRED: codex:1: authenticate using the provider CLI outside gaebokchi
gaebokchi: grok:0: N/A (monthly): grok:0: grok account reports a monthly period, not weekly
```

For usage `--json`, render one compact, newline-terminated object with a
`results` array in target order. Each element has exactly `target_id` (the
invocation-local ID), `provider` (`codex`, `grok`, `claude`), `account` (the
normalized, disambiguated display label), `status` (the domain status string),
`used_percent` (an unrounded JSON number for an OK measurement or the
[REQ-034](../specs/README.md#req-034) Grok weekly omission fallback, otherwise
`null`), and `cleanup_failed` (boolean). A reported zero and the Grok fallback
both encode as numeric zero; the fallback is not a measurement.
Build this explicit projection rather than serializing internal results;
exclude source, reset time, quota ID, timestamps, diagnostics, and raw
provider data. Encoding errors and stdout write failures are output failures
under the existing exit policy. Usage errors leave stdout empty. Help stays
text, and `version --json` keeps its independent two-field schema.

Emit diagnostics in target order, at most one bounded line per target plus a command-level failure if needed. Include no timestamps, progress spinners, raw paths to secrets, or raw upstream errors. A successful numeric row needs no diagnostic unless its cleanup failed. Non-applicable/unavailable outcomes may have a fixed explanatory diagnostic.

| Exit code | Meaning |
|---|---|
| `0` | Valid invocation; all rows are numeric or explicitly non-applicable (monthly/API-key), or help/version was requested |
| `1` | One or more incomplete/error rows, a row whose cleanup failed, internal command execution failure, JSON encoding failure, or stdout write/flush failure |
| `2` | Invalid arguments or unresolvable command-level configuration; launch no providers and print no table |
| `130` | Invocation cancelled by SIGINT |
| `143` | Invocation cancelled by SIGTERM |

On signal cancellation, the first handled signal determines the cancellation exit code; it takes precedence over partial provider failure. If stdout is unwritable, a complete table cannot be guaranteed. Without a cancellation signal, return 1 for output failure and write only a safe diagnostic where possible. Test this policy, including broken-pipe behavior, rather than assuming platform default signal handling matches it.

A normal missing executable/home is target-specific, not a usage error. For a configuration with two Codex accounts, one Grok account, and one Claude account, a missing Grok executable produces both Codex rows, a Grok `NOT INSTALLED` row, and the Claude row, with exit 1.

<a id="arch-10-terminal"></a>
### Interactive terminal view

Related: [REQ-053](../specs/README.md#req-053) through
[REQ-055](../specs/README.md#req-055). Owner:
[TASK-030](../roadmap/README.md#task-030) in
[EPIC-010](../roadmap/README.md#epic-010), whose inline design the owner
approved on 2026-09-25. The table rules above govern non-interactive, non-JSON
stdout.

**Selection.** `cmd/gaebokchi` decides after option parsing. `--json` keeps
the JSON projection. Otherwise the view applies only when stdout is an
`*os.File` whose `TIOCGWINSZ` query succeeds with nonzero columns and rows and
`TERM` is neither empty nor `dumb`. Pipes, regular files, `/dev/null`, and
every other stdout get the unchanged table. Keep the decision in a pure,
table-tested function, and keep the ioctl in a `//go:build unix` file,
following `internal/process/process_unix.go`. A non-empty `NO_COLOR`
suppresses the color parameters only; bold, faint, glyphs, and layout remain.
[S-11](#s-11)

**Rendering model.** `internal/render` builds each frame from lines of styled
spans, so display width, clipping, and styling are computed without parsing
escape sequences, and the colorless frame has the same layout. Reuse
`SanitizeLabel`, `UsageCell`, `Status.Cell`, `Status.ExitClass`, and
`ProviderKind.DisplayName`. Count cells with `golang.org/x/text/width`: East
Asian Wide and Fullwidth runes take two cells, nonspacing and enclosing marks
take none, and every other rune, including East Asian Ambiguous, takes one.
[S-13](#s-13) Write the ANSI output directly instead of adopting a TUI
framework. `golang.org/x/sys` (window size) and `golang.org/x/text` (display
width) become the only direct runtime modules outside the standard library.
Both are already pinned in `go.sum`, so the offline gate needs no network.

**Frame.** The frame is a blank line, the title `🐡 gaebokchi · weekly usage`,
a blank line, one row per target in target order with any reset shown inline or
directly below its target, and a trailing blank line.
A target row is a two-space margin, the provider display name (blank when the
previous row has the same provider), and the account label padded to the widest
label, separated by two spaces. Numeric rows continue with the usage bar, the
`UsageCell` percentage right-aligned in six cells. Other rows continue with
their status cell text. The bar is 20 cells of `█` over a `░` track, rounded
to the nearest whole cell; the percentage retains its precision, including
when a small nonzero value rounds to an empty fill. A qualified reset countdown
appears either after the percentage or on a four-space-indented line directly
below its account row. It uses whole truncated units (`↻ 2d 4h`): days and
hours, hours and minutes, minutes, `<1m`, or `now` once the reset time has
passed. It appears only on
`OK` rows whose `ResetAt` came from the provider (Codex `resetsAt`, Grok period
end, or the reset text on Claude's all-models weekly line). Grok monthly rows
show none.

| Element | Style (SGR parameters) |
|---|---|
| Title | `gaebokchi` bold (1); `· weekly usage` faint (2) |
| Provider name | bold (1) |
| Bar fill and percentage | green (32) below 50, yellow (33) from 50 to below 80, red (31) at 80 and above; the percentage is also bold |
| Bar track, countdown, spinner, `querying…` | faint (2) |
| `N/A (monthly)`, `N/A (API key)`, `CANCELLED` | faint (2) |
| `AUTH REQUIRED`, `N/A (no weekly)`, `N/A (period)`, `N/A (no data)` | yellow (33) |
| Every other status cell | red (31) |

**Width and height.** Every row must fit within `columns - 1` cells. Show all
qualified resets inline only when every reset row fits a full 20-cell bar,
percentage, and countdown within that limit. Otherwise show all qualified
resets below their account rows and compute bar width without the countdown.
When a numeric row does not fit, shrink the bar from 20 toward 10 cells, then
drop the bar if it still does not fit.
Intermediate live frames clip remaining overflow and end it with `…`. The
final frame never truncates an account label; it may wrap instead, because
nothing redraws after it. When the maximum frame height (targets + 4, plus one
possible reset line per target) is not smaller than the terminal
rows, skip live redraws and write only the final frame once.

**Live progress.** `app.Runner` has an optional observation callback that
receives a target's index and terminal result as soon as its slot is set,
including targets cancelled while queued. Several goroutines may call it at
once, and it never changes admission, deadlines, or attempts. The live
renderer hides the cursor and draws pending rows from each target's display
name and configured account alias with a braille spinner and `querying…`. The spinner
advances about every 80 ms, and all drawing happens under one mutex. Each frame
is a single write: carriage return, cursor up by the previously drawn line
count, erase line plus content for every line, then erase below, which also
clears a `^C` echo. After `Run` returns, the renderer draws the final frame
from the ordered results and shows the
cursor again. The first write error stops further drawing and becomes the
command's output failure under the exit policy above. [S-12](#s-12)

**Allowed sequences.** SGR (`ESC[…m`), cursor up (`ESC[nA`), erase line
(`ESC[2K`), erase below (`ESC[J`), and cursor visibility (`ESC[?25l`,
`ESC[?25h`), plus carriage return and newline. Emit no alternate screen,
terminal queries, mouse modes, or other controls. Labels still pass the
display-label policy above, and stderr diagnostics, help flags and behavior, `version`, and exit
codes do not change. Help usage wording describes the new view.

**Known limitations.** The block elements are East Asian Ambiguous, so bars
misalign in terminals configured to draw ambiguous characters at double width.
Narrowing the window during live redraws can leave reflowed lines above the
final frame. Claude reports its reset time only as minute-resolution
human-readable text without a UTC offset, so its countdown can reach `now` up
to a minute early. A reset at a local time that a daylight-saving change
repeats shows no countdown, as do reset text the parser does not recognize
and a zone the host cannot load; the
[compatibility record](../verification/claude-compatibility.md) lists every
discard case.

<a id="arch-11"></a>
## ARCH-11: Verification catalog

Related: [REQ-043](../specs/README.md#req-043) through [REQ-048](../specs/README.md#req-048), [REQ-056](../specs/README.md#req-056) through [REQ-059](../specs/README.md#req-059). Tests are created with the owning implementation task; [TASK-011](../roadmap/README.md#task-011) adds cross-component qualification rather than postponing unit tests.

The following identifiers are referenced by individual requirements and tasks. Implementation records exact test names and commands under these IDs.

<a id="v-contract"></a>
### V-CONTRACT: External contract probe

Record versions, executable identities, successful prompt-free exchanges, precise Grok extension encoding, relevant environment filtering, optional fields, and process ownership. Sanitize before saving. The presence of a source method is not a passed live probe. An unsupported result is a gate failure for positive provider support.

<a id="v-options"></a>
### V-OPTIONS: Target and flag policy

Cover required and invalid YAML, null provider lists, disabled providers, configured order, absolute and `~/` paths, relative-path rejection, empty names or paths, duplicate names or symlinked paths (including a missing suffix under an existing link), missing/inaccessible homes, scalar duplicates, invalid duration/ranges, removed and unknown flags, positional arguments, help/version without configuration, and explicit binary paths.

<a id="v-rpc"></a>
### V-RPC: Framing and protocol dialects

Use a Go fake child executable or test-binary helper subprocess, not a live CLI. Script fragmented/coalesced messages, CRLF, notifications before replies, unknown/duplicate IDs, errors, EOF, null results, `error: null` without a result, error objects with missing or mistyped members, missing or incompatible ACP version markers, malformed/invalid UTF-8 JSON, banners, oversized messages, stderr flooding, and hanging readers/writers. Assert Codex lacks `jsonrpc`, Grok includes it, and only the relevant handshake occurs.

<a id="v-lifecycle"></a>
### V-LIFECYCLE: Finite process ownership

Test early exit, timeout during startup/handshake/read, closed stdin, SIGINT/SIGTERM, child ignoring EOF/SIGTERM, inherited pipe holders, reaping, descriptor/goroutine cleanup, and untouched unrelated/shared processes. Use short injected deadlines and synchronization, not long sleeps. Validate the configured budget and cleanup bound with reasonable scheduler tolerance recorded in tests.

<a id="v-quota"></a>
### V-QUOTA: Pure normalization

Cover explicit 0/100/fractional values, missing/null/wrong-type values, signed zero, negative/over-100 values, no ratio conversion, unknown additive fields, missing/malformed optional reset metadata, and unavailable versus non-applicable semantics. Distinguish Grok's omitted weekly percentage fallback from an explicit null percentage.

<a id="v-codex"></a>
### V-CODEX: Codex contract

Cover two simultaneous homes with different identity/quota values; authenticated, null, API-key and unknown account types; authentication errors; primary-only weekly, secondary-only weekly, swapped slots, five-hour-only, null secondary, no duration, non-empty map with/without codex, empty map legacy behavior, explicit unrelated legacy bucket, duplicate identical weekly windows, conflicting weekly windows, invalid selected bucket, and absent versus zero usage. Assert no thread/turn methods occur.

<a id="v-grok"></a>
### V-GROK: Grok contract

Cover cached-token negotiation, API-key-only advertisement, interactive-only login, incompatible protocol versions, exact extension encoding, method-not-found, successful weekly/monthly config, absent/unknown period, legacy-only monthly fields, valid zero, omitted weekly percentage, explicit null percentage, invalid values, null config, unexpected callbacks, and provider errors. Assert no session creation, prompt, login fallback, or REST call.

<a id="v-claude"></a>
### V-CLAUDE: Claude contract

Use a fake `claude` executable. Cover logged-out, API-key, and unrecognized auth modes, none of which may launch the usage command; a missing executable; a usage timeout; the exact two command lines; and the result metadata. For the usage envelope and weekly line, cover other meters beside the all-models line, decimals, a missing or duplicated line, values above 100 and negative values, a model turn, a nonzero cost, non-JSON output, missing envelope fields, and another local command. For reset text, cover whole hours, minutes, noon and midnight, an explicit year, the zone's year, a year change during the command, the comma join, daylight-saving gaps and overlaps, an unknown zone, unrecognized text, an impossible date, an absent suffix, and another meter's reset. Assert that the child environment drops Anthropic credential and routing overrides, sets `CLAUDE_CONFIG_DIR` for custom homes, and leaves it unset for the default home.

<a id="v-agy"></a>
### V-AGY: agy qualification and integration

TASK-034 owns the metadata-only interface (`agy --print /usage --output-format json --print-timeout 15s`),
runtime version guard (`>= 1.2.15`), update opt-out (`AGY_CLI_DISABLE_AUTO_UPDATE=true`),
and absent-login fast-fail behavior for the single default CLI login. TASK-037
owns fake-process tests for its frozen invocation, single shared query execution
populating `agy:0` (`gemini`) and `agy:1` (`claude/gpt`), child environment
filtering, partial pool failure, absent login, secret exclusion and lifecycle.
TASK-038 verifies positive weekly measurements for both `agy` rows through the
final executable.

<a id="v-cursor"></a>
### V-CURSOR: Planned Cursor qualification and integration

TASK-039 owns the metadata-only subprocess + read-only `NODE_OPTIONS` preload
hook interface (`<cursor-bin> status --format json` intercepting
`./src/commands/status.ts` to read `dashboardClient.getPlanInfo` and
`dashboardClient.getCurrentPeriodUsage`), native weekly/monthly period
classification (`INCLUDED_USAGE_PERIOD_WEEKLY` / 7-day cycle vs
`INCLUDED_USAGE_PERIOD_MONTHLY` / monthly cycle), single-row `Included`
(`totalPercentUsed`, `QuotaID: "cursor_included"`) mapping, `CURSOR_*`
environment filtering, PII omission, and graceful `StatusUnsupported` fallback
when the hook does not fire. TASK-040 owns fake-process coverage for that
contract, weekly (`StatusOK`) and monthly (`StatusNotApplicableMonthly`)
fixtures, `includedSpend/limit` fallback, API-key non-applicability
(`StatusNotApplicableAPIKey`), unauthenticated/partial auth
(`StatusAuthRequired`), missing-data/unsupported cases, spend exclusion,
secrets, and child cleanup. TASK-041 owns positive final-executable weekly usage
verification and existing-provider regressions.

<a id="v-identity"></a>
### V-IDENTITY: Display identity

Cover configured aliases on every provider and status, Codex protocol email not replacing the alias, duplicate normalized aliases rejected before launch, different homes not merged, Unicode, and control characters that would otherwise inject lines or columns.

<a id="v-runner"></a>
### V-RUNNER: Fan-out and isolation

Use controlled fake completion order. Verify at most three active targets, immediate admission of the first three configured targets while later targets wait for a free slot, queued target cancellation, deterministic output order, independent environments, one query attempt per target, and no cancellation of healthy siblings after one failure.

<a id="v-output"></a>
### V-OUTPUT: Output contracts

Golden-test numeric and mixed-status outputs. Assert three headings, no separator/banner/color, exactly N+1 logical lines for N configured targets, and a final newline. Check redirected stdout, Unicode/control sanitization, and diagnostics exclusively on stderr.
For usage JSON, assert the exact field set, target order, number/null semantics,
label normalization, privacy boundary, and one final newline.
For the terminal view, test colorless and styled frames, reset provenance,
display width and clipping, every status and color boundary, the ANSI
allowlist, and the final visible screen after live updates and cancellation.

<a id="v-exit"></a>
### V-EXIT: Exit contract

Test all-clean, all-non-applicable, partial failure, all failure, invalid usage, help/version, broken stdout, SIGINT, and SIGTERM. Assert code and output together rather than testing only internal status enums.

<a id="v-safety"></a>
### V-SAFETY: Read-only and secret boundary

Assert complete outbound method allowlists, false host capabilities, no raw secret leakage, no shell invocation, no provider credential/config reads or writes by gaebokchi, read-only access to the user-authored account configuration, no account-home creation, isolated environments, neutral cwd, and untouched unrelated processes. Static package/dependency inspection complements black-box tests; tests must not claim to sandbox trusted provider internals.

<a id="v-e2e"></a>
### V-E2E: Executable-level offline acceptance

Build gaebokchi into a temporary directory and point the Codex, Grok, Claude, and `agy` executable flags at the test binary acting as a scripted fake provider. Run the actual command with a temporary configuration and provider homes. Verify configured multi-account output, an omitted provider, one failing home, missing CLI, monthly Grok, malformed Grok response, slow/hung provider, unsafe callbacks, signal cancellation, and concurrent account isolation. Through real OS pipes, also verify a broken stdout pipe, each signal with unwritable stdout, and mixed completion under cancellation. Normal CI must not use an actual logged-in home.

<a id="v-build"></a>
### V-BUILD: Reproducible local builds

Run formatting checks, `go vet ./...`, `go test ./...`, native race tests where the selected toolchain supports them, and CGO-disabled cross-builds for all four release targets. Record actual command exit statuses. Ensure the binary has version metadata and no bundled provider binaries.

<a id="v-platform"></a>
### V-PLATFORM: Native operating-system qualification

Execute the fake-provider integration suite natively on macOS and Linux. Record architecture and compiler for each. Other cross-built architectures remain build-qualified until native execution evidence exists. Do not silently label them live-provider tested.

<a id="v-live"></a>
### V-LIVE: Explicit owner-account campaign

On an authorized account host, run the candidate executable against both distinct Codex homes and the existing Grok and Claude logins without submitting a model prompt. Require four ordered `ok` rows with weekly values, no cleanup failure, and no stderr diagnostic. Record the binary revision and build state, provider CLI versions, platform, and command outcome without retaining identifiers or raw responses. A monthly Grok account can qualify monthly provider behavior, but not this four-weekly-row release gate; source-backed weekly fixtures do not become live weekly evidence. Never use unsupported/timeout as a successful integration result. The provider's own usage display can provide additional comparison evidence, but the owner removed that comparison from the v0.1.0 release gate on 2026-09-29.

<a id="v-docs"></a>
### V-DOCS: User-facing documentation

Check README commands against the binary, examples against fixtures, status/exit descriptions against tables, supported/tested versions against evidence, and privacy/side-effect statements against the verified process behavior.

<a id="v-audit"></a>
### V-AUDIT: Cross-document closure

Check every REQ/ARCH/EPIC/TASK reference and local Markdown fragment, every requirement's owner/evidence, all task completion gates, all deferred items, and the final release matrix. A skipped check remains skipped, not passed. This is a bounded audit, not a new design cycle.

### Verification commands

The canonical offline gate is `make test`. `TESTING.md` maps its prepare, unit, integration, and E2E stages, the race-instrumented Ginkgo suites, and the required tools. During a change, run the focused package suites it names. Keep the project-standard full offline suite quick; this scope does not justify hour-long external-agent validation or provider polling.

<a id="arch-12"></a>
## ARCH-12: Compatibility gate and evidence policy

Related: [REQ-005](../specs/README.md#req-005), [REQ-042](../specs/README.md#req-042), [REQ-048](../specs/README.md#req-048), [REQ-056](../specs/README.md#req-056), [REQ-060](../specs/README.md#req-060), [REQ-064](../specs/README.md#req-064), [REQ-065](../specs/README.md#req-065). Owners: [TASK-001](../roadmap/README.md#task-001), [TASK-013](../roadmap/README.md#task-013), [TASK-034](../roadmap/README.md#task-034), [TASK-038](../roadmap/README.md#task-038), [TASK-039](../roadmap/README.md#task-039).

Compatibility status; the [compatibility record](../verification/compatibility.md), the [Claude compatibility record](../verification/claude-compatibility.md), the [agy compatibility record](../verification/agy-compatibility.md), and the [Cursor compatibility record](../verification/cursor-compatibility.md) hold the details and dates:

| Integration | Public evidence | Installed CLI qualification |
|---|---|---|
| Codex account/rate-limit read | Official App Server docs | QUALIFIED live (TASK-001, reconfirmed TASK-013) |
| Two isolated Codex homes | Official CODEX_HOME configuration model | QUALIFIED live (distinct emails/quotas) |
| Grok cached-token ACP auth | Official headless/ACP docs | QUALIFIED live |
| Grok billing handler and payload | Official source snapshot | QUALIFIED live (weekly enum + live percentage) |
| Grok extension wire form through installed stdio | ACP rules plus source candidate | QUALIFIED live as `_x.ai/billing` with `params:{}` |
| Grok 1.0.41 weekly response | Local 1.0.41 source checkout | Metadata-only live probe: weekly period with the percentage member omitted; handled by the ARCH-08 display convention |
| Claude auth status and `/usage` local command | Claude Code command, CLI, and environment references ([S-14](#s-14) through [S-16](#s-16)) | Observed live with Claude Code 2.1.280 on darwin/arm64 |
| Claude weekly reset text | Installed CLI formatter behavior | Observed live with Claude Code 2.1.283 on darwin/arm64 |
| `agy` single-account `/usage` JSON weekly pools (`gemini-weekly`, `3p-weekly`) | Antigravity CLI headless, usage, and troubleshooting references ([S-17](#s-17) through [S-19](#s-19), [S-22](#s-22), [S-23](#s-23)) | QUALIFIED live with `agy 1.4.0` on darwin/arm64 (TASK-034, TASK-038) |
| Cursor single-account `status --format json` + read-only `NODE_OPTIONS` preload hook (`Included` / `totalPercentUsed`, weekly vs monthly period) | Cursor CLI parameters, usage limits, ACP references ([S-20](#s-20), [S-21](#s-21), [S-24](#s-24)), and installed bundle inspection (`./src/commands/status.ts`, `./src/usage/usage-data.ts`) | QUALIFIED live with `cursor-agent 2026.10.16-e3cb207` on darwin/arm64 (TASK-039; final executable acceptance in TASK-041) |
| macOS/Linux lifecycle behavior | Proposed architecture | QUALIFIED darwin/arm64 + linux/arm64 (containerized native suites, TASK-021); TASK-031 and TASK-032 reran the full gates on both; TASK-033 reran them on its corrected uncommitted candidate |

Run small, bounded, developer-only metadata probes before production transport or provider work, as TASK-001 did for Codex and Grok. A disposable standard-library script is acceptable and is not a shipped command. Do not install/upgrade CLIs, relink accounts, launch a model, or edit auth configuration as part of a probe.

Record each qualification in a verification record, such as `docs/verification/compatibility.md` or `docs/verification/claude-compatibility.md`:

```text
provider, observed_at, OS/architecture
resolved executable path (local only) and version output
public source revision/reference, when available
request envelope, initialization/auth flow, successful response shape
Grok internal name versus actual wire name and wrapper
supported auth mode and weekly/monthly semantics
normal exit, timeout cleanup, shared-leader observations
environment variable names filtered (never values)
sanitized fixture paths; each fixture tagged captured or synthetic
qualification status, limitations, and next action
```

Saved transcripts must be field-allowlisted and scrubbed **before** being written into a repository. Replace real emails/IDs with deterministic placeholders; retain semantic numbers/types and relative ordering. Do not save raw auth parameters, redirect URLs, HTTP headers, or environment dumps. Do not commit an unsanitized capture and rely on deleting it later.

Gate outcomes:

| Outcome | Action |
|---|---|
| Each CLI exposes the required prompt-free metadata | Freeze observed contracts; complete the qualification |
| Grok returns valid monthly billing | Qualify monthly behavior; retain weekly fixture qualification and label live limitation |
| Method name/wrapper was encoded incorrectly | Correct the probe from matching source, then perform one bounded confirmation |
| Correctly encoded Grok method is unsupported | Block positive integration; record evidence and needed compatible CLI capability |
| Missing executable/login or host access | Mark gate BLOCKED/NOT RUN; do not pretend fixtures are a live substitute |
| Only token extraction, REST fallback, session creation, or account mutation would work | Stop; retain scope boundary and report the specific gap |

A human-approved scope change is required to drop a provider from a release, add an alternate Grok access route, or change how Claude usage is read ([ADR-0001](../architecture-decision-records/ADR-0001-claude-usage-local-command.md)). That is not delegated to the implementation AI. A verified source-backed clarification of an envelope within the same metadata-only interface is a normal TASK-001 contract update, not a product redesign.

The early probe validates feasibility; TASK-013 validates the delivered application. Both matter. Store tool/source versions separately, and never treat a repository commit SHA as the installed release version. Runtime behavior is capability-driven; the README states tested versions, not unproven universal compatibility.

<a id="arch-13"></a>
## ARCH-13: Build, platform qualification and delivery

Related: [REQ-001](../specs/README.md#req-001), [REQ-046](../specs/README.md#req-046), [REQ-047](../specs/README.md#req-047), [REQ-049](../specs/README.md#req-049), [REQ-051](../specs/README.md#req-051). Owners: [TASK-012](../roadmap/README.md#task-012), [TASK-028](../roadmap/README.md#task-028).

The toolchain is Go 1.27 (`go.mod`), pinned as 1.27.1 in CI and listed with the other required tools in `TESTING.md`; do not assert the latest release number from memory. Prefer standard-library dependencies and `CGO_ENABLED=0` release builds. Go's GOOS/GOARCH target model supplies the build axes; native testing remains separate. [S-10](#s-10)

| Artifact target | Required build evidence | Required runtime evidence |
|---|---|---|
| darwin/arm64 | Cross/native build succeeds | Native macOS evidence on available runner architecture |
| darwin/amd64 | Cross/native build succeeds | Explicitly label native-tested or build-only |
| linux/arm64 | Cross/native build succeeds | Explicitly label native-tested or build-only |
| linux/amd64 | Cross/native build succeeds | Native Linux evidence on available runner architecture |

Native macOS and native Linux tests are mandatory, but native execution on all four CPU/OS combinations is not. Keep a per-target matrix so users can see the difference. TASK-031 and TASK-032 ran full gates on darwin/arm64 and linux/arm64, as did TASK-033's corrected uncommitted candidate. Re-run both native gates on the final release commit before publishing. Live provider acceptance must cover the owner account set on at least one supported host; do not imply live support was tested on the other host or every version.

Resolve executable version metadata in this order: explicit release linker
override, Go main-module build information, then `(devel)`. Release linker
metadata and versioned staging paths use the canonical `v` prefix; the v0.1.0
staging directory is `dist/gaebokchi-v0.1.0`. Public version JSON does not
expose commit, revision, module checksum, or build date.

Produce versioned local archives with the `gaebokchi` prefix and SHA-256 checksums, containing the `gaebokchi` executable, README, LICENSE, THIRD_PARTY_NOTICES.txt, configuration guide, and the three specification documents. Verify before staging that every packaged text file is byte-identical to its source (`scripts/archivecheck.go`). The project license is MIT (`LICENSE`); the archives also carry license and notice texts for bundled dependencies.

Publication is a manual, owner-approved step: the staged archives and `SHA256SUMS` are uploaded to GitHub Releases for `github.com/irootkernel/gaebokchi`, following the procedure in the [implementation tips](../implementation-tips/README.md). The build never publishes, tags, signs, or redistributes provider binaries, and no package-manager tap exists.

README startup instructions assume the user installs and logs into the provider CLIs separately. Explain the required configuration file, account aliases, per-provider home paths, disabled providers, PATH/binary flags, subscription-only interpretation, unsupported protocol errors, and potential provider-owned startup writes.

<a id="arch-14"></a>
## ARCH-14: Change and completion discipline

Related: [REQ-050](../specs/README.md#req-050). Owner: [TASK-014](../roadmap/README.md#task-014).

The three canonical role documents are the specification authority; implementation evidence lives under `docs/verification/`, and decision rationale that should survive a rewrite of this document lives in an [architecture decision record](../architecture-decision-records/README.md). Approved epic dossiers may map work to these owners, but must not create competing lifecycle state, acceptance authority or approval frameworks.

For a contract change, update the relevant requirement, architecture section, task scope/test mapping, and fixtures together, and add a dated Changed note to each affected requirement. Preserve stable IDs. Do not mark later work complete because a helper exists, and do not reopen a completed epic for unrelated improvements.

The implementation AI follows one epic to closure before starting the next. Each task is one coherent commit, including its focused tests and associated document/evidence updates. An Aquarium epic-handler may orchestrate this plan, but Aquarium, Podway, or any external review service is not a product dependency.

Use one planned review where appropriate, fix bounded blocking findings, and allow at most one confirmation review of the changed scope. Remaining blockers are reported with evidence, not fed into a new automatic full review loop. Required deterministic tests are not waived by an absent external reviewer.

<a id="arch-15"></a>
## ARCH-15: Primary-source references

These sources were consulted for this specification on 2026-09-22, and S-11 through S-13 on 2026-09-25 for the terminal view. S-14 through S-16 are the references cited by the [Claude compatibility record](../verification/claude-compatibility.md). All of these sources document external interfaces, not the completion state of this project. The implementation must record observed binary versions separately. Source links can move; retain the fixed source revision used for fixtures.

<a id="s-01"></a>
**S-01: OpenAI, Codex App Server.** Stdio framing, initialization, account reads, and quota fields.  
https://developers.openai.com/codex/app-server/  
Current redirect consulted: https://learn.chatgpt.com/docs/app-server

<a id="s-02"></a>
**S-02: OpenAI, Advanced Configuration.** CODEX_HOME and provider-managed state locations.  
https://developers.openai.com/codex/config-advanced/

<a id="s-03"></a>
**S-03: xAI, Headless & Scripting.** ACP stdio, cached-token authentication, headless metadata, and no-auto-update option.  
https://docs.x.ai/build/cli/headless-scripting

<a id="s-04"></a>
**S-04: Agent Client Protocol, v1 Extensibility.** Underscore-prefixed extension wire methods and method-not-found semantics.  
https://agentclientprotocol.com/protocol/v1/extensibility

<a id="s-05"></a>
**S-05: xai-org/grok-build, billing extension implementation.** Internal handler, period and percentage fields, optional metadata. Inspected public repository commit: `4247f661689354b831191f11eeeac8424993fe3d`. This is a source revision, not a tested CLI version.  
https://github.com/xai-org/grok-build/blob/4247f661689354b831191f11eeeac8424993fe3d/crates/codegen/xai-grok-shell/src/extensions/billing.rs

<a id="s-06"></a>
**S-06: xai-org/grok-build, leader protocol and ACP routing.** Extension prefix distinction and provider-managed process boundary. Same source revision as S-05.  
https://github.com/xai-org/grok-build/blob/4247f661689354b831191f11eeeac8424993fe3d/crates/codegen/xai-grok-shell/src/leader/protocol.rs  
https://github.com/xai-org/grok-build/blob/4247f661689354b831191f11eeeac8424993fe3d/crates/codegen/xai-grok-shell/src/agent/mvp_agent/acp_agent.rs

<a id="s-07"></a>
**S-07: Go, os/exec.** Explicit subprocess execution, environment handling, waiting and current-directory lookup protection.  
https://pkg.go.dev/os/exec

<a id="s-08"></a>
**S-08: Go, encoding/json.** JSON encoding/decoding primitives; the application's presence and framing policy is defined in ARCH-04/06.  
https://pkg.go.dev/encoding/json

<a id="s-09"></a>
**S-09: Go, text/tabwriter.** Standard-library text-column formatting.  
https://pkg.go.dev/text/tabwriter

<a id="s-10"></a>
**S-10: Go, installing from source, environment section.** GOOS/GOARCH build targets.  
https://go.dev/doc/install/source#environment

<a id="s-11"></a>
**S-11: NO_COLOR.** A present, non-empty `NO_COLOR` prevents added ANSI color.  
https://no-color.org/

<a id="s-12"></a>
**S-12: XTerm Control Sequences.** Cursor up (CUU), erase in line (EL), erase in display (ED), cursor visibility (DECTCEM), and character attributes (SGR).  
https://invisible-island.net/xterm/ctlseqs/ctlseqs.html

<a id="s-13"></a>
**S-13: Unicode Standard Annex #11, East Asian Width.** Wide, fullwidth, and ambiguous display-width classes.  
https://www.unicode.org/reports/tr11/

<a id="s-14"></a>
**S-14: Anthropic, Claude Code commands.** Documents `/usage` as a built-in in-session command.\
https://code.claude.com/docs/en/commands

<a id="s-15"></a>
**S-15: Anthropic, Claude Code CLI reference.** Print mode, JSON output, safe mode, and zero-turn limits.\
https://code.claude.com/docs/en/cli-reference

<a id="s-16"></a>
**S-16: Anthropic, Claude Code environment variables.** `ANTHROPIC_API_KEY` overrides a subscription in print mode.\
https://code.claude.com/docs/en/env-vars

### Sources for the planned provider extensions

The following sources inform the 2026-10-15 design and its 2026-10-16
preflight supplement. They do not qualify an installed executable or establish
a metadata response schema. Record retrieval dates and distinguish older cached
pages from current plan variants; runtime response evidence remains authoritative.

<a id="s-17"></a>
**S-17: Google Antigravity, Model quotas.** Interactive `/usage` panel and model quota display.\
https://antigravity.google/docs/cli/commands/usage/

<a id="s-18"></a>
**S-18: Google Antigravity, Installation and auth.** CLI-managed keyring authentication.\
https://antigravity.google/docs/cli/install/

<a id="s-19"></a>
**S-19: Google Antigravity, Plans.** Five-hour and weekly baseline limits.\
https://antigravity.google/docs/plans/

<a id="s-20"></a>
**S-20: Cursor, CLI parameters.** Public CLI command discovery.\
https://cursor.com/docs/cli/reference/parameters

<a id="s-21"></a>
**S-21: Cursor, Usage and limits.** Plan-dependent weekly or monthly pools, including pools shared by several models or products, and separate overage usage. The 2026-10-16 retrieval included an older monthly-only cached page and a newer official page distinguishing weekly and legacy monthly plans; neither identifies the owner's login or live quota.\
https://cursor.com/help/models-and-usage/usage-limits\
https://prod.cursor.com/help/models-and-usage/usage-limits

<a id="s-22"></a>
**S-22: Google Antigravity, Headless mode.** Documents `/usage` as CLI-handled text output rather than a standard event stream, and excludes it from streaming input. This supplies a candidate local-command route, not a qualified structured quota schema or installed-version safety guarantee.\
https://antigravity.google/docs/cli/headless/

<a id="s-23"></a>
**S-23: Google Antigravity, CLI troubleshooting.** Documents `AGY_CLI_DISABLE_AUTO_UPDATE=true` for opting out of automatic updates and describes OS keyring behavior. Installed-version effectiveness still needs qualification.\
https://antigravity.google/docs/cli/troubleshooting/

<a id="s-24"></a>
**S-24: Cursor, ACP.** Documents stdio JSON-RPC, authentication and session operations. Transport availability does not establish a session-free quota method; do not copy its model-prompt example into a quota probe.\
https://cursor.com/docs/cli/acp

<a id="arch-17"></a>
## ARCH-17: Weekly model-pool rows and exact remaining-fraction conversion

**Delivery:** EPIC-012, TASK-035 and TASK-036. **Requirements:** REQ-061 through
REQ-063.

`gaebokchi` preserves its weekly-only presentation across all output modes:
the 3-column plain table (`AI PROVIDER`, `ACCOUNT`, `WEEKLY USAGE`), the
interactive terminal view (`🐡 gaebokchi · weekly usage`), and the existing
`--json` `results` array. Existing Codex, Grok, and Claude normalization and
row semantics remain unchanged.

### Weekly model-pool mapping for `agy`

A single `agy --print /usage --output-format json --print-timeout 15s` response
contains `command.data.groups[]`, where each group has a `buckets[]` list of
windowed entries (`window: "weekly"` or `window: "5h"`). `gaebokchi` extracts
only the two weekly model-group quotas and maps them to two ordered `domain.Result`
rows under provider `agy` (`DisplayName() == "agy"`):

| Target ID | Row label (`Account`) | Matched entry in `command.data.groups[].buckets[]` | Period |
|---|---|---|---|
| `agy:0` | `gemini` | `id == "gemini-weekly"` and `window == "weekly"` | `weekly` (`domain.PeriodWeekly`) |
| `agy:1` | `claude/gpt` | `id == "3p-weekly"` and `window == "weekly"` | `weekly` (`domain.PeriodWeekly`) |

Entries with `window == "5h"` (`gemini-5h`, `3p-5h`) are ignored. Never average
the two weekly pools or select their maximum as a single row. If one weekly pool
is valid and the other is missing or malformed, preserve the valid pool's numeric
row while reporting the affected sibling row as `unavailable/no_weekly` (when
the weekly entry is absent) or `protocol_error` (when its fields are malformed
or conflicting duplicate `id` entries exist). Parse optional RFC 3339 UTC
`reset_time` strings into `ResetAt`; invalid reset timestamps may be discarded
without invalidating a valid percentage.

### Exact decimal remaining-fraction conversion and resource bounds

`agy` reports each quota's remaining allowance as `remaining_fraction` $r \in [0, 1]$.
TASK-035 converts $r$ to `used_percent = 100 * (1 - r)` from the original JSON
decimal token using bounded exact decimal arithmetic before storing or formatting:
- **Resource bounds:** Reject tokens exceeding 128 ASCII bytes, 64 coefficient
  digits, or an explicit exponent outside `[-128, 128]` before expanding powers
  of ten or allocating large buffers. An out-of-budget or out-of-range (`r < 0`
  or `r > 1`) token produces `protocol_error` for that pool row without
  discarding a valid sibling pool row.
- **Exact integer and display behavior:** Computing `100 * (1 - r)` in exact
  decimal arithmetic before converting to the stored percentage guarantees that
  exact integer results (such as `remaining_fraction = 0.9` -> `10%`) never
  suffer IEEE-754 binary floating-point truncation artifacts (`9.999999999999998`
  formatting as `10.0%`). Normalize signed zero to `0`.
- **Required unit-test cases:** `remaining=0.9 -> used=10`,
  `remaining=0.2584281265735626 -> used=74.15718734264374` (display `74.2%`),
  `remaining=0.8723080158233643 -> used=12.76919841766357` (display `12.8%`),
  `0` and `1` endpoints, equivalent exponent forms, signed zero, early rejection
  of `1e-1000000000`, at/over-limit tokens, and unaffected sibling pool rows.

### Two-row presentation and JSON continuity

Because `internal/render/terminal.go`
already blanks `provider` when `result.Provider == previous` and renders `↻`
countdowns for any `StatusOK` result with `Period == PeriodWeekly` and non-nil
`ResetAt`, emitting `agy:0` (`Account: "gemini"`) and `agy:1`
(`Account: "claude/gpt"`) requires no layout changes to `table.go`,
`terminal.go`, `live.go`, or `json.go`:

- **Interactive terminal view:**
  ```text
    agy      gemini      ███████████████░░░░░   74.2%  ↻ 4d 3h
             claude/gpt  ███░░░░░░░░░░░░░░░░░   12.8%  ↻ 6d 3h
  ```
- **Plain table:**
  ```text
  AI PROVIDER  ACCOUNT     WEEKLY USAGE
  agy          gemini      74.2%
  agy          claude/gpt  12.8%
  ```
- **JSON (`--json`):** Keeps the existing `{"results":[...]}` schema with
  `target_id: "agy:0"` (`provider: "agy"`, `account: "gemini"`) and
  `target_id: "agy:1"` (`provider: "agy"`, `account: "claude/gpt"`).

<a id="provider-extension-qualification"></a>
### Shared extension qualification and runtime guards

TASK-034 and TASK-039 freeze a provider contract record before their respective
adapters. The record identifies the actual installed version and platform,
allowed arguments/methods, child environment policy, update suppression,
credential boundary, output schema, period/pool mapping, and the evidence for a
local, session-free execution path.

Where local-command safety or UTC timestamp formatting depends on a minimum CLI
version, run a bounded version check (`--version`) on the resolved executable
within the target timeout budget before invoking the usage command. An
unqualified version must not reach a potentially model-capable invocation:
report `StatusUnsupported` (`UNSUPPORTED`).

All adapters use explicit dispatch: Codex, Grok, Claude, `agy`, and Cursor must
each select their own querier in `Runner.querierFor`. Remove the implicit Codex
fallback; unknown providers or missing adapter wiring fail without launching any
provider process.

<a id="arch-18"></a>
## ARCH-18: agy single-account adapter and shared two-row query

**Delivery:** EPIC-012, TASK-034, TASK-037 and TASK-038. **Requirements:** REQ-060
and REQ-064. The [agy compatibility record](../verification/agy-compatibility.md)
documents the qualified `agy 1.4.0` contract on `darwin/arm64`.

### Configuration and target construction

- `~/.config/gaebokchi/config.yaml` accepts an optional `agy` list with at most
  one entry (for example, `- name: default`; `path` is optional and ignored for
  credential selection because `agy` uses the single default CLI login). More
  than one entry in `agy` is a configuration error (exit 2). Omission or `[]`
  disables `agy`.
- The binary flag is `--agy-bin` with default `agy`.
- Provider order is Codex, Grok, Claude, `agy`, then Cursor.
- When `agy` is enabled, `buildTargets` emits two targets in display order:
  1. `Target{ID: "agy:0", Provider: domain.ProviderAgy, FallbackAccount: "gemini", Bin: o.AgyBin}`
  2. `Target{ID: "agy:1", Provider: domain.ProviderAgy, FallbackAccount: "claude/gpt", Bin: o.AgyBin}`

### Shared single-shot subprocess execution and environment policy

- Although `agy` is represented by two weekly rows (`agy:0` and `agy:1`), the
  `internal/provider/agy` adapter executes the underlying `agy` CLI **at most
  once per `gaebokchi` invocation** (`sync.Once` / shared single-flight result
  within the per-target timeout budget) and projects the parsed response onto
  both targets:
  1. Run `<agy-bin> --version` and verify version `>= 1.2.15`. If the version is
     older or unparseable, return `StatusUnsupported` for both rows without
     invoking `/usage`.
  2. Run `<agy-bin> --print /usage --output-format json --print-timeout 15s`
     with stdin closed (`os.DevNull`), an empty private temporary directory
     (`0700`), and bounded stdout/stderr capture (256 KiB).
  3. **Environment filter:** Preserve `HOME` and platform keyring variables so
     `agy` accesses its default login. Inject `AGY_CLI_DISABLE_AUTO_UPDATE=true`.
     Strip all `ANTIGRAVITY_*` (including `ANTIGRAVITY_LS_ADDRESS`,
     `ANTIGRAVITY_CSRF_TOKEN`, `ANTIGRAVITY_APP_DATA_DIR`), `JETSKI_*`, `AGY_*`,
     `GEMINI_API_KEY`, `GOOGLE_GEMINI_BASE_URL`, `AGY_ADC_AUTH`,
     `GOOGLE_APPLICATION_CREDENTIALS`, and `GOOGLE_CLOUD_QUOTA_PROJECT`.
  4. **Absent-login fast fail:** Monitor bounded stderr; if
     `Authentication required` appears (or `AGY_ERROR` indicates missing auth),
     terminate the child process group immediately instead of waiting for `agy`'s
     60-second OAuth prompt timeout, and return `StatusAuthRequired`
     (`AUTH REQUIRED`) for both rows.
  5. **Response validation:** Require `status == "SUCCESS"`, `num_turns == 0`,
     `command.name == "usage"`, and optional top-level `type` (`""` in `json`
     mode or `"result"`). Extract `gemini-weekly` for `agy:0` and `3p-weekly`
     for `agy:1` from `command.data.groups[].buckets[]`.

<a id="arch-19"></a>
## ARCH-19: Planned Cursor single-account adapter and read-only preload hook

**Delivery:** EPIC-013, TASK-039 through TASK-041. **Requirements:** REQ-065 and
REQ-066. The [Cursor compatibility record](../verification/cursor-compatibility.md)
and [ADR-0002](../architecture-decision-records/ADR-0002-period-aware-provider-usage.md)
document the qualified `cursor-agent 2026.10.16-e3cb207` contract on `darwin/arm64`.

### Configuration and target construction

- `~/.config/gaebokchi/config.yaml` accepts an optional `cursor` list with at most
  one entry (`- name: <alias>`; no `path` field is accepted because `cursor-agent`
  uses a global default credential store). More than one entry in `cursor` is a
  configuration error (exit 2). Omission or `[]` disables Cursor.
- The binary flag is `--cursor-bin` with default `cursor-agent`.
- Provider order is Codex, Grok, Claude, `agy`, then Cursor.
- When `cursor` is enabled, `buildTargets` emits one target:
  `Target{ID: "cursor:0", Provider: domain.ProviderCursor, FallbackAccount: entry.Name, AccountSource: domain.AccountSourceConfiguredAlias, Bin: o.CursorBin}`.

### Subprocess preload-hook execution and environment policy

- For `cursor-agent`, the former "pure public CLI flags only" restriction is
  retired because `cursor-agent -p /usage` sends `"/usage"` as an LLM prompt and
  plain `status`/`about`/`acp` omit quota data, whereas `cursor-agent status --format json`
  already initializes `cursor-agent`'s own `credentialManager` and
  `dashboardClient` (`aiserver.v1.DashboardService`) and passes both into
  `./src/commands/status.ts` (`handleStatus`).
- The `internal/provider/cursor` adapter executes `<cursor-bin> status --format json`
  with a read-only `NODE_OPTIONS` preload hook:
  1. Write the embedded read-only `preload.cjs` into a private mode-`0700`
     temporary directory (`os.MkdirTemp("", "gaebokchi-cursor-")`, removed on
     `defer` with any removal error folded into `Result.CleanupFailed`) and set
     `NODE_OPTIONS=--require=<preloadPath>`.
  2. **Environment filter (`envfilter.Cursor()`):** Preserve `HOME` and platform
     keyring variables so `cursor-agent` accesses its default login. Strip all
     `CURSOR_*` variables (`CURSOR_API_KEY`, `CURSOR_AUTH_TOKEN`,
     `CURSOR_CONFIG_DIR`, `CURSOR_STATSIG_OVERRIDES`) and any ambient parent
     `NODE_OPTIONS`, then apply the controlled `NODE_OPTIONS` preload path.
  3. **Read-only hook behavior:** When `cursor-agent status --format json` loads
     `./src/commands/status.ts`, `preload.cjs` wraps `handleStatus(credentialManager, dashboardClient)`,
     checks `getAccessToken()`, `getRefreshToken()`, and `getApiKey()`, and—when
     authenticated without an API key—calls `dashboardClient.getPlanInfo` and
     `dashboardClient.getCurrentPeriodUsage` concurrently. It writes a single
     compact JSON line synchronously to stdout (`fs.writeSync(1, ...)`) with all
     PII (`email`, `userId`, `firstName`, `lastName`, `teamId`, `teamName`,
     tokens, and keys) omitted.
  4. **Graceful fallback:** If a future `cursor-agent` release alters
     `./src/commands/status.ts` so the preload hook does not intercept
     `handleStatus`, the subprocess still returns standard `status --format json`
     output and `internal/provider/cursor` deterministically returns
     `StatusUnsupported` (`UNSUPPORTED`).
  5. **Response normalization (matching the 7 cases in [cursor-compatibility.md](../verification/cursor-compatibility.md)):**
     - **Unhooked output (`StatusUnsupported` / `UNSUPPORTED`):** Valid JSON that
       lacks the preload-hook discriminator fields (`status`, `hasAccessToken`,
       `hasRefreshToken`, `hasApiKey`—such as plain `{"isAuthenticated": true, ...}`
       output when the hook did not fire) -> `StatusUnsupported`.
     - **Unauthenticated or partial credentials (`StatusAuthRequired` / `AUTH REQUIRED`):**
       `!isAuthenticated || !hasAccessToken || !hasRefreshToken` ->
       `StatusAuthRequired`.
     - **API-key authentication (`StatusNotApplicableAPIKey` / `N/A (API key)`):**
       `hasApiKey == true` -> `StatusNotApplicableAPIKey` (`ExitClean`).
     - **ConnectRPC error (`StatusProviderError` / `ERROR`):** Non-null
       `planError` or `usageError` -> `StatusProviderError` (or
       `StatusAuthRequired` only when the structured RPC error `code` is
       explicitly `"unauthenticated"`), without echoing raw provider error text.
     - **Monthly plan (`StatusNotApplicableMonthly` / `N/A (monthly)`):**
       `planInfo.includedUsagePeriod == 1` (`MONTHLY`), or `0` (`UNSPECIFIED`)
       with a 27–32 day `billingCycleEndMs - billingCycleStartMs` window ->
       `StatusNotApplicableMonthly` (`Period: domain.PeriodMonthly`,
       `UsedPercent: nil`, `ResetAt: nil`, `ExitClean`).
     - **Weekly plan (`StatusOK`):** `planInfo.includedUsagePeriod == 2`
       (`WEEKLY`), or `0` (`UNSPECIFIED`) with a 6–8 day
       `billingCycleEndMs - billingCycleStartMs` window -> require
       `currentPeriodUsage.enabled == true` and `currentPeriodUsage.planUsage != nil`,
       extract `UsedPercent` from `planUsage.totalPercentUsed` (or
       `includedSpend / limit * 100` when `totalPercentUsed` is absent/null and
       `limit > 0`), extract `ResetAt` from `planInfo.includedUsageResetsAtMs`
       (falling back to `currentPeriodUsage.billingCycleEndMs`), and return
       `StatusOK` with `Period: domain.PeriodWeekly`,
       `QuotaID: "cursor_included"`, and `Source: domain.SourceCursorCLIUsage`
       (`"cursor_cli_usage"`). Exclude on-demand spend (`spendLimitUsage`) from
       quota percentages.
     - **Malformed, out-of-range, oversized, or unavailable edge cases:**
       Non-JSON stdout, stdout > 256 KiB, `totalPercentUsed < 0` or `> 100`, or
       invalid spend/limit (`limit > 0` with `includedSpend < 0` or
       `includedSpend > limit`) -> `StatusProtocolError` (`PROTOCOL ERROR`);
       `planInfo == nil`, unknown `includedUsagePeriod` enum, or
       `includedUsagePeriod == 0` without a 6–8 day or 27–32 day cycle window ->
       `StatusUnavailablePeriod` (`N/A (period)`); weekly period where
       `currentPeriodUsage == nil`, `currentPeriodUsage.enabled == false`,
       `planUsage == nil`, or `totalPercentUsed == nil` with `limit <= 0` ->
       `StatusUnavailableNoData` (`N/A (no data)`).
