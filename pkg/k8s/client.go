package k8s

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	clientset *kubernetes.Clientset
	dynamic   dynamic.Interface
	discovery discovery.DiscoveryInterface
	config    *rest.Config

	// Cache for discovered resources
	resourceCache map[string]*metav1.APIResource
	gvrCache      map[string]schema.GroupVersionResource
	cacheMutex    sync.RWMutex
}

// NewClient creates a new Kubernetes client with discovery capabilities
// If contextName is not empty, it will be used instead of the current context in kubeconfig.
func NewClient(kubeconfig string, contextName string) (*Client, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig with optional context override
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
		overrides := &clientcmd.ConfigOverrides{}
		if contextName != "" {
			overrides.CurrentContext = contextName
		}
		cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
		config, err = cfg.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create config: %w", err)
		}
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	// Create dynamic client for CRDs
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Create discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	return &Client{
		clientset:     clientset,
		dynamic:       dynamicClient,
		discovery:     discoveryClient,
		config:        config,
		resourceCache: make(map[string]*metav1.APIResource),
		gvrCache:      make(map[string]schema.GroupVersionResource),
	}, nil
}

// discoverResource finds any resource type in the cluster
func (c *Client) discoverResource(resourceType string) (*metav1.APIResource, schema.GroupVersionResource, error) {
	// Check cache first
	c.cacheMutex.RLock()
	if apiResource, ok := c.resourceCache[resourceType]; ok {
		gvr := c.gvrCache[resourceType]
		c.cacheMutex.RUnlock()
		return apiResource, gvr, nil
	}
	c.cacheMutex.RUnlock()

	// Get all available resources
	resourceList, err := c.discovery.ServerPreferredResources()
	if err != nil {
		// Even with errors, we might have partial results
		if resourceList == nil {
			return nil, schema.GroupVersionResource{}, fmt.Errorf("failed to discover resources: %w", err)
		}
	}

	// Search through all API groups
	for _, group := range resourceList {
		gv, err := schema.ParseGroupVersion(group.GroupVersion)
		if err != nil {
			continue
		}

		for _, resource := range group.APIResources {
			// Check if this is our resource (by name, singular, or short names)
			if strings.EqualFold(resource.Name, resourceType) ||
				strings.EqualFold(resource.SingularName, resourceType) ||
				containsStringIgnoreCase(resource.ShortNames, resourceType) {

				// Build GVR
				gvr := schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: resource.Name,
				}

				// Cache the result
				c.cacheMutex.Lock()
				c.resourceCache[resourceType] = &resource
				c.gvrCache[resourceType] = gvr
				c.cacheMutex.Unlock()

				return &resource, gvr, nil
			}
		}
	}

	return nil, schema.GroupVersionResource{}, fmt.Errorf("resource type '%s' not found in cluster", resourceType)
}

// GatherResources collects the specified Kubernetes resources
func (c *Client) GatherResources(namespace string, resources []string, all bool) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	if all {
		// Get all resources in namespace
		if err := c.gatherAllResources(namespace, result); err != nil {
			return nil, err
		}
	} else {
		// Get specific resources
		for _, resource := range resources {
			if err := c.gatherResource(namespace, resource, result); err != nil {
				// Don't fail completely if one resource fails
				fmt.Printf("Warning: failed to gather %s: %v\n", resource, err)
			}
		}
	}

	// Always add events
	events, err := c.getEvents(namespace)
	if err == nil && len(events.Items) > 0 {
		result["events"] = events
	}

	return result, nil
}

