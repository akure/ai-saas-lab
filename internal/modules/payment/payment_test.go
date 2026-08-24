package payment_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aisaaslab/internal/kernel"
	"aisaaslab/internal/modules/payment"
	"aisaaslab/internal/modules/subscription"
)

func TestPaymentModule_EndToEnd(t *testing.T) {
	subModule := subscription.New()
	subManager := subModule.Manager()
	payModule := payment.New(subManager)

	app := kernel.NewApp(&kernel.Config{})
	if err := subModule.Init(app); err != nil {
		t.Fatalf("failed to init subscription module: %v", err)
	}
	if err := payModule.Init(app); err != nil {
		t.Fatalf("failed to init payment module: %v", err)
	}

	ctx := context.Background()
	gw := payModule.Gateway()

	// 1. Create Checkout Session
	sess, err := gw.CreateCheckoutSession(ctx, payment.CreateCheckoutReq{
		TenantKey:    "tenant_pay_1",
		PlanID:       "pro",
		AmountCents:  4900,
		Currency:     "USD",
		BillingCycle: "monthly",
	})
	if err != nil {
		t.Fatalf("failed to create checkout session: %v", err)
	}
	if sess.Status != payment.StatusPending {
		t.Errorf("expected status pending, got %s", sess.Status)
	}

	// 2. Process Payment (Success)
	tx, err := gw.ProcessPayment(ctx, sess.ID, payment.ProcessPaymentReq{
		PaymentMethod: payment.MethodCard,
		SimulateMode:  "succeed",
		AccountHolder: "Test User",
	})
	if err != nil {
		t.Fatalf("failed to process payment: %v", err)
	}
	if tx.Status != payment.StatusSucceeded {
		t.Errorf("expected tx status succeeded, got %s", tx.Status)
	}

	// Verify Subscription FSM changed state to active
	contract, ok := subManager.GetContract("tenant_pay_1")
	if !ok {
		t.Fatal("expected contract to exist after payment")
	}
	if contract.State != subscription.StateActive {
		t.Errorf("expected contract state active, got %s", contract.State)
	}

	// 3. Test HTTP Routes: POST /v1/payment/checkout
	body, _ := json.Marshal(payment.CreateCheckoutReq{
		TenantKey:    "tenant_pay_2",
		PlanID:       "enterprise",
		AmountCents:  29900,
		Currency:     "USD",
		BillingCycle: "yearly",
	})
	req := httptest.NewRequest("POST", "/v1/payment/checkout", bytes.NewReader(body))
	w := httptest.NewRecorder()
	app.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var createdSess payment.CheckoutSession
	if err := json.Unmarshal(w.Body.Bytes(), &createdSess); err != nil {
		t.Fatalf("failed to parse checkout response: %v", err)
	}

	// 4. Test Webhook Simulation: payment.failed
	webBody, _ := json.Marshal(payment.SimulateWebhookReq{
		Event:     "payment.failed",
		TenantKey: "tenant_pay_1",
	})
	reqWeb := httptest.NewRequest("POST", "/v1/payment/webhooks/simulate", bytes.NewReader(webBody))
	wWeb := httptest.NewRecorder()
	app.Mux.ServeHTTP(wWeb, reqWeb)

	if wWeb.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for webhook, got %d: %s", wWeb.Code, wWeb.Body.String())
	}

	// Verify state transitioned to past_due
	contractPast, _ := subManager.GetContract("tenant_pay_1")
	if contractPast.State != subscription.StatePastDue {
		t.Errorf("expected state past_due after payment.failed webhook, got %s", contractPast.State)
	}
}
