package metrics

import (
	"time"
)

// MetricsData represents collected metrics for a resource
type MetricsData struct {
	ResourceName string                 `json:"resource_name"`
	ResourceType string                 `json:"resource_type"`
	Namespace    string                 `json:"namespace"`
	Metrics      map[string]MetricValue `json:"metrics"`
	Duration     string                 `json:"duration"`
	Timestamp    time.Time              `json:"timestamp"`
}

// MetricValue represents a single metric with its values over time
type MetricValue struct {
	Name    string             `json:"name"`
	Unit    string             `json:"unit"`
	Values  []TimestampedValue `json:"values"`
	Average float64            `json:"average"`
	Peak    float64            `json:"peak"`
	Minimum float64            `json:"minimum"`
	Current float64            `json:"current"`
	Labels  map[string]string  `json:"labels"`
}

// TimestampedValue represents a metric value at a specific time
type TimestampedValue struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// AnalysisRequest represents the parameters for metrics analysis
type AnalysisRequest struct {
	Resources      []interface{}           `json:"resources"`
	MetricsData    map[string]*MetricsData `json:"metrics_data"`
	Duration       string                  `json:"duration"`
	AnalyzeScaling bool                    `json:"analyze_scaling"`
	CompareScaling bool                    `json:"compare_scaling"`
	HPAAnalysis    bool                    `json:"hpa_analysis"`
	KEDAAnalysis   bool                    `json:"keda_analysis"`
	Namespace      string                  `json:"namespace"`
}

// AnalysisResult represents the result of metrics analysis
type AnalysisResult struct {
	ResourceName    string                   `json:"resource_name"`
	ResourceType    string                   `json:"resource_type"`
	Namespace       string                   `json:"namespace"`
	Duration        string                   `json:"duration"`
	Summary         string                   `json:"summary"`
	Recommendations []Recommendation         `json:"recommendations"`
	HPAConfig       *HPARecommendation       `json:"hpa_config,omitempty"`
	KEDAConfig      *KEDARecommendation      `json:"keda_config,omitempty"`
	CurrentConfig   *ScalingConfig           `json:"current_config,omitempty"`
	MetricsSummary  map[string]MetricSummary `json:"metrics_summary"`
	ScalingEvents   []ScalingEvent           `json:"scaling_events"`
	Timestamp       time.Time                `json:"timestamp"`
}

// Recommendation represents a scaling recommendation
type Recommendation struct {
	Type        string `json:"type"`     // "hpa", "keda", "resource", "general"
	Priority    string `json:"priority"` // "high", "medium", "low"
	Title       string `json:"title"`
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	Reasoning   string `json:"reasoning"`
}

// HPARecommendation represents HPA-specific recommendations
type HPARecommendation struct {
	Enabled         bool           `json:"enabled"`
	MinReplicas     int32          `json:"min_replicas"`
	MaxReplicas     int32          `json:"max_replicas"`
	TargetCPU       int32          `json:"target_cpu,omitempty"`
	TargetMemory    int32          `json:"target_memory,omitempty"`
	ScaleUpPolicy   *ScalingPolicy `json:"scale_up_policy,omitempty"`
	ScaleDownPolicy *ScalingPolicy `json:"scale_down_policy,omitempty"`
	YAMLConfig      string         `json:"yaml_config"`
	Reasoning       string         `json:"reasoning"`
}

// KEDARecommendation represents KEDA-specific recommendations
type KEDARecommendation struct {
	Enabled         bool         `json:"enabled"`
	MinReplicas     int32        `json:"min_replicas"`
	MaxReplicas     int32        `json:"max_replicas"`
	PollingInterval int32        `json:"polling_interval"`
	CooldownPeriod  int32        `json:"cooldown_period"`
	Scalers         []KEDAScaler `json:"scalers"`
	YAMLConfig      string       `json:"yaml_config"`
	Reasoning       string       `json:"reasoning"`
}

// KEDAScaler represents a KEDA scaler configuration
type KEDAScaler struct {
	Type      string            `json:"type"`
	Name      string            `json:"name"`
	Metadata  map[string]string `json:"metadata"`
	Threshold string            `json:"threshold"`
	Query     string            `json:"query,omitempty"`
}

// ScalingPolicy represents scaling policies
type ScalingPolicy struct {
	StabilizationWindowSeconds int32  `json:"stabilization_window_seconds"`
	Type                       string `json:"type"`
	Value                      int32  `json:"value"`
	PeriodSeconds              int32  `json:"period_seconds"`
}

