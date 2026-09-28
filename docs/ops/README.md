# Operations

Gaebokchi is a one-shot local CLI. It has no independently operated service,
deployment, listener, database, daemon, or application-owned persistent store,
so this delivery scope has no operations runbook.

End-user installation and troubleshooting belong to the repository-root
`README.md`. Provider login, token refresh, caches, and shared processes remain
owned by the installed Codex, Grok, and Claude CLIs. Building, staging, and
publishing releases to GitHub Releases is release engineering and belongs to
the [implementation tips](../implementation-tips/README.md); a published
release is not an operated environment.
