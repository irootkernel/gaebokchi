// Command gaebokchi renders a one-shot weekly-usage snapshot for Codex
// (ChatGPT) and Grok accounts using the installed provider CLIs.
package main

import (
	"fmt"
	"os"

	"gaebokchi/internal/options"
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
	if opts.Help {
		fmt.Print(usage)
		return 0
	}
	if opts.Version {
		fmt.Printf("gaebokchi %s\n", version)
		return 0
	}

	// TASK-002 scaffold: the runtime snapshot path is intentionally not
	// implemented yet. Fail explicitly rather than inventing usage data.
	fmt.Fprintln(os.Stderr, "gaebokchi: snapshot runtime is not implemented yet")
	return 1
}
