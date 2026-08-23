export type SubscriptionState = 'trial' | 'active' | 'past_due' | 'cancelled';

export interface EntitlementItem {
  metric_id: string;
  allowed: boolean;
  quota: number;
}

export interface SubscriptionPlan {
  id: string;
  name: string;
  entitlements: Record<string, EntitlementItem>;
}

export interface TenantSubscriptionDetails {
  subscription_id: string;
  tenant_key: string;
  plan_id: string;
  state: SubscriptionState;
  anchor_time: string;
  timezone: string;
  is_usable: boolean;
  entitlements: Record<string, EntitlementItem>;
}

export interface FSMTransitionResponse {
  tenant_key: string;
  from: string;
  event: string;
  to: string;
}
