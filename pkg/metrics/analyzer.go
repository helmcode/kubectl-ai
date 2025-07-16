package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/helmcode/kubectl-ai/pkg/k8s"
	"github.com/helmcode/kubectl-ai/pkg/llm"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Analyzer handles metrics analysis using AI
type Analyzer struct {
	llm        llm.LLM
	prometheus *PrometheusClient
	k8sClient  *k8s.Client
}

// NewAnalyzer creates a new metrics analyzer
func NewAnalyzer(llmClient llm.LLM, prometheusClient *PrometheusClient, k8sClient *k8s.Client) *Analyzer {
	return &Analyzer{
		llm:        llmClient,
		prometheus: prometheusClient,
		k8sClient:  k8sClient,
	}
}

// AnalyzeMetrics performs AI-powered metrics analysis
func (a *Analyzer) AnalyzeMetrics(request *AnalysisRequest) (*AnalysisResult, error) {
	results := make(map[string]*AnalysisResult)

	// Analyze each resource
	for key, metricsData := range request.MetricsData {
		result, err := a.analyzeResource(metricsData, request)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze resource %s: %w", key, err)
		}
		results[key] = result
	}

	// For now, return the first result. In the future, we might want to aggregate results
	for _, result := range results {
		return result, nil
	}

	return &AnalysisResult{
		Summary:   "No resources found to analyze",
		Timestamp: time.Now(),
	}, nil
}

// analyzeResource analyzes a single resource
func (a *Analyzer) analyzeResource(metricsData *MetricsData, request *AnalysisRequest) (*AnalysisResult, error) {
	result := &AnalysisResult{
		ResourceName:    metricsData.ResourceName,
		ResourceType:    metricsData.ResourceType,
		Namespace:       metricsData.Namespace,
		Duration:        metricsData.Duration,
		Recommendations: []Recommendation{},
		MetricsSummary:  make(map[string]MetricSummary),
		Timestamp:       time.Now(),
	}

	// Process metrics summary
	for name, metric := range metricsData.Metrics {
		// Convert TimestampedValue to separate arrays
		values := make([]float64, len(metric.Values))
		timestamps := make([]time.Time, len(metric.Values))

		for i, tv := range metric.Values {
			values[i] = tv.Value
			timestamps[i] = tv.Timestamp
		}

		result.MetricsSummary[name] = MetricSummary{
			Name:        metric.Name,
			Unit:        metric.Unit,
			Average:     metric.Average,
			Peak:        metric.Peak,
			Minimum:     metric.Minimum,
			Current:     metric.Current,
			Trend:       calculateTrend(metric.Values),
			Utilization: calculateUtilization(metric.Average, metric.Peak),
			Values:      values,     // Add historical values
			Timestamps:  timestamps, // Add timestamps
		}
	}

	// Extract scaling events from pod_replicas metric
	if podReplicasMetric, exists := metricsData.Metrics["pod_replicas"]; exists {
		result.ScalingEvents = make([]ScalingEvent, len(podReplicasMetric.Values))
		for i, tv := range podReplicasMetric.Values {
			result.ScalingEvents[i] = ScalingEvent{
				Timestamp: tv.Timestamp,
				Replicas:  int(tv.Value),
				Reason:    "pod_scaling", // This could be enhanced to get actual reason from K8s events
			}
		}
	}

	// Get current scaling configuration
	currentConfig, err := a.getCurrentScalingConfig(metricsData.ResourceName, metricsData.Namespace)
	if err != nil {
		// Not an error, just means no scaling is configured
		currentConfig = &ScalingConfig{
			Type:        "none",
			MinReplicas: 1,
			MaxReplicas: 1,
			CurrentSize: 1,
		}
	}
	result.CurrentConfig = currentConfig

	// Perform AI analysis
	if request.AnalyzeScaling || request.HPAAnalysis || request.KEDAAnalysis {
		aiAnalysis, err := a.performAIAnalysis(metricsData, request, currentConfig)
		if err != nil {
			return nil, fmt.Errorf("AI analysis failed: %w", err)
		}
		result.Summary = aiAnalysis
	}

	// Generate HPA recommendations if requested
	if request.HPAAnalysis {
		hpaRecommendation, err := a.generateHPARecommendation(metricsData, currentConfig)
		if err != nil {
			return nil, fmt.Errorf("HPA analysis failed: %w", err)
		}
		result.HPAConfig = hpaRecommendation
	}

	// Generate KEDA recommendations if requested
	if request.KEDAAnalysis {
		kedaRecommendation, err := a.generateKEDARecommendation(metricsData, currentConfig)
		if err != nil {
			return nil, fmt.Errorf("KEDA analysis failed: %w", err)
		}
		result.KEDAConfig = kedaRecommendation
	}

	return result, nil
}

