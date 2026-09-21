# TASK-002: Bootstrap the Go module, domain and target policy

Task: TASK-002
Status: COMPLETE
Goal and implementation scope: Smallest compilable foundation expressing the approved CLI targets and status semantics: go.mod, domain types, pure options parser with path policy, minimal composition root that fails explicitly instead of inventing usage.
Requirements and architecture sections covered: REQ-007, REQ-008 (primary); ARCH-01, ARCH-02, ARCH-03, ARCH-04, ARCH-14.
Changed files: go.mod, .gitignore, cmd/gaebokchi/main.go, internal/domain/{target,status,result,percent}.go, internal/domain/domain_test.go, internal/options/options.go, internal/options/options_test.go, docs/verification/TASK-002.md.
Verification (environment: macOS 26.6.2 arm64, go1.26.6):
  - V-OPTIONS (parse/path scope): `go test ./internal/options` — ok. Covers default homes with ambient CODEX_HOME set (options package reads no provider env), flag-order replacement, tilde/`~user`/relative/empty path policy, duplicate homes incl. symlink alias, missing-home validity, timeout bounds 1s..120s, unknown flags, positional args, scalar-flag repetition, `--flag=value` form, help/version precedence and syntactic-validity requirement.
  - V-QUOTA (percentage policy core): `go test ./internal/domain` — ok. Presence-aware `*float64`, finite [0,100] validation rejecting NaN/Inf/negative/over-100 without clamping, signed-zero normalization, all 15 status cell strings and exit classes.
  - `go vet ./...` — pass. `go test ./...` — pass. `go test -race ./internal/...` — pass.
  - Scaffold exits verified on the built binary: usage error 2, help 0, version 0, unimplemented runtime 1 (explicit stderr line, no table, no provider launch).
  - Module path decision: local-only module `gaebokchi` (repository has no remote identity; per ARCH-01 no remote repository is invented). Toolchain: go1.26.6, `go 1.26` language version in go.mod.
Fixtures: none new; pure code tests.
Findings and bounded remediation: boolean flags now also reject repetition ("may be supplied only once") per ARCH-03; a test expectation initially misread 1200ms as sub-second and was corrected to 999ms.
Compatibility/platform limitations: none for this scope.
Task commit subject: `feat(TASK-002): add domain and CLI target foundation`
Handoff to next task: EPIC-001 local gate complete (qualified contracts + passing foundation); TASK-003 implements the owned-child process lifecycle.
