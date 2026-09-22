package options

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	testHome = "/Users/tester"
	testWD   = "/Users/tester/work/project"
)

func mustParse(t *testing.T, args ...string) Options {
	t.Helper()
	o, err := parse(args, testHome, testWD)
	if err != nil {
		t.Fatalf("parse(%q) failed: %v", args, err)
	}
	return o
}

func mustFail(t *testing.T, args ...string) error {
	t.Helper()
	o, err := parse(args, testHome, testWD)
	if err == nil {
		t.Fatalf("parse(%q) unexpectedly succeeded: %+v", args, o)
	}
	if !IsUsageError(err) {
		t.Fatalf("parse(%q) error is not a usage error: %v", args, err)
	}
	return err
}

func TestDefaultTargets(t *testing.T) {
	o := mustParse(t)
	if len(o.CodexHomes) != 2 {
		t.Fatalf("default homes = %v, want 2", o.CodexHomes)
	}
	if o.CodexHomes[0] != testHome+"/.codex" || o.CodexHomes[1] != testHome+"/.codex-hsy" {
		t.Fatalf("default homes = %v", o.CodexHomes)
	}
	targets := o.Targets()
	if len(targets) != 3 {
		t.Fatalf("default target count = %d, want 3", len(targets))
	}
	wantIDs := []string{"codex:0", "codex:1", "grok:0"}
	wantProviders := []string{"codex", "codex", "grok"}
	wantAccounts := []string{".codex", ".codex-hsy", "default"}
	for i, tt := range targets {
		if tt.ID != wantIDs[i] || string(tt.Provider) != wantProviders[i] {
			t.Errorf("target %d = %s/%s, want %s/%s", i, tt.ID, tt.Provider, wantIDs[i], wantProviders[i])
		}
		if tt.FallbackAccount != wantAccounts[i] {
			t.Errorf("target %d fallback = %q, want %q", i, tt.FallbackAccount, wantAccounts[i])
		}
		if tt.Index != i {
			t.Errorf("target %d index = %d", i, tt.Index)
		}
	}
	if targets[0].Home == targets[1].Home || targets[2].Home != "" {
		t.Error("codex homes must differ and grok home must be empty")
	}
	if o.Timeout != 20*time.Second || o.CodexBin != "codex" || o.GrokBin != "grok" || o.GrokAccount != "default" {
		t.Errorf("defaults not applied: %+v", o)
	}
}

// The options package must not read the environment at all, so an ambient
// CODEX_HOME cannot change the default account list.
func TestAmbientCodexHomeIgnored(t *testing.T) {
	t.Setenv("CODEX_HOME", "/somewhere/else")
	o := mustParse(t)
	if o.CodexHomes[0] != testHome+"/.codex" {
		t.Fatalf("ambient CODEX_HOME leaked into defaults: %v", o.CodexHomes)
	}
}

func TestCodexHomeReplacementOrder(t *testing.T) {
	o := mustParse(t, "--codex-home", "/data/second", "--codex-home", "/data/first")
	if len(o.CodexHomes) != 2 || o.CodexHomes[0] != "/data/second" || o.CodexHomes[1] != "/data/first" {
		t.Fatalf("replacement order not preserved: %v", o.CodexHomes)
	}
	targets := o.Targets()
	if len(targets) != 3 || targets[2].Provider != "grok" {
		t.Fatalf("grok must stay last, got %+v", targets)
	}
}

func TestTildeExpansion(t *testing.T) {
	o := mustParse(t, "--codex-home", "~", "--codex-home", "~/x/y/../z")
	if o.CodexHomes[0] != testHome {
		t.Errorf(`~ expanded to %q`, o.CodexHomes[0])
	}
	if o.CodexHomes[1] != testHome+"/x/z" {
		t.Errorf(`~/x/y/../z expanded to %q`, o.CodexHomes[1])
	}
}

func TestTildeOtherUserRejected(t *testing.T) {
	if err := mustFail(t, "--codex-home", "~other/cache"); err == nil {
		t.Fatal("expected failure")
	}
}

func TestRelativeHomeResolvedAgainstWD(t *testing.T) {
	o := mustParse(t, "--codex-home", "state/a")
	if o.CodexHomes[0] != testWD+"/state/a" {
		t.Fatalf("relative home = %q", o.CodexHomes[0])
	}
}

