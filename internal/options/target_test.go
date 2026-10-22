package options

import (
	"strings"

	ginkgo "github.com/onsi/ginkgo/v2"
)

var _ = ginkgo.It("parses the four target flag forms as one shared single-use option", func() {
	t := testT{ginkgo.GinkgoT()}
	config := "aliases: {codex: [codex]}\n" + aliasAccounts
	for _, args := range [][]string{
		{"-t", "ChatGPT"},
		{"-t=ChatGPT"},
		{"--target", "ChatGPT"},
		{"--target=ChatGPT"},
		{"-t", "CODEX"},
		{"--target=codeX"},
	} {
		o := parseWithConfig(t, config, args...)
		targets := o.Targets()
		if o.TargetName == "" || len(targets) != 2 {
			t.Fatalf("args %v: options = %+v, targets = %+v", args, o, targets)
		}
		for i, want := range []string{"codex:0", "codex:1"} {
			if targets[i].ID != want || targets[i].Index != i || targets[i].Provider != "codex" {
				t.Fatalf("args %v: target %d = %+v, want %s at index %d", args, i, targets[i], want, i)
			}
		}
	}
})

var _ = ginkgo.It("rejects blank, missing, and repeated target values before configuration", func() {
	t := testT{ginkgo.GinkgoT()}
	// No configuration file exists: these errors must not depend on one.
	t.Setenv("HOME", t.TempDir())
	for _, args := range [][]string{
		{"-t", ""}, {"--target", ""}, {"-t="}, {"--target="},
		{"-t", "  "}, {"-t", "\u3000"}, {"--target= "},
		{"-t"}, {"--target"},
		{"-t", "a", "-t", "b"}, {"-t", "a", "--target", "b"},
		{"--target=a", "-t", "b"}, {"--target", "a", "--target=b"}, {"-t=a", "--target=b"},
	} {
		if _, err := Parse(args); !IsUsageError(err) {
			ginkgo.Fail("accepted invalid target args: " + strings.Join(args, " "))
		}
	}
	if _, err := parse([]string{"-t", ""}); !IsUsageError(err) ||
		!strings.Contains(err.Error(), "--target must not be blank") {
		t.Fatalf("blank target = %v, want blank rejection", err)
	}
	if _, err := parse([]string{"-t", "a", "--target", "b"}); !IsUsageError(err) ||
		!strings.Contains(err.Error(), "flag --target may be supplied only once") {
		t.Fatalf("repeated target = %v, want shared duplicate rejection", err)
	}
})

var _ = ginkgo.It("builds only selected provider targets with stable IDs and dense indices", func() {
	t := testT{ginkgo.GinkgoT()}
	config := "aliases: {codex: [codex], cursor: [grok]}\n" + aliasAccounts
	selectedIDs := func(args ...string) []string {
		targets := parseWithConfig(t, config, args...).Targets()
		ids := make([]string, len(targets))
		for i, tgt := range targets {
			if tgt.Index != i {
				t.Fatalf("args %v: target %d has index %d, want %d", args, i, tgt.Index, i)
			}
			ids[i] = tgt.ID
		}
		return ids
	}
	if got := selectedIDs("-t", "ChatGPT"); strings.Join(got, ",") != "codex:0,codex:1" {
		t.Fatalf("chatgpt selection = %v", got)
	}
	if got := selectedIDs("-t", "GROK"); strings.Join(got, ",") != "grok:0,cursor:0" {
		t.Fatalf("shared grok selection = %v", got)
	}
	if got := selectedIDs("-t", "claude"); strings.Join(got, ",") != "claude:0" {
		t.Fatalf("claude selection = %v", got)
	}

	// A provider listed before Codex keeps dense indices while Codex keeps
	// its own local account numbering.
	reordered := "cursor:\n  - name: main\naliases: {codex: [team], cursor: [team]}\ncodex:\n  - name: one\n    path: ~/.codex-one\n  - name: two\n    path: ~/.codex-two\n"
	targets := parseWithConfig(t, reordered, "-t", "TEAM").Targets()
	want := []struct {
		id    string
		index int
		label string
	}{{"cursor:0", 0, "main"}, {"codex:0", 1, "one"}, {"codex:1", 2, "two"}}
	if len(targets) != len(want) {
		t.Fatalf("team selection = %+v, want %d targets", targets, len(want))
	}
	for i, w := range want {
		if targets[i].ID != w.id || targets[i].Index != w.index || targets[i].FallbackAccount != w.label {
			t.Fatalf("team target %d = %+v, want %+v", i, targets[i], w)
		}
	}

	// agy selection retains both pool targets with the configured binary.
	agy := parseWithConfig(t, "codex:\n  - name: skip\n    path: ~/.codex-skip\nagy:\n  - name: default\n",
		"-t", "AGY", "--agy-bin", "/opt/bin/agy").Targets()
	if len(agy) != 2 || agy[0].ID != "agy:0" || agy[0].FallbackAccount != "gemini" ||
		agy[1].ID != "agy:1" || agy[1].FallbackAccount != "claude/gpt" || agy[0].Bin != "/opt/bin/agy" {
		t.Fatalf("agy selection = %+v, want both pools", agy)
	}

	// Raw nonblank spelling survives; only lowercasing participates.
	spaced := parseWithConfig(t, "aliases: {codex: [\" spaced \"]}\n"+aliasAccounts, "-t", " SPACED ").Targets()
	if len(spaced) != 2 || spaced[0].ID != "codex:0" {
		t.Fatalf("spaced selection = %+v, want codex accounts", spaced)
	}
})

