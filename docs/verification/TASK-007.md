# TASK-007: Add qualified Grok ACP authentication and billing

Task: TASK-007
Status: COMPLETE
Goal and implementation scope: Exact metadata-only Grok wire contract from the TASK-001 probe: `grok --no-auto-update agent stdio` child, ACP initialize with fs/terminal capabilities disabled, cached-token-only authentication with headless metadata, auth-method negotiation matrix, and the single frozen billing extension call `_x.ai/billing` with `params:{}`.
Requirements and architecture sections covered: REQ-031, REQ-032, REQ-033 (primary); ARCH-06, ARCH-08, ARCH-09.
Changed files: internal/provider/grok/provider.go, internal/provider/grok/provider_test.go, docs/verification/TASK-007.md.
Verification (environment: macOS 26.6.2 arm64, go1.26.6):
  - V-GROK (wire): outgoing bytes asserted byte-for-byte against the frozen contract — initialize `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientInfo":{"name":"gaebokchi","version":"0.1.0"},"clientCapabilities":{"fs":{"readTextFile":false,"writeTextFile":false},"terminal":false}}}`, authenticate `{"jsonrpc":"2.0","id":2,"method":"authenticate","params":{"methodId":"cached_token","_meta":{"headless":true}}}`, billing `{"jsonrpc":"2.0","id":3,"method":"_x.ai/billing","params":{}}`. The captured-sanitized initialize/authenticate/billing fixtures drive the fake replies.
  - V-GROK (negotiation): cached_token advertised → authenticate proceeds; api_key only → N/A-path plan with no authenticate on the transcript; interactive-only (grok.com) → AUTH-REQUIRED-path plan; unrecognized methods or none → unsupported plan; protocolVersion 2 → version-mismatch plan; null result / missing protocolVersion / non-numeric version → typed errors.
  - V-GROK (billing): correctly encoded method-not-found (-32601) surfaces as a typed rpcCallError preserving the code (mapped to UNSUPPORTED by the TASK-008 Query); authenticate rejection typed the same way; no session creation, prompt, REST, or browser path exists anywhere in the package.
  - V-SAFETY: server-initiated session/prompt callback is answered with exactly one `{"jsonrpc":"2.0","id":50,"error":{"code":-32601,...}}` rejection and the pending authenticate fails; child env filter drops XAI_API_KEY/GROK_API_KEY markers while preserving HOME (secret markers never reach child output).
  - `go vet ./...`, `go test ./...`, `go test -race ./internal/provider/grok` — pass.
Fixtures: reused captured-sanitized TASK-001 grok fixtures; negotiation variants are synthetic inline payloads.
Findings and bounded remediation: the unsafe-callback test needed the fake to consume the buffered rejection before teardown (200 ms settle), exposing no product defect.
Compatibility/platform limitations: none new; wire names remain frozen to grok 1.0.40 evidence.
Task commit subject: `feat(TASK-007): add Grok ACP billing query transport`
Handoff to next task: TASK-008 completes the normalized Grok Query (weekly/monthly/period semantics, alias behavior, typed failure mapping).
