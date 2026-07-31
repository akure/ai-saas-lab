package completion

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// SessionManager manages active multi-turn conversation sessions in memory.
type SessionManager struct {
	mu           sync.RWMutex
	sessions     map[string]*Session
	apiKeyToSess map[string][]string
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:     make(map[string]*Session),
		apiKeyToSess: make(map[string][]string),
	}
}

// GetOrCreateSession retrieves an existing session or initializes a new conversation thread.
func (sm *SessionManager) GetOrCreateSession(apiKey, sessionID, persona string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sessionID != "" {
		if sess, exists := sm.sessions[sessionID]; exists {
			if persona != "" {
				sess.Persona = persona
			}
			sess.UpdatedAt = time.Now()
			return sess
		}
	} else {
		sessionID = generateSessionID()
	}

	newSess := &Session{
		ID:        sessionID,
		APIKey:    apiKey,
		Persona:   persona,
		Messages:  make([]ChatMessage, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	sm.sessions[sessionID] = newSess
	if apiKey != "" {
		sm.apiKeyToSess[apiKey] = append(sm.apiKeyToSess[apiKey], sessionID)
	}

	return newSess
}

// GetSession fetches a session by ID.
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sess, exists := sm.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}
	return sess, nil
}

// AddMessages appends new messages to a session transcript.
func (sm *SessionManager) AddMessages(sessionID string, msgs ...ChatMessage) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sess, exists := sm.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	now := time.Now()
	for i := range msgs {
		if msgs[i].Timestamp.IsZero() {
			msgs[i].Timestamp = now
		}
	}

	sess.Messages = append(sess.Messages, msgs...)
	sess.UpdatedAt = now
	return nil
}

// GetHistory retrieves up to `limit` recent messages from a session history.
func (sm *SessionManager) GetHistory(sessionID string, limit int) []ChatMessage {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sess, exists := sm.sessions[sessionID]
	if !exists {
		return nil
	}

	msgs := sess.Messages
	if limit > 0 && len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}

	out := make([]ChatMessage, len(msgs))
	copy(out, msgs)
	return out
}

// ListSessions returns all conversation sessions owned by an API key.
func (sm *SessionManager) ListSessions(apiKey string) []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	ids, exists := sm.apiKeyToSess[apiKey]
	if !exists {
		return []*Session{}
	}

	res := make([]*Session, 0, len(ids))
	for _, id := range ids {
		if sess, found := sm.sessions[id]; found {
			res = append(res, sess)
		}
	}
	return res
}

// DeleteSession removes a conversation session.
func (sm *SessionManager) DeleteSession(sessionID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sess, exists := sm.sessions[sessionID]
	if !exists {
		return false
	}

	delete(sm.sessions, sessionID)

	// Remove from user index
	if sess.APIKey != "" {
		list := sm.apiKeyToSess[sess.APIKey]
		newIdx := make([]string, 0, len(list))
		for _, id := range list {
			if id != sessionID {
				newIdx = append(newIdx, id)
			}
		}
		sm.apiKeyToSess[sess.APIKey] = newIdx
	}

	return true
}

func generateSessionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("sess-%d-%s", time.Now().Unix(), hex.EncodeToString(b))
}
