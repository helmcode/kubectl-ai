package parser

import (
	"encoding/json"
	"fmt"
)

import "github.com/helmcode/kubectl-ai/pkg/model"

func ParseMetricsResponse(raw string, duration string) (*model.Analysis, error) {
	// Remove markdown code fences if present
	cleaned := stripFences(raw)

	var analysis model.Analysis
	if err := json.Unmarshal([]byte(cleaned), &analysis); err != nil {
		// Fallback – could not parse JSON, embed entire text.
		analysis = model.Analysis{
			Problem:      fmt.Sprintf("Metrics Analysis (%s)", duration),
			RootCause:    "Metrics analysis completed (see full analysis for details)",
			Severity:     "medium",
			FullAnalysis: raw,
			Issues: []model.Issue{{
				Component:   "metrics",
				Severity:    "medium",
				Description: "See full analysis for detailed metrics information",
			}},
			Suggestions: []model.Suggestion{{
				Priority:    "high",
				Action:      "Review the full metrics analysis below",
				Explanation: raw,
			}},
		}
	}
	
	if analysis.Problem == "" {
		analysis.Problem = fmt.Sprintf("Metrics Analysis (%s)", duration)
	}
	
	return &analysis, nil
}