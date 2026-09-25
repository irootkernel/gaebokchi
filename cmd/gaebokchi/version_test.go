package main

import (
	"bytes"
	ginkgo "github.com/onsi/ginkgo/v2"
	"runtime/debug"
)

var _ = ginkgo.It("TestVersionOutput", func() {
	t := testT{ginkgo.GinkgoT()}
	var stdout, stderr bytes.Buffer
	handled, code := handleVersion([]string{"version"}, &stdout, &stderr, "v0.1.0")
	if !handled || code != 0 || stdout.String() != "gaebokchi v0.1.0\n" || stderr.Len() != 0 {
		t.Fatalf("human version = handled:%t code:%d stdout:%q stderr:%q", handled, code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	handled, code = handleVersion([]string{"version", "--json"}, &stdout, &stderr, "v0.1.0")
	if !handled || code != 0 || stdout.String() != "{\"name\":\"gaebokchi\",\"version\":\"v0.1.0\"}\n" || stderr.Len() != 0 {
		t.Fatalf("JSON version = handled:%t code:%d stdout:%q stderr:%q", handled, code, stdout.String(), stderr.String())
	}
})

var _ = ginkgo.It("TestVersionDevelopmentAndBuildResolution", func() {
	t := testT{ginkgo.GinkgoT()}
	if got := versionFrom(nil, ""); got != "(devel)" {
		t.Fatalf("nil build info = %q", got)
	}
	info := &debug.BuildInfo{Main: debug.Module{Version: "v0.0.9"}}
	if got := versionFrom(info, ""); got != "v0.0.9" {
		t.Fatalf("module version = %q", got)
	}
	if got := versionFrom(info, "v0.1.0"); got != "v0.1.0" {
		t.Fatalf("release override = %q", got)
	}
	if got := versionFrom(&debug.BuildInfo{}, ""); got != "(devel)" {
		t.Fatalf("empty build info = %q", got)
	}
})

var _ = ginkgo.It("TestVersionUsageAndOutputFailure", func() {
	t := testT{ginkgo.GinkgoT()}
	for _, args := range [][]string{
		{"version", "--output", "json"},
		{"version", "--json", "extra"},
		{"version", "--help"},
	} {
		var stdout, stderr bytes.Buffer
		handled, code := handleVersion(args, &stdout, &stderr, "v0.1.0")
		if !handled || code != 2 || stdout.Len() != 0 || stderr.String() != "gaebokchi: usage: gaebokchi version [--json]\n" {
			t.Fatalf("usage %q = handled:%t code:%d stdout:%q stderr:%q", args, handled, code, stdout.String(), stderr.String())
		}
	}

	var stderr bytes.Buffer
	handled, code := handleVersion([]string{"version", "--json"}, errWriter{}, &stderr, "v0.1.0")
	if !handled || code != 1 || stderr.String() != "gaebokchi: cannot write version to stdout\n" {
		t.Fatalf("output failure = handled:%t code:%d stderr:%q", handled, code, stderr.String())
	}
})

var _ = ginkgo.It("TestNonVersionArgumentsAreNotHandled", func() {
	t := testT{ginkgo.GinkgoT()}
	for _, args := range [][]string{nil, {"--version"}, {"--help"}} {
		handled, code := handleVersion(args, &bytes.Buffer{}, &bytes.Buffer{}, "v0.1.0")
		if handled || code != 0 {
			t.Fatalf("arguments %q unexpectedly handled: %t/%d", args, handled, code)
		}
	}
})
