package tenant

import (
	"encoding/json"
	"net/http"

	"aisaaslab/internal/kernel"
)

type Handlers struct {
	service *Service
}

func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// HandleServices: POST (Register) / GET (List) /v1/tenant/catalog/services
func (h *Handlers) HandleServices(w http.ResponseWriter, r *http.Request) {
	tenantKey, ok := TenantKeyFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "tenant context missing"})
		return
	}

	switch r.Method {
	case http.MethodPost:
		var descriptor kernel.TenantServiceDescriptor
		if err := json.NewDecoder(r.Body).Decode(&descriptor); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid json body: " + err.Error()})
			return
		}
		res, err := h.service.RegisterService(r.Context(), tenantKey, descriptor)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(res)

	case http.MethodGet:
		list, err := h.service.ListServices(r.Context(), tenantKey)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(list)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
	}
}

// HandleMetrics: POST (Register) / GET (List) /v1/tenant/catalog/metrics
func (h *Handlers) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	tenantKey, ok := TenantKeyFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "tenant context missing"})
		return
	}

	switch r.Method {
	case http.MethodPost:
		var descriptor kernel.TenantMetricDescriptor
		if err := json.NewDecoder(r.Body).Decode(&descriptor); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid json body: " + err.Error()})
			return
		}
		res, err := h.service.RegisterMetric(r.Context(), tenantKey, descriptor)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(res)

	case http.MethodGet:
		list, err := h.service.ListMetrics(r.Context(), tenantKey)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(list)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
	}
}

// HandlePlans: POST (Register) / GET (List) /v1/tenant/catalog/plans
func (h *Handlers) HandlePlans(w http.ResponseWriter, r *http.Request) {
	tenantKey, ok := TenantKeyFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "tenant context missing"})
		return
	}

	switch r.Method {
	case http.MethodPost:
		var descriptor kernel.TenantPlanDescriptor
		if err := json.NewDecoder(r.Body).Decode(&descriptor); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid json body: " + err.Error()})
			return
		}
		res, err := h.service.RegisterPlan(r.Context(), tenantKey, descriptor)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(res)

	case http.MethodGet:
		list, err := h.service.ListPlans(r.Context(), tenantKey)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(list)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
	}
}

// HandleOverview: GET /v1/tenant/catalog/overview
func (h *Handlers) HandleOverview(w http.ResponseWriter, r *http.Request) {
	tenantKey, ok := TenantKeyFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "tenant context missing"})
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
		return
	}

	overview, err := h.service.Overview(r.Context(), tenantKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(overview)
}
