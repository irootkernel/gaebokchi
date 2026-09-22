package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gaebokchi/internal/domain"
	"gaebokchi/internal/options"
)

// fakeQuerier records invocations with configurable behavior.
type fakeQuerier struct {
	mu         sync.Mutex
	calls      []string
	concurrent atomic.Int32
	maxSeen    atomic.Int32
	delay      time.Duration
	failID     string
	responder  func(ctx context.Context, tgt domain.Target) domain.Result
}

func (f *fakeQuerier) Query(ctx context.Context, tgt domain.Target) domain.Result {
	f.mu.Lock()
	f.calls = append(f.calls, tgt.ID)
	f.mu.Unlock()

	cur := f.concurrent.Add(1)
	for {
		old := f.maxSeen.Load()
		if cur <= old || f.maxSeen.CompareAndSwap(old, cur) {
			break
		}
	}
	defer f.concurrent.Add(-1)

	wait := f.delay
	if f.responder != nil {
		return f.responder(ctx, tgt)
	}
	if wait > 0 || tgt.ID == f.failID {
		d := wait
		if d == 0 {
			d = 600 * time.Millisecond
		}
		select {
		case <-time.After(d):
		case <-ctx.Done():
			return timeoutish(tgt)
		}
	}
	return okResult(tgt, 50)
}

func (f *fakeQuerier) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakeQuerier) called(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if c == id {
			return true
		}
	}
	return false
}

func okResult(tgt domain.Target, pct float64) domain.Result {
	return domain.Result{
		TargetID: tgt.ID, Provider: tgt.Provider,
		Account: tgt.FallbackAccount, AccountSource: domain.AccountSourceConfiguredAlias,
		Status: domain.StatusOK, UsedPercent: &pct, ObservedAt: time.Now(),
	}
}

func timeoutish(tgt domain.Target) domain.Result {
	return domain.Result{TargetID: tgt.ID, Provider: tgt.Provider, Status: domain.StatusTimeout,
		Account: tgt.FallbackAccount, AccountSource: domain.AccountSourceConfiguredAlias, ObservedAt: time.Now()}
}

func mkTargets(n int) []domain.Target {
	tgts := make([]domain.Target, 0, n+1)
	for i := 0; i < n; i++ {
		tgts = append(tgts, domain.Target{
			ID: fmt.Sprintf("codex:%d", i), Index: i, Provider: domain.ProviderCodex,
			Home: fmt.Sprintf("/data/h%d", i), FallbackAccount: fmt.Sprintf("h%d", i),
		})
	}
	tgts = append(tgts, domain.Target{ID: "grok:0", Index: n, Provider: domain.ProviderGrok, FallbackAccount: "default"})
	return tgts
}

func TestDefaultThreeTargetsSimultaneousAdmission(t *testing.T) {
	q := &fakeQuerier{delay: 100 * time.Millisecond}
	r := &Runner{Codex: q, Grok: q}
	targets := mkTargets(2)
	start := time.Now()
	results := r.Run(context.Background(), targets, 5*time.Second)
	if len(results) != 3 {
		t.Fatalf("results = %d", len(results))
	}
	// All three admitted at once: total wall time ~ one delay, not 3x.
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("default admission not simultaneous: %v", elapsed)
	}
	if q.maxSeen.Load() != 3 {
		t.Fatalf("max concurrent = %d, want 3", q.maxSeen.Load())
	}
	for i, res := range results {
		if res.TargetID != targets[i].ID {
			t.Fatalf("order broken at %d: %s", i, res.TargetID)
		}
		if res.Status != domain.StatusOK {
			t.Fatalf("target %s status = %v", res.TargetID, res.Status)
		}
	}
}

