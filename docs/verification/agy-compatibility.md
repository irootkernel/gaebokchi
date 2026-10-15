# agy compatibility and qualification prerequisites

## Current finding

The installed Antigravity CLI identifies itself as agy 1.3.5. Static capability
inspection on 2026-10-15 has not established a usable metadata-only quota entry
point or two independently authenticated account namespaces. Positive runtime
support remains unqualified under [TASK-034](../roadmap/README.md#task-034).

The inspected help advertises print output formats and interactive agent
operation, but no dedicated auth-status or usage subcommand. The official
[/usage guide](https://antigravity.google/docs/cli/commands/usage/) describes an
interactive model-quota panel. The
[headless guide](https://antigravity.google/docs/cli/headless/) does not establish
a machine-readable quota response that satisfies this project's full safety
contract.

`agy -p /usage` was not executed. Whether it short-circuits to a local command
without model work is unverified. Do not infer that behavior from its spelling
or test it through an unguarded model-capable invocation.

## Account selection gap

The [authentication guide](https://antigravity.google/docs/cli/install/)
describes provider-owned OS keyring authentication. Binary strings include
`ANTIGRAVITY_APP_DATA_DIR` and `JETSKI_APP_DATA_DIR`, but their presence does not
establish credential or shared-process isolation. `AGY_ACCOUNT` appears in
plugin or sandbox account-selection text; it is not a qualified Google login
selector. No `AGY_HOME` contract has been established.

Two distinct path aliases therefore do not demonstrate two distinct logins.
TASK-034 must resolve the official selector and its authentication namespace
before any account configuration can claim Codex-like isolation.

## Required next qualification

Qualify a bounded CLI path with zero model turns, no agent tools or sessions,
no interactive auth or browser launch, structured auth failures and explicit
quota/period/reset semantics. Then verify two pre-authenticated accounts,
concurrent isolation, absent-login behavior and preservation of the default
login selection. Capture only sanitized, field-allowlisted fixtures.
Under [REQ-060](../specs/README.md#req-060), each account must return a measured
known-period quota. Freeze update suppression and selector/default-namespace
rules, and privately recheck distinct CLI-reported identities after concurrent
cross-namespace reads. A non-path selector requires a configuration decision
before TASK-035. No identity or credential belongs in committed fixtures.

The [plans guide](https://antigravity.google/docs/plans/) describes five-hour
and weekly limits. It does not establish that a particular response exposes
both, or permit classifying a quota by the time remaining before reset.
These discovery sources are [S-17](../architecture/README.md#s-17),
[S-18](../architecture/README.md#s-18), [S-19](../architecture/README.md#s-19)
and [S-22](../architecture/README.md#s-22); none is runtime evidence.

No live usage query, login, account-switch experiment or multi-account acceptance
was performed for this note. Public documentation and static inspection support
the current prerequisite assessment; they are not positive runtime acceptance.
The [planned architecture](../architecture/README.md#arch-18) and
[execution dossier](../todo/TODO-AGY-MULTI-ACCOUNT-SUPPORT.md) define the next work.
