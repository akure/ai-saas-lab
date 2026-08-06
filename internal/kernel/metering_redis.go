package kernel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// ---------------------------------------------------------------------------
// RedisMeteringStore — fast cache backend (L2 in the cache hierarchy).
// ---------------------------------------------------------------------------

// redisHealthCheckInterval is how often the background probe pings Redis.
const redisHealthCheckInterval = 10 * time.Second

// Redis key patterns:
//
//	Subscriptions:  metering:{tenant_key}:subs          → Hash (field=service_id, value=JSON)
//	Events:         metering:{tenant_key}:{service_id}:events → ZSet (score=UnixNano, member=JSON)
//	Event dedup:    metering:event:{event_id}             → String (TTL = 2h, for cross-instance dedup)
const (
	redisSubKeyFmt   = "metering:%s:subs"
	redisEvtKeyFmt   = "metering:%s:%s:events"
	redisEventTTL    = 90 * 24 * time.Hour  // 90 days — covers monthly + yearly cycles
	redisSubTTL      = 0                    // no TTL — deleted explicitly on cancellation
)

// RedisMeteringStore is the Redis implementation of MeteringStore.
//
// Write path:
//   - RegisterServiceSubscription: HSET metering:{tenant}:subs {service} {json}
//   - RecordMeteringEvent: ZADD metering:{tenant}:{service}:events {timestamp_ns} {json}
//
// Read path:
//   - GetServiceSubscriptions: HGETALL metering:{tenant}:subs
//   - GetServiceBillingStatement: ZRANGEBYSCORE + in-memory aggregate
//   - GetTenantBillingOverview: iterate subscriptions → aggregate per service
//
// Health:
//   - Background goroutine pings every 10s. Healthy() reflects last probe result.
type RedisMeteringStore struct {
	client  *redis.Client
	healthy atomic.Bool
	stopCh  chan struct{}
	once    sync.Once
}

// NewRedisMeteringStore creates and connects a Redis backend.
// Returns an error if the initial PING fails.
// The health-check goroutine starts automatically; call Stop() on shutdown.
func NewRedisMeteringStore(ctx context.Context, addr string) (*RedisMeteringStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:            addr,
		MaxRetries:      3,
		DialTimeout:     2 * time.Second,
		ReadTimeout:     2 * time.Second,
		WriteTimeout:    2 * time.Second,
		PoolSize:        20,
		MinIdleConns:    2,
		ConnMaxLifetime: 30 * time.Minute,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: initial ping failed: %w", err)
	}

	s := &RedisMeteringStore{
		client: client,
		stopCh: make(chan struct{}),
	}
	s.healthy.Store(true)
	go s.healthProbe()
	return s, nil
}

// --- MeteringStore identity ---

func (r *RedisMeteringStore) Name() string  { return "redis" }
func (r *RedisMeteringStore) Priority() int { return 10 }
func (r *RedisMeteringStore) Healthy() bool { return r.healthy.Load() }

// Ping verifies Redis connectivity for the health-check goroutine.
func (r *RedisMeteringStore) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Stop terminates the health-check goroutine and closes the Redis client.
func (r *RedisMeteringStore) Stop() {
	r.once.Do(func() {
		close(r.stopCh)
		_ = r.client.Close()
	})
}

// --- Write path ---

// RegisterServiceSubscription caches a subscription in a Redis hash.
// Key: metering:{tenant_key}:subs, Field: service_id, Value: JSON.
func (r *RedisMeteringStore) RegisterServiceSubscription(sub ServiceSubscription) error {
	if !r.healthy.Load() {
		return fmt.Errorf("redis: backend unhealthy")
	}

	data, err := json.Marshal(sub)
	if err != nil {
		return fmt.Errorf("redis: marshal subscription: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	key := fmt.Sprintf(redisSubKeyFmt, sub.TenantKey)
	if err := r.client.HSet(ctx, key, sub.ServiceID, data).Err(); err != nil {
		if isRedisConnectionError(err) {
			r.healthy.Store(false)
		}
		return fmt.Errorf("redis: HSET subscription: %w", err)
	}
	return nil
}

// RecordMeteringEvent adds a metering event to a sorted set keyed by tenant+service.
// Score is the event timestamp in nanoseconds for precise time-range queries.
// TTL is set to redisEventTTL to auto-expire old events.
func (r *RedisMeteringStore) RecordMeteringEvent(event MeteringEvent) error {
	if !r.healthy.Load() {
		return fmt.Errorf("redis: backend unhealthy")
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("redis: marshal event: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	key := fmt.Sprintf(redisEvtKeyFmt, event.TenantKey, event.ServiceID)
	pipe := r.client.Pipeline()
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(event.Timestamp.UnixNano()),
		Member: string(data),
	})
	pipe.Expire(ctx, key, redisEventTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		if isRedisConnectionError(err) {
			r.healthy.Store(false)
		}
		return fmt.Errorf("redis: ZADD event: %w", err)
	}
	return nil
}

// isRedisConnectionError reports whether err is a connection/network-level failure
// rather than a logical Redis server error (e.g. WRONGTYPE key collision).
// Logical errors returned by Redis server implement redis.Error.
func isRedisConnectionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, redis.Nil) {
		return false
	}
	var rErr redis.Error
	if errors.As(err, &rErr) {
		return false
	}
	return true
}

