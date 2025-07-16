package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/helmcode/kubectl-ai/pkg/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PrometheusClient handles communication with Prometheus
type PrometheusClient struct {
	url            string
	client         *http.Client
	portForwardCmd *exec.Cmd
	localPort      string
	isPortForward  bool
}

// PrometheusResponse represents the response from Prometheus API
type PrometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value,omitempty"`
			Values [][]interface{}   `json:"values,omitempty"`
		} `json:"result"`
	} `json:"data"`
	Error     string `json:"error,omitempty"`
	ErrorType string `json:"errorType,omitempty"`
}

// NewPrometheusClient creates a new Prometheus client with auto-detection and port-forward support
func NewPrometheusClient(prometheusURL, prometheusNamespace, kubeconfig string, k8sClient *k8s.Client) (*PrometheusClient, error) {
	var finalURL string
	var portForwardCmd *exec.Cmd
	var localPort string
	var isPortForward bool

	if prometheusURL != "" {
		// Use provided URL
		finalURL = prometheusURL
		green := color.New(color.FgGreen)
		green.Printf("✓ Using provided Prometheus URL: %s\n", prometheusURL)
	} else {
		// Auto-detect Prometheus
		serviceName, serviceNamespace, servicePort, err := detectPrometheusService(k8sClient, prometheusNamespace)
		if err != nil {
			fmt.Printf("❌ Failed to auto-detect Prometheus\n")
			return nil, fmt.Errorf("failed to auto-detect Prometheus: %w", err)
		}
		green := color.New(color.FgGreen)
		green.Printf("✓ Found Prometheus: %s/%s:%d\n", serviceNamespace, serviceName, servicePort)

		// Check if we're running in-cluster or outside
		if isRunningInCluster() {
			// Use cluster-internal URL
			finalURL = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", serviceName, serviceNamespace, servicePort)
			green := color.New(color.FgGreen)
			green.Printf("✓ Running in-cluster, using internal URL\n")
		} else {
			// Set up port-forward for external access
			localPort = "9090" // Use a known port
			green := color.New(color.FgGreen)
			green.Printf("✓ Setting up port-forward %s/%s:%d -> localhost:%s\n",
				serviceNamespace, serviceName, servicePort, localPort)
			portForwardCmd, err = setupPortForward(serviceName, serviceNamespace, servicePort, localPort, kubeconfig)
			if err != nil {
				fmt.Printf("❌ Failed to setup port-forward\n")
				return nil, fmt.Errorf("failed to setup port-forward: %w", err)
			}
			finalURL = fmt.Sprintf("http://localhost:%s", localPort)
			isPortForward = true

			// Wait a bit for port-forward to be ready
			time.Sleep(2 * time.Second)
		}
	}

	// Ensure URL has proper format
	if !strings.HasPrefix(finalURL, "http") {
		finalURL = "http://" + finalURL
	}
	if !strings.HasSuffix(finalURL, "/") {
		finalURL += "/"
	}

	client := &PrometheusClient{
		url:            finalURL,
		client:         &http.Client{Timeout: 30 * time.Second},
		portForwardCmd: portForwardCmd,
		localPort:      localPort,
		isPortForward:  isPortForward,
	}

	// Test connection
	if err := client.testConnection(); err != nil {
		fmt.Printf("❌ Failed to connect to Prometheus at %s\n", finalURL)
		client.Close() // Clean up port-forward if it was created
		return nil, fmt.Errorf("failed to connect to Prometheus at %s: %w", finalURL, err)
	}

	if isPortForward {
		green := color.New(color.FgGreen)
		green.Printf("✓ Port-forward active: %s -> localhost:%s\n", finalURL, localPort)
	} else {
		green := color.New(color.FgGreen)
		green.Printf("✓ Connected to Prometheus: %s\n", finalURL)
	}

	return client, nil
}

// Close cleans up resources, including stopping port-forward if active
func (p *PrometheusClient) Close() error {
	if p.portForwardCmd != nil && p.portForwardCmd.Process != nil {
		return p.portForwardCmd.Process.Kill()
	}
	return nil
}

