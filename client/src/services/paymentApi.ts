export type SessionStatus = 'pending' | 'requires_action' | 'succeeded' | 'failed' | 'cancelled';
export type PaymentMethod = 'card' | 'bank_transfer' | 'digital_wallet' | 'crypto_token';

export interface CheckoutSession {
  id: string;
  tenant_key: string;
  plan_id: string;
  amount_cents: number;
  currency: string;
  billing_cycle: string;
  status: SessionStatus;
  payment_method?: PaymentMethod;
  created_at: string;
  updated_at: string;
  error_reason?: string;
}

export interface Transaction {
  id: string;
  session_id: string;
  tenant_key: string;
  plan_id: string;
  amount_cents: number;
  currency: string;
  payment_method: PaymentMethod;
  status: SessionStatus;
  reference_id: string;
  processed_at: string;
  failure_reason?: string;
}

export interface CreateCheckoutReq {
  tenant_key: string;
  plan_id: string;
  amount_cents: number;
  currency: string;
  billing_cycle: string;
}

export interface ProcessPaymentReq {
  payment_method: PaymentMethod;
  simulate_mode: 'auto' | 'succeed' | 'decline' | 'challenge';
  account_holder?: string;
  token_ref?: string;
  challenge_code?: string;
}

export interface SimulateWebhookReq {
  event: string;
  tenant_key: string;
  session_id?: string;
  plan_id?: string;
}

export interface WebhookResult {
  event: string;
  tenant_key: string;
  fsm_effect: string;
  processed_at: string;
}

const API_BASE = 'http://localhost:8080/v1/payment';

export async function createCheckoutSession(req: CreateCheckoutReq): Promise<CheckoutSession> {
  const res = await fetch(`${API_BASE}/checkout`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Failed to create checkout session' }));
    throw new Error(err.error || 'Failed to create checkout session');
  }
  return res.json();
}

export async function getCheckoutSession(id: string): Promise<CheckoutSession> {
  const res = await fetch(`${API_BASE}/sessions/${encodeURIComponent(id)}`);
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Checkout session not found' }));
    throw new Error(err.error || 'Checkout session not found');
  }
  return res.json();
}

export async function processPayment(id: string, req: ProcessPaymentReq): Promise<Transaction> {
  const res = await fetch(`${API_BASE}/sessions/${encodeURIComponent(id)}/process`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });

  const data = await res.json();
  if (res.status === 202) { // 3DS / Challenge required
    throw { isChallenge: true, message: data.message || 'Verification challenge required' };
  }
  if (!res.ok) {
    throw new Error(data.error || data.failure_reason || 'Payment processing failed');
  }
  return data;
}

export async function simulateWebhook(req: SimulateWebhookReq): Promise<WebhookResult> {
  const res = await fetch(`${API_BASE}/webhooks/simulate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Webhook simulation failed' }));
    throw new Error(err.error || 'Webhook simulation failed');
  }
  return res.json();
}

export async function listTransactions(tenantKey: string): Promise<Transaction[]> {
  const res = await fetch(`${API_BASE}/transactions/${encodeURIComponent(tenantKey)}`);
  if (!res.ok) {
    return [];
  }
  const data = await res.json();
  return data.transactions || [];
}
