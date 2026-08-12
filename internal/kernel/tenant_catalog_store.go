package kernel

import (
	"context"
	"errors"
)

var (
	ErrTenantServiceNotFound = errors.New("tenant service descriptor not found")
	ErrTenantMetricNotFound  = errors.New("tenant metric descriptor not found")
	ErrTenantPlanNotFound    = errors.New("tenant plan descriptor not found")
)

// ServiceCatalog defines sub-interface for Tenant Service operations.
type ServiceCatalog interface {
	RegisterService(ctx context.Context, tenant TenantKey, svc TenantServiceDescriptor) error
	GetService(ctx context.Context, tenant TenantKey, id ServiceID) (TenantServiceDescriptor, bool, error)
	ListServices(ctx context.Context, tenant TenantKey) ([]TenantServiceDescriptor, error)
}

// MetricCatalog defines sub-interface for Tenant Metric operations.
type MetricCatalog interface {
	RegisterMetric(ctx context.Context, tenant TenantKey, metric TenantMetricDescriptor) error
	GetMetric(ctx context.Context, tenant TenantKey, id MetricID) (TenantMetricDescriptor, bool, error)
	ListMetrics(ctx context.Context, tenant TenantKey) ([]TenantMetricDescriptor, error)
}

// PlanCatalog defines sub-interface for Tenant Application Plan operations.
type PlanCatalog interface {
	RegisterPlan(ctx context.Context, tenant TenantKey, plan TenantPlanDescriptor) error
	GetPlan(ctx context.Context, tenant TenantKey, id ApplicationPlanID) (TenantPlanDescriptor, bool, error)
	ListPlans(ctx context.Context, tenant TenantKey) ([]TenantPlanDescriptor, error)
}

// TenantCatalogStore is the composed catalog interface implemented by L1/L2/L3 backends.
type TenantCatalogStore interface {
	ServiceCatalog
	MetricCatalog
	PlanCatalog
	Name() string
	Close() error
}
