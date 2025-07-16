# 🤖 kubectl-ai

**AI-powered Kubectl Plugin for Kubernetes cluster management.**

`kubectl ai` is a Kubernetes plugin that leverages LLM models to analyse your cluster resources and give you clear, actionable troubleshooting advice – right in your terminal.

---

## Why kubectl-ai?

* ✅ **Root-cause analysis** – stop guessing why your deployment is broken.
* ✅ **Actionable fixes** – concrete `kubectl` / `helm` commands you can copy-paste.
* ✅ **Understands the whole picture** – pods, deployments, services, CRDs, ingresses…
* ✅ **Human or machine output** – pretty terminal format, or JSON / YAML for automation.
* ✅ **Multiple LLM providers** – supports Claude (Anthropic) and OpenAI models.

---

## ⚡ Quick start

### 1. Prerequisites

* Go 1.21+
* An **API key** for your chosen LLM provider:
  - **Claude (Anthropic)**: `ANTHROPIC_API_KEY`
  - **OpenAI**: `OPENAI_API_KEY`
* Access to the cluster you want to debug (via `kubectl` context)

```bash
# For Claude (default)
export ANTHROPIC_API_KEY="sk-..."

# For OpenAI
export OPENAI_API_KEY="sk-..."
export LLM_PROVIDER="openai"  # Optional: auto-detects from API key
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
go build
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

# if you have kubectl-ai installed already, you can update it with
kubectl krew install --manifest=krew-manifest.yaml --force
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

# Use specific LLM provider
kubectl ai debug "networking issues" -r deployment/app --provider openai

# Use specific model
kubectl ai debug "memory leaks" -r deployment/app --provider openai --model gpt-4o

# Use environment variables to set provider and model
export LLM_PROVIDER="openai"
export OPENAI_MODEL="gpt-4o-mini"
kubectl ai debug "performance issues" -r deployment/app

# Override environment with command line flags
kubectl ai debug "storage issues" -r deployment/app --provider claude --model claude-3-opus-20240229
```

---

## 🔧 LLM Provider Configuration

### Claude (Anthropic) - Default

```bash
export ANTHROPIC_API_KEY="sk-..."
# Optional: specify model (default: claude-3-5-sonnet-20241022)
export CLAUDE_MODEL="claude-3-5-sonnet-20241022"
```

### OpenAI

```bash
export OPENAI_API_KEY="sk-..."
# Optional: specify model (default: gpt-4o)
export OPENAI_MODEL="gpt-4o"
# Optional: specify provider explicitly (auto-detects from API key if not set)
export LLM_PROVIDER="openai"
```

### Configuration Priority

1. **Command line flags** (`--provider`, `--model`) - highest priority
2. **Environment variables** (`LLM_PROVIDER`, `OPENAI_MODEL`, `CLAUDE_MODEL`)
3. **Auto-detection** - based on available API keys (Claude preferred if both available)

### Command Line Options

- `--provider`: Explicitly choose LLM provider (`claude`, `openai`)
- `--model`: Override the default model for the selected provider
- Auto-detection: If no provider is specified, the tool auto-detects based on available API keys

---

## 📋 Complete Command Reference

```bash
kubectl ai debug PROBLEM [flags]

Flags:
  -h, --help              help for debug
      --kubeconfig string path to kubeconfig file (default "~/.kube/config")
      --context string    kubeconfig context (overrides current-context)
  -n, --namespace string  kubernetes namespace (default "default")
  -r, --resource strings  resources to analyze (e.g., deployment/nginx, pod/nginx-xxx)
      --all               analyze all resources in the namespace
  -o, --output string     output format (human, json, yaml) (default "human")
  -v, --verbose           verbose output
      --provider string   LLM provider (claude, openai). Defaults to auto-detect from env
      --model string      LLM model to use (overrides default)
```

---

## 🤝 Contributing

PRs and issues are welcome!
