package auth

import (
	"context"
	"errors"
	"testing"

	"aisaaslab/internal/kernel"
)

func TestService_Authenticate_ValidKey(t *testing.T) {
	store := kernel.NewStore()
	store.SeedAPIKey("valid-key", kernel.PlanIDPro)

	service := NewService(store)
	if err := service.Authenticate("valid-key"); err != nil {
		t.Fatalf("expected valid key to authenticate, got %v", err)
	}
}

func TestService_Authenticate_RejectsEmptyKey(t *testing.T) {
	store := kernel.NewStore()
	service := NewService(store)

	err := service.Authenticate("   ")
	if !errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("expected ErrInvalidAPIKey, got %v", err)
	}
}

func TestService_Authenticate_RejectsUnknownKey(t *testing.T) {
	store := kernel.NewStore()
	service := NewService(store)

	err := service.Authenticate("missing")
	if !errors.Is(err, ErrUnknownAPIKey) {
		t.Fatalf("expected ErrUnknownAPIKey, got %v", err)
	}
}

func TestService_Authenticate_RejectsInactiveKey(t *testing.T) {
	store := kernel.NewStore()
	store.SeedAPIKey("inactive", kernel.PlanID("basic"))
	store.RevokeAPIKey("inactive")

	service := NewService(store)
	err := service.Authenticate("inactive")
	if !errors.Is(err, ErrInactiveAPIKey) {
		t.Fatalf("expected ErrInactiveAPIKey, got %v", err)
	}
}

func TestService_Authenticate_RejectsMissingPlan(t *testing.T) {
	store := kernel.NewStore()
	store.SeedAPIKey("no-plan", "")

	service := NewService(store)
	err := service.Authenticate("no-plan")
	if !errors.Is(err, ErrMissingPlan) {
		t.Fatalf("expected ErrMissingPlan, got %v", err)
	}
}

func TestService_RegisterAPIKey_RejectsEmptyValues(t *testing.T) {
	store := kernel.NewStore()
	service := NewService(store)

	if err := service.RegisterAPIKey("", kernel.PlanIDPro); !errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("expected ErrInvalidAPIKey for empty key, got %v", err)
	}
	if err := service.RegisterAPIKey("key", ""); !errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("expected ErrInvalidAPIKey for empty plan, got %v", err)
	}
}

func TestService_RevokeAPIKey_UnknownKey(t *testing.T) {
	store := kernel.NewStore()
	service := NewService(store)

	err := service.RevokeAPIKey("missing")
	if !errors.Is(err, ErrUnknownAPIKey) {
		t.Fatalf("expected ErrUnknownAPIKey, got %v", err)
	}
}

func TestService_RevokeAPIKey_RevokesExistingKey(t *testing.T) {
	store := kernel.NewStore()
	store.SeedAPIKey("to-revoke", kernel.PlanIDPro)
	service := NewService(store)

	if err := service.RevokeAPIKey("to-revoke"); err != nil {
		t.Fatalf("expected revocation to succeed, got %v", err)
	}
	if store.IsValidAPIKey("to-revoke") {
		t.Fatal("expected revoked key to be invalid")
	}
}

func TestModule_Init_RegistersPolicy(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	module := New()

	if err := module.Init(app); err != nil {
		t.Fatalf("expected init to succeed, got %v", err)
	}

	ctx := context.Background()
	if err := app.CheckPolicies(ctx, "valid-key", "valid-api-key"); err == nil {
		t.Fatal("expected policy to reject an unknown key")
	}

	app.Store.SeedAPIKey("valid-key", kernel.PlanIDPro)
	if err := app.CheckPolicies(ctx, "valid-key", "valid-api-key"); err != nil {
		t.Fatalf("expected valid key to pass policy, got %v", err)
	}
}
