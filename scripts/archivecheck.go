package main

// archivecheck verifies the raw member list of the staged release
// archives with the standard library (archive/tar + compress/gzip).
// It exists because a host tar listing can hide AppleDouble sidecar
// members — macOS bsdtar collapses `._*` entries by default — so
// membership must be inspected from the archive itself, and because
// document freshness, executable permissions, and the checksum
// manifest belong to the same packaging gate.

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// releaseTargets are the four distribution targets, in archive-name form.
var releaseTargets = []string{
	"darwin-arm64", "darwin-amd64", "linux-arm64", "linux-amd64",
}

// intendedFiles returns the only regular-file members an archive for
// the target may carry, plus the expected directory entries.
func intendedFiles(target string) (files []string, dirs []string) {
	d := "gaebokchi-" + target
	return []string{
			d + "/gaebokchi",
			d + "/README.md",
			d + "/docs/requirement.md",
			d + "/docs/architecture.md",
			d + "/docs/plan.md",
		}, []string{
			d,
			d + "/docs",
		}
}

// repoPathFor maps one intended packaged file to its repository source.
func repoPathFor(member string) string {
	d := member[strings.IndexByte(member, '/')+1:]
	return d // "gaebokchi" stays for the binary; docs/... map directly
}

// verifyArchive reads archivePath raw and verifies, for the given
// target: exactly the intended regular files and expected directory
// entries, no sidecar (.*) or otherwise unexpected members, the
// executable bit on the binary, and byte-identical packaged documents
// against repoRoot. repoRoot is only consulted for the four documents.
func verifyArchive(archivePath, target, repoRoot string) error {
	intended, intendedDirs := intendedFiles(target)
	want := make(map[string]bool, len(intended))
	for _, m := range intended {
		want[m] = true
	}
	got := make(map[string]bool, len(intended))
	gotDirs := make(map[string]bool, len(intendedDirs))

	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("%s: not a gzip archive: %v", archivePath, err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	binaryOK := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("%s: %v", archivePath, err)
		}
		if hdr.Typeflag == tar.TypeXGlobalHeader {
			continue // pax metadata, not a member
		}
		name := strings.TrimSuffix(hdr.Name, "/")
		base := name[strings.LastIndexByte(name, '/')+1:]
		if strings.HasPrefix(base, "._") || base == ".DS_Store" {
			return fmt.Errorf("%s: unintended sidecar member %q", archivePath, hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			gotDirs[name] = true
		case tar.TypeReg:
			if !want[name] {
				return fmt.Errorf("%s: unexpected regular member %q", archivePath, hdr.Name)
			}
			got[name] = true
			if strings.HasSuffix(name, "/gaebokchi") {
				if hdr.Mode&0o111 == 0 {
					return fmt.Errorf("%s: packaged binary %q is not executable", archivePath, hdr.Name)
				}
				binaryOK = true
				continue
			}
			src, err := os.ReadFile(filepath.Join(repoRoot, repoPathFor(name)))
			if err != nil {
				return fmt.Errorf("%s: repository source for %q unavailable: %v", archivePath, hdr.Name, err)
			}
			packaged, err := io.ReadAll(tr)
			if err != nil {
				return fmt.Errorf("%s: reading %q: %v", archivePath, hdr.Name, err)
			}
			if string(src) != string(packaged) {
				return fmt.Errorf("%s: packaged document %q differs from the finalized source", archivePath, hdr.Name)
			}
		default:
			return fmt.Errorf("%s: unexpected non-regular member %q (type %q)", archivePath, hdr.Name, string(rune(hdr.Typeflag)))
		}
	}
	if !binaryOK {
		return fmt.Errorf("%s: intended binary member missing", archivePath)
	}
	var missing []string
	for _, m := range intended {
		if !got[m] {
			missing = append(missing, m)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("%s: missing intended members %v", archivePath, missing)
	}
	for _, d := range intendedDirs {
		if !gotDirs[d] {
			return fmt.Errorf("%s: expected directory entry %q missing", archivePath, d)
		}
	}
	return nil
}

// verifyChecksums re-computes the SHA-256 of every archive listed in
// the staged SHA256SUMS manifest and requires all four targets.
func verifyChecksums(distDir string) error {
	manifest, err := os.ReadFile(filepath.Join(distDir, "SHA256SUMS"))
	if err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(manifest)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return fmt.Errorf("malformed checksum line %q", line)
		}
		want, name := parts[0], parts[1]
		sum, err := fileSHA256(filepath.Join(distDir, name))
		if err != nil {
			return err
		}
		if sum != want {
			return fmt.Errorf("%s: checksum mismatch (manifest %s, actual %s)", name, want, sum)
		}
		seen[strings.TrimPrefix(strings.TrimSuffix(name, ".tar.gz"), "gaebokchi-")] = true
	}
	for _, t := range releaseTargets {
		if !seen[t] {
			return fmt.Errorf("checksum manifest lacks the %s archive", t)
		}
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// checkStagedArchives runs the full packaging gate over a staged dist
// directory: raw membership, permissions, document freshness, and the
// checksum manifest for all four targets.
func checkStagedArchives(distDir, repoRoot string) error {
	for _, t := range releaseTargets {
		archive := filepath.Join(distDir, "gaebokchi-"+t+".tar.gz")
		if err := verifyArchive(archive, t, repoRoot); err != nil {
			return err
		}
	}
	return verifyChecksums(distDir)
}
