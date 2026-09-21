package testutil

import (
	"os"
	"testing"
)

// TestFakeHelperProcess exercises RunFakeScript only when this package's
// own test binary is re-executed as a fake. Other packages copy this
// exact test into their own test files so their test binary can serve as
// the fake child.
func TestFakeHelperProcess(t *testing.T) {
	if os.Getenv("GAEBOKCHI_FAKE") != "1" {
		return
	}
	if err := RunFakeScript(os.Stdin, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