var _ = ginkgo.It("validates the whole configuration before applying a target selection", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, "grok:\n  - name: main\n    path: ~/grok\nclaude:\n  - name: same\n    path: /one\n  - name: same\n    path: /two\n")
	if _, err := Parse([]string{"-t", "grok"}); !IsConfigError(err) ||
		!strings.Contains(err.Error(), "claude account names must be unique") {
		t.Fatalf("unselected invalid account data bypassed validation: %v", err)
	}
	writeConfig(t, home, "aliases: {codex: [1]}\ngrok:\n  - name: main\n    path: ~/grok\n")
	if _, err := Parse([]string{"-t", "grok"}); !IsConfigError(err) ||
		!strings.Contains(err.Error(), "entries must be strings") {
		t.Fatalf("invalid alias metadata bypassed validation: %v", err)
	}
})

var _ = ginkgo.It("returns a usage error when a target yields no actual targets", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, aliasAccounts)
	for _, args := range [][]string{
		{"-t", "codex"}, {"--target", "no-such-name"},
		{"-t", "chat"}, {"-t", "grok,cursor"},
		{"-t", "ChatGPT "}, {"-t", " chatgpt"},
	} {
		o, err := Parse(args)
		if !IsUsageError(err) || len(o.Targets()) != 0 ||
			!strings.Contains(err.Error(), args[len(args)-1]) {
			t.Fatalf("args %v: options = %+v, error = %v; want usage error naming the input", args, o, err)
		}
	}

	// Aliases never enable disabled providers.
	writeConfig(t, home, "aliases: {cursor: [team]}\ngrok:\n  - name: main\n    path: ~/grok\n")
	if o, err := Parse([]string{"-t", "team"}); !IsUsageError(err) || len(o.Targets()) != 0 {
		t.Fatalf("disabled provider matched through its alias: %+v / %v", o, err)
	}

	// A logically matched provider can still produce no targets when its raw
	// legacy provider key never enters the build order; the final slice, not
	// the matcher result, must decide. Without selection the legacy empty
	// snapshot behavior is unchanged.
	writeConfig(t, home, "? !!binary Y29kZXg=\n: [{name: work, path: ~/.codex-work}]\n")
	if o, err := Parse([]string{"-t", "chatgpt"}); !IsUsageError(err) || len(o.Targets()) != 0 {
		t.Fatalf("matched but targetless selection = %+v / %v, want usage error", o, err)
	}
	legacy, err := Parse(nil)
	if err != nil || len(legacy.Targets()) != 0 {
		t.Fatalf("legacy no-selection behavior changed: %+v / %v", legacy, err)
	}
})

var _ = ginkgo.It("keeps help independent of target validation", func() {
	t := testT{ginkgo.GinkgoT()}
	t.Setenv("HOME", t.TempDir())
	o, err := Parse([]string{"-t", "definitely-no-provider", "--help"})
	if err != nil || !o.Help || o.TargetName != "definitely-no-provider" {
		t.Fatalf("help with unknown target = %+v / %v, want config-free help", o, err)
	}
	if _, err := Parse([]string{"--help", "-t", ""}); !IsUsageError(err) {
		t.Fatalf("help did not survive a parser error: %v", err)
	}
})
