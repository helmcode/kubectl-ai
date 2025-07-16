package formatter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/guptarohit/asciigraph"
	"github.com/helmcode/kubectl-ai/pkg/model"
	"gopkg.in/yaml.v3"
)

// DisplayResults formats and displays the analysis results
func DisplayResults(analysis *model.Analysis, format string) error {
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

func displayJSON(analysis *model.Analysis) error {
	output, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

func displayYAML(analysis *model.Analysis) error {
	output, err := yaml.Marshal(analysis)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

func displayHuman(analysis *model.Analysis) {
	// Colors
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	green := color.New(color.FgGreen, color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	white := color.New(color.FgWhite, color.Bold)

	fmt.Println()

	red.Println("💡 ROOT CAUSE IDENTIFIED:")
	fmt.Printf("   %s\n\n", analysis.RootCause)

	severityColor := getSeverityColor(analysis.Severity)
	severityColor.Printf("📊 OVERALL SEVERITY: %s\n\n", strings.ToUpper(analysis.Severity))

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

	if analysis.QuickFix != "" {
		green.Println("🚀 QUICK FIX:")
		fmt.Printf("   %s\n\n", color.GreenString(analysis.QuickFix))
	}

	if len(analysis.Suggestions) > 0 {
		cyan.Println("💡 SUGGESTIONS:")
		for i, suggestion := range analysis.Suggestions {
			priorityIcon := getPriorityIcon(suggestion.Priority)
			fmt.Printf("   %d. %s %s\n", i+1, priorityIcon, suggestion.Action)

			if suggestion.Command != "" {
				fmt.Printf("      Command: %s\n", color.CyanString(suggestion.Command))
			}

			if suggestion.Explanation != "" {
				fmt.Println(wrapText("Why: "+sanitizeText(suggestion.Explanation), 80, "      "))
			}
			fmt.Println()
		}
	}

	if analysis.FullAnalysis != "" {
		white.Println("📄 DETAILED ANALYSIS:")
		fmt.Println(wrapText(sanitizeText(analysis.FullAnalysis), 80, "   "))
		fmt.Println()
	}
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

// sanitizeText removes markdown code fences to keep output clean
func sanitizeText(text string) string {
	// Remove ```json, ```yaml, ``` and matching closing fences
	re := regexp.MustCompile("```[a-zA-Z]*\n|```")
	return re.ReplaceAllString(text, "")
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

// FormatMarkdownText formats markdown text for better display in terminal
func FormatMarkdownText(text string) string {
	// Colors for different elements
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	green := color.New(color.FgGreen, color.Bold)

	// First sanitize the text to remove code fences
	cleanText := sanitizeText(text)

	lines := strings.Split(cleanText, "\n")
	var result strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			result.WriteString("\n")
			continue
		}

		// Handle headers
		if strings.HasPrefix(line, "### ") {
			result.WriteString(green.Sprintf("   %s\n", strings.TrimPrefix(line, "### ")))
		} else if strings.HasPrefix(line, "## ") {
			result.WriteString(cyan.Sprintf("  %s\n", strings.TrimPrefix(line, "## ")))
		} else if strings.HasPrefix(line, "# ") {
			result.WriteString(yellow.Sprintf(" %s\n", strings.TrimPrefix(line, "# ")))
		} else {
			// Handle bold text **text**
			re := regexp.MustCompile(`\*\*(.*?)\*\*`)
			line = re.ReplaceAllStringFunc(line, func(match string) string {
				content := strings.Trim(match, "*")
				return bold.Sprint(content)
			})

			// Add proper indentation with wrapping
			if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
				result.WriteString(wrapText(line, 80, "     "))
			} else {
				result.WriteString(wrapText(line, 80, "   "))
			}
			result.WriteString("\n")
		}
	}

	return result.String()
}

// CreateEnhancedLineChart creates a detailed line chart with timestamps and statistics
func CreateEnhancedLineChart(values []float64, timestamps []time.Time, title string, unit string, duration string) string {
	if len(values) == 0 {
		return ""
	}

	var result strings.Builder

	// Title with enhanced styling
	cyan := color.New(color.FgCyan, color.Bold)
	result.WriteString(cyan.Sprintf("📈 %s\n", title))
	result.WriteString(color.HiBlackString("Duration: %s\n", duration))
	result.WriteString(strings.Repeat("─", 60) + "\n")

	// Create the graph using asciigraph
	graph := asciigraph.Plot(values, asciigraph.Height(12), asciigraph.Width(60), asciigraph.Caption(fmt.Sprintf("%s Usage", title)))

	// Add colors to the graph lines
	lines := strings.Split(graph, "\n")
	for _, line := range lines {
		if strings.Contains(line, "│") || strings.Contains(line, "┤") {
			result.WriteString(color.BlueString(line) + "\n")
		} else if strings.Contains(line, "─") || strings.Contains(line, "└") {
			result.WriteString(color.HiBlackString(line) + "\n")
		} else {
			result.WriteString(line + "\n")
		}
	}

	// Add enhanced X-axis with timeline
	if len(timestamps) > 0 {
		result.WriteString("\n")
		result.WriteString(createEnhancedXAxis(timestamps, duration, 60))
	}

	// Calculate and display statistics
	min := values[0]
	max := values[0]
	sum := 0.0

	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}

	avg := sum / float64(len(values))

	result.WriteString("\n")
	result.WriteString(color.HiBlackString("Statistics:\n"))
	result.WriteString(fmt.Sprintf("  Average: %s\n", color.YellowString("%.2f%s", avg, unit)))
	result.WriteString(fmt.Sprintf("  Minimum: %s\n", color.GreenString("%.2f%s", min, unit)))
	result.WriteString(fmt.Sprintf("  Maximum: %s\n", color.RedString("%.2f%s", max, unit)))
	result.WriteString("\n")

	return result.String()
}

// CreateReplicaBarChart creates a bar chart for replica scaling events
func CreateReplicaBarChart(replicas []int, timestamps []time.Time, title string) string {
	if len(replicas) == 0 {
		return ""
	}

	var result strings.Builder

	// Title
	green := color.New(color.FgGreen, color.Bold)
	result.WriteString(green.Sprintf("� %s\n", title))
	result.WriteString(strings.Repeat("─", 60) + "\n")

	// Convert replicas to float64 for asciigraph
	values := make([]float64, len(replicas))
	for i, r := range replicas {
		values[i] = float64(r)
	}

	// Create bar chart style visualization
	if len(values) > 0 {
		graph := asciigraph.Plot(values,
			asciigraph.Height(8),
			asciigraph.Width(60),
			asciigraph.Caption("Replica Count Over Time"))

		// Color the graph
		lines := strings.Split(graph, "\n")
		for _, line := range lines {
			if strings.Contains(line, "│") || strings.Contains(line, "┤") {
				result.WriteString(color.GreenString(line) + "\n")
			} else if strings.Contains(line, "─") || strings.Contains(line, "└") {
				result.WriteString(color.HiBlackString(line) + "\n")
			} else {
				result.WriteString(line + "\n")
			}
		}
	}

	// Add enhanced X-axis for replica chart
	if len(timestamps) > 0 {
		result.WriteString("\n")
		result.WriteString(createEnhancedXAxis(timestamps, "replica", 60))
	}

	// Scaling events analysis
	scaleUps := 0
	scaleDowns := 0
	stable := 0

	for i := 1; i < len(replicas); i++ {
		if replicas[i] > replicas[i-1] {
			scaleUps++
		} else if replicas[i] < replicas[i-1] {
			scaleDowns++
		} else {
			stable++
		}
	}

	result.WriteString("\n")
	result.WriteString(color.HiBlackString("Scaling Events:\n"))
	result.WriteString(fmt.Sprintf("  Scale Ups: %s\n", color.GreenString("%d", scaleUps)))
	result.WriteString(fmt.Sprintf("  Scale Downs: %s\n", color.RedString("%d", scaleDowns)))
	result.WriteString(fmt.Sprintf("  Stable Periods: %s\n", color.BlueString("%d data points", stable)))
	result.WriteString(fmt.Sprintf("  Total Changes: %s\n", color.YellowString("%d", scaleUps+scaleDowns)))

	if len(replicas) > 0 {
		result.WriteString(fmt.Sprintf("  Current Replicas: %s\n", color.YellowString("%d", replicas[len(replicas)-1])))
	}

	result.WriteString("\n")

	return result.String()
}

// CreateMetricsSummaryDisplay creates a clean summary display for metrics
func CreateMetricsSummaryDisplay(cpuValues []float64, memoryValues []float64, cpuTimestamps []time.Time, memoryTimestamps []time.Time, duration string) string {
	var result strings.Builder

	// Header
	cyan := color.New(color.FgCyan, color.Bold)
	result.WriteString(cyan.Sprintf("� METRICS SUMMARY\n"))
	result.WriteString(strings.Repeat("=", 60) + "\n\n")

	// CPU Metrics
	if len(cpuValues) > 0 {
		result.WriteString(CreateEnhancedLineChart(cpuValues, cpuTimestamps, "CPU Usage", "%", duration))
	}

	// Memory Metrics
	if len(memoryValues) > 0 {
		result.WriteString(CreateEnhancedLineChart(memoryValues, memoryTimestamps, "Memory Usage", "MB", duration))
	}

	return result.String()
}

// createXAxisWithTimestamps creates an X-axis with timeline markers
func createXAxisWithTimestamps(timestamps []time.Time, duration string, width int) string {
	if len(timestamps) == 0 {
		return ""
	}

	var result strings.Builder

	// Create the X-axis baseline
	result.WriteString(color.HiBlackString("Time: "))

	// Format based on duration
	var startFormat, middleFormat, endFormat string
	if duration == "1h" || duration == "6h" {
		// For short durations, show hour and minute
		startFormat = timestamps[0].Format("15:04")
		endFormat = timestamps[len(timestamps)-1].Format("15:04")
		if len(timestamps) > 2 {
			middleFormat = timestamps[len(timestamps)/2].Format("15:04")
		}
	} else {
		// For longer durations, show date and time
		startFormat = timestamps[0].Format("Jan 2 15:04")
		endFormat = timestamps[len(timestamps)-1].Format("Jan 2 15:04")
		if len(timestamps) > 2 {
			middleFormat = timestamps[len(timestamps)/2].Format("Jan 2 15:04")
		}
	}

	// Calculate spacing for the timeline
	timelineWidth := width - 6 // Account for "Time: " prefix
	startPos := 0
	endPos := timelineWidth - len(endFormat)

	// Build the timeline string
	timeline := make([]rune, timelineWidth)
	for i := range timeline {
		timeline[i] = ' '
	}

	// Place start time
	copy(timeline[startPos:], []rune(startFormat))

	// Place middle time if available
	if middleFormat != "" && len(timestamps) > 2 {
		midPos := (timelineWidth - len(middleFormat)) / 2
		if midPos > len(startFormat) && midPos+len(middleFormat) < endPos {
			copy(timeline[midPos:], []rune(middleFormat))
		}
	}

	// Place end time
	copy(timeline[endPos:], []rune(endFormat))

	result.WriteString(color.HiBlackString(string(timeline)))
	result.WriteString("\n")

	return result.String()
}

// createEnhancedXAxis creates an enhanced X-axis with timeline markers and proper date/time formatting
func createEnhancedXAxis(timestamps []time.Time, duration string, width int) string {
	if len(timestamps) == 0 {
		return ""
	}

	var result strings.Builder

	// Create X-axis line
	result.WriteString(strings.Repeat(" ", 8)) // Align with Y-axis
	result.WriteString(color.HiBlackString("└"))
	result.WriteString(color.HiBlackString(strings.Repeat("─", width-10)))
	result.WriteString("\n")

	// Create timeline markers
	result.WriteString(strings.Repeat(" ", 8)) // Align with Y-axis

	// Format timestamps based on duration
	var timeFormat string
	if duration == "1h" || duration == "6h" {
		timeFormat = "15:04"
	} else if duration == "24h" {
		timeFormat = "Jan 2 15:04"
	} else if duration == "replica" {
		// For replica charts, use a more compact format
		timeFormat = "Jan 2 15:04"
	} else {
		// For 7d, 30d etc., show date more prominently
		timeFormat = "Jan 2 15:04"
	}

	startTime := timestamps[0].Format(timeFormat)
	endTime := timestamps[len(timestamps)-1].Format(timeFormat)

	// Calculate positions for start, middle, and end
	availableWidth := width - 10
	startPos := 0
	endPos := availableWidth - len(endTime)

	// Create timeline string
	timeline := make([]rune, availableWidth)
	for i := range timeline {
		timeline[i] = ' '
	}

	// Place start time
	copy(timeline[startPos:], []rune(startTime))

	// Place middle time if we have enough data points
	if len(timestamps) > 4 {
		midIndex := len(timestamps) / 2
		middleTime := timestamps[midIndex].Format(timeFormat)
		midPos := (availableWidth - len(middleTime)) / 2

		// Make sure middle doesn't overlap with start or end
		if midPos > len(startTime)+2 && midPos+len(middleTime)+2 < endPos {
			copy(timeline[midPos:], []rune(middleTime))
		}
	}

	// Place end time
	if endPos > 0 {
		copy(timeline[endPos:], []rune(endTime))
	}

	result.WriteString(color.HiBlackString(string(timeline)))
	result.WriteString("\n")

	// Add time label
	result.WriteString(strings.Repeat(" ", 8))
	result.WriteString(color.HiBlackString("Time"))
	result.WriteString("\n")

	return result.String()
}
