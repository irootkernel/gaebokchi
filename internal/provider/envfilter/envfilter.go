// Package envfilter centralizes the provider child-environment policy
// captured during TASK-001 qualification: copy the parent environment,
// drop conflicting credential overrides, and set per-target values. It
// is a pure construction policy; values are never logged.
package envfilter

import "strings"

// Rule describes one provider's environment adjustments.
type Rule struct {
	// Drop lists variable names to remove from the inherited
	// environment (credential overrides and parent-session markers).
	Drop []string
	// DropPrefixes lists variable name prefixes to remove from the
	// inherited environment.
	DropPrefixes []string
	// Set maps variable names to replacement values applied after the
	// drops (e.g. the per-target CODEX_HOME).
	Set map[string]string
}

// Codex drops provider API-key overrides and every CODEX_HOME entry; the
// target home is set separately by the caller.
func Codex() Rule {
	return Rule{Drop: []string{"OPENAI_API_KEY", "CODEX_API_KEY", "CODEX_HOME"}}
}

// Grok drops xAI API-key overrides so an ambient key cannot silently
// select a different billing account.
func Grok() Rule {
	return Rule{Drop: []string{"XAI_API_KEY", "GROK_API_KEY"}}
}

// Claude removes overrides that would route a print-mode read away from the
// selected Claude subscription. HOME remains inherited; the caller sets
// CLAUDE_CONFIG_DIR only for non-default homes.
func Claude() Rule {
	return Rule{Drop: []string{
		"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_BASE_URL",
		"ANTHROPIC_AWS_API_KEY", "ANTHROPIC_AWS_BASE_URL",
		"CLAUDE_CODE_OAUTH_TOKEN", "CLAUDE_CONFIG_DIR",
		"CLAUDE_CODE_USE_ANTHROPIC_AWS", "CLAUDE_CODE_USE_BEDROCK",
		"CLAUDE_CODE_USE_FOUNDRY", "CLAUDE_CODE_USE_MANTLE", "CLAUDE_CODE_USE_VERTEX",
		"CLAUDE_CODE_PROVIDER_MANAGED_BY_HOST", "CLAUDE_CODE_SIMPLE",
	}}
}

// Agy drops ambient Antigravity/Jetski/Gemini/ADC overrides and disables
// background CLI auto-updates while preserving HOME for the user's main
// Google sign-in.
func Agy() Rule {
	return Rule{
		Drop: []string{
			"GEMINI_API_KEY", "GOOGLE_GEMINI_BASE_URL", "AGY_ADC_AUTH",
			"GOOGLE_APPLICATION_CREDENTIALS", "GOOGLE_CLOUD_QUOTA_PROJECT",
		},
		DropPrefixes: []string{
			"ANTIGRAVITY_", "JETSKI_", "AGY_",
		},
		Set: map[string]string{
			"AGY_CLI_DISABLE_AUTO_UPDATE": "true",
		},
	}
}

// Cursor drops ambient CURSOR_* overrides and parent NODE_OPTIONS while
// preserving HOME and platform keyring variables for the default Cursor login.
// The caller injects NODE_OPTIONS=--require=<preloadPath>.
func Cursor() Rule {
	return Rule{
		Drop: []string{
			"NODE_OPTIONS",
		},
		DropPrefixes: []string{
			"CURSOR_",
		},
	}
}

// Apply returns the child environment for parent. Later duplicate
// entries for the same name win in the parent list; after Apply each
// adjusted name appears exactly once, at its last position, and
// unadjusted duplicates are preserved as they were.
func Apply(parent []string, r Rule) []string {
	drop := make(map[string]bool, len(r.Drop))
	for _, name := range r.Drop {
		drop[name] = true
	}
	shouldDrop := func(name string) bool {
		if drop[name] {
			return true
		}
		for _, prefix := range r.DropPrefixes {
			if strings.HasPrefix(name, prefix) {
				return true
			}
		}
		return false
	}

	// Resolve the winning entry per name while preserving order.
	type entry struct {
		value string
		idx   int
	}
	winning := map[string]entry{}
	order := make([]string, 0, len(parent))
	for i, kv := range parent {
		name, value, _ := strings.Cut(kv, "=")
		if _, seen := winning[name]; !seen {
			order = append(order, name)
		}
		winning[name] = entry{value: value, idx: i}
	}

	out := make([]string, 0, len(parent))
	emitted := make(map[string]bool, len(r.Set))
	for _, name := range order {
		if shouldDrop(name) {
			continue
		}
		if v, ok := r.Set[name]; ok {
			out = append(out, name+"="+v)
			emitted[name] = true
			continue
		}
		out = append(out, name+"="+winning[name].value)
	}
	for name, v := range r.Set {
		if emitted[name] {
			continue // replaced in place above
		}
		out = append(out, name+"="+v)
	}
	return out
}
