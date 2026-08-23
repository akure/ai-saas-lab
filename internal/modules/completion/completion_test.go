package completion

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aisaaslab/internal/kernel"
)

// --- 1. MockProvider Tests ---
func TestMockProvider_GenerateAndStream(t *testing.T) {
	provider := NewMockProvider()
	provider.SimulatedDelay = 0

	req := &ChatRequest{
		Prompt: "Hello Go world",
		Model:  "mock-test",
	}

	// Non-streaming test
	resp, err := provider.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error from mock generate: %v", err)
	}
	if len(resp.Choices) == 0 {
		t.Fatal("expected at least 1 choice in response")
	}
	if !strings.Contains(resp.Choices[0].Message.Content, "Hello Go world") {
		t.Errorf("expected response to contain prompt text, got: %s", resp.Choices[0].Message.Content)
	}

	// Streaming test
	streamChan, err := provider.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error from mock stream: %v", err)
	}

	var sb strings.Builder
	for chunk := range streamChan {
		if chunk.Err != nil {
			t.Fatalf("unexpected stream chunk error: %v", chunk.Err)
		}
		sb.WriteString(chunk.Content)
	}

	if !strings.Contains(sb.String(), "Hello Go world") {
		t.Errorf("expected stream text to contain prompt, got: %s", sb.String())
	}
}

