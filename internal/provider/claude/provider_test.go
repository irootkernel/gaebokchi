package claude

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gaebokchi/internal/domain"
)

func usageJSON(result string, turns int, cost float64) []byte {
	return []byte(fmt.Sprintf(`{"local_command":"usage","num_turns":%d,"total_cost_usd":%g,"is_error":false,"result":%q}`,
		turns, cost, result))
}

func TestParseUsageAcceptsOnlyWeeklyAllModels(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		turns  int
		cost   float64
		want   float64
		status domain.Status
	}{
		{"zero and other meters", "Current session: 75% used\nCurrent week (all models): 0% used · resets later\nCurrent week (Fable): 90% used", 0, 0, 0, domain.StatusOK},
		{"decimal", "Current week (all models): 43.25% used", 0, 0, 43.25, domain.StatusOK},
		{"missing", "Current week (Fable): 43% used", 0, 0, 0, domain.StatusUnsupported},
		{"duplicate", "Current week (all models): 2% used\nCurrent week (all models): 3% used", 0, 0, 0, domain.StatusUnsupported},
		{"over 100", "Current week (all models): 101% used", 0, 0, 0, domain.StatusProtocolError},
		{"negative", "Current week (all models): -1% used", 0, 0, 0, domain.StatusUnsupported},
		{"model turn", "Current week (all models): 1% used", 1, 0, 0, domain.StatusUnsupported},
		{"cost", "Current week (all models): 1% used", 0, 0.01, 0, domain.StatusUnsupported},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, status := parseUsage(usageJSON(tc.text, tc.turns, tc.cost))
			if status != tc.status || got != tc.want {
				t.Fatalf("parseUsage = %v/%v, want %v/%v", got, status, tc.want, tc.status)
			}
		})
	}
	for _, raw := range [][]byte{[]byte(`not json`), []byte(`{"result":"Current week (all models): 1% used"}`),
		[]byte(`{"local_command":"other","num_turns":0,"total_cost_usd":0,"is_error":false,"result":"Current week (all models): 1% used"}`)} {
		if _, status := parseUsage(raw); status != domain.StatusUnsupported {
			t.Fatalf("unexpected acceptance of %q: %v", raw, status)
		}
	}
}

func fakeClaude(t *testing.T, auth, usage string) (bin, log string) {
	t.Helper()
	dir := t.TempDir()
	bin = filepath.Join(dir, "claude")
	log = filepath.Join(dir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + log + "\n" +
		"case \" $* \" in\n" +
		"  *' auth status '*) printf '%s' '" + auth + "' ;;\n" +
		"  *) printf '%s' '" + usage + "' ;;\n" +
		"esac\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return bin, log
}

func claudeTarget(bin string) domain.Target {
	return domain.Target{ID: "claude:0", Provider: domain.ProviderClaude, FallbackAccount: "default", Bin: bin}
}

func TestQueryUsesOnlyLocalCommandAndPreservesMetadata(t *testing.T) {
	bin, log := fakeClaude(t, `{"loggedIn":true,"authMethod":"claude.ai","apiProvider":"firstParty"}`,
		string(usageJSON("Current week (all models): 43.2% used", 0, 0)))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := New().Query(ctx, claudeTarget(bin))
	if result.Status != domain.StatusOK || result.UsedPercent == nil || *result.UsedPercent != 43.2 ||
		result.Period != domain.PeriodWeekly || result.QuotaID != "claude_all_models" || result.Source != domain.SourceClaudeCLIUsage {
		t.Fatalf("unexpected result: %+v", result)
	}
	args, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(args)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "--safe-mode auth status --json") ||
		!strings.Contains(lines[1], "--safe-mode --tools  -p /usage --max-turns 0 --output-format json --no-session-persistence") {
		t.Fatalf("unexpected claude commands: %q", args)
	}
}

func TestQueryAuthAndMissingExecutable(t *testing.T) {
	cases := []struct {
		name, auth string
		want       domain.Status
	}{
		{"logged out", `{"loggedIn":false}`, domain.StatusAuthRequired},
		{"API key", `{"loggedIn":true,"authMethod":"apiKey","apiProvider":"firstParty"}`, domain.StatusNotApplicableAPIKey},
		{"unknown auth", `{"loggedIn":true,"authMethod":"bedrock","apiProvider":"aws"}`, domain.StatusUnsupported},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin, log := fakeClaude(t, tc.auth, string(usageJSON("Current week (all models): 5% used", 0, 0)))
			result := New().Query(context.Background(), claudeTarget(bin))
			if result.Status != tc.want {
				t.Fatalf("status = %v, want %v", result.Status, tc.want)
			}
			args, err := os.ReadFile(log)
			if err != nil || strings.Count(string(args), "\n") != 1 {
				t.Fatalf("usage launched despite auth outcome: %q, %v", args, err)
			}
		})
	}
	missing := New().Query(context.Background(), claudeTarget(filepath.Join(t.TempDir(), "missing")))
	if missing.Status != domain.StatusCLINotFound {
		t.Fatalf("missing CLI status = %v", missing.Status)
	}
}

func TestQueryUsageTimeout(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "claude")
	script := `#!/bin/sh
case " $* " in
  *' auth status '*) printf '%s' '{"loggedIn":true,"authMethod":"claude.ai","apiProvider":"firstParty"}' ;;
  *) exec sleep 5 ;;
esac
`
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	result := New().Query(ctx, claudeTarget(bin))
	if result.Status != domain.StatusTimeout {
		t.Fatalf("status = %v, want TIMEOUT", result.Status)
	}
}

func TestChildEnvironmentDropsBillingOverrides(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "secret")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "secret")
	t.Setenv("CLAUDE_CODE_USE_BEDROCK", "1")
	t.Setenv("CLAUDE_CONFIG_DIR", "/chosen/claude")
	env := childEnv()
	for _, kv := range env {
		if strings.HasPrefix(kv, "ANTHROPIC_API_KEY=") || strings.HasPrefix(kv, "ANTHROPIC_AUTH_TOKEN=") ||
			strings.HasPrefix(kv, "CLAUDE_CODE_USE_BEDROCK=") {
			t.Fatalf("billing override leaked to Claude: %q", kv)
		}
	}
	if !contains(env, "CLAUDE_CONFIG_DIR=/chosen/claude") {
		t.Fatal("selected Claude config dir was lost")
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
