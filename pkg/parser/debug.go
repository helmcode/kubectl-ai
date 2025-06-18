package parser

import (
    "encoding/json"
    "regexp"
    "strings"
)

import "github.com/helmcode/kubectl-ai/pkg/model"

func ParseDebugResponse(raw string, problem string) (*model.Analysis, error) {
    // Remove markdown code fences if present
    cleaned := stripFences(raw)

    var analysis model.Analysis
    if err := json.Unmarshal([]byte(cleaned), &analysis); err != nil {
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

// stripFences removes markdown code fences such as ```json ... ``` so JSON can be parsed
func stripFences(text string) string {
    re := regexp.MustCompile("```[a-zA-Z]*\n|```")
    return strings.TrimSpace(re.ReplaceAllString(text, ""))
}
