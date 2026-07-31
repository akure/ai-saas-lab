package completion

import (
	"errors"
	"time"
)

var (
	ErrInvalidRequest      = errors.New("invalid completion request: missing api_key or prompt/messages")
	ErrProviderUnavailable = errors.New("ai provider unavailable")
	ErrSessionNotFound     = errors.New("session not found")
	ErrRateLimited         = errors.New("rate limit exceeded from upstream provider")
	ErrProviderTimeout     = errors.New("upstream provider request timed out")
)

// ChatRequest represents the unified request shape sent by callers.
type ChatRequest struct {
	APIKey         string        `json:"api_key"`
	Prompt         string        `json:"prompt,omitempty"`
	Messages       []ChatMessage `json:"messages,omitempty"`
	SessionID      string        `json:"session_id,omitempty"`
	Provider       string        `json:"provider,omitempty"`
	Model          string        `json:"model,omitempty"`
	RoleMode       string        `json:"role_mode,omitempty"`
	SystemPrompt   string        `json:"system_prompt,omitempty"`
	Stream         bool          `json:"stream,omitempty"`
	IncludeHistory bool          `json:"include_history,omitempty"`
	MemoryQuery    string        `json:"memory_query,omitempty"`
	Temperature    float64       `json:"temperature,omitempty"`
	MaxTokens      int           `json:"max_tokens,omitempty"`
}

// ChatMessage is a single message item inside a chat conversation thread.
type ChatMessage struct {
	Role      string    `json:"role"` // system, user, assistant
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// ChatResponse is the OpenAI-compatible standard completion response.
type ChatResponse struct {
	ID               string       `json:"id"`
	Object           string       `json:"object"`
	Created          int64        `json:"created"`
	Model            string       `json:"model"`
	Choices          []ChatChoice `json:"choices"`
	Usage            UsageInfo    `json:"usage"`
	SessionID        string       `json:"session_id,omitempty"`
	Persona          string       `json:"persona,omitempty"`
	MemoriesRecalled int          `json:"memories_recalled,omitempty"`
}

// ChatChoice represents a generated choice in a completion response.
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// UsageInfo tracks token counts for billing and reporting.
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk represents a single token chunk flushed over an SSE connection.
type StreamChunk struct {
	ID           string `json:"id,omitempty"`
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason,omitempty"`
	Err          error  `json:"-"`
	TokensUsed   int    `json:"tokens_used,omitempty"`
}

// Session holds the active conversation state and message log for a user thread.
type Session struct {
	ID        string        `json:"id"`
	APIKey    string        `json:"api_key"`
	Persona   string        `json:"persona"`
	Messages  []ChatMessage `json:"messages"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// Persona defines system instructions and behavioral attributes for role modes.
type Persona struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	SystemPrompt string `json:"system_prompt"`
}

// ProviderConfig configures an AI completion service provider.
type ProviderConfig struct {
	Name       string
	BaseURL    string
	APIKey     string
	DefaultModel string
	Timeout    time.Duration
	Headers    map[string]string
}
