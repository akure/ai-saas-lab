package payment

import (
	"time"
)

// SessionStatus represents the state of a payment checkout session.
type SessionStatus string

const (
	StatusPending         SessionStatus = "pending"
	StatusRequiresAction  SessionStatus = "requires_action"
	StatusSucceeded       SessionStatus = "succeeded"
	StatusFailed          SessionStatus = "failed"
	StatusCancelled       SessionStatus = "cancelled"
)

// PaymentMethod specifies the generic type of payment instrument used.
type PaymentMethod string

const (
	MethodCard         PaymentMethod = "card"
	MethodBankTransfer PaymentMethod = "bank_transfer"
	MethodWallet       PaymentMethod = "digital_wallet"
	MethodCryptoToken  PaymentMethod = "crypto_token"
)

// CheckoutSession represents an active or completed payment session.
type CheckoutSession struct {
	ID            string        `json:"id"`
	TenantKey     string        `json:"tenant_key"`
	PlanID        string        `json:"plan_id"`
	AmountCents   int64         `json:"amount_cents"`
	Currency      string        `json:"currency"`
	BillingCycle  string        `json:"billing_cycle"` // "monthly", "yearly"
	Status        SessionStatus `json:"status"`
	PaymentMethod PaymentMethod `json:"payment_method,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	ErrorReason   string        `json:"error_reason,omitempty"`
}

// Transaction records a completed payment ledger entry.
type Transaction struct {
	ID            string        `json:"id"`
	SessionID     string        `json:"session_id"`
	TenantKey     string        `json:"tenant_key"`
	PlanID        string        `json:"plan_id"`
	AmountCents   int64         `json:"amount_cents"`
	Currency      string        `json:"currency"`
	PaymentMethod PaymentMethod `json:"payment_method"`
	Status        SessionStatus `json:"status"`
	ReferenceID   string        `json:"reference_id"`
	ProcessedAt   time.Time     `json:"processed_at"`
	FailureReason string        `json:"failure_reason,omitempty"`
}

// CreateCheckoutReq DTO to initialize a payment checkout session.
type CreateCheckoutReq struct {
	TenantKey    string `json:"tenant_key"`
	PlanID       string `json:"plan_id"`
	AmountCents  int64  `json:"amount_cents"`
	Currency     string `json:"currency"`
	BillingCycle string `json:"billing_cycle"`
}

// ProcessPaymentReq DTO to submit a payment attempt against a checkout session.
type ProcessPaymentReq struct {
	PaymentMethod PaymentMethod `json:"payment_method"`
	SimulateMode  string        `json:"simulate_mode"` // "auto", "succeed", "decline", "challenge"
	AccountHolder string        `json:"account_holder,omitempty"`
	TokenRef      string        `json:"token_ref,omitempty"` // Masked card / account ref / wallet hash
	ChallengeCode string        `json:"challenge_code,omitempty"` // For verification challenges
}

// SimulateWebhookReq DTO to trigger an asynchronous provider webhook event.
type SimulateWebhookReq struct {
	Event     string `json:"event"` // "payment.succeeded", "payment.failed", "subscription.cancelled"
	TenantKey string `json:"tenant_key"`
	SessionID string `json:"session_id,omitempty"`
	PlanID    string `json:"plan_id,omitempty"`
}

// WebhookResult summarizes the outcome of a processed mock webhook event.
type WebhookResult struct {
	Event       string `json:"event"`
	TenantKey   string `json:"tenant_key"`
	FSMEffect   string `json:"fsm_effect"`
	ProcessedAt time.Time `json:"processed_at"`
}
