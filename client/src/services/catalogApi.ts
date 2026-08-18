import {
  TenantServiceDescriptor,
  TenantMetricDescriptor,
  TenantPlanDescriptor,
  CatalogOverview,
  CatalogApiError,
  BatchImportResult,
} from '../types/catalog';

const BASE_URL = 'http://localhost:8080';

// Fallback seed catalog for offline or simulated sandbox mode
const MOCK_STORAGE_KEY = 'ai_saas_mock_catalog_data';

function getMockData(tenantKey: string): CatalogOverview {
  try {
    const raw = localStorage.getItem(`${MOCK_STORAGE_KEY}_${tenantKey}`);
    if (raw) return JSON.parse(raw);
  } catch (e) {}

  return {
    tenant_key: tenantKey,
    services: [
      {
        service_id: 'ai-completion',
        name: 'AI Completion Engine',
        description: 'High-throughput LLM text generation and token metering service',
        created_at: new Date(Date.now() - 86400000 * 5).toISOString(),
      },
      {
        service_id: 'vector-storage',
        name: 'Vector Database & RAG',
        description: 'High-density vector embeddings storage and semantic search index',
        created_at: new Date(Date.now() - 86400000 * 3).toISOString(),
      },
      {
        service_id: 'transcription-svc',
        name: 'Whisper Audio Transcription',
        description: 'Multi-lingual audio-to-text pipeline with millisecond metering',
        created_at: new Date(Date.now() - 86400000 * 1).toISOString(),
      },
    ],
    metrics: [
      {
        metric_id: 'tokens',
        service_id: 'ai-completion',
        name: 'Prompt & Completion Tokens',
        unit: 'tokens',
        description: 'Aggregated input and output token counts computed per request',
        created_at: new Date(Date.now() - 86400000 * 5).toISOString(),
      },
      {
        metric_id: 'requests',
        service_id: 'ai-completion',
        name: 'API Calls',
        unit: 'requests',
        description: 'Total HTTP completions endpoint invocations',
        created_at: new Date(Date.now() - 86400000 * 5).toISOString(),
      },
      {
        metric_id: 'storage_gb_hours',
        service_id: 'vector-storage',
        name: 'Vector Dimension Storage',
        unit: 'bytes',
        description: 'Gigabyte-hour indexing footprint in hot memory',
        created_at: new Date(Date.now() - 86400000 * 3).toISOString(),
      },
      {
        metric_id: 'audio_seconds',
        service_id: 'transcription-svc',
        name: 'Audio Duration Processed',
        unit: 'seconds',
        description: 'Duration of recorded audio transcribed',
        created_at: new Date(Date.now() - 86400000 * 1).toISOString(),
      },
    ],
    plans: [
      {
        plan_id: 'starter_plan',
        service_id: 'ai-completion',
        name: 'Starter Developer Tier',
        rates: { tokens: 0.0015, requests: 0.001 },
        included_quotas: { tokens: 25000, requests: 500 },
        version: 1,
        active: true,
        created_at: new Date(Date.now() - 86400000 * 5).toISOString(),
      },
      {
        plan_id: 'growth_tier',
        service_id: 'ai-completion',
        name: 'Growth Scale Plan',
        rates: { tokens: 0.0008, requests: 0.0005 },
        included_quotas: { tokens: 1000000, requests: 25000 },
        version: 2,
        active: true,
        created_at: new Date(Date.now() - 86400000 * 3).toISOString(),
      },
      {
        plan_id: 'audio_pro',
        service_id: 'transcription-svc',
        name: 'Voice AI Production Plan',
        rates: { audio_seconds: 0.005 },
        included_quotas: { audio_seconds: 3600 },
        version: 1,
        active: true,
        created_at: new Date(Date.now() - 86400000 * 1).toISOString(),
      },
    ],
  };
}

function saveMockData(tenantKey: string, data: CatalogOverview) {
  try {
    localStorage.setItem(`${MOCK_STORAGE_KEY}_${tenantKey}`, JSON.stringify(data));
  } catch (e) {}
}

/**
 * Normalizes HTTP fetch errors into a structured CatalogApiError
 */
