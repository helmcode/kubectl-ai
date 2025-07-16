package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"path/filepath"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/helmcode/kubectl-ai/pkg/analyzer"
	"github.com/helmcode/kubectl-ai/pkg/formatter"
	"github.com/helmcode/kubectl-ai/pkg/k8s"
	"github.com/helmcode/kubectl-ai/pkg/llm"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"
)

var (
	metricsKubeconfig      string
	metricsNamespace       string
	metricsKubeContext     string
	metricsResources       []string
	metricsAllDeployments  bool
	metricsOutputFormat    string
	metricsVerbose         bool
	metricsLLMProvider     string
	metricsLLMModel        string
	metricsAnalyze         bool
	metricsDuration        string
	metricsCompareScaling  bool
	metricsHPAAnalysis     bool
	metricsKEDAAnalysis    bool
)

func NewMetricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics RESOURCE [flags]",
		Short: "Analyze Kubernetes resource metrics with AI assistance",
		Long: `Analyze Kubernetes resource metrics using AI to identify performance issues and scaling opportunities.

Examples:
  # Analyze metrics for a specific deployment
  kubectl ai metrics deployment/nginx --analyze

  # Analyze metrics with duration and scaling config comparison
  kubectl ai metrics deployment/api --duration 7d --compare-scaling-config

  # Analyze all deployments in production namespace
  kubectl ai metrics --all-deployments -n production

  # Analyze with HPA and KEDA analysis
  kubectl ai metrics deployment/worker --hpa-analysis --keda-analysis`,
		Args: cobra.RangeArgs(0, 1),
		RunE: runMetrics,
	}

	// Flags
	if home := homedir.HomeDir(); home != "" {
		cmd.Flags().StringVar(&metricsKubeconfig, "kubeconfig", "~/.kube/config", "Path to kubeconfig file")
	}

	cmd.Flags().StringVarP(&metricsNamespace, "namespace", "n", "default", "Kubernetes namespace")
	cmd.Flags().StringVar(&metricsKubeContext, "context", "", "Kubeconfig context (overrides current-context)")
	cmd.Flags().StringSliceVarP(&metricsResources, "resource", "r", []string{}, "Resources to analyze (e.g., deployment/nginx)")
	cmd.Flags().BoolVar(&metricsAllDeployments, "all-deployments", false, "Analyze all deployments in the namespace")
	cmd.Flags().StringVarP(&metricsOutputFormat, "output", "o", "human", "Output format (human, json, yaml)")
	cmd.Flags().BoolVarP(&metricsVerbose, "verbose", "v", false, "Verbose output")
	cmd.Flags().StringVar(&metricsLLMProvider, "provider", "", "LLM provider (claude, openai). Defaults to auto-detect from env")
	cmd.Flags().StringVar(&metricsLLMModel, "model", "", "LLM model to use (overrides default)")
	cmd.Flags().BoolVar(&metricsAnalyze, "analyze", false, "Enable AI analysis of metrics")
	cmd.Flags().StringVar(&metricsDuration, "duration", "1h", "Duration for metrics analysis (e.g., 1h, 7d)")
	cmd.Flags().BoolVar(&metricsCompareScaling, "compare-scaling-config", false, "Compare metrics with scaling configuration")
	cmd.Flags().BoolVar(&metricsHPAAnalysis, "hpa-analysis", false, "Include HPA analysis in metrics")
	cmd.Flags().BoolVar(&metricsKEDAAnalysis, "keda-analysis", false, "Include KEDA analysis in metrics")

	return cmd
}

func runMetrics(cmd *cobra.Command, args []string) error {
	// Determine what to analyze
	var targetResource string
	if len(args) > 0 {
		targetResource = args[0]
	}

	// Validate inputs
	if !metricsAllDeployments && len(metricsResources) == 0 && targetResource == "" {
		return fmt.Errorf("specify a resource, use --all-deployments flag, or provide resources with -r")
	}

	// Build resource list
	var finalResources []string
	if targetResource != "" {
		finalResources = append(finalResources, targetResource)
	}
	finalResources = append(finalResources, metricsResources...)

	// Show what we're doing
	printMetricsHeader(targetResource, finalResources)

	// Create spinner for visual feedback
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Suffix = " Connecting to Kubernetes cluster..."
	s.Start()

	// Expand home symbol in kubeconfig if needed
	if strings.HasPrefix(metricsKubeconfig, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			metricsKubeconfig = filepath.Join(homeDir, metricsKubeconfig[2:])
		}
	}

	// Initialize K8s client
	k8sClient, err := k8s.NewClient(metricsKubeconfig, metricsKubeContext)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}
	s.Stop()
	printSuccess("Connected to Kubernetes cluster")

	s.Suffix = " Gathering Kubernetes resources and metrics..."
	s.Start()

	// Gather resources and metrics
	resourcesData, err := k8sClient.GatherMetricsResources(metricsNamespace, finalResources, metricsAllDeployments, metricsDuration)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to gather resources and metrics: %w", err)
	}

	s.Stop()
	printSuccess(fmt.Sprintf("Gathered metrics for %d resources", len(resourcesData)))

	// If no AI analysis requested, just show the metrics
	if !metricsAnalyze {
		formatter.DisplayMetrics(resourcesData, metricsOutputFormat)
		return nil
	}

	s.Suffix = " Initializing AI client..."
	s.Start()

	// Initialize LLM client using factory
	llmClient, err := llm.CreateFromEnv(metricsLLMProvider, metricsLLMModel)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	s.Stop()
	printSuccess("AI client initialized")

	// Show LLM provider and model info
	printLLMInfo(llmClient)
	fmt.Println()

	s.Suffix = " Analyzing metrics with AI..."
	s.Start()

	aiAnalyzer := analyzer.NewWithLLM(llmClient)
	analysis, err := aiAnalyzer.AnalyzeMetrics(resourcesData, metricsDuration, metricsCompareScaling, metricsHPAAnalysis, metricsKEDAAnalysis)
	if err != nil {
		s.Stop()
		return fmt.Errorf("AI metrics analysis failed: %w", err)
	}

	s.Stop()
	printSuccess("Metrics analysis complete")

	formatter.DisplayResults(analysis, metricsOutputFormat)

	return nil
}

func printMetricsHeader(targetResource string, resources []string) {
	cyan := color.New(color.FgCyan, color.Bold)
	fmt.Println()
	cyan.Println("📊 Kubernetes AI Metrics Analyzer")
	
	if targetResource != "" {
		fmt.Printf("🎯 Target Resource: %s\n", targetResource)
	}
	
	if len(resources) > 0 {
		fmt.Printf("📋 Resources: %s\n", strings.Join(resources, ", "))
	}
	
	fmt.Printf("📍 Namespace: %s\n", metricsNamespace)
	fmt.Printf("⏱️  Duration: %s\n", metricsDuration)
	
	if metricsAllDeployments {
		fmt.Println("🚀 Target: all deployments")
	}
	
	var features []string
	if metricsAnalyze {
		features = append(features, "AI Analysis")
	}
	if metricsCompareScaling {
		features = append(features, "Scaling Config Comparison")
	}
	if metricsHPAAnalysis {
		features = append(features, "HPA Analysis")
	}
	if metricsKEDAAnalysis {
		features = append(features, "KEDA Analysis")
	}
	
	if len(features) > 0 {
		fmt.Printf("🔧 Features: %s\n", strings.Join(features, ", "))
	}
	
	fmt.Println()
}