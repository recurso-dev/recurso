package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const openAIChatCompletionsURL = "https://api.openai.com/v1/chat/completions"

// openAIHTTPClient is shared and timeout-bounded. Completions can be slow, so
// the timeout is generous, but a missing one lets a stalled connection hang the
// caller forever. Reusing one client also pools connections across requests.
var openAIHTTPClient = &http.Client{Timeout: 60 * time.Second}

const (
	openAIMaxRetries  = 3
	openAIBaseBackoff = 500 * time.Millisecond
)

// openAIBackoff waits an exponentially growing delay before a retry, aborting
// early (returning false) if the caller's context is cancelled.
func openAIBackoff(ctx context.Context, attempt int) bool {
	select {
	case <-time.After(openAIBaseBackoff * time.Duration(int64(1)<<attempt)):
		return true
	case <-ctx.Done():
		return false
	}
}

type OpenAIProvider struct {
	apiKey  string
	model   string
	baseURL string // overridable in tests; defaults to the OpenAI endpoint
}

func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:  apiKey,
		model:   "gpt-4o",
		baseURL: openAIChatCompletionsURL,
	}
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *OpenAIProvider) GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key is not set")
	}

	reqBody := openAIRequest{
		Model: p.model,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Send with a bounded timeout and retry transient failures. A raw
	// &http.Client{} has no timeout, so a stalled connection would hang the
	// caller (an analytics request) forever. The body is a fixed []byte, so we
	// rebuild the request each attempt (a consumed body can't be re-sent).
	var respBody []byte
	var statusCode int
	for attempt := 0; ; attempt++ {
		req, rErr := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(jsonData))
		if rErr != nil {
			return "", rErr
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p.apiKey)

		resp, dErr := openAIHTTPClient.Do(req)
		if dErr != nil {
			if attempt < openAIMaxRetries && ctx.Err() == nil && openAIBackoff(ctx, attempt) {
				continue
			}
			return "", dErr
		}
		respBody, err = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return "", err
		}
		statusCode = resp.StatusCode

		// Retry on rate limit / server errors (transient); other codes are terminal.
		if (statusCode == http.StatusTooManyRequests || statusCode >= 500) &&
			attempt < openAIMaxRetries && ctx.Err() == nil && openAIBackoff(ctx, attempt) {
			continue
		}
		break
	}

	if statusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error (%d): %s", statusCode, string(respBody))
	}

	var res openAIResponse
	if err := json.Unmarshal(respBody, &res); err != nil {
		return "", err
	}

	if res.Error != nil {
		return "", fmt.Errorf("OpenAI error: %s", res.Error.Message)
	}

	if len(res.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from OpenAI")
	}

	return res.Choices[0].Message.Content, nil
}
