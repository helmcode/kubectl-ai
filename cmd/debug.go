package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/helmcode/kubectl-ai/pkg/analyzer"
	"github.com/helmcode/kubectl-ai/pkg/formatter"
	"github.com/helmcode/kubectl-ai/pkg/k8s"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

var (
	kubeconfig string
	namespace  string
	resources  []string
	allResources bool
	outputFormat string
	verbose    bool
)

func NewDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug PROBLEM",
		Short: "Debug Kubernetes resources with AI assistance",
		Long: `Analyze Kubernetes resources using AI to identify issues and provide solutions.

Examples:
  # Debug a specific deployment
  kubectl ai debug "pods are crashing" -r deployment/nginx

  # Debug multiple resources
  kubectl ai debug "secrets not updating" -r deployment/vault -r vaultstaticsecret/db-creds

  # Debug all resources in a namespace
  kubectl ai debug "application not working" -n production --all

  # Get detailed output
  kubectl ai debug "high memory usage" -r deployment/app -v`,
		Args: cobra.ExactArgs(1),
		RunE: runDebug,
	}

	// Flags
	if home := homedir.HomeDir(); home != "" {
		cmd.Flags().StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "Path to kubeconfig file")
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	cmd.Flags().StringSliceVarP(&resources, "resource", "r", []string{}, "Resources to analyze (e.g., deployment/nginx, pod/nginx-xxx)")
	cmd.Flags().BoolVar(&allResources, "all", false, "Analyze all resources in the namespace")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "human", "Output format (human, json, yaml)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	return cmd
}

func runDebug(cmd *cobra.Command, args []string) error {
	problem := args[0]

	// Validate that we have API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	// Validate inputs
	if !allResources && len(resources) == 0 {
		return fmt.Errorf("either specify resources with -r or use --all flag")
	}

	// Show what we're doing
	printHeader(problem)

	// Create spinner for visual feedback
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Suffix = " Connecting to Kubernetes cluster..."
	s.Start()

	// Initialize K8s client
	k8sClient, err := k8s.NewClient(kubeconfig)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}
	s.Stop()
	printSuccess("Connected to Kubernetes cluster")

	s.Suffix = " Gathering Kubernetes resources..."
	s.Start()

	resourcesData, err := k8sClient.GatherResources(namespace, resources, allResources)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to gather resources: %w", err)
	}

	s.Stop()
	printSuccess(fmt.Sprintf("Gathered %d resources", len(resourcesData)))

	s.Suffix = " Analyzing with AI..."
	s.Start()

	aiAnalyzer := analyzer.New(apiKey)
	analysis, err := aiAnalyzer.Analyze(problem, resourcesData)
	if err != nil {
		s.Stop()
		return fmt.Errorf("AI analysis failed: %w", err)
	}

	s.Stop()
	printSuccess("Analysis complete")

	formatter.DisplayResults(analysis, outputFormat)

	return nil
}

func printHeader(problem string) {
	cyan := color.New(color.FgCyan, color.Bold)
	fmt.Println()
	cyan.Println("🔍 Kubernetes AI Debugger")
	fmt.Printf("📝 Problem: %s\n", problem)
	fmt.Printf("📍 Namespace: %s\n", namespace)

	if allResources {
		fmt.Println("📊 Resources: all")
	} else {
		fmt.Printf("📊 Resources: %s\n", strings.Join(resources, ", "))
	}
	fmt.Println()
}

func printSuccess(msg string) {
	green := color.New(color.FgGreen)
	green.Printf("✓ %s\n", msg)
}

func printError(msg string) {
	red := color.New(color.FgRed)
	red.Printf("✗ %s\n", msg)
}