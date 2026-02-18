#!/usr/bin/env bash
# build-all.sh — Cross-platform build script for w3cli.
# Produces binaries in dist/ for all supported platforms.
set -euo pipefail

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")}"
MODULE="github.com/Mohsinsiddi/w3cli"
OUTPUT_DIR="$(cd "$(dirname "$0")/.." && pwd)/dist"
BINARY="w3cli"

# Inject version at build time.
LDFLAGS="-s -w -X ${MODULE}/cmd.Version=${VERSION}"

TARGETS=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

echo "Building w3cli ${VERSION}..."
mkdir -p "${OUTPUT_DIR}"

for target in "${TARGETS[@]}"; do
  GOOS="${target%/*}"
  GOARCH="${target#*/}"

  ext=""
  if [[ "${GOOS}" == "windows" ]]; then
    ext=".exe"
  fi

  out="${OUTPUT_DIR}/${BINARY}-${GOOS}-${GOARCH}${ext}"
  echo "  Building ${out}..."

  GOOS="${GOOS}" GOARCH="${GOARCH}" CGO_ENABLED=0 \
    go build -trimpath -ldflags "${LDFLAGS}" -o "${out}" .

  # Generate SHA-256 checksum.
  if command -v sha256sum &>/dev/null; then
    sha256sum "${out}" >> "${OUTPUT_DIR}/checksums.txt"
  elif command -v shasum &>/dev/null; then
    shasum -a 256 "${out}" >> "${OUTPUT_DIR}/checksums.txt"
  fi
done

echo ""
echo "Build complete. Binaries:"
ls -lh "${OUTPUT_DIR}"
echo ""
echo "Checksums:"
cat "${OUTPUT_DIR}/checksums.txt" 2>/dev/null || true