func TestEmptyValuesRejected(t *testing.T) {
	mustFail(t, "--codex-home", "")
	mustFail(t, "--codex-home=")
	mustFail(t, "--grok-account", "")
	mustFail(t, "--codex-bin", "")
	mustFail(t, "--grok-bin", "")
}

func TestDuplicateHomesRejected(t *testing.T) {
	mustFail(t, "--codex-home", "/data/a", "--codex-home", "/data/a")
	mustFail(t, "--codex-home", "/data/a", "--codex-home", "/data/./a")
	mustFail(t, "--codex-home", "/data/a/", "--codex-home", "/data/a")
}

func TestDuplicateHomesThroughSymlink(t *testing.T) {
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
}

func TestMissingHomeIsValid(t *testing.T) {
	o := mustParse(t, "--codex-home", "/definitely/not/here")
	if o.CodexHomes[0] != "/definitely/not/here" {
		t.Fatalf("missing home altered: %v", o.CodexHomes)
	}
}

func TestTimeoutValidation(t *testing.T) {
	if o := mustParse(t, "--timeout", "40s"); o.Timeout != 40*time.Second {
		t.Errorf("40s parsed as %v", o.Timeout)
	}
	if o := mustParse(t, "--timeout", "120s"); o.Timeout != 120*time.Second {
		t.Errorf("120s parsed as %v", o.Timeout)
	}
	if o := mustParse(t, "--timeout", "1s"); o.Timeout != 1*time.Second {
		t.Errorf("1s parsed as %v", o.Timeout)
	}
	mustFail(t, "--timeout", "999ms")
	mustFail(t, "--timeout", "121s")
	mustFail(t, "--timeout", "0s")
	mustFail(t, "--timeout", "notaduration")
	mustFail(t, "--timeout")
}

func TestUnknownFlagAndPositional(t *testing.T) {
	mustFail(t, "--provider", "codex")
	mustFail(t, "--json")
	mustFail(t, "extra-arg")
	mustFail(t, "--codex-home", "/a", "extra")
	mustFail(t, "-x")
	mustFail(t, "-")
	mustFail(t, "--")
}

func TestScalarFlagsOnlyOnce(t *testing.T) {
	mustFail(t, "--timeout", "30s", "--timeout", "40s")
	mustFail(t, "--grok-account", "a", "--grok-account", "b")
	mustFail(t, "--codex-bin", "codex2", "--codex-bin", "codex3")
	mustFail(t, "--version", "--version")
	mustFail(t, "--help", "--help")
}

func TestEqualsForm(t *testing.T) {
	o := mustParse(t, "--codex-home=/data/eq", "--timeout=40s", "--grok-account=lab")
	if o.CodexHomes[0] != "/data/eq" || o.Timeout != 40*time.Second || o.GrokAccount != "lab" {
		t.Fatalf("equals form broken: %+v", o)
	}
	mustFail(t, "--help=true")
}

func TestHelpAndVersion(t *testing.T) {
	o := mustParse(t, "--help")
	if !o.Help || o.Version {
		t.Error("help not parsed")
	}
	o = mustParse(t, "--version")
	if !o.Version || o.Help {
		t.Error("version not parsed")
	}
	o = mustParse(t, "-h")
	if !o.Help {
		t.Error("-h not parsed as help")
	}
	// Both supplied: help wins, but syntactic validity is still required.
	o = mustParse(t, "--version", "--help")
	if !o.Help || !o.Version {
		t.Error("help+version combination broken")
	}
	mustFail(t, "--help", "--timeout", "bogus")
	mustFail(t, "--help", "--unknown")
}

func TestParseFromEnvironment(t *testing.T) {
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
}

// TestGrokAccountAliasNormalization pins the audit's finding: an alias
// with no displayable characters is a usage error before any provider
// launch, and a padded alias is stored as its normalized label.
func TestGrokAccountAliasNormalization(t *testing.T) {
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
	if tgt[len(tgt)-1].FallbackAccount != "my grok" {
		t.Fatalf("target alias = %q", tgt[len(tgt)-1].FallbackAccount)
	}
}
