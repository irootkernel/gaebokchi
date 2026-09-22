# Gaebokchi (개복치): Architecture

Version: 0.1 specification  
Status: Implemented as designed; provider runtime compatibility qualified live on 2026-09-22 (see docs/verification/compatibility.md)  
Prepared: 2026-09-22  
Project: Gaebokchi (개복치)  
Ecosystem: Aquarium tool  
Repository and executable: `gaebokchi`

## Current Claude extension

The sections below describe the historical v0.1 Codex/Grok architecture.
The current implementation adds one Claude target after Grok through the
same bounded runner and child-process owner. Its adapter invokes
`claude --safe-mode auth status --json` and then the built-in `/usage` local
command in print mode with `--max-turns 0` and no session persistence. It
requires the JSON result to establish a zero-turn, zero-cost local command
before selecting the exact all-models weekly line. The Claude adapter uses
no RPC dialect, credential parsing, or app-owned HTTP connection. See the
[qualification record](../verification/claude-compatibility.md).

## Document contract and navigation

The [requirements](../specs/README.md) define acceptance. The [roadmap](../roadmap/README.md) defines the sequential implementation route. This document defines the intended implementation and its boundaries. Release archives carry this three-document contract, while links to verification records resolve only in a source checkout. **External interface facts** have sources in [ARCH-15](#arch-15); **project decisions** are normative design choices, not claims that providers guarantee them.

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
| [ARCH-09](#arch-09) | Security and read-only policy |
| [ARCH-10](#arch-10) | Rendering, diagnostics and exits |
| [ARCH-11](#arch-11) | Verification catalog |
| [ARCH-12](#arch-12) | Compatibility discovery and live gates |
| [ARCH-13](#arch-13) | Build and distribution |
| [ARCH-14](#arch-14) | Change and completion discipline |
| [ARCH-15](#arch-15) | Primary sources |

<a id="arch-01"></a>
## ARCH-01: Scope and settled decisions

Related: [REQ-003](../specs/README.md#req-003), [REQ-004](../specs/README.md#req-004), [REQ-005](../specs/README.md#req-005). Implemented across [EPIC-001](../roadmap/README.md#epic-001) through [EPIC-006](../roadmap/README.md#epic-006).

| Decision | v0.1 choice | Consequence |
|---|---|---|
| Product | **Gaebokchi (개복치)**, command-line only | No TUI, service, or account dashboard |
| Ecosystem | Aquarium tool | Standalone; no Aquarium runtime or build dependency |
| Repository and executable | `gaebokchi` | One consistent ASCII name for repository, binary and command directory |
| Language | Go, standard library first | Small build and deployment surface |
| Providers | Codex and Grok only | No Claude/PTY dependency |
| Authentication | Existing provider CLI login | No token parsing or account-linking workflow |
| Codex account set | Two default homes, explicit repeatable override | Different homes use different child processes |
| Grok account set | One existing login | Display alias defaults to `default` |
| Metric | Provider-defined weekly allowance consumed | No cross-provider arithmetic or cost comparison |
| Transport | Local stdio to provider executables | No network listener or app-owned HTTP client |
| Execution | One snapshot, up to three targets concurrently | No periodic refresh or automatic retry |
| Persistence | None in the running application | Developer fixtures/evidence are repository artifacts, not runtime caches |
| Output | Three columns, four lines by default | Failures remain in their account row |

A provider may define a rolling or anchored seven-day allowance rather than a Monday-to-Sunday week. Follow the provider's reported period. Do not promise a shared quota across all ChatGPT or Grok products. A shared/unified Grok pool may be present in the billing response, but the application reports the selected allowance without broadening its interpretation.

**Naming contract:** Use **Gaebokchi (개복치)** for human-readable project titles and `gaebokchi` for the repository basename, executable, `cmd/gaebokchi` entry point, protocol client names, diagnostic prefix and release-artifact prefix. Where a protocol has a separate human-readable client title, use `Gaebokchi`. The Go module import path must still follow the actual repository identity selected in TASK-002; the project name does not authorize inventing a remote repository. Aquarium membership is ecosystem identity, not an additional integration feature or runtime dependency.

Runtime target concurrency is unrelated to development parallelism: implementation tasks remain strictly sequential.

<a id="arch-02"></a>
## ARCH-02: Components and dependency direction

Related: [REQ-001](../specs/README.md#req-001), [REQ-041](../specs/README.md#req-041). Owners: [TASK-002](../roadmap/README.md#task-002), [TASK-012](../roadmap/README.md#task-012).

```text
cmd/gaebokchi (composition root, flags, signals, exit)
    |
    +--> internal/app (target runner, bounded fan-out, stable result slots)
    |        |
    |        +--> internal/provider/codex
    |        +--> internal/provider/grok
    |                    |
    |                    +--> internal/rpc (framing, IDs, dialect codecs)
    |                                 |
    |                                 +--> internal/process (owned child lifecycle)
    |
    +--> internal/render (domain result -> plain table and safe diagnostics)

All application/adapters/render packages depend inward on internal/domain.
The domain does not import a provider, RPC, process, or rendering package.
```

Use a small interface at the application boundary, for example `Query(context.Context, Target) Result`. A provider returns exactly one terminal result for one target; it does not render, terminate the program, mutate global environment, or coordinate other accounts. Expected provider failures belong in a typed result. Setup/argument failures before targets exist remain command-level errors.

Two Codex targets share adapter code but not connection state. The shared RPC layer knows framing and message correlation, not quota fields or provider billing semantics. Provider codecs own handshake differences and outbound allowlists.

Suggested layout, with production files added only when their task needs them:

```text
docs/specs/README.md
docs/architecture/README.md
docs/roadmap/README.md
go.mod
README.md
cmd/gaebokchi/main.go
internal/domain/{target,result,status}.go
internal/options/options.go
internal/app/runner.go
internal/process/child.go
internal/process/process_unix.go
internal/rpc/{client,frame,codec}.go
internal/provider/codex/{provider,wire,normalize}.go
internal/provider/grok/{provider,wire,normalize}.go
internal/render/table.go
internal/testutil/                  # fake-process helpers, not installed
internal/provider/codex/testdata/
internal/provider/grok/testdata/
test/integration/
docs/verification/                 # sanitized task/live evidence
scripts/check-docs.go
scripts/build-release.sh
.github/workflows/ci.yml            # when the repository uses GitHub
```

Prefer constructors and ordinary functions over a service container, plugin registry, event bus, job framework, or generic agent SDK. Avoid a new dependency merely for a three-column table. Test process injection through a narrow factory, not global monkey patching. The actual module path and maintained Go toolchain are recorded by TASK-002, based on the real repository/environment; this specification invents neither a remote repository nor an installed compiler version.

<a id="arch-03"></a>
## ARCH-03: Command contract and account targets

Related: [REQ-002](../specs/README.md#req-002), [REQ-006](../specs/README.md#req-006), [REQ-007](../specs/README.md#req-007), [REQ-008](../specs/README.md#req-008), [REQ-051](../specs/README.md#req-051). Owners: [TASK-002](../roadmap/README.md#task-002), [TASK-010](../roadmap/README.md#task-010), [TASK-028](../roadmap/README.md#task-028).

| Flag | Default | Rules |
|---|---|---|
| `--codex-home PATH` | `~/.codex` and `~/.codex-hsy` | Repeatable; supplying any replaces both defaults |
| `--codex-bin PATH_OR_NAME` | `codex` | Executable only, never a shell command string |
| `--grok-bin PATH_OR_NAME` | `grok` | Same executable resolution policy |
| `--grok-account LABEL` | `default` | Display alias only; does not switch Grok login |
| `--timeout DURATION` | `20s` | 1s through 120s inclusive; one query budget per target |
| `--help`, `-h` | Off | Print help and return 0 without provider execution |

The sole subcommand is `version [--json]`. Human output is `<name> <version>`;
JSON output contains exactly the string fields `name` and `version` followed by
a newline. A release build uses a `v`-prefixed version such as `v0.1.0`, while
an ordinary source build without release metadata reports `(devel)`. The former
`--version` flag is unsupported.

No other positional arguments, provider filter, global output-format flag,
configuration file, or implicit account discovery exists in v0.1. Each
non-repeatable flag may occur once; ambiguous repeated scalar options are usage
errors. Help and version do not validate provider installation or
account-directory existence. Version dispatch occurs before resolving HOME or
the invocation directory, and never launches a provider.

Examples:

```sh
gaebokchi
gaebokchi --timeout 40s
gaebokchi --codex-home "$HOME/.codex-work" --codex-home "$HOME/.codex-personal"
gaebokchi --grok-account personal --grok-bin /opt/homebrew/bin/grok
gaebokchi version
gaebokchi version --json
```

Resolve the invoking user's home through `os.UserHomeDir`. Ambient `CODEX_HOME` does not change the default account list. Preserve the caller's actual HOME for provider-managed login/keychain lookup; Codex's separate CODEX_HOME is not a replacement for HOME.

Expand a leading current-user `~` explicitly. Do not implement `~otheruser`, shell variable expansion, globbing, or command substitution. Explicit relative paths are resolved before changing the child working directory. Clean absolute paths, and resolve symlinks for existing paths to detect duplicate homes. Do not deduplicate different homes because their eventual emails are equal.

Invalid syntax/empty labels/duplicate homes are command errors before starting children. A syntactically valid but missing home yields its own `HOME NOT FOUND` row; an inaccessible path or non-directory yields `HOME ERROR`. A missing executable yields `NOT INSTALLED`; a present but unexecutable binary yields `ERROR`. Never initialize a provider home to repair a missing path.

Each `Target` has a deterministic invocation-local ID (`codex:0`, `codex:1`, `grok:0`), an output index, a provider kind, executable identity, a fallback account label, and a Codex home when applicable. Resolve a provider executable once per invocation and reuse the absolute path for its targets. Respect Go's unsafe-current-directory lookup protection; an explicit user-supplied path is a separate deliberate choice. [S-07](#s-07)

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
    QuotaID       string        // e.g. codex; not a cross-provider unit
    ResetAt       *time.Time
    ObservedAt    time.Time
    Source        string        // codex_app_server or grok_acp_billing
    Diagnostic    Diagnostic    // safe code + application-owned message
}
```

Transport structs remain in the provider package. Preserve field presence with pointers or raw JSON decoding. Go zero values must not turn omitted numbers into a measured zero. Decode the percentage as a number, not a numeric string. Ignore additive unknown object fields, but validate all fields needed to establish the metric.

Project percentage policy: accept finite values in `[0,100]`; format once to one decimal place in the renderer, normalizing signed zero. Do not round before selecting a period or comparing duplicate windows. Out-of-range values are a protocol/semantic mismatch, not a reason to clamp. A future upstream definition allowing percentages over 100 requires an explicit contract change.

### Terminal statuses

| Internal status/reason | WEEKLY USAGE cell | Exit class |
|---|---|---|
| `ok` | e.g. `37.0%` | Clean |
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

An unknown Codex authentication type maps to `UNSUPPORTED`, not the API-key status. For Grok, absence of `cached_token` with only an API-key method advertised establishes API-key-only non-applicability; interactive methods without cached authentication mean `AUTH REQUIRED`. Unrecognized auth advertisements mean `UNSUPPORTED`.

For account labels, Codex email is preferred only when obtained from that target's protocol session. Otherwise use its home basename. Grok uses its configured alias on every outcome. After collecting results, disambiguate duplicate Codex account labels by appending `[home-basename]`; append `#<target index + 1>` if that is still ambiguous — the suffix is the row's own configured target index, not a collision ordinal, so the same/other/same example labels its third target `#3`. If the index-qualified label is itself already occupied (for example by an unrelated row whose original label merely resembles a generated suffix), extend it with a deterministic second counter that preserves the index component (`#3-2`, `#3-3`, ...). Never hide or merge a row. Labels are identifiers for display, not authentication selectors.

<a id="arch-05"></a>
## ARCH-05: Execution and process ownership

Related: [REQ-004](../specs/README.md#req-004), [REQ-009](../specs/README.md#req-009), [REQ-020](../specs/README.md#req-020), [REQ-022](../specs/README.md#req-022), [REQ-023](../specs/README.md#req-023), [REQ-040](../specs/README.md#req-040). Owners: [TASK-003](../roadmap/README.md#task-003), [TASK-009](../roadmap/README.md#task-009).

```text
parse/validate -> build ordered targets -> admit up to 3 targets
    each target:
        preflight -> private temporary cwd -> start child + pipes
        -> initialize -> account/auth -> quota request -> normalize
        -> bounded cleanup -> terminal result
all terminal results -> account-label disambiguation -> table -> exit code
```

Use a fixed result slice and index-based assignment. A target's deadline starts when admitted, not when first queued. At most three are active. Default operation admits all three immediately. Additional Codex homes use the same bounded worker pool. A failed target does not cancel the invocation context.

The application issues no automatic RPC retries or process restarts. A provider can internally refresh credentials or retry its own network requests, but the outer deadline still applies. Do not wait for a quota notification as a substitute for the explicit read response.

### Child lifecycle

The process component owns start, stdin/stdout/stderr, process wait, and cleanup. Adapters borrow an RPC connection and always close it. Use `os/exec` with explicit arguments, not `sh -c`. [S-07](#s-07)

At termination, stop accepting new RPC work and close stdin. Allow up to 500 ms for EOF exit, then up to 500 ms after SIGTERM, then use SIGKILL if required and finish bounded reaping/pipe closure within the remaining cleanup budget. The total cleanup allowance is 2 seconds, separate from the query budget. Design cancellation and pipe ownership so a stuck pipe reader cannot extend this allowance indefinitely.

`Wait` has one owner and is called exactly once for every successfully started direct child. Drain stderr concurrently so backpressure cannot deadlock the child. Retain at most 64 KiB in memory and discard the rest; do not emit that raw buffer. Stop/join protocol goroutines and close descriptors on every path. A normal, planned termination after a valid query must not overwrite the collected result with a spurious failure. An exit before a required response is an error.

A private Unix process group is created for every child, but only for signal isolation: the child and anything it starts cannot deliver terminal-control signals to gaebokchi's own group. The group is never signalled during cleanup. Both production providers run under the direct-child-only policy: gaebokchi terminates, reaps, and closes its own pipe ends for the direct child it started, on both the graceful path (EOF, SIGTERM) and the forced path (SIGKILL to the direct child alone). No qualified basis exists for treating a group member as disposable — group membership or ancestry does not distinguish an application-owned descendant from a provider-managed shared leader, and a direct child's unresponsiveness does not either. Descendants with uncertain ownership therefore survive both graceful and forced cleanup; closing gaebokchi's own pipe ends is what unblocks its readers. Grok can use a provider-managed shared leader; never send leader shutdown/control RPCs, never use `pkill grok`/`pkill codex`, and never kill a process merely because it appeared during a query. TASK-001 records the installed CLI's ownership behavior.

Cleanup incompleteness is reported, never hidden: when bounded cleanup cannot fully complete (for example, the private temporary directory cannot be removed), the target's result carries `CleanupFailed`, a bounded application-owned diagnostic is emitted, and the invocation exits 1 — while a measured quota stays numeric and the signal exit codes (130/143) keep their precedence.

Create one mode-0700 temporary cwd per admitted target outside the invocation project. Preserve the real credential homes. Remove only gaebokchi-owned temporary paths, after process cleanup. This reduces accidental project-config discovery; it is not a claim that a trusted provider binary is sandboxed or incapable of reading user-level configuration.

SIGINT/SIGTERM cancel active contexts and mark unstarted targets cancelled. Render once with completed results retained when output remains writable, then return the signal-specific exit code. Do not forcibly clean up the caller's other agent sessions.

<a id="arch-06"></a>
## ARCH-06: Shared RPC transport, separate codecs

Related: [REQ-016](../specs/README.md#req-016) through [REQ-019](../specs/README.md#req-019), [REQ-043](../specs/README.md#req-043). Owner: [TASK-004](../roadmap/README.md#task-004).

A reusable connection provides newline framing, serialization, request-ID correlation, cancellation, bounded buffers, and terminal transport errors. It does **not** implement a universal Codex/ACP handshake. Prefer one outstanding client request per provider connection; different account processes can run concurrently.

| Property | Codex App Server | Grok ACP candidate |
|---|---|---|
| Framing | One JSON object per newline | One JSON object per newline |
| Outgoing `jsonrpc` | Omitted | `"2.0"` |
| Startup | `initialize` response, then `initialized` notification | `initialize` negotiation, then `authenticate` |
| Identity | `account/read` | Configured alias; no identity query |
| Quota method | `account/rateLimits/read` | Qualified billing extension |
| Agent work | Forbidden | Forbidden |

These external protocol properties come from [S-01](#s-01), [S-03](#s-03), and [S-04](#s-04). Exact installed-version envelopes are qualified in TASK-001.

Use a line reader with an explicit 4 MiB ceiling, including delimiter accounting. Do not rely on an unmodified `bufio.Scanner` default. Accept LF and CRLF framing, partial reads, multiple messages in one read, and valid UTF-8. Reject invalid JSON/UTF-8, top-level arrays/batches, and unexpected top-level types. Ignore empty transport lines only; non-JSON banners on stdout are protocol errors, not content to scrape. [S-08](#s-08)

Allocate positive integer request IDs per connection. A response must contain the matching ID and exactly one of `result` or `error`. Preserve explicit JSON null where a schema allows it, and let the adapter reject null when an object is required. Well-formed unknown/duplicate response IDs are ignored and cannot complete a call. A well-formed unrelated notification is ignored. EOF fails unresolved calls immediately.

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

Candidate protocol exchange from official documentation, to qualify against the installed binary [S-01](#s-01):

```json
{"id":1,"method":"initialize","params":{"clientInfo":{"name":"gaebokchi","title":"Gaebokchi","version":"0.1.0"}}}
```

Wait for response 1. Then:

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

Accept the qualified `account.type == "chatgpt"` shape and an optional email. `account:null` is `AUTH REQUIRED`. The known `apiKey` type is non-applicable, and unknown auth variants are `UNSUPPORTED`. A quota call's verified authentication error also produces `AUTH REQUIRED`; do not classify arbitrary English error text as proof of expired login. Preserve the home alias if identity was not read successfully.

### Quota selection algorithm

The following is the application's deterministic selection policy, not an undocumented provider promise:

1. Validate the outer rate-limit result object. When `rateLimitsByLimitId` is a **non-empty map**, select only its `codex` entry. If it is missing, return `N/A (no weekly)` and do not switch to another bucket or the legacy view. A present `codex` value with a wrong structural type is a protocol error; null yields `N/A (no data)`.
2. When the map is absent, null, or empty, consider the backward-compatible `rateLimits` object. Accept an explicit `limitId == "codex"`. An absent/null limitId is allowed only for a legacy shape captured during TASK-001; without that fixture it yields `N/A (no weekly)`. Explicit non-Codex identifiers are not eligible. No legacy fixture means no speculative legacy branch.
3. In the selected bucket, inspect both `primary` and `secondary`. Match an explicitly present numeric integer `windowDurationMins == 10080`. Null or absent windows are allowed. Never infer seven days from slot position or `resetsAt`.
4. If no eligible window exists, return `N/A (no weekly)`. If exactly one exists, require a valid `usedPercent`. Missing/null usage gives `N/A (no data)`; a wrong type/range gives `PROTOCOL ERROR`.
5. If both windows are weekly and their used percentage and optional reset values agree exactly, treat them as a duplicate representation of one window. If either candidate is malformed or they conflict, return `PROTOCOL ERROR`. Do not choose max/min/first.
6. Preserve valid reset metadata as a Unix-seconds timestamp when available. Missing reset metadata does not invalidate an otherwise valid percentage. A malformed optional reset can be discarded with a safe diagnostic; it must not be used to guess the period.

Inspecting only an unrelated malformed bucket must not poison a valid selected Codex bucket. Unknown additive fields, credit balances, and token-summary fields are ignored. All normalization functions are pure and table-tested independently of a child process.

<a id="arch-08"></a>
## ARCH-08: Grok adapter

Related: [REQ-031](../specs/README.md#req-031) through [REQ-036](../specs/README.md#req-036). Owners: [TASK-007](../roadmap/README.md#task-007), [TASK-008](../roadmap/README.md#task-008).

**Compatibility boundary:** the official source includes a billing extension, but this document does not claim the owner's installed stdio binary has already been tested. The public source revision inspected is recorded in [S-05](#s-05). Do not equate a source handler or in-process TUI call with exposed wire support.

Initial invocation to qualify: `grok --no-auto-update agent stdio`. The documented flag suppresses automatic-update checks in automated use. Do not silently remove it and accept update side effects if a binary rejects it; qualify another supported invocation only through the contract gate. [S-03](#s-03)

Candidate handshake:

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientInfo":{"name":"gaebokchi","version":"0.1.0"},"clientCapabilities":{"fs":{"readTextFile":false,"writeTextFile":false},"terminal":false}}}
```

Wait for the response, validate the supported protocol version, and inspect advertised auth methods. Select `cached_token` only. Then:

```json
{"jsonrpc":"2.0","id":2,"method":"authenticate","params":{"methodId":"cached_token","_meta":{"headless":true}}}
```

No interactive method is a fallback. API-key environment variables must not silently select a different billing account. Preserve the provider's normal credential context but never pass a token in RPC parameters. The cached-token/headless path is documented in [S-03](#s-03).

### Internal method name versus wire name

The handler is named `x.ai/billing` internally. ACP reserves underscore-prefixed wire methods for extensions; the inspected Grok source also documents this prefix distinction. [S-04](#s-04), [S-05](#s-05), [S-06](#s-06)

The **initial wire candidate**, not a completed compatibility result, is:

```json
{"jsonrpc":"2.0","id":3,"method":"_x.ai/billing","params":{}}
```

TASK-001 must capture the exact successful envelope and parameter shape from the installed version. An SDK's `ExtRequest("x.ai/billing", ...)` may serialize differently from handwritten JSON. Inspect the matching serializer/router and validate end to end. Do not label the endpoint unsupported merely because a bare internal method name was sent incorrectly, and do not add a runtime shotgun sequence of guessed names/wrappers. Freeze one observed contract in the adapter and fixture; version-specific branches require separate evidence.

Require billing retrieval without `session/new` or a prompt. If it needs an agent session, a browser, token extraction, a private REST call, or a mutation, stop at the gate and record the incompatibility. Do not silently broaden scope. A well-formed method-not-found error from the correctly encoded request is `UNSUPPORTED`.

### Result mapping

The inspected billing types contain `config.creditUsagePercent` and `config.currentPeriod` with a period `type`, optional `start`, and optional `end`. The source describes `USAGE_PERIOD_TYPE_WEEKLY` and weekly/monthly periods. Optional zero-valued fields must retain their actual presence semantics. [S-05](#s-05)

Normalize in this order:

| Observation | Result |
|---|---|
| Outer response is not the qualified object shape | `PROTOCOL ERROR` |
| `config` absent/null | `N/A (no data)` |
| `config` wrong type | `PROTOCOL ERROR` |
| `currentPeriod.type == "USAGE_PERIOD_TYPE_MONTHLY"` | `N/A (monthly)`; never show its percentage as weekly |
| Period/type absent or unknown well-formed enum | `N/A (period)` |
| Required period/type has an invalid structural type | `PROTOCOL ERROR` |
| Explicit weekly enum, percentage absent/null | `N/A (no data)` |
| Explicit weekly enum, valid percentage | Numeric weekly result |
| Explicit weekly enum, invalid percentage | `PROTOCOL ERROR` |
| Legacy monthly credit fields only | `N/A (period)` |

The monthly enum spelling, wrapper, numeric encoding, and auth/error shapes must be confirmed with source/fixtures in TASK-001 rather than guessed from a web UI. A numeric enum is not accepted merely because its value appears plausible; it needs a version-qualified mapping.

Parse an optional period end as RFC 3339 when present. A malformed optional end can be discarded with a diagnostic without replacing the verified weekly percentage. Never infer a period from date subtraction. Ignore prepaid balance, on-demand usage/caps, history, and subscription tier for percentage calculation.

The ACCOUNT column uses `--grok-account`, default `default`. The known billing payload is not treated as an email source. This is a conscious minimal design, not a claim that the actual Grok account's email is unknown to Grok itself.

<a id="arch-09"></a>
## ARCH-09: Security and read-only boundary

Related: [REQ-019](../specs/README.md#req-019), [REQ-021](../specs/README.md#req-021), [REQ-037](../specs/README.md#req-037) through [REQ-041](../specs/README.md#req-041). Owners: [TASK-003](../roadmap/README.md#task-003), [TASK-004](../roadmap/README.md#task-004), [TASK-005](../roadmap/README.md#task-005), [TASK-011](../roadmap/README.md#task-011).

The trust boundary is the user-selected, already installed provider executable. The application is not a sandbox for a malicious binary. It avoids adding credential handling or agent capabilities of its own.

| Boundary | Required behavior |
|---|---|
| Credentials | No auth-file parsing/copying, JWT decoding, token input, or application keychain access |
| Account changes | No login/logout, account switching, device flow, browser launch, or automatic repair |
| Agent capabilities | No thread/session creation, prompts, host tools, approvals, or workspace data |
| Billing | Read allowance only; no purchases, top-up changes, reset credits, or notifications to account owners |
| Configuration | No writes to Codex/Grok config or auth; no changes to existing login |
| Networking | Only provider processes contact their backends; gaebokchi has no HTTP billing client or listener |
| Logs | Application-owned error codes/messages only; raw protocol/stderr remains undisclosed |
| Persistence | No application telemetry or permanent runtime files |
| Shared processes | Do not shut down/kill provider leaders or unrelated coding sessions |

Construct a new child environment for every target. Start from inherited environment only to retain necessary PATH, HOME, XDG/keychain, certificate, and proxy behavior; then filter known conflicting authentication/parent-session variables. Baseline Codex filtering includes `OPENAI_API_KEY`, `CODEX_API_KEY`, and replacement of every `CODEX_HOME` entry. Baseline Grok filtering includes `XAI_API_KEY` and `GROK_API_KEY`. Additional variables affecting installed-version credential selection, agent nesting, or wrapper behavior are identified and listed in TASK-001. Do not dump environment values as evidence or clear all variables indiscriminately.

Shell aliases are not executables. Binary flags accept one program path/name, not arbitrary arguments. Do not bypass Go's `ErrDot` protection or run a binary accidentally found through an implicit current-directory PATH entry. A deliberately specified path containing spaces is passed as one executable value, not split into a command.

“Read-only” here describes application requests and business effects. Provider startup can refresh existing tokens, update provider logs/caches, or use a provider-managed leader. The application must neither promise a byte-for-byte unchanged provider home nor take over those provider internals. Document any material observed startup effect, and block an integration that requires model work or an account/billing mutation.

Treat error data as untrusted. Convert only verified structured codes to named statuses; otherwise use `ERROR` with a fixed safe message and target ID. Never print `error.Error()` blindly when it can include provider payloads, environment values, auth redirects, or local secrets. Tests inject synthetic secret markers to verify they never reach stdout/stderr or checked-in evidence.

<a id="arch-10"></a>
## ARCH-10: Table, diagnostics and exit behavior

Related: [REQ-010](../specs/README.md#req-010) through [REQ-012](../specs/README.md#req-012), [REQ-024](../specs/README.md#req-024). Owner: [TASK-010](../roadmap/README.md#task-010).

Render only after targets have terminal results, in their original order. Use `text/tabwriter` or an equivalent small standard-library implementation to produce space-aligned columns. Padding is cosmetic; headings, cells, order, logical line count, and final newline are contractual. Narrow terminals may visually wrap a long line; the program must not insert additional line breaks to accommodate terminal width. [S-09](#s-09)

Sanitize all externally supplied labels before rendering: replace control/format characters and non-space whitespace with safe spaces, trim, and collapse whitespace. Never emit ESC or bidi formatting controls. A blank sanitized alias is invalid input; an unusable protocol email falls back to the home alias. Preserve ordinary printable Unicode characters. Do not truncate or mask emails by default; the README must warn that stdout can contain account identifiers.

Illustrative partial result:

```text
AI PROVIDER  ACCOUNT                WEEKLY USAGE
ChatGPT      account1@example.com   37.0%
ChatGPT      .codex-hsy              AUTH REQUIRED
Grok         default                N/A (monthly)
```

A corresponding safe stderr line can be:

```text
gaebokchi: codex:1: AUTH REQUIRED: authenticate using the provider CLI outside gaebokchi
```

Emit diagnostics in target order, at most one bounded line per target plus a command-level failure if needed. Include no timestamps, progress spinners, raw paths to secrets, or raw upstream errors. A successful numeric row needs no diagnostic. Non-applicable/unavailable outcomes may have a fixed explanatory diagnostic.

| Exit code | Meaning |
|---|---|
| `0` | Valid invocation; all rows are numeric or explicitly non-applicable (monthly/API-key), or help/version was requested |
| `1` | One or more incomplete/error rows, internal command execution failure, or stdout write/flush failure |
| `2` | Invalid arguments or unresolvable command-level configuration; launch no providers and print no table |
| `130` | Invocation cancelled by SIGINT |
| `143` | Invocation cancelled by SIGTERM |

On signal cancellation, the first handled signal determines the cancellation exit code; it takes precedence over partial provider failure. If stdout is unwritable, a complete table cannot be guaranteed. Without a cancellation signal, return 1 for output failure and write only a safe diagnostic where possible. Test this policy, including broken-pipe behavior, rather than assuming platform default signal handling matches it.

A normal missing executable/home is target-specific, not a usage error. Example: no installed Grok still produces both Codex rows and a Grok `NOT INSTALLED` row, with exit 1.

<a id="arch-11"></a>
## ARCH-11: Verification catalog

Related: [REQ-043](../specs/README.md#req-043) through [REQ-048](../specs/README.md#req-048). Tests are created with the owning implementation task; [TASK-011](../roadmap/README.md#task-011) adds cross-component qualification rather than postponing unit tests.

The following identifiers are referenced by individual requirements and tasks. Implementation records exact test names and commands under these IDs.

<a id="v-contract"></a>
### V-CONTRACT: External contract probe

Record versions, executable identities, successful prompt-free exchanges, precise Grok extension encoding, relevant environment filtering, optional fields, and process ownership. Sanitize before saving. The presence of a source method is not a passed live probe. An unsupported result is a gate failure for positive provider support.

<a id="v-options"></a>
### V-OPTIONS: Target and flag policy

Cover default homes despite ambient CODEX_HOME, replacement semantics, quoted/space-containing paths, relative paths, tilde, invalid `~user`, empty paths/aliases, duplicate homes through symlinks, missing/inaccessible homes, scalar duplicates, invalid duration/ranges, unknown flags, positional arguments, help/version without installed CLIs, and explicit binary paths.

<a id="v-rpc"></a>
### V-RPC: Framing and protocol dialects

Use a Go fake child executable or test-binary helper subprocess, not a live CLI. Script fragmented/coalesced messages, CRLF, notifications before replies, unknown/duplicate IDs, errors, EOF, null results, malformed/invalid UTF-8 JSON, banners, oversized messages, stderr flooding, and hanging readers/writers. Assert Codex lacks `jsonrpc`, Grok includes it, and only the relevant handshake occurs.

<a id="v-lifecycle"></a>
### V-LIFECYCLE: Finite process ownership

Test early exit, timeout during startup/handshake/read, closed stdin, SIGINT/SIGTERM, child ignoring EOF/SIGTERM, inherited pipe holders, reaping, descriptor/goroutine cleanup, and untouched unrelated/shared processes. Use short injected deadlines and synchronization, not long sleeps. Validate the configured budget and cleanup bound with reasonable scheduler tolerance recorded in tests.

<a id="v-quota"></a>
### V-QUOTA: Pure normalization

Cover explicit 0/100/fractional values, missing/null/wrong-type values, signed zero, negative/over-100 values, no ratio conversion, unknown additive fields, missing/malformed optional reset metadata, and unavailable versus non-applicable semantics.

<a id="v-codex"></a>
### V-CODEX: Codex contract

Cover two simultaneous homes with different identity/quota values; authenticated, null, API-key and unknown account types; authentication errors; primary-only weekly, secondary-only weekly, swapped slots, five-hour-only, null secondary, no duration, non-empty map with/without codex, empty map legacy behavior, explicit unrelated legacy bucket, duplicate identical weekly windows, conflicting weekly windows, invalid selected bucket, and absent versus zero usage. Assert no thread/turn methods occur.

<a id="v-grok"></a>
### V-GROK: Grok contract

Cover cached-token negotiation, API-key-only advertisement, interactive-only login, incompatible protocol versions, exact extension encoding, method-not-found, successful weekly/monthly config, absent/unknown period, legacy-only monthly fields, valid zero, invalid values, null config, unexpected callbacks, and provider errors. Assert no session creation, prompt, login fallback, or REST call.

<a id="v-identity"></a>
### V-IDENTITY: Display identity

Cover email from the correct Codex process, fallback labels on failure, duplicate emails and basenames, different homes not merged, Grok alias on all outcomes, Unicode, malicious control characters, blank email fallback, and labels that would otherwise inject lines or columns.

<a id="v-runner"></a>
### V-RUNNER: Fan-out and isolation

Use controlled fake completion order. Verify at most three active targets, default simultaneous admission, queued target cancellation, deterministic output order, independent environments, one query attempt per target, and no cancellation of healthy siblings after one failure.

<a id="v-output"></a>
### V-OUTPUT: Table contract

Golden-test numeric and mixed-status outputs. Assert three headings, no separator/banner/color, exactly four logical lines by default, N+2 with N custom Codex homes, and a final newline. Check redirected stdout, Unicode/control sanitization, and diagnostics exclusively on stderr.

<a id="v-exit"></a>
### V-EXIT: Exit contract

Test all-clean, all-non-applicable, partial failure, all failure, invalid usage, help/version, broken stdout, SIGINT, and SIGTERM. Assert code and output together rather than testing only internal status enums.

<a id="v-safety"></a>
### V-SAFETY: Read-only and secret boundary

Assert complete outbound method allowlists, false host capabilities, no raw secret leakage, no shell invocation, no credential/config reads or writes by gaebokchi, no account-home creation, isolated environments, neutral cwd, and untouched unrelated processes. Static package/dependency inspection complements black-box tests; tests must not claim to sandbox trusted provider internals.

<a id="v-e2e"></a>
### V-E2E: Executable-level offline acceptance

Build gaebokchi and fake codex/grok executables in temporary paths. Run the actual command with injected binary paths and temporary homes. Verify the default happy path, one failing home, missing CLI, monthly Grok, malformed Grok response, slow/hung provider, unsafe callbacks, signal cancellation, and concurrent account isolation. Normal CI must not use an actual logged-in home.

<a id="v-build"></a>
### V-BUILD: Reproducible local builds

Run formatting checks, `go vet ./...`, `go test ./...`, native race tests where the selected toolchain supports them, and CGO-disabled cross-builds for all four release targets. Record actual command exit statuses. Ensure the binary has version metadata and no bundled provider binaries.

<a id="v-platform"></a>
### V-PLATFORM: Native operating-system qualification

Execute the fake-provider integration suite natively on macOS and Linux. Record architecture and compiler for each. Other cross-built architectures remain build-qualified until native execution evidence exists. Do not silently label them live-provider tested.

<a id="v-live"></a>
### V-LIVE: Explicit owner-account campaign

On an authorized account host, run both Codex homes and the Grok login through the implemented executable. Obtain valid quota/billing responses, inspect the provider's own usage display without submitting a model prompt, and compare period/percentage semantics. Record reset boundaries, changing usage, and any precision differences rather than forcing exact values from non-simultaneous observations. A monthly Grok account is qualified as monthly only; source-backed weekly fixtures do not become live weekly evidence. Never use unsupported/timeout as a successful integration result.

<a id="v-docs"></a>
### V-DOCS: User-facing documentation

Check README commands against the binary, examples against fixtures, status/exit descriptions against tables, supported/tested versions against evidence, and privacy/side-effect statements against the verified process behavior.

<a id="v-audit"></a>
### V-AUDIT: Cross-document closure

Check every REQ/ARCH/EPIC/TASK reference and local Markdown fragment, every requirement's owner/evidence, all task completion gates, all deferred items, and the final release matrix. A skipped check remains skipped, not passed. This is a bounded audit, not a new design cycle.

### Suggested deterministic verification commands

Once the module exists:

```sh
go vet ./...
go test ./...
go test -race ./...                 # native supported runners; record exclusions
go run scripts/check-docs.go
```

The implementation may use focused `go test ./internal/provider/codex` or similar during a task. Keep the project-standard full offline suite quick; this scope does not justify hour-long external-agent validation or provider polling.

<a id="arch-12"></a>
## ARCH-12: Compatibility gate and evidence policy

Related: [REQ-005](../specs/README.md#req-005), [REQ-042](../specs/README.md#req-042), [REQ-048](../specs/README.md#req-048). Owners: [TASK-001](../roadmap/README.md#task-001), [TASK-013](../roadmap/README.md#task-013).

Initial compatibility status:

| Integration | Public evidence | Installed CLI qualification |
|---|---|---|
| Codex account/rate-limit read | Official App Server docs | QUALIFIED live (TASK-001, reconfirmed TASK-013) |
| Two isolated Codex homes | Official CODEX_HOME configuration model | QUALIFIED live (distinct emails/quotas) |
| Grok cached-token ACP auth | Official headless/ACP docs | QUALIFIED live |
| Grok billing handler and payload | Official source snapshot | QUALIFIED live (weekly enum + live percentage) |
| Grok extension wire form through installed stdio | ACP rules plus source candidate | QUALIFIED live as `_x.ai/billing` with `params:{}` |
| macOS/Linux lifecycle behavior | Proposed architecture | QUALIFIED darwin/arm64 + linux/arm64 (containerized native suites) |

TASK-001 must run small, bounded, developer-only metadata probes before production transport/provider work. A disposable standard-library script is acceptable and is not a shipped command. Do not install/upgrade CLIs, relink accounts, launch a model, or edit auth configuration as part of a probe.

Record in `docs/verification/compatibility.md`:

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
| Both CLIs expose required prompt-free metadata | Freeze observed contracts; complete TASK-001 |
| Grok returns valid monthly billing | Qualify monthly behavior; retain weekly fixture qualification and label live limitation |
| Method name/wrapper was encoded incorrectly | Correct the probe from matching source, then perform one bounded confirmation |
| Correctly encoded Grok method is unsupported | Block positive integration; record evidence and needed compatible CLI capability |
| Missing executable/login or host access | Mark gate BLOCKED/NOT RUN; do not pretend fixtures are a live substitute |
| Only token extraction, REST fallback, session creation, or account mutation would work | Stop; retain scope boundary and report the specific gap |

A human-approved scope change is required to release Codex-only or add an alternate Grok access route. That is not delegated to the implementation AI. A verified source-backed clarification of an envelope within the same metadata-only interface is a normal TASK-001 contract update, not a product redesign.

The early probe validates feasibility; TASK-013 validates the delivered application. Both matter. Store tool/source versions separately, and never treat a repository commit SHA as the installed release version. Runtime behavior is capability-driven; the README states tested versions, not unproven universal compatibility.

<a id="arch-13"></a>
## ARCH-13: Build, platform qualification and delivery

Related: [REQ-001](../specs/README.md#req-001), [REQ-046](../specs/README.md#req-046), [REQ-047](../specs/README.md#req-047), [REQ-049](../specs/README.md#req-049), [REQ-051](../specs/README.md#req-051). Owners: [TASK-012](../roadmap/README.md#task-012), [TASK-028](../roadmap/README.md#task-028).

Use a maintained Go version selected and recorded during implementation. Pin the tested toolchain in CI and document it; do not assert the latest release number from memory. Prefer standard-library dependencies and `CGO_ENABLED=0` release builds. Go's GOOS/GOARCH target model supplies the build axes; native testing remains separate. [S-10](#s-10)

| Artifact target | Required build evidence | Required runtime evidence |
|---|---|---|
| darwin/arm64 | Cross/native build succeeds | Native macOS evidence on available runner architecture |
| darwin/amd64 | Cross/native build succeeds | Explicitly label native-tested or build-only |
| linux/arm64 | Cross/native build succeeds | Explicitly label native-tested or build-only |
| linux/amd64 | Cross/native build succeeds | Native Linux evidence on available runner architecture |

Native macOS and native Linux tests are mandatory, but native execution on all four CPU/OS combinations is not. Keep a per-target matrix so users can see the difference. Live provider acceptance must cover the owner account set on at least one supported host; do not imply live support was tested on the other host or every version.

Resolve executable version metadata in this order: explicit release linker
override, Go main-module build information, then `(devel)`. Release linker
metadata and versioned staging paths use the canonical `v` prefix; the v0.1.0
staging directory is `dist/gaebokchi-v0.1.0`. Public version JSON does not
expose commit, revision, module checksum, or build date.

Produce versioned local archives with the `gaebokchi` prefix and SHA-256 checksums, containing the `gaebokchi` executable, README, and the three specification documents. Include a license only after the actual project license is chosen; do not invent a company licensing decision. No publishing, package-manager tap, signing credential use, external release upload, or provider binary redistribution is authorized by this plan.

README startup instructions assume the user installs and logs into the provider CLIs separately. Explain PATH/binary flags, the two home directories, how to select a different home set, default Grok alias, subscription-only interpretation, unsupported protocol errors, and potential provider-owned startup writes.

<a id="arch-14"></a>
## ARCH-14: Change and completion discipline

Related: [REQ-050](../specs/README.md#req-050). Owner: [TASK-014](../roadmap/README.md#task-014).

The three canonical role documents are the specification authority; implementation evidence lives under `docs/verification/`. Do not duplicate competing roadmaps or generate extra dossier/approval frameworks for this small project.

For a contract change, update the relevant requirement, architecture section, task scope/test mapping, and fixtures together. Preserve stable IDs. Do not mark later work complete because a helper exists, and do not reopen a completed epic for unrelated improvements.

The implementation AI follows one epic to closure before starting the next. Each task is one coherent commit, including its focused tests and associated document/evidence updates. An Aquarium epic-handler may orchestrate this plan, but Aquarium, Podway, or any external review service is not a product dependency.

Use one planned review where appropriate, fix bounded blocking findings, and allow at most one confirmation review of the changed scope. Remaining blockers are reported with evidence, not fed into a new automatic full review loop. Required deterministic tests are not waived by an absent external reviewer.

<a id="arch-15"></a>
## ARCH-15: Primary-source references

These sources were consulted for this specification on 2026-09-22. They document external interfaces, not the completion state of this project. The implementation must record observed binary versions separately. Source links can move; retain the fixed source revision used for fixtures.

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
