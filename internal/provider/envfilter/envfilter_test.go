package envfilter

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"reflect"
)

var _ = ginkgo.It("TestCodexRuleDropsAndReplaces", func() {
	t := testT{ginkgo.GinkgoT()}
	parent := []string{
		"PATH=/usr/bin:/bin",
		"HOME=/Users/tester",
		"OPENAI_API_KEY=sk-secret-value",
		"CODEX_API_KEY=sk-other-secret",
		"CODEX_HOME=/intruding/default",
		"CODEX_HOME=/intruding/second",
		"HTTPS_PROXY=http://proxy.local:8080",
	}
	got := Apply(parent, Rule{
		Drop: Codex().Drop,
		Set:  map[string]string{"CODEX_HOME": "/data/target-home"},
	})
	want := []string{
		"PATH=/usr/bin:/bin",
		"HOME=/Users/tester",
		"HTTPS_PROXY=http://proxy.local:8080",
		"CODEX_HOME=/data/target-home",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("codex env = %v, want %v", got, want)
	}
	for _, kv := range got {
		if kv == "OPENAI_API_KEY=sk-secret-value" || kv == "CODEX_API_KEY=sk-other-secret" {
			t.Fatalf("credential override leaked: %q", kv)
		}
	}
})

var _ = ginkgo.It("TestGrokRuleDropsKeys", func() {
	t := testT{ginkgo.GinkgoT()}
	parent := []string{
		"PATH=/usr/bin",
		"XAI_API_KEY=xai-secret",
		"GROK_API_KEY=grok-secret",
		"HOME=/Users/tester",
	}
	got := Apply(parent, Grok())
	want := []string{"PATH=/usr/bin", "HOME=/Users/tester"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("grok env = %v, want %v", got, want)
	}
})

var _ = ginkgo.It("TestParentEnvironmentNotMutated", func() {
	t := testT{ginkgo.GinkgoT()}
	parent := []string{"CODEX_HOME=/keep-me", "PATH=/bin"}
	_ = Apply(parent, Rule{Drop: []string{"CODEX_HOME"}})
	if parent[0] != "CODEX_HOME=/keep-me" {
		t.Fatalf("parent slice mutated: %v", parent)
	}
})

var _ = ginkgo.It("TestSetWithoutExistingEntryAppends", func() {
	t := testT{ginkgo.GinkgoT()}
	got := Apply([]string{"PATH=/bin"}, Rule{Set: map[string]string{"NEW_VAR": "v"}})
	if !contains(got, "NEW_VAR=v") || !contains(got, "PATH=/bin") {
		t.Fatalf("apply = %v", got)
	}
})

var _ = ginkgo.It("TestAgyRuleDropsPrefixesAndSetsAutoUpdateDisable", func() {
	t := testT{ginkgo.GinkgoT()}
	parent := []string{
		"PATH=/usr/bin",
		"HOME=/Users/tester",
		"ANTIGRAVITY_APP_DATA_DIR=/tmp/override",
		"JETSKI_TEST=1",
		"AGY_ADC_AUTH=true",
		"AGY_CLI_DISABLE_AUTO_UPDATE=false",
		"GEMINI_API_KEY=gem-secret",
		"GOOGLE_GEMINI_BASE_URL=https://example.invalid",
		"GOOGLE_APPLICATION_CREDENTIALS=/tmp/adc.json",
		"GOOGLE_CLOUD_QUOTA_PROJECT=my-proj",
	}
	got := Apply(parent, Agy())
	want := []string{
		"PATH=/usr/bin",
		"HOME=/Users/tester",
		"AGY_CLI_DISABLE_AUTO_UPDATE=true",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("agy env = %v, want %v", got, want)
	}
})

var _ = ginkgo.It("TestCursorRuleDropsPrefixesAndReplacesNodeOptions", func() {
	t := testT{ginkgo.GinkgoT()}
	parent := []string{
		"PATH=/usr/bin",
		"HOME=/Users/tester",
		"CURSOR_API_KEY=crsr-secret",
		"CURSOR_AUTH_TOKEN=tok-secret",
		"CURSOR_CONFIG_DIR=/tmp/cursor",
		"CURSOR_STATSIG_OVERRIDES={}",
		"NODE_OPTIONS=--inspect",
	}
	r := Cursor()
	r.Set = map[string]string{"NODE_OPTIONS": "--require=/tmp/preload.cjs"}
	got := Apply(parent, r)
	want := []string{
		"PATH=/usr/bin",
		"HOME=/Users/tester",
		"NODE_OPTIONS=--require=/tmp/preload.cjs",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cursor env = %v, want %v", got, want)
	}
})

var _ = ginkgo.It("TestZCodeRuleDropsPrefixesAndReplacesNodeOptions", func() {
	t := testT{ginkgo.GinkgoT()}
	parent := []string{
		"PATH=/usr/bin",
		"HOME=/Users/tester",
		"ZCODE_DATA_BASE_DIR=/tmp/zcode",
		"ZCODE_CREDENTIAL_SECRET=secret",
		"ZAI_BUSINESS_BASE_URL=https://override.invalid",
		"BIGMODEL_API_ORIGIN=https://override.invalid",
		"NODE_OPTIONS=--inspect",
	}
	r := ZCode()
	r.Set = map[string]string{"NODE_OPTIONS": "--require=/tmp/preload.cjs"}
	got := Apply(parent, r)
	want := []string{
		"PATH=/usr/bin",
		"HOME=/Users/tester",
		"NODE_OPTIONS=--require=/tmp/preload.cjs",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("zcode env = %v, want %v", got, want)
	}
})

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