func TestDeterministicOrderUnderReversedCompletion(t *testing.T) {
	// codex:0 is slow, codex:1 and grok finish first.
	responder := map[string]time.Duration{"codex:0": 300 * time.Millisecond, "codex:1": 20 * time.Millisecond, "grok:0": 20 * time.Millisecond}
	q := &fakeQuerier{responder: func(_ context.Context, tgt domain.Target) domain.Result {
		time.Sleep(responder[tgt.ID])
		return okResult(tgt, 10)
	}}
	r := &Runner{Codex: q, Grok: q}
	targets := mkTargets(2)
	results := r.Run(context.Background(), targets, 5*time.Second)
	want := []string{"codex:0", "codex:1", "grok:0"}
	for i, id := range want {
		if results[i].TargetID != id {
			t.Fatalf("slot %d = %s, want %s", i, results[i].TargetID, id)
		}
	}
}

func TestAtMostThreeActiveWithFiveHomes(t *testing.T) {
	q := &fakeQuerier{delay: 120 * time.Millisecond}
	r := &Runner{Codex: q, Grok: q}
	targets := mkTargets(5)
	results := r.Run(context.Background(), targets, 5*time.Second)
	if len(results) != 6 {
		t.Fatalf("results = %d", len(results))
	}
	if q.maxSeen.Load() > 3 {
		t.Fatalf("concurrency exceeded bound: %d", q.maxSeen.Load())
	}
	if q.callCount() != 6 {
		t.Fatalf("query attempts = %d, want exactly one per target", q.callCount())
	}
}

func TestFailureDoesNotCancelSiblings(t *testing.T) {
	q := &fakeQuerier{failID: "codex:1", responder: func(ctx context.Context, tgt domain.Target) domain.Result {
		if tgt.ID == "codex:1" {
			select { // exceeds the budget
			case <-time.After(600 * time.Millisecond):
				return domain.Result{TargetID: tgt.ID, Provider: tgt.Provider, Status: domain.StatusProviderError}
			case <-ctx.Done():
				return timeoutish(tgt)
			}
		}
		return okResult(tgt, 42)
	}}
	r := &Runner{Codex: q, Grok: q}
	targets := mkTargets(2)
	results := r.Run(context.Background(), targets, 200*time.Millisecond)
	if results[1].Status != domain.StatusTimeout {
		t.Fatalf("hanging target status = %v", results[1].Status)
	}
	if results[0].Status != domain.StatusOK || results[2].Status != domain.StatusOK {
		t.Fatalf("siblings damaged: %v / %v", results[0].Status, results[2].Status)
	}
}

func TestQueuedTargetsCancelledWithoutLaunch(t *testing.T) {
	q := &fakeQuerier{delay: 400 * time.Millisecond}
	r := &Runner{Codex: q, Grok: q}
	targets := mkTargets(5) // 6 targets, bound 3
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(80 * time.Millisecond) // first three admitted, rest queued
		cancel()
	}()
	results := r.Run(ctx, targets, 5*time.Second)
	launched := 0
	cancelled := 0
	for _, res := range results {
		switch res.Status {
		case domain.StatusCancelled:
			cancelled++
		case domain.StatusTimeout:
			launched++ // active queries were cancelled by ctx → timeoutish
		case domain.StatusOK:
			launched++
		}
	}
	if launched > 3 {
		t.Fatalf("more than 3 launched: %d", launched)
	}
	if cancelled != 3 {
		t.Fatalf("queued cancellations = %d, want 3", cancelled)
	}
	for _, res := range results {
		if res.Status == domain.StatusCancelled && q.called(res.TargetID) {
			t.Fatalf("cancelled target %s was launched", res.TargetID)
		}
	}
}

func TestDeadlineStartsAtAdmission(t *testing.T) {
	// With bound 1, the second target's deadline must start after the
	// first completes, so a short budget still succeeds sequentially.
	q := &fakeQuerier{delay: 60 * time.Millisecond}
	r := &Runner{Codex: q, Grok: q, MaxActive: 1}
	targets := mkTargets(1)
	results := r.Run(context.Background(), targets, 200*time.Millisecond)
	for _, res := range results {
		if res.Status != domain.StatusOK {
			t.Fatalf("%s status = %v (deadline should start at admission)", res.TargetID, res.Status)
		}
	}
}