func (c *Client) gatherResource(namespace, resource string, result map[string]interface{}) error {
	parts := strings.Split(resource, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid resource format: %s (expected type/name)", resource)
	}

	resourceType := strings.ToLower(parts[0])
	resourceName := parts[1]

	// Try native resources first (for performance)
	if err := c.gatherNativeResource(namespace, resourceType, resourceName, resource, result); err == nil {
		return nil
	}

	// If not a native resource, use discovery
	apiResource, gvr, err := c.discoverResource(resourceType)
	if err != nil {
		return fmt.Errorf("failed to discover resource type %s: %w", resourceType, err)
	}

	// Get the resource using dynamic client
	var obj *unstructured.Unstructured
	if apiResource.Namespaced {
		obj, err = c.dynamic.Resource(gvr).Namespace(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
	} else {
		obj, err = c.dynamic.Resource(gvr).Get(context.TODO(), resourceName, metav1.GetOptions{})
	}

	if err != nil {
		return fmt.Errorf("failed to get %s %s: %w", resourceType, resourceName, err)
	}

	result[resource] = obj

	// If it's a workload, try to get related pods
	if hasSelector(obj) {
		pods, err := c.getPodsForWorkload(namespace, obj)
		if err == nil && len(pods.Items) > 0 {
			result[resource+"_pods"] = pods
		}
	}

	return nil
}

// gatherNativeResource handles built-in Kubernetes resources with typed clients
func (c *Client) gatherNativeResource(namespace, resourceType, resourceName, fullResource string, result map[string]interface{}) error {
	switch resourceType {
	case "deployment", "deploy", "deployments":
		deploy, err := c.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		result[fullResource] = deploy

		// Get related pods
		pods, err := c.getPodsForDeployment(namespace, deploy)
		if err == nil {
			result[fullResource+"_pods"] = pods
		}
		return nil

	case "pod", "pods", "po":
		pod, err := c.clientset.CoreV1().Pods(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		result[fullResource] = pod
		return nil

	case "service", "services", "svc":
		service, err := c.clientset.CoreV1().Services(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		result[fullResource] = service
		return nil

	case "configmap", "configmaps", "cm":
		cm, err := c.clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		result[fullResource] = cm
		return nil

	case "secret", "secrets":
		secret, err := c.clientset.CoreV1().Secrets(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		// Redact secret data for security
		secret.Data = nil
		result[fullResource] = secret
		return nil

	case "statefulset", "statefulsets", "sts":
		sts, err := c.clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		result[fullResource] = sts
		return nil

	case "daemonset", "daemonsets", "ds":
		ds, err := c.clientset.AppsV1().DaemonSets(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		result[fullResource] = ds
		return nil

	case "ingress", "ingresses", "ing":
		ing, err := c.clientset.NetworkingV1().Ingresses(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		result[fullResource] = ing
		return nil

	case "hpa", "horizontalpodautoscaler", "horizontalpodautoscalers":
		hpa, err := c.clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		result[fullResource] = hpa
		return nil

	default:
		return fmt.Errorf("not a native resource")
	}
}

func (c *Client) gatherAllResources(namespace string, result map[string]interface{}) error {
	// Get deployments
	deployments, err := c.clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err == nil && len(deployments.Items) > 0 {
		result["deployments"] = deployments
	}

	// Get pods
	pods, err := c.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err == nil && len(pods.Items) > 0 {
		result["pods"] = pods
	}

	// Get services
	services, err := c.clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err == nil && len(services.Items) > 0 {
		result["services"] = services
	}

	// Get configmaps
	configmaps, err := c.clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err == nil && len(configmaps.Items) > 0 {
		result["configmaps"] = configmaps
	}

	// Get ingresses
	ingresses, err := c.clientset.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err == nil && len(ingresses.Items) > 0 {
		result["ingresses"] = ingresses
	}

	// Get HPAs
	hpas, err := c.clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(context.TODO(), metav1.ListOptions{})
	if err == nil && len(hpas.Items) > 0 {
		result["hpas"] = hpas
	}

	return nil
}

func (c *Client) getPodsForDeployment(namespace string, deployment *appsv1.Deployment) (*corev1.PodList, error) {
	labelSelector := metav1.LabelSelector{MatchLabels: deployment.Spec.Selector.MatchLabels}
	listOptions := metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&labelSelector),
	}
	return c.clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
}

// getPodsForWorkload gets pods for any workload with a label selector
func (c *Client) getPodsForWorkload(namespace string, obj *unstructured.Unstructured) (*corev1.PodList, error) {
	// Extract selector from the unstructured object
	selector, found, err := unstructured.NestedMap(obj.Object, "spec", "selector", "matchLabels")
	if err != nil || !found {
		return nil, fmt.Errorf("no selector found")
	}

	// Convert to label selector
	labels := make(map[string]string)
	for k, v := range selector {
		if str, ok := v.(string); ok {
			labels[k] = str
		}
	}

	labelSelector := metav1.LabelSelector{MatchLabels: labels}
	listOptions := metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&labelSelector),
	}

	return c.clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
}

func (c *Client) getEvents(namespace string) (*corev1.EventList, error) {
	return c.clientset.CoreV1().Events(namespace).List(context.TODO(), metav1.ListOptions{})
}

// Helper functions

func containsStringIgnoreCase(slice []string, str string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, str) {
			return true
		}
	}
	return false
}

