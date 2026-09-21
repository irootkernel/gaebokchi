// Command gaebokchi renders a one-shot weekly-usage snapshot for Codex
// (ChatGPT) and Grok accounts using the installed provider CLIs.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"gaebokchi/internal/app"
	"gaebokchi/internal/domain"
	"gaebokchi/internal/options"
	"gaebokchi/internal/provider/codex"
	"gaebokchi/internal/provider/grok"
	"gaebokchi/internal/render"
)

// version is overridden at release-build time via -ldflags.
var version = "0.1.0"

const usage = `Usage: gaebokchi [flags]

Show weekly subscription usage for Codex (ChatGPT) and Grok accounts using
the existing provider CLI logins. One snapshot, then exit.

Flags:
  --codex-home PATH     Codex home directory; repeatable. Supplying any
                        replaces the defaults ~/.codex and ~/.codex-hsy.
  --codex-bin PATH      codex executable path or name (default "codex")
  --grok-bin PATH       grok executable path or name (default "grok")
  --grok-account LABEL  display alias for the Grok row (default "default")
  --timeout DURATION    per-target query budget, 1s to 120s (default 20s)
  -h, --help            show this help and exit
  --version             print version and exit

Exit codes: 0 ok, 1 incomplete rows, 2 usage error, 130 SIGINT, 143 SIGTERM.

stdout carries the account table and can contain account identifiers;
diagnostics go to stderr.
`

func main() {
	os.Exit(run(os.Args[1:]))
}

// run returns the process exit code for one invocation.
func run(args []string) int {
	opts, err := options.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gaebokchi: %v\n\n%s", err, usage)
		return 2
	}
	return execute(opts, os.Stdout, os.Stderr, func() (app.Querier, app.Querier) {
		return codex.New(), grok.New()
	})
}

// providers is the seam through which tests inject fake providers.
type providers func() (codexQuerier, grokQuerier)

type codexQuerier = app.Querier
type grokQuerier = app.Querier

// execute runs the snapshot and implements the exit contract of
// ARCH-10. The first handled signal determines the cancellation exit
// code and takes precedence over partial provider failure.
func execute(opts options.Options, stdout, stderr io.Writer, newProviders providers) int {
	// Help and version never launch a provider process.
	if opts.Help {
		_, _ = io.WriteString(stdout, usage)
		return 0
	}
	if opts.Version {
		fmt.Fprintf(stdout, "gaebokchi %s\n", version)
		return 0
	}

	// Signal setup precedes any child so an early interrupt cannot
	// bypass cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var signalNum atomic.Int64
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		if sig, ok := <-sigCh; ok {
			signalNum.Store(int64(sig.(syscall.Signal)))
			cancel()
		}
	}()

	targets := opts.Targets()
	cx, gk := newProviders()
	runner := &app.Runner{Codex: cx, Grok: gk}
	results := runner.Run(ctx, targets, opts.Timeout)
	app.DisambiguateCodexAccounts(results, targets)

	table := render.Table(results)
	if _, err := io.WriteString(stdout, table); err != nil {
		// Without a writable stdout a complete table cannot be
		// guaranteed; write only a safe diagnostic where possible.
		fmt.Fprintln(stderr, "gaebokchi: cannot write the usage table to stdout")
		return 1
	}
	render.WriteDiagnostics(stderr, results)

	if n := signalNum.Load(); n != 0 {
		if n == int64(syscall.SIGINT) {
			return 130
		}
		return 143
	}
	return exitCode(results)
}

// exitCode computes the non-signal exit code: 0 when every row is
// numeric or explicitly non-applicable, 1 otherwise.
func exitCode(results []domain.Result) int {
	for _, r := range results {
		if r.Status.ExitClass() != domain.ExitClean {
			return 1
		}
	}
	return 0
}