async function parseErrorResponse(res: Response): Promise<CatalogApiError> {
  let message = `Request failed with status ${res.status}`;
  let code: CatalogApiError['code'] = 'GENERIC';
  let field: string | undefined;

  try {
    const json = await res.json();
    if (json.error) {
      message = json.error;
    }
  } catch (e) {
    try {
      const text = await res.text();
      if (text) message = text;
    } catch (_) {}
  }

  if (res.status === 409) {
    code = 'CONFLICT';
  } else if (res.status === 422) {
    code = 'VALIDATION';
  } else if (res.status === 401) {
    code = 'UNAUTHORIZED';
  } else if (res.status === 404) {
    code = 'NOT_FOUND';
  } else if (res.status === 503) {
    code = 'SERVICE_UNAVAILABLE';
  }

  // Detect field from validation messages
  if (message.includes('service_id')) field = 'service_id';
  else if (message.includes('metric_id')) field = 'metric_id';
  else if (message.includes('plan_id')) field = 'plan_id';
  else if (message.includes('name')) field = 'name';
  else if (message.includes('unit')) field = 'unit';

  return {
    status: res.status,
    message,
    code,
    field,
  };
}

function getHeaders(apiKey: string): HeadersInit {
  return {
    'Content-Type': 'application/json',
    'X-API-Key': apiKey,
    'Authorization': `Bearer ${apiKey}`,
  };
}

// ---------------------------------------------------------------------------
// 1. Overview API
// ---------------------------------------------------------------------------

export async function fetchCatalogOverview(apiKey: string): Promise<CatalogOverview> {
  try {
    const res = await fetch(`${BASE_URL}/v1/tenant/catalog/overview`, {
      method: 'GET',
      headers: getHeaders(apiKey),
    });

    if (!res.ok) {
      throw await parseErrorResponse(res);
    }

    const data: CatalogOverview = await res.json();
    return {
      tenant_key: data.tenant_key || apiKey,
      services: data.services || [],
      metrics: data.metrics || [],
      plans: data.plans || [],
    };
  } catch (err: any) {
    // If backend is unreachable (offline) or returned an error, fallback to local sandbox store
    if (err?.code === 'UNAUTHORIZED') {
      throw err; // Propagate auth errors
    }
    return getMockData(apiKey);
  }
}

// ---------------------------------------------------------------------------
// 2. Services API
// ---------------------------------------------------------------------------

export async function listServices(apiKey: string): Promise<TenantServiceDescriptor[]> {
  try {
    const res = await fetch(`${BASE_URL}/v1/tenant/catalog/services`, {
      method: 'GET',
      headers: getHeaders(apiKey),
    });
    if (!res.ok) throw await parseErrorResponse(res);
    return await res.json();
  } catch (err: any) {
    if (err?.code === 'UNAUTHORIZED') throw err;
    return getMockData(apiKey).services;
  }
}

export async function registerService(
  apiKey: string,
  descriptor: TenantServiceDescriptor
): Promise<TenantServiceDescriptor> {
  try {
    const res = await fetch(`${BASE_URL}/v1/tenant/catalog/services`, {
      method: 'POST',
      headers: getHeaders(apiKey),
      body: JSON.stringify(descriptor),
    });

    if (!res.ok) {
      throw await parseErrorResponse(res);
    }

    return await res.json();
  } catch (err: any) {
    // If backend failed due to network connection, update mock store
    if (!err?.status) {
      const mock = getMockData(apiKey);
      if (mock.services.some((s) => s.service_id === descriptor.service_id)) {
        const conflictErr: CatalogApiError = {
          status: 409,
          code: 'CONFLICT',
          message: `Service with ID '${descriptor.service_id}' already exists`,
          field: 'service_id',
        };
        throw conflictErr;
      }
      const newSvc: TenantServiceDescriptor = {
        ...descriptor,
        created_at: descriptor.created_at || new Date().toISOString(),
      };
      mock.services.unshift(newSvc);
      saveMockData(apiKey, mock);
      return newSvc;
    }
    throw err;
  }
}

// ---------------------------------------------------------------------------
// 3. Metrics API
// ---------------------------------------------------------------------------

export async function listMetrics(apiKey: string): Promise<TenantMetricDescriptor[]> {
  try {
    const res = await fetch(`${BASE_URL}/v1/tenant/catalog/metrics`, {
      method: 'GET',
      headers: getHeaders(apiKey),
    });
    if (!res.ok) throw await parseErrorResponse(res);
    return await res.json();
  } catch (err: any) {
    if (err?.code === 'UNAUTHORIZED') throw err;
    return getMockData(apiKey).metrics;
  }
}

