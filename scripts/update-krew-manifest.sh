#!/bin/bash
set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.1.0"
    exit 1
fi

VERSION=$1
REPO="helmcode/kubectl-ai"

echo "Updating krew-manifest.yaml for version $VERSION..."

# Function to get SHA256 from GitHub release
get_sha256() {
    local filename=$1
    curl -sL "https://github.com/${REPO}/releases/download/${VERSION}/${filename}.sha256" | awk '{print $1}'
}

# Update version
sed -i "s/version: .*/version: ${VERSION}/" krew-manifest.yaml

# Update URLs
sed -i "s|download/.*/kubectl-ai-|download/${VERSION}/kubectl-ai-|g" krew-manifest.yaml

# Update SHA256 checksums
echo "Fetching SHA256 checksums..."

# Linux AMD64
SHA=$(get_sha256 "kubectl-ai-linux-amd64.tar.gz")
sed -i "/os: linux/,/arch: amd64/{/sha256:/s/sha256: .*/sha256: ${SHA}/}" krew-manifest.yaml

# Linux ARM64
SHA=$(get_sha256 "kubectl-ai-linux-arm64.tar.gz")
sed -i "/os: linux/,/arch: arm64/{/sha256:/s/sha256: .*/sha256: ${SHA}/}" krew-manifest.yaml

# Darwin AMD64
SHA=$(get_sha256 "kubectl-ai-darwin-amd64.tar.gz")
sed -i "/os: darwin/,/arch: amd64/{/sha256:/s/sha256: .*/sha256: ${SHA}/}" krew-manifest.yaml

# Darwin ARM64
SHA=$(get_sha256 "kubectl-ai-darwin-arm64.tar.gz")
sed -i "/os: darwin/,/arch: arm64/{/sha256:/s/sha256: .*/sha256: ${SHA}/}" krew-manifest.yaml

# Windows AMD64
SHA=$(get_sha256 "kubectl-ai-windows-amd64.exe.zip")
sed -i "/os: windows/,/arch: amd64/{/sha256:/s/sha256: .*/sha256: ${SHA}/}" krew-manifest.yaml

echo "✅ Updated krew-manifest.yaml for version $VERSION"
echo ""
echo "Next steps:"
echo "1. Review the changes: git diff krew-manifest.yaml"
echo "2. Test locally: kubectl krew install --manifest=krew-manifest.yaml"
echo "3. Submit to krew-index: https://github.com/kubernetes-sigs/krew-index"
