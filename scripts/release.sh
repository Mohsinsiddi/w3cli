#!/usr/bin/env bash
# release.sh — Tag, build, publish GitHub release, and publish to npm.
#
# Prerequisites:
#   - gh CLI authenticated (gh auth login)
#   - npm authenticated (npm login --scope=@siddi_404 or just npm login)
#   - VERSION env var set, e.g.: VERSION=1.0.1 ./scripts/release.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
NPM_DIR="${ROOT_DIR}/npm"
DIST_DIR="${ROOT_DIR}/dist"
BINARY="w3cli"

# Resolve version.
VERSION="${VERSION:-}"
if [[ -z "${VERSION}" ]]; then
  echo "Error: set VERSION before running this script."
  echo "  VERSION=1.2.0 ./scripts/release.sh"
  exit 1
fi

TAG="v${VERSION}"

echo "==> Releasing ${BINARY} ${TAG}"

# ── 1. Sanity checks ────────────────────────────────────────────────────────
echo "==> Running tests..."
cd "${ROOT_DIR}"
go test ./... -race -count=1

# ── 2. Build all binaries ───────────────────────────────────────────────────
echo "==> Building all platforms..."
rm -rf "${DIST_DIR}"
VERSION="${VERSION}" bash "${SCRIPT_DIR}/build-all.sh"

# ── 3. Git tag ──────────────────────────────────────────────────────────────
echo "==> Creating git tag ${TAG}..."
git tag -a "${TAG}" -m "Release ${TAG}"
git push origin "${TAG}"

# ── 4. GitHub release ───────────────────────────────────────────────────────
echo "==> Creating GitHub release ${TAG}..."
ASSETS=()
for f in "${DIST_DIR}/${BINARY}"-*; do
  ASSETS+=("$f")
done
if [[ -f "${DIST_DIR}/checksums.txt" ]]; then
  ASSETS+=("${DIST_DIR}/checksums.txt")
fi

gh release create "${TAG}" \
  --title "${BINARY} ${TAG}" \
  --notes "Release ${TAG} — see CHANGELOG for details." \
  "${ASSETS[@]}"

echo "==> GitHub release created: https://github.com/Mohsinsiddi/w3cli/releases/tag/${TAG}"

# ── 5. Update npm package version and publish ───────────────────────────────
echo "==> Updating npm package version to ${VERSION}..."
cd "${NPM_DIR}"

# Update version in package.json without running lifecycle scripts.
node -e "
  const fs = require('fs');
  const pkg = JSON.parse(fs.readFileSync('package.json', 'utf8'));
  pkg.version = '${VERSION}';
  fs.writeFileSync('package.json', JSON.stringify(pkg, null, 2) + '\n');
  console.log('Updated package.json to version ${VERSION}');
"

echo "==> Publishing to npm..."
npm publish --access public

echo ""
echo "Release ${TAG} complete!"
echo "  GitHub: https://github.com/Mohsinsiddi/w3cli/releases/tag/${TAG}"
echo "  npm:    https://www.npmjs.com/package/w3cli"
