package options

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"

	"gaebokchi/internal/domain"
	providerhome "gaebokchi/internal/provider/home"
)

type account struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type cursorAccount struct {
	Name string `yaml:"name"`
}

type accountConfig struct {
	Codex  []account             `yaml:"codex"`
	Grok   []account             `yaml:"grok"`
	Claude []account             `yaml:"claude"`
	Agy    []account             `yaml:"agy"`
	Cursor []cursorAccount       `yaml:"cursor"`
	order  []domain.ProviderKind `yaml:"-"`
}

func configf(path, format string, args ...any) error {
	return &ConfigError{fmt.Sprintf("%s: %s", path, fmt.Sprintf(format, args...))}
}

func loadConfig(path, home string) (accountConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return accountConfig{}, configf(path, "configuration file not found")
		}
		return accountConfig{}, configf(path, "cannot read configuration file")
	}
	var cfg accountConfig
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return accountConfig{}, configf(path, "invalid configuration file")
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return accountConfig{}, configf(path, "configuration must contain one YAML document")
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return accountConfig{}, configf(path, "invalid configuration file")
	}
	if hasMergeKey(&root) {
		return accountConfig{}, configf(path, "YAML merge keys are not supported")
	}
	if len(root.Content) > 0 && root.Content[0].Kind == yaml.MappingNode {
		fields := root.Content[0].Content
		for i := 0; i < len(fields); i += 2 {
			value := fields[i+1]
			for value.Kind == yaml.AliasNode {
				value = value.Alias
			}
			if value.Tag == "!!null" {
				return accountConfig{}, configf(path, "%s must be a list; omit it or use [] to disable", fields[i].Value)
			}
			cfg.order = append(cfg.order, domain.ProviderKind(fields[i].Value))
		}
	}
	for _, group := range []struct {
		name     string
		accounts []account
	}{
		{"codex", cfg.Codex}, {"grok", cfg.Grok}, {"claude", cfg.Claude},
	} {
		seenNames := map[string]bool{}
		var seenPaths []pathIdentity
		for i := range group.accounts {
			a := &group.accounts[i]
			a.Name = domain.NormalizeLabel(a.Name)
			if a.Name == "" {
				return accountConfig{}, configf(path, "%s account %d needs a visible name", group.name, i+1)
			}
			if seenNames[a.Name] {
				return accountConfig{}, configf(path, "%s account names must be unique", group.name)
			}
			seenNames[a.Name] = true
			expanded, err := expandPath(a.Path, home)
			if err != nil {
				return accountConfig{}, configf(path, "%s account %d path must be absolute or start with ~/", group.name, i+1)
			}
			a.Path = expanded
			identity := identifyPath(expanded)
			for _, prior := range seenPaths {
				if identity.same(prior) {
					return accountConfig{}, configf(path, "%s account paths must be unique", group.name)
				}
			}
			seenPaths = append(seenPaths, identity)
		}
	}
	if len(cfg.Agy) > 1 {
		return accountConfig{}, configf(path, "agy supports at most one account")
	}
	for i := range cfg.Agy {
		a := &cfg.Agy[i]
		a.Name = domain.NormalizeLabel(a.Name)
		if a.Name == "" {
			return accountConfig{}, configf(path, "agy account %d needs a visible name", i+1)
		}
		if a.Path != "" {
			expanded, err := expandPath(a.Path, home)
			if err != nil {
				return accountConfig{}, configf(path, "agy account %d path must be absolute or start with ~/", i+1)
			}
			a.Path = expanded
		}
	}
	if len(cfg.Cursor) > 1 {
		return accountConfig{}, configf(path, "cursor supports at most one account")
	}
	for i := range cfg.Cursor {
		a := &cfg.Cursor[i]
		a.Name = domain.NormalizeLabel(a.Name)
		if a.Name == "" {
			return accountConfig{}, configf(path, "cursor account %d needs a visible name", i+1)
		}
	}
	if len(cfg.Codex)+len(cfg.Grok)+len(cfg.Claude)+len(cfg.Agy)+len(cfg.Cursor) == 0 {
		return accountConfig{}, configf(path, "configuration needs at least one account")
	}
	return cfg, nil
}

func hasMergeKey(node *yaml.Node) bool {
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Tag == "!!merge" {
				return true
			}
		}
	}
	for _, child := range node.Content {
		if hasMergeKey(child) {
			return true
		}
	}
	return false
}

// pathIdentity keeps the filesystem identity of an existing directory or the
// deepest existing parent of a missing path. Missing suffixes use a portable
// caseless key; unusable homes retain their raw path for row-level errors.
type pathIdentity struct {
	raw    string
	dir    os.FileInfo
	parent os.FileInfo
	suffix string
}

