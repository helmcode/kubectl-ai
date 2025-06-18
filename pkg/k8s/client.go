package k8s

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	clientset *kubernetes.Clientset
	dynamic   dynamic.Interface
	config    *rest.Config
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfig string) (*Client, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
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

	return &Client{
		clientset: clientset,
		dynamic:   dynamicClient,
		config:    config,
	}, nil
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
				return nil, err
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

	resourceType := parts[0]
	resourceName := parts[1]

	switch resourceType {
	case "deployment", "deploy":
		deploy, err := c.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get deployment %s: %w", resourceName, err)
		}
		result[resource] = deploy
		
		// Get related pods
		pods, err := c.getPodsForDeployment(namespace, deploy)
		if err == nil {
			result[resource+"_pods"] = pods
		}

	case "pod":
		pod, err := c.clientset.CoreV1().Pods(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get pod %s: %w", resourceName, err)
		}
		result[resource] = pod

	case "service", "svc":
		service, err := c.clientset.CoreV1().Services(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get service %s: %w", resourceName, err)
		}
		result[resource] = service

	case "configmap", "cm":
		cm, err := c.clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get configmap %s: %w", resourceName, err)
		}
		result[resource] = cm

	case "secret":
		secret, err := c.clientset.CoreV1().Secrets(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get secret %s: %w", resourceName, err)
		}
		// Redact secret data
		secret.Data = nil
		result[resource] = secret

	case "statefulset", "sts":
		sts, err := c.clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get statefulset %s: %w", resourceName, err)
		}
		result[resource] = sts

	case "daemonset", "ds":
		ds, err := c.clientset.AppsV1().DaemonSets(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get daemonset %s: %w", resourceName, err)
		}
		result[resource] = ds

	case "hpa", "horizontalpodautoscaler":
		hpa, err := c.clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get hpa %s: %w", resourceName, err)
		}
		result[resource] = hpa

	default:
		// Try as CRD
		obj, err := c.getCustomResource(namespace, resourceType, resourceName)
		if err != nil {
			return fmt.Errorf("unknown resource type or failed to get CRD %s: %w", resourceType, err)
		}
		result[resource] = obj
	}

	return nil
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

func (c *Client) getEvents(namespace string) (*corev1.EventList, error) {
	return c.clientset.CoreV1().Events(namespace).List(context.TODO(), metav1.ListOptions{})
}

func (c *Client) getCustomResource(namespace, resourceType, resourceName string) (*unstructured.Unstructured, error) {
	// Common CRDs - add more as needed
	gvr := schema.GroupVersionResource{}
	
	switch resourceType {
	case "vaultstaticsecret":
		gvr = schema.GroupVersionResource{
			Group:    "secrets.hashicorp.com",
			Version:  "v1beta1",
			Resource: "vaultstaticsecrets",
		}
	default:
		return nil, fmt.Errorf("unknown custom resource type: %s", resourceType)
	}

	return c.dynamic.Resource(gvr).Namespace(namespace).Get(context.TODO(), resourceName, metav1.GetOptions{})
}