package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	ginkgo "github.com/onsi/ginkgo/v2"
)

var _ = ginkgo.It("ignores runtime reports but still checks source documentation", func() {
	t := testT{ginkgo.GinkgoT()}
	root := t.TempDir()
	for _, name := range []string{
		"docs/README.md", "docs/specs/README.md", "docs/architecture/README.md",
		"docs/architecture-decision-records/README.md", "docs/implementation-tips/README.md",
		"docs/ops/README.md", "docs/roadmap/README.md", "docs/todo/README.md",
		"docs/deferred-feedback/README.md", "docs/verification/README.md",
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("# Source document\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{
		".mulgae/run/role-reports/documentation.md", ".sorage/INBOX.md",
		".gaori/runs/report.md", ".podway/runtime/report.md",
		"docs/vendor/.git/report.md", "docs/vendor/dist/report.md",
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("[broken](missing.md)\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	check := func() ([]byte, error) {
		return exec.Command("go", "run", "check-docs.go", root).CombinedOutput()
	}
	output, err := check()
	if err != nil || !strings.Contains(string(output), "10 markdown files, links and identifiers OK") {
		t.Fatalf("runtime reports entered documentation audit: %v\n%s", err, output)
	}

	path := filepath.Join(root, "docs/README.md")
	if err := os.WriteFile(path, []byte("[broken](missing.md)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	output, err = check()
	if err == nil || !strings.Contains(string(output), "docs/README.md: broken link missing.md") {
		t.Fatalf("source document was not checked: %v\n%s", err, output)
	}
})
