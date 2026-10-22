package options

import (
	"reflect"
	"strings"

	ginkgo "github.com/onsi/ginkgo/v2"

	"gaebokchi/internal/domain"
)

const aliasAccounts = `codex:
  - name: work
    path: ~/.codex-work
  - name: personal
    path: ~/.codex-personal
grok:
  - name: personal
    path: ~/.grok-personal
claude:
  - name: work
    path: ~/.claude-work
agy:
  - name: default
cursor:
  - name: default
zcode:
  - name: default
`

func loadAliasConfig(t testT, body string) accountConfig {
	t.Helper()
	home := t.TempDir()
	cfg, err := loadConfig(writeConfig(t, home, body), home)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func assertProviderMatches(t testT, cfg accountConfig, name string, want ...domain.ProviderKind) {
	t.Helper()
	expected := map[domain.ProviderKind]bool{}
	for _, kind := range want {
		expected[kind] = true
	}
	if got := cfg.matchProviders(name); !reflect.DeepEqual(got, expected) {
		t.Fatalf("matchProviders(%q) = %v, want %v", name, got, expected)
	}
}

var _ = ginkgo.It("matches all six display names without implicit config-key aliases", func() {
	t := testT{ginkgo.GinkgoT()}
	for _, metadata := range []string{"", "aliases: {}\n", "aliases: {codex: []}\n"} {
		cfg := loadAliasConfig(t, metadata+aliasAccounts)
		for _, tc := range []struct {
			name string
			kind domain.ProviderKind
		}{
			{"ChatGPT", domain.ProviderCodex}, {"GROK", domain.ProviderGrok},
			{"cLaUdE", domain.ProviderClaude}, {"AGY", domain.ProviderAgy},
			{"Cursor", domain.ProviderCursor}, {"ZCODE", domain.ProviderZCode},
		} {
			assertProviderMatches(t, cfg, tc.name, tc.kind)
		}
		for _, name := range []string{"codex", "chat", "ChatGPT ", " chatgpt", "grok,cursor", "", "  "} {
			assertProviderMatches(t, cfg, name)
		}
	}
})

var _ = ginkgo.It("unions shared aliases and default-name collisions once per enabled provider", func() {
	t := testT{ginkgo.GinkgoT()}
	cfg := loadAliasConfig(t, `aliases:
  codex: [codex, chatgpt, ChatGPT, codex, team]
  grok: [chatgpt, team]
  cursor: [grok, team]
`+aliasAccounts)
	assertProviderMatches(t, cfg, "CODEX", domain.ProviderCodex)
	assertProviderMatches(t, cfg, "ChatGPT", domain.ProviderCodex, domain.ProviderGrok)
	assertProviderMatches(t, cfg, "GROK", domain.ProviderGrok, domain.ProviderCursor)
	assertProviderMatches(t, cfg, "TEAM", domain.ProviderCodex, domain.ProviderGrok, domain.ProviderCursor)
	assertProviderMatches(t, cfg, "cursor", domain.ProviderCursor)
	sharedList := loadAliasConfig(t, "aliases: {codex: &names [team], grok: *names}\n"+aliasAccounts)
	assertProviderMatches(t, sharedList, "team", domain.ProviderCodex, domain.ProviderGrok)

	withoutAliases := loadAliasConfig(t, aliasAccounts)
	assertProviderMatches(t, withoutAliases, "codex")
	assertProviderMatches(t, withoutAliases, "team")
})

var _ = ginkgo.It("keeps nonblank alias spelling intact and accepts string YAML aliases", func() {
	t := testT{ginkgo.GinkgoT()}
	cfg := loadAliasConfig(t, `codex:
  - name: &selection_key aliases
    path: ~/.codex-work
? *selection_key
: {codex: [&numeric "1", *numeric, " spaced ", "grok,cursor"]}
`)
	assertProviderMatches(t, cfg, "1", domain.ProviderCodex)
	assertProviderMatches(t, cfg, " SPACED ", domain.ProviderCodex)
	assertProviderMatches(t, cfg, "spaced")
	assertProviderMatches(t, cfg, "grok,cursor", domain.ProviderCodex)
	if !reflect.DeepEqual(cfg.order, []domain.ProviderKind{domain.ProviderCodex}) {
		t.Fatalf("anchored aliases key changed provider order: %v", cfg.order)
	}
})

var _ = ginkgo.It("does not enable missing or empty providers through their aliases", func() {
	t := testT{ginkgo.GinkgoT()}
	for _, disabled := range []string{"", "cursor: []\n"} {
		cfg := loadAliasConfig(t, `aliases: {cursor: [grok], claude: [grok]}
grok:
  - name: personal
    path: ~/.grok-personal
`+disabled)
		assertProviderMatches(t, cfg, "grok", domain.ProviderGrok)
		assertProviderMatches(t, cfg, "cursor")
		assertProviderMatches(t, cfg, "claude")
	}
	home := t.TempDir()
	_, err := loadConfig(writeConfig(t, home, "aliases: {codex: [codex]}\n"), home)
	if !IsConfigError(err) || !strings.Contains(err.Error(), "needs at least one account") {
		t.Fatalf("alias-only configuration = %v, want account requirement error", err)
	}
})

var _ = ginkgo.It("excludes alias metadata from account targets and provider order at every position", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, aliasAccounts)
	baseline, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	metadata := "aliases: {codex: [codex], cursor: [grok], zcode: [team]}\n"
	for _, body := range []string{
		metadata + aliasAccounts,
		strings.Replace(aliasAccounts, "grok:\n", metadata+"grok:\n", 1),
		aliasAccounts + metadata,
	} {
		writeConfig(t, home, body)
		got, err := Parse(nil)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.Targets(), baseline.Targets()) {
			t.Fatalf("alias metadata changed default targets: got %+v, want %+v", got.Targets(), baseline.Targets())
		}
		cfg, err := loadConfig(writeConfig(t, home, body), home)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(cfg.order, defaultProviderOrder) {
			t.Fatalf("provider order = %v, want %v", cfg.order, defaultProviderOrder)
		}
	}
})

