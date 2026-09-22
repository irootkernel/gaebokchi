# TASK-017: Fix executable-path resolution and CLI output/signal behavior

Task: TASK-017
Status: COMPLETE
Goal and implementation scope: Resolve explicit relative executable paths against the invocation environment before the neutral child cwd can break them; implement the documented SIGPIPE and exit-code precedence contract through the actual executable with real OS pipes.
Requirements and architecture sections covered: REQ-006 (binary-override scope), REQ-024, REQ-039, REQ-045 (re-qualification); ARCH-03, ARCH-10.
Changed files: internal/provider/codex/provider.go and internal/provider/grok/provider.go (ResolveBin absolutizes explicit relative results; ErrDot stays an error), cmd/gaebokchi/main.go (SIGPIPE notified so a broken fd-1/2 write becomes the documented output failure instead of signal death; exit precedence centralized in finalExit: first handled signal 130/143 > output failure 1 > partial failure; help/version write failures handled; signal watcher released when execution finishes), plus tests: internal/provider/{codex,grok}/provider_test.go additions and the new test/integration/pipes_test.go.
Verification (environment: macOS 26.6.2 arm64, go1.26.6):
  - V-EXIT / V-E2E against the actual executable with real OS pipes (`go test -count=1 ./test/integration`): TestClosedStdoutPipeExitsOneNotSignalled (closed stdout reader → normal exit 1 with the safe diagnostic, not SIGPIPE death — the audit's first finding), TestSignalWithBrokenStdoutKeepsSignalCode (SIGINT → 130 and SIGTERM → 143 while stdout is unwritable — the audit's inverted-precedence finding), TestSignalMixedCompletionKeepsNumericRow (writable stdout: completed grok row stays numeric, in-flight codex rows render CANCELLED, four logical lines, exit 130), TestHelpVersionBrokenPipeExitsOne (help/version output failures exit 1 without launching providers).
  - V-OPTIONS/V-CODEX/V-GROK (`go test -count=1 ./internal/provider/codex ./internal/provider/grok ./cmd/gaebokchi`): TestResolveBinRelativeMadeAbsolute (relative "./name" with a space resolves to the absolute invocation-cwd path), TestResolveBinErrDotProtectionPreserved (implicit current-directory PATH lookup still fails with exec.ErrDot), TestResolveBinMissingAndUnexecutable, and end-to-end TestRelativeBinaryOverrideFromInvocationCwd in the integration suite (the audit's exact reproduction: a "./my codex" wrapper script launched from the invocation cwd completes its query instead of becoming NOT INSTALLED; no shell expansion — the path is used as a single argv element by process.Start).
  - Full regression: `go vet ./...` pass; `go test -count=1 ./...` all 12 packages pass.
Design notes:
  - SIGPIPE policy: signal.Notify(SIGPIPE) is installed for the whole execute() scope so the runtime delivers the signal harmlessly while the failing write returns an error; the output-failure path (exit 1) is driven by the write error, which is deterministic. This matches architecture.md's requirement to test broken-pipe behavior rather than assume platform defaults.
  - finalExit centralizes precedence, removing the audit's `return 1`-before-signal-check ordering bug; diagnostics are still written to stderr after a failed table write (stderr is a separate descriptor).
  - The signal watcher goroutine is released through a watchDone channel closed at the end of execute(), so no goroutine stays blocked on the signal channel after execution finishes.
Fixtures: unchanged.
Findings and bounded remediation: none outstanding in this scope.
Compatibility/platform limitations: SIGPIPE notification semantics are Unix; the project targets macOS/Linux only (windows is out of scope), so no portable divergence was introduced.
Task commit subject: `fix(TASK-017): resolve relative executable paths and enforce output signal exit contract`
Handoff: proceed to TASK-018 (inbound RPC validation).
