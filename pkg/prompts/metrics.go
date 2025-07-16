package prompts

import (
	"encoding/json"
	"fmt"
	"strings"
)

func BuildMetricsPrompt(resources map[string]interface{}, duration string, compareScaling, hpaAnalysis, kedaAnalysis bool) (string, error) {
	resourcesJSON, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal resources: %w", err)
	}

	// Build analysis options
	var analysisOptions []string
	if compareScaling {
		analysisOptions = append(analysisOptions, "scaling configuration comparison")
	}
	if hpaAnalysis {
		analysisOptions = append(analysisOptions, "HPA (Horizontal Pod Autoscaler) analysis")
	}
	if kedaAnalysis {
		analysisOptions = append(analysisOptions, "KEDA (Kubernetes Event-Driven Autoscaling) analysis")
	}

	analysisOptionsText := ""
	if len(analysisOptions) > 0 {
		analysisOptionsText = fmt.Sprintf("\n\nAdditional Analysis Required:\n- %s", strings.Join(analysisOptions, "\n- "))
	}

	return fmt.Sprintf(`You are a Kubernetes performance expert analyzing resource metrics and scaling behavior.

Duration: %s
%s

Kubernetes Resources and Metrics:
%s

Please analyze these Kubernetes resources and their metrics to provide:
1. Performance analysis based on current metrics
2. Scaling recommendations and optimization opportunities
3. Resource utilization insights
4. Potential performance bottlenecks
5. Actionable suggestions for improvement

Focus on:
- Resource utilization (CPU, memory, replicas)
- Scaling patterns and efficiency
- Performance trends and anomalies
- Optimization opportunities
- Best practices for scaling and resource management

Respond in JSON format with this structure:
{
  "root_cause": "Brief summary of key findings",
  "severity": "low|medium|high|critical",
  "issues": [
    {
      "component": "resource type/name",
      "severity": "low|medium|high|critical",
      "description": "performance or scaling issue found",
      "evidence": "specific metrics or configuration"
    }
  ],
  "suggestions": [
    {
      "priority": "high|medium|low",
      "action": "recommended action",
      "command": "kubectl command if applicable",
      "explanation": "why this optimization helps"
    }
  ],
  "quick_fix": "single kubectl command for immediate optimization if applicable",
  "full_analysis": "detailed explanation of metrics analysis and recommendations"
}

Be specific about metrics values and provide actionable scaling recommendations.`, duration, analysisOptionsText, string(resourcesJSON)), nil
}
