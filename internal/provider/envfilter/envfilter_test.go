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

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
