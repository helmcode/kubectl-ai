#!/usr/bin/env python3
"""
Script to automatically update krew-manifest.yaml with new release URLs and SHA256 checksums
Usage: python3 scripts/update-krew-manifest.py <version>
Example: python3 scripts/update-krew-manifest.py v0.1.3
"""

import sys
import re
import hashlib
import urllib.request
import yaml
import os

def get_sha256_from_url(url):
    """Download file and calculate SHA256"""
    print(f"📥 Downloading {url}...")
    try:
        with urllib.request.urlopen(url) as response:
            content = response.read()
            sha256 = hashlib.sha256(content).hexdigest()
            print(f"✅ SHA256: {sha256}")
            return sha256
    except Exception as e:
        print(f"❌ Failed to download {url}: {e}")
        sys.exit(1)

def update_krew_manifest(version, repo="helmcode/kubectl-ai"):
    """Update krew-manifest.yaml with new version and SHA256 checksums"""
    
    manifest_file = "krew-manifest.yaml"
    
    if not os.path.exists(manifest_file):
        print(f"❌ {manifest_file} not found!")
        sys.exit(1)
    
    # Load the manifest
    with open(manifest_file, 'r') as f:
        content = f.read()
    
    print(f"🔄 Updating krew manifest for version {version}...")
    
    # Update version
    content = re.sub(r'version: v[0-9]+\.[0-9]+\.[0-9]+', f'version: {version}', content)
    
    # Update URLs
    content = re.sub(
        r'download/v[0-9]+\.[0-9]+\.[0-9]+/',
        f'download/{version}/',
        content
    )
    
    # Parse YAML to update SHA256 checksums
    try:
        data = yaml.safe_load(content)
    except yaml.YAMLError as e:
        print(f"❌ Error parsing YAML: {e}")
        sys.exit(1)
    
    # Platform to filename mapping
    platform_files = {
        ('linux', 'amd64'): 'kubectl-ai-linux-amd64.tar.gz',
        ('linux', 'arm64'): 'kubectl-ai-linux-arm64.tar.gz',
        ('darwin', 'amd64'): 'kubectl-ai-darwin-amd64.tar.gz',
        ('darwin', 'arm64'): 'kubectl-ai-darwin-arm64.tar.gz',
        ('windows', 'amd64'): 'kubectl-ai-windows-amd64.exe.zip'
    }
    
    print("🔍 Calculating SHA256 checksums...")
    
    # Update SHA256 for each platform
    for platform in data['spec']['platforms']:
        os_name = platform['selector']['matchLabels']['os']
        arch = platform['selector']['matchLabels']['arch']
        
        if (os_name, arch) in platform_files:
            filename = platform_files[(os_name, arch)]
            url = f"https://github.com/{repo}/releases/download/{version}/{filename}"
            
            # Get SHA256
            sha256 = get_sha256_from_url(url)
            
            # Update the platform SHA256
            platform['sha256'] = sha256
            
            print(f"✅ Updated {os_name}/{arch}: {sha256}")
    
    # Write back to file
    with open(manifest_file, 'w') as f:
        yaml.dump(data, f, default_flow_style=False, sort_keys=False)
    
    print(f"✅ Krew manifest updated successfully!")
    print(f"📄 Updated file: {manifest_file}")
    return True

def main():
    if len(sys.argv) != 2:
        print("Usage: python3 scripts/update-krew-manifest.py <version>")
        print("Example: python3 scripts/update-krew-manifest.py v0.1.3")
        sys.exit(1)
    
    version = sys.argv[1]
    
    # Validate version format
    if not re.match(r'v\d+\.\d+\.\d+', version):
        print("❌ Invalid version format. Use format: v0.1.3")
        sys.exit(1)
    
    success = update_krew_manifest(version)
    
    if success:
        print("")
        print("🎉 Krew manifest update completed!")
        print("📋 Next steps:")
        print("   1. Review the changes in krew-manifest.yaml")
        print("   2. Commit and push the updated manifest")
        print("   3. Submit to krew-index if needed")

if __name__ == "__main__":
    main()