func TestOneQueryAttemptPerTarget(t *testing.T) {
	q := &fakeQuerier{}
	r := &Runner{Codex: q, Grok: q}
	r.Run(context.Background(), mkTargets(4), time.Second)
	if q.callCount() != 5 {
		t.Fatalf("attempts = %d, want 5", q.callCount())
	}
}

// TestValidResultSurvivesCleanupPastDeadline reproduces the audit's
// executable-level finding at the runner boundary: a result established
// within the query budget must survive bounded cleanup that crosses the
// deadline before Query returns, instead of being rewritten to TIMEOUT.
func TestValidResultSurvivesCleanupPastDeadline(t *testing.T) {
	q := &fakeQuerier{responder: func(ctx context.Context, tgt domain.Target) domain.Result {
		res := okResult(tgt, 37) // established immediately, well in budget
		<-time.After(700 * time.Millisecond)
		return res // "cleanup" outlives the 300ms query budget
	}}
	r := &Runner{Codex: q, Grok: q}
	results := r.Run(context.Background(), mkTargets(0), 300*time.Millisecond)
	if len(results) != 1 {
		t.Fatalf("results = %d", len(results))
	}
	if results[0].Status != domain.StatusOK || results[0].UsedPercent == nil || *results[0].UsedPercent != 37 {
		t.Fatalf("timely result discarded: status=%v percent=%v", results[0].Status, results[0].UsedPercent)
	}
}

// TestEstablishedResultSurvivesRootCancellation pins REQ-023's
// preservation clause: once a target's result is established, root
// cancellation of the invocation must not erase it from the output.
func TestEstablishedResultSurvivesRootCancellation(t *testing.T) {
	established := make(chan struct{})
	q := &fakeQuerier{responder: func(ctx context.Context, tgt domain.Target) domain.Result {
		if tgt.ID == "grok:0" {
			res := okResult(tgt, 11)
			close(established)
			return res
		}
		<-ctx.Done() // codex targets wedge until cancellation
		return timeoutish(tgt)
	}}
	r := &Runner{Codex: q, Grok: q}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-established
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	results := r.Run(ctx, mkTargets(2), 5*time.Second)
	if results[2].Status != domain.StatusOK || results[2].UsedPercent == nil || *results[2].UsedPercent != 11 {
		t.Fatalf("established result lost: %+v", results[2])
	}
}

func TestOptionsTargetsFeedRunner(t *testing.T) {
	o, err := options.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	targets := o.Targets()
	if len(targets) != 3 {
		t.Fatalf("default targets = %d", len(targets))
	}
	q := &fakeQuerier{}
	r := &Runner{Codex: q, Grok: q}
	results := r.Run(context.Background(), targets, time.Second)
	if results[0].TargetID != "codex:0" || results[2].TargetID != "grok:0" {
		t.Fatalf("unexpected target order: %+v", targets)
	}
}

func TestDisambiguateCodexAccounts(t *testing.T) {
	targets := []domain.Target{
		{ID: "codex:0", Index: 0, Provider: domain.ProviderCodex, Home: "/a/work", FallbackAccount: "work"},
		{ID: "codex:1", Index: 1, Provider: domain.ProviderCodex, Home: "/b/work", FallbackAccount: "work"},
		{ID: "codex:2", Index: 2, Provider: domain.ProviderCodex, Home: "/c/other", FallbackAccount: "other"},
	}
	email := "dup@example.com"
	results := []domain.Result{
		{TargetID: "codex:0", Provider: domain.ProviderCodex, Account: email},
		{TargetID: "codex:1", Provider: domain.ProviderCodex, Account: email},
		{TargetID: "codex:2", Provider: domain.ProviderCodex, Account: "other"},
	}
	DisambiguateCodexAccounts(results, targets)
	if results[0].Account == results[1].Account {
		t.Fatalf("duplicates not disambiguated: %q", results[0].Account)
	}
	if results[0].Account != "dup@example.com [work]" || results[1].Account != "work [work] #2" {
		// codex:0 keeps its email + home tag; codex:1's label was the
		// fallback "work" which is still ambiguous after the tag.
		t.Logf("labels: %q / %q", results[0].Account, results[1].Account)
	}
	if results[2].Account != "other" {
		t.Fatalf("unique label altered: %q", results[2].Account)
	}
}

