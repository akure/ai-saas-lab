package completion

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAICompatibleProvider connects to any OpenAI API specification endpoint.
type OpenAICompatibleProvider struct {
	name         string
	baseURL      string
	apiKey       string
	defaultModel string
	httpClient   *http.Client
	maxRetries   int
}

func NewOpenAICompatibleProvider(cfg ProviderConfig) *OpenAICompatibleProvider {
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := cfg.DefaultModel
	if model == "" {
		model = "gpt-4o-mini"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	name := cfg.Name
	if name == "" {
		name = "openai"
	}

	return &OpenAICompatibleProvider{
		name:         name,
		baseURL:      baseURL,
		apiKey:       cfg.APIKey,
		defaultModel: model,
		httpClient:   &http.Client{Timeout: timeout},
		maxRetries:   3,
	}
}

func (p *OpenAICompatibleProvider) Name() string {
	return p.name
}

type openAIChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage UsageInfo `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

type openAIStreamChunk struct {
	ID      string `json:"id"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func (p *OpenAICompatibleProvider) Generate(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	msgs := req.Messages
	if len(msgs) == 0 {
		msgs = []ChatMessage{
			{Role: "user", Content: req.Prompt},
		}
	}

	body := openAIChatRequest{
		Model:       model,
		Messages:    msgs,
		Stream:      false,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal openai request: %w", err)
	}

	var respObj openAIChatResponse
	err = p.executeWithRetry(ctx, func() (*http.Response, error) {
		httpReq, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(payload))
		if reqErr != nil {
			return nil, reqErr
		}
		p.applyHeaders(httpReq)
		return p.httpClient.Do(httpReq)
	}, func(r *http.Response) error {
		defer r.Body.Close()
		respBytes, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			return fmt.Errorf("failed reading response body: %w", readErr)
		}
		if r.StatusCode >= 400 {
			return p.parseHTTPError(r.StatusCode, respBytes)
		}
		return json.Unmarshal(respBytes, &respObj)
	})

	if err != nil {
		return nil, err
	}

	if respObj.Error != nil {
		return nil, fmt.Errorf("openai api error: %s", respObj.Error.Message)
	}

	choices := make([]ChatChoice, len(respObj.Choices))
	for i, c := range respObj.Choices {
		choices[i] = ChatChoice{
			Index:        c.Index,
			Message:      c.Message,
			FinishReason: c.FinishReason,
		}
	}

	return &ChatResponse{
		ID:      respObj.ID,
		Object:  respObj.Object,
		Created: respObj.Created,
		Model:   respObj.Model,
		Choices: choices,
		Usage:   respObj.Usage,
	}, nil
}

func (p *OpenAICompatibleProvider) Stream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	msgs := req.Messages
	if len(msgs) == 0 {
		msgs = []ChatMessage{
			{Role: "user", Content: req.Prompt},
		}
	}

	body := openAIChatRequest{
		Model:       model,
		Messages:    msgs,
		Stream:      true,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stream request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	p.applyHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, p.parseHTTPError(resp.StatusCode, respBytes)
	}

	out := make(chan StreamChunk, 20)

	go func() {
		defer resp.Body.Close()
		defer close(out)

		scanner := bufio.NewScanner(resp.Body)
		tokensCount := 0

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, ":") {
				continue // skip comments / empty lines
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			dataStr := strings.TrimPrefix(line, "data: ")
			if dataStr == "[DONE]" {
				out <- StreamChunk{FinishReason: "stop", TokensUsed: tokensCount}
				return
			}

			var chunk openAIStreamChunk
			if unmarshalErr := json.Unmarshal([]byte(dataStr), &chunk); unmarshalErr != nil {
				continue
			}

			if len(chunk.Choices) > 0 {
				content := chunk.Choices[0].Delta.Content
				if content != "" {
					tokensCount++
					out <- StreamChunk{
						ID:         chunk.ID,
						Content:    content,
						TokensUsed: tokensCount,
					}
				}
				if chunk.Choices[0].FinishReason != "" {
					out <- StreamChunk{
						ID:           chunk.ID,
						FinishReason: chunk.Choices[0].FinishReason,
						TokensUsed:   tokensCount,
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			out <- StreamChunk{Err: err}
		}
	}()

	return out, nil
}

func (p *OpenAICompatibleProvider) applyHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
}

func (p *OpenAICompatibleProvider) executeWithRetry(ctx context.Context, makeReq func() (*http.Response, error), processResp func(*http.Response) error) error {
	var lastErr error
	backoff := 100 * time.Millisecond

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := makeReq()
		if err != nil {
			lastErr = err
		} else {
			procErr := processResp(resp)
			if procErr == nil {
				return nil
			}
			lastErr = procErr
			// Retry only on rate limits or 5xx server errors
			if !isRetryableError(procErr) {
				return procErr
			}
		}

		if attempt < p.maxRetries {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	return lastErr
}

func (p *OpenAICompatibleProvider) parseHTTPError(statusCode int, body []byte) error {
	var errBody struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}

	_ = json.Unmarshal(body, &errBody)
	msg := errBody.Error.Message
	if msg == "" {
		msg = string(body)
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed (401): %s", msg)
	case http.StatusTooManyRequests:
		return fmt.Errorf("%w: %s", ErrRateLimited, msg)
	case http.StatusGatewayTimeout, http.StatusServiceUnavailable:
		return fmt.Errorf("%w: %s", ErrProviderTimeout, msg)
	default:
		return fmt.Errorf("upstream API error (status %d): %s", statusCode, msg)
	}
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), ErrRateLimited.Error()) ||
		strings.Contains(err.Error(), ErrProviderTimeout.Error()) ||
		strings.Contains(err.Error(), "500") ||
		strings.Contains(err.Error(), "502") ||
		strings.Contains(err.Error(), "503") ||
		strings.Contains(err.Error(), "504") {
		return true
	}
	return false
}