// performAIAnalysis uses AI to analyze metrics and provide recommendations
func (a *Analyzer) performAIAnalysis(metricsData *MetricsData, request *AnalysisRequest, currentConfig *ScalingConfig) (string, error) {
	prompt := a.buildAnalysisPrompt(metricsData, request, currentConfig)

	response, err := a.llm.Chat(prompt)
	if err != nil {
		return "", fmt.Errorf("AI analysis failed: %w", err)
	}

	return response, nil
}

// buildAnalysisPrompt creates the prompt for AI analysis
func (a *Analyzer) buildAnalysisPrompt(metricsData *MetricsData, request *AnalysisRequest, currentConfig *ScalingConfig) string {
	var prompt strings.Builder

	prompt.WriteString("You are a Kubernetes expert analyzing metrics for scaling recommendations.\n\n")
	prompt.WriteString(fmt.Sprintf("Resource: %s/%s (type: %s)\n", metricsData.Namespace, metricsData.ResourceName, metricsData.ResourceType))
	prompt.WriteString(fmt.Sprintf("Analysis Duration: %s\n\n", metricsData.Duration))

	// Add metrics data
	prompt.WriteString("METRICS DATA:\n")
	for name, metric := range metricsData.Metrics {
		prompt.WriteString(fmt.Sprintf("- %s (%s): avg=%.2f, peak=%.2f, min=%.2f, current=%.2f\n",
			name, metric.Unit, metric.Average, metric.Peak, metric.Minimum, metric.Current))
	}
	prompt.WriteString("\n")

	// Add current scaling configuration
	prompt.WriteString("CURRENT SCALING CONFIGURATION:\n")
	prompt.WriteString(fmt.Sprintf("- Type: %s\n", currentConfig.Type))
	prompt.WriteString(fmt.Sprintf("- Min Replicas: %d\n", currentConfig.MinReplicas))
	prompt.WriteString(fmt.Sprintf("- Max Replicas: %d\n", currentConfig.MaxReplicas))
	prompt.WriteString(fmt.Sprintf("- Current Size: %d\n", currentConfig.CurrentSize))
	if currentConfig.TargetCPU > 0 {
		prompt.WriteString(fmt.Sprintf("- Target CPU: %d%%\n", currentConfig.TargetCPU))
	}
	if currentConfig.TargetMemory > 0 {
		prompt.WriteString(fmt.Sprintf("- Target Memory: %d%%\n", currentConfig.TargetMemory))
	}
	prompt.WriteString("\n")

	// Add analysis requirements
	prompt.WriteString("ANALYSIS REQUIREMENTS:\n")
	if request.AnalyzeScaling {
		prompt.WriteString("- Provide general scaling analysis and recommendations\n")
	}
	if request.HPAAnalysis {
		prompt.WriteString("- Provide HPA (Horizontal Pod Autoscaler) specific recommendations\n")
	}
	if request.KEDAAnalysis {
		prompt.WriteString("- Provide KEDA scaling recommendations\n")
	}
	if request.CompareScaling {
		prompt.WriteString("- Compare current configuration with optimal recommendations\n")
	}
	prompt.WriteString("\n")

	// Add specific instructions
	prompt.WriteString("Please provide:\n")
	prompt.WriteString("1. Analysis of current resource utilization patterns\n")
	prompt.WriteString("2. Specific scaling recommendations with reasoning\n")
	prompt.WriteString("3. Optimal scaling parameters (min/max replicas, thresholds)\n")
	prompt.WriteString("4. Any potential issues or improvements\n")
	prompt.WriteString("5. Concrete kubectl commands for implementation\n\n")

	prompt.WriteString("Focus on practical, actionable recommendations based on the actual metrics data.")

	return prompt.String()
}