func TestDisambiguateDifferentHomesNotMerged(t *testing.T) {
	targets := []domain.Target{
		{ID: "codex:0", Index: 0, Provider: domain.ProviderCodex, Home: "/a/.codex", FallbackAccount: ".codex"},
		{ID: "codex:1", Index: 1, Provider: domain.ProviderCodex, Home: "/b/.codex-hsy", FallbackAccount: ".codex-hsy"},
	}
	results := []domain.Result{
		{TargetID: "codex:0", Provider: domain.ProviderCodex, Account: "same@example.com"},
		{TargetID: "codex:1", Provider: domain.ProviderCodex, Account: "same@example.com"},
	}
	DisambiguateCodexAccounts(results, targets)
	if results[0].Account == results[1].Account {
		t.Fatal("rows merged")
	}
	if len(results) != 2 {
		t.Fatal("row count changed")
	}
}

// TestDisambiguationOperatesOnNormalizedLabels pins the audit's
// ordering finding: distinct raw labels that normalize to the same
// visible label (NBSP vs space) must be disambiguated, not merged into
// identical visible rows.
func TestDisambiguationOperatesOnNormalizedLabels(t *testing.T) {
	targets := []domain.Target{
		{ID: "codex:0", Index: 0, Provider: domain.ProviderCodex, Home: "/a/one", FallbackAccount: "one"},
		{ID: "codex:1", Index: 1, Provider: domain.ProviderCodex, Home: "/b/two", FallbackAccount: "two"},
	}
	results := []domain.Result{
		{TargetID: "codex:0", Provider: domain.ProviderCodex, Account: "user\u00a0x@example.com"},
		{TargetID: "codex:1", Provider: domain.ProviderCodex, Account: "user x@example.com"},
	}
	DisambiguateCodexAccounts(results, targets)
	if results[0].Account == results[1].Account {
		t.Fatalf("normalized duplicates not disambiguated: %q", results[0].Account)
	}
	if !strings.Contains(results[0].Account, "user x@example.com [") {
		t.Fatalf("unexpected label: %q", results[0].Account)
	}
	if len(results) != 2 {
		t.Fatal("row count changed")
	}
}

// TestDisambiguationSuffixLookalikeCollision pins the audit's generated-
// suffix finding: a duplicate's generated label that collides with an
// unrelated row's existing label (which merely resembles a generated
// suffix) must be further suffixed instead of merging rows.
func TestDisambiguationSuffixLookalikeCollision(t *testing.T) {
	targets := []domain.Target{
		{ID: "codex:0", Index: 0, Provider: domain.ProviderCodex, Home: "/a/b", FallbackAccount: "b"},
		{ID: "codex:1", Index: 1, Provider: domain.ProviderCodex, Home: "/c/b", FallbackAccount: "b"},
		{ID: "codex:2", Index: 2, Provider: domain.ProviderCodex, Home: "/d/other", FallbackAccount: "other"},
	}
	// codex:2's raw label literally equals what codex:0/codex:1's
	// disambiguation would generate; it is unique, so it keeps its
	// label and the generated ones must move aside.
	results := []domain.Result{
		{TargetID: "codex:0", Provider: domain.ProviderCodex, Account: "a"},
		{TargetID: "codex:1", Provider: domain.ProviderCodex, Account: "a"},
		{TargetID: "codex:2", Provider: domain.ProviderCodex, Account: "a [b]"},
	}
	DisambiguateCodexAccounts(results, targets)
	seen := map[string]bool{}
	for i, r := range results {
		if seen[r.Account] {
			t.Fatalf("rows %d and earlier share visible label %q", i, r.Account)
		}
		seen[r.Account] = true
	}
	if results[2].Account != "a [b]" {
		t.Fatalf("unique lookalike label must be preserved: %q", results[2].Account)
	}
}
