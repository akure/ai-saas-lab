package payment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"aisaaslab/internal/modules/subscription"
)

var (
	ErrSessionNotFound     = errors.New("checkout session not found")
	ErrSessionAlreadyDone  = errors.New("checkout session is already completed")
	ErrInvalidAmount       = errors.New("amount must be greater than zero")
	ErrPaymentDeclined     = errors.New("payment attempt declined by sandbox issuer")
	ErrVerificationReq     = errors.New("multi-factor payment verification challenge required")
	ErrUnsupportedMethod   = errors.New("unsupported payment method requested")
	ErrTransactionNotFound = errors.New("transaction record not found")
	ErrCannotRefund        = errors.New("only successful transactions can be refunded")
)

// MockGateway implements the vendor-decoupled Gateway interface with rich testing simulation.
type MockGateway struct {
	mu           sync.RWMutex
	sessions     map[string]*CheckoutSession
	transactions map[string][]Transaction
	subManager   *subscription.Manager
}

// NewMockGateway initializes a thread-safe MockGateway instance.
func NewMockGateway(subManager *subscription.Manager) *MockGateway {
	return &MockGateway{
		sessions:     make(map[string]*CheckoutSession),
		transactions: make(map[string][]Transaction),
		subManager:   subManager,
	}
}

func generateID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

