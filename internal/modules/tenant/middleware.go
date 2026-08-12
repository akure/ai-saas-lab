package tenant

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"aisaaslab/internal/kernel"
	"aisaaslab/internal/modules/auth"
)

type tenantKeyContextKey struct{}

// TenantKeyFromContext extracts the authenticated TenantKey from request context.
func TenantKeyFromContext(ctx context.Context) (kernel.TenantKey, bool) {
	tk, ok := ctx.Value(tenantKeyContextKey{}).(kernel.TenantKey)
	return tk, ok
}

// TenantAuthMiddleware authenticates requests to /v1/tenant/catalog/... and injects TenantKey into Context.
func TenantAuthMiddleware(app *kernel.App, authService *auth.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if apiKey == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				apiKey = strings.TrimSpace(authHeader[7:])
			}
		}

		if apiKey == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": "authentication required: missing X-API-Key or Authorization Bearer header",
			})
			return
		}

		// 1. Policy check
		if err := app.CheckPolicies(r.Context(), apiKey, "valid-api-key"); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": "invalid or revoked api key",
			})
			return
		}

		// 2. Fetch record details from auth service
		rec, err := authService.AuthenticateAndGetRecord(apiKey)
		if err != nil || !rec.Active {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": "api key inactive or invalid",
			})
			return
		}

		// Use the key string as the TenantKey identifier
		tenantKey, err := kernel.NewTenantKey(rec.Key)
		if err != nil {
			tenantKey = kernel.MustTenantKey("tenant-default")
		}

		ctx := context.WithValue(r.Context(), tenantKeyContextKey{}, tenantKey)
		next(w, r.WithContext(ctx))
	}
}
