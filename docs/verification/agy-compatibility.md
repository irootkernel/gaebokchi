# agy compatibility and qualification record

## Current finding

On 2026-10-17, bounded metadata-only probes against installed **`agy 1.4.0`** on
`darwin/arm64` qualified the structured `/usage` JSON contract for the single
default CLI login (`REQ-060`, `ARCH-18`). Under the owner's 2026-10-17 scope
decision, `gaebokchi` queries only the single main (default) `agy` login and
renders its two weekly model pools (`gemini` and `claude/gpt`) as two weekly
rows in the existing output formats. Multi-account `agy` isolation and non-weekly
output sections are out of scope.

The [dated preflight](provider-extension-preflight-2026-10-16.md) records the
initial 2026-10-16 static discovery and the 2026-10-17 live metadata and
absent-login probes.

## Qualified invocation and safety contract

- **Executable and version guard:** Resolve `--agy-bin` (default `agy`) once per
  invocation and run `agy --version` inside the shared target timeout budget
   before `/usage`. Require version **`>= 1.2.15`** (changelog v1.1.11 added
  turn-free JSON responses for `-p "/usage"` and rejected interactive-only slash
  commands in print mode; v1.2.15 normalized piped/JSON reset timestamps to UTC).
  Older or unparseable versions return `StatusUnsupported` (`UNSUPPORTED`)
  without invoking `/usage`.
- **Quota command:**
  ```sh
  agy --print /usage --output-format json --print-timeout 15s
  ```
  Run with stdin closed (`os.DevNull`), an empty private temporary working
  directory (`0700`), and the outer per-target context deadline.
- **Zero-turn / read-only verification:** Observed live execution exited `0` in
  ~5.2s with `status: "SUCCESS"`, `num_turns: 0`, `usage.total_tokens: 0`,
  `conversation_id: ""`, empty `errors`, and no created files in the temporary
  working directory.
- **Child environment policy:**
  - Preserve `HOME` and platform keyring context so `agy` reads its default
    CLI-managed login.
  - Set `AGY_CLI_DISABLE_AUTO_UPDATE=true` ([S-23](../architecture/README.md#s-23)).
  - Strip all `ANTIGRAVITY_*` (including `ANTIGRAVITY_LS_ADDRESS`,
    `ANTIGRAVITY_CSRF_TOKEN`, `ANTIGRAVITY_APP_DATA_DIR`, and
    `ANTIGRAVITY_EDITOR_APP_ROOT` exported when running inside an `agy` host
    session), `JETSKI_*`, and `AGY_*` variables from the parent environment.
  - Strip credential and routing overrides: `GEMINI_API_KEY`,
    `GOOGLE_GEMINI_BASE_URL`, `AGY_ADC_AUTH`,
    `GOOGLE_APPLICATION_CREDENTIALS`, and `GOOGLE_CLOUD_QUOTA_PROJECT`.

## Sanitized captured response fixture (`agy 1.4.0`, `darwin/arm64`)

 Provenance: captured live on 2026-10-17 from `agy 1.4.0` using the qualified
command above; field-allowlisted with no account identifiers or credentials.

```json
{
  "type": "result",
  "status": "SUCCESS",
  "num_turns": 0,
  "conversation_id": "",
  "usage": {
    "input_tokens": 0,
    "output_tokens": 0,
    "total_tokens": 0,
    "cache_read_tokens": 0,
    "cache_write_tokens": 0,
    "reasoning_tokens": 0
  },
  "command": {
    "name": "usage",
    "message": "Model quota retrieved",
    "data": {
      "groups": [
        {
          "name": "Gemini Models",
          "description": "Models within this group: Gemini Flash, Gemini Pro",
          "quotas": [
            {
              "id": "gemini-weekly",
              "label": "Weekly",
              "window": "weekly",
              "remaining_fraction": 0.2584281265735626,
              "reset_time": "2026-10-21T15:42:40Z"
            },
            {
              "id": "gemini-5h",
              "label": "5-Hour",
              "window": "5h",
              "remaining_fraction": 0.8503220677375793,
              "reset_time": "2026-10-16T18:01:19Z"
            }
          ]
        },
        {
          "name": "Claude and GPT models",
          "description": "Models within this group: Claude Opus, Claude Sonnet, GPT-OSS",
          "quotas": [
            {
              "id": "3p-weekly",
              "label": "Weekly",
              "window": "weekly",
              "remaining_fraction": 0.8723080158233643,
              "reset_time": "2026-10-23T15:13:44Z"
            },
            {
              "id": "3p-5h",
              "label": "5-Hour",
              "window": "5h",
              "remaining_fraction": 0.7446160912513733,
              "reset_time": "2026-10-16T20:13:44Z"
            }
          ]
        }
      ]
    }
  }
}
```

## Weekly pool mapping and conversion

One shared `agy` query per `gaebokchi` invocation populates two weekly rows under
provider `agy` (`DisplayName: "agy"`):

| Target ID | Row label (`ACCOUNT` column) | Matched quota (`command.data.groups[].quotas[]`) | Used percentage calculation | Observed sample |
|---|---|---|---|---|
| `agy:0` | `gemini` | `id == "gemini-weekly"` and `window == "weekly"` | `100 * (1 - remaining_fraction)` via exact decimal token math | `74.15718734264374%` (`74.2%`, `↻ 4d 3h`) |
| `agy:1` | `claude/gpt` | `id == "3p-weekly"` and `window == "weekly"` | `100 * (1 - remaining_fraction)` via exact decimal token math | `12.76919841766357%` (`12.8%`, `↻ 6d 3h`) |

Quotas with `window: "5h"` (`gemini-5h`, `3p-5h`) are ignored because
`gaebokchi` reports weekly usage only.

## Absent-login and failure behavior

- **Unauthenticated behavior (observed 2026-10-17):** When no login is present,
  `agy --print /usage --output-format json --print-timeout 15s` does not exit
  immediately even with stdin closed, `CI=true`, or
  `AGY_CLI_NONINTERACTIVE_HEADLESS=1`. Instead, it writes an OAuth login prompt
  to stderr (`Authentication required. Please visit the URL to log in: ...
  Waiting for authentication (timeout 60s)...`) and waits up to 60 seconds.
- **Adapter fast-fail requirement:** The adapter must inspect bounded stderr while
  the child runs (or upon early termination) and immediately terminate the child
  process group when `Authentication required` appears on stderr, returning
  `StatusAuthRequired` (`AUTH REQUIRED`) for both `agy` rows without waiting for
  the 60-second OAuth timeout.
- **Data-directory variables:** Login-free probes on 2026-10-17 confirmed that
  `ANTIGRAVITY_APP_DATA_DIR` and `JETSKI_APP_DATA_DIR` are ignored as input
  selectors by `agy 1.4.0` (the binary unconditionally uses
  `$HOME/.gemini/antigravity-cli` and exports those variables to children).
  Because `gaebokchi` supports only the single main `agy` login, no `HOME`
  relocation or multi-account selector is used.

Sources: [S-17](../architecture/README.md#s-17),
[S-18](../architecture/README.md#s-18), [S-19](../architecture/README.md#s-19),
[S-22](../architecture/README.md#s-22), and
[S-23](../architecture/README.md#s-23). See
[ARCH-18](../architecture/README.md#arch-18) and the
[execution dossier](../todo/TODO-AGY-MULTI-ACCOUNT-SUPPORT.md).
