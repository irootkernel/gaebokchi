// Package options parses the small v0.1 command surface and builds the
// ordered query targets. It never reads provider environment variables:
// the ambient CODEX_HOME cannot influence the default account list.
package options

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gaebokchi/internal/domain"
)

// Defaults and limits for the v0.1 surface.
const (
	DefaultCodexBin    = "codex"
	DefaultGrokBin     = "grok"
	DefaultClaudeBin   = "claude"
	DefaultGrokAccount = "default"
	DefaultTimeout     = 20 * time.Second
	minTimeout         = 1 * time.Second
	maxTimeout         = 120 * time.Second
)

// Options is the parsed command configuration.
type Options struct {
	// CodexHomes holds absolute, cleaned Codex home paths in flag order.
	// When no --codex-home flag is supplied it holds the two defaults.
	CodexHomes  []string
	CodexBin    string
	GrokBin     string
	ClaudeBin   string
	GrokAccount string
	Timeout     time.Duration
	Help        bool
}

// UsageError marks parse failures that must exit with code 2 before any
// provider process is launched.
type UsageError struct{ msg string }

func (e *UsageError) Error() string { return e.msg }

func usagef(format string, args ...any) error {
	return &UsageError{msg: fmt.Sprintf(format, args...)}
}

// IsUsageError reports whether err is a command-line usage failure.
func IsUsageError(err error) bool {
	var ue *UsageError
	return errors.As(err, &ue)
}

type flagKind int

const (
	kindString flagKind = iota
	kindDuration
	kindBool
)

// flagSpec describes one recognized flag.
type flagSpec struct {
	name   string
	kind   flagKind
	repeat bool
	set    func(o *Options, value string) error
}

func setScalar(field func(*Options) *string, flagName string) func(*Options, string) error {
	return func(o *Options, v string) error {
		if v == "" {
			return usagef("--%s must not be empty", flagName)
		}
		*field(o) = v
		return nil
	}
}

func specs() map[string]*flagSpec {
	return map[string]*flagSpec{
		"codex-home": {name: "codex-home", kind: kindString, repeat: true, set: func(o *Options, v string) error {
			if v == "" {
				return usagef("--codex-home must not be empty")
			}
			o.CodexHomes = append(o.CodexHomes, v)
			return nil
		}},
		"codex-bin":  {name: "codex-bin", kind: kindString, set: setScalar(func(o *Options) *string { return &o.CodexBin }, "codex-bin")},
		"grok-bin":   {name: "grok-bin", kind: kindString, set: setScalar(func(o *Options) *string { return &o.GrokBin }, "grok-bin")},
		"claude-bin": {name: "claude-bin", kind: kindString, set: setScalar(func(o *Options) *string { return &o.ClaudeBin }, "claude-bin")},
		"grok-account": {name: "grok-account", kind: kindString, set: func(o *Options, v string) error {
			// The alias is validated through the display-label policy
			// before any provider can launch: an alias with no
			// displayable characters is a usage error, and the stored
			// value is the normalized label the table will show.
			normalized := domain.NormalizeLabel(v)
			if normalized == "" {
				return usagef("--grok-account must not be blank: it contains no displayable characters")
			}
			o.GrokAccount = normalized
			return nil
		}},
		"timeout": {name: "timeout", kind: kindDuration, set: func(o *Options, v string) error {
			d, err := parseTimeout(v)
			if err != nil {
				return err
			}
			o.Timeout = d
			return nil
		}},
		"help": {name: "help", kind: kindBool, set: func(o *Options, _ string) error { o.Help = true; return nil }},
	}
}

func parseTimeout(v string) (time.Duration, error) {
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, usagef("invalid --timeout value %q: %v", v, err)
	}
	if d < minTimeout || d > maxTimeout {
		return 0, usagef("--timeout must be between 1s and 120s, got %v", d)
	}
	return d, nil
}

// Parse parses argv (without the program name) against the v0.1 surface.
func Parse(args []string) (Options, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Options{}, usagef("cannot resolve user home directory: %v", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		return Options{}, usagef("cannot resolve working directory: %v", err)
	}
	return parse(args, home, wd)
}

