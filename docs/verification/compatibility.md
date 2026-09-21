# Provider compatibility record

Captured by the TASK-001 qualification probes. External contract evidence for ARCH-12 / REQ-042. Sanitized before storage; see the sanitization table at the end.

## Environment

| Item | Observed value |
|---|---|
| Observed at | 2026-09-22T01:32:38+09:00 through 2026-09-22T01:32:46+09:00 |
| OS/architecture | macOS 26.6.2 (darwin), arm64 |
| Go toolchain (probe only) | go1.26.6 |
| `codex --version` | `codex-cli 0.155.1` (resolved: `~/.local/bin/codex`) |
| `grok --version` | `grok 1.0.40 (eb1a2256660d) [stable]` (resolved: `~/.grok/bin/grok`) |
| Grok source revision consulted | `4247f661689354b831191f11eeeac8424993fe3d` (S-05/S-06; source reference, not a tested CLI version) |

## Codex App Server (codex-cli 0.155.1)

Invocation: `codex app-server` with per-target `CODEX_HOME`; neutral 0700 temporary cwd; environment copied with `OPENAI_API_KEY`, `CODEX_API_KEY` removed and every `CODEX_HOME` entry replaced by the target home.

Frozen exchange (newline-delimited JSON on stdio; no `jsonrpc` field anywhere):

```text
-> {"id":1,"method":"initialize","params":{"clientInfo":{"name":"gaebokchi","title":"Gaebokchi","version":"0.1.0"}}}
<- {"id":1,"result":{"userAgent":"...","codexHome":"...","platformFamily":"unix","platformOs":"macos"}}
-> {"method":"initialized","params":{}}
-> {"id":2,"method":"account/read","params":{"refreshToken":false}}
<- {"id":2,"result":{"account":{"type":"chatgpt","email":"...","planType":"pro"},"requiresOpenaiAuth":true}}
-> {"id":3,"method":"account/rateLimits/read"}
<- {"id":3,"result":{"rateLimits":{...},"rateLimitsByLimitId":{"codex":{...}},...}}
```

Observations:

- Interleaved notification observed between requests: `remoteControl/status/changed` (no `id`). Responses correlate strictly by `id`; notifications must be skipped.
- `account.type == "chatgpt"` with an `email`; `planType` present ("pro" on both homes).
- Rate-limit result carries both a legacy `rateLimits` object (`limitId:"codex"`) and a non-empty `rateLimitsByLimitId` map whose only key is `codex`. Both homes report identical values in the two views.
- Weekly window observed in `primary` (`windowDurationMins: 10080`, integer-encoded `usedPercent`: 37 and 38 on the two homes); `secondary` is JSON `null` on both. `resetsAt` is Unix seconds (present).
- Full field-allowlisted fixture: `internal/provider/codex/testdata/ratelimits_weekly_primary.json` (captured-sanitized). Handshake fixtures: `initialize_response.json`, `account_chatgpt_response.json`.
- Child exits cleanly on stdin EOF; no provider-managed shared leader was observed to be required for `app-server` (the CLI does maintain daemon-related state files in the home; gaebokchi never touches them).

## Grok ACP (grok 1.0.40)

Invocation: `grok --no-auto-update agent stdio`; neutral temporary cwd; environment copied with `XAI_API_KEY`, `GROK_API_KEY` removed.

Frozen exchange (JSON-RPC 2.0 envelopes with `jsonrpc:"2.0"` on requests and responses):

```text
-> {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientInfo":{"name":"gaebokchi","version":"0.1.0"},"clientCapabilities":{"fs":{"readTextFile":false,"writeTextFile":false},"terminal":false}}}
<- {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":1,"authMethods":[{"id":"cached_token",...},{"id":"grok.com",...}],"defaultAuthMethodId":"cached_token",...}}
-> {"jsonrpc":"2.0","id":2,"method":"authenticate","params":{"methodId":"cached_token","_meta":{"headless":true}}}
<- {"jsonrpc":"2.0","id":2,"result":{"_meta":{"auth_mode":"Oidc","subscription_tier":"...",...}}}
-> {"jsonrpc":"2.0","id":3,"method":"_x.ai/billing","params":{}}
<- {"jsonrpc":"2.0","id":3,"result":{"config":{"creditUsagePercent":28.0,"currentPeriod":{"type":"USAGE_PERIOD_TYPE_WEEKLY","start":"...","end":"..."},...},"subscription_tier":"..."}}
```

Observations:

- The internal handler name `x.ai/billing` is exposed on the wire as `_x.ai/billing` (underscore-prefixed extension method), exactly one encoding; no wrapper object around the method name. Sending the bare internal name was NOT tested and is not shipped.
- `params:{}` accepted; response result is an object with a `config` object.
- `creditUsagePercent` is float-encoded (28.0); `currentPeriod.type` uses the string enum `USAGE_PERIOD_TYPE_WEEKLY`; `start`/`end` are RFC 3339. This owner account is on a weekly period — live weekly evidence is achievable (no monthly limitation needed).
- `authenticate` result `_meta` includes an `email`, but per ARCH-08 the ACCOUNT column still uses the configured alias; the email is not read by gaebokchi.
- Many extension notifications interleave (`_x.ai/mcp/servers_updated`, `_x.ai/models/update`, `_x.ai/settings/update`, `_x.ai/announcements/update`); all carry `method` without `id` and must be skipped.
- `initialize` result `_meta.currentWorkingDirectory` echoed the neutral temporary cwd, confirming cwd isolation is visible to the child.
- Child exits cleanly on stdin EOF.

## Environment filtering recorded

Names only, never values: Codex children drop `OPENAI_API_KEY`, `CODEX_API_KEY`, all `CODEX_HOME` entries (then set the target home); Grok children drop `XAI_API_KEY`, `GROK_API_KEY`. No other credential-bearing variables were observed to change outcomes during qualification; `HOME` is preserved for provider keychain access.

## Sanitization table (applied to every captured fixture before it entered this repository)

| Real-world kind | Replacement |
|---|---|
| Codex account emails | `account1@example.com`, `account2@example.com` |
| Grok owner email | `grokowner@example.test` |
| accountId UUIDs | `00000000-0000-4000-8000-0000000000c1`/`...0c2` |
| agentId / agentInstanceId / team_id / installationId | deterministic `00000000-0000-4000-8000-0000000000XX` |
| Hostname | `host.example.test` |
| Temporary probe cwd, local absolute paths | `/tmp/probe-workdir`, `/opt/tool/bin/...` |
| RateLimitResetCredit ids | `RateLimitResetCredit_0000000000000000000000000000000N` |
| userAgent platform string | preserved shape, no identity |

Semantic numbers (percentages, durations, timestamps, period boundaries), field names, types, null placement, and relative ordering are preserved. No auth parameters, redirect URLs, HTTP headers, tokens, or environment dumps were captured.

## Qualification status

| Integration | Status |
|---|---|
| Codex account/rate-limit read via App Server | QUALIFIED (live, two homes) |
| Two isolated Codex homes with distinct CODEX_HOME | QUALIFIED (distinct emails and percentages observed) |
| Grok cached-token headless authenticate | QUALIFIED (live) |
| Grok billing extension `_x.ai/billing` through installed stdio | QUALIFIED (live, weekly enum, real percentage) |
| macOS lifecycle behavior | QUALIFIED (darwin/arm64) |
| Linux lifecycle behavior | NOT RUN (deferred to TASK-012 native Linux matrix by design) |