// ScalingConfig represents current scaling configuration
type ScalingConfig struct {
	Type         string                 `json:"type"` // "hpa", "keda", "none"
	MinReplicas  int32                  `json:"min_replicas"`
	MaxReplicas  int32                  `json:"max_replicas"`
	CurrentSize  int32                  `json:"current_size"`
	TargetCPU    int32                  `json:"target_cpu,omitempty"`
	TargetMemory int32                  `json:"target_memory,omitempty"`
	Scalers      []string               `json:"scalers,omitempty"`
	RawConfig    map[string]interface{} `json:"raw_config,omitempty"`
}

// MetricSummary represents a summary of a specific metric
type MetricSummary struct {
	Name        string      `json:"name"`
	Unit        string      `json:"unit"`
	Average     float64     `json:"average"`
	Peak        float64     `json:"peak"`
	Minimum     float64     `json:"minimum"`
	Current     float64     `json:"current"`
	Trend       string      `json:"trend"`       // "increasing", "decreasing", "stable"
	Utilization string      `json:"utilization"` // "low", "medium", "high", "critical"
	Values      []float64   `json:"values"`      // Historical values for charts
	Timestamps  []time.Time `json:"timestamps"`  // Timestamps for values
}

// ScalingEvent represents a scaling event
type ScalingEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Replicas  int       `json:"replicas"`
	Reason    string    `json:"reason"`
}

// PrometheusQuery represents a Prometheus query configuration
type PrometheusQuery struct {
	Name        string `json:"name"`
	Query       string `json:"query"`
	Unit        string `json:"unit"`
	Description string `json:"description"`
}

// Standard Prometheus queries for common metrics
var (
	// CPU metrics - improved with better rate and container selector
	CPUUtilizationQuery = PrometheusQuery{
		Name:        "cpu_utilization",
		Query:       `avg(rate(container_cpu_usage_seconds_total{pod=~"RESOURCE_NAME.*", namespace="NAMESPACE", container!="", container!="POD"}[5m])) * 100`,
		Unit:        "percent",
		Description: "CPU utilization percentage",
	}

	CPURequestsQuery = PrometheusQuery{
		Name:        "cpu_requests",
		Query:       `avg(kube_pod_container_resource_requests{pod=~"RESOURCE_NAME.*", namespace="NAMESPACE", resource="cpu"})`,
		Unit:        "cores",
		Description: "CPU requests",
	}

	CPULimitsQuery = PrometheusQuery{
		Name:        "cpu_limits",
		Query:       `avg(kube_pod_container_resource_limits{pod=~"RESOURCE_NAME.*", namespace="NAMESPACE", resource="cpu"})`,
		Unit:        "cores",
		Description: "CPU limits",
	}

	// Memory metrics - improved with better container selector
	MemoryUtilizationQuery = PrometheusQuery{
		Name:        "memory_utilization",
		Query:       `avg(container_memory_usage_bytes{pod=~"RESOURCE_NAME.*", namespace="NAMESPACE", container!="", container!="POD"}) / 1024 / 1024`,
		Unit:        "MB",
		Description: "Memory utilization in MB",
	}

	MemoryRequestsQuery = PrometheusQuery{
		Name:        "memory_requests",
		Query:       `avg(kube_pod_container_resource_requests{pod=~"RESOURCE_NAME.*", namespace="NAMESPACE", resource="memory"}) / 1024 / 1024`,
		Unit:        "MB",
		Description: "Memory requests in MB",
	}

	MemoryLimitsQuery = PrometheusQuery{
		Name:        "memory_limits",
		Query:       `avg(kube_pod_container_resource_limits{pod=~"RESOURCE_NAME.*", namespace="NAMESPACE", resource="memory"}) / 1024 / 1024`,
		Unit:        "MB",
		Description: "Memory limits in MB",
	}

	// Pod metrics
	PodReplicasQuery = PrometheusQuery{
		Name:        "pod_replicas",
		Query:       `kube_deployment_status_replicas{deployment="RESOURCE_NAME", namespace="NAMESPACE"}`,
		Unit:        "count",
		Description: "Number of pod replicas",
	}

	PodAvailableQuery = PrometheusQuery{
		Name:        "pod_available",
		Query:       `kube_deployment_status_replicas_available{deployment="RESOURCE_NAME", namespace="NAMESPACE"}`,
		Unit:        "count",
		Description: "Number of available pod replicas",
	}
)

// GetStandardQueries returns the standard set of Prometheus queries
func GetStandardQueries() []PrometheusQuery {
	return []PrometheusQuery{
		CPUUtilizationQuery,
		CPURequestsQuery,
		CPULimitsQuery,
		MemoryUtilizationQuery,
		MemoryRequestsQuery,
		MemoryLimitsQuery,
		PodReplicasQuery,
		PodAvailableQuery,
	}
}
