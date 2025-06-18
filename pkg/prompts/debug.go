package prompts

import (
    "encoding/json"
    "fmt"
)

func BuildDebugPrompt(problem string, resources map[string]interface{}) (string, error) {
    resourcesJSON, err := json.MarshalIndent(resources, "", "  ")
    if err != nil {
        return "", fmt.Errorf("marshal resources: %w", err)
    }

    return fmt.Sprintf(`You are a Kubernetes expert helping to debug configuration issues.

User's Problem: %s

Kubernetes Resources:
%s

Please analyze these Kubernetes resources and provide:
1. The root cause of the problem
2. Specific issues found in the configuration
3. Actionable suggestions to fix the problem
4. If possible, a quick fix command

Respond in JSON format with this structure:
{
  "root_cause": "Brief explanation of the root cause",
  "severity": "low|medium|high|critical",
  "issues": [
    {
      "component": "resource type/name",
      "severity": "low|medium|high|critical",
      "description": "what's wrong",
      "evidence": "specific config line or value"
    }
  ],
  "suggestions": [
    {
      "priority": "high|medium|low",
      "action": "what to do",
      "command": "kubectl command if applicable",
      "explanation": "why this helps"
    }
  ],
  "quick_fix": "single kubectl command for immediate fix if possible",
  "full_analysis": "detailed explanation of the problem and solution"
}

Focus on the specific problem mentioned. Be concise but thorough.`, problem, string(resourcesJSON)), nil
}
