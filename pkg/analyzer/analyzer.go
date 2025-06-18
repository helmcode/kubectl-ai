package analyzer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Analyzer struct {
	apiKey string
	client *http.Client
}

type Analysis struct {
	Problem      string            `json:"problem"`
	RootCause    string            `json:"root_cause"`
	Severity     string            `json:"severity"`
	Issues       []Issue           `json:"issues"`
	Suggestions  []Suggestion      `json:"suggestions"`
	QuickFix     string            `json:"quick_fix,omitempty"`
	FullAnalysis string            `json:"full_analysis"`
}

type Issue struct {
	Component   string `json:"component"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Evidence    string `json:"evidence,omitempty"`
}

type Suggestion struct {
	Priority    string `json:"priority"`
	Action      string `json:"action"`
	Command     string `json:"command,omitempty"`
	Explanation string `json:"explanation"`
}

// New creates a new AI analyzer
func New(apiKey string) *Analyzer {
	return &Analyzer{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Analyze sends the resources to Claude for analysis
func (a *Analyzer) Analyze(problem string, resources map[string]interface{}) (*Analysis, error) {
	// Convert resources to YAML for better readability
	resourcesYAML, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resources: %w", err)
	}

	prompt := a.buildPrompt(problem, string(resourcesYAML))
	
	// Call Claude API
	response, err := a.callClaude(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Claude API: %w", err)
	}

	// Parse the response
	analysis, err := a.parseResponse(response, problem)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return analysis, nil
}

func (a *Analyzer) buildPrompt(problem string, resources string) string {
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

Focus on the specific problem mentioned. Be concise but thorough.`, problem, resources)
}

func (a *Analyzer) callClaude(prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": "claude-3-5-sonnet-20241022",
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens": 4000,
		"temperature": 0,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Claude API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse Claude's response
	var claudeResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", err
	}

	if claudeResp.Error.Message != "" {
		return "", fmt.Errorf("Claude API error: %s", claudeResp.Error.Message)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from Claude")
	}

	return claudeResp.Content[0].Text, nil
}

func (a *Analyzer) parseResponse(response string, problem string) (*Analysis, error) {
	// Try to parse as JSON
	var analysis Analysis
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		// If JSON parsing fails, create a basic analysis from the text response
		analysis = Analysis{
			Problem:      problem,
			RootCause:    "Analysis completed (see full analysis for details)",
			Severity:     "medium",
			FullAnalysis: response,
			Issues: []Issue{
				{
					Component:   "general",
					Severity:    "medium",
					Description: "See full analysis for detailed information",
				},
			},
			Suggestions: []Suggestion{
				{
					Priority:    "high",
					Action:      "Review the full analysis below",
					Explanation: response,
				},
			},
		}
	}

	// Ensure problem is set
	if analysis.Problem == "" {
		analysis.Problem = problem
	}

	return &analysis, nil
}