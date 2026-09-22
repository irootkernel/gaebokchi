package main

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime/debug"
)

const productName = "gaebokchi"

type versionOutput struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// currentVersion resolves release metadata without consulting the project,
// provider installations, or user configuration. An explicit release-build
// override wins over Go module metadata; ordinary source builds are devel.
func currentVersion() string {
	info, _ := debug.ReadBuildInfo()
	return versionFrom(info, buildVersion)
}

func versionFrom(info *debug.BuildInfo, override string) string {
	version := "(devel)"
	if info != nil && info.Main.Version != "" {
		version = info.Main.Version
	}
	if override != "" {
		version = override
	}
	return version
}

// handleVersion handles the project-independent version command before the
// general flag parser resolves HOME, the working directory, or any provider.
func handleVersion(args []string, stdout, stderr io.Writer, version string) (bool, int) {
	if len(args) == 0 || args[0] != "version" {
		return false, 0
	}

	jsonOutput := false
	switch {
	case len(args) == 1:
	case len(args) == 2 && args[1] == "--json":
		jsonOutput = true
	default:
		_, _ = io.WriteString(stderr, "gaebokchi: usage: gaebokchi version [--json]\n")
		return true, 2
	}

	if jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(versionOutput{Name: productName, Version: version}); err != nil {
			_, _ = io.WriteString(stderr, "gaebokchi: cannot write version to stdout\n")
			return true, 1
		}
		return true, 0
	}
	if _, err := fmt.Fprintf(stdout, "%s %s\n", productName, version); err != nil {
		_, _ = io.WriteString(stderr, "gaebokchi: cannot write version to stdout\n")
		return true, 1
	}
	return true, 0
}
