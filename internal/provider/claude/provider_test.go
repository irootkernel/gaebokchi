package claude

import (
	"context"
	"fmt"
	ginkgo "github.com/onsi/ginkgo/v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gaebokchi/internal/domain"
)

func usageJSON(result string, turns int, cost float64) []byte {
	return []byte(fmt.Sprintf(`{"local_command":"usage","num_turns":%d,"total_cost_usd":%g,"is_error":false,"result":%q}`,
		turns, cost, result))
}

var _ = ginkgo.Describe("TestParseUsageAcceptsOnlyWeeklyAllModels", func() {
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
		ginkgo.It(tc.name, func() {
			t := testT{ginkgo.GinkgoT()}
			got, _, status := parseUsage(usageJSON(tc.text, tc.turns, tc.cost), time.Now(), time.Now())
			if status != tc.status || got != tc.want {
				t.Fatalf("parseUsage = %v/%v, want %v/%v", got, status, tc.want, tc.status)
			}
		})
	}
	for _, raw := range [][]byte{[]byte(`not json`), []byte(`{"result":"Current week (all models): 1% used"}`),
		[]byte(`{"local_command":"other","num_turns":0,"total_cost_usd":0,"is_error":false,"result":"Current week (all models): 1% used"}`)} {
		ginkgo.It(string(raw), func() {
			t := testT{ginkgo.GinkgoT()}
			if _, _, status := parseUsage(raw, time.Now(), time.Now()); status != domain.StatusUnsupported {
				t.Fatalf("unexpected acceptance of %q: %v", raw, status)
			}
		})
	}
})

var _ = ginkgo.Describe("TestParseUsageReadsOptionalWeeklyReset", func() {
	now := time.Date(2026, 9, 27, 15, 0, 0, 0, time.UTC)
	seoulNewYear := time.Date(2026, 12, 31, 20, 0, 0, 0, time.UTC)
	cases := []struct {
		name, text string
		start, end time.Time
		want       string // RFC 3339; empty when no reset is established
	}{
		{"whole hour", "Current week (all models): 54% used · resets Oct 3 at 10am (Asia/Seoul)", now, now, "2026-10-03T10:00:00+09:00"},
		{"minutes", "Current week (all models): 54% used · resets Oct 3 at 9:59pm (Asia/Seoul)", now, now, "2026-10-03T21:59:00+09:00"},
		{"noon", "Current week (all models): 54% used · resets Oct 3 at 12pm (UTC)", now, now, "2026-10-03T12:00:00Z"},
		{"explicit year at midnight", "Current week (all models): 54% used · resets Jan 2, 2027 at 12am (America/Los_Angeles)", now, now, "2027-01-02T00:00:00-08:00"},
		{"year of the named zone", "Current week (all models): 54% used · resets Jan 3 at 10am (Asia/Seoul)", seoulNewYear, seoulNewYear, "2027-01-03T10:00:00+09:00"},
		{"zone year changed during the query", "Current week (all models): 54% used · resets Dec 31 at 11:59pm (America/Los_Angeles)",
			time.Date(2027, 1, 1, 7, 59, 58, 0, time.UTC), time.Date(2027, 1, 1, 8, 0, 3, 0, time.UTC), ""},
		{"comma separator", "Current week (all models): 54% used · resets Oct 3, 9:59pm (Asia/Seoul)", now, now, "2026-10-03T21:59:00+09:00"},
		{"comma separator with year", "Current week (all models): 54% used · resets Jan 2, 2027, 12am (America/Los_Angeles)", now, now, "2027-01-02T00:00:00-08:00"},
		{"time repeated by daylight saving", "Current week (all models): 54% used · resets Nov 1 at 1:30am (America/Los_Angeles)", now, now, ""},
		{"time skipped by daylight saving", "Current week (all models): 54% used · resets Mar 8 at 2:30am (America/Los_Angeles)", now, now, ""},
		{"single time after a daylight-saving change", "Current week (all models): 54% used · resets Nov 1 at 2am (America/Los_Angeles)", now, now, "2026-11-01T02:00:00-08:00"},
		{"unknown zone", "Current week (all models): 54% used · resets Oct 3 at 10am (Mars/Olympus)", now, now, ""},
		{"unrecognized text", "Current week (all models): 54% used · resets later", now, now, ""},
		{"impossible date", "Current week (all models): 54% used · resets Feb 30 at 10am (UTC)", now, now, ""},
		{"absent", "Current week (all models): 54% used", now, now, ""},
		{"another meter's reset", "Current session: 5% used · resets Sep 27 at 3pm (UTC)\nCurrent week (all models): 54% used", now, now, ""},
	}
	for _, tc := range cases {
		ginkgo.It(tc.name, func() {
			t := testT{ginkgo.GinkgoT()}
			got, reset, status := parseUsage(usageJSON(tc.text, 0, 0), tc.start, tc.end)
			if status != domain.StatusOK || got != 54 {
				t.Fatalf("parseUsage = %v/%v, want 54/OK", got, status)
			}
			if tc.want == "" {
				if reset != nil {
					t.Fatalf("unexpected reset %v", reset)
				}
				return
			}
			want, err := time.Parse(time.RFC3339, tc.want)
			if err != nil {
				t.Fatal(err)
			}
			if reset == nil || !reset.Equal(want) {
				t.Fatalf("reset = %v, want %v", reset, want)
			}
		})
	}
})

func fakeClaude(t testT, auth, usage string) (bin, log string) {
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

var _ = ginkgo.It("TestQueryUsesOnlyLocalCommandAndPreservesMetadata", func() {
	t := testT{ginkgo.GinkgoT()}
	bin, log := fakeClaude(t, `{"loggedIn":true,"authMethod":"claude.ai","apiProvider":"firstParty"}`,
		string(usageJSON("Current week (all models): 43.2% used · resets Oct 3, 2027 at 10am (UTC)", 0, 0)))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := New().Query(ctx, claudeTarget(bin))
	if result.Status != domain.StatusOK || result.UsedPercent == nil || *result.UsedPercent != 43.2 ||
		result.Period != domain.PeriodWeekly || result.QuotaID != "claude_all_models" || result.Source != domain.SourceClaudeCLIUsage ||
		result.ResetAt == nil || !result.ResetAt.Equal(time.Date(2027, 10, 3, 10, 0, 0, 0, time.UTC)) {
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
})

var _ = ginkgo.Describe("TestQueryAuthAndMissingExecutable", func() {
	cases := []struct {
		name, auth string
		want       domain.Status
	}{
		{"logged out", `{"loggedIn":false}`, domain.StatusAuthRequired},
		{"API key", `{"loggedIn":true,"authMethod":"apiKey","apiProvider":"firstParty"}`, domain.StatusNotApplicableAPIKey},
		{"unknown auth", `{"loggedIn":true,"authMethod":"bedrock","apiProvider":"aws"}`, domain.StatusUnsupported},
	}
	for _, tc := range cases {
		ginkgo.It(tc.name, func() {
			t := testT{ginkgo.GinkgoT()}
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
	ginkgo.It("missing executable", func() {
		t := testT{ginkgo.GinkgoT()}
		missing := New().Query(context.Background(), claudeTarget(filepath.Join(t.TempDir(), "missing")))
		if missing.Status != domain.StatusCLINotFound {
			t.Fatalf("missing CLI status = %v", missing.Status)
		}
	})
})

var _ = ginkgo.It("TestQueryUsageTimeout", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

var _ = ginkgo.It("TestChildEnvironmentDropsBillingOverrides", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