// --- 2. OpenAICompatibleProvider Tests with httptest.Server ---
func TestOpenAICompatibleProvider_RealHTTPAndSSE(t *testing.T) {
	// Mock OpenAI API Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-secret-key" {
			http.Error(w, `{"error":{"message":"Invalid API key"}}`, http.StatusUnauthorized)
			return
		}

		var req openAIChatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("data: {\"id\":\"cmpl-1\",\"choices\":[{\"delta\":{\"content\":\"OpenAI \"}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"id\":\"cmpl-1\",\"choices\":[{\"delta\":{\"content\":\"Stream\"}}]}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}

		// Non-streaming response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(openAIChatResponse{
			ID:      "chatcmpl-mock",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []struct {
				Index        int         `json:"index"`
				Message      ChatMessage `json:"message"`
				FinishReason string      `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: ChatMessage{
						Role:    "assistant",
						Content: "OpenAI mock response OK",
					},
					FinishReason: "stop",
				},
			},
			Usage: UsageInfo{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		})
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(ProviderConfig{
		Name:         "openai",
		BaseURL:      server.URL,
		APIKey:       "test-secret-key",
		DefaultModel: "gpt-4o-mini",
	})

	// Test Non-streaming Generate
	resp, err := provider.Generate(context.Background(), &ChatRequest{Prompt: "test prompt"})
	if err != nil {
		t.Fatalf("OpenAI provider generate error: %v", err)
	}
	if resp.Choices[0].Message.Content != "OpenAI mock response OK" {
		t.Errorf("unexpected output content: %s", resp.Choices[0].Message.Content)
	}

	// Test Streaming
	streamChan, err := provider.Stream(context.Background(), &ChatRequest{Prompt: "test stream", Stream: true})
	if err != nil {
		t.Fatalf("OpenAI provider stream error: %v", err)
	}

	var sb strings.Builder
	for chunk := range streamChan {
		sb.WriteString(chunk.Content)
	}
	if sb.String() != "OpenAI Stream" {
		t.Errorf("expected 'OpenAI Stream', got: %q", sb.String())
	}
}

// --- 3. SessionManager & MemoryEngine Tests ---
func TestSessionManagerAndMemoryEngine(t *testing.T) {
	sm := NewSessionManager()
	apiKey := "test-user-key"

	// Create 2 sessions
	sess1 := sm.GetOrCreateSession(apiKey, "", "developer")
	sess2 := sm.GetOrCreateSession(apiKey, "", "analyst")

	// Add messages to Session 1
	err := sm.AddMessages(sess1.ID,
		ChatMessage{Role: "user", Content: "I prefer Go language for backend AI apps."},
		ChatMessage{Role: "assistant", Content: "Go is great for microservices."},
	)
	if err != nil {
		t.Fatalf("failed to add messages: %v", err)
	}

	// Add messages to Session 2
	_ = sm.AddMessages(sess2.ID,
		ChatMessage{Role: "user", Content: "We are building an AI SaaS MVP in Go."},
	)

	// Verify ListSessions
	sessions := sm.ListSessions(apiKey)
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions for user, got: %d", len(sessions))
	}

	// Test MemoryEngine context retrieval for new session (Session 3)
	memoryEngine := NewMemoryEngine(sm)
	sess3 := sm.GetOrCreateSession(apiKey, "", "tutor")

	memories, contextSnippet := memoryEngine.RetrieveContext(apiKey, sess3.ID, "SaaS MVP", 5)
	if len(memories) == 0 {
		t.Fatal("expected memories to be recalled for query 'SaaS MVP'")
	}
	if !strings.Contains(contextSnippet, "AI SaaS MVP") {
		t.Errorf("expected context snippet to contain recalled user topic, got: %s", contextSnippet)
	}

	// Delete Session 1
	deleted := sm.DeleteSession(sess1.ID)
	if !deleted {
		t.Errorf("expected session 1 deletion to succeed")
	}

	if len(sm.ListSessions(apiKey)) != 2 { // sess2 and sess3 remain
		t.Errorf("expected 2 active sessions remaining after deletion")
	}
}

// --- 4. PersonaManager Tests ---
func TestPersonaManager_RoleSwitching(t *testing.T) {
	pm := NewPersonaManager()

	developer, ok := pm.GetPersona("developer")
	if !ok || !strings.Contains(developer.SystemPrompt, "software engineer") {
		t.Fatalf("expected developer persona, got: %+v", developer)
	}

	analyst, ok := pm.GetPersona("analyst")
	if !ok || !strings.Contains(analyst.SystemPrompt, "analyst") {
		t.Fatalf("expected analyst persona, got: %+v", analyst)
	}

	// Custom System Prompt construction
	prompt := pm.ComposeSystemPrompt("developer", "Focus on clean concurrency.", "[RECALLED CONTEXT: User likes Go]")
	if !strings.Contains(prompt, "software engineer") || !strings.Contains(prompt, "Focus on clean concurrency") || !strings.Contains(prompt, "RECALLED CONTEXT") {
		t.Errorf("composed prompt missing expected sections: %s", prompt)
	}
}

// --- 5. End-to-End HTTP Handler & Kernel Integration Test ---
func TestModule_HTTPHandlersAndSSE(t *testing.T) {
	cfg := kernel.LoadConfig("")
	app := kernel.NewApp(cfg)
	kernel.RegisterDefaultEncoders(app)

	// Register test policies for valid-api-key and under-quota

	app.RegisterPolicy(kernel.FuncPolicy{
		PolicyName: "valid-api-key",
		Fn: func(ctx context.Context, subject any) (bool, error) {
			key, ok := subject.(string)
			return ok && key == "test-api-key", nil
		},
	})
	app.RegisterPolicy(kernel.FuncPolicy{
		PolicyName: "has-active-subscription",
		Fn: func(ctx context.Context, subject any) (bool, error) {
			return true, nil
		},
	})
	app.RegisterPolicy(kernel.FuncPolicy{
		PolicyName: "under-quota",
		Fn: func(ctx context.Context, subject any) (bool, error) {
			return true, nil
		},
	})

	module := New()
	module.MockProvider.SimulatedDelay = 0
	if err := module.Init(app); err != nil {
		t.Fatalf("failed initializing completion module: %v", err)
	}

	// Test non-streaming HTTP POST /v1/chat/completions
	reqBody, _ := json.Marshal(ChatRequest{
		APIKey:      "test-api-key",
		Prompt:      "Hello lab test",
		RoleMode:    "developer",
		MemoryQuery: "test",
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBody))

	module.handleChatCompletion(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got: %d, body: %s", w.Code, w.Body.String())
	}

	var resp ChatResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed decoding json response: %v", err)
	}
	if resp.SessionID == "" {
		t.Error("expected non-empty session_id in response")
	}
	if resp.Persona != "developer" {
		t.Errorf("expected persona 'developer', got: %s", resp.Persona)
	}

	// Test GET /v1/chat/sessions
	wSess := httptest.NewRecorder()
	rSess := httptest.NewRequest(http.MethodGet, "/v1/chat/sessions?api_key=test-api-key", nil)
	module.handleListSessions(wSess, rSess)

	if wSess.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 for list sessions, got: %d", wSess.Code)
	}

	var sessList struct {
		Sessions []*Session `json:"sessions"`
	}
	_ = json.NewDecoder(wSess.Body).Decode(&sessList)
	if len(sessList.Sessions) == 0 {
		t.Error("expected at least 1 session in session list")
	}

	// Test GET /v1/chat/personas
	wPer := httptest.NewRecorder()
	rPer := httptest.NewRequest(http.MethodGet, "/v1/chat/personas", nil)
	module.handleListPersonas(wPer, rPer)

	if wPer.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 for personas list, got: %d", wPer.Code)
	}
}
