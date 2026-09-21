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
	ProviderGrok ProviderKind = "grok"
)

// DisplayName returns the AI PROVIDER column text for a provider kind.
func (p ProviderKind) DisplayName() string {
	switch p {
	case ProviderCodex:
		return "ChatGPT"
	case ProviderGrok:
		return "Grok"
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
	// Home is the absolute normalized Codex home; empty for Grok.
	Home string
	// FallbackAccount is the display label used when the provider cannot
	// supply one: the Codex home basename or the configured Grok alias.
	FallbackAccount string
	// Bin is the configured executable name or explicit path. Absolute
	// resolution happens once per invocation at preflight.
	Bin string
}
