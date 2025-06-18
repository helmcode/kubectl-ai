package formatter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/helmcode/kubectl-ai/pkg/analyzer"
	"gopkg.in/yaml.v3"
)

// DisplayResults formats and displays the analysis results
func DisplayResults(analysis *analyzer.Analysis, format string) error {
	switch format {
	case "json":
		return displayJSON(analysis)
	case "yaml":
		return displayYAML(analysis)
	case "human":
		fallthrough
	default:
		displayHuman(analysis)
	}
	return nil
}

func displayJSON(analysis *analyzer.Analysis) error {
	output, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

func displayYAML(analysis *analyzer.Analysis) error {
	output, err := yaml.Marshal(analysis)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

func displayHuman(analysis *analyzer.Analysis) {
	// Colors
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	white := color.New(color.FgWhite, color.Bold)

	fmt.Println()
	
	// Root Cause
	red.Println("💡 ROOT CAUSE IDENTIFIED:")
	fmt.Printf("   %s\n\n", analysis.RootCause)

	// Severity
	severityColor := getSeverityColor(analysis.Severity)
	severityColor.Printf("📊 OVERALL SEVERITY: %s\n\n", strings.ToUpper(analysis.Severity))

	// Issues found
	if len(analysis.Issues) > 0 {
		yellow.Println("⚠️  ISSUES FOUND:")
		for i, issue := range analysis.Issues {
			severityIcon := getSeverityIcon(issue.Severity)
			fmt.Printf("   %d. %s %s\n", i+1, severityIcon, issue.Component)
			fmt.Printf("      %s\n", issue.Description)
			if issue.Evidence != "" {
				fmt.Printf("      Evidence: %s\n", color.YellowString(issue.Evidence))
			}
			fmt.Println()
		}
	}

	// Quick Fix
	if analysis.QuickFix != "" {
		green.Println("🚀 QUICK FIX:")
		fmt.Printf("   %s\n\n", color.GreenString(analysis.QuickFix))
	}

	// Suggestions
	if len(analysis.Suggestions) > 0 {
		cyan.Println("💡 SUGGESTIONS:")
		for i, suggestion := range analysis.Suggestions {
			priorityIcon := getPriorityIcon(suggestion.Priority)
			fmt.Printf("   %d. %s %s\n", i+1, priorityIcon, suggestion.Action)
			
			if suggestion.Command != "" {
				fmt.Printf("      Command: %s\n", color.CyanString(suggestion.Command))
			}
			
			if suggestion.Explanation != "" {
				fmt.Printf("      Why: %s\n", suggestion.Explanation)
			}
			fmt.Println()
		}
	}

	// Full Analysis (if verbose)
	if analysis.FullAnalysis != "" {
		white.Println("📄 DETAILED ANALYSIS:")
		fmt.Println(wrapText(analysis.FullAnalysis, 80, "   "))
		fmt.Println()
	}

	// Footer
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("💡 %s\n", color.HiBlackString("Run with -o json or -o yaml for machine-readable output"))
}

func getSeverityColor(severity string) *color.Color {
	switch strings.ToLower(severity) {
	case "critical":
		return color.New(color.FgRed, color.Bold)
	case "high":
		return color.New(color.FgRed)
	case "medium":
		return color.New(color.FgYellow)
	case "low":
		return color.New(color.FgGreen)
	default:
		return color.New(color.FgWhite)
	}
}

func getSeverityIcon(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

func getPriorityIcon(priority string) string {
	switch strings.ToLower(priority) {
	case "high":
		return "⚡"
	case "medium":
		return "🔹"
	case "low":
		return "▫️"
	default:
		return "•"
	}
}

func wrapText(text string, width int, indent string) string {
	var result strings.Builder
	lines := strings.Split(text, "\n")
	
	for _, line := range lines {
		words := strings.Fields(line)
		if len(words) == 0 {
			result.WriteString("\n")
			continue
		}
		
		currentLine := indent
		for _, word := range words {
			if len(currentLine)+len(word)+1 > width {
				result.WriteString(currentLine + "\n")
				currentLine = indent + word
			} else if currentLine == indent {
				currentLine += word
			} else {
				currentLine += " " + word
			}
		}
		
		if currentLine != indent {
			result.WriteString(currentLine + "\n")
		}
	}
	
	return strings.TrimSuffix(result.String(), "\n")
}