package analyzer

import (
    "fmt"

    "github.com/helmcode/kubectl-ai/pkg/llm"
    "github.com/helmcode/kubectl-ai/pkg/parser"
    "github.com/helmcode/kubectl-ai/pkg/prompts"
    "github.com/helmcode/kubectl-ai/pkg/model"
)

type Analyzer struct {
    llm llm.LLM
}

func New(apiKey string) *Analyzer {
    return &Analyzer{llm: llm.NewClaude(apiKey)}
}

func NewWithLLM(l llm.LLM) *Analyzer {
	return &Analyzer{llm: l}
}

func (a *Analyzer) Analyze(problem string, resources map[string]interface{}) (*model.Analysis, error) {
	prompt, err := prompts.BuildDebugPrompt(problem, resources)
	if err != nil {
		return nil, err
	}

	rawResp, err := a.llm.Chat(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM chat: %w", err)
	}

	return parser.ParseDebugResponse(rawResp, problem)
}
