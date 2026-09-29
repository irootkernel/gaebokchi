// Package integration qualifies the actual gaebokchi executable with
// fake provider CLIs: no real accounts, credentials, provider
// installations, or network access.
package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"gaebokchi/internal/testutil"
)

// ownedDirs records the temporary directories created by this test
// invocation. Only registered directories are removed after the run:
// a similarly named directory owned by a concurrent invocation, a
// crashed earlier run, or the user is never touched.
var ownedDirs []string

func registerOwnedDir(dir string) { ownedDirs = append(ownedDirs, dir) }

// cleanupOwnedDirs removes exactly the directories this invocation
// created. Broad prefix matching is not ownership evidence.
func cleanupOwnedDirs() {
	for _, dir := range ownedDirs {
		_ = os.RemoveAll(dir)
	}
	ownedDirs = nil
}

func fakePercent(variable, home string, fallback int) int {
	raw := os.Getenv(variable)
	if raw == "" {
		return fallback
	}
	var byHome map[string]int
	if err := json.Unmarshal([]byte(raw), &byHome); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	percent, ok := byHome[home]
	if !ok {
		fmt.Fprintf(os.Stderr, "no fake usage for home %q\n", home)
		os.Exit(1)
	}
	return percent
}

// TestMain is the fake-provider entry point: the gaebokchi binary under
// test launches this same test binary as its codex/grok child, and the
// argv decides which provider script runs.
func TestMain(m *testing.M) {
	if os.Getenv("GAEBOKCHI_FAKE") == "1" {
		isClaude := false
		isClaudeAuth := false
		for _, arg := range os.Args[1:] {
			if arg == "--safe-mode" {
				isClaude = true
			}
			if arg == "auth" {
				isClaudeAuth = true
			}
		}
		if path := os.Getenv("GAEBOKCHI_FAKE_ARGLOG"); path != "" {
			argv := "argv:"
			for _, a := range os.Args[1:] {
				argv += " " + a
			}
			f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			if err == nil {
				_, _ = f.WriteString(argv + "\n")
				_ = f.Close()
			}
		}
		if p := os.Getenv("GAEBOKCHI_FAKE_ENVLOG"); p != "" {
			var env strings.Builder
			env.WriteString("=== child")
			for _, a := range os.Args[1:] {
				env.WriteString(" " + a)
			}
			env.WriteString("\n")
			for _, kv := range os.Environ() {
				env.WriteString(kv + "\n")
			}
			f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			if err == nil {
				_, _ = f.WriteString(env.String())
				_ = f.Close()
			}
		}
		if isClaude {
			if isClaudeAuth {
				_, _ = os.Stdout.WriteString(`{"loggedIn":true,"authMethod":"claude.ai","apiProvider":"firstParty"}`)
			} else {
				pct := fakePercent("GAEBOKCHI_FAKE_CLAUDE_USAGE_BY_HOME", os.Getenv("CLAUDE_CONFIG_DIR"), 43)
				_, _ = fmt.Fprintf(os.Stdout, `{"local_command":"usage","num_turns":0,"total_cost_usd":0,"is_error":false,"result":%q}`,
					fmt.Sprintf("Current week (all models): %d%% used", pct))
			}
			os.Exit(0)
		}
		steps := os.Getenv("GAEBOKCHI_FAKE_STEPS")
		if steps == "" {
			// The provider kind is selected by argv: codex children are
			// invoked as "app-server", grok children as "agent stdio".
			isCodex, isGrok := false, false
			for _, a := range os.Args[1:] {
				if a == "app-server" {
					isCodex = true
				}
				if a == "agent" || a == "stdio" {
					isGrok = true
				}
			}
			if isCodex && !isGrok {
				steps = os.Getenv("GAEBOKCHI_FAKE_CODEX_STEPS")
			} else {
				steps = os.Getenv("GAEBOKCHI_FAKE_GROK_STEPS")
				if os.Getenv("GAEBOKCHI_FAKE_GROK_USAGE_BY_HOME") != "" {
					pct := fakePercent("GAEBOKCHI_FAKE_GROK_USAGE_BY_HOME", os.Getenv("GROK_HOME"), 0)
					encoded, err := json.Marshal(grokWeekly(pct))
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
						os.Exit(1)
					}
					steps = string(encoded)
				}
			}
		}
		if err := os.Setenv("GAEBOKCHI_FAKE_STEPS", steps); err != nil {
			os.Exit(1)
		}
		if err := testutil.RunFakeScript(os.Stdin, os.Stdout, os.Stderr); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	code := m.Run()
	cleanupOwnedDirs()
	os.Exit(code)
}
