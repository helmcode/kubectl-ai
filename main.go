package main

import (
	"fmt"
	"os"

	"github.com/helmcode/kubectl-ai/cmd"
	"github.com/spf13/cobra"
)

var (
	version = "v0.1.2" // Overwritten at build time
)

func main() {
	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "kubectl-ai",
		Short: "AI-powered Kubernetes debugging",
		Long: `kubectl-ai uses AI to analyze Kubernetes resources and help identify
configuration issues, performance problems, and provide recommendations.`,
		SilenceUsage: true,
	}

	// Disable automatic 'completion' command added by cobra
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Add subcommands
	rootCmd.AddCommand(
		cmd.NewDebugCmd(),
		cmd.NewMetricsCmd(),
		newVersionCmd(),
	)

	return rootCmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("kubectl-ai version %s\n", version)
		},
	}
}
