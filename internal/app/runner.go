// Package app coordinates the bounded fan-out of provider queries and
// post-collection label disambiguation. It owns no daemon, cache, or
// retry loop: one finite snapshot per invocation.
package app

import (
	"context"
	"sync"
	"time"

	"gaebokchi/internal/domain"
)

// DefaultMaxActive bounds concurrent target queries per invocation.
const DefaultMaxActive = 3

// Querier is the one-target application boundary implemented by both
// providers.
type Querier interface {
	Query(ctx context.Context, tgt domain.Target) domain.Result
}

// Runner fans targets out to their providers with bounded concurrency
// and fixed result slots.
type Runner struct {
	Codex Querier
	Grok  Querier
	// MaxActive defaults to DefaultMaxActive.
	MaxActive int
}

func (r *Runner) maxActive() int {
	if r.MaxActive <= 0 {
		return DefaultMaxActive
	}
	return r.MaxActive
}

func (r *Runner) querierFor(p domain.ProviderKind) Querier {
	switch p {
	case domain.ProviderGrok:
		return r.Grok
	default:
		return r.Codex
	}
}

// Run queries every target once, at most maxActive at a time, and
// returns results in configured order. A target's deadline starts when
// it is admitted, not when it was queued. One target's failure never
// cancels its siblings; only root-context cancellation does. Targets
// still queued at cancellation are marked cancelled without launching.
func (r *Runner) Run(ctx context.Context, targets []domain.Target, timeout time.Duration) []domain.Result {
	results := make([]domain.Result, len(targets))
	started := make([]bool, len(targets))
	sem := make(chan struct{}, r.maxActive())
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := range targets {
		tgt := targets[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				if ctx.Err() != nil {
					// A slot freed after cancellation: still do not
					// launch a queued target.
					<-sem
					mu.Lock()
					results[tgt.Index] = cancelledResult(tgt)
					mu.Unlock()
					return
				}
			case <-ctx.Done():
				mu.Lock()
				results[tgt.Index] = cancelledResult(tgt)
				mu.Unlock()
				return
			}
			defer func() { <-sem }()

			mu.Lock()
			started[tgt.Index] = true
			mu.Unlock()

			qctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			res := r.querierFor(tgt.Provider).Query(qctx, tgt)
			if qctx.Err() == context.DeadlineExceeded && res.Status.ExitClass() == domain.ExitClean {
				// A clean result cannot arrive after its own deadline;
				// treat as timeout rather than trusting a late write.
				res = timedOutResult(tgt)
			}
			mu.Lock()
			results[tgt.Index] = res
			mu.Unlock()
		}()
	}
	wg.Wait()
	_ = started
	return results
}

func cancelledResult(tgt domain.Target) domain.Result {
	return domain.Result{
		TargetID:      tgt.ID,
		Provider:      tgt.Provider,
		Account:       tgt.FallbackAccount,
		AccountSource: domain.AccountSourceConfiguredAlias,
		Status:        domain.StatusCancelled,
		ObservedAt:    time.Now(),
		Diagnostic: domain.Diagnostic{
			Code:    "cancelled",
			Message: tgt.ID + ": query cancelled before start",
		},
	}
}

func timedOutResult(tgt domain.Target) domain.Result {
	return domain.Result{
		TargetID:      tgt.ID,
		Provider:      tgt.Provider,
		Account:       tgt.FallbackAccount,
		AccountSource: domain.AccountSourceConfiguredAlias,
		Status:        domain.StatusTimeout,
		ObservedAt:    time.Now(),
		Diagnostic: domain.Diagnostic{
			Code:    "timeout",
			Message: tgt.ID + ": query exceeded its deadline",
		},
	}
}

// DisambiguateCodexAccounts makes duplicate visible Codex labels
// distinguishable without merging rows: append " [<home-basename>]",
// then "#<index+1>" if that is still ambiguous. Rows are never hidden.
func DisambiguateCodexAccounts(results []domain.Result, targets []domain.Target) {
	homeBase := make(map[string]string, len(targets))
	for _, tgt := range targets {
		if tgt.Provider == domain.ProviderCodex {
			homeBase[tgt.ID] = baseName(tgt.Home)
		}
	}

	counts := make(map[string]int)
	for i := range results {
		if results[i].Provider == domain.ProviderCodex {
			counts[results[i].Account]++
		}
	}
	seen := make(map[string]int)
	for i := range results {
		if results[i].Provider != domain.ProviderCodex || counts[results[i].Account] < 2 {
			continue
		}
		label := results[i].Account + " [" + homeBase[results[i].TargetID] + "]"
		seen[label]++
		if seen[label] > 1 {
			label = label + " #" + itoa(seen[label])
		}
		results[i].Account = label
	}
}

func baseName(path string) string {
	if path == "" {
		return ""
	}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == len(path)-1 {
				continue
			}
			return path[i+1:]
		}
	}
	return path
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}
