# TASK-010: Deliver the final table and CLI exit behavior

Task: TASK-010
Status: COMPLETE
Goal and implementation scope: cmd/gaebokchi composition root (options, signals, providers, runner, renderer) plus internal/render: three-column table with one-decimal percentages, label sanitization, stderr-only diagnostics, help/version without provider launch, exit codes 0/1/2/130/143.
Requirements and architecture sections covered: REQ-002, REQ-006, REQ-010, REQ-011, REQ-012, REQ-024 (primary); ARCH-03, ARCH-04, ARCH-10.
Changed files: cmd/gaebokchi/main.go, cmd/gaebokchi/main_test.go, internal/render/table.go, internal/render/table_test.go, docs/verification/TASK-010.md.
Verification (environment: macOS 26.6.2 arm64, go1.26.6):
  - V-OUTPUT: golden mixed-status table — exactly the three contractual headings, one-decimal cells, final newline, no title/separator/blank lines/color/control sequences; exactly 4 logical lines by default and N+2 with three custom homes (5 lines asserted); window provenance (quota id, reset time, source) never renders; sanitization replaces tabs/newlines/ESC/bidi/C1/DEL/invalid-UTF-8 with collapsed safe spaces while preserving printable Unicode (개복치 🐡 kept); sanitized labels cannot add columns or rows; diagnostics only on stderr, one bounded line per non-clean row, none for successful numeric rows.
  - V-EXIT: usage error → 2 (no table); help/version → 0 with no provider launched (asserted with bomb fakes); all-clean and monthly/API-key non-applicability → 0; one AUTH REQUIRED row → 1 with the full table retained and its diagnostic on stderr; broken stdout → 1 with a single safe diagnostic; SIGINT mid-query → 130 with the CANCELLED table still rendered (re-exec child asserting the wait status).
  - V-IDENTITY (composition): protocol email preferred, home-alias fallback on failure, duplicate labels disambiguated (TASK-009 suite), Grok alias on every outcome.
  - Live smoke (informational, pre-TASK-013): the built binary against the real installed CLIs printed the exact three-account table in 1.8 s (ChatGPT 39.0%, ChatGPT 39.0%, Grok 30.0%) with exit 0.
  - `go vet ./...`, `go test ./...` — pass.
Fixtures: none new (golden strings inline; live values not persisted).
Findings and bounded remediation: help/version handling moved into the testable execute path after a test caught it bypassing the no-provider assertion; sanitize expectations aligned to the spec's replace-with-space semantics including invalid-UTF-8 replacement characters.
Compatibility/platform limitations: signal exit codes verified on darwin (unix wait status); linux re-verified in TASK-012's native matrix.
Task commit subject: `feat(TASK-010): expose stable usage table and exit contract`
Handoff to next task: TASK-011 qualifies the executable offline with fault/safety/e2e suites.
