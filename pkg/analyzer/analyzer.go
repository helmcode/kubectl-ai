package analyzer

import (
	"fmt"

	"github.com/helmcode/kubectl-ai/pkg/llm"
	"github.com/helmcode/kubectl-ai/pkg/model"
	"github.com/helmcode/kubectl-ai/pkg/parser"
	"github.com/helmcode/kubectl-ai/pkg/prompts"
)

type Analyzer struct {
	llm llm.LLM
}

func New(apiKey string) *Analyzer {
	// For backward compatibility, default to Claude
	return &Analyzer{llm: llm.NewClaude(apiKey)}
}

func NewWithProvider(provider llm.Provider, config map[string]string) (*Analyzer, error) {
	factory := llm.NewFactory()
	llmInstance, err := factory.CreateLLM(provider, config)
	if err != nil {
		return nil, err
	}
	return &Analyzer{llm: llmInstance}, nil
}

func NewFromEnv() (*Analyzer, error) {
	factory := llm.NewFactory()
	llmInstance, err := factory.CreateFromEnv()
	if err != nil {
		return nil, err
	}
	return &Analyzer{llm: llmInstance}, nil
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

func (a *Analyzer) AnalyzeMetrics(resources map[string]interface{}, duration string, compareScaling, hpaAnalysis, kedaAnalysis bool) (*model.Analysis, error) {
	prompt, err := prompts.BuildMetricsPrompt(resources, duration, compareScaling, hpaAnalysis, kedaAnalysis)
	if err != nil {
		return nil, err
	}

	rawResp, err := a.llm.Chat(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM chat: %w", err)
	}

	return parser.ParseMetricsResponse(rawResp, duration)
}
