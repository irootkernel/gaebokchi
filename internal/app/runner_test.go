package app

import (
	"context"
	"fmt"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
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

var _ = ginkgo.It("TestDefaultThreeTargetsSimultaneousAdmission", func() {
	t := testT{ginkgo.GinkgoT()}
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
	gomega.Expect(q.maxSeen.Load()).To(gomega.Equal(int32(3)))
	for i, res := range results {
		if res.TargetID != targets[i].ID {
			t.Fatalf("order broken at %d: %s", i, res.TargetID)
		}
		if res.Status != domain.StatusOK {
			t.Fatalf("target %s status = %v", res.TargetID, res.Status)
		}
	}
})

var _ = ginkgo.It("TestDeterministicOrderUnderReversedCompletion", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

var _ = ginkgo.It("TestAtMostThreeActiveWithFiveHomes", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

var _ = ginkgo.It("TestFailureDoesNotCancelSiblings", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

var _ = ginkgo.It("TestQueuedTargetsCancelledWithoutLaunch", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

var _ = ginkgo.It("observes each stored result once, including queued cancellation", func() {
	t := testT{ginkgo.GinkgoT()}
	started := make(chan struct{}, 1)
	q := &fakeQuerier{responder: func(ctx context.Context, tgt domain.Target) domain.Result {
		started <- struct{}{}
		<-ctx.Done()
		return domain.Result{TargetID: tgt.ID, Provider: tgt.Provider, Account: tgt.FallbackAccount, Status: domain.StatusCancelled}
	}}
	var mu sync.Mutex
	observed := make(map[int][]domain.Result)
	r := &Runner{Codex: q, Grok: q, MaxActive: 1, Observe: func(index int, result domain.Result) {
		mu.Lock()
		observed[index] = append(observed[index], result)
		mu.Unlock()
	}}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan []domain.Result, 1)
	go func() { done <- r.Run(ctx, mkTargets(3), 5*time.Second) }()
	<-started
	cancel()
	results := <-done
	mu.Lock()
	defer mu.Unlock()
	if q.callCount() != 1 || len(observed) != len(results) {
		t.Fatalf("queries=%d observations=%d results=%d", q.callCount(), len(observed), len(results))
	}
	for i, result := range results {
		if len(observed[i]) != 1 || observed[i][0] != result || result.Status != domain.StatusCancelled {
			t.Fatalf("slot %d: callbacks=%v result=%v", i, observed[i], result)
		}
	}
})

var _ = ginkgo.It("observes normally completed results as they are stored", func() {
	t := testT{ginkgo.GinkgoT()}
	q := &fakeQuerier{}
	var mu sync.Mutex
	observed := make(map[int][]domain.Result)
	r := &Runner{Codex: q, Grok: q, MaxActive: 1, Observe: func(index int, result domain.Result) {
		mu.Lock()
		observed[index] = append(observed[index], result)
		mu.Unlock()
	}}
	results := r.Run(context.Background(), mkTargets(3), time.Second)
	mu.Lock()
	defer mu.Unlock()
	if q.callCount() != len(results) || len(observed) != len(results) {
		t.Fatalf("queries=%d observations=%d results=%d", q.callCount(), len(observed), len(results))
	}
	for i, result := range results {
		if len(observed[i]) != 1 || observed[i][0] != result || result.Status != domain.StatusOK || result.UsedPercent == nil || *result.UsedPercent != 50 {
			t.Fatalf("slot %d: callbacks=%v result=%v", i, observed[i], result)
		}
	}
})

var _ = ginkgo.It("TestDeadlineStartsAtAdmission", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

var _ = ginkgo.It("TestOneQueryAttemptPerTarget", func() {
	t := testT{ginkgo.GinkgoT()}
	q := &fakeQuerier{}
	r := &Runner{Codex: q, Grok: q}
	r.Run(context.Background(), mkTargets(4), time.Second)
	if q.callCount() != 5 {
		t.Fatalf("attempts = %d, want 5", q.callCount())
	}
})

// TestValidResultSurvivesCleanupPastDeadline reproduces the audit's
// executable-level finding at the runner boundary: a result established
// within the query budget must survive bounded cleanup that crosses the
// deadline before Query returns, instead of being rewritten to TIMEOUT.
var _ = ginkgo.It("TestValidResultSurvivesCleanupPastDeadline", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

// TestEstablishedResultSurvivesRootCancellation pins REQ-023's
// preservation clause: once a target's result is established, root
// cancellation of the invocation must not erase it from the output.
var _ = ginkgo.It("TestEstablishedResultSurvivesRootCancellation", func() {
	t := testT{ginkgo.GinkgoT()}
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
})

var _ = ginkgo.It("TestOptionsTargetsFeedRunner", func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".config", "gaebokchi", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("codex:\n  - name: a\n    path: ~/a\n  - name: b\n    path: ~/b\ngrok:\n  - name: g\n    path: ~/g\nclaude:\n  - name: c\n    path: ~/c\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	o, err := options.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	targets := o.Targets()
	if len(targets) != 4 {
		t.Fatalf("default targets = %d", len(targets))
	}
	q := &fakeQuerier{}
	r := &Runner{Codex: q, Grok: q, Claude: q}
	results := r.Run(context.Background(), targets, time.Second)
	if results[0].TargetID != "codex:0" || results[2].TargetID != "grok:0" || results[3].TargetID != "claude:0" {
		t.Fatalf("unexpected target order: %+v", targets)
	}
})
