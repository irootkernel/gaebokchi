# TASK-001: Qualify metadata-only provider contracts

Task: TASK-001
Status: COMPLETE
Goal and implementation scope: Remove feasibility risk by proving both installed CLIs expose prompt-free metadata over stdio: Codex App Server account/quota reads for two isolated homes, Grok ACP cached-token authentication plus the billing extension.
Requirements and architecture sections covered: REQ-042; ARCH-06, ARCH-07, ARCH-08, ARCH-09, ARCH-12, ARCH-15.
Changed files: docs/verification/compatibility.md, docs/verification/TASK-001.md, internal/provider/codex/testdata/*.json, internal/provider/grok/testdata/*.json, plus the initial specification import (.gitignore, docs/requirement.md, docs/architecture.md, docs/plan.md).
Verification:
  - V-CONTRACT: disposable Go probe (built in /tmp, never committed) driving the exact candidate envelopes. All three targets returned valid structured responses. Environment: macOS 26.6.2 arm64, go1.26.6.
  - `codex --version` -> `codex-cli 0.155.1`; `grok --version` -> `grok 1.0.40 (eb1a2256660d) [stable]`.
  - Codex home 1 (~/.codex): initialize ok; account type=chatgpt; rateLimitsByLimitId.codex.primary usedPercent=37, windowDurationMins=10080.
  - Codex home 2 (~/.codex-hsy): same flow; usedPercent=38, windowDurationMins=10080.
  - Grok: initialize (protocolVersion 1, cached_token advertised), authenticate cached_token+headless ok, `_x.ai/billing` returned config.creditUsagePercent=28 with currentPeriod.type=USAGE_PERIOD_TYPE_WEEKLY. NOT method-not-found.
  - V-SAFETY (probe scope): no session/new, no prompt, no browser, no login/logout, no billing mutation, no credential file reads by the probe itself; environment filtered (OPENAI_API_KEY, CODEX_API_KEY, CODEX_HOME replaced for Codex; XAI_API_KEY, GROK_API_KEY removed for Grok); neutral temporary cwd; direct child terminated cleanly on stdin EOF without killing shared leaders.
Fixtures: captured-sanitized (emails, account/agent/team UUIDs, hostname, installation id, local paths, working directory replaced with deterministic placeholders; semantic numbers, types, and ordering preserved). See `docs/verification/compatibility.md` sanitization table. Synthetic edge-case fixtures are added with later tasks and tagged synthetic.
Findings and bounded remediation: none blocking. Percentages arrive as JSON numbers (Codex integer 37; Grok float 28.0) matching the decode-as-number policy. Interleaved notifications observed on both providers (Codex `remoteControl/status/changed`; Grok `_x.ai/models/update`, `_x.ai/settings/update`, `_x.ai/announcements/update`, `_x.ai/mcp/servers_updated`) — correlation must skip them, confirming ARCH-06.
Compatibility/platform limitations: qualifications below were run on darwin/arm64 only; linux qualification is deferred to TASK-012 by design.
Task commit subject: `docs(TASK-001): qualify Codex and Grok metadata contracts`
Handoff to next task: frozen observed envelopes in docs/verification/compatibility.md and captured-sanitized testdata fixtures; TASK-002 bootstraps the Go module, domain types, and target policy against these contracts.
