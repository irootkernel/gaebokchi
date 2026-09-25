package options

import (
	"fmt"
	ginkgo "github.com/onsi/ginkgo/v2"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	testHome = "/Users/tester"
	testWD   = "/Users/tester/work/project"
)

func mustParse(t testT, args ...string) Options {
	t.Helper()
	o, err := parse(args, testHome, testWD)
	if err != nil {
		t.Fatalf("parse(%q) failed: %v", args, err)
	}
	return o
}

func mustFail(t testT, args ...string) {
	t.Helper()
	o, err := parse(args, testHome, testWD)
	if err == nil {
		t.Fatalf("parse(%q) unexpectedly succeeded: %+v", args, o)
	}
	if !IsUsageError(err) {
		t.Fatalf("parse(%q) error is not a usage error: %v", args, err)
	}
}

var _ = ginkgo.It("TestDefaultTargets", func() {
	t := testT{ginkgo.GinkgoT()}
	var failures []string
	fail := func(format string, args ...any) { failures = append(failures, fmt.Sprintf(format, args...)) }
	o := mustParse(t)
	if len(o.CodexHomes) != 2 {
		t.Fatalf("default homes = %v, want 2", o.CodexHomes)
	}
	if o.CodexHomes[0] != testHome+"/.codex" || o.CodexHomes[1] != testHome+"/.codex-hsy" {
		t.Fatalf("default homes = %v", o.CodexHomes)
	}
	targets := o.Targets()
	if len(targets) != 4 {
		t.Fatalf("default target count = %d, want 4", len(targets))
	}
	wantIDs := []string{"codex:0", "codex:1", "grok:0", "claude:0"}
	wantProviders := []string{"codex", "codex", "grok", "claude"}
	wantAccounts := []string{".codex", ".codex-hsy", "default", "default"}
	for i, tt := range targets {
		if tt.ID != wantIDs[i] || string(tt.Provider) != wantProviders[i] {
			fail("target %d = %s/%s, want %s/%s", i, tt.ID, tt.Provider, wantIDs[i], wantProviders[i])
		}
		if tt.FallbackAccount != wantAccounts[i] {
			fail("target %d fallback = %q, want %q", i, tt.FallbackAccount, wantAccounts[i])
		}
		if tt.Index != i {
			fail("target %d index = %d", i, tt.Index)
		}
	}
	if targets[0].Home == targets[1].Home || targets[2].Home != "" || targets[3].Home != "" {
		fail("%s", "codex homes must differ and other homes must be empty")
	}
	if o.Timeout != 20*time.Second || o.CodexBin != "codex" || o.GrokBin != "grok" || o.ClaudeBin != "claude" || o.GrokAccount != "default" {
		fail("defaults not applied: %+v", o)
	}
	if len(failures) > 0 {
		t.Fatalf("%s", strings.Join(failures, "\n"))
	}
})

// The options package must not read the environment at all, so an ambient
// CODEX_HOME cannot change the default account list.
var _ = ginkgo.It("TestAmbientCodexHomeIgnored", func() {
	t := testT{ginkgo.GinkgoT()}
	t.Setenv("CODEX_HOME", "/somewhere/else")
	o := mustParse(t)
	if o.CodexHomes[0] != testHome+"/.codex" {
		t.Fatalf("ambient CODEX_HOME leaked into defaults: %v", o.CodexHomes)
	}
})

var _ = ginkgo.It("TestCodexHomeReplacementOrder", func() {
	t := testT{ginkgo.GinkgoT()}
	o := mustParse(t, "--codex-home", "/data/second", "--codex-home", "/data/first")
	if len(o.CodexHomes) != 2 || o.CodexHomes[0] != "/data/second" || o.CodexHomes[1] != "/data/first" {
		t.Fatalf("replacement order not preserved: %v", o.CodexHomes)
	}
	targets := o.Targets()
	if len(targets) != 4 || targets[2].Provider != "grok" || targets[3].Provider != "claude" {
		t.Fatalf("grok and claude must follow codex, got %+v", targets)
	}
})

