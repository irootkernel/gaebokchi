# ADR-0002: Preserve native quota periods and CLI credential ownership

- **Status:** Accepted design direction on 2026-10-15; implementation pending.
- **Delivery:** [EPIC-012](../roadmap/README.md#epic-012) and
  [EPIC-013](../roadmap/README.md#epic-013).
- **Related:** [REQ-060](../specs/README.md#req-060) through
  [REQ-066](../specs/README.md#req-066),
  [ARCH-17](../architecture/README.md#arch-17) through
  [ARCH-19](../architecture/README.md#arch-19).

## Context

The implemented product selects a weekly quota for each configured account.
agy needs multiple independent accounts and can have model-specific limits
across five-hour and weekly periods. Cursor plans can use weekly or monthly
allowances. A single weekly percentage cannot represent these results.

The installed CLIs have not yet passed the new providers' metadata-only
qualification gates. agy's authenticated account isolation is also unproven.
Adding a provider requires the same credential and business-level read-only
boundaries as the existing adapters.

## Decision

Represent each account query as a result containing quota results. Preserve the
provider's quota identity, model or pool label, period, used percentage, reset
instant and per-quota outcome.

Show weekly, five-hour and monthly sections in that order. Repeat an account in
each applicable section and keep model pools separate. Show account-wide errors
and explicit non-applicable accounts once in a final status area. Omit empty sections and avoid inferred periods or
cross-pool arithmetic.

Change `--json` to a versioned v2 snapshot containing `schema_version: 2` and a
`quotas` array in each account result. This is an intentional compatibility
break; there is no additional legacy-output flag. The grouped plain-text layout
also changes its public contract. Existing JSON and text consumers need
to migrate when TASK-036 delivers the change.

Leave credential discovery, login, refresh and storage with each installed CLI.
Require agy to support at least two independent authenticated account namespaces.
Cursor starts with one default CLI login. Production adapters follow successful
qualification of the actual installed metadata interfaces.
Require measured known-period quotas for both agy accounts; unknown-only results
cannot establish positive support. ARCH-17 owns provider mappings, status/exit
semantics and stable native or documented app-owned meter keys. App-owned keys
identify application meters and never claim provider provenance. The spec's
conditional remaining-fraction rule is the sole permitted ratio conversion.

## Alternatives considered

- Keeping weekly output alone would hide the additional supported periods.
- Combining periods or model pools into one percentage would change the metric.
- A second JSON flag would retain two output contracts; the owner chose to
  migrate the existing flag to the new versioned structure.
- Direct provider APIs, credential extraction and interactive screen scraping
  would exceed the retained product boundary.

## Consequences

The common model, rendering, JSON and existing-provider tests change in EPIC-012.
EPIC-013 consumes those outcomes. CLI capability gaps can block provider
implementation, and an `UNSUPPORTED` row cannot close either epic.

Current weekly behavior and the original JSON shape remain implemented until
their owning tasks land. This ADR does not qualify a provider, implement an
interface, authorize live account operations, or change release status.
