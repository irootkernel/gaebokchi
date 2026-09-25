// Command gaebokchi renders a one-shot weekly-usage snapshot using the
// installed Codex, Grok, and Claude CLIs.
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
	"gaebokchi/internal/provider/claude"
	"gaebokchi/internal/provider/codex"
	"gaebokchi/internal/provider/grok"
	"gaebokchi/internal/render"
)

// buildVersion is overridden at release-build time via -ldflags.
var buildVersion string

const usage = `Usage:
  gaebokchi [flags]
  gaebokchi version [--json]

Show weekly subscription usage for Codex (ChatGPT), Grok, and Claude using
the existing provider CLI logins. One snapshot, then exit.

Commands:
  version               print version and exit; use --json for JSON

Flags:
  --codex-home PATH     Codex home directory; repeatable. Supplying any
                        replaces the defaults ~/.codex and ~/.codex-hsy.
  --codex-bin PATH      codex executable path or name (default "codex")
  --grok-bin PATH       grok executable path or name (default "grok")
  --claude-bin PATH     claude executable path or name (default "claude")
  --grok-account LABEL  display alias for the Grok row (default "default")
  --timeout DURATION    per-target query budget, 1s to 120s (default 20s)
  --json                emit one JSON usage snapshot instead of the table
  -h, --help            show this help and exit

Exit codes: 0 ok, 1 incomplete rows, 2 usage error, 130 SIGINT, 143 SIGTERM.

stdout carries the account table or JSON snapshot and can contain account
identifiers; diagnostics go to stderr.
`

func main() {
	os.Exit(run(os.Args[1:]))
}

// run returns the process exit code for one invocation.
func run(args []string) int {
	return runWithIO(args, os.Stdout, os.Stderr, func() (app.Querier, app.Querier, app.Querier) {
		return codex.New(), grok.New(), claude.New()
	})
}

// runWithIO keeps early command dispatch and provider construction testable.
func runWithIO(args []string, stdout, stderr io.Writer, newProviders providers) int {
	// Broken stdout/stderr pipes are reported through write errors instead of
	// terminating the process before the command can apply its exit contract.
	signal.Ignore(syscall.SIGPIPE)
	if handled, exitCode := handleVersion(args, stdout, stderr, currentVersion()); handled {
		return exitCode
	}
	opts, err := options.Parse(args)
	if err != nil {
		// No recovery path remains if stderr itself cannot accept the diagnostic.
		_, _ = fmt.Fprintf(stderr, "gaebokchi: %v\n\n%s", err, usage)
		return 2
	}
	return execute(opts, stdout, stderr, newProviders)
}

// providers is the seam through which tests inject fake providers.
type providers func() (codexQuerier, grokQuerier, claudeQuerier)

type codexQuerier = app.Querier
type grokQuerier = app.Querier
type claudeQuerier = app.Querier

// execute runs the snapshot and implements the exit contract of
// ARCH-10. Exit-code precedence is centralized in finalExit: the first
// handled cancellation signal outranks an output failure, which
// outranks partial provider failure. A closed stdout reader is an
// output failure (exit 1), never SIGPIPE signal death: SIGPIPE is
// notified below so the runtime delivers it harmlessly while the
// failing write itself returns an error the output-failure path acts
// on.
func execute(opts options.Options, stdout, stderr io.Writer, newProviders providers) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var signalNum atomic.Int64
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	// Explicit SIGPIPE notification preserves the existing snapshot-command
	// behavior while run's process-wide policy protects the early version path.
	pipeCh := make(chan os.Signal, 1)
	signal.Notify(pipeCh, syscall.SIGPIPE)
	defer signal.Stop(pipeCh)
	watchDone := make(chan struct{})
	defer close(watchDone)
	go func() {
		defer signal.Stop(sigCh)
		select {
		case sig, ok := <-sigCh:
			if ok {
				signalNum.Store(int64(sig.(syscall.Signal)))
				cancel()
			}
		case <-watchDone:
			// Execution finished; release the signal watcher.
		}
	}()

	writeOut := func(s string) bool {
		_, err := io.WriteString(stdout, s)
		return err == nil
	}

	// Help never launches a provider process; its output failure is an
	// output failure all the same.
	if opts.Help {
		if !writeOut(usage) {
			_, _ = fmt.Fprintln(stderr, "gaebokchi: cannot write help to stdout")
			return finalExit(signalNum.Load(), true, nil)
		}
		return finalExit(signalNum.Load(), false, nil)
	}
	targets := opts.Targets()
	cx, gk, cl := newProviders()
	runner := &app.Runner{Codex: cx, Grok: gk, Claude: cl}
	results := runner.Run(ctx, targets, opts.Timeout)
	app.DisambiguateCodexAccounts(results, targets)

	var output string
	if opts.JSON {
		var err error
		output, err = render.JSON(results)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "gaebokchi: cannot encode usage JSON")
			render.WriteDiagnostics(stderr, results)
			return finalExit(signalNum.Load(), true, results)
		}
	} else {
		output = render.Table(results)
	}
	outputOK := writeOut(output)
	if !outputOK {
		// Without a writable stdout a complete snapshot cannot be
		// guaranteed; write only safe diagnostics where possible.
		if opts.JSON {
			_, _ = fmt.Fprintln(stderr, "gaebokchi: cannot write usage JSON to stdout")
		} else {
			_, _ = fmt.Fprintln(stderr, "gaebokchi: cannot write the usage table to stdout")
		}
	}
	render.WriteDiagnostics(stderr, results)
	return finalExit(signalNum.Load(), !outputOK, results)
}

// finalExit picks the process exit code by the documented precedence:
// 130/143 for the first handled cancellation signal, then 1 for an
// output failure, then the row-based 0/1 contract. results is nil for
// the help path.
func finalExit(sigNum int64, outputFailed bool, results []domain.Result) int {
	switch {
	case sigNum == int64(syscall.SIGINT):
		return 130
	case sigNum != 0:
		return 143
	case outputFailed:
		return 1
	case results == nil:
		return 0
	default:
		return exitCode(results)
	}
}

// exitCode computes the non-signal exit code: 0 when every row is
// numeric or explicitly non-applicable and every target's cleanup
// completed, 1 otherwise. Incomplete cleanup must not be reported as
// complete operational success.
func exitCode(results []domain.Result) int {
	for _, r := range results {
		if r.Status.ExitClass() != domain.ExitClean {
			return 1
		}
		if r.CleanupFailed {
			return 1
		}
	}
	return 0
}