// getCurrentScalingConfig retrieves current scaling configuration
func (a *Analyzer) getCurrentScalingConfig(resourceName, namespace string) (*ScalingConfig, error) {
	// Check for HPA first
	hpaConfig, err := a.getHPAConfig(resourceName, namespace)
	if err == nil {
		return hpaConfig, nil
	}

	// Check for KEDA ScaledObject
	kedaConfig, err := a.getKEDAConfig(resourceName, namespace)
	if err == nil {
		return kedaConfig, nil
	}

	// No scaling configured
	return nil, fmt.Errorf("no scaling configuration found")
}

// getHPAConfig retrieves HPA configuration
func (a *Analyzer) getHPAConfig(resourceName, namespace string) (*ScalingConfig, error) {
	// Try v2 HPA first
	hpaV2, err := a.k8sClient.GetClientset().AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	if err == nil {
		config := &ScalingConfig{
			Type:        "hpa",
			MinReplicas: *hpaV2.Spec.MinReplicas,
			MaxReplicas: hpaV2.Spec.MaxReplicas,
			CurrentSize: hpaV2.Status.CurrentReplicas,
		}

		// Extract CPU and memory targets
		for _, metric := range hpaV2.Spec.Metrics {
			if metric.Type == autoscalingv2.ResourceMetricSourceType {
				if metric.Resource.Name == "cpu" && metric.Resource.Target.AverageUtilization != nil {
					config.TargetCPU = *metric.Resource.Target.AverageUtilization
				}
				if metric.Resource.Name == "memory" && metric.Resource.Target.AverageUtilization != nil {
					config.TargetMemory = *metric.Resource.Target.AverageUtilization
				}
			}
		}

		return config, nil
	}

	// Try v1 HPA
	hpaV1, err := a.k8sClient.GetClientset().AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	if err == nil {
		config := &ScalingConfig{
			Type:        "hpa",
			MinReplicas: *hpaV1.Spec.MinReplicas,
			MaxReplicas: hpaV1.Spec.MaxReplicas,
			CurrentSize: hpaV1.Status.CurrentReplicas,
		}

		if hpaV1.Spec.TargetCPUUtilizationPercentage != nil {
			config.TargetCPU = *hpaV1.Spec.TargetCPUUtilizationPercentage
		}

		return config, nil
	}

	return nil, fmt.Errorf("HPA not found")
}

// getKEDAConfig retrieves KEDA ScaledObject configuration
func (a *Analyzer) getKEDAConfig(resourceName, namespace string) (*ScalingConfig, error) {
	// For now, we'll implement a simplified version
	// In a real implementation, we would query the KEDA API properly

	// This is a placeholder - in a real implementation we would:
	// 1. Use the dynamic client to query KEDA CRDs
	// 2. Parse the ScaledObject configuration
	// 3. Extract scaling parameters

	return nil, fmt.Errorf("KEDA ScaledObject not found")
}

// generateHPARecommendation generates HPA recommendations
func (a *Analyzer) generateHPARecommendation(metricsData *MetricsData, currentConfig *ScalingConfig) (*HPARecommendation, error) {
	recommendation := &HPARecommendation{
		Enabled:     true,
		MinReplicas: 2,
		MaxReplicas: 10,
	}

	// Analyze CPU metrics
	if cpuMetric, ok := metricsData.Metrics["cpu_utilization"]; ok {
		if cpuMetric.Average > 0 {
			// Recommend target CPU based on observed patterns
			if cpuMetric.Peak > 80 {
				recommendation.TargetCPU = 70
			} else if cpuMetric.Peak > 60 {
				recommendation.TargetCPU = 60
			} else {
				recommendation.TargetCPU = 50
			}
		}
	}

	// Analyze memory metrics
	if memMetric, ok := metricsData.Metrics["memory_utilization"]; ok {
		if memMetric.Average > 0 {
			// Recommend target memory based on observed patterns
			if memMetric.Peak > 80 {
				recommendation.TargetMemory = 70
			} else if memMetric.Peak > 60 {
				recommendation.TargetMemory = 60
			} else {
				recommendation.TargetMemory = 50
			}
		}
	}

	// Generate YAML configuration
	recommendation.YAMLConfig = a.generateHPAYAML(metricsData.ResourceName, metricsData.Namespace, recommendation)
	recommendation.Reasoning = "Based on observed CPU and memory patterns over the specified duration"

	return recommendation, nil
}

