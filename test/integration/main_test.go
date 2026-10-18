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
		isAgyVersion := false
		isAgyUsage := false
		isCursorStatus := len(os.Args) == 4 && os.Args[1] == "status" && os.Args[2] == "--format" && os.Args[3] == "json"
		isZCodeDoctor := len(os.Args) == 3 && os.Args[1] == "doctor" && os.Args[2] == "--json"
		for _, arg := range os.Args[1:] {
			if arg == "--safe-mode" {
				isClaude = true
			}
			if arg == "auth" {
				isClaudeAuth = true
			}
			if arg == "--version" {
				isAgyVersion = true
			}
			if arg == "--print" {
				isAgyUsage = true
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
		if isAgyVersion {
			_, _ = os.Stdout.WriteString("1.4.0\n")
			os.Exit(0)
		}
		if isAgyUsage {
			_, _ = os.Stdout.WriteString(`{"status":"SUCCESS","num_turns":0,"usage":{"total_tokens":0},"command":{"name":"usage","data":{"groups":[{"name":"Gemini Models","buckets":[{"id":"gemini-weekly","name":"Weekly Limit Remaining","window":"weekly","remaining_fraction":0.2584281265735626,"reset_time":"2026-10-21T15:42:40Z"},{"id":"gemini-5h","name":"Five Hour Limit Remaining","window":"5h","remaining_fraction":0.85,"reset_time":"2026-10-16T18:01:19Z"}]},{"name":"Claude and GPT models","buckets":[{"id":"3p-weekly","name":"Weekly Limit Remaining","window":"weekly","remaining_fraction":0.8723080158233643,"reset_time":"2026-10-23T15:13:44Z"},{"id":"3p-5h","name":"Five Hour Limit Remaining","window":"5h","remaining_fraction":0.74,"reset_time":"2026-10-16T20:13:44Z"}]}]}}}`)
			os.Exit(0)
		}
		if isCursorStatus {
			payload := os.Getenv("GAEBOKCHI_FAKE_CURSOR_PAYLOAD")
			if payload == "" {
				payload = `{"status":"authenticated","isAuthenticated":true,"hasAccessToken":true,"hasRefreshToken":true,"hasApiKey":false,"planInfo":{"planName":"Ultra","includedUsagePeriod":2,"includedUsageResetsAtMs":"1792591073284","billingCycleEndMs":"1794664361314"},"planError":null,"currentPeriodUsage":{"billingCycleStartMs":"1791986273284","billingCycleEndMs":"1792591073284","enabled":true,"planUsage":{"totalSpend":0,"includedSpend":0,"bonusSpend":0,"remaining":0,"limit":0,"autoPercentUsed":100,"apiPercentUsed":0,"totalPercentUsed":25}},"usageError":null}`
			}
			_, _ = os.Stdout.WriteString(payload + "\n")
			os.Exit(0)
		}
		if isZCodeDoctor {
			payload := os.Getenv("GAEBOKCHI_FAKE_ZCODE_PAYLOAD")
			if payload == "" {
				payload = `{"status":"authenticated","isAuthenticated":true,"family":"zai","quota":{"httpStatus":200,"code":200,"success":true,"limits":[{"type":"TOKENS_LIMIT","unit":3,"number":5,"percentage":18,"nextResetTime":"1792341502263"},{"type":"TIME_LIMIT","unit":5,"number":1,"percentage":2,"nextResetTime":"1794927968098"}]}}`
			}
			_, _ = os.Stdout.WriteString(payload + "\n")
			os.Exit(0)
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
