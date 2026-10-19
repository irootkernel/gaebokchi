# Changelog

## v0.1.1 - Unreleased

### Added

- Add weekly usage for the single default `agy` login, with separate `gemini` and `claude/gpt` pool rows.
- Add weekly Included usage for the single default Cursor login, with monthly plans reported as `N/A (monthly)`.
- Add 5-hour ZCode Coding Plan usage for the single default login; the active account needs a stored Coding Plan API key, and OAuth-only logins report `AUTH REQUIRED`.

### Changed

- Follow the provider key and account order in `config.yaml`, preserving that order within each displayed usage period.
- Split plain-table and interactive output into weekly and 5-hour sections while keeping JSON results in one flat array.

## v0.1.0 - 2026-09-29

### Added

- Show provider-reported weekly subscription usage for configured Codex (ChatGPT), Grok, and Claude accounts.
- Provide an interactive terminal view with usage bars and reset countdowns, a plain table, and JSON snapshots.
- Configure multiple provider CLI homes through YAML while keeping credentials with their CLIs and sending no model prompts.
