# TASK-021: Perform final qualification and record the release decision truthfully

Task: TASK-021
Status: COMPLETE
Goal and implementation scope: Qualify the corrected candidate tree end to end, re-verify every requirement unchecked in TASK-015, and record a release decision that keeps code-remediation completion distinct from full release acceptance.
Requirements and architecture sections covered: REQ-044, REQ-050 (re-qualified); REQ-048 (state recorded, remains open); REQ-047 (corrected-tree platform evidence appended); ARCH-11, ARCH-13, ARCH-14.
Changed files: docs/requirement.md (16 checkboxes restored with re-qualification notes; header status; REQ-047 evidence), docs/plan.md (TASK-021 and EPIC-007 closed; header status; final release gate re-evaluated), docs/verification/live-acceptance.md (artifact-scope correction note distinguishing the pre-remediation binary from the corrected candidate), docs/verification/TASK-021.md (this record), dist/gaebokchi-0.1.0/ (rebuilt artifacts).
Final integration gate — exact commands, outcomes, environments (all against the corrected tree, source revision `1846f731435c9fe945411ddfcb2e9d615cf8e383` plus this task's documentation-only commit):
  - Formatting: `gofmt -l internal cmd test scripts` — empty (no output). The CI workflow enforces this gate (.github/workflows/ci.yml).
  - `go vet ./...` — pass on darwin/arm64 (go1.26.6, macOS 26.6.2) and linux/arm64 (golang:1.26.6-bookworm container).
  - `go test -count=1 ./...` — all 12 packages pass on darwin/arm64 and linux/arm64 (containerized).
  - `go test -race -count=1 ./...` — all packages pass on darwin/arm64; `go test -race -count=1 ./internal/rpc ./internal/process ./internal/app ./test/integration` — pass on linux/arm64 (containerized).
  - `go run scripts/check-docs.go` — 29 markdown files, links and identifiers OK (both platforms).
  - Four CGO-disabled builds: `sh scripts/build-release.sh 0.1.0` — darwin/arm64, darwin/amd64, linux/arm64, linux/amd64 archives built with -trimpath and `-X main.version=0.1.0`.
  - Archive contents: each tar.gz contains exactly the binary, README.md, and the three specification documents (verified with `tar -tzf`; no provider binaries, no secrets).
  - Version output: extracted darwin/arm64 binary prints `gaebokchi 0.1.0`; `--help` prints the usage contract without launching a provider.
  - Checksums (`shasum -a 256 -c SHA256SUMS` in dist/gaebokchi-0.1.0 — all OK):
    - 6bfdfe55fe90b463721ba6b8873e8d372c8f960af8a22c5f6f1032f0b65b91fe  gaebokchi-darwin-amd64.tar.gz
    - 6c11264525532940fcbcc374c549d31137c2e0859dd5d2a950e4b0ddd31b72df  gaebokchi-darwin-arm64.tar.gz
    - 6c335e3765519b2e90db3d09e823ad7391d2e5b0625a8581b56b69a67c5ff9f8  gaebokchi-linux-amd64.tar.gz
    - 701e7bf1c20aa205c196a90e438677a82fcf016b11c448315c777047f517e073  gaebokchi-linux-arm64.tar.gz
    These replace the pre-remediation archives; the TASK-012/TASK-013 recorded checksums belong to the old artifact at commit `f8b180d` and are not evidence for this candidate.
Platform matrix (corrected tree): darwin/arm64 NATIVE-TESTED (full suite + race); linux/arm64 NATIVE-TESTED (containerized: vet + full suite + race subset + docs check); darwin/amd64 and linux/amd64 CROSS-BUILT ONLY. A first container pull accidentally ran linux/amd64 under emulation and also passed the full suite, but emulated execution is not counted as native evidence.
Live acceptance: NOT RUN — no owner authorization was available in this environment. Live work remains explicit and opt-in; no account query was run by any test or gate above.
Requirement re-verification outcome:
  - Re-checked (16): REQ-006, REQ-009, REQ-010, REQ-015, REQ-016, REQ-018, REQ-020, REQ-021, REQ-023, REQ-024, REQ-032, REQ-039, REQ-044, REQ-045, REQ-049, REQ-050 — each with a re-qualification note naming the corrected-tree evidence; full acceptance scope re-verified per requirement, not just the changed helper.
  - Still unchecked (1): REQ-048 — the owner-account live campaign's metadata-query evidence is preserved for the old artifact, but the provider-display comparison was never performed and the corrected candidate has not been live-tested at all.
Release decision (truthful): code remediation is COMPLETE — every verified audit finding is fixed with regression coverage, and the corrected tree passes the full offline gate on two native platforms. v0.1 is NOT fully release-accepted: REQ-048 remains open.
Owner actions to complete REQ-048 (not waivable by the implementation):
  1. Authorize a live campaign for the corrected candidate binary (checksums above) on a host with both Codex homes and the existing Grok login.
  2. Compare each provider's own quota display with the candidate's rows within the same quota period, read-only, without submitting prompts or changing account/billing state; record timestamps, period boundaries, and legitimate precision/consumption differences. The earlier claim that no headless display command exists does not by itself prove every read-only display inspection requires a model turn; verify rather than assume.
  3. Record the comparison evidence; then (and only then) re-check REQ-048 and the final release gate.
Fixtures: unchanged.
Findings and bounded remediation: none outstanding on the corrective scope. Review stayed bounded as required: one planned assessment (the received audit), one focused remediation pass (EPIC-007), no recursive review loops.
Task commit subject: `test(TASK-021): record final qualification and release decision`
Handoff: stop. Corrected behavior with trustworthy evidence is delivered; the single remaining gate (REQ-048) requires the owner's authorized live comparison. Do not start another feature automatically.
