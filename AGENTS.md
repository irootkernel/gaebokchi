# AGENTS.md

Repository guidance for AI coding agents working on gaebokchi.

## Core Behavior

### 1. Lead with Conclusions

- State the result or current finding first, followed by useful evidence and material limits.
- Do not repeatedly restate requirements or narrate routine work.

### 2. Reuse Verified Information

- Inspect the requested code and its named authorities before changing anything. Resolve discoverable facts before asking Master.
- Reuse established facts instead of reading or searching for them again. Recheck only the affected information when relevant state changes, evidence conflicts, or missing context makes it unreliable.
- State material assumptions and surface meaningful trade-offs. Ask when unresolved ambiguity would materially change the result, and push back on conflicts with repository authority, safety, or Master's goal.

### 3. Act on Sufficient Evidence

- Stop investigating once the evidence supports action. When the root cause is established, implement the smallest complete, durable fix within the authorized scope.
- Weigh correctness, performance, maintainability, and structural fit rather than diff size alone. If a broader design exceeds scope, complete a bounded step that satisfies current acceptance criteria.
- Reuse established patterns. Avoid speculative features, abstractions, configurability, compatibility layers, and handling for states repository invariants make impossible. Simplify complexity that the required behavior does not justify.
- Touch only what the outcome and its verification require. Preserve unrelated user work, match local style, and remove only artifacts made obsolete by this change.
- Record only independent remaining work in the canonical `deferred-feedback` owner. If none exists, propose the entry and obtain approval before creating an owner. Promote epic-sized work to a TODO candidate or roadmap unit; never defer current correctness or acceptance work.

### 4. Carry Authorization Forward

- Continue already approved work without asking for confirmation again. Ask only when a material change exceeds that authorization or an applicable rule requires a distinct approval.
- Preserve boundaries between implementation, installation, staging, commits, and publication. Check for relevant state changes before acting on an approved proposal.

### 5. Verify in Proportion to Risk

- Define success checks before implementation. Verify the affected behavior and relevant failure paths with rigor proportionate to the actual risk.
- Run focused checks first and honor required repository gates. Broaden or repeat checks when changes, failures, or unresolved concerns justify it.
- Do not add tests merely to appear rigorous or use prose matching as a substitute for behavior verification.

### 6. Finish When Complete

- Continue until deliverables and required verification are complete or a concrete blocker prevents progress.
- Once material constraints are resolved or clearly reported, provide the handoff and stop. Report the result, necessary evidence, skipped checks and their reasons, and remaining uncertainty without opening unrelated work.

### 7. Delegate Selectively

- Use a sub-agent only for an independent task when the expected benefit outweighs coordination cost.
- Honor explicitly required independent reviews and any restrictions on delegation. Keep tightly coupled work local.

## Master Preferences

- Respond to Master in Korean using polite speech. When directly addressing the user, use exactly `Master`.
- Keep repository artifacts in their established language and style. Use English when no convention exists unless Master requests otherwise.
- Provide concise conclusions and useful evidence without exposing private chain-of-thought.

## Aquarium Development Guide

- Use `$aquarium:task-handler` for one named roadmap task and `$aquarium:epic-handler` for one roadmap epic when a roadmap exists. Use `$aquarium:epic-validator` to validate a completed epic.
- Use `$aquarium:new-project`, `$aquarium:new-feature`, or `$aquarium:refactor` for explicitly requested Ouroboros-assisted design work. Use `$aquarium:war-room` for a difficult bug.
- Use `$aquarium:dev-setup-global` for supported user-global tools and `$aquarium:dev-setup` for repository-local tooling and guidance. Use `$aquarium:docs-setup` and `$aquarium:test-setup` when their documented setup is requested.
- Use `$use-dolgorae` for explicitly requested workspace or Profile operations, `$use-mulgae` for authorized reviews, and `$use-gaori` for selected checks. Follow each paired skill's native contract.
- Use `$use-sorage` only when Master explicitly requests a broker operation. Project registration does not authorize inbox or outbox discovery. Follow its Handoff and Vault rules; never edit the managed Vault or derived `.sorage/INBOX.md` directly.
- Aquarium workflows use Podway by default for Git-backed work unless Master opts out before the first managed-session mutation. Use `$use-podway` for an explicitly requested Procedure v2 operation and `$create-podway-procedure` for Procedure authoring.
- Treat `.podway/procedures/aquarium-*-v2.yaml` as repository-local workflow evidence and routing authority. Keep `.mulgae/` runtime, `.gaori/runs/`, `.podway/runtime/`, and derived `.sorage/` out of Git.
- Use `$humanizer` once for final English human-authored prose and `$humanize-korean` once for final Korean human-authored prose. Preserve meaning, commands, identifiers, and citations.

