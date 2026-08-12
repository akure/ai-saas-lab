package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"aisaaslab/internal/kernel"
)

var (
	ErrInvalidAPIKey  = errors.New("invalid api key")
	ErrUnknownAPIKey  = errors.New("unknown api key")
	ErrInactiveAPIKey = errors.New("inactive api key")
	ErrMissingPlan    = errors.New("missing api key plan")
)

// APIKeyRecord is the internal representation of an API key managed by the auth module.
type APIKeyRecord struct {
	Key    string            `json:"key"`
	Plan   kernel.MaaSPlanID `json:"plan"`
	Active bool              `json:"active"`
}

// Module registers an authentication policy and API key management routes.
type Module struct {
	service *Service
}

func New() *Module {
	return &Module{
		service: NewService(),
	}
}

func (m *Module) Name() string { return "auth" }

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) Init(app *kernel.App) error {
	if m.service == nil {
		m.service = NewService()
	}
	if app.Config != nil && app.Config.LocalTest {
		m.seedDemoKeys()
	}
	app.Mux.HandleFunc("POST /admin/api-keys", m.handleCreateAPIKey)
	app.RegisterPolicy(kernel.FuncPolicy{
		PolicyName: "valid-api-key",
		Fn: func(ctx context.Context, subject any) (bool, error) {
			key, ok := subject.(string)
			if !ok {
				return false, nil
			}
			err := m.service.Authenticate(key)
			if err != nil {
				return false, nil
			}
			return true, nil
		},
	})
	return nil
}

func (m *Module) Start(ctx context.Context) error { return nil }
func (m *Module) Stop(ctx context.Context) error  { return nil }

func (m *Module) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
		return
	}

	var req struct {
		Plan kernel.MaaSPlanID `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid json body"})
		return
	}

	key, err := m.service.CreateAPIKey(req.Plan)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"api_key": key, "plan": req.Plan})
}

// Service provides thread-safe API key authentication & credential management logic.
type Service struct {
	mu      sync.RWMutex
	apiKeys map[string]APIKeyRecord
}

func NewService() *Service {
	return &Service{
		apiKeys: make(map[string]APIKeyRecord),
	}
}

func (s *Service) Authenticate(apiKey string) error {
	_, err := s.AuthenticateAndGetRecord(apiKey)
	return err
}

func (s *Service) AuthenticateAndGetRecord(apiKey string) (APIKeyRecord, error) {
	trimmed := strings.TrimSpace(apiKey)
	if trimmed == "" {
		return APIKeyRecord{}, ErrInvalidAPIKey
	}

	record, ok := s.APIKeyInfo(trimmed)
	if !ok {
		return APIKeyRecord{}, ErrUnknownAPIKey
	}
	if !record.Active {
		return APIKeyRecord{}, ErrInactiveAPIKey
	}
	if record.Plan.IsZero() {
		return APIKeyRecord{}, ErrMissingPlan
	}
	return record, nil
}

func (s *Service) RegisterAPIKey(apiKey string, plan kernel.PlanID) error {
	trimmedKey := strings.TrimSpace(apiKey)
	if trimmedKey == "" || plan.IsZero() {
		return ErrInvalidAPIKey
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.apiKeys[trimmedKey] = APIKeyRecord{Key: trimmedKey, Plan: plan, Active: true}
	return nil
}

func (s *Service) IsValidAPIKey(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.apiKeys[key]
	return ok && rec.Active
}

func (s *Service) APIKeyInfo(key string) (APIKeyRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.apiKeys[key]
	return rec, ok
}

func (s *Service) CreateAPIKey(plan kernel.PlanID) (string, error) {
	if plan.IsZero() {
		return "", ErrInvalidAPIKey
	}

	key := "demo-" + plan.String() + "-" + randomSuffix(6)
	if err := s.RegisterAPIKey(key, plan); err != nil {
		return "", err
	}
	return key, nil
}

func (s *Service) RevokeAPIKey(apiKey string) error {
	trimmed := strings.TrimSpace(apiKey)
	if trimmed == "" {
		return ErrInvalidAPIKey
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.apiKeys[trimmed]
	if !ok {
		return ErrUnknownAPIKey
	}
	rec.Active = false
	s.apiKeys[trimmed] = rec
	return nil
}

func (m *Module) seedDemoKeys() {
	if m.service == nil {
		return
	}
	for _, item := range []struct {
		plan kernel.PlanID
		key  string
	}{
		{plan: kernel.PlanIDFree, key: "demo-free-key"},
		{plan: kernel.PlanID("basic"), key: "demo-basic-key"},
		{plan: kernel.PlanIDPro, key: "demo-pro-key"},
	} {
		_ = m.service.RegisterAPIKey(item.key, item.plan)
	}
}

func randomSuffix(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
