# Provider target selection

This temporary execution dossier supports
[EPIC-015](../roadmap/README.md#epic-015). The
[roadmap](../roadmap/README.md#epic-015) owns task membership, execution order
and lifecycle status. Registration on 2026-10-22 covered documentation only.
TASK-043 implemented alias configuration and matching; TASK-044 implements the
selection option and execution boundary. Integrated acceptance remains pending
TASK-045.

## Outcome and authority

Allow one `-t` / `--target` name to select every configured account of every
matching provider. Default display names and aliases from `config.yaml` match
case-insensitively. Shared aliases may select several providers, whose usage
continues to appear in separate rows.

| Decision | Durable owner |
|---|---|
| Optional provider alias configuration and matching | [REQ-069](../specs/README.md#req-069) |
| Selected account snapshots and no-launch behavior | [REQ-070](../specs/README.md#req-070) |
| Schema, name resolution, target ordering and execution boundary | [ARCH-21](../architecture/README.md#arch-21) |
| Task order, acceptance and dossier retirement | [EPIC-015](../roadmap/README.md#epic-015) |

The new names are provider-selection aliases. Existing account `name` fields
remain display labels. No additional implicit accounts, provider aliases,
credential access or live probes are introduced by this plan.

## Configuration and expected selection

The following configuration example uses placeholder account homes. Alias
metadata and the target option are implemented; TASK-045 owns final integrated
acceptance and this dossier's retirement.

```yaml
aliases:
  codex: [codex]
  cursor: [grok]
codex:
  - name: work
    path: ~/.codex-work
  - name: personal
    path: ~/.codex-personal
grok:
  - name: personal
    path: ~/.grok-personal
cursor:
  - name: default
```

| Invocation | Expected targets |
|---|---|
| `gaebokchi` | Both ChatGPT accounts, Grok, Cursor |
| `gaebokchi -t ChatGPT` | Both ChatGPT accounts |
| `gaebokchi --target=CODEX` | Both ChatGPT accounts, through the configured alias |
| `gaebokchi -t GROK` | Grok and Cursor |
| `gaebokchi -t cursor --json` | Cursor only, in the existing JSON schema |

Without `aliases.codex`, `codex` matches no provider unless another provider
explicitly registers it. `chatgpt` always names the Codex-backed provider.
The complete validation and matching rules are in ARCH-21.

## Work map

| Task | Required work and handoff |
|---|---|
| [TASK-043](../roadmap/README.md#task-043) | Add alias metadata parsing, strict validation, default-name matching and provider-set resolution with focused tests. Preserve existing default execution and provider order. Hand the tested configuration and matcher to TASK-044. |
| [TASK-044](../roadmap/README.md#task-044) | Add the single-use target flag, filter before provider work, retain all selected accounts and pool targets with stable IDs/dense indices, and cover output and no-launch behavior with focused fake-provider tests. Update help, public usage/configuration instructions and affected canonical documents. |
| [TASK-045](../roadmap/README.md#task-045) | Run complete offline acceptance, reconcile requirement coverage and user guidance, record final-candidate evidence, and retire this dossier and its incoming links after retaining lasting content. |

Run tasks sequentially. Tests accompany each implementation task. Follow the
roadmap's task-evidence and task-commit rules; registering this dossier does
not complete a task or authorize installation or publication.

## Acceptance scenarios

- Configuration compatibility: unchanged account lists; omitted/empty aliases;
  metadata before, between or after provider lists; aliases on disabled providers;
  alias-only configuration still failing the account requirement.
- Strict validation: unknown provider keys, wrong mapping/list shapes, nulls,
  non-string aliases and blank aliases fail before provider work. Invalid
  unselected account data continues to fail full configuration validation.
- Matching: mixed-case names and aliases; default-only behavior; configured
  `codex`; duplicate matches; shared aliases and default-name collisions;
  prefix, substring and comma-list inputs do not expand into selections.
- Targets: two Codex accounts; Grok plus Cursor; disabled matching providers;
  no enabled targets; dense indices, stable IDs and configured order; `agy`
  pool expansion and weekly/5h section behavior.
- Execution and output: unselected missing or failing CLIs are never resolved
  or launched; selected failures keep their rows and existing exit behavior;
  the selected set agrees across table, deterministic terminal tests and JSON;
  default execution, help and version remain compatible.
- Arguments: all short/long and equals forms; missing, empty and blank values;
  repeated short/long target flags; unmatched names return exit 2 with no usage
  stdout or provider execution.

Use task-scoped records under `docs/verification/` for actual commands,
candidate identity, results and limitations. TASK-045 requires `make test`,
`go test ./... -count=1` and final document checks as specified in the roadmap.
Real provider queries are outside this extension's acceptance scope.

## Retirement during TASK-045 closeout

1. Verify REQ-069 and REQ-070 against the final candidate and retain the evidence
   in the three task records, indexed by the verification index.
2. Ensure the specifications, architecture and public configuration/usage
   instructions retain every lasting decision needed to use or maintain selection.
3. Delete this dossier after that acceptance evidence and durable content exist.
   Replace or remove its links in the roadmap, documentation index, TODO index
   and any new consumers. Preserve canonical identifiers and lifecycle history.
4. Run `go run scripts/check-docs.go` and `git diff --check`, and inspect all
   references to the retired path. Complete the task and epic only under the
   roadmap's evidence and commit gates.

This dossier's deletion is part of the final feature closeout. Keep it during
implementation; retain the unrelated agy and Cursor dossiers.
