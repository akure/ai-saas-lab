package completion

import (
	"strings"
	"sync"
)

// PersonaManager manages user roles, personas, and dynamic system prompts.
type PersonaManager struct {
	mu       sync.RWMutex
	personas map[string]Persona
}

func NewPersonaManager() *PersonaManager {
	pm := &PersonaManager{
		personas: make(map[string]Persona),
	}
	pm.registerDefaults()
	return pm
}

func (pm *PersonaManager) registerDefaults() {
	defaults := []Persona{
		{
			ID:          "default",
			Name:        "Default Assistant",
			Description: "General-purpose helpful AI assistant",
			SystemPrompt: "You are a helpful, precise, and capable AI assistant.",
		},
		{
			ID:          "developer",
			Name:        "Senior Developer",
			Description: "Expert software engineer focused on idiomatic code, architecture, and best practices",
			SystemPrompt: "You are an expert senior software engineer. Provide clean, modular, production-ready code, elegant architecture suggestions, and robust error handling.",
		},
		{
			ID:          "analyst",
			Name:        "Executive Analyst",
			Description: "Concise, data-driven analyst presenting clear facts and actionable summaries",
			SystemPrompt: "You are an executive technical analyst. Provide direct, concise, bullet-pointed summaries with clear takeaways and zero fluff.",
		},
		{
			ID:          "creative",
			Name:        "Product Strategist",
			Description: "Innovative, creative ideation partner for design and strategy",
			SystemPrompt: "You are an imaginative creative product strategist. Explore novel concepts, offer engaging design ideas, and suggest innovative features.",
		},
		{
			ID:          "tutor",
			Name:        "Technical Tutor",
			Description: "Patient mentor explaining complex concepts with clear step-by-step examples",
			SystemPrompt: "You are a patient senior technical mentor. Explain complex concepts clearly using step-by-step breakdowns, code snippets, and intuitive analogies.",
		},
		{
			ID:          "support",
			Name:        "Customer Support",
			Description: "Empathetic, helpful agent for user guidance and issue resolution",
			SystemPrompt: "You are a polite, empathetic customer support engineer. Guide users gently, address their concerns step-by-step, and offer clear resolution steps.",
		},
	}

	for _, p := range defaults {
		pm.personas[p.ID] = p
	}
}

// GetPersona retrieves a persona by ID/roleMode name.
func (pm *PersonaManager) GetPersona(roleMode string) (Persona, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	p, ok := pm.personas[strings.ToLower(strings.TrimSpace(roleMode))]
	if !ok {
		p, ok = pm.personas["default"]
	}
	return p, ok
}

// ListPersonas returns all registered personas.
func (pm *PersonaManager) ListPersonas() []Persona {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	res := make([]Persona, 0, len(pm.personas))
	for _, p := range pm.personas {
		res = append(res, p)
	}
	return res
}

// RegisterPersona registers a new dynamic persona.
func (pm *PersonaManager) RegisterPersona(p Persona) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.personas[strings.ToLower(p.ID)] = p
}

// ComposeSystemPrompt constructs the final system prompt by combining persona instructions, user custom system prompt, and recalled memories.
func (pm *PersonaManager) ComposeSystemPrompt(roleMode string, customPrompt string, recalledMemoryContext string) string {
	var parts []string

	if roleMode != "" {
		if p, ok := pm.GetPersona(roleMode); ok {
			parts = append(parts, p.SystemPrompt)
		}
	} else if customPrompt == "" {
		if p, ok := pm.GetPersona("default"); ok {
			parts = append(parts, p.SystemPrompt)
		}
	}

	if customPrompt != "" {
		parts = append(parts, customPrompt)
	}

	if recalledMemoryContext != "" {
		parts = append(parts, recalledMemoryContext)
	}

	return strings.Join(parts, "\n\n")
}

func (pm *PersonaManager) BuildMessages(req *ChatRequest, session *Session, systemPrompt string) []ChatMessage {
	msgs := make([]ChatMessage, 0)

	// Add composed system prompt first if present
	if systemPrompt != "" {
		msgs = append(msgs, ChatMessage{Role: "system", Content: systemPrompt})
	}

	// Add prior session history if available
	if session != nil && len(session.Messages) > 0 {
		msgs = append(msgs, session.Messages...)
	}

	// Add current request input if not already part of session messages
	if len(req.Messages) > 0 {
		// Append user messages passed in current request payload
		msgs = append(msgs, req.Messages...)
	} else if req.Prompt != "" {
		// Single prompt mode
		msgs = append(msgs, ChatMessage{Role: "user", Content: req.Prompt})
	}

	return msgs
}
