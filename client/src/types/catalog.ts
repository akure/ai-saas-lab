export type ServiceID = string;
export type MetricID = string;
export type ApplicationPlanID = string;

export interface TenantServiceDescriptor {
  service_id: ServiceID;
  name: string;
  description?: string;
  created_at?: string;
}

export interface TenantMetricDescriptor {
  metric_id: MetricID;
  service_id: ServiceID;
  name: string;
  unit: 'tokens' | 'requests' | 'seconds' | 'bytes' | 'custom' | string;
  description?: string;
  created_at?: string;
}

export interface TenantPlanDescriptor {
  plan_id: ApplicationPlanID;
  service_id: ServiceID;
  name: string;
  rates: Record<MetricID, number>; // Cost per unit (e.g. "tokens": 0.002)
  included_quotas: Record<MetricID, number>; // Free quota per cycle (e.g. "tokens": 10000)
  version: number;
  active: boolean;
  created_at?: string;
}

export interface CatalogOverview {
  tenant_key: string;
  services: TenantServiceDescriptor[];
  metrics: TenantMetricDescriptor[];
  plans: TenantPlanDescriptor[];
}

export interface CatalogApiError {
  status: number;
  message: string;
  code?: 'CONFLICT' | 'VALIDATION' | 'UNAUTHORIZED' | 'NOT_FOUND' | 'SERVICE_UNAVAILABLE' | 'GENERIC';
  field?: string;
  details?: string;
}

export interface CatalogDateFilterState {
  rangeType: 'all' | 'today' | '7d' | '30d' | 'custom';
  startDate?: string;
  endDate?: string;
}

export interface BatchImportResult {
  servicesAdded: number;
  metricsAdded: number;
  plansAdded: number;
  errors: string[];
}
