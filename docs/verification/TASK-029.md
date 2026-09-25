# TASK-029: JSON usage output

Date: 2026-09-25. Checks ran on the TASK-029 implementation tree based on
`82da01a`; the later lifecycle edits only close the verified task. This is not
a release or installed-binary qualification.

Requirement: [REQ-052](../specs/README.md#req-052). Architecture:
[ARCH-03](../architecture/README.md#arch-03),
[ARCH-10](../architecture/README.md#arch-10).

Changed scope: usage flag parsing, explicit JSON result projection, CLI output
selection, offline contract and executable tests, help, README, and canonical
documentation.

Environment: macOS darwin/arm64, Go 1.27.1, golangci-lint 2.13.2. The
executable tests use synthetic responses from the suite's fake Codex, Grok,
and Claude processes with private temporary Codex homes.

Checks on the working tree:

- `go test ./internal/options ./internal/render ./cmd/gaebokchi -count=1` — passed.
- `go test ./test/integration -count=1` — passed, including real-executable
  JSON success, partial failure, cancellation, and broken-stdout cases.
- `go test ./... -count=1` — passed, including offline integration tests.
- `go vet ./...` — passed.
- `golangci-lint run ./...` — passed with 0 issues (v2.13.2).
- `gofmt -d` on the exact changed Go files below — no diff:

  ```sh
  gofmt -d \
    cmd/gaebokchi/main.go cmd/gaebokchi/main_test.go \
    internal/options/options.go internal/options/options_test.go \
    internal/render/json.go internal/render/json_test.go internal/render/table.go \
    test/integration/e2e_test.go test/integration/pipes_test.go
  ```

- `go run scripts/check-docs.go` — passed; 50 Markdown files, links and IDs OK.

Focused tests cover the exact six-field JSON schema, target order, fractional
precision, numeric zero, missing-value null, account normalization, duplicate
disambiguation, cleanup failure, partial failure, clean output, text help,
invalid arguments, stdout failure, safe stderr diagnostics, and exclusion of
private fields. Existing table and version tests passed in the full offline
suite. Executable-level tests use separate stdout/stderr buffers and real OS
pipes to assert one JSON document, partial-result retention, signal exit-code
precedence, and safe output-failure diagnostics.

Review corrections: the roadmap order now includes EPIC-009/TASK-029; this
record names the environment and exact formatting scope; executable-level
JSON tests cover the previously missing process boundary. The task-scoped
commit subject remains `feat(TASK-029): add JSON usage output`.

No real provider account query, installation, release staging, or publication
was performed. The task-scoped commit subject above records TASK-029,
EPIC-009, and REQ-052 closure without a self-referential commit hash.
