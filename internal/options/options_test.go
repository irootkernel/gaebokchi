package options

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"

	providerhome "gaebokchi/internal/provider/home"
)

func writeConfig(t testT, home, body string) string {
	t.Helper()
	path := filepath.Join(home, ".config", "gaebokchi", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func parseWithConfig(t testT, body string, args ...string) Options {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, body)
	o, err := Parse(args)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

var _ = ginkgo.It("loads only configured accounts in provider and file order", func() {
	t := testT{ginkgo.GinkgoT()}
	o := parseWithConfig(t, `claude:
  - name: "  Claude Work  "
    path: ~/.claude-work
codex:
  - name: One
    path: ~/.codex-one
  - name: Two
    path: /tmp/codex-two
grok:
  - name: Grok Work
    path: ~/.grok-work
  - name: Grok Home
    path: ~/.grok-home
`, "--timeout", "40s", "--json")
	targets := o.Targets()
	if len(targets) != 5 || o.Timeout != 40*time.Second || !o.JSON {
		t.Fatalf("options = %+v, targets = %+v", o, targets)
	}
	for i, want := range []struct{ id, name string }{
		{"codex:0", "One"}, {"codex:1", "Two"}, {"grok:0", "Grok Work"},
		{"grok:1", "Grok Home"}, {"claude:0", "Claude Work"},
	} {
		if targets[i].ID != want.id || targets[i].Index != i || targets[i].FallbackAccount != want.name {
			t.Fatalf("target %d = %+v", i, targets[i])
		}
	}
	if targets[0].Home != filepath.Join(os.Getenv("HOME"), ".codex-one") || targets[1].Home != "/tmp/codex-two" {
		t.Fatalf("codex paths = %q, %q", targets[0].Home, targets[1].Home)
	}
})

var _ = ginkgo.It("disabled providers have no targets", func() {
	t := testT{ginkgo.GinkgoT()}
	o := parseWithConfig(t, "grok: []\nclaude:\n  - name: only\n    path: ~/.claude\n")
	targets := o.Targets()
	if len(targets) != 1 || targets[0].ID != "claude:0" {
		t.Fatalf("targets = %+v", targets)
	}
})

var _ = ginkgo.It("rejects null provider lists instead of silently disabling them", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	for _, value := range []string{"", "null", "~"} {
		path := writeConfig(t, home, "codex:\n  - name: active\n    path: /active\ngrok: "+value+"\n")
		_, err := loadConfig(path, home)
		if !IsConfigError(err) || !strings.Contains(err.Error(), "grok must be a list") {
			t.Fatalf("grok: %q must be rejected: %v", value, err)
		}
	}
})

var _ = ginkgo.It("rejects YAML merge keys before loading account lists", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	for _, body := range []string{
		"<<: {grok: null}\ncodex:\n  - name: active\n    path: /active\n",
		"codex:\n  - &account\n    name: first\n    path: /first\n  - <<: *account\n    name: second\n",
	} {
		path := writeConfig(t, home, body)
		_, err := loadConfig(path, home)
		if !IsConfigError(err) || !strings.Contains(err.Error(), "YAML merge keys are not supported") {
			t.Fatalf("merge key was accepted: %v", err)
		}
	}
})

var _ = ginkgo.It("requires configuration for usage but not help", func() {
	t := testT{ginkgo.GinkgoT()}
	t.Setenv("HOME", t.TempDir())
	_, err := Parse(nil)
	if !IsConfigError(err) || !strings.Contains(err.Error(), "configuration file not found") {
		t.Fatalf("missing config = %v", err)
	}
	o, err := Parse([]string{"--help"})
	if err != nil || !o.Help {
		t.Fatalf("help = %+v / %v", o, err)
	}
})

var _ = ginkgo.It("rejects malformed, unknown and duplicate YAML fields", func() {
	t := testT{ginkgo.GinkgoT()}
	for _, body := range []string{
		"codex: [", "unknown: []\n", "codex: []\ncodex: []\n",
		"codex:\n  - name: a\n    path: /x\n    extra: z\n",
		"---\ncodex: []\n---\ngrok: []\n",
		"codex: []\n", "codex:\n  - name: x\n    path: /x\n  - name: x\n    path: /y\n",
		"claude:\n  - name: \"\\u3000\"\n    path: /x\n",
		"grok:\n  - name: x\n    path: relative\n",
		"codex:\n  - name: x\n    path: ~other/codex\n",
	} {
		home := t.TempDir()
		path := writeConfig(t, home, body)
		_, err := loadConfig(path, home)
		if !IsConfigError(err) || !strings.Contains(err.Error(), path) {
			t.Fatalf("invalid config %q must report its path %q: %v", body, path, err)
		}
	}
})