// generateKEDARecommendation generates KEDA recommendations
func (a *Analyzer) generateKEDARecommendation(metricsData *MetricsData, currentConfig *ScalingConfig) (*KEDARecommendation, error) {
	recommendation := &KEDARecommendation{
		Enabled:         true,
		MinReplicas:     0,
		MaxReplicas:     10,
		PollingInterval: 30,
		CooldownPeriod:  300,
		Scalers:         []KEDAScaler{},
	}

	// Add Prometheus scaler based on available metrics
	if _, ok := metricsData.Metrics["cpu_utilization"]; ok {
		scaler := KEDAScaler{
			Type:      "prometheus",
			Name:      "cpu-scaler",
			Threshold: "70",
			Query:     fmt.Sprintf(`rate(container_cpu_usage_seconds_total{pod=~"%s.*", namespace="%s"}[5m]) * 100`, metricsData.ResourceName, metricsData.Namespace),
			Metadata: map[string]string{
				"serverAddress": a.prometheus.GetURL(),
				"threshold":     "70",
				"query":         fmt.Sprintf(`rate(container_cpu_usage_seconds_total{pod=~"%s.*", namespace="%s"}[5m]) * 100`, metricsData.ResourceName, metricsData.Namespace),
			},
		}
		recommendation.Scalers = append(recommendation.Scalers, scaler)
	}

	// Generate YAML configuration
	recommendation.YAMLConfig = a.generateKEDAYAML(metricsData.ResourceName, metricsData.Namespace, recommendation)
	recommendation.Reasoning = "KEDA allows more flexible scaling with custom metrics from Prometheus"

	return recommendation, nil
}

// generateHPAYAML generates HPA YAML configuration
func (a *Analyzer) generateHPAYAML(resourceName, namespace string, config *HPARecommendation) string {
	yaml := fmt.Sprintf(`apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: %s
  namespace: %s
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: %s
  minReplicas: %d
  maxReplicas: %d
  metrics:`, resourceName, namespace, resourceName, config.MinReplicas, config.MaxReplicas)

	if config.TargetCPU > 0 {
		yaml += fmt.Sprintf(`
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: %d`, config.TargetCPU)
	}

	if config.TargetMemory > 0 {
		yaml += fmt.Sprintf(`
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: %d`, config.TargetMemory)
	}

	return yaml
}

// generateKEDAYAML generates KEDA YAML configuration
func (a *Analyzer) generateKEDAYAML(resourceName, namespace string, config *KEDARecommendation) string {
	yaml := fmt.Sprintf(`apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: %s
  namespace: %s
spec:
  scaleTargetRef:
    name: %s
  minReplicaCount: %d
  maxReplicaCount: %d
  pollingInterval: %d
  cooldownPeriod: %d
  triggers:`, resourceName, namespace, resourceName, config.MinReplicas, config.MaxReplicas, config.PollingInterval, config.CooldownPeriod)

	for _, scaler := range config.Scalers {
		yaml += fmt.Sprintf(`
  - type: %s
    metadata:
      serverAddress: %s
      threshold: '%s'
      query: %s`, scaler.Type, scaler.Metadata["serverAddress"], scaler.Threshold, scaler.Query)
	}

	return yaml
}

// calculateTrend calculates the trend of metric values
func calculateTrend(values []TimestampedValue) string {
	if len(values) < 2 {
		return "stable"
	}

	first := values[0].Value
	last := values[len(values)-1].Value
	diff := last - first

	if diff > first*0.1 {
		return "increasing"
	} else if diff < -first*0.1 {
		return "decreasing"
	}
	return "stable"
}

// calculateUtilization calculates utilization level
func calculateUtilization(average, peak float64) string {
	if peak > 90 {
		return "critical"
	} else if peak > 70 {
		return "high"
	} else if peak > 30 {
		return "medium"
	}
	return "low"
}
