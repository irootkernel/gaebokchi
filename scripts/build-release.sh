#!/bin/sh
# Build local release artifacts for the four v0.1 targets with version
# metadata and SHA-256 checksums. Stages into dist/ only; never
# publishes, signs, or bundles provider binaries.
set -eu

VERSION="${1:-0.1.0}"
DIST="dist/gaebokchi-${VERSION}"
GO_SHA_ASMFLAGS=""
export CGO_ENABLED=0

TARGETS="darwin/arm64 darwin/amd64 linux/arm64 linux/amd64"

rm -rf "$DIST"
mkdir -p "$DIST"

for target in $TARGETS; do
  GOOS="${target%/*}"
  GOARCH="${target#*/}"
  name="gaebokchi-${GOOS}-${GOARCH}"
  dir="${DIST}/${name}"
  mkdir -p "${dir}/docs"
  GOOS="$GOOS" GOARCH="$GOARCH" go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o "${dir}/gaebokchi" ./cmd/gaebokchi
  cp README.md "${dir}/README.md"
  cp docs/requirement.md docs/architecture.md docs/plan.md "${dir}/docs/"
  # COPYFILE_DISABLE keeps macOS tar from embedding AppleDouble ._*
  # sidecar members for extended attributes; GNU tar on Linux ignores
  # the variable, so the same invocation is portable.
  COPYFILE_DISABLE=1 tar -czf "${DIST}/${name}.tar.gz" -C "$DIST" "${name}"
  rm -rf "${dir}"
done

(cd "$DIST" && shasum -a 256 gaebokchi-*.tar.gz > SHA256SUMS)
echo "Release staged in ${DIST}:"
cat "${DIST}/SHA256SUMS"
