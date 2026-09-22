package integration

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCleanupRemovesOnlyOwnedDirs pins the audit's test-hygiene finding:
// cleanup must remove only directories created by this invocation. A
// similarly named directory belonging to someone else (a concurrent
// invocation, a crashed run, or the user) must survive — a shared name
// prefix is not ownership evidence.
func TestCleanupRemovesOnlyOwnedDirs(t *testing.T) {
	// Operate on a private registry so the suite's shared build
	// directory (registered by gaebokchiBin) is never removed here,
	// including under -count=N re-runs of the whole package.
	saved := ownedDirs
	ownedDirs = nil
	defer func() { ownedDirs = saved }()

	// A "foreign" directory that merely shares the naming convention.
	foreign, err := os.MkdirTemp("", "gaebokchi-e2e-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(foreign) })
	if _, err := os.Stat(filepath.Join(foreign, "marker")); err != nil {
		if err := os.WriteFile(filepath.Join(foreign, "marker"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	owned, err := os.MkdirTemp("", "gaebokchi-e2e-")
	if err != nil {
		t.Fatal(err)
	}
	registerOwnedDir(owned)

	cleanupOwnedDirs()

	if _, err := os.Stat(owned); !os.IsNotExist(err) {
		t.Fatalf("owned directory %s not removed: %v", owned, err)
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Fatalf("foreign directory %s was removed by cleanup: %v", foreign, err)
	}
	// Repeated cleanup after the registry was drained stays a no-op.
	cleanupOwnedDirs()
	if _, err := os.Stat(foreign); err != nil {
		t.Fatalf("foreign directory %s removed on second cleanup: %v", foreign, err)
	}
}
