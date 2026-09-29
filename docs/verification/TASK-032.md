# TASK-032: Accept the current YAML and Claude contracts

Task: TASK-032
Status: COMPLETE
Goal and implementation scope: Re-qualify the current configured-account CLI and Claude adapter after the YAML account-selection change.
Requirements and architecture sections covered: REQ-002, REQ-006 through REQ-008, REQ-010 through REQ-011, REQ-039, REQ-056 through REQ-059; ARCH-03, ARCH-04, ARCH-09, ARCH-10, ARCH-12, ARCH-16.
Changed files: Claude output-bound regression test, RPC test synchronization, README compatibility summary, requirement checkboxes and trace lines, verification indexes and compatibility records.

Verification on 2026-09-29: the live binary was built from clean commit `af5ac0e424fd6c0546e6f7721f4ff081db2236fd`; the offline gates ran on that product code plus this task's test and documentation changes.

- `go test ./internal/options ./internal/provider/claude ./internal/provider/envfilter ./cmd/gaebokchi -count=1`: exit 0. The suites cover YAML ordering, omission, strict keys, nulls, aliases, path identity and missing paths; CLI error/no-launch and ordered output; per-provider child environments; Claude auth, exact commands, weekly envelope, reset edge cases, and failure classifications.
- `go test ./internal/provider/claude -count=1`: exit 0 after adding an oversized-output regression. The 256 KiB bound now has a direct failure-path check.
- The first full `make test` on the TASK-032 worktree exited 2 because `TestBlockedRejectionWriteReleasedByAbort` checked the abort count immediately after `Call` saw `deadCh`, before `die` could invoke the hook. The test now waits for the hook's completion signal. `go test -race ./internal/rpc -count=1`: exit 0 after that test-only correction. The initial full gate is a failed attempt, not acceptance evidence.
- The corrected `make test`: exit 0 on darwin/arm64 with Go 1.27.1, including vet, lint, race-enabled unit and integration suites, and E2E 32/32.
- `docker run --rm --platform linux/arm64 -v "$PWD":/src -v "$(go env GOMODCACHE)":/go/pkg/mod -w /src golang:1.27.1-bookworm sh -c 'go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 && make test'`: exit 0 on native linux/arm64 with the same full stages and E2E 32/32. The make gate itself used `GOPROXY=off` and `GOSUMDB=off`.
- After the full gates, the same abort-wait pattern was applied to the terminal-EOF RPC test; `go test -race ./internal/rpc -count=1` and `go run scripts/check-docs.go` passed on the final test/documentation candidate. This last edit changes only test synchronization. The full gates above ran before that last test-only edit.
- `codex review --uncommitted`: exit 0; the separate read-only reviewer reported no actionable finding. Its own `go test -race ./internal/rpc -count=3` attempt was invalid because Ginkgo rejects `go test -count` values above 1; the review's test claim is not used as acceptance evidence. The local full and focused gates above provide that evidence.
- TASK-031's earlier full `make test` ran on the same product code on darwin/arm64 and native linux/arm64. Its archives were dirty-tree packaging evidence, not the final release artifacts.
- `go build -o /tmp/gaebokchi-task032-bin ./cmd/gaebokchi`: exit 0. `go version -m` reported `vcs.revision=af5ac0e424fd6c0546e6f7721f4ff081db2236fd` and `vcs.modified=false`.
- One prompt-free `--json` usage run of that binary against the owner's existing configuration exited 0 in about four seconds with four ordered `ok` rows: two Codex, one Grok, one Claude. Each had a numeric weekly value, no cleanup failure, and no stderr diagnostic. Only provider, status, value-presence, and cleanup flags were emitted from the observation; no account alias, identifier, credential, percentage, or raw provider output was retained.
- Installed CLI versions were `codex-cli 0.158.0`, `grok 1.0.44`, and Claude Code `2.1.284`. The live result qualifies those versions' positive path on this host. Earlier version-specific protocol and reset-text details remain in the compatibility records; synthetic tests cover negative branches.

Contract assessment:

- REQ-002 and REQ-006 through REQ-008: the current parser, target construction, CLI dispatch, and fake-provider E2E cover each configured target, provider omission, help/version without configuration, pre-launch errors, rejected flags, name and path validation, child home selection, and missing-home rows. The two real Codex targets and one real target of each other provider exercised the positive configured path.
- REQ-010 through REQ-011: configured aliases are the output identity even when Codex reports an email; table and JSON tests cover target order, each status row, headings, and line count. The live JSON snapshot kept four ordered rows.
- REQ-039: process and environment tests verify argv-only invocation, neutral cwd, relative executable resolution, credential override filtering, and per-target home replacement. The live success provides a positive child-process path, not direct inspection of child environments.
- REQ-056 through REQ-059: the current Claude CLI returned a successful first-party subscription usage row. The provider and parser suites cover command safety, auth and unsupported branches, envelope validation, weekly all-models selection, oversized output, and reset ambiguity. Source inspection confirms immediate stdin close. The live run did not retain a reset timestamp, so it is not evidence that this account's current response included one; reset parsing is qualified by the earlier observed text and current offline cases.

Compatibility limit: this host has two distinct configured Codex homes but only one authenticated Grok home and one authenticated Claude home. Distinct real Grok or Claude logins behind multiple configured paths were **not** exercised; aliases alone do not establish account isolation. The [deferred feedback](../deferred-feedback/README.md) owns that additional qualification. No live Linux provider query or amd64 runtime test was performed for this task. REQ-048 requires a separate comparison with each provider's own usage display on the final delivered candidate.

Findings and bounded remediation: the README's version and Linux-evidence summary lagged the current candidate. The Claude output cap lacked a direct failure-path test. The first full gate also exposed a test race between observing `deadCh` and the abort hook; the RPC test now waits for the hook. Product code was unchanged.
Task commit subject: `test(TASK-032): requalify current account and Claude contracts`.
Handoff: TASK-033 owns bounded release review, final native gates and archives, and REQ-048.
