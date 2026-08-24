package payment

import (
	"context"

	"aisaaslab/internal/kernel"
	"aisaaslab/internal/modules/subscription"
)

// Module integrates the payment engine into the MaaS platform.
type Module struct {
	gateway    Gateway
	subManager *subscription.Manager
}

func New(subManager *subscription.Manager) *Module {
	return &Module{
		gateway:    NewMockGateway(subManager),
		subManager: subManager,
	}
}

func (m *Module) Name() string { return "payment" }

func (m *Module) Gateway() Gateway {
	return m.gateway
}

func (m *Module) Init(app *kernel.App) error {
	// Register HTTP REST routes
	app.Mux.HandleFunc("POST /v1/payment/checkout", m.handleCreateCheckout)
	app.Mux.HandleFunc("GET /v1/payment/sessions/{id}", m.handleGetSession)
	app.Mux.HandleFunc("POST /v1/payment/sessions/{id}/process", m.handleProcessPayment)
	app.Mux.HandleFunc("POST /v1/payment/refunds", m.handleRefundPayment)
	app.Mux.HandleFunc("POST /v1/payment/webhooks/simulate", m.handleSimulateWebhook)
	app.Mux.HandleFunc("GET /v1/payment/transactions/{key}", m.handleListTransactions)

	return nil
}

func (m *Module) Start(ctx context.Context) error { return nil }
func (m *Module) Stop(ctx context.Context) error  { return nil }