var _ = ginkgo.It("TestTildeExpansion", func() {
	t := testT{ginkgo.GinkgoT()}
	var failures []string
	fail := func(format string, args ...any) { failures = append(failures, fmt.Sprintf(format, args...)) }
	o := mustParse(t, "--codex-home", "~", "--codex-home", "~/x/y/../z")
	if o.CodexHomes[0] != testHome {
		fail(`~ expanded to %q`, o.CodexHomes[0])
	}
	if o.CodexHomes[1] != testHome+"/x/z" {
		fail(`~/x/y/../z expanded to %q`, o.CodexHomes[1])
	}
	if len(failures) > 0 {
		t.Fatalf("%s", strings.Join(failures, "\n"))
	}
})

var _ = ginkgo.It("TestTildeOtherUserRejected", func() {
	t := testT{ginkgo.GinkgoT()}
	mustFail(t, "--codex-home", "~other/cache")
})

var _ = ginkgo.It("TestRelativeHomeResolvedAgainstWD", func() {
	t := testT{ginkgo.GinkgoT()}
	o := mustParse(t, "--codex-home", "state/a")
	if o.CodexHomes[0] != testWD+"/state/a" {
		t.Fatalf("relative home = %q", o.CodexHomes[0])
	}
})

var _ = ginkgo.It("TestEmptyValuesRejected", func() {
	t := testT{ginkgo.GinkgoT()}
	mustFail(t, "--codex-home", "")
	mustFail(t, "--codex-home=")
	mustFail(t, "--grok-account", "")
	mustFail(t, "--codex-bin", "")
	mustFail(t, "--grok-bin", "")
	mustFail(t, "--claude-bin", "")
})

var _ = ginkgo.It("TestDuplicateHomesRejected", func() {
	t := testT{ginkgo.GinkgoT()}
	mustFail(t, "--codex-home", "/data/a", "--codex-home", "/data/a")
	mustFail(t, "--codex-home", "/data/a", "--codex-home", "/data/./a")
	mustFail(t, "--codex-home", "/data/a/", "--codex-home", "/data/a")
})

var _ = ginkgo.It("TestDuplicateHomesThroughSymlink", func() {
	t := testT{ginkgo.GinkgoT()}
	dir := t.TempDir()
	real := filepath.Join(dir, "real-home")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "alias-home")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	_, err := parse([]string{"--codex-home", real, "--codex-home", link}, testHome, testWD)
	if err == nil {
		t.Fatal("symlink alias duplicate not rejected")
	}
})

var _ = ginkgo.It("TestMissingHomeIsValid", func() {
	t := testT{ginkgo.GinkgoT()}
	o := mustParse(t, "--codex-home", "/definitely/not/here")
	if o.CodexHomes[0] != "/definitely/not/here" {
		t.Fatalf("missing home altered: %v", o.CodexHomes)
	}
})

var _ = ginkgo.It("TestTimeoutValidation", func() {
	t := testT{ginkgo.GinkgoT()}
	var failures []string
	fail := func(format string, args ...any) { failures = append(failures, fmt.Sprintf(format, args...)) }
	if o := mustParse(t, "--timeout", "40s"); o.Timeout != 40*time.Second {
		fail("40s parsed as %v", o.Timeout)
	}
	if o := mustParse(t, "--timeout", "120s"); o.Timeout != 120*time.Second {
		fail("120s parsed as %v", o.Timeout)
	}
	if o := mustParse(t, "--timeout", "1s"); o.Timeout != 1*time.Second {
		fail("1s parsed as %v", o.Timeout)
	}
	mustFail(t, "--timeout", "999ms")
	mustFail(t, "--timeout", "121s")
	mustFail(t, "--timeout", "0s")
	mustFail(t, "--timeout", "notaduration")
	mustFail(t, "--timeout")
	if len(failures) > 0 {
		t.Fatalf("%s", strings.Join(failures, "\n"))
	}
})

var _ = ginkgo.It("TestUnknownFlagAndPositional", func() {
	t := testT{ginkgo.GinkgoT()}
	mustFail(t, "--provider", "codex")
	mustFail(t, "--version")
	mustFail(t, "extra-arg")
	mustFail(t, "--codex-home", "/a", "extra")
	mustFail(t, "-x")
	mustFail(t, "-")
	mustFail(t, "--")
})

