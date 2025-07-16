package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"path/filepath"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/helmcode/kubectl-ai/pkg/formatter"
	"github.com/helmcode/kubectl-ai/pkg/k8s"
	"github.com/helmcode/kubectl-ai/pkg/llm"
	"github.com/helmcode/kubectl-ai/pkg/metrics"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"
)

var (
	// Common flags (similar to debug command)
	metricsKubeconfig   string
	metricsNamespace    string
	metricsKubeContext  string
	metricsResources    []string
	metricsAllResources bool
	metricsOutputFormat string
	metricsVerbose      bool
	metricsLLMProvider  string
	metricsLLMModel     string

	// Metrics-specific flags
	analyzeScaling      bool
	duration            string
	hpaAnalysis         bool
	kedaAnalysis        bool
	prometheusURL       string
	prometheusNamespace string
)

func NewMetricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics RESOURCE [flags]",
		Short: "Analyze Kubernetes resources metrics with visual charts and AI insights",
		Long: `Analyze Kubernetes resources metrics using Prometheus data with visual charts and AI analysis.

By default, this command shows:
- CPU usage line chart with statistics (average, min, max)
- Memory usage line chart with statistics (average, min, max)  
- Replica scaling events bar chart

With the --analyze flag, it additionally provides:
- AI analysis of the metrics patterns
- HPA (Horizontal Pod Autoscaler) recommendations
- KEDA scaling recommendations

Examples:
  # Show metrics charts for a deployment
  kubectl ai metrics deploy/backend -n backend

  # Show metrics with AI analysis and recommendations
  kubectl ai metrics deploy/backend -n backend --analyze

  # Analyze with specific duration
  kubectl ai metrics deploy/api --duration 7d --analyze

  # Analyze all deployments in a namespace
  kubectl ai metrics --all-deployments -n production --analyze

  # Get HPA and KEDA recommendations
  kubectl ai metrics deployment/worker --hpa-analysis --keda-analysis

  # Use specific Prometheus URL
  kubectl ai metrics deployment/app --prometheus-url http://prometheus.monitoring:9090`,
		Args: cobra.MaximumNArgs(1),
		RunE: runMetrics,
	}

	// Common flags (similar to debug command)
	if home := homedir.HomeDir(); home != "" {
		cmd.Flags().StringVar(&metricsKubeconfig, "kubeconfig", "~/.kube/config", "Path to kubeconfig file")
	}

	cmd.Flags().StringVarP(&metricsNamespace, "namespace", "n", "default", "Kubernetes namespace")
	cmd.Flags().StringVar(&metricsKubeContext, "context", "", "Kubeconfig context (overrides current-context)")
	cmd.Flags().StringSliceVarP(&metricsResources, "resource", "r", []string{}, "Resources to analyze (e.g., deployment/nginx)")
	cmd.Flags().BoolVar(&metricsAllResources, "all", false, "Analyze all deployments in the namespace")
	cmd.Flags().StringVarP(&metricsOutputFormat, "output", "o", "human", "Output format (human, json, yaml)")
	cmd.Flags().BoolVarP(&metricsVerbose, "verbose", "v", false, "Verbose output")
	cmd.Flags().StringVar(&metricsLLMProvider, "provider", "", "LLM provider (claude, openai). Defaults to auto-detect from env")
	cmd.Flags().StringVar(&metricsLLMModel, "model", "", "LLM model to use (overrides default)")

	// Metrics-specific flags
	cmd.Flags().BoolVar(&analyzeScaling, "analyze", false, "Perform scaling analysis based on metrics")
	cmd.Flags().StringVar(&duration, "duration", "24h", "Duration for metrics analysis (1h, 6h, 24h, 7d, 30d)")
	cmd.Flags().BoolVar(&hpaAnalysis, "hpa-analysis", false, "Perform HPA-specific analysis")
	cmd.Flags().BoolVar(&kedaAnalysis, "keda-analysis", false, "Perform KEDA-specific analysis")
	cmd.Flags().StringVar(&prometheusURL, "prometheus-url", "", "Prometheus server URL (auto-detects if not provided)")
	cmd.Flags().StringVar(&prometheusNamespace, "prometheus-namespace", "", "Prometheus namespace for auto-detection")

	return cmd
}

