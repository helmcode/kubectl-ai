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
* ✅ **Metrics analysis** – visual charts and AI-powered insights from Prometheus data.
* ✅ **Scaling recommendations** – intelligent HPA and KEDA configuration suggestions.

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

# Analyze metrics with visual charts
kubectl ai metrics deployment/api -n production

# Get AI-powered scaling recommendations
kubectl ai metrics deployment/backend --analyze --hpa-analysis

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

## 📊 Metrics Analysis

The `kubectl ai metrics` command provides comprehensive metrics analysis with visual charts and AI-powered insights.

### Basic Metrics Visualization

```bash
# Show basic metrics charts for a deployment
kubectl ai metrics deployment/backend -n production

# Show metrics for specific duration
kubectl ai metrics deployment/api --duration 7d

# Analyze all deployments in namespace
kubectl ai metrics --all -n production
```

### AI-Powered Analysis

```bash
# Get AI analysis of metrics patterns
kubectl ai metrics deployment/backend --analyze

# Get HPA recommendations
kubectl ai metrics deployment/api --hpa-analysis

# Get KEDA scaling recommendations  
kubectl ai metrics deployment/worker --keda-analysis

# Combined analysis with all insights
kubectl ai metrics deployment/app --analyze --hpa-analysis --keda-analysis
```

### Advanced Configuration

```bash
# Use specific Prometheus server
kubectl ai metrics deployment/app --prometheus-url http://prometheus.monitoring:9090

# Analyze with custom duration and specific provider
kubectl ai metrics deployment/api --duration 30d --analyze --provider openai
```

### What You Get

**📈 Visual Charts:**
- CPU usage over time with statistics (avg, min, max)
- Memory usage trends and patterns
- Replica scaling events timeline

**🤖 AI Analysis (with --analyze flag):**
- Intelligent pattern recognition in metrics
- Performance bottleneck identification
- Scaling behavior analysis

**🎯 HPA Recommendations (with --hpa-analysis flag):**
- Optimal min/max replica settings
- CPU/Memory target thresholds
- Complete HPA YAML configuration

**🚀 KEDA Recommendations (with --keda-analysis flag):**
- Event-driven scaling configuration
- Custom scalers for different workloads
- Complete KEDA ScaledObject YAML

**💡 Smart Recommendations:**
- Prioritized action items (high/medium/low)
- Resource optimization suggestions
- Best practices for scaling configuration

---

## 🔧 LLM Provider Configuration

### Claude (Anthropic) - Default

```bash
export ANTHROPIC_API_KEY="sk-..."
# Optional: specify model (default: claude-sonnet-4-20250514)
export CLAUDE_MODEL="claude-sonnet-4-20250514"
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

### Debug Command

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

### Metrics Command

```bash
kubectl ai metrics RESOURCE [flags]

Flags:
  -h, --help                    help for metrics
      --kubeconfig string       path to kubeconfig file (default "~/.kube/config")
      --context string          kubeconfig context (overrides current-context)
  -n, --namespace string        kubernetes namespace (default "default")
  -r, --resource strings        resources to analyze (e.g., deployment/nginx)
      --all                     analyze all deployments in the namespace
  -o, --output string           output format (human, json, yaml) (default "human")
  -v, --verbose                 verbose output
      --provider string         LLM provider (claude, openai). Defaults to auto-detect from env
      --model string            LLM model to use (overrides default)
      --analyze                 perform AI analysis of metrics patterns
      --duration string         duration for metrics analysis (1h, 6h, 24h, 7d, 30d) (default "24h")
      --hpa-analysis            perform HPA-specific analysis
      --keda-analysis           perform KEDA-specific analysis
      --prometheus-url string   Prometheus server URL (auto-detects if not provided)
      --prometheus-namespace    Prometheus namespace for auto-detection
```

---

## 🤝 Contributing

PRs and issues are welcome!