## Project Configuration

### Repository Index and Authorities

- `README.md` owns the public product overview, installation from GitHub Releases or source, supported usage, privacy boundary, compatibility, and license notice.
- `docs/README.md` indexes project documents; `docs/specs/README.md`, `docs/architecture/README.md`, and `docs/roadmap/README.md` own requirements, architecture, and roadmap status. `docs/implementation-tips/README.md` lists the local verification commands.
- This is a Go 1.27 module. Run `go vet ./...`, `golangci-lint run ./...`, and `go test ./... -count=1` for the documented offline checks; use the narrower package tests and race command listed in `docs/implementation-tips/README.md` when relevant. User installation is in `README.md`; release staging and publication steps are in `docs/implementation-tips/README.md` and `scripts/build-release.sh`.

### Commit Messages

- Every developer-authored subject uses one header and a concise English imperative summary: `[FEAT]`, `[FIX]`, `[DEV]`, `[TEST]`, `[DOC]`, `[CI]`, `[REL]`, or `[INT]`.
- Use `[FEAT]` for new user-facing capabilities, `[FIX]` for defects, and `[DEV]` for development tools, build configuration, dependencies, or internal integration. Use `[TEST]`, `[DOC]`, or `[CI]` for changes limited to that concern, and `[INT]` for internal contracts or refactoring.
- Release metadata commits use exactly `[REL] Release v<version>`. Tool-generated commits retain their native subjects and provenance; never hand-author a `[SANHO]` synchronization commit.
- End every developer-authored message with one trailer block containing exactly one `Aquarium-Workflow:`, `Aquarium-Epic:`, and `Aquarium-Task:`, in that order. Separate the block from the subject or optional body with one blank line; do not put blank lines between trailers.
- `Aquarium-Workflow` records the responsible `aquarium:<canonical-skill-name>`. Preserve a verified original skill through later commits, including changes from documentation, development, or test setup. Child phases and reviews inherit an established parent's owner; independently approved workflows own their candidate. Use `aquarium:task-commit` only for a direct commit without an original Aquarium workflow, and `none` for work outside Aquarium.
- Resolve a Task and its parent Epic from the accepted relationship and canonical roadmap, including a standing `EPIC-000` where applicable. Preserve native IDs, suffixes, and scope qualification. Epic-owned work uses `Aquarium-Task: none`; work without either owner uses `none` for both. Resolve unknown relationships instead of treating them as absent.
- Put primary work IDs in these attributes rather than the subject or legacy `Task:` and `Task-Relationship:` aliases. Use `$lore-commits` for useful decision context on non-trivial changes; optional Lore trailers may repeat. Preserve required evidence, release-note, and tool provenance trailers.
- Parse the prepared message and actual post-hook commit with `git interpret-trailers --parse`; verify all three attributes against the accepted candidate. Attribution never grants staging, commit, amend, push, or publication authority.

### Project-Specific Operating Rules

- Provider CLIs may refresh their own login state during a usage query. Do not run a live account query as a setup check; use the documented offline tests.
- Keep credentials with the provider CLIs. Never print real account identifiers or authentication data in setup reports.
- The distribution channel is GitHub Releases for `github.com/irootkernel/gaebokchi` plus source builds. Staging archives or documenting the channel does not authorize tagging or publishing; publish only when Master explicitly requests it.