func runMetrics(cmd *cobra.Command, args []string) error {
	// Parse resource if provided
	var targetResource string
	if len(args) > 0 {
		targetResource = args[0]
		metricsResources = append(metricsResources, targetResource)
	}

	// Validate inputs
	if !metricsAllResources && len(metricsResources) == 0 {
		return fmt.Errorf("either specify a resource, use -r flag, or use --all flag")
	}

	// Show what we're doing
	printMetricsHeader(targetResource)

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

	// Initialize Prometheus client with auto-detection (no spinner - we show detailed progress)
	prometheusClient, err := metrics.NewPrometheusClient(prometheusURL, prometheusNamespace, metricsKubeconfig, k8sClient)
	if err != nil {
		return fmt.Errorf("failed to connect to Prometheus: %w", err)
	}

	// Ensure cleanup of port-forward when function exits
	defer prometheusClient.Close()

	s.Suffix = " Gathering Kubernetes resources..."
	s.Start()

	// Determine which resources to analyze
	var resourcesToAnalyze []string
	if metricsAllResources {
		resourcesToAnalyze = []string{} // Will be handled by GatherResources
	} else {
		resourcesToAnalyze = metricsResources
	}

	resourcesData, err := k8sClient.GatherResources(metricsNamespace, resourcesToAnalyze, metricsAllResources)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to gather resources: %w", err)
	}

	s.Stop()
	printSuccess(fmt.Sprintf("Gathered %d resources", len(resourcesData)))

	s.Suffix = " Collecting metrics data..."
	s.Start()

	// Gather metrics for the specified duration
	// Convert map to slice for metrics collection
	resourcesList := make([]interface{}, 0, len(resourcesData))
	for _, resource := range resourcesData {
		resourcesList = append(resourcesList, resource)
	}

	metricsData, err := prometheusClient.GatherMetrics(resourcesList, duration)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to gather metrics: %w", err)
	}

	s.Stop()
	printSuccess(fmt.Sprintf("Collected metrics for %s duration", duration))

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

	// Create metrics analyzer
	metricsAnalyzer := metrics.NewAnalyzer(llmClient, prometheusClient, k8sClient)

	// Perform analysis based on flags
	analysisRequest := &metrics.AnalysisRequest{
		Resources:      resourcesList,
		MetricsData:    metricsData,
		Duration:       duration,
		AnalyzeScaling: analyzeScaling,
		HPAAnalysis:    hpaAnalysis,
		KEDAAnalysis:   kedaAnalysis,
		Namespace:      metricsNamespace,
	}

	analysis, err := metricsAnalyzer.AnalyzeMetrics(analysisRequest)
	if err != nil {
		s.Stop()
		return fmt.Errorf("metrics analysis failed: %w", err)
	}

	s.Stop()
	printSuccess("Metrics analysis complete")

	// Display results
	displayMetricsResults(analysis, metricsOutputFormat)

	return nil
}

// displayMetricsResults displays the metrics analysis results
func displayMetricsResults(analysis *metrics.AnalysisResult, outputFormat string) {
	switch outputFormat {
	case "json":
		displayMetricsJSON(analysis)
	case "yaml":
		displayMetricsYAML(analysis)
	default:
		displayMetricsHuman(analysis)
	}
}

