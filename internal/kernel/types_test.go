package kernel_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"aisaaslab/internal/kernel"
)

func TestTenantKey(t *testing.T) {
	_, err := kernel.NewTenantKey("")
	if err == nil {
		t.Error("expected error for empty tenant key")
	}

	tk, err := kernel.NewTenantKey("tenant-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tk.String() != "tenant-123" {
		t.Errorf("expected tenant-123, got %s", tk.String())
	}

	// Test JSON Marshaling & Unmarshaling
	data, err := json.Marshal(tk)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded kernel.TenantKey
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded != tk {
		t.Errorf("expected %v, got %v", tk, decoded)
	}

	// Test Unmarshal Empty String Error
	var emptyDecoded kernel.TenantKey
	if err := json.Unmarshal([]byte(`""`), &emptyDecoded); err != nil {
		t.Errorf("unmarshaling empty string should succeed as zero value: %v", err)
	}
	if !emptyDecoded.IsZero() {
		t.Error("expected zero value TenantKey")
	}
}

func TestAPIKeyRedactionAndSafety(t *testing.T) {
	keyStr := "sk-pro-secret-12345678"
	key, err := kernel.NewAPIKey(keyStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify Stringer interface returns REDACTED key
	loggedOutput := fmt.Sprintf("%s", key)
	if loggedOutput == keyStr {
		t.Fatal("SECURITY RISK: APIKey printed raw secret in fmt.Sprintf!")
	}

	expectedRedacted := "sk-p...5678"
	if loggedOutput != expectedRedacted {
		t.Errorf("expected %s, got %s", expectedRedacted, loggedOutput)
	}

	// Verify Raw() returns exact secret
	if key.Raw() != keyStr {
		t.Errorf("expected raw key %s, got %s", keyStr, key.Raw())
	}

	// JSON roundtrip
	data, err := json.Marshal(key)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded kernel.APIKey
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Raw() != keyStr {
		t.Errorf("expected %s, got %s", keyStr, decoded.Raw())
	}
}

func TestPlanID(t *testing.T) {
	p := kernel.PlanIDPro
	if err := p.Validate(); err != nil {
		t.Errorf("expected valid plan, got %v", err)
	}

	if p.QuotaLimit() != 100000 {
		t.Errorf("expected 100000 quota for pro plan, got %d", p.QuotaLimit())
	}

	invalidPlan := kernel.PlanID("super-ultra")
	if err := invalidPlan.Validate(); err == nil {
		t.Error("expected error for invalid plan")
	}
}