var _ = ginkgo.It("rejects normalized and symlink duplicate paths", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	real := filepath.Join(home, "real")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	for _, second := range []string{real + "/.", link} {
		body := "codex:\n  - name: a\n    path: " + real + "\n  - name: b\n    path: " + second + "\n"
		path := writeConfig(t, home, body)
		_, err := loadConfig(path, home)
		if !IsConfigError(err) || !strings.Contains(err.Error(), path) {
			t.Fatalf("duplicate %q must report config path %q: %v", second, path, err)
		}
	}
	missing := filepath.Join("missing", "nested")
	body := "codex:\n  - name: a\n    path: " + filepath.Join(real, missing) +
		"\n  - name: b\n    path: " + filepath.Join(link, missing) + "\n"
	path := writeConfig(t, home, body)
	_, err := loadConfig(path, home)
	if !IsConfigError(err) || !strings.Contains(err.Error(), "account paths must be unique") {
		t.Fatalf("duplicate missing path under a symlink was accepted: %v", err)
	}
	for _, alias := range []struct {
		name, target string
	}{
		{"dangling-absolute", filepath.Join(real, "missing")},
		{"dangling-relative", filepath.Join("real", "missing")},
	} {
		link := filepath.Join(home, alias.name)
		if err := os.Symlink(alias.target, link); err != nil {
			t.Fatal(err)
		}
		body := "codex:\n  - name: a\n    path: " + filepath.Join(link, "nested") +
			"\n  - name: b\n    path: " + filepath.Join(real, missing) + "\n"
		path := writeConfig(t, home, body)
		_, err := loadConfig(path, home)
		if !IsConfigError(err) || !strings.Contains(err.Error(), "account paths must be unique") {
			t.Fatalf("duplicate path under %s was accepted: %v", alias.name, err)
		}
	}
	outside := filepath.Join(home, "outside")
	if err := os.MkdirAll(filepath.Join(outside, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{filepath.Join(outside, "acct"), filepath.Join(home, "acct")} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("outside/sub", filepath.Join(home, "pivot")); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(home, "through-parent")
	if err := os.Symlink("pivot/../acct", alias); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		other     string
		duplicate bool
	}{
		{filepath.Join(outside, "acct"), true},
		{filepath.Join(home, "acct"), false},
	} {
		body := "codex:\n  - name: a\n    path: " + alias +
			"\n  - name: b\n    path: " + tc.other + "\n"
		path := writeConfig(t, home, body)
		_, err := loadConfig(path, home)
		if tc.duplicate && (!IsConfigError(err) || !strings.Contains(err.Error(), "account paths must be unique")) {
			t.Fatalf("actual symlink duplicate was accepted: %v", err)
		}
		if !tc.duplicate && err != nil {
			t.Fatalf("distinct symlink paths were rejected: %v", err)
		}
	}
	broken := filepath.Join(home, "missing-parent")
	if err := os.Symlink("missing/../acct", broken); err != nil {
		t.Fatal(err)
	}
	body = "codex:\n  - name: a\n    path: " + broken +
		"\n  - name: b\n    path: " + filepath.Join(home, "acct") + "\n"
	path = writeConfig(t, home, body)
	if _, err := loadConfig(path, home); err != nil {
		t.Fatalf("broken symlink was treated as an existing path: %v", err)
	}
})

var _ = ginkgo.It("rejects existing and missing case aliases on case-insensitive filesystems", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	first := filepath.Join(home, "Account")
	second := filepath.Join(home, "ACCOUNT")
	if err := os.Mkdir(first, 0o700); err != nil {
		t.Fatal(err)
	}
	firstInfo, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	secondInfo, err := os.Stat(second)
	if os.IsNotExist(err) {
		ginkgo.Skip("filesystem is case-sensitive")
	}
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(firstInfo, secondInfo) {
		ginkgo.Skip("case variants are distinct directories")
	}
	path := writeConfig(t, home, "codex:\n  - name: first\n    path: "+first+
		"\n  - name: second\n    path: "+second+"\n")
	if _, err := loadConfig(path, home); !IsConfigError(err) || !strings.Contains(err.Error(), "account paths must be unique") {
		t.Fatalf("same directory was accepted twice: %v", err)
	}
	path = writeConfig(t, home, "codex:\n  - name: first\n    path: "+first+"/missing/child"+
		"\n  - name: second\n    path: "+second+"/missing/child\n")
	if _, err := loadConfig(path, home); !IsConfigError(err) || !strings.Contains(err.Error(), "account paths must be unique") {
		t.Fatalf("same missing path under case aliases was accepted: %v", err)
	}
	path = writeConfig(t, home, "codex:\n  - name: first\n    path: "+first+"/missing/child"+
		"\n  - name: second\n    path: "+first+"/MISSING/CHILD\n")
	if _, err := loadConfig(path, home); !IsConfigError(err) || !strings.Contains(err.Error(), "account paths must be unique") {
		t.Fatalf("case aliases of the missing suffix were accepted: %v", err)
	}
})

