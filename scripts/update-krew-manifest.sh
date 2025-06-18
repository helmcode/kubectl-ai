#!/bin/bash
set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.1.0"
    exit 1
fi

VERSION=$1
FILE_PATH="../krew-manifest.yaml"

# Determine sed inline flag ('' on macOS, nothing on GNU)
if [[ "$(uname)" == "Darwin" ]]; then
    SED_INLINE=("-i" "")
else
    SED_INLINE=("-i")
fi
REPO="helmcode/kubectl-ai"

echo "Updating krew-manifest.yaml for version $VERSION..."

# Function to get SHA256 from GitHub release
get_sha256() {
    local filename=$1
    curl -sL "https://github.com/${REPO}/releases/download/${VERSION}/${filename}.sha256" | awk '{print $1}'
}

# Update version
sed "${SED_INLINE[@]}" "s/version: .*/version: ${VERSION}/" ${FILE_PATH}

# Update URLs
sed "${SED_INLINE[@]}" "s|download/.*/kubectl-ai-|download/${VERSION}/kubectl-ai-|g" ${FILE_PATH}

# Update SHA256 checksums
echo "Fetching SHA256 checksums..."

# Linux AMD64
SHA=$(get_sha256 "kubectl-ai-linux-amd64.tar.gz")
sed "${SED_INLINE[@]}" "/os: linux/,/arch: amd64/{/sha256:/s/sha256: .*/sha256: ${SHA}/;}" ${FILE_PATH}

# Linux ARM64
SHA=$(get_sha256 "kubectl-ai-linux-arm64.tar.gz")
sed "${SED_INLINE[@]}" "/os: linux/,/arch: arm64/{/sha256:/s/sha256: .*/sha256: ${SHA}/;}" ${FILE_PATH}

# Darwin AMD64
SHA=$(get_sha256 "kubectl-ai-darwin-amd64.tar.gz")
sed "${SED_INLINE[@]}" "/os: darwin/,/arch: amd64/{/sha256:/s/sha256: .*/sha256: ${SHA}/;}" ${FILE_PATH}

# Darwin ARM64
SHA=$(get_sha256 "kubectl-ai-darwin-arm64.tar.gz")
sed "${SED_INLINE[@]}" "/os: darwin/,/arch: arm64/{/sha256:/s/sha256: .*/sha256: ${SHA}/;}" ${FILE_PATH}

# Windows AMD64
SHA=$(get_sha256 "kubectl-ai-windows-amd64.exe.zip")
sed "${SED_INLINE[@]}" "/os: windows/,/arch: amd64/{/sha256:/s/sha256: .*/sha256: ${SHA}/;}" ${FILE_PATH}

echo "✅ Updated krew-manifest.yaml for version $VERSION"
echo ""
echo "Next steps:"
echo "1. Review the changes: git diff ${FILE_PATH}"
echo "2. Test locally: kubectl krew install --manifest=${FILE_PATH}"
echo "3. Submit to krew-index: https://github.com/kubernetes-sigs/krew-index"
