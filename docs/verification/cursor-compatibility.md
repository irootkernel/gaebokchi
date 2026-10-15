# Cursor Agent compatibility and qualification prerequisites

## Current finding

The installed Cursor Agent identifies itself as 2026.10.14-0f9edde. Static
inspection on 2026-10-15 found usage code in its package, but no qualified
metadata-only noninteractive quota command. Positive runtime support remains
unqualified under [TASK-039](../roadmap/README.md#task-039).

The command help lists `status` and `about` with JSON output, but their inspected
implementations return identity, tier or system information rather than quota
data. Use the unambiguous executable `cursor-agent`; an executable named
`agent` can belong to a different provider.

## Inspected package behavior

In the installed version's `9627.index.js`, the bundled modules
`src/usage/usage-command.ts`, `src/usage/usage-data.ts` and
`src/components/usage-pager.tsx` implement the interactive `/usage` path.
The data loader calls `getCurrentPeriodUsage`, with supporting plan and
spend-limit reads, and opens a pager. This does not expose a standalone quota
JSON command.

The package includes current-period usage percentages and billing-cycle fields;
its plan metadata can distinguish weekly and monthly periods. The inspected
pager uses a monthly title and does not establish a safe period interpretation
for every plan. Its percentage defaults and display clamping are not the
Gaebokchi normalization contract. The
[official usage guide](https://cursor.com/help/models-and-usage/usage-limits)
also distinguishes weekly plans from legacy monthly plans.

The installed `index.js` configuration-path module honors `CURSOR_CONFIG_DIR`.
Its credential module uses the Cursor credential domain and, on macOS by
default, Keychain entries independent of that configuration directory.
Changing the configuration path is not proof of a different authenticated
account. This epic uses one existing default CLI login.

## Required next qualification

TASK-039 must qualify an actual installed CLI entry point that returns usage
without model turns, agent tools, session creation, login or billing changes.
Freeze its response fields, explicit period, model/pool semantics, auth failures,
missing-data behavior, reset timestamps and process lifecycle before production
implementation. The [CLI parameters reference](https://cursor.com/docs/cli/reference/parameters)
is a discovery source, not proof that a particular binary supports an interface.
Freeze automatic-update suppression, identity-affecting ambient overrides and
credential selection on every claimed platform. Investigate `CURSOR_CONFIG_DIR`
without treating the macOS observation as Linux qualification. Mark untested
platforms explicitly. [S-20](../architecture/README.md#s-20) and
[S-21](../architecture/README.md#s-21) are discovery sources only.

`cursor-agent -p /usage` was not executed and is not a qualified local-command
path. Neither a wrapper around bundled private modules nor a direct Dashboard
API client meets the retained CLI boundary.

No live usage query, login or account-switch experiment was performed for this
note. Weekly/monthly source declarations are not live plan acceptance.
The [planned architecture](../architecture/README.md#arch-19) and
[execution dossier](../todo/TODO-CURSOR-AGENT-SUPPORT.md) define the next work.
