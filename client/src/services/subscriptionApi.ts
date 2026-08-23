import {
  SubscriptionPlan,
  TenantSubscriptionDetails,
  FSMTransitionResponse,
} from '../types/subscription';

const BASE_URL = 'http://localhost:8080';

export async function fetchSubscriptionPlans(): Promise<SubscriptionPlan[]> {
  try {
    const res = await fetch(`${BASE_URL}/v1/subscription/plans`);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const data = await res.json();
    return data.plans || [];
  } catch (err) {
    // Offline simulation fallback
    return [
      {
        id: 'starter',
        name: 'Starter Tier',
        entitlements: {
          total_tokens: { metric_id: 'total_tokens', allowed: true, quota: 1000000 },
        },
      },
      {
        id: 'pro',
        name: 'Pro Tier',
        entitlements: {
          total_tokens: { metric_id: 'total_tokens', allowed: true, quota: 10000000 },
        },
      },
      {
        id: 'enterprise',
        name: 'Enterprise Tier',
        entitlements: {
          total_tokens: { metric_id: 'total_tokens', allowed: true, quota: 100000000 },
        },
      },
    ];
  }
}

export async function fetchTenantSubscription(
  tenantKey: string
): Promise<TenantSubscriptionDetails> {
  try {
    const res = await fetch(`${BASE_URL}/v1/subscription/${encodeURIComponent(tenantKey)}`);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return await res.json();
  } catch (err) {
    // Offline simulation fallback
    return {
      subscription_id: `sub_pro_${tenantKey}`,
      tenant_key: tenantKey,
      plan_id: 'pro',
      state: 'active',
      anchor_time: new Date().toISOString(),
      timezone: 'UTC',
      is_usable: true,
      entitlements: {
        total_tokens: { metric_id: 'total_tokens', allowed: true, quota: 10000000 },
      },
    };
  }
}

export async function createSubscriptionContract(
  tenantKey: string,
  planId: string,
  timezone: string = 'UTC'
): Promise<TenantSubscriptionDetails> {
  const res = await fetch(`${BASE_URL}/v1/subscription/contracts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      tenant_key: tenantKey,
      plan_id: planId,
      timezone,
    }),
  });
  if (!res.ok) {
    const errData = await res.json().catch(() => ({}));
    throw new Error(errData.error || `HTTP ${res.status}`);
  }
  return await res.json();
}

export async function fireSubscriptionEvent(
  tenantKey: string,
  event: string
): Promise<FSMTransitionResponse> {
  const res = await fetch(`${BASE_URL}/v1/subscription/${encodeURIComponent(tenantKey)}/event`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ event }),
  });
  if (!res.ok) {
    const errData = await res.json().catch(() => ({}));
    throw new Error(errData.error || `HTTP ${res.status}`);
  }
  return await res.json();
}

export async function registerSubscriptionPlan(
  plan: SubscriptionPlan
): Promise<SubscriptionPlan> {
  const res = await fetch(`${BASE_URL}/v1/subscription/plans`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(plan),
  });
  if (!res.ok) {
    const errData = await res.json().catch(() => ({}));
    throw new Error(errData.error || `HTTP ${res.status}`);
  }
  return await res.json();
}
