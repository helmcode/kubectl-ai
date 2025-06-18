# 🤖 kubectl-ai

**AI-powered Kubectl Plugin for Kubernetes cluster management.**

`kubectl ai` is a Kubernetes plugin that leverages LLM models to analyse your cluster resources and give you clear, actionable troubleshooting advice – right in your terminal.

---

## Why kubectl-ai?

* ✅ **Root-cause analysis** – stop guessing why your deployment is broken.
* ✅ **Actionable fixes** – concrete `kubectl` / `helm` commands you can copy-paste.
* ✅ **Understands the whole picture** – pods, deployments, services, CRDs, ingresses…
* ✅ **Human or machine output** – pretty terminal format, or JSON / YAML for automation.

---

## ⚡ Quick start

### 1. Prerequisites

* Go 1.21+
* An Anthropic **API key** exported as `ANTHROPIC_API_KEY`
* Access to the cluster you want to debug (via `kubectl` context)

```bash
export ANTHROPIC_API_KEY="sk-..."
```

---

## 📦 Install

### A) Build from source

```bash
git clone https://github.com/helmcode/kubectl-ai.git
cd kubectl-ai
# build the binary for your OS/ARCH
GOOS=$(uname -s | tr '[:upper:]' '[:lower:]') \
GOARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') \
go build -o kubectl-ai ./...
# add it to PATH
sudo mv kubectl-ai /usr/local/bin/
```

### B) Download a pre-compiled release

Grab the appropriate archive from the [Releases page](https://github.com/helmcode/kubectl-ai/releases) – e.g.:

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -LO "https://github.com/helmcode/kubectl-ai/releases/latest/download/kubectl-ai-${OS}-${ARCH}.tar.gz"

tar -xzf kubectl-ai-${OS}-${ARCH}.tar.gz
chmod +x kubectl-ai-${OS}-${ARCH}
sudo mv kubectl-ai-${OS}-${ARCH} /usr/local/bin/kubectl-ai
```

### C) Install with Krew (recommended)

[Krew](https://krew.sigs.k8s.io/) is the package manager for `kubectl` plugins. Install Krew from [here](https://krew.sigs.k8s.io/docs/user-guide/setup/install/).

```bash
# install kubectl-ai from manifest (Waiting for us to be accepted into krew-index)
git clone https://github.com/helmcode/kubectl-ai.git
cd kubectl-ai
kubectl krew install --manifest=krew-manifest.yaml
```

---

## 📚 Usage examples

```bash
# Analyse a crashing deployment
kubectl ai debug "pods are crashing" -r deployment/nginx

# Analyse multiple resources
kubectl ai debug "secrets not updating" \
  -r deployment/vault -r vaultstaticsecret/creds

# Analyse all resources in a namespace
kubectl ai debug "high memory usage" -n production --all

# Output as JSON
kubectl ai debug "slow startup" -r deployment/api -o json
```

---

## 🤝 Contributing

PRs and issues are welcome!
