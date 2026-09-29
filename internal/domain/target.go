// Package domain holds the provider-neutral account-target and result model.
// It must not import provider, RPC, process, or rendering packages.
package domain

// ProviderKind identifies which installed CLI backs a target.
type ProviderKind string

const (
	// ProviderCodex is the OpenAI Codex CLI (App Server protocol), shown
	// under the display name "ChatGPT".
	ProviderCodex ProviderKind = "codex"
	// ProviderGrok is the xAI Grok CLI (ACP stdio protocol).
	ProviderGrok   ProviderKind = "grok"
	ProviderClaude ProviderKind = "claude"
)

// DisplayName returns the AI PROVIDER column text for a provider kind.
func (p ProviderKind) DisplayName() string {
	switch p {
	case ProviderCodex:
		return "ChatGPT"
	case ProviderGrok:
		return "Grok"
	case ProviderClaude:
		return "Claude"
	default:
		return string(p)
	}
}

// Target is one invocation-local query target with a deterministic ID.
type Target struct {
	// ID is the invocation-local identity, e.g. "codex:0" or "grok:0".
	ID string
	// Index is the output row position, assigned in configured order.
	Index int
	// Provider selects the adapter that owns the query.
	Provider ProviderKind
	// Home is the absolute provider state directory, with ~/ expanded.
	Home string
	// FallbackAccount is the configured display alias for this target.
	FallbackAccount string
	// Bin is the configured executable name or explicit path. Absolute
	// resolution happens once per invocation at preflight.
	Bin string
}