var _ = ginkgo.It("reserves caseless missing names but accepts distinct existing directories", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	first := filepath.Join(home, "Account")
	second := filepath.Join(home, "ACCOUNT")
	if err := os.Mkdir(first, 0o700); err != nil {
		t.Fatal(err)
	}
	path := writeConfig(t, home, "codex:\n  - name: first\n    path: "+first+"/missing"+
		"\n  - name: second\n    path: "+first+"/MISSING\n")
	if _, err := loadConfig(path, home); !IsConfigError(err) || !strings.Contains(err.Error(), "account paths must be unique") {
		t.Fatalf("caseless missing paths were accepted: %v", err)
	}
	if _, err := os.Stat(second); !os.IsNotExist(err) {
		ginkgo.Skip("filesystem does not distinguish case variants")
	}
	if err := os.Mkdir(second, 0o700); err != nil {
		t.Fatal(err)
	}
	path = writeConfig(t, home, "codex:\n  - name: first\n    path: "+first+
		"\n  - name: second\n    path: "+second+"\n")
	if _, err := loadConfig(path, home); err != nil {
		t.Fatalf("distinct existing directories were rejected: %v", err)
	}
})

var _ = ginkgo.It("rejects canonically equivalent missing Unicode paths", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	first := filepath.Join(home, "caf\u00e9")
	second := filepath.Join(home, "cafe\u0301")
	path := writeConfig(t, home, "codex:\n  - name: first\n    path: "+first+
		"\n  - name: second\n    path: "+second+"\n")
	if _, err := loadConfig(path, home); !IsConfigError(err) || !strings.Contains(err.Error(), "account paths must be unique") {
		t.Fatalf("canonical Unicode aliases of a missing path were accepted: %v", err)
	}
})

var _ = ginkgo.It("detects aliases of a searchable home without directory read permission", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	account := filepath.Join(home, "account")
	if err := os.Mkdir(account, 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(home, "alias")
	if err := os.Symlink(account, alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(account, 0o300); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(account, 0o700) })
	if dir, err := os.Open(account); err == nil {
		_ = dir.Close()
		ginkgo.Skip("directory read permission checks are bypassed by this user")
	}
	path := writeConfig(t, home, "codex:\n  - name: first\n    path: "+account+
		"\n  - name: second\n    path: "+alias+"\n")
	if _, err := loadConfig(path, home); !IsConfigError(err) || !strings.Contains(err.Error(), "account paths must be unique") {
		t.Fatalf("searchable home alias was accepted twice: %v", err)
	}
})

var _ = ginkgo.It("keeps hardlinked files as separate invalid home rows", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	valid := filepath.Join(home, "valid")
	if err := os.Mkdir(valid, 0o700); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(home, "first-file")
	second := filepath.Join(home, "second-file")
	if err := os.WriteFile(first, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(first, second); err != nil {
		t.Fatal(err)
	}
	path := writeConfig(t, home, "codex:\n  - name: valid\n    path: "+valid+
		"\n  - name: first\n    path: "+first+
		"\n  - name: second\n    path: "+second+"\n")
	cfg, err := loadConfig(path, home)
	if err != nil || len(cfg.Codex) != 3 {
		t.Fatalf("non-directory homes blocked the valid account: %+v / %v", cfg.Codex, err)
	}
	for _, account := range cfg.Codex[1:] {
		if status := providerhome.Inspect(account.Path); status != providerhome.Invalid {
			t.Fatalf("hardlinked file %q status = %v, want Invalid", account.Name, status)
		}
	}
})