// parse is the pure, testable core of Parse.
func parse(args []string, homeDir, wd string) (Options, error) {
	table := specs()
	seen := map[string]bool{}

	o := Options{
		CodexBin:    DefaultCodexBin,
		GrokBin:     DefaultGrokBin,
		ClaudeBin:   DefaultClaudeBin,
		GrokAccount: DefaultGrokAccount,
		Timeout:     DefaultTimeout,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			return Options{}, usagef("unexpected positional argument %q", arg)
		}
		name := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
		value := ""
		hasValue := false
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			value, hasValue = name[eq+1:], true
			name = name[:eq]
		}
		if name == "h" {
			name = "help"
		}
		spec := table[name]
		if spec == nil {
			return Options{}, usagef("unknown flag %q", arg)
		}
		if !spec.repeat && seen[spec.name] {
			return Options{}, usagef("flag --%s may be supplied only once", name)
		}
		if spec.kind == kindBool {
			if hasValue {
				return Options{}, usagef("flag --%s does not take a value", name)
			}
			seen[spec.name] = true
			if err := spec.set(&o, ""); err != nil {
				return Options{}, err
			}
			continue
		}
		if !hasValue {
			if i+1 >= len(args) {
				return Options{}, usagef("flag --%s requires a value", name)
			}
			i++
			value = args[i]
		}
		seen[spec.name] = true
		if err := spec.set(&o, value); err != nil {
			return Options{}, err
		}
	}

	// Normalize Codex homes: tilde expansion, relative resolution, cleanup.
	homes := o.CodexHomes
	if len(homes) == 0 {
		homes = []string{filepath.Join(homeDir, ".codex"), filepath.Join(homeDir, ".codex-hsy")}
	}
	normalized := make([]string, 0, len(homes))
	for _, h := range homes {
		n, err := normalizeHome(h, homeDir, wd)
		if err != nil {
			return Options{}, err
		}
		normalized = append(normalized, n)
	}
	if err := rejectDuplicates(normalized); err != nil {
		return Options{}, err
	}
	o.CodexHomes = normalized
	return o, nil
}

// normalizeHome expands a leading current-user ~, resolves relatives
// against the invocation directory, and cleans the path. It never creates
// directories and does not implement ~otheruser or shell expansion.
func normalizeHome(raw, homeDir, wd string) (string, error) {
	if raw == "" {
		return "", usagef("--codex-home must not be empty")
	}
	expanded := raw
	switch {
	case raw == "~":
		expanded = homeDir
	case strings.HasPrefix(raw, "~/"):
		expanded = filepath.Join(homeDir, strings.TrimPrefix(raw, "~/"))
	case strings.HasPrefix(raw, "~"):
		return "", usagef("unsupported home path %q: only the current user's ~ is expandable", raw)
	}
	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(wd, expanded)
	}
	return filepath.Clean(expanded), nil
}

// rejectDuplicates errors when two normalized homes refer to the same
// location, including through existing symlink aliases.
func rejectDuplicates(homes []string) error {
	canon := make(map[string]string, len(homes))
	for _, h := range homes {
		key := h
		if resolved, err := filepath.EvalSymlinks(h); err == nil {
			key = resolved
		}
		if prev, dup := canon[key]; dup {
			return usagef("duplicate codex home %q (same location as %q)", h, prev)
		}
		canon[key] = h
	}
	return nil
}

// Targets builds the ordered query targets from parsed options.
func (o Options) Targets() []domain.Target {
	targets := make([]domain.Target, 0, len(o.CodexHomes)+2)
	for i, home := range o.CodexHomes {
		targets = append(targets, domain.Target{
			ID:              fmt.Sprintf("codex:%d", i),
			Index:           i,
			Provider:        domain.ProviderCodex,
			Home:            home,
			FallbackAccount: filepath.Base(home),
			Bin:             o.CodexBin,
		})
	}
	targets = append(targets, domain.Target{
		ID:              "grok:0",
		Index:           len(o.CodexHomes),
		Provider:        domain.ProviderGrok,
		FallbackAccount: o.GrokAccount,
		Bin:             o.GrokBin,
	})
	targets = append(targets, domain.Target{
		ID:              "claude:0",
		Index:           len(o.CodexHomes) + 1,
		Provider:        domain.ProviderClaude,
		FallbackAccount: "default",
		Bin:             o.ClaudeBin,
	})
	return targets
}
