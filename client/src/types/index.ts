export type PlanTier = 'free' | 'pro' | 'enterprise';

export interface ApiKeyItem {
  id: string;
  name: string;
  key: string;
  plan: PlanTier;
  createdAt: string;
  status: 'active' | 'revoked';
  rateLimitRpm: number;
  totalTokensUsed: number;
  lastUsedAt: string;
}

export interface SimulationParams {
  concurrentUsers: number;
  requestsPerSec: number;
  inputTokensPerReq: number;
  outputTokensPerReq: number;
  vectorStorageGb: number;
  modelTier: 'fast-lite' | 'pro-reasoning' | 'ultra-vision';
  cacheHitRatioPercent: number;
}

export interface SimulationResults {
  totalTokensPerSec: number;
  bandwidthMbps: number;
  estimatedDailyCost: number;
  estimatedMonthlyCost: number;
  vectorEmbeddingsCount: number;
  storageUtilizationPercent: number;
  rateLimitStatus: 'optimal' | 'warning' | 'throttled';
}

export interface UsageTelemetry {
  apiKey: string;
  totalRequests: number;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  storageUsedBytes: number;
  lastRequestTime: string;
  hourlyHistory: Array<{ hour: string; requests: number; tokens: number }>;
}

export interface ToastMessage {
  id: string;
  type: 'success' | 'error' | 'info' | 'warning';
  title: string;
  message: string;
}
