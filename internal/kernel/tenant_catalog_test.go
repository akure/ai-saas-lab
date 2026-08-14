package kernel

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Validation edge-case tests for ServiceID and TenantServiceDescriptor
// ---------------------------------------------------------------------------

func TestServiceID_Validate_EdgeCases(t *testing.T) {
	cases := []struct {
		name    string
		id      ServiceID
		wantErr bool
		errFrag string
	}{
		{"valid simple", "ai-writer", false, ""},
		{"valid single char", "a", false, ""},
		{"valid with underscore", "ai_writer_v2", false, ""},
		{"empty", "", true, "empty"},
		{"too long (65 chars)", ServiceID(strings.Repeat("a", 65)), true, "64"},
		{"leading hyphen", "-bad-id", true, "invalid"},
		{"trailing hyphen", "bad-id-", true, "invalid"},
		{"uppercase letter", "AiWriter", true, "invalid"},
		{"spaces", "ai writer", true, "invalid"},
		{"special chars", "ai@writer!", true, "invalid"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.id.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected error for %q but got nil", tc.id)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error for %q but got: %v", tc.id, err)
			}
			if tc.wantErr && tc.errFrag != "" && !strings.Contains(err.Error(), tc.errFrag) {
				t.Errorf("expected error containing %q, got: %v", tc.errFrag, err)
			}
		})
	}
}

