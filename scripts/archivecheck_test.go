package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
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

// intendedLayoutFor returns the expected member set for a synthetic
// target archive.
func intendedLayoutFor(target string) []member {
	d := "gaebokchi-" + target
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
	buildTestArchive(t, path, intendedLayoutFor("darwin-arm64"))
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
		buildTestArchive(t, path, append(intendedLayoutFor("darwin-arm64"), extra))
		wantErrContaining(t, verifyArchive(path, "darwin-arm64", root), "unintended sidecar")
	}
}

func TestVerifyArchiveRejectsStructuralDefects(t *testing.T) {
	root := writeTestRepo(t)

	// Missing intended member.
	missing := intendedLayoutFor("darwin-arm64")
	missing = append(missing[:2], missing[3:]...) // drop README.md
	path := filepath.Join(t.TempDir(), "gaebokchi-darwin-arm64.tar.gz")
	buildTestArchive(t, path, missing)
	wantErrContaining(t, verifyArchive(path, "darwin-arm64", root), "missing intended members")

	// Unexpected extra regular member.
	path = filepath.Join(t.TempDir(), "gaebokchi-darwin-arm64.tar.gz")
	buildTestArchive(t, path, append(intendedLayoutFor("darwin-arm64"), member{name: "gaebokchi-darwin-arm64/extra.txt", mode: 0o644, body: "x"}))
	wantErrContaining(t, verifyArchive(path, "darwin-arm64", root), "unexpected regular member")

	// Non-executable binary.
	nonExec := intendedLayoutFor("darwin-arm64")
	nonExec[1].mode = 0o644
	path = filepath.Join(t.TempDir(), "gaebokchi-darwin-arm64.tar.gz")
	buildTestArchive(t, path, nonExec)
	wantErrContaining(t, verifyArchive(path, "darwin-arm64", root), "not executable")

	// Stale packaged document.
	stale := intendedLayoutFor("darwin-arm64")
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

// TestStagedReleaseArchives is the developer-side packaging gate
// behind the same pass/fail/skip decision used everywhere: when the
// staging directory exists, its archives' raw membership, permissions,
// document freshness, and checksum manifest are verified — incomplete
// staging fails. A skip is allowed only when the staging directory
// genuinely does not exist and GAEBOKCHI_REQUIRE_STAGED_RELEASE is not
// set (ordinary checkout tests); with the variable set to 1 (CI
// release job) even a missing staging directory fails.
func TestStagedReleaseArchives(t *testing.T) {
	root := repoRoot(t)
	dist := os.Getenv("GAEBOKCHI_STAGED_RELEASE_DIR") // test-only staging override
	if dist == "" {
		dist = filepath.Join(root, "dist", "gaebokchi-0.1.0")
	}
	require := os.Getenv("GAEBOKCHI_REQUIRE_STAGED_RELEASE") == "1"
	err := checkStagedRelease(root, dist, require)
	if errors.Is(err, errReleaseNotStaged) {
		t.Skip("release staging directory not present; run scripts/build-release.sh first")
	}
	if err != nil {
		t.Fatalf("staged release validation failed: %v", err)
	}
}

// TestVerifyArchiveRejectsUnexpectedAndDuplicateMembers pins the
// strict member-set contract: directories are allowlisted, every
// intended entry appears exactly once, and an expected path presented
// with the wrong entry type is rejected.
func TestVerifyArchiveRejectsUnexpectedAndDuplicateMembers(t *testing.T) {
	root := writeTestRepo(t)
	d := "gaebokchi-darwin-arm64"
	cases := []struct {
		name  string
		extra []member
		want  string
	}{
		{"unexpected directory", []member{{name: d + "/unexpected", dir: true, mode: 0o755}}, "unexpected directory member"},
		{"nested unexpected directory", []member{{name: d + "/docs/extra", dir: true, mode: 0o755}}, "unexpected directory member"},
		{"duplicate README with identical content", []member{{name: d + "/README.md", mode: 0o644, body: docBody("README.md")}}, "duplicate member"},
		{"duplicate binary entry", []member{{name: d + "/gaebokchi", mode: 0o755, body: "fake binary"}}, "duplicate member"},
		{"duplicate expected directory", []member{{name: d + "/docs", dir: true, mode: 0o755}}, "duplicate directory member"},
		{"duplicate target directory", []member{{name: d, dir: true, mode: 0o755}}, "duplicate directory member"},
		{"expected file path presented as directory", []member{{name: d + "/README.md", dir: true, mode: 0o755}}, "unexpected directory member"},
		{"expected directory path presented as file", []member{{name: d + "/docs", mode: 0o644, body: "x"}}, "unexpected regular member"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "gaebokchi-darwin-arm64.tar.gz")
			buildTestArchive(t, path, append(intendedLayoutFor("darwin-arm64"), tc.extra...))
			wantErrContaining(t, verifyArchive(path, "darwin-arm64", root), tc.want)
		})
	}
}

