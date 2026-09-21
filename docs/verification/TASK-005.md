# TASK-005: Add isolated Codex authentication and account reads

Task: TASK-005
Status: COMPLETE
Goal and implementation scope: Codex child specification with per-target CODEX_HOME environment policy (centralized in internal/provider/envfilter for Grok reuse), initialize/initialized handshake ordering, account/read with refreshToken:false, and typed auth mapping (chatgpt/apiKey/absent/unknown). Home inspection and executable resolution helpers.
Requirements and architecture sections covered: REQ-025, REQ-026, REQ-027, REQ-039 (primary); ARCH-03, ARCH-05, ARCH-07, ARCH-09.
Changed files: internal/provider/envfilter/envfilter.go, internal/provider/envfilter/envfilter_test.go, internal/provider/codex/provider.go, internal/provider/codex/provider_test.go, internal/testutil/fake.go (Wait now skips id-less notification lines and reports every consumed line for transcript assertions), docs/verification/TASK-005.md.
Verification (environment: macOS 26.6.2 arm64, go1.26.6):
  - V-CODEX (auth scope): `go test ./internal/provider/codex` — ok. Handshake transcript asserted from the fake's captured stderr: initialize (with title Gaebokchi) → wait for response → initialized notification → account/read with `"refreshToken":false`; no jsonrpc marker; no session/thread/turn/prompt methods; account mapping table: captured fixture `account_chatgpt_home1.json` → AuthChatGPT + email; apiKey → AuthAPIKey; unknown type → AuthUnknownType (never treated as ChatGPT or apiKey); account:null and result:null → AuthAbsent; chatgpt without email keeps fallback; control-character emails rejected while type remains chatgpt; typed rpcCallError for remote rejections (no English-text guessing).
  - V-IDENTITY (isolation scope): two simultaneous fake sessions over two homes answered in interleaved order keep distinct emails — no shared or cached identity state; each session owns its connection.
  - V-SAFETY (env scope): `go test ./internal/provider/envfilter` — Codex rule drops OPENAI_API_KEY/CODEX_API_KEY/every CODEX_HOME entry and sets the target home; Grok rule drops XAI_API_KEY/GROK_API_KEY; parent slice never mutated; drop+replace ordering deterministic. Pure construction policy, reusable by the Grok adapter.
  - Home inspection: HomeOK/HomeMissing/HomeInvalid for existing dir, missing path, and non-directory file.
  - `go vet ./...`, `go test ./...`, `go test -race ./internal/provider/...` — all pass.
Fixtures: reused captured-sanitized TASK-001 fixtures (account_chatgpt_home1.json); no new fixtures.
Findings and bounded remediation: fake harness hardened during this task — request slots now skip notification lines so scripts align with the real handshake, and every consumed line is echoed for transcript assertions. envfilter drop-then-set ordering bug (dropped name never re-added) found by test and fixed.
Compatibility/platform limitations: none new.
Task commit subject: `feat(TASK-005): add isolated Codex account handshake`
Handoff to next task: TASK-006 completes rateLimits/read, the deterministic quota bucket/window selection, and the full Query lifecycle on this session layer.