// isRunningInCluster checks if we're running inside a Kubernetes cluster
func isRunningInCluster() bool {
	// Check for service account token (standard way to detect in-cluster)
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		return true
	}
	return false
}

// detectPrometheusService detects the Prometheus service and returns its details
func detectPrometheusService(k8sClient *k8s.Client, prometheusNamespace string) (string, string, int, error) {
	// Common Prometheus service patterns
	servicePatterns := []string{
		"prometheus-server",
		"prometheus-service",
		"prometheus",
		"kube-prometheus-stack-prometheus",
		"prometheus-kube-prometheus-prometheus",
	}

	// Common Prometheus namespaces
	namespaces := []string{
		"prometheus-system",
		"prometheus",
		"monitoring",
		"kube-prometheus-stack",
		"observability",
		"default",
	}

	if prometheusNamespace != "" {
		namespaces = []string{prometheusNamespace}
	}

	for _, ns := range namespaces {
		for _, pattern := range servicePatterns {
			// Try to find the service
			service, err := k8sClient.GetClientset().CoreV1().Services(ns).Get(context.TODO(), pattern, metav1.GetOptions{})
			if err == nil {
				// Found service, return details
				port := 80
				if len(service.Spec.Ports) > 0 {
					port = int(service.Spec.Ports[0].Port)
				}
				return service.Name, ns, port, nil
			}
		}
	}

	return "", "", 0, fmt.Errorf("could not auto-detect Prometheus service in any of the following namespaces: %v", namespaces)
}

// setupPortForward creates a kubectl port-forward to the Prometheus service
func setupPortForward(serviceName, namespace string, servicePort int, localPort, kubeconfig string) (*exec.Cmd, error) {
	// Build kubectl port-forward command
	args := []string{
		"port-forward",
		fmt.Sprintf("service/%s", serviceName),
		fmt.Sprintf("%s:%d", localPort, servicePort),
		"-n", namespace,
	}

	// Add kubeconfig if provided
	if kubeconfig != "" {
		args = append(args, "--kubeconfig", kubeconfig)
	}

	cmd := exec.Command("kubectl", args...)

	// Set up output pipes for debugging
	cmd.Stdout = nil // Suppress output
	cmd.Stderr = nil // Suppress error output

	// Start the port-forward in the background
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start port-forward: %w", err)
	}

	return cmd, nil
}

// detectPrometheus attempts to auto-detect Prometheus in the cluster (legacy function)
func detectPrometheus(k8sClient *k8s.Client, prometheusNamespace string) (string, error) {
	serviceName, serviceNamespace, servicePort, err := detectPrometheusService(k8sClient, prometheusNamespace)
	if err != nil {
		return "", err
	}

	// Return cluster-internal URL
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", serviceName, serviceNamespace, servicePort), nil
}

// testConnection tests the connection to Prometheus
func (p *PrometheusClient) testConnection() error {
	testURL := p.url + "api/v1/query?query=up"

	resp, err := p.client.Get(testURL)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Try to parse response
	var promResp PrometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if promResp.Status != "success" {
		return fmt.Errorf("Prometheus API error: %s", promResp.Error)
	}

	return nil
}

// GetURL returns the Prometheus URL
func (p *PrometheusClient) GetURL() string {
	return p.url
}

// GatherMetrics collects metrics for the specified resources
func (p *PrometheusClient) GatherMetrics(resources []interface{}, duration string) (map[string]*MetricsData, error) {
	metricsData := make(map[string]*MetricsData)

	for _, resource := range resources {
		resourceName, resourceType, namespace, err := extractResourceInfoFromK8sObject(resource)
		if err != nil {
			// Skip resources that can't be processed (like Lists)
			continue
		}

		// Only analyze deployments for now
		if resourceType != "Deployment" {
			continue
		}

		// Collect metrics for this resource
		metrics, err := p.collectResourceMetrics(resourceName, namespace, duration)
		if err != nil {
			return nil, fmt.Errorf("failed to collect metrics for %s/%s: %w", namespace, resourceName, err)
		}

		key := fmt.Sprintf("%s/%s", namespace, resourceName)
		metricsData[key] = &MetricsData{
			ResourceName: resourceName,
			ResourceType: resourceType,
			Namespace:    namespace,
			Metrics:      metrics,
			Duration:     duration,
			Timestamp:    time.Now(),
		}
	}

	return metricsData, nil
}