// writeCompleteStaging builds a complete, valid four-target staging
// directory against the given repository root and returns its path.
func writeCompleteStaging(t *testing.T, root string) string {
	t.Helper()
	dist := t.TempDir()
	var manifest strings.Builder
	for _, tgt := range releaseTargets {
		name := "gaebokchi-" + tgt + ".tar.gz"
		path := filepath.Join(dist, name)
		buildTestArchive(t, path, intendedLayoutFor(tgt))
		sum, err := fileSHA256(path)
		if err != nil {
			t.Fatal(err)
		}
		manifest.WriteString(sum + "  " + name + "\n")
	}
	if err := os.WriteFile(filepath.Join(dist, "SHA256SUMS"), []byte(manifest.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return dist
}

// TestCheckStagedReleaseDecision exercises the exact pass/fail/skip
// decision the test entry point uses, over isolated test-owned
// staging roots (never the canonical dist/).
func TestCheckStagedReleaseDecision(t *testing.T) {
	root := writeTestRepo(t)

	t.Run("absent staging skips in optional mode", func(t *testing.T) {
		err := checkStagedRelease(root, filepath.Join(t.TempDir(), "absent"), false)
		if !errors.Is(err, errReleaseNotStaged) {
			t.Fatalf("err = %v, want the not-staged skip", err)
		}
	})
	t.Run("absent staging fails in mandatory mode", func(t *testing.T) {
		err := checkStagedRelease(root, filepath.Join(t.TempDir(), "absent"), true)
		if err == nil || errors.Is(err, errReleaseNotStaged) {
			t.Fatalf("err = %v, want a hard failure", err)
		}
	})
	t.Run("existing but empty staging fails", func(t *testing.T) {
		if err := checkStagedRelease(root, t.TempDir(), false); err == nil || errors.Is(err, errReleaseNotStaged) {
			t.Fatalf("err = %v, want a hard failure for empty staging", err)
		}
	})
	t.Run("partial staging fails", func(t *testing.T) {
		dist := t.TempDir()
		buildTestArchive(t, filepath.Join(dist, "gaebokchi-darwin-arm64.tar.gz"), intendedLayoutFor("darwin-arm64"))
		if err := checkStagedRelease(root, dist, false); err == nil {
			t.Fatal("partial staging must fail, not pass or skip")
		}
	})
	t.Run("missing manifest fails", func(t *testing.T) {
		dist := t.TempDir()
		for _, tgt := range releaseTargets {
			buildTestArchive(t, filepath.Join(dist, "gaebokchi-"+tgt+".tar.gz"), intendedLayoutFor(tgt))
		}
		if err := checkStagedRelease(root, dist, false); err == nil {
			t.Fatal("missing manifest must fail")
		}
	})
	t.Run("non-directory staging path fails", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "not-a-dir")
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := checkStagedRelease(root, path, false); err == nil {
			t.Fatal("non-directory staging path must fail")
		}
	})
	t.Run("inspection failure distinct from absence fails", func(t *testing.T) {
		// Deterministic seam: an expected archive path occupied by a
		// directory makes inspection fail on every platform without
		// relying on permission bits.
		dist := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dist, "gaebokchi-darwin-arm64.tar.gz"), 0o755); err != nil {
			t.Fatal(err)
		}
		err := checkStagedRelease(root, dist, false)
		if err == nil || errors.Is(err, errReleaseNotStaged) {
			t.Fatalf("err = %v, want an inspection failure, not a skip", err)
		}
	})
	t.Run("complete valid staging passes in both modes", func(t *testing.T) {
		dist := writeCompleteStaging(t, root)
		if err := checkStagedRelease(root, dist, false); err != nil {
			t.Fatalf("optional mode rejected valid staging: %v", err)
		}
		if err := checkStagedRelease(root, dist, true); err != nil {
			t.Fatalf("mandatory mode rejected valid staging: %v", err)
		}
	})
}

// TestMandatoryModeCommandExit runs the actual CI-facing command as a
// subprocess and asserts its process exit: mandatory mode must exit
// nonzero for absent and incomplete staging, while optional mode exits
// zero (skip) for absent staging.
func TestMandatoryModeCommandExit(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain unavailable for the subprocess check")
	}
	root := repoRoot(t)
	run := func(env ...string) int {
		t.Helper()
		cmd := exec.Command("go", "test", "./scripts/", "-run", "TestStagedReleaseArchives", "-count=1")
		cmd.Dir = root
		cmd.Env = append(os.Environ(), env...)
		out, err := cmd.CombinedOutput()
		if err == nil {
			return 0
		}
		if ee, ok := err.(*exec.ExitError); ok {
			_ = out
			return ee.ExitCode()
		}
		t.Fatalf("go test could not run: %v", err)
		return -1
	}

	// REQUIRE is set explicitly in every invocation so an inherited
	// value from the surrounding environment cannot flip the mode.
	absent := filepath.Join(t.TempDir(), "absent")
	if code := run("GAEBOKCHI_REQUIRE_STAGED_RELEASE=", "GAEBOKCHI_STAGED_RELEASE_DIR="+absent); code != 0 {
		t.Fatalf("optional mode with absent staging exited %d, want 0 (skip)", code)
	}
	if code := run("GAEBOKCHI_REQUIRE_STAGED_RELEASE=1", "GAEBOKCHI_STAGED_RELEASE_DIR="+absent); code == 0 {
		t.Fatal("mandatory mode with absent staging exited 0, want nonzero")
	}
	if code := run("GAEBOKCHI_REQUIRE_STAGED_RELEASE=1", "GAEBOKCHI_STAGED_RELEASE_DIR="+t.TempDir()); code == 0 {
		t.Fatal("mandatory mode with empty staging exited 0, want nonzero")
	}
}