var _ = ginkgo.It("keeps aliases of an inaccessible directory as error rows", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	valid := filepath.Join(home, "valid")
	locked := filepath.Join(home, "locked")
	for _, dir := range []string{valid, locked} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	alias := filepath.Join(home, "alias")
	if err := os.Symlink(locked, alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	if providerhome.Inspect(locked) != providerhome.Invalid {
		ginkgo.Skip("current user can still access chmod 000 directories")
	}
	path := writeConfig(t, home, "codex:\n  - name: valid\n    path: "+valid+
		"\n  - name: locked\n    path: "+locked+
		"\n  - name: alias\n    path: "+alias+"\n")
	cfg, err := loadConfig(path, home)
	if err != nil || len(cfg.Codex) != 3 {
		t.Fatalf("inaccessible directory blocked the valid account: %+v / %v", cfg.Codex, err)
	}
})

var _ = ginkgo.It("accepts missing target directories for row-level errors", func() {
	t := testT{ginkgo.GinkgoT()}
	o := parseWithConfig(t, "grok:\n  - name: absent\n    path: /definitely/not/here\n")
	if got := o.Targets()[0].Home; got != "/definitely/not/here" {
		t.Fatalf("path = %q", got)
	}
})

var _ = ginkgo.It("rejects duplicate paths with dot components below a missing directory", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	real := filepath.Join(home, "real")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(home, "alias")
	if err := os.Symlink(real, alias); err != nil {
		t.Fatal(err)
	}
	first := real + "/missing/child"
	for _, second := range []string{real + "/missing/./child", alias + "/missing/./child", alias + "/missing//child"} {
		path := writeConfig(t, home, "codex:\n  - name: first\n    path: "+first+
			"\n  - name: second\n    path: "+second+"\n")
		if _, err := loadConfig(path, home); !IsConfigError(err) || !strings.Contains(err.Error(), "account paths must be unique") {
			t.Fatalf("duplicate missing path %q was accepted: %v", second, err)
		}
	}
})

