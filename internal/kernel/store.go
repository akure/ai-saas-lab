package kernel

import (
	"sync"
	"time"
)

// APIKeyRecord is the in-memory representation of a key used by the auth layer.
type APIKeyRecord struct {
	Key    string
	Plan   string
	Active bool
}

// Store is thread-safe, multi-tenant storage supporting API keys, subscriptions,
// and service-specific metering events.
type Store struct {
	mu                   sync.RWMutex
	apiKeys              map[string]APIKeyRecord
	usage                map[string]int                         // key -> tokens used today (legacy)
	subscriptions        map[string]State                       // key -> overall subscription state
	serviceSubscriptions map[string]map[string]ServiceSubscription // tenantKey -> serviceID -> ServiceSubscription
	meteringEvents       map[string][]MeteringEvent             // tenantKey -> []MeteringEvent
}

func NewStore() *Store {
	return &Store{
		apiKeys:              make(map[string]APIKeyRecord),
		usage:                make(map[string]int),
		subscriptions:        make(map[string]State),
		serviceSubscriptions: make(map[string]map[string]ServiceSubscription),
		meteringEvents:       make(map[string][]MeteringEvent),
	}
}

func (s *Store) SeedAPIKey(key, plan string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key == "" {
		return
	}
	s.apiKeys[key] = APIKeyRecord{Key: key, Plan: plan, Active: true}
}

func (s *Store) IsValidAPIKey(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.apiKeys[key]
	return ok && rec.Active
}

func (s *Store) ActivateAPIKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.apiKeys[key]; ok {
		rec.Active = true
		s.apiKeys[key] = rec
	}
}

func (s *Store) RevokeAPIKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.apiKeys[key]; ok {
		rec.Active = false
		s.apiKeys[key] = rec
	}
}

func (s *Store) APIKeyInfo(key string) (APIKeyRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.apiKeys[key]
	return rec, ok
}

func (s *Store) AddUsage(key string, tokens int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usage[key] += tokens
	return s.usage[key]
}

func (s *Store) UsageFor(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.usage[key]
}

func (s *Store) SetSubscriptionState(key string, st State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscriptions[key] = st
}

func (s *Store) SubscriptionState(key string) State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if st, ok := s.subscriptions[key]; ok {
		return st
	}
	return "unknown"
}

// RegisterServiceSubscription registers or updates a service-specific subscription contract for a tenant.
func (s *Store) RegisterServiceSubscription(sub ServiceSubscription) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	if _, ok := s.serviceSubscriptions[sub.TenantKey]; !ok {
		s.serviceSubscriptions[sub.TenantKey] = make(map[string]ServiceSubscription)
	}
	s.serviceSubscriptions[sub.TenantKey][sub.ServiceID] = sub
}

// GetServiceSubscriptions retrieves all active service subscriptions for a given tenant.
func (s *Store) GetServiceSubscriptions(tenantKey string) []ServiceSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	subsMap, ok := s.serviceSubscriptions[tenantKey]
	if !ok {
		return nil
	}
	res := make([]ServiceSubscription, 0, len(subsMap))
	for _, sub := range subsMap {
		res = append(res, sub)
	}
	return res
}

// RecordMeteringEvent appends a new billable metering event to the tenant's time-series log.
func (s *Store) RecordMeteringEvent(event MeteringEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if event.TenantKey == "" {
		return
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	s.meteringEvents[event.TenantKey] = append(s.meteringEvents[event.TenantKey], event)
	if event.MetricID == "total_tokens" || event.MetricID == "tokens" {
		s.usage[event.TenantKey] += int(event.Quantity)
	}
}

// GetServiceBillingStatement computes the billing statement for a specific service subscription at targetTime.
func (s *Store) GetServiceBillingStatement(tenantKey, serviceID string, targetTime time.Time) (ServiceBillingStatement, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subsMap, ok := s.serviceSubscriptions[tenantKey]
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

	events := s.meteringEvents[tenantKey]
	for _, evt := range events {
		if evt.ServiceID != serviceID {
			continue
		}
		if (evt.Timestamp.Equal(startUTC) || evt.Timestamp.After(startUTC)) && evt.Timestamp.Before(endUTC) {
			m, exists := stmt.Metrics[evt.MetricID]
			if !exists {
				m = &MetricSummary{
					MetricID: evt.MetricID,
					Unit:     evt.Unit,
				}
				stmt.Metrics[evt.MetricID] = m
			}
			m.CycleTotal += evt.Quantity
			m.TotalEvents++
		}
	}

	return stmt, true
}

// GetTenantBillingOverview returns all service billing statements for a tenant at targetTime.
func (s *Store) GetTenantBillingOverview(tenantKey string, targetTime time.Time) TenantBillingOverview {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st := s.SubscriptionState(tenantKey)
	overview := TenantBillingOverview{
		TenantKey:         tenantKey,
		SubscriptionState: string(st),
		Statements:        make([]ServiceBillingStatement, 0),
		GeneratedAt:       time.Now().UTC(),
	}

	subsMap, ok := s.serviceSubscriptions[tenantKey]
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

		events := s.meteringEvents[tenantKey]
		for _, evt := range events {
			if evt.ServiceID != serviceID {
				continue
			}
			if (evt.Timestamp.Equal(startUTC) || evt.Timestamp.After(startUTC)) && evt.Timestamp.Before(endUTC) {
				m, exists := stmt.Metrics[evt.MetricID]
				if !exists {
					m = &MetricSummary{
						MetricID: evt.MetricID,
						Unit:     evt.Unit,
					}
					stmt.Metrics[evt.MetricID] = m
				}
				m.CycleTotal += evt.Quantity
				m.TotalEvents++
			}
		}
		overview.Statements = append(overview.Statements, stmt)
	}

	return overview
}