func hasSelector(obj *unstructured.Unstructured) bool {
	_, found, _ := unstructured.NestedMap(obj.Object, "spec", "selector")
	return found
}

// GatherMetricsResources collects resources and their metrics for analysis
func (c *Client) GatherMetricsResources(namespace string, resources []string, allDeployments bool, duration string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	if allDeployments {
		// Get all deployments and their metrics
		if err := c.gatherAllDeploymentMetrics(namespace, result, duration); err != nil {
			return nil, err
		}
	} else {
		// Get specific resources and their metrics
		for _, resource := range resources {
			if err := c.gatherResourceMetrics(namespace, resource, result, duration); err != nil {
				// Don't fail completely if one resource fails
				fmt.Printf("Warning: failed to gather metrics for %s: %v\n", resource, err)
			}
		}
	}

	// Always add cluster metrics context
	clusterMetrics, err := c.getClusterMetrics()
	if err == nil {
		result["cluster_metrics"] = clusterMetrics
	}

	// Add events
	events, err := c.getEvents(namespace)
	if err == nil && len(events.Items) > 0 {
		result["events"] = events
	}

	return result, nil
}

func (c *Client) gatherResourceMetrics(namespace, resource string, result map[string]interface{}, duration string) error {
	parts := strings.Split(resource, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid resource format: %s (expected type/name)", resource)
	}

	resourceType := strings.ToLower(parts[0])
	resourceName := parts[1]

	// Get the resource itself
	if err := c.gatherResource(namespace, resource, result); err != nil {
		return err
	}

	// Get metrics for the resource
	metrics, err := c.getResourceMetrics(namespace, resourceType, resourceName, duration)
	if err != nil {
		fmt.Printf("Warning: failed to gather metrics for %s: %v\n", resource, err)
	} else {
		result[resource+"_metrics"] = metrics
	}

	// Get HPA if exists
	hpa, err := c.getHPAForResource(namespace, resourceType, resourceName)
	if err == nil && hpa != nil {
		result[resource+"_hpa"] = hpa
	}

	return nil
}

func (c *Client) gatherAllDeploymentMetrics(namespace string, result map[string]interface{}, duration string) error {
	// Get all deployments
	deployments, err := c.clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	result["deployments"] = deployments

	// Get metrics for each deployment
	for _, deployment := range deployments.Items {
		resourceKey := fmt.Sprintf("deployment/%s", deployment.Name)

		// Get metrics
		metrics, err := c.getResourceMetrics(namespace, "deployment", deployment.Name, duration)
		if err != nil {
			fmt.Printf("Warning: failed to gather metrics for deployment %s: %v\n", deployment.Name, err)
		} else {
			result[resourceKey+"_metrics"] = metrics
		}

		// Get HPA if exists
		hpa, err := c.getHPAForResource(namespace, "deployment", deployment.Name)
		if err == nil && hpa != nil {
			result[resourceKey+"_hpa"] = hpa
		}

		// Get related pods
		pods, err := c.getPodsForDeployment(namespace, &deployment)
		if err == nil && len(pods.Items) > 0 {
			result[resourceKey+"_pods"] = pods
		}
	}

	return nil
}

