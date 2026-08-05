package kernel

import (
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// MemoryMeteringStore — fast in-memory backend (L1 in the cache hierarchy).
// ---------------------------------------------------------------------------

// MemoryMeteringStore is the in-memory implementation of MeteringStore.
// It is always present in the MeteringChain as the L1 (fastest) backend.
// Data is lost on process restart — that's by design; the persistent
// backends (PostgreSQL) provide durability.
//
// This implementation is extracted from the original Store struct in store.go.
// The data structures and algorithms are identical; only the ownership has
// moved to enable the multi-backend chain pattern.
type MemoryMeteringStore struct {
	mu                   sync.RWMutex
	serviceSubscriptions map[string]map[string]ServiceSubscription // tenantKey → serviceID → sub
	meteringEvents       map[string][]MeteringEvent                // tenantKey → events
}

// NewMemoryMeteringStore creates a new in-memory metering backend.
func NewMemoryMeteringStore() *MemoryMeteringStore {
	return &MemoryMeteringStore{
		serviceSubscriptions: make(map[string]map[string]ServiceSubscription),
		meteringEvents:       make(map[string][]MeteringEvent),
	}
}

// --- MeteringStore identity ---

func (m *MemoryMeteringStore) Name() string  { return "memory" }
func (m *MemoryMeteringStore) Priority() int { return 0 }
func (m *MemoryMeteringStore) Healthy() bool { return true } // RAM doesn't go down

// --- Write path ---

// RegisterServiceSubscription registers or updates a service-specific
// subscription contract for a tenant.
func (m *MemoryMeteringStore) RegisterServiceSubscription(sub ServiceSubscription) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sub.TenantKey == "" || sub.ServiceID == "" {
		return
	}
	if sub.Timezone == "" {
		sub.Timezone = "UTC"
	}
	if sub.AnchorTime.IsZero() {
		sub.AnchorTime = time.Now().UTC()
	}
	if sub.Status == "" {
		sub.Status = "active"
	}
	if _, ok := m.serviceSubscriptions[sub.TenantKey]; !ok {
		m.serviceSubscriptions[sub.TenantKey] = make(map[string]ServiceSubscription)
	}
	m.serviceSubscriptions[sub.TenantKey][sub.ServiceID] = sub
}

// RecordMeteringEvent appends a new billable metering event to the tenant's
// time-series log.
func (m *MemoryMeteringStore) RecordMeteringEvent(event MeteringEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if event.TenantKey == "" {
		return
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	m.meteringEvents[event.TenantKey] = append(m.meteringEvents[event.TenantKey], event)
}

// --- Read path ---

// GetServiceSubscriptions retrieves all active service subscriptions for a
// given tenant.
func (m *MemoryMeteringStore) GetServiceSubscriptions(tenantKey string) []ServiceSubscription {
	m.mu.RLock()
	defer m.mu.RUnlock()

	subsMap, ok := m.serviceSubscriptions[tenantKey]
	if !ok {
		return nil
	}
	res := make([]ServiceSubscription, 0, len(subsMap))
	for _, sub := range subsMap {
		res = append(res, sub)
	}
	return res
}

// GetServiceBillingStatement computes the billing statement for a specific
// service subscription at targetTime. The cycle window is resolved using the
// subscription's AnchorTime and Timezone.
func (m *MemoryMeteringStore) GetServiceBillingStatement(tenantKey, serviceID string, targetTime time.Time) (ServiceBillingStatement, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	subsMap, ok := m.serviceSubscriptions[tenantKey]
	if !ok {
		return ServiceBillingStatement{}, false
	}
	sub, ok := subsMap[serviceID]
	if !ok {
		return ServiceBillingStatement{}, false
	}

	startUTC, endUTC := sub.CurrentCycleWindow(targetTime)
	stmt := ServiceBillingStatement{
		SubscriptionID: sub.SubscriptionID,
		TenantKey:      sub.TenantKey,
		ServiceID:      sub.ServiceID,
		PlanID:         sub.PlanID,
		ChargeType:     sub.ChargeType,
		Timezone:       sub.Timezone,
		CycleStartUTC:  startUTC,
		CycleEndUTC:    endUTC,
		Metrics:        make(map[string]*MetricSummary),
		GeneratedAt:    time.Now().UTC(),
	}

	events := m.meteringEvents[tenantKey]
	for _, evt := range events {
		if evt.ServiceID != serviceID {
			continue
		}
		if (evt.Timestamp.Equal(startUTC) || evt.Timestamp.After(startUTC)) && evt.Timestamp.Before(endUTC) {
			ms, exists := stmt.Metrics[evt.MetricID]
			if !exists {
				ms = &MetricSummary{
					MetricID: evt.MetricID,
					Unit:     evt.Unit,
				}
				stmt.Metrics[evt.MetricID] = ms
			}
			ms.CycleTotal += evt.Quantity
			ms.TotalEvents++
		}
	}

	return stmt, true
}

// GetTenantBillingOverview returns all service billing statements for a tenant
// at targetTime.
func (m *MemoryMeteringStore) GetTenantBillingOverview(tenantKey string, targetTime time.Time) TenantBillingOverview {
	m.mu.RLock()
	defer m.mu.RUnlock()

	overview := TenantBillingOverview{
		TenantKey:         tenantKey,
		SubscriptionState: "unknown",
		Statements:        make([]ServiceBillingStatement, 0),
		GeneratedAt:       time.Now().UTC(),
	}

	subsMap, ok := m.serviceSubscriptions[tenantKey]
	if !ok {
		return overview
	}

	for serviceID, sub := range subsMap {
		startUTC, endUTC := sub.CurrentCycleWindow(targetTime)
		stmt := ServiceBillingStatement{
			SubscriptionID: sub.SubscriptionID,
			TenantKey:      sub.TenantKey,
			ServiceID:      sub.ServiceID,
			PlanID:         sub.PlanID,
			ChargeType:     sub.ChargeType,
			Timezone:       sub.Timezone,
			CycleStartUTC:  startUTC,
			CycleEndUTC:    endUTC,
			Metrics:        make(map[string]*MetricSummary),
			GeneratedAt:    time.Now().UTC(),
		}

		events := m.meteringEvents[tenantKey]
		for _, evt := range events {
			if evt.ServiceID != serviceID {
				continue
			}
			if (evt.Timestamp.Equal(startUTC) || evt.Timestamp.After(startUTC)) && evt.Timestamp.Before(endUTC) {
				ms, exists := stmt.Metrics[evt.MetricID]
				if !exists {
					ms = &MetricSummary{
						MetricID: evt.MetricID,
						Unit:     evt.Unit,
					}
					stmt.Metrics[evt.MetricID] = ms
				}
				ms.CycleTotal += evt.Quantity
				ms.TotalEvents++
			}
		}
		overview.Statements = append(overview.Statements, stmt)
	}

	return overview
}
