package kernel

import (
	"context"
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

func (m *MemoryMeteringStore) Name() string                   { return "memory" }
func (m *MemoryMeteringStore) Priority() int                  { return 0 }
func (m *MemoryMeteringStore) Healthy() bool                  { return true } // RAM doesn't go down
func (m *MemoryMeteringStore) Ping(_ context.Context) error   { return nil } // always reachable

// --- Write path ---

// RegisterServiceSubscription registers or updates a service-specific
// subscription contract for a tenant. Returns nil always (memory never fails).
func (m *MemoryMeteringStore) RegisterServiceSubscription(sub ServiceSubscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sub.TenantKey == "" || sub.ServiceID == "" {
		return nil
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
	return nil
}

// RecordMeteringEvent appends a new billable metering event.
// Returns nil always (memory never fails).
func (m *MemoryMeteringStore) RecordMeteringEvent(event MeteringEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if event.TenantKey == "" {
		return nil
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	m.meteringEvents[event.TenantKey] = append(m.meteringEvents[event.TenantKey], event)
	return nil
}

// --- Read path ---

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

	for _, evt := range m.meteringEvents[tenantKey] {
		if evt.ServiceID != serviceID {
			continue
		}
		if (evt.Timestamp.Equal(startUTC) || evt.Timestamp.After(startUTC)) && evt.Timestamp.Before(endUTC) {
			ms, exists := stmt.Metrics[evt.MetricID]
			if !exists {
				ms = &MetricSummary{MetricID: evt.MetricID, Unit: evt.Unit}
				stmt.Metrics[evt.MetricID] = ms
			}
			ms.CycleTotal += evt.Quantity
			ms.TotalEvents++
		}
	}
	return stmt, true
}

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
		for _, evt := range m.meteringEvents[tenantKey] {
			if evt.ServiceID != serviceID {
				continue
			}
			if (evt.Timestamp.Equal(startUTC) || evt.Timestamp.After(startUTC)) && evt.Timestamp.Before(endUTC) {
				ms, exists := stmt.Metrics[evt.MetricID]
				if !exists {
					ms = &MetricSummary{MetricID: evt.MetricID, Unit: evt.Unit}
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