// extractResourceInfoFromK8sObject extracts resource information from Kubernetes native objects
func extractResourceInfoFromK8sObject(resource interface{}) (string, string, string, error) {
	switch obj := resource.(type) {
	case map[string]interface{}:
		// Handle map format (legacy)
		return extractResourceInfo(obj)
	default:
		// Use reflection to get metadata from Kubernetes objects
		return extractResourceInfoFromObject(obj)
	}
}

// extractResourceInfoFromObject uses reflection to extract metadata from Kubernetes objects
func extractResourceInfoFromObject(obj interface{}) (string, string, string, error) {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return "", "", "", fmt.Errorf("resource is not a struct")
	}

	// Get ObjectMeta field
	objectMetaField := val.FieldByName("ObjectMeta")
	if !objectMetaField.IsValid() {
		return "", "", "", fmt.Errorf("resource missing ObjectMeta")
	}

	// Get name from ObjectMeta
	nameField := objectMetaField.FieldByName("Name")
	if !nameField.IsValid() {
		return "", "", "", fmt.Errorf("resource missing Name")
	}

	// Get namespace from ObjectMeta
	namespaceField := objectMetaField.FieldByName("Namespace")
	if !namespaceField.IsValid() {
		return "", "", "", fmt.Errorf("resource missing Namespace")
	}

	// Get resource type from the struct type
	resourceType := val.Type().Name()

	return nameField.String(), resourceType, namespaceField.String(), nil
}

// extractResourceInfo extracts resource information from the resource map
func extractResourceInfo(resource map[string]interface{}) (string, string, string, error) {
	metadata, ok := resource["metadata"].(map[string]interface{})
	if !ok {
		return "", "", "", fmt.Errorf("missing metadata")
	}

	name, ok := metadata["name"].(string)
	if !ok {
		return "", "", "", fmt.Errorf("missing name")
	}

	namespace, ok := metadata["namespace"].(string)
	if !ok {
		namespace = "default"
	}

	kind, ok := resource["kind"].(string)
	if !ok {
		return "", "", "", fmt.Errorf("missing kind")
	}

	return name, kind, namespace, nil
}

// collectResourceMetrics collects metrics for a specific resource
func (p *PrometheusClient) collectResourceMetrics(resourceName, namespace, duration string) (map[string]MetricValue, error) {
	metrics := make(map[string]MetricValue)

	// Get time range
	endTime := time.Now()
	startTime, err := parseDuration(duration)
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %w", err)
	}

	// Collect standard metrics
	queries := GetStandardQueries()

	for _, query := range queries {
		// Replace placeholders in query
		finalQuery := strings.ReplaceAll(query.Query, "RESOURCE_NAME", resourceName)
		finalQuery = strings.ReplaceAll(finalQuery, "NAMESPACE", namespace)

		// Execute query
		values, err := p.queryRange(finalQuery, startTime, endTime)
		if err != nil {
			// Log error but continue with other metrics
			continue
		}

		if len(values) == 0 {
			continue
		}

		// If we have very few data points for CPU/Memory, try alternative queries
		if len(values) < 10 && (query.Name == "cpu_utilization" || query.Name == "memory_utilization") {
			// Try alternative query for recent data
			alternativeQuery := ""
			if query.Name == "cpu_utilization" {
				alternativeQuery = `avg(rate(container_cpu_usage_seconds_total{pod=~"RESOURCE_NAME.*", namespace="NAMESPACE"}[1m])) * 100`
			} else if query.Name == "memory_utilization" {
				alternativeQuery = `avg(container_memory_usage_bytes{pod=~"RESOURCE_NAME.*", namespace="NAMESPACE"}) / 1024 / 1024`
			}

			if alternativeQuery != "" {
				altQuery := strings.ReplaceAll(alternativeQuery, "RESOURCE_NAME", resourceName)
				altQuery = strings.ReplaceAll(altQuery, "NAMESPACE", namespace)

				// Try with a shorter time range (last 24 hours)
				altStartTime := endTime.Add(-24 * time.Hour)
				altValues, altErr := p.queryRange(altQuery, altStartTime, endTime)

				if altErr == nil && len(altValues) > len(values) {
					values = altValues
				}
			}
		}

		// Calculate statistics
		avg, peak, min, current := calculateStats(values)

		metrics[query.Name] = MetricValue{
			Name:    query.Name,
			Unit:    query.Unit,
			Values:  values,
			Average: avg,
			Peak:    peak,
			Minimum: min,
			Current: current,
			Labels:  make(map[string]string),
		}
	}

	return metrics, nil
}

