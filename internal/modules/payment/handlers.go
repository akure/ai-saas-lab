package payment

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (m *Module) handleCreateCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		writeJSONError(w, "request body is required", http.StatusBadRequest)
		return
	}
	var req CreateCheckoutReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid json request body", http.StatusBadRequest)
		return
	}

	session, err := m.gateway.CreateCheckoutSession(r.Context(), req)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(session)
}

func (m *Module) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeJSONError(w, "session id parameter is required", http.StatusBadRequest)
		return
	}

	session, ok := m.gateway.GetCheckoutSession(r.Context(), id)
	if !ok {
		writeJSONError(w, "checkout session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(session)
}

func (m *Module) handleProcessPayment(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeJSONError(w, "session id parameter is required", http.StatusBadRequest)
		return
	}

	if r.Body == nil {
		writeJSONError(w, "request body is required", http.StatusBadRequest)
		return
	}

	var req ProcessPaymentReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			writeJSONError(w, "request body cannot be empty", http.StatusBadRequest)
			return
		}
		writeJSONError(w, "invalid json request body", http.StatusBadRequest)
		return
	}

	tx, err := m.gateway.ProcessPayment(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			writeJSONError(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrSessionAlreadyDone) {
			writeJSONError(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, ErrVerificationReq) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":         string(StatusRequiresAction),
				"message":        err.Error(),
				"challenge_type": "otp_verification",
			})
			return
		}
		if errors.Is(err, ErrPaymentDeclined) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			_ = json.NewEncoder(w).Encode(tx)
			return
		}
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tx)
}

func (m *Module) handleRefundPayment(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		writeJSONError(w, "request body is required", http.StatusBadRequest)
		return
	}

	var req RefundReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid json request body", http.StatusBadRequest)
		return
	}

	tx, err := m.gateway.RefundPayment(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrTransactionNotFound) {
			writeJSONError(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrCannotRefund) {
			writeJSONError(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tx)
}

func (m *Module) handleSimulateWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		writeJSONError(w, "request body is required", http.StatusBadRequest)
		return
	}

	var req SimulateWebhookReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid json request body", http.StatusBadRequest)
		return
	}

	res, err := m.gateway.SimulateWebhook(r.Context(), req)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (m *Module) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.PathValue("key"))
	if key == "" {
		writeJSONError(w, "tenant key parameter is required", http.StatusBadRequest)
		return
	}

	txs, err := m.gateway.ListTransactions(r.Context(), key)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"tenant_key":   key,
		"count":        len(txs),
		"transactions": txs,
	})
}
