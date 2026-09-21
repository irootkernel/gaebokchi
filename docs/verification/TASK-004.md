# TASK-004: Implement framed RPC with provider-specific codecs

Task: TASK-004
Status: COMPLETE
Goal and implementation scope: Reusable stdio JSON transport: bounded newline framing (LF/CRLF, fragmentation, coalescing, 4 MiB ceiling, UTF-8 validation), numeric request-ID correlation ignoring notifications and unknown/duplicate ids, single outstanding request, dialect codecs with outbound allowlists, server-initiated-request rejection, context cancellation aborting the transport. Includes the scripted fake-child harness in internal/testutil.
Requirements and architecture sections covered: REQ-016, REQ-017, REQ-018, REQ-019, REQ-043 (primary); ARCH-06, ARCH-09, ARCH-11.
Changed files: internal/rpc/frame.go, internal/rpc/codec.go, internal/rpc/client.go, internal/rpc/frame_test.go, internal/rpc/codec_test.go, internal/rpc/client_test.go, internal/rpc/fakechild_test.go, internal/testutil/fake.go, internal/testutil/fake_helper_test.go, docs/verification/TASK-004.md.
Verification (environment: macOS 26.6.2 arm64, go1.26.6):
  - V-RPC (framing): `go test ./internal/rpc -run 'TestFrame'` — LF/CRLF/empty-line skipping, piecewise fragmented and coalesced messages, exact 4 MiB boundary acceptance and rejection including delimiter accounting, invalid UTF-8 rejection, truncated final frame rejection.
  - V-RPC (dialects): exact-byte assertions — Codex requests/notifications/error-responses carry no `jsonrpc`; ACP messages always carry `"jsonrpc":"2.0"`; `initialized` encodes as notification; `_x.ai/billing` and `account/rateLimits/read` (params omitted when nil) byte-exact; both allowlists reject session/prompt/turn-start/x.ai-billing-unprefixed/initialized-as-ACP etc.
  - V-RPC (correlation): notification-before-reply and unknown-id frames never satisfy the pending call; duplicate response ids ignored; remote errors surfaced as typed RPCError; `result:null` preserved verbatim; EOF and malformed/oversized/banner frames fail pending calls immediately; top-level array (batch) rejected; server-initiated requests get one dialect-correct -32601 rejection echoing the exact id (ACP with jsonrpc, Codex without) and then abort the connection; context deadline unblocks reads, writes (blocked-writer case), and fires the abort hook.
  - V-RPC (fake child): real subprocess pipes through internal/testutil scripted fakes — fragmented reply across two writes, stderr flood alongside a valid reply, banner line as protocol error, >4 MiB frame killing the connection, hung server timing out with bounded cleanup, CRLF replies with duplicate id not resatisfying a second call.
  - V-SAFETY (transport scope): no capability execution path exists — server requests are only rejected; outbound methods are gated per codec; rejection writes are bounded (500 ms watchdog) so a full pipe cannot wedge the read loop.
  - `go vet ./...` pass; `go test ./...` pass; `go test -race ./internal/rpc -count=3` pass; `go test ./internal/rpc -count=5` pass.
Fixtures: no new provider fixtures (pure transport tests use inline literals and the compiled fake harness).
Findings and bounded remediation: two races found and fixed before completion — (1) rejection written from the read loop could block forever on an unread pipe (now bounded); (2) a reply routed just before connection death could be lost to the death signal (the wait loop now checks the reply channel first). The previously deadlocking single-outstanding test was rewritten with a reading server side.
Compatibility/platform limitations: none new.
Task commit subject: `feat(TASK-004): add bounded stdio RPC codecs`
Handoff to next task: EPIC-002 closes with a tested process+RPC boundary and fake harness; TASK-005 builds the isolated Codex handshake on it.
