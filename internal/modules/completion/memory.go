package completion

import (
	"fmt"
	"strings"
)

// MemoryEngine searches past conversation histories and synthesizes recalled context.
type MemoryEngine struct {
	sessionManager *SessionManager
}

func NewMemoryEngine(sm *SessionManager) *MemoryEngine {
	return &MemoryEngine{
		sessionManager: sm,
	}
}

// RecalledMemory represents a snippet retrieved from previous sessions.
type RecalledMemory struct {
	SessionID string
	Role      MessageRole
	Content   string
}

// RetrieveContext searches across past conversation threads belonging to an APIKey.
func (me *MemoryEngine) RetrieveContext(apiKey string, currentSessionID string, query string, maxItems int) ([]RecalledMemory, string) {
	if me.sessionManager == nil || apiKey == "" {
		return nil, ""
	}

	sessions := me.sessionManager.ListSessions(apiKey)
	if len(sessions) == 0 {
		return nil, ""
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))
	memories := make([]RecalledMemory, 0)

	for _, sess := range sessions {
		// Skip current session to retrieve memories strictly from PREVIOUS conversations
		if currentSessionID != "" && sess.ID == currentSessionID {
			continue
		}

		for _, msg := range sess.Messages {
			if msg.Role == "system" {
				continue
			}

			// If query is provided, match keywords; otherwise collect recent key user messages
			match := false
			if queryLower != "" {
				if strings.Contains(strings.ToLower(msg.Content), queryLower) {
					match = true
				}
			} else {
				// Default heuristic: recall user questions or assistant responses from past conversations
				match = true
			}

			if match {
				memories = append(memories, RecalledMemory{
					SessionID: sess.ID,
					Role:      msg.Role,
					Content:   msg.Content,
				})
			}
		}
	}

	if maxItems > 0 && len(memories) > maxItems {
		memories = memories[len(memories)-maxItems:]
	}

	if len(memories) == 0 {
		return nil, ""
	}

	// Format system prompt snippet containing memories
	var sb strings.Builder
	sb.WriteString("[RECALLED CONTEXT FROM PREVIOUS CONVERSATIONS]\n")
	for _, m := range memories {
		sb.WriteString(fmt.Sprintf("- (Session %s) %s: %s\n", m.SessionID, strings.ToUpper(m.Role.String()), m.Content))
	}
	sb.WriteString("[END RECALLED CONTEXT]\n Use this context to answer the user if relevant.")

	return memories, sb.String()
}
