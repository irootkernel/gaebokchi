#!/bin/sh
# Build local release artifacts for the four supported platforms with version
# metadata and SHA-256 checksums. Stages into dist/ only; never
# publishes, signs, or bundles provider binaries.
set -eu

RELEASE_VERSION="${1:-v0.1.0}"
case "$RELEASE_VERSION" in
  v?*) ;;
  *) echo "usage: $0 vVERSION" >&2; exit 2 ;;
esac
DIST="dist/gaebokchi-${RELEASE_VERSION}"
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
  mkdir -p "${dir}/docs/specs" "${dir}/docs/architecture" "${dir}/docs/roadmap"
  GOOS="$GOOS" GOARCH="$GOARCH" go build -trimpath \
    -ldflags "-s -w -X main.buildVersion=${RELEASE_VERSION}" \
    -o "${dir}/gaebokchi" ./cmd/gaebokchi
  cp README.md "${dir}/README.md"
  cp docs/specs/README.md "${dir}/docs/specs/README.md"
  cp docs/architecture/README.md "${dir}/docs/architecture/README.md"
  cp docs/roadmap/README.md "${dir}/docs/roadmap/README.md"
  # COPYFILE_DISABLE keeps macOS tar from embedding AppleDouble ._*
  # sidecar members for extended attributes; GNU tar on Linux ignores
  # the variable, so the same invocation is portable.
  COPYFILE_DISABLE=1 tar -czf "${DIST}/${name}.tar.gz" -C "$DIST" "${name}"
  rm -rf "${dir}"
done

(cd "$DIST" && shasum -a 256 gaebokchi-*.tar.gz > SHA256SUMS)
echo "Release staged in ${DIST}:"
cat "${DIST}/SHA256SUMS"