var _ = ginkgo.It("TestScalarFlagsOnlyOnce", func() {
	t := testT{ginkgo.GinkgoT()}
	mustFail(t, "--timeout", "30s", "--timeout", "40s")
	mustFail(t, "--grok-account", "a", "--grok-account", "b")
	mustFail(t, "--codex-bin", "codex2", "--codex-bin", "codex3")
	mustFail(t, "--claude-bin", "claude2", "--claude-bin", "claude3")
	mustFail(t, "--help", "--help")
	mustFail(t, "--json", "--json")
})

var _ = ginkgo.It("TestEqualsForm", func() {
	t := testT{ginkgo.GinkgoT()}
	o := mustParse(t, "--codex-home=/data/eq", "--timeout=40s", "--grok-account=lab")
	if o.CodexHomes[0] != "/data/eq" || o.Timeout != 40*time.Second || o.GrokAccount != "lab" {
		t.Fatalf("equals form broken: %+v", o)
	}
	mustFail(t, "--help=true")
	mustFail(t, "--json=true")
})

var _ = ginkgo.It("TestHelp", func() {
	t := testT{ginkgo.GinkgoT()}
	var failures []string
	fail := func(format string, args ...any) { failures = append(failures, fmt.Sprintf(format, args...)) }
	o := mustParse(t, "--help")
	if !o.Help {
		fail("%s", "help not parsed")
	}
	o = mustParse(t, "-h")
	if !o.Help {
		fail("%s", "-h not parsed as help")
	}
	mustFail(t, "--help", "--timeout", "bogus")
	mustFail(t, "--help", "--unknown")
	if len(failures) > 0 {
		t.Fatalf("%s", strings.Join(failures, "\n"))
	}
})

var _ = ginkgo.It("TestJSONFlag", func() {
	t := testT{ginkgo.GinkgoT()}
	if mustParse(t).JSON {
		t.Fatal("default output must remain the table")
	}
	o := mustParse(t, "--json", "--timeout", "5s")
	if !o.JSON || o.Timeout != 5*time.Second {
		t.Fatalf("JSON option or adjacent flag not parsed: %+v", o)
	}
})

var _ = ginkgo.It("TestParseFromEnvironment", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	t.Setenv("HOME", home)
	wd := t.TempDir()
	t.Chdir(wd)
	o, err := Parse([]string{"--codex-home", "~/.codex-work"})
	if err != nil {
		t.Fatal(err)
	}
	if o.CodexHomes[0] != filepath.Join(home, ".codex-work") {
		t.Fatalf("public Parse tilde expansion = %q", o.CodexHomes[0])
	}
})

// TestGrokAccountAliasNormalization pins the audit's finding: an alias
// with no displayable characters is a usage error before any provider
// launch, and a padded alias is stored as its normalized label.
var _ = ginkgo.It("TestGrokAccountAliasNormalization", func() {
	t := testT{ginkgo.GinkgoT()}
	mustFail(t, "--grok-account", "   ")
	mustFail(t, "--grok-account", "\t\n")
	mustFail(t, "--grok-account", "\u200e\u202e")

	o, err := Parse([]string{"--grok-account", "  my\u00a0grok  "})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if o.GrokAccount != "my grok" {
		t.Fatalf("alias = %q, want normalized %q", o.GrokAccount, "my grok")
	}
	// The normalized alias is what targets carry; no provider is
	// launched during parsing by construction.
	tgt := o.Targets()
	if tgt[len(tgt)-2].FallbackAccount != "my grok" {
		t.Fatalf("target alias = %q", tgt[len(tgt)-2].FallbackAccount)
	}
})

// TestGrokAccountUnicodeWhitespaceRejected pins the confirmation
// review's finding: aliases built only from Unicode spaces (U+3000,
// U+205F, U+1680) are usage errors, not silently accepted labels.
var _ = ginkgo.It("TestGrokAccountUnicodeWhitespaceRejected", func() {
	t := testT{ginkgo.GinkgoT()}
	mustFail(t, "--grok-account", "\u3000")
	mustFail(t, "--grok-account", "\u205f")
	mustFail(t, "--grok-account", "\u1680")
	mustFail(t, "--grok-account", " \u3000\t\u205f\u1680\n")

	o, err := Parse([]string{"--grok-account", "\u3000계정\u3000"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if o.GrokAccount != "계정" {
		t.Fatalf("alias = %q, want trimmed CJK label %q", o.GrokAccount, "계정")
	}
})
