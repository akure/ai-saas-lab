package kernel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisTenantCatalogStore is an L2 Redis cache implementation of TenantCatalogStore.
type RedisTenantCatalogStore struct {
	client *redis.Client
	addr   string
}

func NewRedisTenantCatalogStore(ctx context.Context, addr string) (*RedisTenantCatalogStore, error) {
	opt, err := redis.ParseURL(addr)
	if err != nil {
		opt = &redis.Options{Addr: addr}
	}
	opt.PoolSize = 20
	opt.MinIdleConns = 5

	rdb := redis.NewClient(opt)
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis catalog: ping %s: %w", addr, err)
	}

	return &RedisTenantCatalogStore{
		client: rdb,
		addr:   addr,
	}, nil
}

func (r *RedisTenantCatalogStore) Name() string { return "redis" }
func (r *RedisTenantCatalogStore) Close() error {
	return r.client.Close()
}

// Redis keys:
//   catalog:{tenant_key}:services  -> Hash (field=service_id, value=JSON)
//   catalog:{tenant_key}:metrics   -> Hash (field=metric_id, value=JSON)
//   catalog:{tenant_key}:plans     -> Hash (field=plan_id, value=JSON)

// --- Service Catalog ---

func (r *RedisTenantCatalogStore) RegisterService(ctx context.Context, tenant TenantKey, svc TenantServiceDescriptor) error {
	data, err := json.Marshal(svc)
	if err != nil {
		return fmt.Errorf("marshal service: %w", err)
	}
	key := fmt.Sprintf("catalog:%s:services", tenant.String())
	return r.client.HSet(ctx, key, svc.ServiceID.String(), data).Err()
}

func (r *RedisTenantCatalogStore) GetService(ctx context.Context, tenant TenantKey, id ServiceID) (TenantServiceDescriptor, bool, error) {
	key := fmt.Sprintf("catalog:%s:services", tenant.String())
	val, err := r.client.HGet(ctx, key, id.String()).Result()
	if err != nil {
		if err == redis.Nil {
			return TenantServiceDescriptor{}, false, nil
		}
		return TenantServiceDescriptor{}, false, err
	}
	var svc TenantServiceDescriptor
	if err := json.Unmarshal([]byte(val), &svc); err != nil {
		return TenantServiceDescriptor{}, false, err
	}
	return svc, true, nil
}

func (r *RedisTenantCatalogStore) ListServices(ctx context.Context, tenant TenantKey) ([]TenantServiceDescriptor, error) {
	key := fmt.Sprintf("catalog:%s:services", tenant.String())
	vals, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var list []TenantServiceDescriptor
	for _, val := range vals {
		var svc TenantServiceDescriptor
		if err := json.Unmarshal([]byte(val), &svc); err == nil {
			list = append(list, svc)
		}
	}
	return list, nil
}

// --- Metric Catalog ---

func (r *RedisTenantCatalogStore) RegisterMetric(ctx context.Context, tenant TenantKey, metric TenantMetricDescriptor) error {
	data, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("marshal metric: %w", err)
	}
	key := fmt.Sprintf("catalog:%s:metrics", tenant.String())
	return r.client.HSet(ctx, key, metric.MetricID.String(), data).Err()
}

func (r *RedisTenantCatalogStore) GetMetric(ctx context.Context, tenant TenantKey, id MetricID) (TenantMetricDescriptor, bool, error) {
	key := fmt.Sprintf("catalog:%s:metrics", tenant.String())
	val, err := r.client.HGet(ctx, key, id.String()).Result()
	if err != nil {
		if err == redis.Nil {
			return TenantMetricDescriptor{}, false, nil
		}
		return TenantMetricDescriptor{}, false, err
	}
	var metric TenantMetricDescriptor
	if err := json.Unmarshal([]byte(val), &metric); err != nil {
		return TenantMetricDescriptor{}, false, err
	}
	return metric, true, nil
}

func (r *RedisTenantCatalogStore) ListMetrics(ctx context.Context, tenant TenantKey) ([]TenantMetricDescriptor, error) {
	key := fmt.Sprintf("catalog:%s:metrics", tenant.String())
	vals, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var list []TenantMetricDescriptor
	for _, val := range vals {
		var metric TenantMetricDescriptor
		if err := json.Unmarshal([]byte(val), &metric); err == nil {
			list = append(list, metric)
		}
	}
	return list, nil
}

// --- Plan Catalog ---

func (r *RedisTenantCatalogStore) RegisterPlan(ctx context.Context, tenant TenantKey, plan TenantPlanDescriptor) error {
	data, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	key := fmt.Sprintf("catalog:%s:plans", tenant.String())
	return r.client.HSet(ctx, key, plan.PlanID.String(), data).Err()
}

func (r *RedisTenantCatalogStore) GetPlan(ctx context.Context, tenant TenantKey, id ApplicationPlanID) (TenantPlanDescriptor, bool, error) {
	key := fmt.Sprintf("catalog:%s:plans", tenant.String())
	val, err := r.client.HGet(ctx, key, id.String()).Result()
	if err != nil {
		if err == redis.Nil {
			return TenantPlanDescriptor{}, false, nil
		}
		return TenantPlanDescriptor{}, false, err
	}
	var plan TenantPlanDescriptor
	if err := json.Unmarshal([]byte(val), &plan); err != nil {
		return TenantPlanDescriptor{}, false, err
	}
	return plan, true, nil
}

func (r *RedisTenantCatalogStore) ListPlans(ctx context.Context, tenant TenantKey) ([]TenantPlanDescriptor, error) {
	key := fmt.Sprintf("catalog:%s:plans", tenant.String())
	vals, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var list []TenantPlanDescriptor
	for _, val := range vals {
		var plan TenantPlanDescriptor
		if err := json.Unmarshal([]byte(val), &plan); err == nil {
			list = append(list, plan)
		}
	}
	return list, nil
}
