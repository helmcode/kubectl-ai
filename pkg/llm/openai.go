package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenAI struct {
	apiKey string
	client *http.Client
	model  string
}

func NewOpenAI(apiKey string) *OpenAI {
	return &OpenAI{
		apiKey: apiKey,
		client: &http.Client{Timeout: 60 * time.Second},
		model:  "gpt-4o", // Latest GPT-4 model
	}
}

func NewOpenAIWithModel(apiKey, model string) *OpenAI {
	return &OpenAI{
		apiKey: apiKey,
		client: &http.Client{Timeout: 60 * time.Second},
		model:  model,
	}
}

func (o *OpenAI) Chat(prompt string) (string, error) {
	body := map[string]interface{}{
		"model": o.model,
		"messages": []map[string]string{{
			"role":    "user",
			"content": prompt,
		}},
		"max_tokens":  4000,
		"temperature": 0,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.apiKey))

	resp, err := o.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	// OpenAI response structure
	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBytes, &openaiResp); err != nil {
		return "", err
	}
	if openaiResp.Error.Message != "" {
		return "", fmt.Errorf("OpenAI API error: %s", openaiResp.Error.Message)
	}
	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenAI")
	}
	return openaiResp.Choices[0].Message.Content, nil
}

// GetModel returns the model being used by this OpenAI client
func (o *OpenAI) GetModel() string {
	return o.model
}
