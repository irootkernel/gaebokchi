package options

import (
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"gaebokchi/internal/domain"
)

func dereference(node *yaml.Node) *yaml.Node {
	for node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	return node
}

// validateAliases checks the original YAML types before decoded strings are
// used: yaml.v3 also accepts numbers and booleans when decoding into strings.
func validateAliases(path string, node *yaml.Node) error {
	node = dereference(node)
	if node.Kind != yaml.MappingNode {
		return configf(path, "aliases must be a mapping; omit it or use {} to disable")
	}
	seen := map[string]bool{}
	for i := 0; i < len(node.Content); i += 2 {
		key := dereference(node.Content[i])
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" ||
			!slices.Contains(defaultProviderOrder, domain.ProviderKind(key.Value)) {
			return configf(path, "aliases key %q is not a known provider", key.Value)
		}
		if seen[key.Value] {
			return configf(path, "aliases.%s is repeated", key.Value)
		}
		seen[key.Value] = true
		value := dereference(node.Content[i+1])
		if value.Kind != yaml.SequenceNode {
			return configf(path, "aliases.%s must be a list of strings; omit it or use [] to disable", key.Value)
		}
		for _, entry := range value.Content {
			entry = dereference(entry)
			if entry.Kind != yaml.ScalarNode || entry.Tag != "!!str" {
				return configf(path, "aliases.%s entries must be strings", key.Value)
			}
			if strings.TrimSpace(entry.Value) == "" {
				return configf(path, "aliases.%s entries must not be blank", key.Value)
			}
		}
	}
	return nil
}

func (cfg accountConfig) hasAccounts(kind domain.ProviderKind) bool {
	switch kind {
	case domain.ProviderCodex:
		return len(cfg.Codex) > 0
	case domain.ProviderGrok:
		return len(cfg.Grok) > 0
	case domain.ProviderClaude:
		return len(cfg.Claude) > 0
	case domain.ProviderAgy:
		return len(cfg.Agy) > 0
	case domain.ProviderCursor:
		return len(cfg.Cursor) > 0
	case domain.ProviderZCode:
		return len(cfg.ZCode) > 0
	default:
		return false
	}
}

func (cfg accountConfig) selectionNames(kind domain.ProviderKind) []string {
	names := []string{strings.ToLower(kind.DisplayName())}
	for _, alias := range cfg.Aliases[string(kind)] {
		names = append(names, strings.ToLower(alias))
	}
	return names
}

// matchProviders resolves one name without changing account or provider order.
// Disabled providers never enter the selection, even when they have aliases.
func (cfg accountConfig) matchProviders(name string) map[domain.ProviderKind]bool {
	want := strings.ToLower(name)
	matches := map[domain.ProviderKind]bool{}
	for _, kind := range defaultProviderOrder {
		if !cfg.hasAccounts(kind) {
			continue
		}
		for _, candidate := range cfg.selectionNames(kind) {
			if candidate == want {
				matches[kind] = true
				break
			}
		}
	}
	return matches
}
