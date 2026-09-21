# TASK-003: Implement bounded owned-child lifecycle

Task: TASK-003
Status: COMPLETE
Goal and implementation scope: Finite, owned subprocess execution for provider adapters: explicit argv with os.Pipe-based plumbing, private 0700 temporary cwd, exactly-once wait, EOF(500ms) -> SIGTERM(500ms) -> group SIGKILL cleanup within a 2s budget, bounded 64 KiB stderr drain, descriptor/goroutine cleanup, temp-dir removal.
Requirements and architecture sections covered: REQ-020, REQ-040 (primary); ARCH-02, ARCH-05, ARCH-09.
Changed files: internal/process/child.go, internal/process/process_unix.go, internal/process/child_test.go, docs/verification/TASK-003.md.
Verification (environment: macOS 26.6.2 arm64, go1.26.6):
  - V-LIFECYCLE: `go test ./internal/process -count=1` — ok (2.975s). Covers clean EOF exit with temp-dir removal, early exit (exit code 3 observed; dead-stdin write harmless), stuck child stopped within the 2s budget, SIGTERM-ignoring child stopped by group SIGKILL within budget, 1 MiB stderr flood drained with 64 KiB retention cap, private temp cwd outside the project tree, verbatim child environment, inherited pipe holder not blocking cleanup (reader unblocked), unrelated process survival during cleanup, idempotent double Close with exactly-once reap, and start-failure resource cleanup.
  - V-SAFETY (process scope): no shell wrapper anywhere (exec.Command with argv array); group signals reach only the child's own process group created via Setpgid at start; an unrelated `sleep` process survived cleanup; the private cwd is created under os.TempDir and removed only by the owning child's Close.
  - `go vet ./...` — pass; `go test ./... -count=1` — pass; `go test -race ./internal/process` — pass (9.056s).
Fixtures: none (helper scenarios are compiled from the test binary itself).
Findings and bounded remediation: an initial test version deadlocked by reading stdout before delivering stdin EOF; fixed by exposing Child.CloseStdin and closing after write, which also gives adapters an explicit EOF signal. The first run was stopped and rerun cleanly.
Compatibility/platform limitations: unix-only signal/group implementation (`//go:build unix`, darwin+linux); Windows is out of v0.1 scope.
Task commit subject: `feat(TASK-003): add bounded provider process lifecycle`
Handoff to next task: TASK-004 builds newline JSON framing, request-ID correlation, and the Codex/ACP codecs on top of this Child/Factory boundary.
