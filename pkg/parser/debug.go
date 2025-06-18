package parser

import "encoding/json"

import "github.com/helmcode/kubectl-ai/pkg/model"

func ParseDebugResponse(raw string, problem string) (*model.Analysis, error) {
    var analysis model.Analysis
    if err := json.Unmarshal([]byte(raw), &analysis); err != nil {
        // Fallback – could not parse JSON, embed entire text.
        analysis = model.Analysis{
            Problem:   problem,
            RootCause: "Analysis completed (see full analysis for details)",
            Severity:  "medium",
            FullAnalysis: raw,
            Issues: []model.Issue{{
                Component:   "general",
                Severity:    "medium",
                Description: "See full analysis for detailed information",
            }},
            Suggestions: []model.Suggestion{{
                Priority:    "high",
                Action:      "Review the full analysis below",
                Explanation: raw,
            }},
        }
    }
    if analysis.Problem == "" {
        analysis.Problem = problem
    }
    return &analysis, nil
}
