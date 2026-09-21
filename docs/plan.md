# Gaebokchi (개복치): Sequential Implementation Plan

Version: 0.1 specification  
Status: NOT STARTED  
Prepared: 2026-09-22  
Project: Gaebokchi (개복치)  
Ecosystem: Aquarium tool  
Repository and executable: `gaebokchi`

## How to use this plan

This document owns **execution order and completion state** for **Gaebokchi (개복치)**, an Aquarium tool. Implement against [architecture.md](architecture.md), and verify product coverage against the checkboxes in [requirement.md](requirement.md). This package contains a specification, not an implementation report: all work is initially pending.

Keep all three documents at the repository root. Start with the [execution rules](#plan-execution), then the first incomplete task in the current epic. Read the linked architecture sections and requirements for that task rather than repeatedly loading all documentation for a narrow change.

<a id="plan-execution"></a>
## Execution rules

1. Run one epic at a time, in the exact roadmap order. Finish all its tasks and its local acceptance gate before starting the next epic. There is no interleaving, partial-epic jump, or hidden dependency on a later epic.
2. Run one task at a time. Each task is one coherent implementation/evidence commit, with its own focused tests. Do not auto-commit unrelated user changes or rewrite existing history. The first task may include the initial specification import in a new repository.
3. An epic is a component milestone. Its local gate must be complete, but the full product is not declared complete until EPIC-006. Cross-cutting requirement checkboxes can remain pending until the integration/release evidence exists, without reopening a correctly completed component epic.
4. Mark a task `IN PROGRESS` when work begins. Complete it only after its acceptance criteria pass, evidence is recorded and the task-scoped commit exists. Otherwise mark it `BLOCKED` with the exact unsatisfied prerequisite and safe next action. An implementation AI must not silently weaken a gate.
5. Update architecture, affected requirements, plan and fixtures together for actual contract changes. Keep REQ/ARCH/EPIC/TASK IDs stable. File names in tasks are suggested locations, not permission to ignore a better existing equivalent in the actual repository.
6. Tests accompany implementation rather than being deferred to a final testing task. Use focused tests during development and the relevant complete offline suite at epic closeout. Real account queries are explicit, bounded qualification work only.
7. Review is bounded: one planned assessment where appropriate, one focused remediation pass, and at most one confirmation review of the affected scope. If a blocking issue remains, stop and report it. Do not automatically launch another full review/rewrite cycle. A missing external reviewer never permits skipping deterministic acceptance tests.
8. With Aquarium, the epic-handler can own the current epic and its task/commit lifecycle. Do not add Aquarium/Podway/Mulgae as runtime or build dependencies, or require a separate external workflow engine merely to use these documents.

The compatibility probe in TASK-001 is deliberately first. Codex and Grok are both in the agreed final scope; a correctly encoded but unsupported Grok endpoint blocks positive support, not a reason to claim the project complete with an `UNSUPPORTED` row.

<a id="plan-roadmap"></a>
## Roadmap in execution order

```text
EPIC-001 -> EPIC-002 -> EPIC-003 -> EPIC-004 -> EPIC-005 -> EPIC-006
```

| Order | Epic | Tasks, executed left to right | Entry dependency | Initial status |
|---|---|---|---|---|
| 1 | [EPIC-001: Contract qualification and foundation](#epic-001) | [TASK-001](#task-001) -> [TASK-002](#task-002) | None | NOT STARTED |
| 2 | [EPIC-002: Safe subprocess and RPC foundation](#epic-002) | [TASK-003](#task-003) -> [TASK-004](#task-004) | [EPIC-001](#epic-001) complete | NOT STARTED |
| 3 | [EPIC-003: Codex multi-account provider](#epic-003) | [TASK-005](#task-005) -> [TASK-006](#task-006) | [EPIC-002](#epic-002) complete | NOT STARTED |
| 4 | [EPIC-004: Grok single-account provider](#epic-004) | [TASK-007](#task-007) -> [TASK-008](#task-008) | [EPIC-003](#epic-003) complete | NOT STARTED |
| 5 | [EPIC-005: Usable snapshot CLI and offline acceptance](#epic-005) | [TASK-009](#task-009) -> [TASK-010](#task-010) -> [TASK-011](#task-011) | [EPIC-004](#epic-004) complete | NOT STARTED |
| 6 | [EPIC-006: Platform delivery and final qualification](#epic-006) | [TASK-012](#task-012) -> [TASK-013](#task-013) -> [TASK-014](#task-014) | [EPIC-005](#epic-005) complete | NOT STARTED |

The task order is globally sequential: TASK-001 through TASK-014. Runtime account queries may be concurrent; development tasks may not.

<a id="plan-evidence"></a>
## Evidence and commit contract

Each task includes its evidence in `docs/verification/TASK-NNN.md` or an equivalently clear section in an existing verification record. Keep evidence short and specific. A task must not be marked complete merely because its code compiles.

```text
Task:
Status: NOT STARTED | IN PROGRESS | BLOCKED | COMPLETE
Goal and implementation scope:
Requirements and architecture sections covered:
Changed files:
Verification:
  - verification ID, exact command/test, result, environment
  - explicitly identify NOT RUN / SKIPPED / BLOCKED checks
Fixtures: captured or synthetic; source/version; sanitization policy
Findings and bounded remediation, if any:
Compatibility/platform limitations:
Task commit subject or already-known commit reference:
Handoff to next task/epic:
```

Avoid a self-referential commit-hash loop. The evidence inside a task commit can identify that commit by its unique `TASK-NNN` commit subject. After committing, report its hash in the handoff or a later already-needed record. Do not create another commit solely to insert the current commit's own hash.

The normal task commit includes code, focused tests and documentation/evidence updates. Record test outcomes against the tested tree; do not claim a changed tree passed an earlier test. On a dirty repository, exclude unrelated user changes and report them rather than cleaning or committing them.

Check a requirement only when its complete acceptance scope is verified. The “primary owner” in requirement.md provides accountability; supporting component evidence may come from other tasks. A cross-cutting requirement is allowed to remain pending through component milestones and be closed by its integrating/final audit task once all cited evidence exists.

## Epics and tasks

<a id="epic-001"></a>
## EPIC-001: Contract qualification and foundation

- [x] Epic complete. **Status: COMPLETE.**

**Goal:** Prove the approved provider paths and establish the domain/CLI target contract before investing in production integration.

**Entry:** No earlier epic.

**Task sequence:** [TASK-001](#task-001) -> [TASK-002](#task-002)

**Local epic acceptance:** The actual Codex/Grok metadata interfaces are qualified and documented; the Go foundation and target-policy tests pass. This is a component milestone, not a claim that the final executable already queries providers.

**Handoff:** EPIC-002 receives frozen protocol fixtures, process/environment observations, domain types and ordered target construction.

<a id="task-001"></a>
### TASK-001: Qualify metadata-only provider contracts

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Remove the largest feasibility risk before building the production adapters: both installed CLIs must expose usable metadata without model work or application-held credentials.

**Dependency:** None. The owner-approved scope is Codex plus Grok, with the two stated Codex homes.

**Architecture:** [ARCH-06](architecture.md#arch-06), [ARCH-07](architecture.md#arch-07), [ARCH-08](architecture.md#arch-08), [ARCH-09](architecture.md#arch-09), [ARCH-12](architecture.md#arch-12), [ARCH-15](architecture.md#arch-15).

**Primary requirement ownership:** [REQ-042](requirement.md#req-042).

**Implementation scope:**

- Inspect `codex --version`, `grok --version`, relevant help, and the matching public interface/source. Record actual versions separately from source revisions.
- Use bounded, disposable developer-only probes in neutral temporary directories. Probe both Codex homes through initialize/initialized/account/read/rateLimits/read. Probe Grok initialize/cached-token authenticate/billing without creating a session.
- Resolve the Grok internal extension name versus actual underscore-prefixed wire envelope, including any serializer wrapper. Confirm the response wrapper, period enum, percentage encoding, error shape, and no-auto-update invocation.
- Observe required environment filtering, absent-login behavior, direct-child exit, and any provider-managed leader boundary. Do not kill existing leaders or run login/install/update commands.
- Save field-allowlisted, anonymized captured fixtures and a compatibility record; tag synthetic edge-case fixtures explicitly. Update architecture contract details only where actual evidence requires correction.

**Deliverables:** `docs/verification/compatibility.md`, provider `testdata` fixtures with provenance, and any evidence-based corrections to the three root documents. No shipped probe command.

**Acceptance criteria:**

- [x] Both Codex homes return structured account/quota data under their own CODEX_HOME, or a blocker is recorded rather than claiming success.
- [x] Grok returns an actual valid billing response through the qualified metadata-only stdio path. A valid monthly response is acceptable with an explicit weekly-live limitation; method-not-found is not.
- [x] Exact envelopes, CLI versions, semantic types, and process ownership are captured without secrets. No prompts, agent sessions, interactive auth, or billing mutations occurred.
- [x] Future provider tasks can implement one frozen observed contract instead of exploring guessed methods at runtime.

**Verification:** [V-CONTRACT](architecture.md#v-contract), [V-SAFETY](architecture.md#v-safety). Record the exact bounded probe invocations and results. Parse saved JSON fixtures with a local JSON parser and inspect the outgoing allowlist. There is no Go-module test requirement before TASK-002.

**Out of scope / stop conditions:** Do not implement REST/token/browser/PTY fallbacks, broaden the account model, or treat source inspection alone as a live pass. A correctly encoded unsupported Grok endpoint blocks the epic; report the specific prerequisite instead of implementing around it.

**Commit boundary:** `docs(TASK-001): qualify Codex and Grok metadata contracts`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-002](#task-002).

<a id="task-002"></a>
### TASK-002: Bootstrap the Go module, domain and target policy

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Create the smallest compilable foundation that expresses the approved CLI targets and status semantics without depending on a live provider.

**Dependency:** TASK-001 complete, including a successful Grok metadata-only contract gate.

**Architecture:** [ARCH-01](architecture.md#arch-01), [ARCH-02](architecture.md#arch-02), [ARCH-03](architecture.md#arch-03), [ARCH-04](architecture.md#arch-04), [ARCH-14](architecture.md#arch-14).

**Primary requirement ownership:** [REQ-007](requirement.md#req-007), [REQ-008](requirement.md#req-008).

**Implementation scope:**

- Initialize the module from the actual repository identity, or use a documented local-only module path. Use `gaebokchi` as the repository basename and `cmd/gaebokchi/main.go` as the command entry point, following the [ARCH-01 naming contract](architecture.md#arch-01). Record a maintained tested Go toolchain rather than assuming a version.
- Create the domain target/result/status types, constructors or validation helpers, and the one-target Query boundary. Use presence-aware percentage representation.
- Implement the pure options parser and ordered target construction: default homes, repeatable replacement semantics, binary selection, Grok alias, timeout validation, and path normalization.
- Validate empty and duplicate homes, including existing symlink aliases. Preserve missing valid paths as target outcomes rather than creating directories.
- Add focused option/domain tests and a minimal composition-root scaffold. Any unfinished runtime path must fail explicitly; do not emit invented successful usage.

**Deliverables:** `go.mod`, domain/options packages and tests, minimal command scaffold, and a recorded toolchain/module decision.

**Acceptance criteria:**

- [x] The repository builds and focused tests pass with no provider executables, network or credentials.
- [x] The default target list is exactly two explicit Codex homes followed by one Grok target, even with ambient CODEX_HOME set.
- [x] Target identities, statuses, missing-value semantics and parser errors match architecture.md.
- [x] No production provider access, transport implementation, cache, or runtime config format has been introduced.

**Verification:** [V-OPTIONS](architecture.md#v-options), [V-QUOTA](architecture.md#v-quota), [V-IDENTITY](architecture.md#v-identity). Run `go test ./internal/domain ./internal/options`, `go test ./...`, and `go vet ./...`. Record the selected compiler and module path.

**Out of scope / stop conditions:** Do not introduce a plugin framework, configuration-file discovery, a remote module identity not supplied by the repository, or placeholder zero percentages.

**Commit boundary:** `feat(TASK-002): add domain and CLI target foundation`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** Close [EPIC-001](#epic-001) using its local gate, then start [TASK-003](#task-003) in [EPIC-002](#epic-002).

<a id="epic-002"></a>
## EPIC-002: Safe subprocess and RPC foundation

- [x] Epic complete. **Status: COMPLETE.**

**Goal:** Create finite, owned subprocess execution and correctly separated Codex/ACP wire handling.

**Entry:** [EPIC-001](#epic-001) complete.

**Task sequence:** [TASK-003](#task-003) -> [TASK-004](#task-004)

**Local epic acceptance:** The transport/process packages pass the offline framing, cleanup and callback-denial suites. No provider-specific quota implementation is needed to close this epic.

**Handoff:** EPIC-003 receives a tested process/RPC boundary and fake-provider harness.

<a id="task-003"></a>
### TASK-003: Implement bounded owned-child lifecycle

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Make provider execution finite and safe before protocol code is allowed to use it.

**Dependency:** EPIC-001 complete. Consume the qualified process-ownership and invocation constraints.

**Architecture:** [ARCH-02](architecture.md#arch-02), [ARCH-05](architecture.md#arch-05), [ARCH-09](architecture.md#arch-09).

**Primary requirement ownership:** [REQ-020](requirement.md#req-020), [REQ-040](requirement.md#req-040).

**Implementation scope:**

- Add an injectable process factory using explicit executable and argument arrays. Each child gets separate pipe handles, environment input, and a private empty temporary cwd.
- Implement the single-owner start/wait lifecycle and the EOF, SIGTERM and SIGKILL cleanup sequence within the specified budget.
- Drain stderr concurrently with bounded retention and no raw forwarding. Ensure cancellation releases readers/writers and joins owned goroutines.
- Implement Unix ownership handling on macOS/Linux, with private process-group termination only where ownership was established. Shared provider leaders remain outside gaebokchi control.
- Add fake-process tests for successful exit, early exit, stuck child, ignored termination, inherited pipes, cleanup, temporary-directory deletion and unrelated-process survival.

**Deliverables:** Process lifecycle package, Unix-specific implementation and deterministic process tests.

**Acceptance criteria:**

- [x] Every successfully started direct child is waited on exactly once and no test hangs beyond its bounded budget.
- [x] The 64 KiB stderr cap does not stop draining or leak output; temporary resources are removed safely.
- [x] Cancellation/timeout cleanup cannot kill the caller’s existing Codex/Grok sessions or a provider-managed shared leader.
- [x] Process tests operate on test helpers only and need no real credentials or provider installation.

**Verification:** [V-LIFECYCLE](architecture.md#v-lifecycle), [V-SAFETY](architecture.md#v-safety). Run `go test ./internal/process` and the focused race/lifecycle tests on the current native host. Cross-platform qualification is owned by TASK-012.

**Out of scope / stop conditions:** Do not use `pkill`, shell wrappers, a shared application daemon, parent-environment mutation, or raw provider stderr in diagnostics.

**Commit boundary:** `feat(TASK-003): add bounded provider process lifecycle`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-004](#task-004).

<a id="task-004"></a>
### TASK-004: Implement framed RPC with provider-specific codecs

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Provide a reusable, well-tested message transport without flattening the differences between Codex App Server and ACP.

**Dependency:** TASK-003 complete.

**Architecture:** [ARCH-06](architecture.md#arch-06), [ARCH-09](architecture.md#arch-09), [ARCH-11](architecture.md#arch-11).

**Primary requirement ownership:** [REQ-016](requirement.md#req-016), [REQ-017](requirement.md#req-017), [REQ-018](requirement.md#req-018), [REQ-019](requirement.md#req-019), [REQ-043](requirement.md#req-043).

**Implementation scope:**

- Implement bounded newline-delimited UTF-8 JSON framing and request/response correlation over the owned process pipes.
- Add separate Codex and ACP envelope codecs, preserving the qualified Grok extension envelope from TASK-001. Use one outstanding request per connection.
- Handle interleaved notifications, wrong/duplicate IDs, EOF, remote errors, malformed frames, blocked writes and context cancellation.
- Reject server-initiated requests without executing capabilities; apply dialect-correct rejection and abort the affected query.
- Build the reusable Go fake-child/scripted-transcript harness and assert actual outgoing method allowlists. Keep all default tests offline.

**Deliverables:** RPC framing/client/codecs, transcript fixtures and fake-process test utilities.

**Acceptance criteria:**

- [x] Codex omits the jsonrpc marker, Grok includes it, and one codec does not impose the other provider’s handshake.
- [x] Fragmentation, coalescing, CRLF, notifications, malformed UTF-8/JSON, 4 MiB ceiling and EOF behavior match the contract.
- [x] Unknown response IDs never satisfy a request and host capability requests cause no execution or data exposure.
- [x] The transport and process packages pass together without a live provider or an unbounded wait.

**Verification:** [V-RPC](architecture.md#v-rpc), [V-SAFETY](architecture.md#v-safety), [V-LIFECYCLE](architecture.md#v-lifecycle). Run `go test ./internal/rpc ./internal/process`, relevant race tests, and `go test ./...`.

**Out of scope / stop conditions:** Do not add an arbitrary-RPC CLI, model/tool functionality, retry/backoff, websocket transport, or a generic agent-session framework.

**Commit boundary:** `feat(TASK-004): add bounded stdio RPC codecs`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** Close [EPIC-002](#epic-002) using its local gate, then start [TASK-005](#task-005) in [EPIC-003](#epic-003).

<a id="epic-003"></a>
## EPIC-003: Codex multi-account provider

- [x] Epic complete. **Status: COMPLETE.**

**Goal:** Complete Codex metadata queries with isolated account attribution and duration-based weekly quota selection.

**Entry:** [EPIC-002](#epic-002) complete.

**Task sequence:** [TASK-005](#task-005) -> [TASK-006](#task-006)

**Local epic acceptance:** The complete Codex Query implementation and two-home offline contract tests pass, including all unavailable/auth/malformed cases. No future Grok work is required to close it.

**Handoff:** EPIC-004 receives an established adapter pattern and completed Codex behavior; do not redesign Codex while adding Grok.

<a id="task-005"></a>
### TASK-005: Add isolated Codex authentication and account reads

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Bind a Codex connection to exactly one home and establish its subscription identity without generating an agent turn.

**Dependency:** EPIC-002 complete.

**Architecture:** [ARCH-03](architecture.md#arch-03), [ARCH-05](architecture.md#arch-05), [ARCH-07](architecture.md#arch-07), [ARCH-09](architecture.md#arch-09).

**Primary requirement ownership:** [REQ-025](requirement.md#req-025), [REQ-026](requirement.md#req-026), [REQ-027](requirement.md#req-027), [REQ-039](requirement.md#req-039).

**Implementation scope:**

- Implement the Codex child specification and filtered, per-target environment. Centralize provider-environment policy from TASK-001 so Grok can reuse its own rule set later.
- Implement initialize-response/initialized ordering and the account/read helper with refreshToken:false.
- Map known ChatGPT, absent, API-key and unknown authentication cases to the agreed domain outcomes.
- Preserve the target fallback alias and ensure an email cannot be taken from a different process or cached global variable.
- Test two independently initialized fake Codex processes, inherited auth-key overrides, parent-environment immutability, and prohibited-method absence.

**Deliverables:** Codex wire/auth helpers and environment-policy tests. The full quota Query is completed in TASK-006.

**Acceptance criteria:**

- [x] Different Codex homes receive distinct absolute CODEX_HOME values and independent handshake state.
- [x] Known authentication outcomes are mapped without auth-file parsing, forced relogin, experimental features by default, or a model prompt.
- [x] Account identity and fallback behavior are deterministic and cross-account data cannot mix.
- [x] Environment filtering rules for both providers are represented/tested as pure construction policy, even though the Grok adapter is not implemented yet.

**Verification:** [V-CODEX](architecture.md#v-codex), [V-IDENTITY](architecture.md#v-identity), [V-SAFETY](architecture.md#v-safety). Run focused `go test ./internal/provider/codex` and environment tests, then `go test ./...`.

**Out of scope / stop conditions:** Do not switch accounts inside one process, read auth.json/keychain directly, use the ambient CODEX_HOME as the default selector, or emit an unfinished Query as a success.

**Commit boundary:** `feat(TASK-005): add isolated Codex account handshake`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-006](#task-006).

<a id="task-006"></a>
### TASK-006: Complete Codex weekly quota normalization

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Deliver a complete, truthful Codex Query implementation for every configured home.

**Dependency:** TASK-005 complete.

**Architecture:** [ARCH-04](architecture.md#arch-04), [ARCH-07](architecture.md#arch-07), [ARCH-11](architecture.md#arch-11).

**Primary requirement ownership:** [REQ-003](requirement.md#req-003), [REQ-028](requirement.md#req-028), [REQ-029](requirement.md#req-029), [REQ-030](requirement.md#req-030).

**Implementation scope:**

- Implement the rateLimits/read call and provider Query lifecycle using the completed auth helpers.
- Implement the authoritative multi-bucket policy, qualified legacy branch, and duration-based weekly window selection as pure functions.
- Handle explicit zero, missing usage, primary/secondary reversal, duplicate identical windows and conflicting weekly windows without guessing or averaging.
- Preserve source/quota/reset metadata internally and convert authentication, unavailable, transport and protocol failures to typed results.
- Add the full Codex normalization and isolated two-home transcript test matrix from V-CODEX.

**Deliverables:** Complete Codex provider Query and pure normalization functions, with captured/synthetic fixture provenance and tests.

**Acceptance criteria:**

- [x] Both weekly slot positions work and a five-hour or unrelated-model quota cannot be presented as the weekly Codex value.
- [x] A non-empty multi-bucket map without codex does not silently fall back to unrelated/legacy data.
- [x] Missing/null/invalid values remain distinguishable from a real 0.0% result; duplicates/conflicts follow the explicit policy.
- [x] The complete provider Query passes fake-process tests and uses only the four allowlisted Codex methods.

**Verification:** [V-CODEX](architecture.md#v-codex), [V-QUOTA](architecture.md#v-quota), [V-IDENTITY](architecture.md#v-identity). Run `go test ./internal/provider/codex`, relevant race tests with two homes, and `go test ./...`.

**Out of scope / stop conditions:** Do not read session history/token logs, fetch account/usage/read, redeem reset credits, clamp percentages or assume secondary means weekly.

**Commit boundary:** `feat(TASK-006): normalize Codex weekly subscription quotas`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** Close [EPIC-003](#epic-003) using its local gate, then start [TASK-007](#task-007) in [EPIC-004](#epic-004).

<a id="epic-004"></a>
## EPIC-004: Grok single-account provider

- [x] Epic complete. **Status: COMPLETE.**

**Goal:** Complete Grok metadata-only ACP billing access and explicit weekly/monthly normalization.

**Entry:** [EPIC-003](#epic-003) complete.

**Task sequence:** [TASK-007](#task-007) -> [TASK-008](#task-008)

**Local epic acceptance:** The complete Grok Query matches the previously qualified wire contract and passes the weekly/monthly/auth/unsupported/alias fixture suite. The early live capability gate must not have been waived.

**Handoff:** EPIC-005 receives both complete provider Query implementations.

<a id="task-007"></a>
### TASK-007: Add qualified Grok ACP authentication and billing

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Implement the exact metadata-only Grok wire contract established during the feasibility gate.

**Dependency:** EPIC-003 complete. TASK-001 Grok compatibility evidence must still match the implementation target.

**Architecture:** [ARCH-06](architecture.md#arch-06), [ARCH-08](architecture.md#arch-08), [ARCH-09](architecture.md#arch-09).

**Primary requirement ownership:** [REQ-031](requirement.md#req-031), [REQ-032](requirement.md#req-032), [REQ-033](requirement.md#req-033).

**Implementation scope:**

- Implement the qualified no-auto-update stdio child invocation and Grok-specific environment rules.
- Negotiate the verified ACP protocol version with filesystem/terminal capabilities disabled.
- Select only advertised cached_token authentication and send the qualified headless metadata; map absent/API-key/unknown advertisements without interactive fallback.
- Call the single qualified billing extension using the frozen wire name and wrapper. Do not send a bare internal name or iterate guessed encodings at runtime.
- Implement raw typed billing/error decoding and fake transcripts for success, auth rejection, protocol mismatch, method-not-found and unsafe callbacks.

**Deliverables:** Grok wire/auth/billing helpers, exact-envelope fixture tests and safe error mapping. Normalized Query completion belongs to TASK-008.

**Acceptance criteria:**

- [x] Actual outgoing bytes match the qualified ACP fixtures, including extension prefix/wrapper and jsonrpc marker.
- [x] No initialized notification, session creation, prompt, API-key fallback, browser login or REST request is introduced.
- [x] Correctly encoded method-not-found maps to UNSUPPORTED and remains a runtime compatibility failure, not a successful positive integration.
- [x] Authentication and callback failures terminate safely without leaking raw provider data.

**Verification:** [V-GROK](architecture.md#v-grok), [V-RPC](architecture.md#v-rpc), [V-SAFETY](architecture.md#v-safety). Run `go test ./internal/provider/grok ./internal/rpc`, exact-transcript assertions, and `go test ./...`.

**Out of scope / stop conditions:** Do not assume an internal Rust ExtRequest name is the raw wire method, ship alternate speculative contracts, or automatically upgrade/restart a shared Grok leader.

**Commit boundary:** `feat(TASK-007): add Grok ACP billing query transport`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-008](#task-008).

<a id="task-008"></a>
### TASK-008: Complete Grok weekly semantics and alias behavior

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Turn Grok billing metadata into the same truthful domain result without conflating monthly, prepaid or on-demand credits.

**Dependency:** TASK-007 complete.

**Architecture:** [ARCH-04](architecture.md#arch-04), [ARCH-08](architecture.md#arch-08), [ARCH-11](architecture.md#arch-11).

**Primary requirement ownership:** [REQ-013](requirement.md#req-013), [REQ-014](requirement.md#req-014), [REQ-015](requirement.md#req-015), [REQ-034](requirement.md#req-034), [REQ-035](requirement.md#req-035), [REQ-036](requirement.md#req-036).

**Implementation scope:**

- Implement a complete Grok Query around the established billing helper and pure result normalizer.
- Require the qualified explicit weekly period enum before reading creditUsagePercent as weekly usage.
- Map monthly, missing/unknown period, null config, missing/invalid percentage, and legacy-only payloads according to ARCH-08.
- Keep optional reset/source metadata and use the configured Grok alias consistently on both success and failure.
- Add pure and subprocess-backed cases for all Grok normalization outcomes and shared percentage-validation rules.

**Deliverables:** Complete Grok provider Query, normalization/alias tests and source-backed weekly/monthly fixtures.

**Acceptance criteria:**

- [x] Valid explicit weekly values, including zero, produce numeric results, while monthly values never occupy a weekly percentage cell.
- [x] Legacy monthlyLimit/used, prepaid balances and billing history are not converted into a weekly estimate.
- [x] Missing/null/unknown data remain explicit statuses, and valid optional fields do not become mandatory.
- [x] The full Grok adapter can now be consumed by the application runner without further protocol or authentication work.

**Verification:** [V-GROK](architecture.md#v-grok), [V-QUOTA](architecture.md#v-quota), [V-IDENTITY](architecture.md#v-identity). Run `go test ./internal/provider/grok ./internal/domain` and `go test ./...`.

**Out of scope / stop conditions:** Do not implement an email lookup, auth-file parsing, account discovery, cost conversion, or guessed numeric-enum mapping.

**Commit boundary:** `feat(TASK-008): normalize Grok weekly billing usage`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** Close [EPIC-004](#epic-004) using its local gate, then start [TASK-009](#task-009) in [EPIC-005](#epic-005).

<a id="epic-005"></a>
## EPIC-005: Usable snapshot CLI and offline acceptance

- [x] Epic complete. **Status: COMPLETE.**

**Goal:** Compose the providers into the requested output and verify the full program under normal and adverse conditions.

**Entry:** [EPIC-004](#epic-004) complete.

**Task sequence:** [TASK-009](#task-009) -> [TASK-010](#task-010) -> [TASK-011](#task-011)

**Local epic acceptance:** The actual CLI executable passes the offline default-table, concurrency, failure, security and exit-code acceptance tests. Product wiring must not be deferred to the release epic.

**Handoff:** EPIC-006 receives a feature-complete, offline-qualified program ready for native delivery and real-account acceptance.

<a id="task-009"></a>
### TASK-009: Integrate bounded snapshot coordination

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Run all targets independently and collect one ordered terminal result per account.

**Dependency:** EPIC-004 complete; both providers expose complete Query implementations.

**Architecture:** [ARCH-03](architecture.md#arch-03), [ARCH-04](architecture.md#arch-04), [ARCH-05](architecture.md#arch-05).

**Primary requirement ownership:** [REQ-004](requirement.md#req-004), [REQ-009](requirement.md#req-009), [REQ-022](requirement.md#req-022), [REQ-023](requirement.md#req-023).

**Implementation scope:**

- Compose Codex/Grok providers behind the one-target Query interface and fixed result slots.
- Implement at most three active targets, with per-target admission-time query budgets and independent errors.
- Propagate root signal cancellation to active work and mark queued work cancelled without launching it.
- Preserve successful results when another provider fails or hangs; no fail-fast cancellation or automatic retry.
- Test deterministic ordering under deliberately reversed completion order, more than three custom Codex homes, and the default three-target case.

**Deliverables:** Application runner, signal-context integration boundary and coordinator tests.

**Acceptance criteria:**

- [x] The default three targets run independently and the output model remains in configured order.
- [x] There is exactly one Query attempt for each admitted target and no more than three active targets.
- [x] A timeout in one home does not erase or cancel another account’s valid result.
- [x] Queued/active cancellation and cleanup follow the configured deadline and signal contracts.

**Verification:** [V-RUNNER](architecture.md#v-runner), [V-LIFECYCLE](architecture.md#v-lifecycle). Run `go test ./internal/app`, relevant race tests, and `go test ./...`.

**Out of scope / stop conditions:** Do not add a scheduler service, watch loop, progress UI, global CODEX_HOME switching, cache, or background worker that survives the invocation.

**Commit boundary:** `feat(TASK-009): coordinate bounded account snapshots`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-010](#task-010).

<a id="task-010"></a>
### TASK-010: Deliver the final table and CLI exit behavior

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Expose the user-requested command as a complete four-line default view with stable errors and identities.

**Dependency:** TASK-009 complete.

**Architecture:** [ARCH-03](architecture.md#arch-03), [ARCH-04](architecture.md#arch-04), [ARCH-10](architecture.md#arch-10).

**Primary requirement ownership:** [REQ-002](requirement.md#req-002), [REQ-006](requirement.md#req-006), [REQ-010](requirement.md#req-010), [REQ-011](requirement.md#req-011), [REQ-012](requirement.md#req-012), [REQ-024](requirement.md#req-024).

**Implementation scope:**

- Wire the approved options parser, signal handler, providers, runner and renderer in cmd/gaebokchi.
- Implement the three-column header and data rows, one-decimal percentages, final newline, and all defined status cells.
- Prefer protocol Codex emails, apply fallback/disambiguation and sanitize account labels; keep Grok’s explicit alias.
- Route safe bounded diagnostics to stderr only. Handle output write/flush failures without dumping provider data.
- Implement help/version without process launch and exit codes 0/1/2/130/143, including non-applicable versus incomplete results.

**Deliverables:** Working command composition root, renderer, command-output and exit tests, initial README invocation examples.

**Acceptance criteria:**

- [x] The default executable emits exactly a header and three account lines, and custom homes obey the N+2 rule.
- [x] Success, partial failure and monthly/API-key non-applicability produce the exact output/exit behavior defined by architecture.md.
- [x] Labels cannot inject columns/lines/control sequences and duplicate Codex identities remain distinguishable.
- [x] Help/version never invoke a provider; stderr/stdout contracts and broken-output behavior are tested.

**Verification:** [V-OPTIONS](architecture.md#v-options), [V-IDENTITY](architecture.md#v-identity), [V-OUTPUT](architecture.md#v-output), [V-EXIT](architecture.md#v-exit). Run `go test ./internal/render ./cmd/gaebokchi` plus options/app tests and `go test ./...`.

**Out of scope / stop conditions:** Do not add reset-time columns, borders, extra titles, totals, color, prompts, JSON mode or interactive account linking.

**Commit boundary:** `feat(TASK-010): expose stable usage table and exit contract`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-011](#task-011).

<a id="task-011"></a>
### TASK-011: Qualify the executable with offline fault and safety tests

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Verify that the integrated program satisfies product behavior under failure, rather than only testing provider helpers.

**Dependency:** TASK-010 complete.

**Architecture:** [ARCH-05](architecture.md#arch-05), [ARCH-06](architecture.md#arch-06), [ARCH-09](architecture.md#arch-09), [ARCH-10](architecture.md#arch-10), [ARCH-11](architecture.md#arch-11).

**Primary requirement ownership:** [REQ-021](requirement.md#req-021), [REQ-037](requirement.md#req-037), [REQ-038](requirement.md#req-038), [REQ-041](requirement.md#req-041), [REQ-044](requirement.md#req-044), [REQ-045](requirement.md#req-045).

**Implementation scope:**

- Build/run the actual application with fake codex/grok executables and temporary non-secret account homes.
- Cover default success, missing home/binary, failed auth, monthly/unknown Grok period, malformed frames, wrong IDs, long stderr, hung provider, callbacks, signals and output errors.
- Assert all outgoing requests, capabilities, env isolation, row order/count, stderr separation and exit status. Use secret markers to detect leaks.
- Inspect production imports/file operations for unexpected credential reads, HTTP billing clients, persistent state, shell execution or agent-work capabilities.
- Close only evidence-backed implementation defects within this task’s integration scope. Record unrelated refinements separately rather than expanding the release.

**Deliverables:** Executable-level offline acceptance suite, safety/fault report, and evidence links for cross-cutting requirements.

**Acceptance criteria:**

- [x] The full offline suite passes without provider installations, network access or actual logged-in homes.
- [x] A failing or hung target cannot prevent other account rows from being produced within the execution policy.
- [x] No synthetic secret reaches stdout/stderr or saved test evidence and no forbidden outbound method is used.
- [x] All EPIC-005 local acceptance criteria are satisfied; no product wiring is left for the release epic.

**Verification:** [V-E2E](architecture.md#v-e2e), [V-SAFETY](architecture.md#v-safety), [V-RPC](architecture.md#v-rpc), [V-QUOTA](architecture.md#v-quota), [V-EXIT](architecture.md#v-exit). Run `go test ./test/integration/...`, `go test ./...`, `go vet ./...`, and the supported native race suite.

**Out of scope / stop conditions:** Do not replace deterministic tests with live-account calls, add a mandatory external reviewer dependency, or start an unbounded review/repair loop.

**Commit boundary:** `test(TASK-011): qualify CLI isolation and failure behavior`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** Close [EPIC-005](#epic-005) using its local gate, then start [TASK-012](#task-012) in [EPIC-006](#epic-006).

<a id="epic-006"></a>
## EPIC-006: Platform delivery and final qualification

- [ ] Epic complete. **Status: NOT STARTED.**

**Goal:** Build and test macOS/Linux deliverables, verify real accounts, and close the specification with evidence.

**Entry:** [EPIC-005](#epic-005) complete.

**Task sequence:** [TASK-012](#task-012) -> [TASK-013](#task-013) -> [TASK-014](#task-014)

**Local epic acceptance:** All requirements are evidence-backed; native macOS/Linux tests and the authorized live account campaign pass; local artifacts and user documentation are complete. Publication is not part of this epic.

**Handoff:** Stop implementation. Provide the verified release handoff and limitations; do not start Claude or another feature automatically.

<a id="task-012"></a>
### TASK-012: Add portable build delivery and user documentation

- [ ] Task complete. **Status: NOT STARTED.**

**Purpose:** Make the implemented command reproducibly buildable and usable on macOS/Linux without overstating platform evidence.

**Dependency:** EPIC-005 complete.

**Architecture:** [ARCH-02](architecture.md#arch-02), [ARCH-11](architecture.md#arch-11), [ARCH-13](architecture.md#arch-13), [ARCH-14](architecture.md#arch-14).

**Primary requirement ownership:** [REQ-001](requirement.md#req-001), [REQ-046](requirement.md#req-046), [REQ-047](requirement.md#req-047), [REQ-049](requirement.md#req-049).

**Implementation scope:**

- Add native macOS and Linux CI or equivalent recorded runs for unit/fake-provider integration suites. Pin the tested compiler.
- Add a portable release-build script for the four GOOS/GOARCH targets, embedded version information and SHA-256 checksums.
- Build locally staged release archives with the `gaebokchi` prefix and the `gaebokchi` executable, without publishing or bundling provider executables.
- Finish the **Gaebokchi (개복치)** README with its Aquarium-tool identity, install/build/use/troubleshooting documentation, flags, statuses, exits, existing-login prerequisites, privacy and provider-owned side effects.
- Add the cross-document/link check and an honest build/native/live matrix, distinguishing native OS tests from cross-built CPU architectures.

**Deliverables:** README, CI configuration or equivalent native evidence, build/checksum script, versioned local artifacts and documentation-link checker.

**Acceptance criteria:**

- [ ] All four binaries build; native application/fake-provider tests pass on macOS and Linux.
- [ ] Version/checksum information is present and no provider CLI binary or secret is packaged.
- [ ] The README matches the implemented command and labels actual tested versions/platforms without unsupported compatibility claims.
- [ ] The document-link check passes and local release staging requires no signing, upload or publishing permission.

**Verification:** [V-BUILD](architecture.md#v-build), [V-PLATFORM](architecture.md#v-platform), [V-DOCS](architecture.md#v-docs), [V-AUDIT](architecture.md#v-audit). Run the documented build script, native test matrix, `go run scripts/check-docs.go`, README smoke commands and checksum verification.

**Out of scope / stop conditions:** Do not create a Homebrew tap, upload releases, choose a project license without authorization, include credentials, or claim all architectures were natively tested.

**Commit boundary:** `build(TASK-012): add macOS Linux delivery and usage guide`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-013](#task-013).

<a id="task-013"></a>
### TASK-013: Run delivered-binary live acceptance

- [ ] Task complete. **Status: NOT STARTED.**

**Purpose:** Confirm that the completed executable, not just the early probe or fake servers, works with the actual requested account set.

**Dependency:** TASK-012 complete; an authorized host with both Codex homes and an existing Grok login is available.

**Architecture:** [ARCH-07](architecture.md#arch-07), [ARCH-08](architecture.md#arch-08), [ARCH-11](architecture.md#arch-11), [ARCH-12](architecture.md#arch-12), [ARCH-13](architecture.md#arch-13).

**Primary requirement ownership:** [REQ-005](requirement.md#req-005), [REQ-048](requirement.md#req-048).

**Implementation scope:**

- Record binary/CLI versions, source revision, host OS/CPU and the approved account-target aliases. Keep identifying details local where unnecessary.
- Run the delivered binary against ~/.codex, ~/.codex-hsy and the existing Grok login without prompting or account repair.
- Compare each returned allowance against the provider’s own usage display in the same period, using read-only UI/commands only.
- Record successful numeric Codex results and a valid Grok weekly or explicit monthly result. Describe a monthly account as monthly, not live weekly coverage.
- Check normal cleanup, partial-error presentation with an isolated missing target override, absence of app-owned persistent state, and unaffected existing agent sessions.

**Deliverables:** Sanitized `docs/verification/live-acceptance.md`, updated compatibility/native/live matrix and evidence-linked requirement checkboxes.

**Acceptance criteria:**

- [ ] Both requested Codex home queries work through the completed binary and their identity/quota attribution is correct.
- [ ] The Grok path produces real valid billing metadata; UNSUPPORTED, TIMEOUT or a fixture replay cannot satisfy this gate.
- [ ] Provider-display comparisons document period resets, concurrent usage and numeric precision rather than claiming exact simultaneity.
- [ ] The evidence explicitly separates live-tested behavior, fixture-only weekly cases, cross-built targets and untested combinations.

**Verification:** [V-LIVE](architecture.md#v-live), [V-CONTRACT](architecture.md#v-contract), [V-SAFETY](architecture.md#v-safety). Record the exact delivered-binary invocation and safe comparison steps. Run focused deterministic regressions after any discovered defect; do not automatically repeat the entire live campaign.

**Out of scope / stop conditions:** Do not run login/logout, open a new model turn to refresh statistics, alter real auth files to simulate failure, auto-upgrade the CLI, or convert a missing environment into a pass.

**Commit boundary:** `test(TASK-013): record real-account usage acceptance`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-014](#task-014).

<a id="task-014"></a>
### TASK-014: Close requirements and prepare the v0.1 handoff

- [ ] Task complete. **Status: NOT STARTED.**

**Purpose:** Finish the project with complete traceability and truthful evidence, without turning closeout into another redesign cycle.

**Dependency:** TASK-013 complete and no blocking product defect remains.

**Architecture:** [ARCH-11](architecture.md#arch-11), [ARCH-12](architecture.md#arch-12), [ARCH-13](architecture.md#arch-13), [ARCH-14](architecture.md#arch-14).

**Primary requirement ownership:** [REQ-050](requirement.md#req-050).

**Implementation scope:**

- Audit every requirement against its implementation, tests, architecture reference and live evidence where required.
- Check all relative links/anchors and reconcile task/epic/requirement statuses. A skipped or blocked verification remains visible and prevents unsupported completion claims.
- Run the final offline regression/document checks once against the intended release tree and review only unresolved acceptance gaps.
- Update the three root documents and release summary with verified scope, limitations, local artifact locations and next maintenance guidance.
- Complete the one task-scoped closeout commit. Do not self-reference its unknown hash or create another commit merely to insert that hash.

**Deliverables:** Completed requirement/plan checklists, final verification/release summary and consistent architecture/README compatibility statements.

**Acceptance criteria:**

- [ ] All 50 requirement checkboxes have valid evidence, including positive provider qualification rather than fallback-only tests.
- [ ] All six epics satisfy their local gates, and no task remains BLOCKED or falsely marked complete.
- [ ] Required deterministic, native-platform and live-account checks are accurately represented; known limitations are not hidden.
- [ ] The handoff is complete with locally staged artifacts; no remote publication or next-version feature implementation occurred.

**Verification:** [V-AUDIT](architecture.md#v-audit), [V-DOCS](architecture.md#v-docs), [V-BUILD](architecture.md#v-build). Run `go test ./...`, `go vet ./...`, `go run scripts/check-docs.go`, and audit the final evidence matrix. Reuse unchanged valid native/live evidence rather than rerunning it for documentation-only edits.

**Out of scope / stop conditions:** Do not reopen completed epics for cosmetic preferences, regenerate fixtures from a changed upstream without review, or repeat full reviews indefinitely.

**Commit boundary:** `docs(TASK-014): close v0.1 requirements and handoff`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** Close EPIC-006 and stop. Return the final verified handoff; do not implement another provider automatically.

<a id="plan-release-gate"></a>
## Final release gate

These are release-level checks, not extra implementation tasks:

- [ ] Every checkbox in [requirement.md](requirement.md) has evidence, with live and fixture-only claims distinguished.
- [ ] All six epics and fourteen tasks have completed their stated local acceptance criteria in sequence.
- [ ] Default output is exactly three columns and four logical lines, including on partial provider failure.
- [ ] Codex works through both specified homes; Grok produces valid real billing metadata through the qualified prompt-free CLI protocol.
- [ ] Offline tests, native macOS/Linux tests, four target builds, document-link checks and owner-account acceptance are recorded truthfully.
- [ ] No Claude/PTY, private REST fallback, credential parsing, agent prompt/session, app-owned daemon or persistent usage store has been introduced.
- [ ] Local release artifacts and README are ready, with tested-version/platform limitations and no unauthorized publication.

A missing live environment is a real qualification gap, not a reason to invent evidence. Stop with a precise incomplete-gate report instead of asking an implementation AI to loop until an unavailable service becomes available.

## Final handoff format

Return the implemented scope, task commit references, verification summary, local artifact locations/checksums, and remaining compatibility limitations. Explain any live monthly Grok result explicitly. Report failures or unrun environments without presenting them as completed support.

The next version is outside this plan. In particular, Claude support, machine-readable JSON output, persistent configuration, reset-time columns, additional Grok accounts, and watch mode require separate scope decisions.
