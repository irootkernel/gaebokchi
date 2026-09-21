// Package integration qualifies the actual gaebokchi executable with
// fake provider CLIs: no real accounts, credentials, provider
// installations, or network access.
package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gaebokchi/internal/testutil"
)

// TestMain is the fake-provider entry point: the gaebokchi binary under
// test launches this same test binary as its codex/grok child, and the
// argv decides which provider script runs.
func TestMain(m *testing.M) {
	if os.Getenv("GAEBOKCHI_FAKE") == "1" {
		steps := os.Getenv("GAEBOKCHI_FAKE_STEPS")
		if steps == "" {
			// The provider kind is selected by argv: codex children are
			// invoked as "app-server", grok children as "agent stdio".
			isCodex, isGrok := false, false
			for _, a := range os.Args[1:] {
				if a == "app-server" {
					isCodex = true
				}
				if a == "agent" || a == "stdio" {
					isGrok = true
				}
			}
			if isCodex && !isGrok {
				steps = os.Getenv("GAEBOKCHI_FAKE_CODEX_STEPS")
			} else {
				steps = os.Getenv("GAEBOKCHI_FAKE_GROK_STEPS")
			}
		}
		os.Setenv("GAEBOKCHI_FAKE_STEPS", steps)
		// Record the invocation for arg and env assertions.
		if path := os.Getenv("GAEBOKCHI_FAKE_ARGLOG"); path != "" {
			argv := "argv:"
			for _, a := range os.Args[1:] {
				argv += " " + a
			}
			f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			if err == nil {
				_, _ = f.WriteString(argv + "\n")
				_ = f.Close()
			}
		}
		if p := os.Getenv("GAEBOKCHI_FAKE_ENVLOG"); p != "" {
			var env strings.Builder
			env.WriteString("=== child")
			for _, a := range os.Args[1:] {
				env.WriteString(" " + a)
			}
			env.WriteString("\n")
			for _, kv := range os.Environ() {
				env.WriteString(kv + "\n")
			}
			f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			if err == nil {
				_, _ = f.WriteString(env.String())
				_ = f.Close()
			}
		}
		if err := testutil.RunFakeScript(os.Stdin, os.Stdout, os.Stderr); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	code := m.Run()
	// The shared e2e build directory is test-owned state; remove it.
	for _, prefix := range []string{"gaebokchi-e2e-"} {
		if matches, err := filepath.Glob(filepath.Join(os.TempDir(), prefix+"*")); err == nil {
			for _, m := range matches {
				_ = os.RemoveAll(m)
			}
		}
	}
	os.Exit(code)
}
