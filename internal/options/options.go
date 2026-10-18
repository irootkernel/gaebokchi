// Package options parses CLI flags and the required account configuration.
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

const (
	DefaultCodexBin  = "codex"
	DefaultGrokBin   = "grok"
	DefaultClaudeBin = "claude"
	DefaultAgyBin    = "agy"
	DefaultCursorBin = "cursor-agent"
	DefaultZCodeBin  = "zcode"
	DefaultTimeout   = 20 * time.Second
	minTimeout       = 1 * time.Second
	maxTimeout       = 120 * time.Second
	ConfigGuideURL   = "https://github.com/irootkernel/gaebokchi/blob/main/configuration/README.md"
)

type Options struct {
	CodexBin  string
	GrokBin   string
	ClaudeBin string
	AgyBin    string
	CursorBin string
	ZCodeBin  string
	Timeout   time.Duration
	Help      bool
	JSON      bool
	targets   []domain.Target
}

type UsageError struct{ msg string }

func (e *UsageError) Error() string           { return e.msg }
func usagef(format string, args ...any) error { return &UsageError{fmt.Sprintf(format, args...)} }
func IsUsageError(err error) bool {
	var e *UsageError
	return errors.As(err, &e)
}

type ConfigError struct{ msg string }

func (e *ConfigError) Error() string { return e.msg }
func IsConfigError(err error) bool {
	var e *ConfigError
	return errors.As(err, &e)
}

type flagKind int

const (
	kindString flagKind = iota
	kindDuration
	kindBool
)

type flagSpec struct {
	name string
	kind flagKind
	set  func(*Options, string) error
}

func setBin(field func(*Options) *string, name string) func(*Options, string) error {
	return func(o *Options, v string) error {
		if v == "" {
			return usagef("--%s must not be empty", name)
		}
		*field(o) = v
		return nil
	}
}

func specs() map[string]*flagSpec {
	return map[string]*flagSpec{
		"codex-bin":  {name: "codex-bin", kind: kindString, set: setBin(func(o *Options) *string { return &o.CodexBin }, "codex-bin")},
		"grok-bin":   {name: "grok-bin", kind: kindString, set: setBin(func(o *Options) *string { return &o.GrokBin }, "grok-bin")},
		"claude-bin": {name: "claude-bin", kind: kindString, set: setBin(func(o *Options) *string { return &o.ClaudeBin }, "claude-bin")},
		"agy-bin":    {name: "agy-bin", kind: kindString, set: setBin(func(o *Options) *string { return &o.AgyBin }, "agy-bin")},
		"cursor-bin": {name: "cursor-bin", kind: kindString, set: setBin(func(o *Options) *string { return &o.CursorBin }, "cursor-bin")},
		"zcode-bin":  {name: "zcode-bin", kind: kindString, set: setBin(func(o *Options) *string { return &o.ZCodeBin }, "zcode-bin")},
		"timeout": {name: "timeout", kind: kindDuration, set: func(o *Options, v string) error {
			d, err := time.ParseDuration(v)
			if err != nil || d < minTimeout || d > maxTimeout {
				return usagef("--timeout must be a duration between 1s and 120s")
			}
			o.Timeout = d
			return nil
		}},
		"help": {name: "help", kind: kindBool, set: func(o *Options, _ string) error { o.Help = true; return nil }},
		"json": {name: "json", kind: kindBool, set: func(o *Options, _ string) error { o.JSON = true; return nil }},
	}
}

func Parse(args []string) (Options, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Options{}, usagef("cannot resolve user home directory")
	}
	o, err := parse(args)
	if err != nil || o.Help {
		return o, err
	}
	accounts, err := loadConfig(filepath.Join(home, ".config", "gaebokchi", "config.yaml"), home)
	if err != nil {
		return Options{}, err
	}
	o.targets = buildTargets(accounts, o)
	return o, nil
}

func parse(args []string) (Options, error) {
	table := specs()
	seen := map[string]bool{}
	o := Options{
		CodexBin: DefaultCodexBin, GrokBin: DefaultGrokBin,
		ClaudeBin: DefaultClaudeBin, AgyBin: DefaultAgyBin,
		CursorBin: DefaultCursorBin, ZCodeBin: DefaultZCodeBin, Timeout: DefaultTimeout,
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			return Options{}, usagef("unexpected positional argument %q", arg)
		}
		name := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
		value, hasValue := "", false
		if left, right, ok := strings.Cut(name, "="); ok {
			name, value, hasValue = left, right, true
		}
		if name == "h" {
			name = "help"
		}
		spec := table[name]
		if spec == nil {
			return Options{}, usagef("unknown flag %q", arg)
		}
		if seen[spec.name] {
			return Options{}, usagef("flag --%s may be supplied only once", name)
		}
		if spec.kind == kindBool {
			if hasValue {
				return Options{}, usagef("flag --%s does not take a value", name)
			}
		} else if !hasValue {
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
	return o, nil
}

func (o Options) Targets() []domain.Target { return append([]domain.Target(nil), o.targets...) }
