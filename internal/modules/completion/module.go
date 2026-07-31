package completion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"aisaaslab/internal/kernel"
)

// UsageRecord is an internal message encoded with gob traveling from HTTP handler -> Dispatch -> EventBus -> Billing.
type UsageRecord struct {
	APIKey string
	Tokens int
	Model  string
}

type Module struct {
	app            *kernel.App
	Providers      *ProviderRegistry
	Sessions       *SessionManager
	Memory         *MemoryEngine
	Personas       *PersonaManager
	MockProvider   *MockProvider
	OpenAIProvider *OpenAICompatibleProvider
}

func New() *Module {
	sessions := NewSessionManager()
	providers := NewProviderRegistry()
	mock := NewMockProvider()
	providers.Register(mock)

	return &Module{
		Providers:    providers,
		Sessions:     sessions,
		Memory:       NewMemoryEngine(sessions),
		Personas:     NewPersonaManager(),
		MockProvider: mock,
	}
}

func (m *Module) Name() string { return "completion" }

func (m *Module) Init(app *kernel.App) error {
	m.app = app

	// Configure real OpenAI-compatible provider if API key or BaseURL provided
	openAIKey := os.Getenv("OPENAI_API_KEY")
	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")
	defaultProvider := os.Getenv("COMPLETION_PROVIDER_DEFAULT")

	if openAIKey != "" || openAIBaseURL != "" {
		m.OpenAIProvider = NewOpenAICompatibleProvider(ProviderConfig{
			Name:         "openai",
			BaseURL:      openAIBaseURL,
			APIKey:       openAIKey,
			DefaultModel: "gpt-4o-mini",
			Timeout:      30 * time.Second,
		})
		m.Providers.Register(m.OpenAIProvider)
		if defaultProvider == "" {
			defaultProvider = "openai"
		}
	}

	if defaultProvider != "" {
		m.Providers.SetDefault(defaultProvider)
	}

	// Register gob codec for usage reporting
	app.RegisterMessage(kernel.MessageDescriptor{
		Type:    "usage.recorded",
		Encoder: "gob",
		New:     func() any { return &UsageRecord{} },
	})

	// Bridge dispatch to event bus for Billing module
	app.RegisterHandler("usage.recorded", func(ctx context.Context, msg any) error {
		rec := msg.(*UsageRecord)
		app.Events.Publish("usage.recorded", rec)
		return nil
	})

	// HTTP routes
	app.Mux.HandleFunc("POST /v1/chat/completions", m.handleChatCompletion)
	app.Mux.HandleFunc("GET /v1/chat/sessions", m.handleListSessions)
	app.Mux.HandleFunc("GET /v1/chat/sessions/{id}", m.handleGetSession)
	app.Mux.HandleFunc("DELETE /v1/chat/sessions/{id}", m.handleDeleteSession)
	app.Mux.HandleFunc("GET /v1/chat/personas", m.handleListPersonas)

	return nil
}

func (m *Module) Start(ctx context.Context) error { return nil }
func (m *Module) Stop(ctx context.Context) error  { return nil }