export async function registerMetric(
  apiKey: string,
  descriptor: TenantMetricDescriptor
): Promise<TenantMetricDescriptor> {
  try {
    const res = await fetch(`${BASE_URL}/v1/tenant/catalog/metrics`, {
      method: 'POST',
      headers: getHeaders(apiKey),
      body: JSON.stringify(descriptor),
    });

    if (!res.ok) {
      throw await parseErrorResponse(res);
    }

    return await res.json();
  } catch (err: any) {
    if (!err?.status) {
      const mock = getMockData(apiKey);
      if (!mock.services.some((s) => s.service_id === descriptor.service_id)) {
        const valErr: CatalogApiError = {
          status: 422,
          code: 'VALIDATION',
          message: `Referenced service '${descriptor.service_id}' does not exist`,
          field: 'service_id',
        };
        throw valErr;
      }
      if (mock.metrics.some((m) => m.metric_id === descriptor.metric_id)) {
        const conflictErr: CatalogApiError = {
          status: 409,
          code: 'CONFLICT',
          message: `Metric with ID '${descriptor.metric_id}' already exists`,
          field: 'metric_id',
        };
        throw conflictErr;
      }
      const newMetric: TenantMetricDescriptor = {
        ...descriptor,
        created_at: descriptor.created_at || new Date().toISOString(),
      };
      mock.metrics.unshift(newMetric);
      saveMockData(apiKey, mock);
      return newMetric;
    }
    throw err;
  }
}

// ---------------------------------------------------------------------------
// 4. Plans API
// ---------------------------------------------------------------------------

export async function listPlans(apiKey: string): Promise<TenantPlanDescriptor[]> {
  try {
    const res = await fetch(`${BASE_URL}/v1/tenant/catalog/plans`, {
      method: 'GET',
      headers: getHeaders(apiKey),
    });
    if (!res.ok) throw await parseErrorResponse(res);
    return await res.json();
  } catch (err: any) {
    if (err?.code === 'UNAUTHORIZED') throw err;
    return getMockData(apiKey).plans;
  }
}

export async function registerPlan(
  apiKey: string,
  descriptor: TenantPlanDescriptor
): Promise<TenantPlanDescriptor> {
  try {
    const res = await fetch(`${BASE_URL}/v1/tenant/catalog/plans`, {
      method: 'POST',
      headers: getHeaders(apiKey),
      body: JSON.stringify(descriptor),
    });

    if (!res.ok) {
      throw await parseErrorResponse(res);
    }

    return await res.json();
  } catch (err: any) {
    if (!err?.status) {
      const mock = getMockData(apiKey);
      if (!mock.services.some((s) => s.service_id === descriptor.service_id)) {
        const valErr: CatalogApiError = {
          status: 422,
          code: 'VALIDATION',
          message: `Referenced service '${descriptor.service_id}' does not exist`,
          field: 'service_id',
        };
        throw valErr;
      }
      if (mock.plans.some((p) => p.plan_id === descriptor.plan_id)) {
        const conflictErr: CatalogApiError = {
          status: 409,
          code: 'CONFLICT',
          message: `Plan with ID '${descriptor.plan_id}' already exists`,
          field: 'plan_id',
        };
        throw conflictErr;
      }
      const newPlan: TenantPlanDescriptor = {
        ...descriptor,
        created_at: descriptor.created_at || new Date().toISOString(),
      };
      mock.plans.unshift(newPlan);
      saveMockData(apiKey, mock);
      return newPlan;
    }
    throw err;
  }
}

// ---------------------------------------------------------------------------
// 5. Batch Catalog Importer
// ---------------------------------------------------------------------------

export async function batchImportCatalog(
  apiKey: string,
  catalogData: {
    services?: TenantServiceDescriptor[];
    metrics?: TenantMetricDescriptor[];
    plans?: TenantPlanDescriptor[];
  }
): Promise<BatchImportResult> {
  const result: BatchImportResult = {
    servicesAdded: 0,
    metricsAdded: 0,
    plansAdded: 0,
    errors: [],
  };

  // 1. Import Services
  if (catalogData.services && Array.isArray(catalogData.services)) {
    for (const svc of catalogData.services) {
      try {
        await registerService(apiKey, svc);
        result.servicesAdded++;
      } catch (err: any) {
        result.errors.push(`Service '${svc.service_id}': ${err.message || err}`);
      }
    }
  }

  // 2. Import Metrics
  if (catalogData.metrics && Array.isArray(catalogData.metrics)) {
    for (const m of catalogData.metrics) {
      try {
        await registerMetric(apiKey, m);
        result.metricsAdded++;
      } catch (err: any) {
        result.errors.push(`Metric '${m.metric_id}': ${err.message || err}`);
      }
    }
  }

  // 3. Import Plans
  if (catalogData.plans && Array.isArray(catalogData.plans)) {
    for (const p of catalogData.plans) {
      try {
        await registerPlan(apiKey, p);
        result.plansAdded++;
      } catch (err: any) {
        result.errors.push(`Plan '${p.plan_id}': ${err.message || err}`);
      }
    }
  }

  return result;
}
