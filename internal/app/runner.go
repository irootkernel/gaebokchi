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
	Codex  Querier
	Grok   Querier
	Claude Querier
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
	case domain.ProviderClaude:
		return r.Claude
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
				results[tgt.Index] = cancelledResult(tgt)
				continue
			}
		case <-ctx.Done():
			results[tgt.Index] = cancelledResult(tgt)
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			qctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			// A clean result here means its final response was routed
			// before the deadline (the rpc layer rejects later ones), so
			// it survives even when the provider's bounded cleanup runs
			// past the deadline inside Query.
			res := r.querierFor(tgt.Provider).Query(qctx, tgt)
			results[tgt.Index] = res
		}()
	}
	wg.Wait()
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

// DisambiguateCodexAccounts makes duplicate visible Codex labels
// distinguishable without merging rows. Labels are normalized through
// the domain display-label policy first, so counting and suffixing
// operate on exactly what the table will show; duplicate rows then get
// " [<home-basename>]" and, when that is still occupied (by a unique
// row's label or an earlier assignment), the documented suffix
// "#<target index + 1>" based on the row's own configured target
// index. If the index-qualified label is itself already occupied, a
// deterministic second counter extends it while preserving the index
// component ("#3-2", "#3-3", ...). Rows are never hidden.
func DisambiguateCodexAccounts(results []domain.Result, targets []domain.Target) {
	homeBase := make(map[string]string, len(targets))
	targetIndex := make(map[string]int, len(targets))
	for _, tgt := range targets {
		if tgt.Provider == domain.ProviderCodex {
			homeBase[tgt.ID] = domain.NormalizeLabel(baseName(tgt.Home))
			targetIndex[tgt.ID] = tgt.Index
		}
	}

	for i := range results {
		if results[i].Provider == domain.ProviderCodex {
			results[i].Account = domain.NormalizeLabel(results[i].Account)
		}
	}

	counts := make(map[string]int)
	for i := range results {
		if results[i].Provider == domain.ProviderCodex {
			counts[results[i].Account]++
		}
	}

	// Unique rows occupy their labels; generated labels must not collide
	// with them (including labels that merely resemble generated ones).
	occupied := make(map[string]bool)
	for i := range results {
		if results[i].Provider == domain.ProviderCodex && counts[results[i].Account] < 2 {
			occupied[results[i].Account] = true
		}
	}

	for i := range results {
		if results[i].Provider != domain.ProviderCodex || counts[results[i].Account] < 2 {
			continue
		}
		base := results[i].Account + " [" + homeBase[results[i].TargetID] + "]"
		label := base
		if occupied[label] {
			// ARCH-04: the suffix identifies the target by its
			// configured index, not by the collision ordinal.
			indexed := base + " #" + itoa(targetIndex[results[i].TargetID]+1)
			label = indexed
			for extra := 2; occupied[label]; extra++ {
				label = indexed + "-" + itoa(extra)
			}
		}
		occupied[label] = true
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