func (g *MockGateway) CreateCheckoutSession(ctx context.Context, req CreateCheckoutReq) (*CheckoutSession, error) {
	req.TenantKey = strings.TrimSpace(req.TenantKey)
	req.PlanID = strings.TrimSpace(req.PlanID)
	if req.TenantKey == "" || req.PlanID == "" {
		return nil, errors.New("tenant_key and plan_id are required")
	}
	if req.AmountCents < 0 {
		return nil, ErrInvalidAmount
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.BillingCycle == "" {
		req.BillingCycle = "monthly"
	}

	session := &CheckoutSession{
		ID:           generateID("cs"),
		TenantKey:    req.TenantKey,
		PlanID:       req.PlanID,
		AmountCents:  req.AmountCents,
		Currency:     strings.ToUpper(req.Currency),
		BillingCycle: req.BillingCycle,
		Status:       StatusPending,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	g.mu.Lock()
	g.sessions[session.ID] = session
	g.mu.Unlock()

	return session, nil
}

func (g *MockGateway) GetCheckoutSession(ctx context.Context, id string) (*CheckoutSession, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	sess, ok := g.sessions[id]
	if !ok {
		return nil, false
	}
	cp := *sess
	return &cp, true
}

func (g *MockGateway) ProcessPayment(ctx context.Context, sessionID string, req ProcessPaymentReq) (*Transaction, error) {
	g.mu.Lock()
	sess, ok := g.sessions[sessionID]
	if !ok {
		g.mu.Unlock()
		return nil, ErrSessionNotFound
	}

	if sess.Status == StatusSucceeded || sess.Status == StatusFailed {
		g.mu.Unlock()
		return nil, ErrSessionAlreadyDone
	}

	mode := strings.ToLower(strings.TrimSpace(req.SimulateMode))
	if mode == "" {
		mode = "auto"
	}

	// Validate generic payment method
	switch req.PaymentMethod {
	case MethodCard, MethodBankTransfer, MethodWallet, MethodCryptoToken:
		sess.PaymentMethod = req.PaymentMethod
	default:
		g.mu.Unlock()
		return nil, ErrUnsupportedMethod
	}

	// Handle 3DS / Challenge state
	if mode == "challenge" && req.ChallengeCode != "123456" {
		sess.Status = StatusRequiresAction
		sess.UpdatedAt = time.Now().UTC()
		g.mu.Unlock()
		return nil, ErrVerificationReq
	}

	now := time.Now().UTC()
	tx := Transaction{
		ID:            generateID("txn"),
		SessionID:     sess.ID,
		TenantKey:     sess.TenantKey,
		PlanID:        sess.PlanID,
		AmountCents:   sess.AmountCents,
		Currency:      sess.Currency,
		PaymentMethod: req.PaymentMethod,
		ReferenceID:   generateID("ref"),
		ProcessedAt:   now,
	}

	if mode == "decline" {
		sess.Status = StatusFailed
		sess.ErrorReason = "decline_card_insufficient_funds"
		sess.UpdatedAt = now
		tx.Status = StatusFailed
		tx.FailureReason = sess.ErrorReason

		g.transactions[sess.TenantKey] = append(g.transactions[sess.TenantKey], tx)
		g.mu.Unlock()

		if g.subManager != nil {
			_, _ = g.subManager.FireTransition(sess.TenantKey, "payment_failed")
		}
		return &tx, ErrPaymentDeclined
	}

	// Success flow
	sess.Status = StatusSucceeded
	sess.UpdatedAt = now
	tx.Status = StatusSucceeded

	g.transactions[sess.TenantKey] = append(g.transactions[sess.TenantKey], tx)
	g.mu.Unlock()

	// Connect with Subscription Manager
	if g.subManager != nil {
		contract, exists := g.subManager.GetContract(sess.TenantKey)
		if !exists || contract.PlanID != sess.PlanID {
			g.subManager.CreateContract(subscription.Contract{
				SubscriptionID: "sub_" + sess.TenantKey,
				TenantKey:      sess.TenantKey,
				PlanID:         sess.PlanID,
				State:          subscription.StateActive,
				AnchorTime:     now,
				Timezone:       "UTC",
			})
		} else {
			_, _ = g.subManager.FireTransition(sess.TenantKey, "payment_succeeded")
		}
	}

	return &tx, nil
}

func (g *MockGateway) RefundPayment(ctx context.Context, req RefundReq) (*Transaction, error) {
	req.TransactionID = strings.TrimSpace(req.TransactionID)
	if req.TransactionID == "" {
		return nil, errors.New("transaction_id is required for refund")
	}

	g.mu.Lock()
	var targetTx *Transaction
	var tenantKey string

	for key, txList := range g.transactions {
		for i, tx := range txList {
			if tx.ID == req.TransactionID || tx.ReferenceID == req.TransactionID {
				if tx.Status != StatusSucceeded {
					g.mu.Unlock()
					return nil, ErrCannotRefund
				}
				g.transactions[key][i].Status = StatusRefunded
				targetTx = &g.transactions[key][i]
				tenantKey = key
				break
			}
		}
		if targetTx != nil {
			break
		}
	}

	if targetTx == nil {
		g.mu.Unlock()
		return nil, ErrTransactionNotFound
	}

	now := time.Now().UTC()
	refundTx := Transaction{
		ID:            generateID("ref_tx"),
		SessionID:     targetTx.SessionID,
		TenantKey:     tenantKey,
		PlanID:        targetTx.PlanID,
		AmountCents:   targetTx.AmountCents,
		Currency:      targetTx.Currency,
		PaymentMethod: targetTx.PaymentMethod,
		Status:        StatusRefunded,
		ReferenceID:   generateID("ref"),
		ProcessedAt:   now,
		FailureReason: req.Reason,
	}

	g.transactions[tenantKey] = append(g.transactions[tenantKey], refundTx)
	g.mu.Unlock()

	if g.subManager != nil {
		_, _ = g.subManager.FireTransition(tenantKey, "payment_failed")
	}

	return &refundTx, nil
}

func (g *MockGateway) SimulateWebhook(ctx context.Context, req SimulateWebhookReq) (*WebhookResult, error) {
	req.TenantKey = strings.TrimSpace(req.TenantKey)
	req.Event = strings.TrimSpace(req.Event)

	if req.TenantKey == "" || req.Event == "" {
		return nil, errors.New("tenant_key and event are required")
	}

	fsmEffect := "none"
	now := time.Now().UTC()

	if g.subManager != nil {
		switch req.Event {
		case "payment.succeeded", "invoice.paid", "subscription.renewed":
			if req.PlanID != "" {
				g.subManager.CreateContract(subscription.Contract{
					SubscriptionID: "sub_" + req.TenantKey,
					TenantKey:      req.TenantKey,
					PlanID:         req.PlanID,
					State:          subscription.StateActive,
					AnchorTime:     now,
					Timezone:       "UTC",
				})
				fsmEffect = "contract_created_active"
			} else {
				st, err := g.subManager.FireTransition(req.TenantKey, "payment_succeeded")
				if err == nil {
					fsmEffect = "transitioned_to_" + string(st)
				}
			}
		case "payment.failed", "invoice.payment_failed":
			st, err := g.subManager.FireTransition(req.TenantKey, "payment_failed")
			if err == nil {
				fsmEffect = "transitioned_to_" + string(st)
			}
		case "subscription.cancelled":
			st, err := g.subManager.FireTransition(req.TenantKey, "cancel")
			if err == nil {
				fsmEffect = "transitioned_to_" + string(st)
			}
		default:
			return nil, fmt.Errorf("unsupported webhook event type: %s", req.Event)
		}
	}

	return &WebhookResult{
		Event:       req.Event,
		TenantKey:   req.TenantKey,
		FSMEffect:   fsmEffect,
		ProcessedAt: now,
	}, nil
}

func (g *MockGateway) ListTransactions(ctx context.Context, tenantKey string) ([]Transaction, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	txs, ok := g.transactions[tenantKey]
	if !ok {
		return []Transaction{}, nil
	}
	res := make([]Transaction, len(txs))
	copy(res, txs)
	return res, nil
}