// displayMetricsHuman displays results in human-readable format with enhanced charts
func displayMetricsHuman(analysis *metrics.AnalysisResult) {
	cyan := color.New(color.FgCyan, color.Bold)
	fmt.Println()
	cyan.Println("📊 METRICS ANALYSIS")
	fmt.Println(strings.Repeat("=", 60))

	// Resource information
	fmt.Printf("📦 Resource: %s/%s (%s)\n", analysis.Namespace, analysis.ResourceName, analysis.ResourceType)
	fmt.Printf("📅 Duration: %s\n", analysis.Duration)
	fmt.Println()

	// Display metrics charts
	if len(analysis.MetricsSummary) > 0 {
		// CPU Usage Chart
		if cpuMetric, exists := analysis.MetricsSummary["cpu_utilization"]; exists && len(cpuMetric.Values) > 0 {
			cpuChart := formatter.CreateEnhancedLineChart(cpuMetric.Values, cpuMetric.Timestamps, "CPU", "%", analysis.Duration)
			fmt.Print(cpuChart)
		} else {
			fmt.Println("⚠️  No CPU metrics data available")
		}

		// Memory Usage Chart
		if memoryMetric, exists := analysis.MetricsSummary["memory_utilization"]; exists && len(memoryMetric.Values) > 0 {
			memoryChart := formatter.CreateEnhancedLineChart(memoryMetric.Values, memoryMetric.Timestamps, "Memory", "MB", analysis.Duration)
			fmt.Print(memoryChart)
		} else {
			fmt.Println("⚠️  No Memory metrics data available")
		}
	} else {
		fmt.Println("⚠️  No metrics summary data available")
	}

	// Replica Scaling Chart
	if len(analysis.ScalingEvents) > 0 {
		replicas := make([]int, len(analysis.ScalingEvents))
		timestamps := make([]time.Time, len(analysis.ScalingEvents))

		for i, event := range analysis.ScalingEvents {
			replicas[i] = event.Replicas
			timestamps[i] = event.Timestamp
		}

		replicaChart := formatter.CreateReplicaBarChart(replicas, timestamps, "Replica Scaling Events")
		fmt.Print(replicaChart)
	} else {
		fmt.Println("⚠️  No scaling events data available")
	}

	// Only show AI Analysis and Recommendations when --analyze flag is used
	if analyzeScaling {
		// AI Analysis
		if analysis.Summary != "" {
			cyan.Println("🤖 AI ANALYSIS")
			fmt.Println(strings.Repeat("=", 40))
			fmt.Print(formatter.FormatMarkdownText(analysis.Summary))
			fmt.Println()
		}

		// HPA Recommendations
		if analysis.HPAConfig != nil {
			green := color.New(color.FgGreen, color.Bold)
			green.Println("🎯 HPA RECOMMENDATIONS")
			fmt.Println(strings.Repeat("=", 40))
			fmt.Printf("  Min/Max Replicas: %d/%d\n", analysis.HPAConfig.MinReplicas, analysis.HPAConfig.MaxReplicas)
			if analysis.HPAConfig.TargetCPU > 0 {
				fmt.Printf("  Target CPU: %d%%\n", analysis.HPAConfig.TargetCPU)
			}
			if analysis.HPAConfig.TargetMemory > 0 {
				fmt.Printf("  Target Memory: %d%%\n", analysis.HPAConfig.TargetMemory)
			}
			fmt.Printf("  Reasoning: %s\n", analysis.HPAConfig.Reasoning)
			fmt.Println()

			if analysis.HPAConfig.YAMLConfig != "" {
				fmt.Println("  YAML Configuration:")
				fmt.Printf("```yaml\n%s\n```\n", analysis.HPAConfig.YAMLConfig)
				fmt.Println()
			}
		}

		// KEDA Recommendations
		if analysis.KEDAConfig != nil {
			green := color.New(color.FgGreen, color.Bold)
			green.Println("🚀 KEDA RECOMMENDATIONS")
			fmt.Println(strings.Repeat("=", 40))
			fmt.Printf("  Min/Max Replicas: %d/%d\n", analysis.KEDAConfig.MinReplicas, analysis.KEDAConfig.MaxReplicas)
			fmt.Printf("  Polling Interval: %ds\n", analysis.KEDAConfig.PollingInterval)
			fmt.Printf("  Cooldown Period: %ds\n", analysis.KEDAConfig.CooldownPeriod)
			fmt.Printf("  Reasoning: %s\n", analysis.KEDAConfig.Reasoning)
			fmt.Println()

			if len(analysis.KEDAConfig.Scalers) > 0 {
				fmt.Println("  Scalers:")
				for _, scaler := range analysis.KEDAConfig.Scalers {
					fmt.Printf("    - %s: %s (threshold: %s)\n", scaler.Type, scaler.Name, scaler.Threshold)
				}
				fmt.Println()
			}

			if analysis.KEDAConfig.YAMLConfig != "" {
				fmt.Println("  YAML Configuration:")
				fmt.Printf("```yaml\n%s\n```\n", analysis.KEDAConfig.YAMLConfig)
				fmt.Println()
			}
		}

		// General recommendations
		if len(analysis.Recommendations) > 0 {
			cyan.Println("💡 RECOMMENDATIONS")
			fmt.Println(strings.Repeat("=", 40))
			for _, rec := range analysis.Recommendations {
				var colorFunc func(a ...interface{}) string
				var icon string
				switch rec.Priority {
				case "high":
					colorFunc = color.New(color.FgRed, color.Bold).SprintFunc()
					icon = "🚨"
				case "medium":
					colorFunc = color.New(color.FgYellow, color.Bold).SprintFunc()
					icon = "⚠️"
				default:
					colorFunc = color.New(color.FgGreen).SprintFunc()
					icon = "💡"
				}

				fmt.Printf("  %s %s [%s] %s\n", icon, colorFunc("●"), rec.Priority, rec.Title)

				// Format description with proper indentation
				formattedDesc := formatter.FormatMarkdownText(rec.Description)
				lines := strings.Split(formattedDesc, "\n")
				for _, line := range lines {
					if strings.TrimSpace(line) != "" {
						fmt.Printf("    %s\n", line)
					}
				}

				// Remove the "Implementation Commands" section as requested
				fmt.Println()
			}
		}
	}

	fmt.Println()
}

// displayMetricsJSON displays results in JSON format
func displayMetricsJSON(analysis *metrics.AnalysisResult) {
	fmt.Println("JSON output not implemented yet")
}

// displayMetricsYAML displays results in YAML format
func displayMetricsYAML(analysis *metrics.AnalysisResult) {
	fmt.Println("YAML output not implemented yet")
}

func printMetricsHeader(resource string) {
	cyan := color.New(color.FgCyan, color.Bold)
	fmt.Println()
	cyan.Println("📊 Kubernetes AI Metrics Analyzer")
	if resource != "" {
		fmt.Printf("📦 Resource: %s\n", resource)
	}
	fmt.Printf("📍 Namespace: %s\n", metricsNamespace)
	fmt.Printf("📅 Duration: %s\n", duration)

	if metricsAllResources {
		fmt.Println("📊 Scope: all deployments")
	} else {
		fmt.Printf("📊 Resources: %s\n", strings.Join(metricsResources, ", "))
	}

	// Show analysis flags
	var analyses []string
	if analyzeScaling {
		analyses = append(analyses, "scaling")
	}
	if hpaAnalysis {
		analyses = append(analyses, "HPA")
	}
	if kedaAnalysis {
		analyses = append(analyses, "KEDA")
	}
	if len(analyses) > 0 {
		fmt.Printf("🔍 Analysis: %s\n", strings.Join(analyses, ", "))
	}

	fmt.Println()
}
