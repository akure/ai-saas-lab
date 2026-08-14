package tenant

import (
	"encoding/json"
	"errors"
	"net/http"

	"aisaaslab/internal/kernel"
)

// maxBodyBytes caps incoming request bodies to prevent memory abuse.
const maxBodyBytes = 64 * 1024 // 64 KB

// Handlers exposes HTTP endpoints for the tenant self-service catalog.
type Handlers struct {
	service *Service
}

func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// writeError maps a CatalogError (or any error) to the correct HTTP status and
// writes a consistent JSON error envelope. Content-Type is set by TenantAuthMiddleware.
func writeError(w http.ResponseWriter, err error) {
	code := http.StatusBadRequest
	switch {
	case kernel.IsCatalogConflict(err):
		code = http.StatusConflict // 409
	case kernel.IsCatalogValidation(err):
		code = http.StatusUnprocessableEntity // 422
	case kernel.IsCatalogNotFound(err):
		code = http.StatusNotFound // 404
	case kernel.IsCatalogBackend(err):
		code = http.StatusServiceUnavailable // 503
	default:
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			code = http.StatusRequestEntityTooLarge // 413
		}
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
}

// echoRequestID propagates X-Request-ID from request → response header.
func echoRequestID(w http.ResponseWriter, r *http.Request) {
	if rid := r.Header.Get("X-Request-ID"); rid != "" {
		w.Header().Set("X-Request-ID", rid)
	}
}

// limitBody wraps the request body with a 64 KB max-bytes reader.
func limitBody(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
}

// ---------------------------------------------------------------------------
// POST /v1/tenant/catalog/services  — Register
// GET  /v1/tenant/catalog/services  — List
// ---------------------------------------------------------------------------

func (h *Handlers) HandleServices(w http.ResponseWriter, r *http.Request) {
	tenantKey, ok := TenantKeyFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "tenant context missing"})
		return
	}
	echoRequestID(w, r)

	switch r.Method {
	case http.MethodPost:
		limitBody(w, r)
		var descriptor kernel.TenantServiceDescriptor
		if err := json.NewDecoder(r.Body).Decode(&descriptor); err != nil {
			writeError(w, err)
			return
		}
		res, err := h.service.RegisterService(r.Context(), tenantKey, descriptor)
		if err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(res)

	case http.MethodGet:
		list, err := h.service.ListServices(r.Context(), tenantKey)
		if err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(list)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
	}
}

// GET /v1/tenant/catalog/services/{id}
func (h *Handlers) HandleGetService(w http.ResponseWriter, r *http.Request) {
	tenantKey, ok := TenantKeyFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "tenant context missing"})
		return
	}
	echoRequestID(w, r)

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
		return
	}

	id := kernel.ServiceID(r.PathValue("id"))
	svc, err := h.service.GetService(r.Context(), tenantKey, id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(svc)
}

// ---------------------------------------------------------------------------
// POST /v1/tenant/catalog/metrics  — Register
// GET  /v1/tenant/catalog/metrics  — List
// ---------------------------------------------------------------------------

func (h *Handlers) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	tenantKey, ok := TenantKeyFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "tenant context missing"})
		return
	}
	echoRequestID(w, r)

	switch r.Method {
	case http.MethodPost:
		limitBody(w, r)
		var descriptor kernel.TenantMetricDescriptor
		if err := json.NewDecoder(r.Body).Decode(&descriptor); err != nil {
			writeError(w, err)
			return
		}
		res, err := h.service.RegisterMetric(r.Context(), tenantKey, descriptor)
		if err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(res)

	case http.MethodGet:
		list, err := h.service.ListMetrics(r.Context(), tenantKey)
		if err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(list)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
	}
}

// GET /v1/tenant/catalog/metrics/{id}
func (h *Handlers) HandleGetMetric(w http.ResponseWriter, r *http.Request) {
	tenantKey, ok := TenantKeyFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "tenant context missing"})
		return
	}
	echoRequestID(w, r)

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
		return
	}

	id := kernel.MetricID(r.PathValue("id"))
	m, err := h.service.GetMetric(r.Context(), tenantKey, id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(m)
}

// ---------------------------------------------------------------------------
// POST /v1/tenant/catalog/plans  — Register
// GET  /v1/tenant/catalog/plans  — List
// ---------------------------------------------------------------------------

func (h *Handlers) HandlePlans(w http.ResponseWriter, r *http.Request) {
	tenantKey, ok := TenantKeyFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "tenant context missing"})
		return
	}
	echoRequestID(w, r)

	switch r.Method {
	case http.MethodPost:
		limitBody(w, r)
		var descriptor kernel.TenantPlanDescriptor
		if err := json.NewDecoder(r.Body).Decode(&descriptor); err != nil {
			writeError(w, err)
			return
		}
		res, err := h.service.RegisterPlan(r.Context(), tenantKey, descriptor)
		if err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(res)

	case http.MethodGet:
		list, err := h.service.ListPlans(r.Context(), tenantKey)
		if err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(list)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
	}
}

// GET /v1/tenant/catalog/plans/{id}
func (h *Handlers) HandleGetPlan(w http.ResponseWriter, r *http.Request) {
	tenantKey, ok := TenantKeyFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "tenant context missing"})
		return
	}
	echoRequestID(w, r)

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
		return
	}

	id := kernel.ApplicationPlanID(r.PathValue("id"))
	p, err := h.service.GetPlan(r.Context(), tenantKey, id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(p)
}

// ---------------------------------------------------------------------------
// GET /v1/tenant/catalog/overview
// ---------------------------------------------------------------------------

func (h *Handlers) HandleOverview(w http.ResponseWriter, r *http.Request) {
	tenantKey, ok := TenantKeyFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "tenant context missing"})
		return
	}
	echoRequestID(w, r)

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
		return
	}

	overview, err := h.service.Overview(r.Context(), tenantKey)
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(overview)
}
