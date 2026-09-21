# Fixture provenance

All files in this directory are captured-sanitized from the TASK-001 live
qualification probes (2026-09-22, macOS arm64, codex-cli 0.155.1 / grok 1.0.40)
unless a later task adds a fixture explicitly tagged synthetic in its own
header or in the owning task's evidence file.

`initialize_response.json` is field-allowlisted: the live response also
carried large `_meta` (models, announcements, host identity) that gaebokchi
never reads; those fields were dropped rather than anonymized. Fields the
adapters consume (`protocolVersion`, `authMethods`,
`defaultAuthMethodId`) are verbatim after sanitization.

Sanitization applied per docs/verification/compatibility.md. No tokens,
headers, redirect URLs, or environment values are stored in this repository.