func TestTenantServiceDescriptor_Validate(t *testing.T) {
	cases := []struct {
		name    string
		d       TenantServiceDescriptor
		wantErr bool
		errFrag string
	}{
		{
			name:    "valid",
			d:       TenantServiceDescriptor{ServiceID: "ai-writer", Name: "AI Writer"},
			wantErr: false,
		},
		{
			name:    "empty service_id",
			d:       TenantServiceDescriptor{ServiceID: "", Name: "AI Writer"},
			wantErr: true,
			errFrag: "service_id",
		},
		{
			name:    "empty name",
			d:       TenantServiceDescriptor{ServiceID: "ai-writer", Name: ""},
			wantErr: true,
			errFrag: "name",
		},
		{
			name:    "name too long (129 chars)",
			d:       TenantServiceDescriptor{ServiceID: "ai-writer", Name: strings.Repeat("x", 129)},
			wantErr: true,
			errFrag: "128",
		},
		{
			name:    "description too long (513 chars)",
			d:       TenantServiceDescriptor{ServiceID: "ai-writer", Name: "AI", Description: strings.Repeat("d", 513)},
			wantErr: true,
			errFrag: "512",
		},
		{
			name:    "whitespace-only name",
			d:       TenantServiceDescriptor{ServiceID: "ai-writer", Name: "   "},
			wantErr: true,
			errFrag: "name",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.d.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if tc.wantErr && tc.errFrag != "" && !strings.Contains(err.Error(), tc.errFrag) {
				t.Errorf("error %q should contain %q", err.Error(), tc.errFrag)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// MemoryStore — multi-tenant isolation and duplicate handling
// ---------------------------------------------------------------------------

func TestMemoryStore_MultiTenantIsolation(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryTenantCatalogStore()

	tenantA := MustTenantKey("tenant-a")
	tenantB := MustTenantKey("tenant-b")

	svc := TenantServiceDescriptor{ServiceID: "shared-id", Name: "Service"}
	if err := store.RegisterService(ctx, tenantA, svc); err != nil {
		t.Fatalf("register for A: %v", err)
	}

	// Tenant B should NOT see tenant A's service.
	_, found, err := store.GetService(ctx, tenantB, "shared-id")
	if err != nil || found {
		t.Errorf("tenant B should not see tenant A's service: found=%v err=%v", found, err)
	}

	// Tenant A should see it.
	_, found, err = store.GetService(ctx, tenantA, "shared-id")
	if err != nil || !found {
		t.Errorf("tenant A should see its own service: found=%v err=%v", found, err)
	}
}

func TestMemoryStore_DuplicateServiceDoesNotErrorAtStoreLevel(t *testing.T) {
	// Note: duplicate detection is enforced at the service layer, NOT the store.
	// The memory store uses upsert semantics (map overwrite).
	ctx := context.Background()
	store := NewMemoryTenantCatalogStore()
	tenant := MustTenantKey("acme")

	svc1 := TenantServiceDescriptor{ServiceID: "svc1", Name: "Original"}
	if err := store.RegisterService(ctx, tenant, svc1); err != nil {
		t.Fatalf("register: %v", err)
	}

	svc1Updated := TenantServiceDescriptor{ServiceID: "svc1", Name: "Updated"}
	if err := store.RegisterService(ctx, tenant, svc1Updated); err != nil {
		t.Fatalf("upsert should not error at store level: %v", err)
	}

	got, found, err := store.GetService(ctx, tenant, "svc1")
	if err != nil || !found {
		t.Fatalf("get: %v %v", found, err)
	}
	if got.Name != "Updated" {
		t.Errorf("expected upserted name 'Updated', got %q", got.Name)
	}
}

// ---------------------------------------------------------------------------
// CatalogError helpers
// ---------------------------------------------------------------------------

func TestCatalogError_SentinelMatching(t *testing.T) {
	conflict := NewConflictError("service", "svc-x")
	if !IsCatalogConflict(conflict) {
		t.Error("conflict error should pass IsCatalogConflict")
	}
	if !errors.Is(conflict, ErrServiceAlreadyExists) {
		t.Error("conflict error should unwrap to ErrServiceAlreadyExists")
	}

	validation := NewValidationError("service_id", "must not be empty")
	if !IsCatalogValidation(validation) {
		t.Error("validation error should pass IsCatalogValidation")
	}
	if !errors.Is(validation, ErrCatalogValidation) {
		t.Error("validation error should unwrap to ErrCatalogValidation")
	}

	backend := NewBackendError("postgres", errors.New("connection refused"))
	if !IsCatalogBackend(backend) {
		t.Error("backend error should pass IsCatalogBackend")
	}
	if !errors.Is(backend, ErrCatalogBackendUnavailable) {
		t.Error("backend error should unwrap to ErrCatalogBackendUnavailable")
	}

	notFound := NewNotFoundError("service", "missing-id")
	if !IsCatalogNotFound(notFound) {
		t.Error("not-found error should pass IsCatalogNotFound")
	}
	if !errors.Is(notFound, ErrServiceNotFound) {
		t.Error("not-found error should unwrap to ErrServiceNotFound")
	}
}

// ---------------------------------------------------------------------------
// CatalogChain — unit tests with mock/memory backends
// ---------------------------------------------------------------------------

func TestCatalogChain_WriteThrough_SingleBackend(t *testing.T) {
	ctx := context.Background()
	mem := NewMemoryTenantCatalogStore()
	chain, err := NewCatalogChain([]TenantCatalogStore{mem}, nil)
	if err != nil {
		t.Fatalf("new chain: %v", err)
	}
	defer chain.Close()

	tenant := MustTenantKey("chain-tenant")
	svc := TenantServiceDescriptor{ServiceID: "chain-svc", Name: "Chain Service"}

	if err := chain.RegisterService(ctx, tenant, svc); err != nil {
		t.Fatalf("register: %v", err)
	}

	got, found, err := chain.GetService(ctx, tenant, "chain-svc")
	if err != nil || !found {
		t.Fatalf("get: found=%v err=%v", found, err)
	}
	if got.Name != "Chain Service" {
		t.Errorf("expected 'Chain Service', got %q", got.Name)
	}
}

func TestCatalogChain_ReadThrough_BackfillsL1(t *testing.T) {
	ctx := context.Background()
	l1 := NewMemoryTenantCatalogStore()
	l2 := NewMemoryTenantCatalogStore() // simulates Redis

	chain, err := NewCatalogChain([]TenantCatalogStore{l1, l2}, nil)
	if err != nil {
		t.Fatalf("new chain: %v", err)
	}
	defer chain.Close()

	tenant := MustTenantKey("backfill-tenant")
	svc := TenantServiceDescriptor{ServiceID: "svc-l2", Name: "L2 Only Service"}

	// Write directly to L2, skipping L1.
	if err := l2.RegisterService(ctx, tenant, svc); err != nil {
		t.Fatalf("l2 write: %v", err)
	}

	// L1 should miss initially.
	_, l1Hit, _ := l1.GetService(ctx, tenant, "svc-l2")
	if l1Hit {
		t.Fatal("L1 should not have the service before read-through")
	}

	// Chain read: should hit L2 and back-fill L1.
	got, found, err := chain.GetService(ctx, tenant, "svc-l2")
	if err != nil || !found {
		t.Fatalf("chain get: found=%v err=%v", found, err)
	}
	if got.Name != "L2 Only Service" {
		t.Errorf("expected 'L2 Only Service', got %q", got.Name)
	}

	// L1 should now be populated (back-fill).
	_, l1HitAfter, _ := l1.GetService(ctx, tenant, "svc-l2")
	if !l1HitAfter {
		t.Error("L1 should be back-filled after read-through")
	}
}

func TestCatalogChain_ListServices_AuthoritativeFromLast(t *testing.T) {
	ctx := context.Background()
	l1 := NewMemoryTenantCatalogStore()
	l3 := NewMemoryTenantCatalogStore() // simulates Postgres

	chain, err := NewCatalogChain([]TenantCatalogStore{l1, l3}, nil)
	if err != nil {
		t.Fatalf("new chain: %v", err)
	}
	defer chain.Close()

	tenant := MustTenantKey("list-tenant")

	// Write 3 services directly to L3.
	for i, id := range []ServiceID{"svc-a", "svc-b", "svc-c"} {
		_ = l3.RegisterService(ctx, tenant, TenantServiceDescriptor{
			ServiceID: id,
			Name:      string(rune('A' + i)),
		})
	}

	// Chain list should read from L3.
	list, err := chain.ListServices(ctx, tenant)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 services, got %d", len(list))
	}
}

// ---------------------------------------------------------------------------
// Memory store existing test (extended with CreatedAt check)
// ---------------------------------------------------------------------------

func TestMemoryTenantCatalogStore_CreatedAt(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryTenantCatalogStore()
	tenant := MustTenantKey("ts-tenant")

	svc := TenantServiceDescriptor{ServiceID: "ts-svc", Name: "TS Svc"}
	if err := store.RegisterService(ctx, tenant, svc); err != nil {
		t.Fatalf("register: %v", err)
	}

	got, found, _ := store.GetService(ctx, tenant, "ts-svc")
	if !found {
		t.Fatal("not found")
	}
	if got.CreatedAt == "" {
		t.Error("CreatedAt should be set by memory store")
	}
}