func (c *Client) getResourceMetrics(namespace, resourceType, resourceName, duration string) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// This is a simplified implementation - in a real scenario, you'd integrate with
	// metrics systems like Prometheus, metrics-server, etc.
	// For now, we'll collect basic resource information that can indicate metrics

	switch resourceType {
	case "deployment", "deploy", "deployments":
		deployment, err := c.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		// Get replica information
		metrics["replicas"] = map[string]interface{}{
			"desired":   deployment.Spec.Replicas,
			"ready":     deployment.Status.ReadyReplicas,
			"available": deployment.Status.AvailableReplicas,
			"updated":   deployment.Status.UpdatedReplicas,
		}

		// Get resource requests/limits
		if len(deployment.Spec.Template.Spec.Containers) > 0 {
			container := deployment.Spec.Template.Spec.Containers[0]
			if container.Resources.Requests != nil || container.Resources.Limits != nil {
				metrics["resources"] = map[string]interface{}{
					"requests": container.Resources.Requests,
					"limits":   container.Resources.Limits,
				}
			}
		}

		// Get recent events for this deployment
		events, err := c.getEventsForResource(namespace, "Deployment", resourceName)
		if err == nil && len(events.Items) > 0 {
			metrics["recent_events"] = events
		}

		// Get pod metrics
		pods, err := c.getPodsForDeployment(namespace, deployment)
		if err == nil {
			podMetrics := make([]map[string]interface{}, 0)
			for _, pod := range pods.Items {
				podMetric := map[string]interface{}{
					"name":     pod.Name,
					"phase":    pod.Status.Phase,
					"restarts": getPodRestartCount(&pod),
					"age":      time.Since(pod.CreationTimestamp.Time).String(),
				}
				podMetrics = append(podMetrics, podMetric)
			}
			metrics["pods"] = podMetrics
		}

	default:
		return nil, fmt.Errorf("metrics not supported for resource type: %s", resourceType)
	}

	metrics["duration"] = duration
	metrics["timestamp"] = time.Now().Format(time.RFC3339)

	return metrics, nil
}

func (c *Client) getHPAForResource(namespace, resourceType, resourceName string) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	hpas, err := c.clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, hpa := range hpas.Items {
		if hpa.Spec.ScaleTargetRef.Name == resourceName &&
			strings.ToLower(hpa.Spec.ScaleTargetRef.Kind) == resourceType {
			return &hpa, nil
		}
	}

	return nil, nil // No HPA found, not an error
}

func (c *Client) getClusterMetrics() (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Get cluster nodes
	nodes, err := c.clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nodeMetrics := make([]map[string]interface{}, 0)
	for _, node := range nodes.Items {
		nodeMetric := map[string]interface{}{
			"name":        node.Name,
			"status":      getNodeStatus(&node),
			"capacity":    node.Status.Capacity,
			"allocatable": node.Status.Allocatable,
		}
		nodeMetrics = append(nodeMetrics, nodeMetric)
	}
	metrics["nodes"] = nodeMetrics

	// Get cluster resource usage summary
	metrics["cluster_summary"] = map[string]interface{}{
		"total_nodes": len(nodes.Items),
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	return metrics, nil
}

func (c *Client) getEventsForResource(namespace, kind, name string) (*corev1.EventList, error) {
	events, err := c.clientset.CoreV1().Events(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Filter events for the specific resource
	filteredEvents := &corev1.EventList{}
	for _, event := range events.Items {
		if event.InvolvedObject.Kind == kind && event.InvolvedObject.Name == name {
			filteredEvents.Items = append(filteredEvents.Items, event)
		}
	}

	return filteredEvents, nil
}

func getPodRestartCount(pod *corev1.Pod) int32 {
	var restarts int32
	for _, containerStatus := range pod.Status.ContainerStatuses {
		restarts += containerStatus.RestartCount
	}
	return restarts
}

func getNodeStatus(node *corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}
