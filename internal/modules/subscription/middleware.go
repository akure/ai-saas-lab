package subscription

import (
	"encoding/json"
	"net/http"
	"strings"

	"aisaaslab/internal/kernel"
)

// RequireActiveSubscription creates an HTTP middleware that checks if the tenant has an active/usable subscription.
func RequireActiveSubscription(app *kernel.App, manager *Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantKey := extractTenantKey(r)
			if tenantKey == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing tenant key header or parameter"})
				return
			}

			if app != nil {
				if err := app.CheckPolicies(r.Context(), tenantKey, "has-active-subscription"); err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
					return
				}
			} else if manager != nil {
				st := manager.GetState(tenantKey)
				if !st.IsUsable() {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "subscription is inactive or cancelled"})
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireEntitlement creates an HTTP middleware that checks if the tenant is under quota for a specific metric.
func RequireEntitlement(app *kernel.App, manager *Manager, metricID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantKey := extractTenantKey(r)
			if tenantKey == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing tenant key header or parameter"})
				return
			}

			if app != nil {
				if err := app.CheckPolicies(r.Context(), tenantKey, "under-subscription-quota"); err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusPaymentRequired)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
					return
				}
			} else if manager != nil {
				ok, err := manager.CheckEntitlement(tenantKey, metricID, 0)
				if !ok || err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusPaymentRequired)
					errMsg := "quota exceeded or metric disallowed"
					if err != nil {
						errMsg = err.Error()
					}
					_ = json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractTenantKey(r *http.Request) string {
	if key := r.Header.Get("X-Tenant-Key"); key != "" {
		return strings.TrimSpace(key)
	}
	if ctxVal, ok := r.Context().Value("tenant_key").(string); ok && ctxVal != "" {
		return strings.TrimSpace(ctxVal)
	}
	if key := r.URL.Query().Get("tenant_key"); key != "" {
		return strings.TrimSpace(key)
	}
	if key := r.URL.Query().Get("api_key"); key != "" {
		return strings.TrimSpace(key)
	}
	return ""
}