var _ = ginkgo.It("preserves path traversal through symlinks and missing components", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	outer := filepath.Join(home, "outer")
	if err := os.MkdirAll(filepath.Join(outer, "child"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{filepath.Join(outer, "account"), filepath.Join(home, "account")} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	alias := filepath.Join(home, "alias")
	if err := os.Symlink(filepath.Join(outer, "child"), alias); err != nil {
		t.Fatal(err)
	}
	viaAlias := alias + "/../account"
	missingParent := filepath.Join(home, "missing") + "/../account"
	path := writeConfig(t, home, "codex:\n  - name: alias\n    path: "+viaAlias+
		"\n  - name: direct\n    path: "+filepath.Join(home, "account")+
		"\n  - name: missing\n    path: "+missingParent+"\n")
	cfg, err := loadConfig(path, home)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Codex[0].Path != viaAlias || cfg.Codex[2].Path != missingParent {
		t.Fatalf("configured paths were changed: %+v", cfg.Codex)
	}
	got, err := os.Stat(cfg.Codex[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.Stat(filepath.Join(outer, "account"))
	if err != nil || !os.SameFile(got, want) {
		t.Fatalf("symlink parent selected a different home: %v", err)
	}
	if status := providerhome.Inspect(cfg.Codex[2].Path); status != providerhome.Missing {
		t.Fatalf("missing parent status = %v, want Missing", status)
	}
	path = writeConfig(t, home, "codex:\n  - name: alias\n    path: ~/alias/../account\n"+
		"  - name: direct\n    path: "+filepath.Join(outer, "account")+"\n")
	if _, err := loadConfig(path, home); !IsConfigError(err) || !strings.Contains(err.Error(), "account paths must be unique") {
		t.Fatalf("actual duplicate through expanded ~/ path was accepted: %v", err)
	}
})

var _ = ginkgo.It("keeps a symlink through a regular file as an invalid home", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	account := filepath.Join(home, "account")
	if err := os.Mkdir(account, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "plain-file"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(home, "bad")
	if err := os.Symlink("plain-file/../account", bad); err != nil {
		t.Fatal(err)
	}
	path := writeConfig(t, home, "codex:\n  - name: invalid\n    path: "+bad+
		"\n  - name: valid\n    path: "+account+"\n")
	cfg, err := loadConfig(path, home)
	if err != nil {
		t.Fatalf("invalid home was treated as a duplicate: %v", err)
	}
	if len(cfg.Codex) != 2 || providerhome.Inspect(cfg.Codex[0].Path) != providerhome.Invalid {
		t.Fatalf("invalid home was not retained for its error row: %+v", cfg.Codex)
	}
})

var _ = ginkgo.It("rejects removed account flags and invalid scalar flags", func() {
	for _, args := range [][]string{
		{"--codex-home", "/x"}, {"--grok-account", "x"}, {"--version"}, {"extra"},
		{"--timeout", "999ms"}, {"--timeout", "121s"}, {"--timeout", "bad"},
		{"--timeout"}, {"--json=true"}, {"--help", "--help"},
		{"--codex-bin", ""}, {"--claude-bin", "a", "--claude-bin", "b"},
		{"--agy-bin", ""}, {"--agy-bin", "a", "--agy-bin", "b"},
		{"--cursor-bin", ""}, {"--cursor-bin", "a", "--cursor-bin", "b"},
	} {
		_, err := parse(args)
		if !IsUsageError(err) {
			ginkgo.Fail("accepted invalid args: " + strings.Join(args, " "))
		}
	}
	o, err := parse([]string{"--codex-bin=/tmp/codex", "--agy-bin=/tmp/agy", "--cursor-bin=/tmp/cursor-agent", "-h"})
	if err != nil || o.CodexBin != "/tmp/codex" || o.AgyBin != "/tmp/agy" || o.CursorBin != "/tmp/cursor-agent" || !o.Help {
		ginkgo.Fail("valid flags rejected")
	}
})

var _ = ginkgo.It("emits two weekly pool targets for a single configured agy account and rejects more than one", func() {
	t := testT{ginkgo.GinkgoT()}
	o := parseWithConfig(t, "agy:\n  - name: default\n", "--agy-bin", "/opt/bin/agy")
	targets := o.Targets()
	if len(targets) != 2 {
		t.Fatalf("targets = %+v, want 2 agy pool targets", targets)
	}
	if targets[0].ID != "agy:0" || targets[0].FallbackAccount != "gemini" || targets[0].Bin != "/opt/bin/agy" ||
		targets[1].ID != "agy:1" || targets[1].FallbackAccount != "claude/gpt" || targets[1].Bin != "/opt/bin/agy" {
		t.Fatalf("unexpected agy targets: %+v", targets)
	}

	home := t.TempDir()
	path := writeConfig(t, home, "agy:\n  - name: first\n  - name: second\n")
	if _, err := loadConfig(path, home); !IsConfigError(err) || !strings.Contains(err.Error(), "agy supports at most one account") {
		t.Fatalf("multiple agy accounts were accepted: %v", err)
	}
})

var _ = ginkgo.It("emits one cursor target after agy and rejects invalid cursor configurations", func() {
	t := testT{ginkgo.GinkgoT()}
	o := parseWithConfig(t, "agy:\n  - name: default\ncursor:\n  - name: \"  work  \"\n", "--cursor-bin", "/opt/bin/cursor-agent")
	targets := o.Targets()
	if len(targets) != 3 {
		t.Fatalf("targets = %+v, want 3 targets (2 agy + 1 cursor)", targets)
	}
	if targets[2].ID != "cursor:0" || targets[2].Index != 2 || targets[2].FallbackAccount != "work" || targets[2].Bin != "/opt/bin/cursor-agent" || targets[2].Home != "" {
		t.Fatalf("unexpected cursor target: %+v", targets[2])
	}

	oDisabled := parseWithConfig(t, "agy:\n  - name: default\ncursor: []\n")
	if len(oDisabled.Targets()) != 2 {
		t.Fatalf("disabled cursor targets = %+v, want 2 agy targets", oDisabled.Targets())
	}

	home := t.TempDir()
	path := writeConfig(t, home, "cursor:\n  - name: first\n  - name: second\n")
	if _, err := loadConfig(path, home); !IsConfigError(err) || !strings.Contains(err.Error(), "cursor supports at most one account") {
		t.Fatalf("multiple cursor accounts were accepted: %v", err)
	}

	path = writeConfig(t, home, "cursor:\n  - name: default\n    path: ~/.cursor\n")
	if _, err := loadConfig(path, home); !IsConfigError(err) || !strings.Contains(err.Error(), "invalid configuration file") {
		t.Fatalf("cursor path field was accepted: %v", err)
	}

	path = writeConfig(t, home, "cursor:\n  - name: \"   \"\n")
	if _, err := loadConfig(path, home); !IsConfigError(err) || !strings.Contains(err.Error(), "cursor account 1 needs a visible name") {
		t.Fatalf("blank cursor name was accepted: %v", err)
	}

	path = writeConfig(t, home, "codex:\n  - name: active\n    path: /active\ncursor: null\n")
	if _, err := loadConfig(path, home); !IsConfigError(err) || !strings.Contains(err.Error(), "cursor must be a list") {
		t.Fatalf("null cursor list was accepted: %v", err)
	}
})
