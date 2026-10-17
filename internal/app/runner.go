// Package app coordinates the bounded fan-out of provider queries.
// It owns no daemon, cache, or
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

// Querier is the one-target application boundary implemented by each provider.
type Querier interface {
	Query(ctx context.Context, tgt domain.Target) domain.Result
}

// Runner fans targets out to their providers with bounded concurrency
// and fixed result slots.
type Runner struct {
	Codex  Querier
	Grok   Querier
	Claude Querier
	Agy    Querier
	// Observe receives a stored terminal result once for each target. Calls
	// can arrive concurrently; the observer must not mutate the result.
	Observe func(index int, result domain.Result)
	// MaxActive defaults to DefaultMaxActive.
	MaxActive int
}

func (r *Runner) store(results []domain.Result, tgt domain.Target, result domain.Result) {
	results[tgt.Index] = result
	if r.Observe != nil {
		r.Observe(tgt.Index, result)
	}
}

func (r *Runner) maxActive() int {
	if r.MaxActive <= 0 {
		return DefaultMaxActive
	}
	return r.MaxActive
}

func (r *Runner) querierFor(p domain.ProviderKind) Querier {
	switch p {
	case domain.ProviderCodex:
		return r.Codex
	case domain.ProviderGrok:
		return r.Grok
	case domain.ProviderClaude:
		return r.Claude
	case domain.ProviderAgy:
		return r.Agy
	default:
		return nil
	}
}

// Run queries every target once, at most maxActive at a time, and
// returns results in configured order. A target's deadline starts when
// it is admitted, not when it was queued. One target's failure never
// cancels its siblings; only root-context cancellation does. Targets
// still queued at cancellation are marked cancelled without launching.
func (r *Runner) Run(ctx context.Context, targets []domain.Target, timeout time.Duration) []domain.Result {
	results := make([]domain.Result, len(targets))
	sem := make(chan struct{}, r.maxActive())
	var wg sync.WaitGroup

	for i := range targets {
		tgt := targets[i]
		select {
		case sem <- struct{}{}:
			if ctx.Err() != nil {
				// A slot freed after cancellation must not launch
				// a queued target.
				<-sem
				r.store(results, tgt, cancelledResult(tgt))
				continue
			}
		case <-ctx.Done():
			r.store(results, tgt, cancelledResult(tgt))
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			q := r.querierFor(tgt.Provider)
			if q == nil {
				r.store(results, tgt, unsupportedProviderResult(tgt))
				return
			}
			qctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			// A clean result here means its final response was routed
			// before the deadline (the rpc layer rejects later ones), so
			// it survives even when the provider's bounded cleanup runs
			// past the deadline inside Query.
			res := q.Query(qctx, tgt)
			r.store(results, tgt, res)
		}()
	}
	wg.Wait()
	return results
}

func unsupportedProviderResult(tgt domain.Target) domain.Result {
	return domain.Result{
		TargetID:      tgt.ID,
		Provider:      tgt.Provider,
		Account:       tgt.FallbackAccount,
		AccountSource: domain.AccountSourceConfiguredAlias,
		Status:        domain.StatusUnsupported,
		ObservedAt:    time.Now(),
		Diagnostic: domain.Diagnostic{
			Code:    "provider_unsupported",
			Message: tgt.ID + ": unsupported provider",
		},
	}
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
