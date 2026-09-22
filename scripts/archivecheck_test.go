package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// member describes one synthetic archive entry for the checker tests.
type member struct {
	name string
	dir  bool
	mode int64
	body string
}

// writeTestRepo creates a disposable repository root whose four
// documents match the bodies used by synthetic archives.
func writeTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, doc := range []string{"README.md", "docs/requirement.md", "docs/architecture.md", "docs/plan.md"} {
		p := filepath.Join(root, doc)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("content of "+doc+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func docBody(repoRel string) string { return "content of " + repoRel + "\n" }

// intendedLayout returns the expected member set for the synthetic
// darwin-arm64 target archive.
func intendedLayout() []member {
	d := "gaebokchi-darwin-arm64"
	return []member{
		{name: d, dir: true, mode: 0o755},
		{name: d + "/gaebokchi", mode: 0o755, body: "fake binary"},
		{name: d + "/README.md", mode: 0o644, body: docBody("README.md")},
		{name: d + "/docs", dir: true, mode: 0o755},
		{name: d + "/docs/requirement.md", mode: 0o644, body: docBody("docs/requirement.md")},
		{name: d + "/docs/architecture.md", mode: 0o644, body: docBody("docs/architecture.md")},
		{name: d + "/docs/plan.md", mode: 0o644, body: docBody("docs/plan.md")},
	}
}

func buildTestArchive(t *testing.T, path string, members []member) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	for _, m := range members {
		hdr := &tar.Header{Name: m.name, Mode: m.mode}
		if m.dir {
			hdr.Typeflag = tar.TypeDir
			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatal(err)
			}
			continue
		}
		hdr.Typeflag = tar.TypeReg
		hdr.Size = int64(len(m.body))
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(m.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
}

func wantErrContaining(t *testing.T, err error, fragment string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), fragment) {
		t.Fatalf("err = %v, want it to contain %q", err, fragment)
	}
}

func TestVerifyArchiveAcceptsIntendedLayout(t *testing.T) {
	root := writeTestRepo(t)
	path := filepath.Join(t.TempDir(), "gaebokchi-darwin-arm64.tar.gz")
	buildTestArchive(t, path, intendedLayout())
	if err := verifyArchive(path, "darwin-arm64", root); err != nil {
		t.Fatalf("intended layout rejected: %v", err)
	}
}

// TestVerifyArchiveRejectsSidecarMembers is the negative fixture for
// the exact shape the confirmation review observed: AppleDouble ._*
// entries for the directory, the binary, and a packaged document.
func TestVerifyArchiveRejectsSidecarMembers(t *testing.T) {
	root := writeTestRepo(t)
	for _, extra := range []member{
		{name: "._gaebokchi-darwin-arm64", mode: 0o644, body: "x"},
		{name: "gaebokchi-darwin-arm64/._gaebokchi", mode: 0o644, body: "x"},
		{name: "gaebokchi-darwin-arm64/docs/._requirement.md", mode: 0o644, body: "x"},
		{name: "gaebokchi-darwin-arm64/.DS_Store", mode: 0o644, body: "x"},
	} {
		path := filepath.Join(t.TempDir(), "gaebokchi-darwin-arm64.tar.gz")
		buildTestArchive(t, path, append(intendedLayout(), extra))
		wantErrContaining(t, verifyArchive(path, "darwin-arm64", root), "unintended sidecar")
	}
}

