# TASK-031: Align release documentation and archive contents

Task: TASK-031
Status: COMPLETE
Goal and implementation scope: Reconcile the current public release documents and package the project license plus notices for bundled components.
Requirements and architecture sections covered: REQ-046, REQ-049, REQ-050; ARCH-13, ARCH-14.
Changed files: README, documentation indexes and current-state summaries, release build and archive checker, their tests, LICENSE packaging, THIRD_PARTY_NOTICES.txt.

Verification on macOS darwin/arm64 with Go 1.27.1, based on HEAD `aec4fd14dc56794327a45aee790098813defb845` plus this task's uncommitted changes:

- `make test`: exit 0; formatting, vet, lint (0 issues), build, documentation check, race-enabled unit and integration suites, and E2E (32/32) passed.
- `docker run --rm --platform linux/arm64 -v "$PWD":/src -v "$(go env GOMODCACHE)":/go/pkg/mod -w /src golang:1.27.1-bookworm sh -c 'go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 && make test'`: exit 0 on native linux/arm64; the same prepare, unit, integration, and 32/32 E2E stages passed. The gate itself set GOPROXY and GOSUMDB to `off`.
- `go run scripts/check-docs.go`: exit 0; 53 Markdown files, links and identifiers OK.
- `go test ./scripts -count=1`: exit 0; archive allowlist and negative fixtures passed.
- `sh scripts/build-release.sh v0.1.0`: exit 0; four platform archives and SHA256SUMS staged under ignored `dist/gaebokchi-v0.1.0/`.
- `GAEBOKCHI_REQUIRE_STAGED_RELEASE=1 go run github.com/onsi/ginkgo/v2/ginkgo --fail-on-empty --label-filter=staged-release ./scripts`: exit 0; 1/1 selected staged-release spec passed, including exact members and byte-identical packaged text.
- `$aquarium:docs-setup` inspector: `aquarium-docs-inspection/v2` structural status `conforming`, zero findings. This is structural evidence, not a semantic or product acceptance verdict.
- `codex review --uncommitted`: exit 0; a separate read-only review of the package and documentation change reported no actionable finding. Mulgae capture was rejected by its enabled content policy before provider execution, so no Mulgae verdict is claimed.

The staged archives above came from a dirty worktree and are package-layout evidence only. TASK-033 must rebuild from the final clean commit, verify `vcs.revision` and `vcs.modified=false`, and record replacement checksums. Live provider comparison is not a TASK-031 claim.

Findings and bounded remediation: the original release archives omitted LICENSE, and the public README omitted the bundled YAML module from its third-party summary. The archive now includes LICENSE and THIRD_PARTY_NOTICES.txt, and the member checker requires both. Historical roadmap and TODO text describing current features as future work was corrected without changing past verification records.
Task commit subject: `build(TASK-031): align public release documentation and archives`.
Handoff: TASK-032 owns current YAML and Claude acceptance. TASK-033 owns final platform, artifact, review, and REQ-048 gates.
