package render

import (
	"strings"

	ginkgo "github.com/onsi/ginkgo/v2"

	"gaebokchi/internal/domain"
)

var _ = ginkgo.It("TestJSONSnapshotContract", func() {
	t := testT{ginkgo.GinkgoT()}
	results := []domain.Result{
		{TargetID: "codex:0", Provider: domain.ProviderCodex, Account: " a\tb@example.com ", Status: domain.StatusOK, UsedPercent: pct(12.25), Source: "secret-source"},
		{TargetID: "codex:1", Provider: domain.ProviderCodex, Account: "other", Status: domain.StatusOK, UsedPercent: pct(0), CleanupFailed: true,
			Diagnostic: domain.Diagnostic{Code: "cleanup_failed", Message: "secret-message"}},
		{TargetID: "grok:0", Provider: domain.ProviderGrok, Account: "default", Status: domain.StatusNotApplicableMonthly},
		{TargetID: "claude:0", Provider: domain.ProviderClaude, Account: "default", Status: domain.StatusAuthRequired},
	}
	got, err := JSON(results)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"results":[{"target_id":"codex:0","provider":"codex","account":"a b@example.com","status":"ok","used_percent":12.25,"cleanup_failed":false},{"target_id":"codex:1","provider":"codex","account":"other","status":"ok","used_percent":0,"cleanup_failed":true},{"target_id":"grok:0","provider":"grok","account":"default","status":"not_applicable/monthly","used_percent":null,"cleanup_failed":false},{"target_id":"claude:0","provider":"claude","account":"default","status":"auth_required","used_percent":null,"cleanup_failed":false}]}` + "\n"
	if got != want {
		t.Fatalf("JSON contract mismatch:\n got: %s\nwant: %s", got, want)
	}
	if strings.Contains(got, "secret") || strings.Contains(got, "AI PROVIDER") {
		t.Fatal("private fields or table leaked into JSON")
	}
})
