# Testing

## Contract

Contract: aquarium-test-contract/v1
Profile: make
Enrollment status: configured. The executable Make targets are authoritative; a complete local run and CI provide verification evidence.

## Canonical Commands

| Command | Purpose |
| --- | --- |
| `make test` | Run prepare, unit, integration, and E2E serially; stop on the first failure. |
| `make test-prepare` | Format Go source, vet, lint, build, and check documentation. |
| `make test-unit` | Run isolated Go package suites. |
| `make test-int` | Run self-contained component and package suites. |
| `make test-e2e` | Build and exercise the CLI through its public interface. |

The Makefile prepends `scripts/` to `PATH`. Its `ginkgo` wrapper invokes the Ginkgo CLI from the versions pinned by `go.mod` and `go.sum`; no global Ginkgo installation is needed. Go 1.27 and `golangci-lint` v2.13.2 are required. The test commands never install dependencies. Make sets `GOPROXY=off` and `GOSUMDB=off` for every stage; a missing tool or uncached module fails the stage. Run `go mod download` separately when initially preparing the pinned modules; that setup command can contact the Go module proxy. CI does so before invoking the offline gate.

## Stage Mapping

- Prepare: `gofmt -w` over regular Go files under `cmd`, `internal`, `scripts`, and `test`; then `go vet ./...`, `golangci-lint run ./...`, `go build ./...`, and `go run scripts/check-docs.go`. Formatting may rewrite repository Go source before behavioral checks. Go commands can populate the local build and module caches.
- Unit: `internal/domain`, `internal/options`, `internal/render`, and `internal/provider/envfilter`. These exercise one logical unit with inputs, fixtures, and private temporary files only.
- Integration: `internal/app`, `internal/process`, `internal/provider/{claude,codex,grok}`, `internal/rpc`, `cmd/gaebokchi`, and `scripts`. These use in-process fakes, test-owned files, pipes, and local child processes. The `scripts` suite also checks release archives. Its optional staged-candidate spec skips only when no candidate was selected; the rest of the integration stage still runs. The release CI job requires that candidate and fails when it is missing or incomplete.
- E2E: `test/integration` builds the real `gaebokchi` CLI once and checks output, status codes, failure rows, signal behavior, and provider isolation through public CLI flags and process I/O. Its package name predates this stage mapping. It contains one ancillary cleanup-ownership spec for the E2E harness. There is no successful skip for missing E2E prerequisites.

The Ginkgo unit and integration runs are race-instrumented and fail on empty or pending suites. They execute serially within each suite; the aggregate executes stages serially even under parallel Make. The E2E Ginkgo suite is also race-instrumented and serial, while its production-equivalent CLI build is not instrumented.

## Test Frameworks

All Go unit, integration, and E2E suites run under Ginkgo v2 `v2.33.0` and Gomega `v1.43.1`, pinned in `go.mod` and `go.sum`. Each package has one Ginkgo suite entrypoint; `scripts/ginkgo` invokes the pinned CLI. Gomega is registered in every suite and used for matcher assertions. Migrated checks that need detailed diagnostics use Ginkgo's `GinkgoT` methods. Test-only `TestMain` entrypoints dispatch fake subprocess modes before running a suite. The Go E2E runner has the approved `AQTEST-008` waiver below; no `AQTEST-009` waiver applies.

## Gaori Mapping

No `.gaori/tester.yaml` exists in this repository, so no Gaori parser mapping is configured. If added later, aggregate and prepare output use `generic`; each single-format Ginkgo stage can use `ginkgo` after checking the installed Gaori parser catalog.

## E2E Environment

The E2E harness builds `./cmd/gaebokchi` into a private temporary directory and invokes it only as a child process. It explicitly directs Codex, Grok, and Claude executable flags to the test binary acting as a fake provider; it creates private Codex homes and scripted provider replies. It asserts stdout, stderr, exit status, and child invocation behavior through the public CLI. No real provider CLI, account, database, container, listener, network endpoint, sandbox, or paid API is used. Required identities are the test binary path and test-owned temporary homes; setup failures fail the suite before a provider query. The harness registers and removes only directories created by that invocation, including after suite failure. It does not delete older or similarly named directories. There are no credential variables. Test-only variables include `GAEBOKCHI_FAKE`, `GAEBOKCHI_FAKE_STEPS`, `GAEBOKCHI_FAKE_CODEX_STEPS`, `GAEBOKCHI_FAKE_GROK_STEPS`, `GAEBOKCHI_FAKE_ARGLOG`, and `GAEBOKCHI_FAKE_ENVLOG`; their values are never required from a user or committed. Production-looking targets cannot be reached because the harness supplies explicit fake executable paths and private homes.

The separate release-candidate CI job builds local archives under `dist/` and requires `GAEBOKCHI_REQUIRE_STAGED_RELEASE`; it is not the E2E environment. `GAEBOKCHI_STAGED_RELEASE_DIR` may select a test-owned staging fixture for negative checks.

## Language Diagnostics

`go vet ./...` and `golangci-lint run ./...` run in prepare. All behavioral stages use Ginkgo `--race`; no suite uses Ginkgo parallel workers. Pure Go source has no CGO or `unsafe` use requiring a separate undefined-behavior sanitizer. The optional `govulncheck` queries a public database and remains outside the offline complete gate.

## Legacy Waivers

- Local waiver ID: `GAEBOKCHI-AQTEST-008-001`.
  Rule: `AQTEST-008`, preferred Python black-box E2E implementation only.
  Scope: the pre-existing Go runner for `test/integration`, now executed as a Ginkgo suite from `make test-e2e`; this does not waive the Make entrypoints, isolation, black-box boundary, missing-prerequisite failure, or production safety.
  Pre-existing implementation: the Go E2E harness and its fake provider `TestMain` were in Git before this test-setup proposal. The harness builds the real CLI and checks its public flags, output, exit codes, signals, and cleanup with test-owned resources.
  Equivalence evidence: no real provider dependency; explicit fake binary flags and private homes; actual CLI subprocess assertions; ownership-scoped teardown. The E2E gate fails when building or launching the artifact fails.
  Migration risk: translating the scripted Go provider protocol, process and signal cases, and cleanup ownership checks to Python could lose coverage or introduce a second fake-provider implementation.
  Residual risk: the E2E harness remains Go-specific and does not share Python runner tooling or parser behavior.
  Approval: Approved by Master for this exact Go E2E runner scope.
  Revalidate when the runner command or framework, E2E package scope, fake-provider identity and isolation, cleanup or failure semantics, CLI public boundary, execution-affecting CI or dependency authority, or the Aquarium test contract changes.