var _ = ginkgo.It("rejects malformed provider aliases before constructing any targets", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, aliases := range []string{
		"null", "~", "", "[]", "scalar", "{codex: scalar}", "{codex: {a: b}}",
		"{gpt: [x]}", "{CODEX: [x]}", "{Codex: [x]}", "{1: [x]}",
		"{codex: null}", "{codex: ~}", "{codex:}",
		"{codex: [1]}", "{codex: [true]}", "{codex: [1.5]}", "{codex: [2026-10-22]}",
		"{codex: [null]}", "{codex: [~]}", "{codex: [[]]}", "{codex: [{}]}",
		`{codex: [""]}`, `{codex: ["  "]}`, `{codex: ["\u3000"]}`,
		"{codex: [x], codex: [y]}", "{&key codex: [x], *key: [y]}",
	} {
		path := writeConfig(t, home, "aliases: "+aliases+"\n"+aliasAccounts)
		got, err := Parse(nil)
		if !IsConfigError(err) || !strings.Contains(err.Error(), path) || len(got.Targets()) != 0 {
			t.Fatalf("aliases %q: options=%+v, error=%v; want path-prefixed ConfigError and no targets", aliases, got, err)
		}
	}
	for _, body := range []string{
		"aliases: {}\naliases: {}\n" + aliasAccounts,
		"aliases: {}\n" + aliasAccounts + "---\n{}\n",
		"aliases: {codex: &names [x], <<: {grok: *names}}\n" + aliasAccounts,
		"aliases: {}\ncodex: []\ncodex: []\n",
		"aliases: {codex: [x]}\ncodex: null\n",
	} {
		writeConfig(t, home, body)
		if _, err := Parse(nil); !IsConfigError(err) {
			t.Fatalf("configuration %q: error=%v, want ConfigError", body, err)
		}
	}
})

var _ = ginkgo.It("resolves tagged root keys through the typed decoder's logical field name", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, aliasAccounts)
	baseline, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}

	// The typed decoder base64-decodes a !!binary key to "aliases" while its
	// raw node value keeps the encoded spelling, so validation must apply.
	path := writeConfig(t, home, "? !!binary YWxpYXNlcw==\n: {codex: [123]}\n"+aliasAccounts)
	rejected, err := Parse(nil)
	if !IsConfigError(err) || !strings.Contains(err.Error(), path) ||
		!strings.Contains(err.Error(), "entries must be strings") || len(rejected.Targets()) != 0 {
		t.Fatalf("tagged aliases = %+v, error = %v; want rejected numeric alias without targets", rejected, err)
	}

	for _, metadata := range []string{
		"!!str aliases: {codex: [team]}\n",
		"? !!binary YWxpYXNlcw==\n: {codex: [team]}\n",
	} {
		writeConfig(t, home, metadata+aliasAccounts)
		tagged, err := Parse(nil)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(tagged.Targets(), baseline.Targets()) {
			t.Fatalf("metadata %q changed default targets: got %+v, want %+v", metadata, tagged.Targets(), baseline.Targets())
		}
		cfg, err := loadConfig(writeConfig(t, home, metadata+aliasAccounts), home)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(cfg.order, defaultProviderOrder) {
			t.Fatalf("metadata %q changed provider order: %v", metadata, cfg.order)
		}
		assertProviderMatches(t, cfg, "team", domain.ProviderCodex)
	}

	anchored := "grok:\n  - name: &meta aliases\n    path: ~/.grok-work\n? *meta\n: {grok: [team]}\n"
	writeConfig(t, home, anchored)
	anchoredOptions, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	writeConfig(t, home, "grok:\n  - name: aliases\n    path: ~/.grok-work\n")
	plain, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(anchoredOptions.Targets(), plain.Targets()) {
		t.Fatalf("anchored aliases key targets = %+v, want %+v", anchoredOptions.Targets(), plain.Targets())
	}
	cfg, err := loadConfig(writeConfig(t, home, anchored), home)
	if err != nil {
		t.Fatal(err)
	}
	assertProviderMatches(t, cfg, "team", domain.ProviderGrok)
})
