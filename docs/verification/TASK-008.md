# TASK-008: Complete Grok weekly semantics and alias behavior

Task: TASK-008
Status: COMPLETE
Goal and implementation scope: Complete Grok Query around the billing helper with the pure ARCH-08 result mapping: explicit weekly enum required before reading creditUsagePercent, monthly/absent/unknown period and legacy-only payloads mapped truthfully, optional reset metadata preserved, configured alias on every outcome.
Requirements and architecture sections covered: REQ-013, REQ-014, REQ-015, REQ-034, REQ-035, REQ-036 (primary); ARCH-04, ARCH-08, ARCH-11.
Changed files: internal/provider/grok/normalize.go, internal/provider/grok/provider.go (Query lifecycle + failure classification), internal/provider/grok/normalize_test.go, docs/verification/TASK-008.md.
Verification (environment: macOS 26.6.2 arm64, go1.26.6):
  - V-GROK / V-QUOTA (pure normalization): 21-case table — captured weekly fixture shape (28.0 with period end, prepaid/tier fields ignored), valid zero, integer encoding, boundary 100; monthly → N/A (monthly) with the percentage never becoming a weekly value; period absent (plain and legacy monthlyLimit/used-only) → N/A (period); unknown enum and null type → N/A (period); weekly with credit absent/null → N/A (no data); string-typed/negative/over-100 credit → PROTOCOL ERROR (no clamping); config null/absent → N/A (no data); config wrong type, result not an object, period/type wrong types → PROTOCOL ERROR; malformed optional period end discarded while the verified percentage survives; legacy monthly credits never converted.
  - V-GROK (lifecycle): full captured-fixture Query returns ok 28% with weekly period, source grok_acp_billing, and reset metadata; monthly Query is exit-clean non-applicable; API-key-only / interactive-only / authenticate-rejected / method-not-found / version-mismatch paths map to N/A (API key), AUTH REQUIRED, AUTH REQUIRED, UNSUPPORTED, and UNSUPPORTED; the configured alias is the ACCOUNT value on success, auth failure, and unavailable outcomes — the authenticate `_meta.email` and billing payloads are never identity sources; missing executable → NOT INSTALLED with alias intact; hung server → TIMEOUT within budget with bounded cleanup.
  - `go vet ./...`, `go test ./...`, `go test -race ./internal/provider/grok` — pass.
Fixtures: reused captured-sanitized TASK-001 grok fixtures; matrix payloads synthetic inline.
Findings and bounded remediation: a nil-session early-return path (non-cached-token plans) was dereferenced by the deferred close; restructured so plan-based outcomes return before the session defer and the unreachable post-session plan switch was removed.
Compatibility/platform limitations: none new.
Task commit subject: `feat(TASK-008): normalize Grok weekly billing usage`
Handoff to next task: EPIC-004 closes with both complete provider Query implementations; TASK-009 composes the bounded snapshot runner.
