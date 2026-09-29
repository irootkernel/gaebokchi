package main

import (
	"gaebokchi/internal/testutil"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"os"
	"path/filepath"
	"testing"
)

var _ = ginkgo.BeforeEach(func() {
	t := testT{ginkgo.GinkgoT()}
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, name := range []string{"codex-a", "codex-b", "grok", "claude"} {
		if err := os.Mkdir(filepath.Join(home, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	setTestConfig(t, "codex:\n  - name: first\n    path: ~/codex-a\n  - name: second\n    path: ~/codex-b\ngrok:\n  - name: default\n    path: ~/grok\nclaude:\n  - name: default\n    path: ~/claude\n")
})

func setTestConfig(t testT, body string) {
	t.Helper()
	path := filepath.Join(os.Getenv("HOME"), ".config", "gaebokchi", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestMainSuite(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Main Suite")
}

type testT struct{ ginkgo.GinkgoTInterface }

func TestMain(m *testing.M) {
	if os.Getenv("GAEBOKCHI_SIG_TEST") == "1" {
		runSignalTestChild()
	}
	if os.Getenv("GAEBOKCHI_FAKE") == "1" {
		if err := testutil.RunFakeScript(os.Stdin, os.Stdout, os.Stderr); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}
