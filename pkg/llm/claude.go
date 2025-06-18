package llm

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type Claude struct {
    apiKey string
    client *http.Client
    model  string
}

func NewClaude(apiKey string) *Claude {
    return &Claude{
        apiKey: apiKey,
        client: &http.Client{Timeout: 60 * time.Second},
        model:  "claude-sonnet-4-20250514",
    }
}

func (c *Claude) Chat(prompt string) (string, error) {
    body := map[string]interface{}{
        "model": c.model,
        "messages": []map[string]string{{
            "role":    "user",
            "content": prompt,
        }},
        "max_tokens": 4000,
        "temperature": 0,
    }

    jsonBody, err := json.Marshal(body)
    if err != nil {
        return "", err
    }

    req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
    if err != nil {
        return "", err
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("x-api-key", c.apiKey)
    req.Header.Set("anthropic-version", "2023-06-01")

    resp, err := c.client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    respBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("Claude API error (status %d): %s", resp.StatusCode, string(respBytes))
    }

    // Minimal struct to pull out the content text.
    var claudeResp struct {
        Content []struct {
            Text string `json:"text"`
        } `json:"content"`
        Error struct {
            Message string `json:"message"`
        } `json:"error"`
    }
    if err := json.Unmarshal(respBytes, &claudeResp); err != nil {
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
