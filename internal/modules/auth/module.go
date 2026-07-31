package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"aisaaslab/internal/kernel"
)

var (
	ErrInvalidAPIKey  = errors.New("invalid api key")
	ErrUnknownAPIKey  = errors.New("unknown api key")
	ErrInactiveAPIKey = errors.New("inactive api key")
	ErrMissingPlan    = errors.New("missing api key plan")
)

// Module registers a policy other modules can require by name. It never
// exposes an HTTP route itself — pure policy provider.
type Module struct {
	service *Service
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "auth" }

func (m *Module) Init(app *kernel.App) error {
	m.service = NewService(app.Store)
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
		Plan string `json:"plan"`
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

// Service provides MVP-friendly API key authentication logic for modules that
// need to validate credentials independently of the rest of the application.
type Service struct {
	store *kernel.Store
}

func NewService(store *kernel.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Authenticate(apiKey string) error {
	if s.store == nil {
		return ErrInvalidAPIKey
	}
	trimmed := strings.TrimSpace(apiKey)
	if trimmed == "" {
		return ErrInvalidAPIKey
	}

	record, ok := s.store.APIKeyInfo(trimmed)
	if !ok {
		return ErrUnknownAPIKey
	}
	if !record.Active {
		return ErrInactiveAPIKey
	}
	if strings.TrimSpace(record.Plan) == "" {
		return ErrMissingPlan
	}
	return nil
}

func (s *Service) RegisterAPIKey(apiKey, plan string) error {
	if s.store == nil {
		return ErrInvalidAPIKey
	}
	trimmedKey := strings.TrimSpace(apiKey)
	trimmedPlan := strings.TrimSpace(plan)
	if trimmedKey == "" || trimmedPlan == "" {
		return ErrInvalidAPIKey
	}
	s.store.SeedAPIKey(trimmedKey, trimmedPlan)
	return nil
}

func (s *Service) CreateAPIKey(plan string) (string, error) {
	if s.store == nil {
		return "", ErrInvalidAPIKey
	}
	trimmedPlan := strings.TrimSpace(plan)
	if trimmedPlan == "" {
		return "", ErrInvalidAPIKey
	}

	key := "demo-" + trimmedPlan + "-" + randomSuffix(6)
	if err := s.RegisterAPIKey(key, trimmedPlan); err != nil {
		return "", err
	}
	return key, nil
}

func (m *Module) seedDemoKeys() {
	if m.service == nil {
		return
	}
	for _, item := range []struct {
		plan string
		key  string
	}{
		{plan: "free", key: "demo-free-key"},
		{plan: "basic", key: "demo-basic-key"},
		{plan: "pro", key: "demo-pro-key"},
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

func (s *Service) RevokeAPIKey(apiKey string) error {
	if s.store == nil {
		return ErrInvalidAPIKey
	}
	trimmed := strings.TrimSpace(apiKey)
	if trimmed == "" {
		return ErrInvalidAPIKey
	}
	_, ok := s.store.APIKeyInfo(trimmed)
	if !ok {
		return ErrUnknownAPIKey
	}
	s.store.RevokeAPIKey(trimmed)
	return nil
}
