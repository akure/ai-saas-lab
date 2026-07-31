package completion

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// MockProvider provides a deterministic mock AI provider for testing and offline development.
type MockProvider struct {
	SimulatedDelay time.Duration
	SimulateError  error
	reqCounter     uint64
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		SimulatedDelay: 20 * time.Millisecond,
	}
}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) Generate(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if m.SimulateError != nil {
		return nil, m.SimulateError
	}

	prompt := m.extractPrompt(req)
	words := mockTokens(prompt)
	fullText := strings.Join(words, " ")

	id := fmt.Sprintf("mock-cmpl-%d", atomic.AddUint64(&m.reqCounter, 1))
	model := req.Model
	if model == "" {
		model = "mock-mini-1"
	}

	return &ChatResponse{
		ID:      id,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []ChatChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:      "assistant",
					Content:   fullText,
					Timestamp: time.Now(),
				},
				FinishReason: "stop",
			},
		},
		Usage: UsageInfo{
			PromptTokens:     len(strings.Fields(prompt)),
			CompletionTokens: len(words),
			TotalTokens:      len(strings.Fields(prompt)) + len(words),
		},
	}, nil
}

func (m *MockProvider) Stream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	if m.SimulateError != nil {
		return nil, m.SimulateError
	}

	out := make(chan StreamChunk, 20)
	prompt := m.extractPrompt(req)
	words := mockTokens(prompt)
	id := fmt.Sprintf("mock-cmpl-%d", atomic.AddUint64(&m.reqCounter, 1))

	go func() {
		defer close(out)
		tokensSent := 0

		for idx, w := range words {
			select {
			case <-ctx.Done():
				return
			default:
			}

			chunkText := w
			if idx < len(words)-1 {
				chunkText += " "
			}

			tokensSent++
			out <- StreamChunk{
				ID:         id,
				Content:    chunkText,
				TokensUsed: tokensSent,
			}

			if m.SimulatedDelay > 0 {
				time.Sleep(m.SimulatedDelay)
			}
		}

		out <- StreamChunk{
			ID:           id,
			Content:      "",
			FinishReason: "stop",
			TokensUsed:   tokensSent,
		}
	}()

	return out, nil
}

func (m *MockProvider) extractPrompt(req *ChatRequest) string {
	if req.Prompt != "" {
		return req.Prompt
	}
	if len(req.Messages) > 0 {
		// Return content of the last user message
		for i := len(req.Messages) - 1; i >= 0; i-- {
			if req.Messages[i].Role == "user" {
				return req.Messages[i].Content
			}
		}
		return req.Messages[len(req.Messages)-1].Content
	}
	return "Hello"
}

func mockTokens(prompt string) []string {
	words := strings.Fields(prompt)
	if len(words) == 0 {
		words = []string{"(empty", "prompt)"}
	}
	res := []string{"[Mock", "Response]:"}
	res = append(res, words...)
	return res
}