func (p pathIdentity) same(other pathIdentity) bool {
	if p.raw == other.raw {
		return true
	}
	if p.dir != nil && other.dir != nil {
		return os.SameFile(p.dir, other.dir)
	}
	if p.parent == nil || other.parent == nil || !os.SameFile(p.parent, other.parent) {
		return false
	}
	return missingPathKey(p.suffix) == missingPathKey(other.suffix)
}

// A missing path has no filesystem identity yet. Conservatively treat Unicode
// canonical and case-fold aliases as one target on every supported filesystem.
func missingPathKey(suffix string) string {
	return norm.NFD.String(cases.Fold().String(norm.NFD.String(suffix)))
}

func identifyPath(path string) pathIdentity {
	identity := pathIdentity{raw: path}
	separator := string(os.PathSeparator)
	pending := strings.Split(strings.TrimPrefix(path, separator), separator)
	resolved := separator
	rootInfo, err := os.Stat(resolved)
	if err != nil {
		return identity
	}
	resolvedInfo := rootInfo
	links := 0
	for len(pending) > 0 {
		part := pending[0]
		pending = pending[1:]
		switch part {
		case "", ".":
			continue
		case "..":
			resolved = filepath.Dir(resolved)
			resolvedInfo, err = os.Stat(resolved)
			if err != nil {
				return identity
			}
			continue
		}
		next := filepath.Join(resolved, part)
		info, err := os.Lstat(next)
		if errors.Is(err, os.ErrNotExist) {
			// A missing component cannot be traversed. Remove only no-op
			// components; resolving ".." could invent an existing path.
			rest := []string{part}
			for _, component := range pending {
				if component != "" && component != "." {
					rest = append(rest, component)
				}
			}
			identity.parent = resolvedInfo
			identity.suffix = strings.Join(rest, separator)
			return identity
		}
		if err != nil {
			return identity
		}
		if info.Mode()&os.ModeSymlink == 0 {
			if !info.IsDir() {
				return identity
			}
			resolved = next
			resolvedInfo = info
			continue
		}
		links++
		if links > 40 {
			return identity
		}
		target, err := os.Readlink(next)
		if err != nil {
			return identity
		}
		if filepath.IsAbs(target) {
			resolved = separator
			resolvedInfo = rootInfo
		}
		pending = append(strings.Split(strings.TrimPrefix(target, separator), separator), pending...)
	}
	if providerhome.Inspect(path) == providerhome.OK {
		identity.dir = resolvedInfo
	}
	return identity
}

func expandPath(raw, home string) (string, error) {
	if strings.HasPrefix(raw, "~/") {
		return home + string(os.PathSeparator) + raw[2:], nil
	}
	if !filepath.IsAbs(raw) {
		return "", errors.New("relative path")
	}
	// Keep dot components for filesystem traversal: cleaning first can turn
	// symlink/../home or missing/../home into a different path.
	return raw, nil
}

var defaultProviderOrder = []domain.ProviderKind{
	domain.ProviderCodex,
	domain.ProviderGrok,
	domain.ProviderClaude,
	domain.ProviderAgy,
	domain.ProviderCursor,
}

func buildTargets(cfg accountConfig, o Options) []domain.Target {
	targets := make([]domain.Target, 0, len(cfg.Codex)+len(cfg.Grok)+len(cfg.Claude)+2*len(cfg.Agy)+len(cfg.Cursor))
	order := cfg.order
	if len(order) == 0 {
		order = defaultProviderOrder
	}
	appendAccounts := func(kind domain.ProviderKind, accounts []account, bin string) {
		for i, a := range accounts {
			targets = append(targets, domain.Target{
				ID: fmt.Sprintf("%s:%d", kind, i), Index: len(targets), Provider: kind,
				Home: a.Path, FallbackAccount: a.Name, Bin: bin,
			})
		}
	}
	for _, kind := range order {
		switch kind {
		case domain.ProviderCodex:
			appendAccounts(domain.ProviderCodex, cfg.Codex, o.CodexBin)
		case domain.ProviderGrok:
			appendAccounts(domain.ProviderGrok, cfg.Grok, o.GrokBin)
		case domain.ProviderClaude:
			appendAccounts(domain.ProviderClaude, cfg.Claude, o.ClaudeBin)
		case domain.ProviderAgy:
			if len(cfg.Agy) > 0 {
				for i, pool := range []string{"gemini", "claude/gpt"} {
					targets = append(targets, domain.Target{
						ID: fmt.Sprintf("%s:%d", domain.ProviderAgy, i), Index: len(targets),
						Provider: domain.ProviderAgy, FallbackAccount: pool, Bin: o.AgyBin,
					})
				}
			}
		case domain.ProviderCursor:
			for i, a := range cfg.Cursor {
				targets = append(targets, domain.Target{
					ID: fmt.Sprintf("%s:%d", domain.ProviderCursor, i), Index: len(targets),
					Provider: domain.ProviderCursor, FallbackAccount: a.Name, Bin: o.CursorBin,
				})
			}
		}
	}
	return targets
}