func (m *Module) handleChatCompletion(w http.ResponseWriter, r *http.Request) {
	app := m.app
	ctx := r.Context()

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad request body"}`, http.StatusBadRequest)
		return
	}

	if req.APIKey == "" {
		http.Error(w, `{"error":"missing api_key"}`, http.StatusUnauthorized)
		return
	}

	// Policy check via kernel app (valid-api-key, under-quota)
	if err := app.CheckPolicies(ctx, req.APIKey, "valid-api-key", "under-quota"); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Validate input content
	prompt := req.Prompt
	if prompt == "" && len(req.Messages) > 0 {
		for i := len(req.Messages) - 1; i >= 0; i-- {
			if req.Messages[i].Role == "user" {
				prompt = req.Messages[i].Content
				break
			}
		}
	}
	if prompt == "" && len(req.Messages) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": ErrInvalidRequest.Error()})
		return
	}

	// Get or initialize chat session
	session := m.Sessions.GetOrCreateSession(req.APIKey, req.SessionID, req.RoleMode)

	// Memory & context retrieval across past conversations
	var recalledCtx string
	var recalledMemories []RecalledMemory
	shouldRetrieveMemory := req.IncludeHistory || req.MemoryQuery != "" || isMemoryTriggerPrompt(prompt)

	if shouldRetrieveMemory {
		recalledMemories, recalledCtx = m.Memory.RetrieveContext(req.APIKey, session.ID, req.MemoryQuery, 5)
	}

	// Compose system prompt with persona instructions & recalled memory context
	composedSystemPrompt := m.Personas.ComposeSystemPrompt(req.RoleMode, req.SystemPrompt, recalledCtx)

	// Prepare payload messages
	req.Messages = m.Personas.BuildMessages(&req, session, composedSystemPrompt)

	// If streaming requested or client header prefers stream
	if req.Stream {
		m.streamCompletion(w, r, req, session, prompt, len(recalledMemories))
		return
	}

	// Non-streaming completion
	resp, providerUsed, err := m.Providers.GenerateWithFallback(ctx, req.Provider, &req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp.SessionID = session.ID
	resp.Persona = session.Persona
	resp.MemoriesRecalled = len(recalledMemories)

	// Update session history with prompt and assistant response
	_ = m.Sessions.AddMessages(session.ID,
		ChatMessage{Role: "user", Content: prompt, Timestamp: time.Now()},
		ChatMessage{Role: "assistant", Content: extractAssistantResponse(resp), Timestamp: time.Now()},
	)

	// Record usage via kernel gob dispatch
	m.recordUsage(req.APIKey, resp.Usage.TotalTokens, providerUsed)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (m *Module) streamCompletion(w http.ResponseWriter, r *http.Request, req ChatRequest, session *Session, userPrompt string, memoriesCount int) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming unsupported"}`, http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	streamChan, providerUsed, err := m.Providers.StreamWithFallback(ctx, req.Provider, &req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	var sb strings.Builder
	tokensSent := 0

	for chunk := range streamChan {
		select {
		case <-ctx.Done():
			return // Client disconnected
		default:
		}

		if chunk.Err != nil {
			fmt.Fprintf(w, "event: error\ndata: {\"error\":%q}\n\n", chunk.Err.Error())
			flusher.Flush()
			return
		}

		if chunk.Content != "" {
			sb.WriteString(chunk.Content)
			tokensSent++
			fmt.Fprintf(w, "data: %s\n\n", chunk.Content)
			flusher.Flush()
		}

		if chunk.FinishReason != "" {
			tokensSent = chunk.TokensUsed
		}
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	if tokensSent == 0 {
		tokensSent = len(strings.Fields(sb.String()))
	}

	// Update session history
	_ = m.Sessions.AddMessages(session.ID,
		ChatMessage{Role: "user", Content: userPrompt, Timestamp: time.Now()},
		ChatMessage{Role: "assistant", Content: sb.String(), Timestamp: time.Now()},
	)

	// Record usage
	m.recordUsage(req.APIKey, tokensSent, providerUsed)
}

func (m *Module) handleListSessions(w http.ResponseWriter, r *http.Request) {
	apiKey := extractAPIKey(r)
	if apiKey == "" {
		http.Error(w, `{"error":"missing api_key param or header"}`, http.StatusUnauthorized)
		return
	}

	sessions := m.Sessions.ListSessions(apiKey)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"sessions": sessions})
}

func (m *Module) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		http.Error(w, `{"error":"missing session id"}`, http.StatusBadRequest)
		return
	}

	session, err := m.Sessions.GetSession(sessionID)
	if err != nil {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(session)
}

func (m *Module) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		http.Error(w, `{"error":"missing session id"}`, http.StatusBadRequest)
		return
	}

	ok := m.Sessions.DeleteSession(sessionID)
	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "id": sessionID})
}

func (m *Module) handleListPersonas(w http.ResponseWriter, r *http.Request) {
	personas := m.Personas.ListPersonas()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"personas": personas})
}

func (m *Module) recordUsage(apiKey string, tokens int, model string) {
	if m.app == nil || apiKey == "" {
		return
	}
	rec := &UsageRecord{APIKey: apiKey, Tokens: tokens, Model: model}
	enc, err := m.app.Encoder("gob")
	if err != nil {
		return
	}
	raw, err := enc.Encode(rec)
	if err != nil {
		return
	}
	_ = m.app.Dispatch(context.Background(), "usage.recorded", raw)
}

func extractAssistantResponse(resp *ChatResponse) string {
	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content
	}
	return ""
}

func extractAPIKey(r *http.Request) string {
	if key := r.URL.Query().Get("api_key"); key != "" {
		return key
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func isMemoryTriggerPrompt(prompt string) bool {
	p := strings.ToLower(prompt)
	return strings.Contains(p, "what did we talk about") ||
		strings.Contains(p, "previous conversation") ||
		strings.Contains(p, "remember when") ||
		strings.Contains(p, "what was my last") ||
		strings.Contains(p, "recap prior")
}