func TestVerifyArchiveRejectsStructuralDefects(t *testing.T) {
	root := writeTestRepo(t)

	// Missing intended member.
	missing := intendedLayout()
	missing = append(missing[:2], missing[3:]...) // drop README.md
	path := filepath.Join(t.TempDir(), "gaebokchi-darwin-arm64.tar.gz")
	buildTestArchive(t, path, missing)
	wantErrContaining(t, verifyArchive(path, "darwin-arm64", root), "missing intended members")

	// Unexpected extra regular member.
	path = filepath.Join(t.TempDir(), "gaebokchi-darwin-arm64.tar.gz")
	buildTestArchive(t, path, append(intendedLayout(), member{name: "gaebokchi-darwin-arm64/extra.txt", mode: 0o644, body: "x"}))
	wantErrContaining(t, verifyArchive(path, "darwin-arm64", root), "unexpected regular member")

	// Non-executable binary.
	nonExec := intendedLayout()
	nonExec[1].mode = 0o644
	path = filepath.Join(t.TempDir(), "gaebokchi-darwin-arm64.tar.gz")
	buildTestArchive(t, path, nonExec)
	wantErrContaining(t, verifyArchive(path, "darwin-arm64", root), "not executable")

	// Stale packaged document.
	stale := intendedLayout()
	stale[4].body = "stale content\n"
	path = filepath.Join(t.TempDir(), "gaebokchi-darwin-arm64.tar.gz")
	buildTestArchive(t, path, stale)
	wantErrContaining(t, verifyArchive(path, "darwin-arm64", root), "differs from the finalized source")
}

func TestVerifyChecksums(t *testing.T) {
	dist := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dist, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifest := func() string {
		var b strings.Builder
		for _, name := range []string{"gaebokchi-darwin-arm64.tar.gz", "gaebokchi-darwin-amd64.tar.gz", "gaebokchi-linux-arm64.tar.gz", "gaebokchi-linux-amd64.tar.gz"} {
			body, _ := os.ReadFile(filepath.Join(dist, name))
			sum := sha256.Sum256(body)
			b.WriteString(hex.EncodeToString(sum[:]) + "  " + name + "\n")
		}
		return b.String()
	}
	for _, name := range []string{"gaebokchi-darwin-arm64.tar.gz", "gaebokchi-darwin-amd64.tar.gz", "gaebokchi-linux-arm64.tar.gz", "gaebokchi-linux-amd64.tar.gz"} {
		write(name, "archive-"+name)
	}
	os.WriteFile(filepath.Join(dist, "SHA256SUMS"), []byte(manifest()), 0o644)
	if err := verifyChecksums(dist); err != nil {
		t.Fatalf("correct manifest rejected: %v", err)
	}

	// Tampered manifest entry.
	os.WriteFile(filepath.Join(dist, "SHA256SUMS"), []byte(strings.Replace(manifest(), "  gaebokchi-linux-arm64", "  gaebokchi-linux-amd64", 1)), 0o644)
	wantErrContaining(t, verifyChecksums(dist), "checksum mismatch")

	// Manifest with a correct hash but only one target: the defect is
	// the missing archive, not a bad hash.
	body, _ := os.ReadFile(filepath.Join(dist, "gaebokchi-darwin-arm64.tar.gz"))
	sum := sha256.Sum256(body)
	os.WriteFile(filepath.Join(dist, "SHA256SUMS"), []byte(hex.EncodeToString(sum[:])+"  gaebokchi-darwin-arm64.tar.gz\n"), 0o644)
	wantErrContaining(t, verifyChecksums(dist), "lacks the")
}

// repoRoot locates the repository root by walking up from the test's
// working directory to go.mod, so the staged-archive gate works no
// matter which package directory the test binary runs from.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root (go.mod) not found")
		}
		dir = parent
	}
}

// TestStagedReleaseArchives is the developer-side packaging gate: when
// the four release archives are staged in dist/gaebokchi-0.1.0, their
// raw membership, permissions, document freshness, and checksum
// manifest are verified against this repository. It skips when no
// release is staged (for example in CI test-only jobs).
func TestStagedReleaseArchives(t *testing.T) {
	root := repoRoot(t)
	dist := filepath.Join(root, "dist", "gaebokchi-0.1.0")
	missing := false
	for _, tgt := range releaseTargets {
		if _, err := os.Stat(filepath.Join(dist, "gaebokchi-"+tgt+".tar.gz")); err != nil {
			missing = true
		}
	}
	if missing {
		t.Skip("release archives not staged; run scripts/build-release.sh first")
	}
	if err := checkStagedArchives(dist, root); err != nil {
		t.Fatalf("staged archives failed the packaging gate: %v", err)
	}
}
