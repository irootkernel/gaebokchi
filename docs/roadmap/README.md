# Gaebokchi (개복치): Sequential Implementation Plan

Version: 0.1 specification  
Status: EPIC-001 through EPIC-010 COMPLETE; v0.1.0 public release acceptance remains open for current configuration, Claude, and REQ-048.
Prepared: 2026-09-22  
Project: Gaebokchi (개복치)  
Ecosystem: Aquarium tool  
Repository and executable: `gaebokchi`

The Claude extension and YAML account configuration are described in the
[current requirements](../specs/README.md) and [architecture](../architecture/README.md).
Historical task records below retain the meaning they had when completed;
the [current release gate](#current-release-gate) governs v0.1.0 publication.

## How to use this plan

This document owns **execution order and completion state** for **Gaebokchi (개복치)**, an Aquarium tool. Implement against the [architecture](../architecture/README.md), and verify product coverage against the checkboxes in the [requirements](../specs/README.md). Release archives carry this three-document contract, while links to verification records resolve only in a source checkout. The original six epics were implemented and committed with per-task verification records under docs/verification/; EPIC-007 completed the post-closeout corrective work. [EPIC-008](#epic-008) owns the version-command contract, [EPIC-009](#epic-009) owns usage JSON, and [EPIC-010](#epic-010) owns the interactive terminal view.

Keep all three canonical documents under their role directories. Start with the [execution rules](#plan-execution), then the first incomplete task in the current epic. Read the linked architecture sections and requirements for that task rather than repeatedly loading all documentation for a narrow change.

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
EPIC-001 -> EPIC-002 -> EPIC-003 -> EPIC-004 -> EPIC-005 -> EPIC-006 -> EPIC-007 -> EPIC-008 -> EPIC-009 -> EPIC-010 -> EPIC-011
```

| Order | Epic | Tasks, executed left to right | Entry dependency | Current status (reconciled) |
|---|---|---|---|---|
| 1 | [EPIC-001: Contract qualification and foundation](#epic-001) | [TASK-001](#task-001) -> [TASK-002](#task-002) | None | COMPLETE (implementation milestone) |
| 2 | [EPIC-002: Safe subprocess and RPC foundation](#epic-002) | [TASK-003](#task-003) -> [TASK-004](#task-004) | [EPIC-001](#epic-001) complete | COMPLETE (implementation milestone) |
| 3 | [EPIC-003: Codex multi-account provider](#epic-003) | [TASK-005](#task-005) -> [TASK-006](#task-006) | [EPIC-002](#epic-002) complete | COMPLETE (implementation milestone) |
| 4 | [EPIC-004: Grok single-account provider](#epic-004) | [TASK-007](#task-007) -> [TASK-008](#task-008) | [EPIC-003](#epic-003) complete | COMPLETE (implementation milestone) |
| 5 | [EPIC-005: Usable snapshot CLI and offline acceptance](#epic-005) | [TASK-009](#task-009) -> [TASK-010](#task-010) -> [TASK-011](#task-011) | [EPIC-004](#epic-004) complete | COMPLETE (implementation milestone) |
| 6 | [EPIC-006: Platform delivery and final qualification](#epic-006) | [TASK-012](#task-012) -> [TASK-013](#task-013) -> [TASK-014](#task-014) | [EPIC-005](#epic-005) complete | COMPLETE (implementation milestone) |
| 7 | [EPIC-007: Corrective remediation and re-acceptance](#epic-007) | [TASK-015](#task-015) -> [TASK-016](#task-016) -> [TASK-017](#task-017) -> [TASK-018](#task-018) -> [TASK-019](#task-019) -> [TASK-020](#task-020) -> [TASK-021](#task-021) -> [TASK-022](#task-022) -> [TASK-023](#task-023) -> [TASK-024](#task-024) -> [TASK-025](#task-025) -> [TASK-026](#task-026) -> [TASK-027](#task-027) | [EPIC-006](#epic-006) implemented | COMPLETE (final release acceptance pending REQ-048) |
| 8 | [EPIC-008: Version command and release identity](#epic-008) | [TASK-028](#task-028) | [EPIC-007](#epic-007) complete | COMPLETE (final release acceptance pending REQ-048) |
| 9 | [EPIC-009: Machine-readable usage snapshot](#epic-009) | [TASK-029](#task-029) | [EPIC-008](#epic-008) complete | COMPLETE (final release acceptance pending REQ-048) |
| 10 | [EPIC-010: Interactive terminal view](#epic-010) | [TASK-030](#task-030) | [EPIC-009](#epic-009) complete | COMPLETE (final release acceptance pending REQ-048) |
| 11 | [EPIC-011: Public v0.1.0 release readiness](#epic-011) | [TASK-031](#task-031) -> [TASK-032](#task-032) -> [TASK-033](#task-033) | [EPIC-010](#epic-010) complete | IN PROGRESS |

Historical note: every epic began NOT STARTED at plan authoring. The reconciled
column above shows each epic's current state. The original six epics keep their
implementation milestones and task commits (with EPIC-006's final acceptance
superseded by EPIC-007, as the TASK-013/TASK-014 supersession notes record).
EPIC-007's corrective work is complete. EPIC-008 is the version-contract
extension; EPIC-009 completed the JSON-output extension. EPIC-010 completed
the interactive terminal view. EPIC-011 owns current public-release readiness:
YAML and Claude acceptance, packaging, and the REQ-048 live gate.

The task order is globally sequential: TASK-001 through TASK-033. Runtime
account queries may be concurrent; development tasks may not.

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

**Architecture:** [ARCH-06](../architecture/README.md#arch-06), [ARCH-07](../architecture/README.md#arch-07), [ARCH-08](../architecture/README.md#arch-08), [ARCH-09](../architecture/README.md#arch-09), [ARCH-12](../architecture/README.md#arch-12), [ARCH-15](../architecture/README.md#arch-15).

**Primary requirement ownership:** [REQ-042](../specs/README.md#req-042).

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

**Verification:** [V-CONTRACT](../architecture/README.md#v-contract), [V-SAFETY](../architecture/README.md#v-safety). Record the exact bounded probe invocations and results. Parse saved JSON fixtures with a local JSON parser and inspect the outgoing allowlist. There is no Go-module test requirement before TASK-002.

**Out of scope / stop conditions:** Do not implement REST/token/browser/PTY fallbacks, broaden the account model, or treat source inspection alone as a live pass. A correctly encoded unsupported Grok endpoint blocks the epic; report the specific prerequisite instead of implementing around it.

**Commit boundary:** `docs(TASK-001): qualify Codex and Grok metadata contracts`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-002](#task-002).

<a id="task-002"></a>
### TASK-002: Bootstrap the Go module, domain and target policy

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Create the smallest compilable foundation that expresses the approved CLI targets and status semantics without depending on a live provider.

**Dependency:** TASK-001 complete, including a successful Grok metadata-only contract gate.

**Architecture:** [ARCH-01](../architecture/README.md#arch-01), [ARCH-02](../architecture/README.md#arch-02), [ARCH-03](../architecture/README.md#arch-03), [ARCH-04](../architecture/README.md#arch-04), [ARCH-14](../architecture/README.md#arch-14).

**Primary requirement ownership:** [REQ-007](../specs/README.md#req-007), [REQ-008](../specs/README.md#req-008).

**Implementation scope:**

- Initialize the module from the actual repository identity, or use a documented local-only module path. Use `gaebokchi` as the repository basename and `cmd/gaebokchi/main.go` as the command entry point, following the [ARCH-01 naming contract](../architecture/README.md#arch-01). Record a maintained tested Go toolchain rather than assuming a version.
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

**Verification:** [V-OPTIONS](../architecture/README.md#v-options), [V-QUOTA](../architecture/README.md#v-quota), [V-IDENTITY](../architecture/README.md#v-identity). Run `go test ./internal/domain ./internal/options`, `go test ./...`, and `go vet ./...`. Record the selected compiler and module path.

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

**Architecture:** [ARCH-02](../architecture/README.md#arch-02), [ARCH-05](../architecture/README.md#arch-05), [ARCH-09](../architecture/README.md#arch-09).

**Primary requirement ownership:** [REQ-020](../specs/README.md#req-020), [REQ-040](../specs/README.md#req-040).

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

**Verification:** [V-LIFECYCLE](../architecture/README.md#v-lifecycle), [V-SAFETY](../architecture/README.md#v-safety). Run `go test ./internal/process` and the focused race/lifecycle tests on the current native host. Cross-platform qualification is owned by TASK-012.

**Out of scope / stop conditions:** Do not use `pkill`, shell wrappers, a shared application daemon, parent-environment mutation, or raw provider stderr in diagnostics.

**Commit boundary:** `feat(TASK-003): add bounded provider process lifecycle`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-004](#task-004).

<a id="task-004"></a>
### TASK-004: Implement framed RPC with provider-specific codecs

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Provide a reusable, well-tested message transport without flattening the differences between Codex App Server and ACP.

**Dependency:** TASK-003 complete.

**Architecture:** [ARCH-06](../architecture/README.md#arch-06), [ARCH-09](../architecture/README.md#arch-09), [ARCH-11](../architecture/README.md#arch-11).

**Primary requirement ownership:** [REQ-016](../specs/README.md#req-016), [REQ-017](../specs/README.md#req-017), [REQ-018](../specs/README.md#req-018), [REQ-019](../specs/README.md#req-019), [REQ-043](../specs/README.md#req-043).

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

**Verification:** [V-RPC](../architecture/README.md#v-rpc), [V-SAFETY](../architecture/README.md#v-safety), [V-LIFECYCLE](../architecture/README.md#v-lifecycle). Run `go test ./internal/rpc ./internal/process`, relevant race tests, and `go test ./...`.

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

**Architecture:** [ARCH-03](../architecture/README.md#arch-03), [ARCH-05](../architecture/README.md#arch-05), [ARCH-07](../architecture/README.md#arch-07), [ARCH-09](../architecture/README.md#arch-09).

**Primary requirement ownership:** [REQ-025](../specs/README.md#req-025), [REQ-026](../specs/README.md#req-026), [REQ-027](../specs/README.md#req-027), [REQ-039](../specs/README.md#req-039).

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

**Verification:** [V-CODEX](../architecture/README.md#v-codex), [V-IDENTITY](../architecture/README.md#v-identity), [V-SAFETY](../architecture/README.md#v-safety). Run focused `go test ./internal/provider/codex` and environment tests, then `go test ./...`.

**Out of scope / stop conditions:** Do not switch accounts inside one process, read auth.json/keychain directly, use the ambient CODEX_HOME as the default selector, or emit an unfinished Query as a success.

**Commit boundary:** `feat(TASK-005): add isolated Codex account handshake`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-006](#task-006).

<a id="task-006"></a>
### TASK-006: Complete Codex weekly quota normalization

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Deliver a complete, truthful Codex Query implementation for every configured home.

**Dependency:** TASK-005 complete.

**Architecture:** [ARCH-04](../architecture/README.md#arch-04), [ARCH-07](../architecture/README.md#arch-07), [ARCH-11](../architecture/README.md#arch-11).

**Primary requirement ownership:** [REQ-003](../specs/README.md#req-003), [REQ-028](../specs/README.md#req-028), [REQ-029](../specs/README.md#req-029), [REQ-030](../specs/README.md#req-030).

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

**Verification:** [V-CODEX](../architecture/README.md#v-codex), [V-QUOTA](../architecture/README.md#v-quota), [V-IDENTITY](../architecture/README.md#v-identity). Run `go test ./internal/provider/codex`, relevant race tests with two homes, and `go test ./...`.

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

**Architecture:** [ARCH-06](../architecture/README.md#arch-06), [ARCH-08](../architecture/README.md#arch-08), [ARCH-09](../architecture/README.md#arch-09).

**Primary requirement ownership:** [REQ-031](../specs/README.md#req-031), [REQ-032](../specs/README.md#req-032), [REQ-033](../specs/README.md#req-033).

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

**Verification:** [V-GROK](../architecture/README.md#v-grok), [V-RPC](../architecture/README.md#v-rpc), [V-SAFETY](../architecture/README.md#v-safety). Run `go test ./internal/provider/grok ./internal/rpc`, exact-transcript assertions, and `go test ./...`.

**Out of scope / stop conditions:** Do not assume an internal Rust ExtRequest name is the raw wire method, ship alternate speculative contracts, or automatically upgrade/restart a shared Grok leader.

**Commit boundary:** `feat(TASK-007): add Grok ACP billing query transport`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-008](#task-008).

<a id="task-008"></a>
### TASK-008: Complete Grok weekly semantics and alias behavior

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Turn Grok billing metadata into the same truthful domain result without conflating monthly, prepaid or on-demand credits.

**Dependency:** TASK-007 complete.

**Architecture:** [ARCH-04](../architecture/README.md#arch-04), [ARCH-08](../architecture/README.md#arch-08), [ARCH-11](../architecture/README.md#arch-11).

**Primary requirement ownership:** [REQ-013](../specs/README.md#req-013), [REQ-014](../specs/README.md#req-014), [REQ-015](../specs/README.md#req-015), [REQ-034](../specs/README.md#req-034), [REQ-035](../specs/README.md#req-035), [REQ-036](../specs/README.md#req-036).

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

**Verification:** [V-GROK](../architecture/README.md#v-grok), [V-QUOTA](../architecture/README.md#v-quota), [V-IDENTITY](../architecture/README.md#v-identity). Run `go test ./internal/provider/grok ./internal/domain` and `go test ./...`.

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

**Architecture:** [ARCH-03](../architecture/README.md#arch-03), [ARCH-04](../architecture/README.md#arch-04), [ARCH-05](../architecture/README.md#arch-05).

**Primary requirement ownership:** [REQ-004](../specs/README.md#req-004), [REQ-009](../specs/README.md#req-009), [REQ-022](../specs/README.md#req-022), [REQ-023](../specs/README.md#req-023).

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

**Verification:** [V-RUNNER](../architecture/README.md#v-runner), [V-LIFECYCLE](../architecture/README.md#v-lifecycle). Run `go test ./internal/app`, relevant race tests, and `go test ./...`.

**Out of scope / stop conditions:** Do not add a scheduler service, watch loop, progress UI, global CODEX_HOME switching, cache, or background worker that survives the invocation.

**Commit boundary:** `feat(TASK-009): coordinate bounded account snapshots`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-010](#task-010).

<a id="task-010"></a>
### TASK-010: Deliver the final table and CLI exit behavior

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Expose the user-requested command as a complete four-line default view with stable errors and identities.

**Dependency:** TASK-009 complete.

**Architecture:** [ARCH-03](../architecture/README.md#arch-03), [ARCH-04](../architecture/README.md#arch-04), [ARCH-10](../architecture/README.md#arch-10).

**Primary requirement ownership:** [REQ-002](../specs/README.md#req-002), [REQ-006](../specs/README.md#req-006), [REQ-010](../specs/README.md#req-010), [REQ-011](../specs/README.md#req-011), [REQ-012](../specs/README.md#req-012), [REQ-024](../specs/README.md#req-024).

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

**Verification:** [V-OPTIONS](../architecture/README.md#v-options), [V-IDENTITY](../architecture/README.md#v-identity), [V-OUTPUT](../architecture/README.md#v-output), [V-EXIT](../architecture/README.md#v-exit). Run `go test ./internal/render ./cmd/gaebokchi` plus options/app tests and `go test ./...`.

**Out of scope / stop conditions:** Do not add reset-time columns, borders, extra titles, totals, color, prompts, JSON mode or interactive account linking.

**Commit boundary:** `feat(TASK-010): expose stable usage table and exit contract`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-011](#task-011).

<a id="task-011"></a>
### TASK-011: Qualify the executable with offline fault and safety tests

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Verify that the integrated program satisfies product behavior under failure, rather than only testing provider helpers.

**Dependency:** TASK-010 complete.

**Architecture:** [ARCH-05](../architecture/README.md#arch-05), [ARCH-06](../architecture/README.md#arch-06), [ARCH-09](../architecture/README.md#arch-09), [ARCH-10](../architecture/README.md#arch-10), [ARCH-11](../architecture/README.md#arch-11).

**Primary requirement ownership:** [REQ-021](../specs/README.md#req-021), [REQ-037](../specs/README.md#req-037), [REQ-038](../specs/README.md#req-038), [REQ-041](../specs/README.md#req-041), [REQ-044](../specs/README.md#req-044), [REQ-045](../specs/README.md#req-045).

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

**Verification:** [V-E2E](../architecture/README.md#v-e2e), [V-SAFETY](../architecture/README.md#v-safety), [V-RPC](../architecture/README.md#v-rpc), [V-QUOTA](../architecture/README.md#v-quota), [V-EXIT](../architecture/README.md#v-exit). Run `go test ./test/integration/...`, `go test ./...`, `go vet ./...`, and the supported native race suite.

**Out of scope / stop conditions:** Do not replace deterministic tests with live-account calls, add a mandatory external reviewer dependency, or start an unbounded review/repair loop.

**Commit boundary:** `test(TASK-011): qualify CLI isolation and failure behavior`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** Close [EPIC-005](#epic-005) using its local gate, then start [TASK-012](#task-012) in [EPIC-006](#epic-006).

<a id="epic-006"></a>
## EPIC-006: Platform delivery and final qualification

- [x] Epic complete. **Status: COMPLETE.**

**Goal:** Build and test macOS/Linux deliverables, verify real accounts, and close the specification with evidence.

**Entry:** [EPIC-005](#epic-005) complete.

**Task sequence:** [TASK-012](#task-012) -> [TASK-013](#task-013) -> [TASK-014](#task-014)

**Local epic acceptance:** All requirements are evidence-backed; native macOS/Linux tests and the authorized live account campaign pass; local artifacts and user documentation are complete. Publication is not part of this epic.

**Handoff:** Stop implementation. Provide the verified release handoff and limitations; do not start Claude or another feature automatically.

<a id="task-012"></a>
### TASK-012: Add portable build delivery and user documentation

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Make the implemented command reproducibly buildable and usable on macOS/Linux without overstating platform evidence.

**Dependency:** EPIC-005 complete.

**Architecture:** [ARCH-02](../architecture/README.md#arch-02), [ARCH-11](../architecture/README.md#arch-11), [ARCH-13](../architecture/README.md#arch-13), [ARCH-14](../architecture/README.md#arch-14).

**Primary requirement ownership:** [REQ-001](../specs/README.md#req-001), [REQ-046](../specs/README.md#req-046), [REQ-047](../specs/README.md#req-047), [REQ-049](../specs/README.md#req-049).

**Implementation scope:**

- Add native macOS and Linux CI or equivalent recorded runs for unit/fake-provider integration suites. Pin the tested compiler.
- Add a portable release-build script for the four GOOS/GOARCH targets, embedded version information and SHA-256 checksums.
- Build locally staged release archives with the `gaebokchi` prefix and the `gaebokchi` executable, without publishing or bundling provider executables.
- Finish the **Gaebokchi (개복치)** README with its Aquarium-tool identity, install/build/use/troubleshooting documentation, flags, statuses, exits, existing-login prerequisites, privacy and provider-owned side effects.
- Add the cross-document/link check and an honest build/native/live matrix, distinguishing native OS tests from cross-built CPU architectures.

**Deliverables:** README, CI configuration or equivalent native evidence, build/checksum script, versioned local artifacts and documentation-link checker.

**Acceptance criteria:**

- [x] All four binaries build; native application/fake-provider tests pass on macOS and Linux.
- [x] Version/checksum information is present and no provider CLI binary or secret is packaged.
- [x] The README matches the implemented command and labels actual tested versions/platforms without unsupported compatibility claims.
- [x] The document-link check passes and local release staging requires no signing, upload or publishing permission.

**Verification:** [V-BUILD](../architecture/README.md#v-build), [V-PLATFORM](../architecture/README.md#v-platform), [V-DOCS](../architecture/README.md#v-docs), [V-AUDIT](../architecture/README.md#v-audit). Run the documented build script, native test matrix, `go run scripts/check-docs.go`, README smoke commands and checksum verification.

**Out of scope / stop conditions:** Do not create a Homebrew tap, upload releases, choose a project license without authorization, include credentials, or claim all architectures were natively tested.

**Commit boundary:** `build(TASK-012): add macOS Linux delivery and usage guide`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-013](#task-013).

<a id="task-013"></a>
### TASK-013: Run delivered-binary live acceptance

- [x] Task complete. **Status: COMPLETE.**

> Supersession note (2026-09-22): the live prompt-free metadata queries recorded in [docs/verification/live-acceptance.md](../verification/live-acceptance.md) remain valid evidence for the artifact and source revision tested at that time. However, the provider's own usage-display comparison required by [REQ-048](../specs/README.md#req-048) was not performed, so the acceptance portion of this task is superseded and reopened under [EPIC-007](#epic-007) (TASK-021) pending a properly authorized comparison. The "Task complete" marker above reflects the historical commit, not current acceptance.

**Purpose:** Confirm that the completed executable, not just the early probe or fake servers, works with the actual requested account set.

**Dependency:** TASK-012 complete; an authorized host with both Codex homes and an existing Grok login is available.

**Architecture:** [ARCH-07](../architecture/README.md#arch-07), [ARCH-08](../architecture/README.md#arch-08), [ARCH-11](../architecture/README.md#arch-11), [ARCH-12](../architecture/README.md#arch-12), [ARCH-13](../architecture/README.md#arch-13).

**Primary requirement ownership:** [REQ-005](../specs/README.md#req-005), [REQ-048](../specs/README.md#req-048).

**Implementation scope:**

- Record binary/CLI versions, source revision, host OS/CPU and the approved account-target aliases. Keep identifying details local where unnecessary.
- Run the delivered binary against ~/.codex, ~/.codex-hsy and the existing Grok login without prompting or account repair.
- Compare each returned allowance against the provider’s own usage display in the same period, using read-only UI/commands only.
- Record successful numeric Codex results and a valid Grok weekly or explicit monthly result. Describe a monthly account as monthly, not live weekly coverage.
- Check normal cleanup, partial-error presentation with an isolated missing target override, absence of app-owned persistent state, and unaffected existing agent sessions.

**Deliverables:** Sanitized `docs/verification/live-acceptance.md`, updated compatibility/native/live matrix and evidence-linked requirement checkboxes.

**Acceptance criteria:**

- [x] Both requested Codex home queries work through the completed binary and their identity/quota attribution is correct.
- [x] The Grok path produces real valid billing metadata; UNSUPPORTED, TIMEOUT or a fixture replay cannot satisfy this gate.
- [x] Provider-display comparisons document period resets, concurrent usage and numeric precision rather than claiming exact simultaneity.
- [x] The evidence explicitly separates live-tested behavior, fixture-only weekly cases, cross-built targets and untested combinations.

**Verification:** [V-LIVE](../architecture/README.md#v-live), [V-CONTRACT](../architecture/README.md#v-contract), [V-SAFETY](../architecture/README.md#v-safety). Record the exact delivered-binary invocation and safe comparison steps. Run focused deterministic regressions after any discovered defect; do not automatically repeat the entire live campaign.

**Out of scope / stop conditions:** Do not run login/logout, open a new model turn to refresh statistics, alter real auth files to simulate failure, auto-upgrade the CLI, or convert a missing environment into a pass.

**Commit boundary:** `test(TASK-013): record real-account usage acceptance`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** [TASK-014](#task-014).

<a id="task-014"></a>
### TASK-014: Close requirements and prepare the v0.1 handoff

- [x] Task complete. **Status: COMPLETE.**

> Supersession note (2026-09-22): a post-closeout adversarial audit of commit `f8b180d` found executable-level correctness defects the recorded test gates did not catch (valid results discarded as timeouts, malformed RPC responses such as `error:null` accepted as success, exit-contract violations on broken stdout and signal combinations, relative executable paths unresolvable from the neutral child cwd, Grok authentication misclassification, label-normalization ordering, double cleanup budgets, and 14 files differing from `gofmt`), plus acceptance claims beyond the recorded evidence (REQ-048's provider-display comparison). The "all 50 requirements have valid evidence" claim recorded here is therefore superseded. Requirement re-qualification is tracked in [EPIC-007](#epic-007). The marker above reflects the historical commit, not current acceptance.

**Purpose:** Finish the project with complete traceability and truthful evidence, without turning closeout into another redesign cycle.

**Dependency:** TASK-013 complete and no blocking product defect remains.

**Architecture:** [ARCH-11](../architecture/README.md#arch-11), [ARCH-12](../architecture/README.md#arch-12), [ARCH-13](../architecture/README.md#arch-13), [ARCH-14](../architecture/README.md#arch-14).

**Primary requirement ownership:** [REQ-050](../specs/README.md#req-050).

**Implementation scope:**

- Audit every requirement against its implementation, tests, architecture reference and live evidence where required.
- Check all relative links/anchors and reconcile task/epic/requirement statuses. A skipped or blocked verification remains visible and prevents unsupported completion claims.
- Run the final offline regression/document checks once against the intended release tree and review only unresolved acceptance gaps.
- Update the three root documents and release summary with verified scope, limitations, local artifact locations and next maintenance guidance.
- Complete the one task-scoped closeout commit. Do not self-reference its unknown hash or create another commit merely to insert that hash.

**Deliverables:** Completed requirement/plan checklists, final verification/release summary and consistent architecture/README compatibility statements.

**Acceptance criteria:**

- [x] All 50 requirement checkboxes have valid evidence, including positive provider qualification rather than fallback-only tests.
- [x] All six epics satisfy their local gates, and no task remains BLOCKED or falsely marked complete.
- [x] Required deterministic, native-platform and live-account checks are accurately represented; known limitations are not hidden.
- [x] The handoff is complete with locally staged artifacts; no remote publication or next-version feature implementation occurred.

**Verification:** [V-AUDIT](../architecture/README.md#v-audit), [V-DOCS](../architecture/README.md#v-docs), [V-BUILD](../architecture/README.md#v-build). Run `go test ./...`, `go vet ./...`, `go run scripts/check-docs.go`, and audit the final evidence matrix. Reuse unchanged valid native/live evidence rather than rerunning it for documentation-only edits.

**Out of scope / stop conditions:** Do not reopen completed epics for cosmetic preferences, regenerate fixtures from a changed upstream without review, or repeat full reviews indefinitely.

**Commit boundary:** `docs(TASK-014): close v0.1 requirements and handoff`. Include only this task’s coherent changes, focused tests and evidence.

**Next:** Close EPIC-006 and stop. Return the final verified handoff; do not implement another provider automatically.

> Historical note: EPIC-006 was closed at commit `f8b180d` (TASK-014). The corrective audit superseded its final acceptance claims; see the supersession notes above and [EPIC-007](#epic-007).

<a id="epic-007"></a>
## EPIC-007: Corrective remediation and re-acceptance

- [x] Epic complete. **Status: COMPLETE** — every residual finding from the confirmation review of `c340245` is fixed with regression coverage (TASK-022..023), documentation is reconciled (TASK-024), and the final gate plus document-verified packaging is recorded (TASK-025). The epic's own scope is closed; v0.1 release acceptance remains pending REQ-048 alone. *Supersession note (2026-09-22): the confirmation review had found five residual defects, reopening the earlier completion claim; the historical TASK-015..021 work itself stands.*

**Goal:** Correct the executable-level defects found by the 2026-09-22 adversarial audit of commit `f8b180d`, restore the documented contracts with regression coverage, and make the recorded acceptance state truthful again. This is a bounded remediation of the existing v0.1 implementation, not a redesign: the product scope, architecture, and the original six epics' IDs and commits are preserved.

**Entry:** EPIC-001 through EPIC-006 implemented (historical commits preserved); corrective audit findings accepted.

**Task sequence:** [TASK-015](#task-015) -> [TASK-016](#task-016) -> [TASK-017](#task-017) -> [TASK-018](#task-018) -> [TASK-019](#task-019) -> [TASK-020](#task-020) -> [TASK-021](#task-021) -> [TASK-022](#task-022) -> [TASK-023](#task-023) -> [TASK-024](#task-024) -> [TASK-025](#task-025) -> [TASK-026](#task-026) -> [TASK-027](#task-027)

**Local epic acceptance:** Every audit finding is either fixed with focused regression coverage or explicitly recorded as an owner action (REQ-048's provider-display comparison). The affected requirement checkboxes are re-checked only against the corrected tree. A mandatory gate that cannot be run in this environment is reported as blocked, never waived.

**Handoff:** Stop implementation. Report corrected behavior with trustworthy evidence, the requirements still unchecked and why, and any owner action needed to complete REQ-048.

**Non-negotiable boundaries:** Keep Gaebokchi a small, one-shot, read-only Go CLI. No Claude support, watch mode, retries, daemons, persistence, credential parsing, custom OAuth, REST fallback, provider installation/login/logout, remote publication, or architectural rewrite. Normal tests remain offline with synthetic accounts and fake providers. Requirements are not weakened to make tests pass.

<a id="task-015"></a>
### TASK-015: Correct the acceptance-state baseline and register this remediation

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Make the recorded acceptance state truthful before any code is changed: uncheck the requirements whose acceptance scope the audit superseded, and register this corrective epic with the next available IDs.

**Dependency:** Corrective audit accepted; no code changes yet.

**Architecture:** [ARCH-14](../architecture/README.md#arch-14).

**Primary requirement ownership:** none closed; reopens affected checkboxes in [requirement.md](../specs/README.md).

**Implementation scope:**

- Change the overall completion claims in `requirement.md` and `plan.md` from COMPLETE to the corrective state.
- Uncheck exactly the requirements whose full acceptance scope is affected by verified audit findings (REQ-006, REQ-009, REQ-010, REQ-015, REQ-016, REQ-018, REQ-020, REQ-021, REQ-023, REQ-024, REQ-032, REQ-039, REQ-044, REQ-045, REQ-048, REQ-049, REQ-050), each with a supersession note naming the defect and the re-qualifying task. Leave unaffected, adequately supported requirements checked.
- Add supersession notes to TASK-013/TASK-014 (preserving their historical records and commit identities) and reconcile the roadmap overview with the current state.
- Register EPIC-007 and TASK-015 through TASK-021 with this scope.

**Acceptance criteria:**

- [x] No requirement remains checked whose full acceptance scope is superseded by a verified audit finding; every unchecked box names its defect and re-qualifying task.
- [x] Historical records, dates, and artifact identities are preserved; corrections appear as explicit supersession notes, not rewrites.
- [x] The roadmap and document headers state the corrective state without resetting the six original epics' historical milestones.

**Verification:** [V-AUDIT](../architecture/README.md#v-audit). Run `go run scripts/check-docs.go` after the edits.

**Commit boundary:** `docs(TASK-015): correct acceptance baseline and register remediation`.

**Next:** [TASK-016](#task-016).

<a id="task-016"></a>
### TASK-016: Fix query deadlines, transport cancellation, and owned-process cleanup

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** A valid result established within the query budget must survive bounded cleanup; cancellation and terminal failure must unblock owned I/O exactly once within one cleanup budget.

**Dependency:** TASK-015 complete.

**Architecture:** [ARCH-05](../architecture/README.md#arch-05), [ARCH-06](../architecture/README.md#arch-06).

**Primary requirement ownership:** re-qualifies [REQ-009](../specs/README.md#req-009), [REQ-020](../specs/README.md#req-020), [REQ-023](../specs/README.md#req-023).

**Implementation scope:**

- Separate the point at which a query result becomes valid from the completion of deferred cleanup in the provider `Query` paths and the runner's post-hoc deadline check. A result established within the query budget survives subsequent bounded cleanup; a response first established after the query deadline must not become a clean result.
- Make notification writes cancellable (`Notify` currently writes synchronously without a context) so a blocked write or write lock cannot defeat the target deadline or signal cancellation; reconcile `die`/`Close`/`sendRejection` so terminal failure, explicit closure, and cancellation abort owned I/O safely and exactly once, without reentrant-close deadlocks, double waits, or abandoned writer goroutines.
- Pass one absolute cleanup deadline through `shutdown` and `finish` instead of granting `finish` a fresh full budget; audit pipe closure, drain completion, and temporary-directory removal so incomplete cleanup is not hidden behind an unconditional success result.
- Investigate the descendant/shared-process boundary: where a descendant is proven application-owned and disposable, clean it up; where ownership is provider-managed or uncertain, preserve it and document the boundary. No indiscriminate descendant or process-group killing.

**Acceptance criteria:**

- [x] A valid response before the deadline remains numeric after slower cleanup; a response arriving only after the deadline does not become successful; startup, handshake, account/authentication, and quota reads share one query budget while cleanup uses its separate budget. *(Verified by the confirmation review and the TASK-016 regressions.)*
- [x] Cancellation unblocks a blocked notification write and a blocked callback-rejection write; repeated closure is safe; the direct child is reaped exactly once; unrelated processes and provider-managed leaders survive. *(Follow-up TASK-022: leader/descendant survival is now guaranteed by the direct-child-only policy rather than by the group-kill path this item originally rode on.)*
- [x] One total cleanup budget is enforced and measured end-to-end (not inferred from stage timers). *(TestCloseHonorsSingleAbsoluteBudgetEndToEnd.)*

**Verification:** [V-LIFECYCLE](../architecture/README.md#v-lifecycle), [V-RUNNER](../architecture/README.md#v-runner). Focused race-enabled tests using synchronization and timing margins, not exact scheduler timing.

**Commit boundary:** `fix(TASK-016): separate query deadlines from cleanup and harden transport cancellation`.

**Next:** [TASK-017](#task-017).

<a id="task-017"></a>
### TASK-017: Fix executable-path resolution and CLI output/signal behavior

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Explicit relative executable paths must work from the neutral child cwd, and the executable must implement the documented output and signal exit contract.

**Dependency:** TASK-016 complete.

**Architecture:** [ARCH-03](../architecture/README.md#arch-03), [ARCH-10](../architecture/README.md#arch-10).

**Primary requirement ownership:** re-qualifies [REQ-006](../specs/README.md#req-006) (binary-override scope), [REQ-024](../specs/README.md#req-024), [REQ-039](../specs/README.md#req-039), [REQ-045](../specs/README.md#req-045).

**Implementation scope:**

- Resolve explicit relative executable paths against the invocation environment (make them absolute) before the child's working directory changes; retain Go's unsafe-current-directory PATH-lookup protection and once-per-invocation resolution; no shell expansion or command splitting.
- Implement an explicit SIGPIPE policy so a closed stdout pipe yields the documented output-failure exit code 1 rather than signal death; centralize final exit-code precedence so the first handled cancellation signal (130/143) takes precedence over output failure and partial provider failure.
- On writable stdout, retain completed results and render unresolved rows as `CANCELLED`; never launch queued targets after cancellation; release signal-related goroutines when execution finishes; handle output failures consistently on help/version paths.

**Acceptance criteria:**

- [x] Absolute and explicit relative binary overrides (including paths with spaces) resolve correctly despite the neutral child cwd; missing and present-but-unexecutable binaries keep their documented statuses. *(Verified by the confirmation review.)*
- [x] Closed stdout pipe, SIGINT and SIGTERM separately, and each signal combined with unwritable stdout produce the exact documented exit codes with real OS pipes against the actual executable; correct rows, no late launch, and safe diagnostics without raw provider output. *(Verified by the confirmation review.)*

**Verification:** [V-EXIT](../architecture/README.md#v-exit), [V-E2E](../architecture/README.md#v-e2e). Executable-level tests with real OS pipes, not `io.Writer` stubs.

**Commit boundary:** `fix(TASK-017): resolve relative executable paths and enforce output signal exit contract`.

**Next:** [TASK-018](#task-018).

<a id="task-018"></a>
### TASK-018: Harden inbound RPC validation

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Malformed or incompatible inbound responses must fail safely; only a structurally valid response alternative may satisfy a pending call.

**Dependency:** TASK-017 complete.

**Architecture:** [ARCH-06](../architecture/README.md#arch-06), [ARCH-11](../architecture/README.md#arch-11).

**Primary requirement ownership:** re-qualifies [REQ-016](../specs/README.md#req-016), [REQ-018](../specs/README.md#req-018).

**Implementation scope:**

- Require exactly one valid response alternative: `result` (with explicit null permitted only where the operation's schema allows it) or a structurally valid `error` object with correctly typed required fields. `error:null`, incomplete error objects, and missing results must not satisfy a pending call.
- Validate the Codex initialization response and the Grok authenticate response against their dialects; validate the ACP envelope (including inbound jsonrpc version markers) without imposing ACP-required fields on Codex's distinct dialect.
- Preserve forward compatibility for unknown additive fields and well-formed unrelated notifications; keep unknown/duplicate response IDs from satisfying pending calls (already safe — pin with tests).

**Acceptance criteria:**

- [x] Replacing an initialization or authentication response with `error:null`, a both-members or neither-member object, or a mistyped error object yields a safe failure status, never numeric success or unsupported authentication claims. *(Verified by the confirmation review.)*
- [x] Invalid/incompatible ACP version markers and dialect-inappropriate envelopes are rejected; valid operation-specific null results, additive fields, and unrelated notifications still work; invalid UTF-8, oversize frames, and non-empty unterminated frames keep their terminal handling. *(Verified by the confirmation review.)*

**Verification:** [V-RPC](../architecture/README.md#v-rpc). Framing/dialect matrix extension with malformed-error and envelope cases.

**Commit boundary:** `fix(TASK-018): reject malformed RPC responses and incompatible envelopes`.

**Next:** [TASK-019](#task-019).

<a id="task-019"></a>
### TASK-019: Fix identity handling, Grok authentication classification, and provenance

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Account labels must be normalized before validation and disambiguation, and Grok authentication/provenance must be classified from verified structure rather than guesses.

**Dependency:** TASK-018 complete.

**Architecture:** [ARCH-04](../architecture/README.md#arch-04), [ARCH-08](../architecture/README.md#arch-08), [ARCH-10](../architecture/README.md#arch-10).

**Primary requirement ownership:** re-qualifies [REQ-006](../specs/README.md#req-006) (alias-validation scope), [REQ-010](../specs/README.md#req-010), [REQ-015](../specs/README.md#req-015), [REQ-021](../specs/README.md#req-021), [REQ-032](../specs/README.md#req-032).

**Implementation scope:**

- Establish one display-label normalization policy applied before validity checks and duplicate disambiguation: a Grok alias that is empty after normalization is a usage error before provider launch; an unusable Codex protocol email falls back to the configured home basename; disambiguation operates on final display identities, produces unique visible labels without merging rows, and handles collisions with labels that already resemble generated suffixes.
- Implement the Grok authentication advertisement decision table: advertised cached-token support wins; a genuinely API-key-only advertisement can be non-applicable; a recognized interactive subscription path without cached authentication requires authentication; unknown combinations are not declared clean without a qualified basis; never initiate interactive authentication as a fallback.
- Map only verified structured authentication failures to `AUTH REQUIRED`; use safe generic provider/protocol/unsupported classifications for other remote errors; never infer expired credentials from arbitrary error text or the RPC method name alone.
- Preserve established period information and applicable source/reset/quota provenance in the in-memory result (the monthly branch currently drops `PeriodMonthly`); never invent an upstream quota identifier.

**Acceptance criteria:**

- [x] Blank/whitespace-only aliases and emails, control/format-only labels, Unicode whitespace, sanitization collisions, duplicate emails/basenames, and generated-suffix collisions behave per the normalization policy with no extra rows/columns and no provider launch for invalid aliases. *(Follow-up TASK-023: the Unicode-whitespace clause is satisfied by the complete predicate; ASCII-only cases were already verified by the confirmation review.)*
- [x] Every branch of the authentication decision table, non-authentication remote errors, monthly provenance, and weekly periods with missing usage are covered by tests; dependency direction stays clean (domain code never imports the renderer). *(Verified by the confirmation review.)*

**Verification:** [V-IDENTITY](../architecture/README.md#v-identity), [V-GROK](../architecture/README.md#v-grok), [V-SAFETY](../architecture/README.md#v-safety).

**Commit boundary:** `fix(TASK-019): normalize account labels and correct Grok auth classification`.

**Next:** [TASK-020](#task-020).

<a id="task-020"></a>
### TASK-020: Fix test hygiene, formatting, and documentation inconsistencies

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Test-owned cleanup must not delete unrelated directories, formatting must be verifiable rather than assumed, and the documentation must state exactly what was verified.

**Dependency:** TASK-019 complete.

**Architecture:** [ARCH-11](../architecture/README.md#arch-11), [ARCH-13](../architecture/README.md#arch-13), [ARCH-14](../architecture/README.md#arch-14).

**Primary requirement ownership:** re-qualifies [REQ-049](../specs/README.md#req-049).

**Implementation scope:**

- Track and remove only directories created by the current integration-test invocation; add a regression proving a similarly named unrelated directory survives.
- Apply `gofmt` to all Go files (formatting only, no style refactoring); make the verification workflow check formatting rather than assume it.
- Correct documentation inconsistencies: roadmap/task/epic status contradictions, obsolete fixture filenames in compatibility.md, README wording that conflates Linux fake-provider tests with live provider qualification, stale COMPLETE/no-outstanding-findings claims, and evidence links/requirement mappings affected by the fixes. Distinguish cross-build, native offline, real-provider acceptance, and provider-display comparison evidence.

**Acceptance criteria:**

- [x] Integration cleanup removes only owned directories; the unrelated-directory survival regression exists and passes.
- [x] `gofmt -l` reports nothing and the workflow fails on formatting drift. *(The CI gofmt gate added in TASK-020.)*
- [x] Documentation claims are semantically consistent with the recorded evidence and distinguish the four evidence kinds at TASK-020's tree state. *(Follow-up TASK-024: the confirmation review found later drift — roadmap/acceptance/packaging inconsistency at `c340245` — reconciled here and re-packaged in TASK-025.)*

**Verification:** [V-DOCS](../architecture/README.md#v-docs), [V-E2E](../architecture/README.md#v-e2e). `gofmt -l`, `go run scripts/check-docs.go`, integration suite.

**Commit boundary:** `test(TASK-020): fix test hygiene formatting and documentation consistency`.

**Next:** [TASK-021](#task-021).

<a id="task-021"></a>
### TASK-021: Perform final qualification and record the release decision truthfully

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Qualify the corrected candidate tree end-to-end and record a truthful release decision, keeping code-remediation completion distinct from full release acceptance.

**Dependency:** TASK-020 complete.

**Architecture:** [ARCH-11](../architecture/README.md#arch-11), [ARCH-13](../architecture/README.md#arch-13), [ARCH-14](../architecture/README.md#arch-14).

**Primary requirement ownership:** re-qualifies [REQ-044](../specs/README.md#req-044), [REQ-050](../specs/README.md#req-050); records the state of [REQ-048](../specs/README.md#req-048).

**Implementation scope:**

- Re-verify the full acceptance scope of every requirement unchecked in TASK-015 against the corrected tree; re-check only those whose scope is now satisfied with current evidence.
- Run the final integration gate: formatting check, `go vet ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./...` where supported, `go run scripts/check-docs.go`, four CGO-disabled target builds, archive contents/version output/SHA-256 checksums, and native macOS/Linux offline qualification where available.
- Run explicit authorized live acceptance where available: compare each relevant provider's own quota display with the candidate binary in the same quota period without submitting prompts or changing account/billing state, recording timestamps, period boundaries, and legitimate precision/consumption differences. When authorization, display access, or the required environment is unavailable, mark the remaining checks NOT RUN/blocked and report the acceptance gap without waiving the criterion on the owner's behalf.
- Record exact commands, outcomes, OS/architecture, toolchain, and source/artifact identity. Historical Linux success or earlier live acceptance of the old binary is not evidence for the corrected candidate; distinguish artifacts by checksum and source revision.

**Acceptance criteria:**

- [x] Every re-checked requirement has current, tree-matching evidence; requirements still unchecked carry exact reasons and owner actions. *(Follow-up TASK-022/023/024: six of those re-checks were themselves reopened by the confirmation review and are re-qualified against the current tree; REQ-048 remains the only unchecked requirement.)*
- [x] The final gate results distinguish run, not-run, and blocked checks; v0.1 is not declared fully accepted while a mandatory gate remains open. *(Follow-up: the TASK-021 gate ran against a tree whose packaged documents were stale — re-run with document-freshness verification in TASK-025.)*

**Verification:** [V-AUDIT](../architecture/README.md#v-audit), [V-BUILD](../architecture/README.md#v-build), [V-PLATFORM](../architecture/README.md#v-platform), [V-LIVE](../architecture/README.md#v-live).

**Commit boundary:** `test(TASK-021): record final qualification and release decision`.


<a id="task-022"></a>
### TASK-022: Reopen corrective state, propagate cleanup failures, and own only the direct child

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Execute the confirmation review's first two residual findings: make lifecycle cleanup failures visible to the CLI without invalidating measured quotas, and make forced termination depend on proven ownership (direct-child only).

**Dependency:** TASK-021's record superseded by the confirmation review; no unrelated work reopened.

**Architecture:** [ARCH-05](../architecture/README.md#arch-05) (ownership and cleanup-failure policy rewritten accordingly).

**Primary requirement ownership:** re-qualifies [REQ-020](../specs/README.md#req-020), [REQ-040](../specs/README.md#req-040); reopens the acceptance baseline for [REQ-006](../specs/README.md#req-006), [REQ-010](../specs/README.md#req-010), [REQ-012](../specs/README.md#req-012), [REQ-050](../specs/README.md#req-050) pending TASK-023/TASK-024.

**Implementation scope:**

- Reopen the corrective-completion claims and register TASK-022 through TASK-025.
- Stage-3 cleanup kills the direct child only; the private group is never signalled (Setpgid stays for signal isolation). killGroup removed.
- Cleanup outcomes propagate: provider sessions return Close's error through a deferred capture (idempotent, covers the RPC-abort path); `domain.Result.CleanupFailed` carries the outcome; the renderer emits one bounded safe diagnostic for numeric rows; the CLI exits 1; signal codes 130/143 keep precedence.
- `process.Config.TempParent` is a narrow documented test seam for deterministic cleanup-failure injection.

**Acceptance criteria:**

- [x] A numeric quota plus failed cleanup keeps the percentage, emits exactly one safe diagnostic, and exits 1; cleanup failure in one target leaves healthy siblings untouched; signal codes outrank cleanup failure; repeated closure keeps one outcome; RPC-abort-initiated cleanup failures reach the result; a real permission-based removal failure is reproduced where the environment permits (test-owned directories only, permissions restored). *(TASK-022 evidence record.)*

**Verification:** [V-LIFECYCLE](../architecture/README.md#v-lifecycle), [V-EXIT](../architecture/README.md#v-exit), [V-E2E](../architecture/README.md#v-e2e).

**Commit boundary:** `fix(TASK-022): propagate cleanup failures and own only direct-child termination`.

**Next:** [TASK-023](#task-023).

<a id="task-023"></a>
### TASK-023: Complete Unicode label normalization and the target-index suffix rule

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Execute the confirmation review's findings 3 and 4: whitespace-only labels built from Unicode spaces (U+3000, U+205F, U+1680, ...) must normalize to empty and be rejected or fall back; duplicate-account suffixes must follow the documented `#<target index + 1>` rule.

**Architecture:** [ARCH-04](../architecture/README.md#arch-04).

**Primary requirement ownership:** re-qualifies [REQ-006](../specs/README.md#req-006), [REQ-010](../specs/README.md#req-010), [REQ-012](../specs/README.md#req-012).

**Implementation scope:**

- `domain.NormalizeLabel` uses a complete Unicode whitespace predicate (`unicode.IsSpace`) together with the existing control/format/replacement policy; ordinary printable Unicode (including Korean/CJK) is preserved.
- Disambiguation suffixes are based on the target's configured index (`#<index+1>`); a narrow documented deterministic extension (`#<index+1>-2`, ...) covers collisions with reserved lookalike labels while preserving the index component.

**Acceptance criteria:**

- [x] U+3000/U+205F/U+1680-only aliases exit 2 before provider launch; a U+3000-only email falls back to its home basename; the `same/other/same` example yields `#3`; labels are deterministic across repeated invocations and completion orders; existing sanitization-collision and lookalike protections still hold. *(TASK-023 evidence record.)*

**Verification:** [V-OPTIONS](../architecture/README.md#v-options), [V-IDENTITY](../architecture/README.md#v-identity).

**Commit boundary:** `fix(TASK-023): complete unicode label normalization and index suffix rule`.

**Next:** [TASK-024](#task-024).

<a id="task-024"></a>
### TASK-024: Reconcile current documentation and acceptance state

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Execute the confirmation review's fifth finding: make headers, roadmap rows, epic/task statuses, requirement checkboxes, acceptance checklists, and verification records agree, without bulk-checking or rewriting history.

**Architecture:** [ARCH-14](../architecture/README.md#arch-14).

**Primary requirement ownership:** re-qualifies [REQ-050](../specs/README.md#req-050); closes the reopened label/cleanup requirements with evidence.

**Implementation scope:**

- Roadmap rows and epic/task statuses reconciled with the current state; each of the previously unchecked acceptance items checked only with matching evidence or left unchecked with the exact remaining issue; TASK-016/TASK-019/TASK-021 records gain explicit follow-up notes; requirement.md reflects the final corrective state.

**Acceptance criteria:**

- [x] Document navigation is consistent; no acceptance checkbox is checked without matching evidence; the final state distinguishes corrective code completion from the pending REQ-048 live gate (candidate live test and provider-display comparison). *(TASK-024 evidence record.)*

**Verification:** [V-AUDIT](../architecture/README.md#v-audit), [V-DOCS](../architecture/README.md#v-docs). `go run scripts/check-docs.go`.

**Commit boundary:** `docs(TASK-024): reconcile corrective state and acceptance checklists`.

**Next:** [TASK-025](#task-025).

<a id="task-025"></a>
### TASK-025: Final verification and candidate packaging

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Re-qualify the corrected tree end to end and package the candidate with document freshness verified (the previous archives were staged before the final document edits).

**Architecture:** [ARCH-13](../architecture/README.md#arch-13), [ARCH-14](../architecture/README.md#arch-14).

**Primary requirement ownership:** evidence for all re-qualified requirements; artifact records.

**Implementation scope:**

- Full gate (gofmt/vet/test/race/docs) on darwin/arm64 and linux/arm64 (containerized); four CGO-disabled builds.
- Packaging sequence: finalize documents, build, inspect all four archives, byte-compare every packaged document with the intended source, verify version output and checksums, record source identity. No document edits after staging without rebuilding.

**Acceptance criteria:**

- [x] Gate green on available platforms with NOT RUN items marked; packaged documents byte-identical to the working tree; checksums recorded; the handoff distinguishes code completion from the open REQ-048 gate. *(TASK-025 evidence record.)*

**Verification:** [V-BUILD](../architecture/README.md#v-build), [V-PLATFORM](../architecture/README.md#v-platform), [V-DOCS](../architecture/README.md#v-docs).

**Commit boundary:** `test(TASK-025): requalify candidate and repackage archives`.

**Next:** [TASK-026](#task-026).

<a id="task-026"></a>
### TASK-026: Final documentation and archive cleanup

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Two narrow housekeeping findings from the confirmation review of `75e16f5`: a stale progress sentence in this plan, and unintended AppleDouble `._*` sidecar members in the release archives (hidden from the macOS `tar -tzf` listing used by TASK-025, present in the raw member lists). The accepted runtime work is out of scope and untouched.

**Dependency:** TASK-025 complete; no other work reopened.

**Architecture:** [ARCH-13](../architecture/README.md#arch-13).

**Primary requirement ownership:** packaging evidence only; no requirement state changes (REQ-048 stays open).

**Implementation scope:**

- Correct the stale roadmap sentence (original milestones complete; corrective runtime work complete; full release acceptance pending REQ-048 — a live gate, not unfinished implementation).
- `scripts/build-release.sh` sets `COPYFILE_DISABLE=1` for the archive step: the macOS-bsdtar-supported way to suppress AppleDouble sidecars, inert under GNU tar on Linux.
- `scripts/archivecheck.go` + tests: a standard-library raw-member gate (exact intended member set, no sidecars, executable bit, byte-identical documents, checksum manifest), validated against synthetic negative fixtures and wired into CI after the release build.

**Acceptance criteria:**

- [x] The corrected script produces archives whose only regular members are the five intended files per target; the raw-member gate passes for all four targets and demonstrably rejects sidecar/missing/unexpected/non-executable/stale-document fixtures; all 16 packaged documents remain byte-identical to the finalized sources; checksums identify the replacement candidate; the script is exercised on macOS and Linux with unavailable environments marked NOT RUN. *(TASK-026 evidence record.)*

**Commit boundary:** `build(TASK-026): strip archive sidecars and verify raw members`.

**Next:** [TASK-027](#task-027).

<a id="task-027"></a>
### TASK-027: Close the remaining archive-validation gaps

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Three narrow findings from the confirmation review of `7b5dbcc`: the archive checker accepted unexpected directories and duplicate members; the staged-release test skipped on any incomplete staging instead of failing; and the roadmap overview omitted TASK-026. No runtime change.

**Dependency:** TASK-026 complete; no other work reopened.

**Architecture:** [ARCH-13](../architecture/README.md#arch-13).

**Primary requirement ownership:** packaging evidence only; REQ-048 stays open.

**Implementation scope:**

- verifyArchive enforces the exact filesystem-member set: allowlisted directory entries as well as files, duplicate rejection for both, expected-type validation per path, exactly one occurrence of each intended entry.
- The staged-release pass/fail/skip decision (checkStagedRelease) skips only when the staging directory genuinely does not exist in optional mode; empty, partial, missing-manifest, non-directory, and inspection-failure states fail in both modes. GAEBOKCHI_REQUIRE_STAGED_RELEASE=1 makes even absent staging fail and is what the CI release job runs.
- The roadmap overview, task sequence, and current-state text include TASK-026 and this task.

**Acceptance criteria:**

- [x] The strict member-set regressions (unexpected directory, duplicate README/binary/directory, wrong entry type) fail the checker; the staging-decision and mandatory-command-exit tests pass in both modes; the CI release job runs the checker in mandatory mode; focused checker tests pass on macOS and Linux; the rebuilt candidate passes the strict staged check with 16/16 byte-identical documents and verified checksums. *(TASK-027 evidence record.)*

**Commit boundary:** `test(TASK-027): enforce exact archive members and mandatory staging validation`.

**Next:** Stop. Runtime corrections remain accepted; archive validation and the roadmap follow-up are complete; full v0.1 release acceptance remains pending REQ-048.

<a id="epic-008"></a>
## EPIC-008: Version command and release identity

- [x] Epic complete. **Status: COMPLETE.**

**Goal:** Replace the legacy version flag with a project-independent Mulgae-style
version command, a minimal machine-readable projection, and one canonical
v-prefixed release identity without changing quota behavior.

**Entry:** EPIC-007 complete; owner approved the breaking removal of
`--version`, retention of product version v0.1.0, and v-prefixed staging paths.

**Task sequence:** [TASK-028](#task-028).

**Local epic acceptance:** TASK-028 is committed with focused CLI/build tests,
the full offline gate, and document-fresh release artifacts. REQ-048 remains an
independent live gate and is neither run nor waived by this epic.

<a id="task-028"></a>
### TASK-028: Implement the stable version command and release identity

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Align Gaebokchi's public version surface and build metadata flow
with the approved Mulgae-style contract while keeping public JSON minimal.

**Dependency:** TASK-027 complete.

**Architecture:** [ARCH-03](../architecture/README.md#arch-03),
[ARCH-13](../architecture/README.md#arch-13),
[ARCH-14](../architecture/README.md#arch-14).

**Primary requirement ownership:** [REQ-051](../specs/README.md#req-051).

**Implementation scope:**

- Dispatch `version` and `version --json` before general option, HOME, cwd, or
  provider handling; reject the legacy `--version` form and all unsupported
  version arguments.
- Emit `gaebokchi v0.1.0` for release builds and exactly `name` plus `version`
  in JSON. Resolve linker override before Go build information and use
  `(devel)` when neither supplies a release version.
- Make the release linker value and staging directory v-prefixed, update CI and
  archive validation, and synchronize the public README and canonical contract.
- Preserve historical evidence records and the unrelated REQ-048 state. Do not
  tag, publish, install, or run live provider queries.

**Acceptance criteria:**

- [x] Exact human and JSON outputs, development fallback, metadata precedence,
  invalid syntax, removed `--version`, output failure, and no-provider behavior
  have focused regression coverage. *(Verified on the TASK-028 working tree.)*
- [x] Formatting, vet, full offline tests, race tests, documentation validation,
  four target builds, archive membership/document freshness, checksums, and a
  native version smoke pass on the finalized tree. *(Verified on darwin/arm64;
  the other three artifacts are cross-build evidence.)*
- [x] The task-scoped commit is identified by its unique commit subject and
  [TASK-028 evidence](../verification/TASK-028.md) records exact outcomes and
  unavailable environments.

**Commit boundary:** `feat(TASK-028): align version command and release identity`.

**Next:** Stop. EPIC-008 is closed. REQ-048 remains open unless a separate
owner-authorized live campaign is performed.

<a id="epic-009"></a>
## EPIC-009: Machine-readable usage snapshot

- [x] Epic complete. **Status: COMPLETE.**

**Goal:** Add an opt-in JSON projection of the current usage snapshot for
scripts without changing provider queries or the default table.

**Entry:** EPIC-008 complete. This extension is separate from the historical
v0.1 release gate and from REQ-048's owner-authorized live acceptance.

**Task sequence:** [TASK-029](#task-029).

**Local epic acceptance:** The task-scoped commit contains the implementation,
offline regression evidence, and synchronized public and canonical contracts.

<a id="task-029"></a>
### TASK-029: Add JSON usage output

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Give callers one stable machine-readable snapshot while preserving
the current query, diagnostic, and exit behavior.

**Architecture:** [ARCH-03](../architecture/README.md#arch-03),
[ARCH-10](../architecture/README.md#arch-10).

**Primary requirement ownership:** [REQ-052](../specs/README.md#req-052).

**Implementation scope:**

- Parse a single boolean `--json` for usage queries and render the ordered
  result set through an explicit six-field JSON projection.
- Keep the default table, text help, `version --json`, stderr diagnostics, and
  exit-code precedence. Keep provider internals and credentials out of JSON.
- Document the new command and output contract in README, specifications,
  architecture, and help. Leave historical evidence unchanged.

**Acceptance criteria:**

- [x] Exact schema, order, numeric zero and precision, missing-value null,
  partial results, cleanup failure, account normalization, and private-field
  exclusion pass focused offline tests.
- [x] Invalid and duplicate options, output failure, default table,
  help/version, and exit behavior pass regression checks.
- [x] Required offline checks and documentation link audit pass, with results
  recorded in [TASK-029 evidence](../verification/TASK-029.md).
- [x] The task-scoped commit records the implementation, evidence, and closure
  of TASK-029, EPIC-009, and REQ-052.

**Commit boundary:** `feat(TASK-029): add JSON usage output`. No release,
installation, publication, or live provider query is implied.

<a id="epic-010"></a>
## EPIC-010: Interactive terminal view

- [x] Epic complete. **Status: COMPLETE.**

**Goal:** When a person runs the usage command in an interactive terminal,
show every target's progress while queries run and finish with a glanceable
frame of usage bars and reset countdowns. Piped, redirected, and JSON output
stay byte-for-byte unchanged.

**Entry:** EPIC-009 complete. On 2026-09-25 the owner approved an inline live
view instead of a full-screen, alternate-screen, or watch-mode interface; one
dashboard row per target; and a reset countdown as the only added datum.
Inline failure hints and pace markers were considered and not adopted. This
extension is separate from the historical v0.1 release gate and from REQ-048.

**Task sequence:** [TASK-030](#task-030).

**Local epic acceptance:** The task-scoped commit contains the implementation,
offline regression evidence, a recorded real-terminal check, and synchronized
public and canonical contracts.

<a id="task-030"></a>
### TASK-030: Add the live terminal usage view

- [x] Task complete. **Status: COMPLETE.**

**Purpose:** Replace the silent wait and plain table on an interactive
terminal with the approved live view, while scripts keep receiving the current
table and JSON.

**Dependency:** TASK-029 complete.

**Architecture:** [ARCH-05](../architecture/README.md#arch-05),
[ARCH-10](../architecture/README.md#arch-10-terminal).

**Primary requirement ownership:** [REQ-053](../specs/README.md#req-053),
[REQ-054](../specs/README.md#req-054), [REQ-055](../specs/README.md#req-055).

**Implementation scope:**

- Implement the [interactive terminal view](../architecture/README.md#arch-10-terminal):
  the selection function and a `//go:build unix` window-size file in
  `cmd/gaebokchi`, frame and live renderers in `internal/render` (suggested
  `terminal.go` and `live.go`), and the optional observation callback on
  `app.Runner`. The illustrative frame is in the
  [requirements](../specs/README.md#current-terminal-view-extension).
- Write the ANSI output directly; add no TUI framework and no flag. Promote only
  `golang.org/x/sys` and `golang.org/x/text` to direct requirements, and keep
  the offline `GOPROXY=off` gate working.
- Keep the plain table, JSON, help commands and flags, `version`, stderr
  diagnostics, exit-code precedence, provider queries, and deadlines unchanged.
  Update help usage wording to describe the terminal view.
- On completion, turn the planned text into current-state documentation:
  convert both "Planned terminal view extension" sections into current
  extensions; fold the ARCH-10 planned subsection into ARCH-02 (layout and
  dependency rationale), ARCH-05 (observation hook), ARCH-10, and V-OUTPUT;
  add dated notes under REQ-011, REQ-012, and REQ-015 recording that the scope
  change took effect; and update the README example, usage, and privacy
  wording, the help text, and roadmap text that describes default stdout.
  Leave historical evidence unchanged.

**Acceptance criteria:**

- [x] Colorless golden frames at a fixed width and time cover numeric rows with
  and without a countdown, provider grouping, sanitized control characters,
  and an East Asian wide alias. Colorless and styled frames cover every status
  class; styled frames cover the 49.9/50/79.9/80 color boundaries. Every emitted
  escape is on the ARCH-10 allowlist.
- [x] Bar eighths (0, 0.1, 37, 99.99, 100), countdown formatting, width tiers,
  intermediate clipping, untruncated final labels, and the height fallback pass
  focused tests.
- [x] A test-local screen model shows that successful live output ends with
  exactly the final frame on screen and the cursor visible, including concurrent
  observations under the race detector and cancellation.
- [x] With a failing writer, the first write error stops further drawing and
  follows the existing output-failure exit policy; no final frame or cursor
  restoration is required after that error.
- [x] Runner observation fires exactly once per target with the stored result,
  including targets cancelled while queued.
- [x] With `TERM=xterm-256color`, piped stdout is byte-identical to the current
  table; the existing table, JSON, help, version, diagnostic, and exit tests
  pass unchanged.
- [x] A real-terminal check, for example tmux driving the fake providers,
  records progress and the final frame at 100, 60, and 40 columns, with
  `NO_COLOR=1`, `TERM=dumb`, a pipe, and Ctrl-C (exit 130, `CANCELLED` rows,
  visible cursor).
- [x] `make test` and the documentation audit pass on the final tree, with
  exact results and any unrun checks recorded in `docs/verification/TASK-030.md`.

**Verification:** [V-OUTPUT](../architecture/README.md#v-output),
[V-RUNNER](../architecture/README.md#v-runner),
[V-EXIT](../architecture/README.md#v-exit),
[V-E2E](../architecture/README.md#v-e2e).

**Commit boundary:** `feat(TASK-030): add live terminal usage view`. No
release, installation, publication, or live provider query is implied.

**Next:** Stop. EPIC-010 closes with TASK-030. REQ-048 remains open unless a
separate owner-authorized live campaign is performed.

<a id="plan-release-gate"></a>
## Historical pre-extension release gate

These checks describe the pre-Claude, pre-YAML v0.1 candidate. They are not
the current publication gate or extra implementation tasks.
The checks below were marked passed at commit `f8b180d` (TASK-014); the
corrective audit superseded several of them, and they are re-owned by
[EPIC-007](#epic-007) / [TASK-021](#task-021). EPIC-009 passed its own local gate:

- [ ] Every v0.1 checkbox in [the specifications](../specs/README.md) has evidence, with live and fixture-only claims distinguished. *(REQ-048 remains pending the owner-authorized live campaign; the later REQ-052 was completed by EPIC-009.)*
- [x] The v0.1 through EPIC-009 tasks completed their stated local acceptance criteria in sequence. *(TASK-029 closed EPIC-009 with offline evidence and its task-scoped commit subject.)*
- [x] Non-interactive, non-JSON output remains the three-column table, including on partial provider failure. *(Held: the table shape was not invalidated by the audit; EPIC-010 adds a separate interactive view.)*
- [ ] Codex works through both specified homes; Grok produces valid real billing metadata through the qualified prompt-free CLI protocol. *(The recorded live evidence belongs to the pre-remediation artifact at commit f8b180d; the final corrected candidate (TASK-025 archives) has not been live-tested because owner authorization was unavailable. Offline qualification of the corrected tree is complete on darwin/arm64 and linux/arm64.)*
- [x] Offline tests, native macOS/Linux tests, four target builds, document-link checks and owner-account acceptance are recorded truthfully. *(Corrected tree: darwin/arm64 full suite + race green; linux/arm64 (containerized) vet + suite + race subset + docs check green; four CGO-disabled builds with verified checksums; live acceptance recorded as NOT RUN for the corrected candidate — see [TASK-021](../verification/TASK-021.md).)*
- [x] No Claude/PTY, private REST fallback, credential parsing, agent prompt/session, app-owned daemon or persistent usage store had been introduced at this historical candidate. *(Claude support was added later; this check is not current acceptance.)*
- [x] Local release artifacts and README are ready, with tested-version/platform limitations and no unauthorized publication. *(Artifacts rebuilt from the corrected tree with new SHA-256 checksums; README separates live qualification from offline Linux evidence.)*

A missing live environment is a real qualification gap, not a reason to invent evidence. Stop with a precise incomplete-gate report instead of asking an implementation AI to loop until an unavailable service becomes available.

<a id="epic-011"></a>
## EPIC-011: Public v0.1.0 release readiness

- [ ] Epic open. **Status: IN PROGRESS.**

**Goal:** Reconcile public documentation and package contents, accept the current
YAML and Claude contracts, and qualify the final v0.1.0 candidate. Earlier
epic and task completion remains historical; publication requires the current
release gate below.

**Tasks:** [TASK-031](#task-031) -> [TASK-032](#task-032) -> [TASK-033](#task-033).

<a id="task-031"></a>
### TASK-031: Align release documentation and archive contents

- [x] Task complete. **Status: COMPLETE.**

**Scope:** Make the public README, documentation indexes, roadmap current-state
summary, ARCH-13, release procedure, and archive member checker agree on the
current CLI and the files distributed in v0.1.0. Include the project LICENSE
and bundled-component license notices in every archive. Preserve historical
verification records.

**Acceptance:** Documentation links and archive fixtures pass; a newly staged
candidate contains byte-identical public documents and license texts with no extra
members. Record the exact tested tree and any platform limits in
`docs/verification/TASK-031.md`.

**Commit boundary:** `build(TASK-031): align public release documentation and archives`.

<a id="task-032"></a>
### TASK-032: Accept the current YAML and Claude contracts

- [x] Task complete. **Status: COMPLETE.**

**Scope:** Re-verify REQ-002, REQ-006 through REQ-008, REQ-010 through REQ-011,
REQ-039, and REQ-056 through REQ-059 against current code, offline tests,
compatible live evidence, and their full acceptance criteria. Keep unproven
multi-login behavior explicit. Update the checkboxes, ownership index, and
`docs/verification/TASK-032.md` only as evidence permits.

**Acceptance:** Each current requirement has a task owner and exact current
evidence, or remains unchecked with a precise blocker. No historical test is
presented as proof for a changed contract.

**Commit boundary:** `test(TASK-032): requalify current account and Claude contracts`.

<a id="task-033"></a>
### TASK-033: Qualify the final release candidate

- [ ] Task open. **Status: NOT STARTED.**

**Scope:** Review the candidate's public behavior, privacy, process safety,
and documentation; run the full offline gates natively on macOS and Linux;
build and verify four archives; and complete REQ-048 with the final delivered
binary and the providers' own read-only usage displays. Record exact commit,
artifact checksums, command outcomes, and limits in
`docs/verification/TASK-033.md`.

**Acceptance:** The [current release gate](#current-release-gate) is fully
satisfied. If live comparison cannot be performed without a model turn, keep
REQ-048 and this task blocked. Tagging and hosted publication require their
own owner approval after the candidate is reviewable.

**Commit boundary:** `test(TASK-033): qualify v0.1.0 release candidate`.

<a id="current-release-gate"></a>
## Current v0.1.0 release gate

- [ ] Every current specification requirement is checked against applicable
  evidence, including the YAML configuration, Claude, and REQ-048.
- [ ] One bounded release review has no unresolved blocking finding.
- [ ] The final release commit passes the offline gate natively on macOS and
  Linux; other CPU targets are labeled with their actual build/test evidence.
- [ ] All four archives contain the executable, current documents, LICENSE,
  THIRD_PARTY_NOTICES.txt, and valid checksums; version and build provenance
  match the release commit.
- [ ] The delivered candidate passes the owner-account read-only comparison
  in REQ-048 within the same quota period.

## Final handoff format

Return the implemented scope, task commit references, verification summary, local artifact locations/checksums, and remaining compatibility limitations. Explain any live monthly Grok result explicitly. Report failures or unrun environments without presenting them as completed support.

The current YAML configuration supports provider omission and multiple Grok
homes. EPIC-010's reset countdown remains exclusive to the interactive view;
the plain table and JSON keep no reset-time field. Full-screen interfaces and
watch mode remain outside the v0.1.0 scope.
