package llm

import (
	"fmt"
	"os"
	"strings"
)

// Provider represents the LLM provider type
type Provider string

const (
	ProviderClaude Provider = "claude"
	ProviderOpenAI Provider = "openai"
)

// Factory creates LLM instances based on provider
type Factory struct{}

// NewFactory creates a new LLM factory
func NewFactory() *Factory {
	return &Factory{}
}

// CreateLLM creates an LLM instance based on provider and configuration
func (f *Factory) CreateLLM(provider Provider, config map[string]string) (LLM, error) {
	switch provider {
	case ProviderClaude:
		apiKey := config["api_key"]
		if apiKey == "" {
			return nil, fmt.Errorf("Claude API key is required")
		}
		if model := config["model"]; model != "" {
			return NewClaudeWithModel(apiKey, model), nil
		}
		return NewClaude(apiKey), nil

	case ProviderOpenAI:
		apiKey := config["api_key"]
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key is required")
		}
		if model := config["model"]; model != "" {
			return NewOpenAIWithModel(apiKey, model), nil
		}
		return NewOpenAI(apiKey), nil

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}
}

// CreateFromEnv creates an LLM instance from environment variables
func (f *Factory) CreateFromEnv() (LLM, error) {
	// Check which provider is configured
	provider := strings.ToLower(os.Getenv("LLM_PROVIDER"))

	switch provider {
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
		}
		model := os.Getenv("OPENAI_MODEL")
		if model != "" {
			return NewOpenAIWithModel(apiKey, model), nil
		}
		return NewOpenAI(apiKey), nil

	case "claude", "":
		// Default to Claude for backward compatibility
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
		}
		model := os.Getenv("CLAUDE_MODEL")
		if model != "" {
			return NewClaudeWithModel(apiKey, model), nil
		}
		return NewClaude(apiKey), nil

	default:
		return nil, fmt.Errorf("unsupported LLM_PROVIDER: %s (supported: claude, openai)", provider)
	}
}

// GetAvailableProviders returns a list of available LLM providers
func (f *Factory) GetAvailableProviders() []Provider {
	return []Provider{ProviderClaude, ProviderOpenAI}
}

// CreateFromEnv creates an LLM instance from environment variables
// This is a convenience function that creates a new factory and uses it
func CreateFromEnv(providerOverride, modelOverride string) (LLM, error) {
	factory := &Factory{}

	// If provider is explicitly set, use that
	if providerOverride != "" {
		provider := strings.ToLower(providerOverride)
		switch provider {
		case "openai":
			apiKey := os.Getenv("OPENAI_API_KEY")
			if apiKey == "" {
				return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
			}
			model := modelOverride
			if model == "" {
				model = os.Getenv("OPENAI_MODEL")
			}
			if model != "" {
				return NewOpenAIWithModel(apiKey, model), nil
			}
			return NewOpenAI(apiKey), nil

		case "claude":
			apiKey := os.Getenv("ANTHROPIC_API_KEY")
			if apiKey == "" {
				return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
			}
			model := modelOverride
			if model == "" {
				model = os.Getenv("CLAUDE_MODEL")
			}
			if model != "" {
				return NewClaudeWithModel(apiKey, model), nil
			}
			return NewClaude(apiKey), nil

		default:
			return nil, fmt.Errorf("unsupported provider: %s (supported: claude, openai)", provider)
		}
	}

	// Otherwise, auto-detect from environment
	return factory.CreateFromEnv()
}