// --- Read path ---

// GetServiceSubscriptions retrieves all subscriptions for a tenant from the hash.
func (r *RedisMeteringStore) GetServiceSubscriptions(tenantKey string) []ServiceSubscription {
	if !r.healthy.Load() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	key := fmt.Sprintf(redisSubKeyFmt, tenantKey)
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil || len(result) == 0 {
		return nil
	}

	subs := make([]ServiceSubscription, 0, len(result))
	for _, v := range result {
		var sub ServiceSubscription
		if err := json.Unmarshal([]byte(v), &sub); err == nil {
			subs = append(subs, sub)
		}
	}
	return subs
}

// GetServiceBillingStatement loads a subscription from Redis, computes the cycle
// window, queries the sorted set with ZRANGEBYSCORE, and aggregates in memory.
func (r *RedisMeteringStore) GetServiceBillingStatement(tenantKey, serviceID string, targetTime time.Time) (ServiceBillingStatement, bool) {
	if !r.healthy.Load() {
		return ServiceBillingStatement{}, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Load subscription to determine cycle window.
	subKey := fmt.Sprintf(redisSubKeyFmt, tenantKey)
	subData, err := r.client.HGet(ctx, subKey, serviceID).Bytes()
	if err != nil {
		return ServiceBillingStatement{}, false
	}
	var sub ServiceSubscription
	if err := json.Unmarshal(subData, &sub); err != nil {
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

	// Query sorted set for events within [startUTC, endUTC).
	evtKey := fmt.Sprintf(redisEvtKeyFmt, tenantKey, serviceID)
	members, err := r.client.ZRangeByScore(ctx, evtKey, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", startUTC.UnixNano()),
		Max: fmt.Sprintf("%d", endUTC.UnixNano()-1),
	}).Result()
	if err != nil {
		return stmt, true // partial result
	}

	for _, member := range members {
		var evt MeteringEvent
		if err := json.Unmarshal([]byte(member), &evt); err != nil {
			continue
		}
		ms, exists := stmt.Metrics[evt.MetricID]
		if !exists {
			ms = &MetricSummary{MetricID: evt.MetricID, Unit: evt.Unit}
			stmt.Metrics[evt.MetricID] = ms
		}
		ms.CycleTotal += evt.Quantity
		ms.TotalEvents++
	}
	return stmt, true
}

// GetTenantBillingOverview iterates all subscriptions and aggregates per-service.
func (r *RedisMeteringStore) GetTenantBillingOverview(tenantKey string, targetTime time.Time) TenantBillingOverview {
	overview := TenantBillingOverview{
		TenantKey:         tenantKey,
		SubscriptionState: "unknown",
		Statements:        make([]ServiceBillingStatement, 0),
		GeneratedAt:       time.Now().UTC(),
	}

	subs := r.GetServiceSubscriptions(tenantKey)
	for _, sub := range subs {
		stmt, ok := r.GetServiceBillingStatement(tenantKey, sub.ServiceID, targetTime)
		if ok {
			overview.Statements = append(overview.Statements, stmt)
		}
	}
	return overview
}

// --- Background health-check goroutine ---

// healthProbe pings Redis every redisHealthCheckInterval and updates the
// healthy flag. Called once at startup from NewRedisMeteringStore.
func (r *RedisMeteringStore) healthProbe() {
	ticker := time.NewTicker(redisHealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := r.client.Ping(ctx).Err()
			cancel()
			if err != nil {
				r.healthy.Store(false)
			} else {
				r.healthy.Store(true)
			}
		case <-r.stopCh:
			return
		}
	}
}
