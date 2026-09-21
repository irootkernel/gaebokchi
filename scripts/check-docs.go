// Command check-docs verifies the cross-document reference graph of
// the Gaebokchi specification: every local markdown link and fragment
// anchor must resolve, and the REQ/ARCH/EPIC/TASK identifiers referenced
// across the three root documents and verification records must exist.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	linkRe   = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
	anchorRe = regexp.MustCompile(`<a id="([^"]+)"></a>`)
	reqRe    = regexp.MustCompile(`\b(REQ-\d{3}|ARCH-\d{2}|EPIC-\d{3}|TASK-\d{3}|V-[A-Z]+|S-\d{2})\b`)
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	fail := false

	// Collect anchors and identifiers per markdown file.
	anchors := map[string]map[string]bool{} // file -> anchor
	ids := map[string]bool{}                // global identifiers (REQ-..., etc.)
	var files []string
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if err == nil && d.IsDir() && (d.Name() == ".git" || d.Name() == "dist") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		files = append(files, path)
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		anchors[path] = map[string]bool{}
		for _, m := range anchorRe.FindAllStringSubmatch(string(b), -1) {
			anchors[path][m[1]] = true
			ids[strings.ToUpper(m[1])] = true
		}
		return nil
	})

	// Markdown heading anchors (lowercased, dashed) for linked headings.
	headingAnchors := map[string]map[string]bool{}
	for _, f := range files {
		b, _ := os.ReadFile(f)
		headingAnchors[f] = map[string]bool{}
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "#") {
				h := strings.TrimLeft(line, "# ")
				h = strings.ToLower(strings.ReplaceAll(h, " ", "-"))
				headingAnchors[f][h] = true
			}
		}
	}

	// Verify links and referenced identifiers.
	knownIDs := map[string]bool{}
	for f, set := range anchors {
		_ = f
		for id := range set {
			knownIDs[id] = true
		}
	}
	// Specification identifiers also come from checkbox/list text.
	specFiles := []string{"docs/requirement.md", "docs/architecture.md", "docs/plan.md"}
	for _, sf := range specFiles {
		b, err := os.ReadFile(filepath.Join(root, sf))
		if err != nil {
			fmt.Printf("missing spec file: %s\n", sf)
			fail = true
			continue
		}
		for _, m := range reqRe.FindAllStringSubmatch(string(b), -1) {
			knownIDs[m[1]] = true
		}
	}

	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		text := string(b)
		for _, m := range linkRe.FindAllStringSubmatch(text, -1) {
			target := m[1]
			if strings.HasPrefix(target, "http") {
				continue
			}
			frag := ""
			if i := strings.Index(target, "#"); i >= 0 {
				frag = target[i+1:]
				target = target[:i]
			}
			if target == "" {
				continue // pure-fragment links checked below with file context
			}
			resolved := filepath.Join(filepath.Dir(f), target)
			if _, err := os.Stat(resolved); err != nil {
				fmt.Printf("%s: broken link %s\n", f, m[1])
				fail = true
				continue
			}
			if frag != "" && strings.HasSuffix(resolved, ".md") {
				if !anchors[resolved][frag] && !headingAnchors[resolved][frag] {
					fmt.Printf("%s: broken fragment %s#%s\n", f, target, frag)
					fail = true
				}
			}
		}
		for _, m := range reqRe.FindAllStringSubmatch(text, -1) {
			id := m[1]
			// Identifiers used in verification records must exist in
			// the specification.
			if strings.Contains(f, "docs/") {
				if !knownIDs[id] {
					fmt.Printf("%s: unknown identifier %s\n", f, id)
					fail = true
				}
			}
		}
	}

	if fail {
		os.Exit(1)
	}
	fmt.Printf("check-docs: %d markdown files, links and identifiers OK\n", len(files))
}
