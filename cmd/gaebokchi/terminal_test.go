package main

import (
	"bytes"
	"io"
	"os"
	"strings"

	ginkgo "github.com/onsi/ginkgo/v2"
)

var _ = ginkgo.It("selects the live view only for a measurable interactive stdout", func() {
	t := testT{ginkgo.GinkgoT()}
	for _, tc := range []struct {
		name string
		json bool
		term string
		file bool
		cols int
		rows int
		ok   bool
		want bool
	}{
		{"terminal", false, "xterm-256color", true, 100, 30, true, true},
		{"json", true, "xterm-256color", true, 100, 30, true, false},
		{"pipe", false, "xterm-256color", false, 100, 30, true, false},
		{"dumb", false, "dumb", true, 100, 30, true, false},
		{"empty term", false, "", true, 100, 30, true, false},
		{"ioctl failure", false, "xterm", true, 100, 30, false, false},
		{"zero columns", false, "xterm", true, 0, 30, true, false},
		{"zero rows", false, "xterm", true, 100, 0, true, false},
	} {
		if got := terminalSelection(tc.json, tc.term, tc.file, tc.cols, tc.rows, tc.ok); got != tc.want {
			t.Fatalf("%s: got %t, want %t", tc.name, got, tc.want)
		}
	}
})

var _ = ginkgo.It("wires terminal selection, NO_COLOR, and live write failures through execute", func() {
	t := testT{ginkgo.GinkgoT()}
	size := func(*os.File) (int, int, error) { return 100, 30, nil }
	for _, tc := range []struct {
		name    string
		term    string
		noColor string
		live    bool
		json    bool
	}{
		{"styled", "xterm-256color", "", true, false},
		{"colorless", "xterm-256color", "1", true, false},
		{"dumb", "dumb", "", false, false},
		{"json terminal", "xterm-256color", "", false, true},
	} {
		out, err := os.CreateTemp(t.TempDir(), "terminal-output-")
		if err != nil {
			t.Fatal(err)
		}
		var stderr bytes.Buffer
		getenv := func(key string) string {
			if key == "TERM" {
				return tc.term
			}
			if key == "NO_COLOR" {
				return tc.noColor
			}
			return ""
		}
		args := []string{}
		if tc.json {
			args = append(args, "--json")
		}
		code := executeWithTerminal(parse(t, args...), out, &stderr, fakes, size, getenv)
		if _, err := out.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		output, err := io.ReadAll(out)
		_ = out.Close()
		if err != nil || code != 0 || stderr.Len() != 0 {
			t.Fatalf("%s: code=%d read=%v stderr=%q", tc.name, code, err, stderr.String())
		}
		if tc.live {
			if !strings.Contains(string(output), "querying…") || !strings.Contains(string(output), "\x1b[?25h") || strings.Contains(string(output), "AI PROVIDER") {
				t.Fatalf("%s: live view not selected: %q", tc.name, output)
			}
		} else if tc.json {
			if !bytes.HasPrefix(output, []byte("{")) || !bytes.HasSuffix(output, []byte("}\n")) ||
				bytes.Contains(output, []byte("\x1b")) || bytes.Contains(output, []byte("querying…")) {
				t.Fatalf("%s: JSON not selected: %q", tc.name, output)
			}
		} else if !strings.HasPrefix(string(output), "AI PROVIDER") || strings.Contains(string(output), "\x1b[") {
			t.Fatalf("%s: table not selected: %q", tc.name, output)
		}
		if tc.name == "colorless" && (!strings.Contains(string(output), "\x1b[1m") ||
			strings.Contains(string(output), "\x1b[1;") || strings.Contains(string(output), "\x1b[32m") ||
			strings.Contains(string(output), "\x1b[33m") || strings.Contains(string(output), "\x1b[31m")) {
			t.Fatalf("NO_COLOR lost bold or emitted a malformed SGR: %q", output)
		}
	}

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = reader.Close()
	defer func() { _ = writer.Close() }()
	var stderr bytes.Buffer
	code := executeWithTerminal(parse(t), writer, &stderr, fakes, size, func(key string) string {
		if key == "TERM" {
			return "xterm-256color"
		}
		return ""
	})
	if code != 1 || !strings.Contains(stderr.String(), "cannot write the usage view to stdout") {
		t.Fatalf("live write failure: code=%d stderr=%q", code, stderr.String())
	}
})