// queryRange executes a range query against Prometheus
func (p *PrometheusClient) queryRange(query string, startTime, endTime time.Time) ([]TimestampedValue, error) {
	// Build URL
	queryURL := p.url + "api/v1/query_range"

	// Calculate appropriate step based on duration
	duration := endTime.Sub(startTime)
	var step string

	switch {
	case duration <= 6*time.Hour:
		step = "300" // 5 minutes for short periods
	case duration <= 24*time.Hour:
		step = "900" // 15 minutes for 1 day
	case duration <= 7*24*time.Hour:
		step = "3600" // 1 hour for 1 week
	default:
		step = "7200" // 2 hours for longer periods
	}

	params := url.Values{}
	params.Add("query", query)
	params.Add("start", strconv.FormatInt(startTime.Unix(), 10))
	params.Add("end", strconv.FormatInt(endTime.Unix(), 10))
	params.Add("step", step)

	fullURL := queryURL + "?" + params.Encode()

	resp, err := p.client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var promResp PrometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if promResp.Status != "success" {
		return nil, fmt.Errorf("Prometheus API error: %s", promResp.Error)
	}

	// Parse results
	var values []TimestampedValue
	if len(promResp.Data.Result) > 0 {
		result := promResp.Data.Result[0]
		for _, valuePoint := range result.Values {
			if len(valuePoint) >= 2 {
				timestamp, _ := valuePoint[0].(float64)
				valueStr, _ := valuePoint[1].(string)
				value, err := strconv.ParseFloat(valueStr, 64)
				if err != nil {
					continue
				}

				values = append(values, TimestampedValue{
					Timestamp: time.Unix(int64(timestamp), 0),
					Value:     value,
				})
			}
		}
	}

	return values, nil
}

// parseDuration parses duration string to time.Time
func parseDuration(duration string) (time.Time, error) {
	now := time.Now()

	// Parse duration string
	re := regexp.MustCompile(`^(\d+)([hdm])$`)
	matches := re.FindStringSubmatch(duration)
	if len(matches) != 3 {
		return now, fmt.Errorf("invalid duration format: %s", duration)
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return now, fmt.Errorf("invalid duration value: %s", matches[1])
	}

	unit := matches[2]
	switch unit {
	case "h":
		return now.Add(-time.Duration(value) * time.Hour), nil
	case "d":
		return now.Add(-time.Duration(value) * 24 * time.Hour), nil
	case "m":
		return now.Add(-time.Duration(value) * time.Minute), nil
	default:
		return now, fmt.Errorf("invalid duration unit: %s", unit)
	}
}

// calculateStats calculates basic statistics for metric values
func calculateStats(values []TimestampedValue) (avg, peak, min, current float64) {
	if len(values) == 0 {
		return 0, 0, 0, 0
	}

	sum := 0.0
	peak = values[0].Value
	min = values[0].Value
	current = values[len(values)-1].Value

	for _, v := range values {
		sum += v.Value
		if v.Value > peak {
			peak = v.Value
		}
		if v.Value < min {
			min = v.Value
		}
	}

	avg = sum / float64(len(values))
	return avg, peak, min, current
}
