package payment

import (
	"context"
)

// Gateway defines the clean, vendor-decoupled contract for payment processing.
// Open-source adopters can implement this interface for any commercial payment provider.
type Gateway interface {
	CreateCheckoutSession(ctx context.Context, req CreateCheckoutReq) (*CheckoutSession, error)
	GetCheckoutSession(ctx context.Context, id string) (*CheckoutSession, bool)
	ProcessPayment(ctx context.Context, sessionID string, req ProcessPaymentReq) (*Transaction, error)
	SimulateWebhook(ctx context.Context, req SimulateWebhookReq) (*WebhookResult, error)
	ListTransactions(ctx context.Context, tenantKey string) ([]Transaction, error)
}
