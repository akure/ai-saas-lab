import React, { useState, useEffect, useMemo } from 'react';
import {
  Layers,
  Gauge,
  CreditCard,
  Plus,
  RefreshCw,
  GitFork,
  FileJson,
  Key,
  Calculator,
  ShieldCheck,
  AlertCircle,
  Sparkles,
} from 'lucide-react';
import {
  CatalogOverview,
  TenantServiceDescriptor,
  TenantMetricDescriptor,
  TenantPlanDescriptor,
  CatalogDateFilterState,
  BatchImportResult,
} from '../../types/catalog';
import {
  fetchCatalogOverview,
  registerService,
  registerMetric,
  registerPlan,
  batchImportCatalog,
} from '../../services/catalogApi';
import { CatalogStatsHeader } from './CatalogStatsHeader';
import { CatalogDateFilter } from './CatalogDateFilter';
import { ServiceCardList } from './ServiceCardList';
import { ServiceModal } from './ServiceModal';
import { MetricModal } from './MetricModal';
import { PlanModal } from './PlanModal';
import { CatalogDependencyTree } from './CatalogDependencyTree';
import { CatalogJsonStudio } from './CatalogJsonStudio';
import { MeteringCalculatorModal } from './MeteringCalculatorModal';
import { ApiKeyItem } from '../../types';

interface TenantCatalogTabProps {
  apiKeys: ApiKeyItem[];
  currentTenantKey: string;
  onSelectTenantKey: (key: string) => void;
  onShowToast: (title: string, message: string, type?: 'success' | 'error' | 'info' | 'warning') => void;
}

export const TenantCatalogTab: React.FC<TenantCatalogTabProps> = ({
  apiKeys,
  currentTenantKey,
  onSelectTenantKey,
  onShowToast,
}) => {
  // Views: explorer, tree, json-studio
  const [activeView, setActiveView] = useState<'explorer' | 'tree' | 'json-studio'>('explorer');

  // Core Catalog Data State
  const [overview, setOverview] = useState<CatalogOverview>({
    tenant_key: currentTenantKey || 'demo-key-starter',
    services: [],
    metrics: [],
    plans: [],
  });
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);

  // Search & Date Filter State
  const [searchQuery, setSearchQuery] = useState('');
  const [dateFilter, setDateFilter] = useState<CatalogDateFilterState>({ rangeType: 'all' });
  const [entityTypeFilter, setEntityTypeFilter] = useState<'all' | 'services' | 'metrics' | 'plans'>('all');

  // Modal Visibility States
  const [isServiceModalOpen, setIsServiceModalOpen] = useState(false);
  const [isMetricModalOpen, setIsMetricModalOpen] = useState(false);
  const [isPlanModalOpen, setIsPlanModalOpen] = useState(false);
  const [preselectedServiceId, setPreselectedServiceId] = useState<string | undefined>();
  const [selectedPlanForCalculator, setSelectedPlanForCalculator] = useState<TenantPlanDescriptor | null>(null);

  // Load Catalog Data from Backend / API Client
  const loadData = async (tenantKey: string) => {
    try {
      setIsLoading(true);
      const data = await fetchCatalogOverview(tenantKey);
      setOverview(data);
    } catch (err: any) {
      onShowToast(
        'Catalog Sync Error',
        err.message || 'Failed to fetch tenant catalog data',
        'error'
      );
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  };

  useEffect(() => {
    if (currentTenantKey) {
      loadData(currentTenantKey);
    }
  }, [currentTenantKey]);

  const handleRefresh = () => {
    setIsRefreshing(true);
    loadData(currentTenantKey);
    onShowToast('Catalog Refreshed', 'Synced catalog with backend storage');
  };

  // Handlers for Registration
  const handleRegisterService = async (svc: TenantServiceDescriptor) => {
    const created = await registerService(currentTenantKey, svc);
    setOverview((prev) => ({
      ...prev,
      services: [created, ...prev.services.filter((s) => s.service_id !== created.service_id)],
    }));
    onShowToast('Service Registered', `Service '${created.name}' (${created.service_id}) created successfully`);
  };

  const handleRegisterMetric = async (metric: TenantMetricDescriptor) => {
    const created = await registerMetric(currentTenantKey, metric);
    setOverview((prev) => ({
      ...prev,
      metrics: [created, ...prev.metrics.filter((m) => m.metric_id !== created.metric_id)],
    }));
    onShowToast('Metric Registered', `Metric '${created.name}' (${created.metric_id}) created successfully`);
  };

  const handleRegisterPlan = async (plan: TenantPlanDescriptor) => {
    const created = await registerPlan(currentTenantKey, plan);
    setOverview((prev) => ({
      ...prev,
      plans: [created, ...prev.plans.filter((p) => p.plan_id !== created.plan_id)],
    }));
    onShowToast('Plan Registered', `Plan '${created.name}' (v${created.version}) created successfully`);
  };

  const handleBatchImport = async (jsonPayload: any): Promise<BatchImportResult> => {
    const res = await batchImportCatalog(currentTenantKey, jsonPayload);
    await loadData(currentTenantKey);
    return res;
  };

  // Quick Open Modal Helpers
  const handleOpenMetricModalForService = (serviceId?: string) => {
    setPreselectedServiceId(serviceId);
    setIsMetricModalOpen(true);
  };

  const handleOpenPlanModalForService = (serviceId?: string) => {
    setPreselectedServiceId(serviceId);
    setIsPlanModalOpen(true);
  };

  // Apply Search & Date Filtering to Display List
  const filteredServices = useMemo(() => {
    let list = overview.services;

    // Search filter
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      list = list.filter(
        (s) =>
          s.service_id.toLowerCase().includes(q) ||
          s.name.toLowerCase().includes(q) ||
          (s.description && s.description.toLowerCase().includes(q))
      );
    }

    // Date filter
    if (dateFilter.rangeType !== 'all') {
      const now = Date.now();
      list = list.filter((s) => {
        if (!s.created_at) return true;
        const time = new Date(s.created_at).getTime();
        if (dateFilter.rangeType === 'today') return now - time <= 86400000;
        if (dateFilter.rangeType === '7d') return now - time <= 86400000 * 7;
        if (dateFilter.rangeType === '30d') return now - time <= 86400000 * 30;
        if (dateFilter.rangeType === 'custom' && dateFilter.startDate && dateFilter.endDate) {
          const start = new Date(dateFilter.startDate).getTime();
          const end = new Date(dateFilter.endDate).getTime() + 86400000;
          return time >= start && time <= end;
        }
        return true;
      });
    }

    return list;
  }, [overview.services, searchQuery, dateFilter]);

  return (
    <div className="space-y-6 animate-fadeIn pb-12">
      {/* Top Banner: Tenant Key Switcher & Action Bar */}
      <div className="p-5 rounded-2xl glass-panel border border-gold-500/20 flex flex-col lg:flex-row lg:items-center justify-between gap-4">
        {/* Left: Tenant Context */}
        <div className="flex items-center space-x-3.5">
          <div className="p-3 rounded-xl bg-gold-500/10 text-gold-400 border border-gold-500/30">
            <Key className="w-5 h-5" />
          </div>
          <div>
            <div className="flex items-center space-x-2">
              <h2 className="text-lg font-bold text-slate-100 gold-text-gradient">
                Tenant Metering Catalog
              </h2>
              <span className="text-[10px] px-2 py-0.5 rounded bg-gold-500/15 text-gold-300 font-mono font-semibold">
                MaaS Core
              </span>
            </div>
            <div className="flex items-center space-x-2 mt-1">
              <span className="text-xs text-slate-400">Active Tenant Context:</span>
              <select
                value={currentTenantKey}
                onChange={(e) => onSelectTenantKey(e.target.value)}
                className="bg-obsidian-900 border border-gold-500/30 rounded-lg text-xs font-mono text-gold-300 px-2.5 py-1 focus:outline-none focus:border-gold-500/70 cursor-pointer"
              >
                {apiKeys.map((k) => (
                  <option key={k.id} value={k.key} className="bg-obsidian-900 text-slate-100">
                    {k.name} ({k.key.substring(0, 14)}...)
                  </option>
                ))}
                {/* Fallback default keys if none in list */}
                {!apiKeys.some((k) => k.key === 'demo-key-starter') && (
                  <option value="demo-key-starter" className="bg-obsidian-900 text-slate-100">
                    demo-key-starter (Starter Tier)
                  </option>
                )}
                {!apiKeys.some((k) => k.key === 'demo-key-pro') && (
                  <option value="demo-key-pro" className="bg-obsidian-900 text-slate-100">
                    demo-key-pro (Pro Tier)
                  </option>
                )}
              </select>
            </div>
          </div>
        </div>

        {/* Right: Primary Registration Action Buttons */}
        <div className="flex items-center flex-wrap gap-2">
          <button
            onClick={handleRefresh}
            disabled={isRefreshing}
            className="px-3 py-2 rounded-xl bg-obsidian-900 hover:bg-obsidian-800 text-slate-300 border border-gold-500/20 text-xs font-medium flex items-center space-x-1.5 transition-all"
            title="Refresh Catalog data from backend"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${isRefreshing ? 'animate-spin text-gold-400' : ''}`} />
            <span>Sync</span>
          </button>

          <button
            onClick={() => setIsServiceModalOpen(true)}
            className="gold-button px-3.5 py-2 rounded-xl text-xs font-semibold flex items-center space-x-1.5 shadow-gold-sm"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>+ Service</span>
          </button>

          <button
            onClick={() => handleOpenMetricModalForService()}
            className="px-3.5 py-2 rounded-xl bg-obsidian-800 hover:bg-obsidian-700 text-slate-200 border border-gold-500/30 text-xs font-semibold flex items-center space-x-1.5 transition-all"
          >
            <Gauge className="w-3.5 h-3.5 text-gold-400" />
            <span>+ Metric</span>
          </button>

          <button
            onClick={() => handleOpenPlanModalForService()}
            className="px-3.5 py-2 rounded-xl bg-obsidian-800 hover:bg-obsidian-700 text-slate-200 border border-gold-500/30 text-xs font-semibold flex items-center space-x-1.5 transition-all"
          >
            <CreditCard className="w-3.5 h-3.5 text-gold-400" />
            <span>+ Plan</span>
          </button>
        </div>
      </div>

      {/* KPI Stats Header */}
      <CatalogStatsHeader overview={overview} />

      {/* View Switcher Tabs */}
      <div className="flex items-center space-x-2 border-b border-gold-500/20 pb-3">
        <button
          onClick={() => setActiveView('explorer')}
          className={`px-4 py-2 rounded-xl text-xs font-bold flex items-center space-x-2 transition-all ${
            activeView === 'explorer'
              ? 'bg-gold-500/20 text-gold-300 border border-gold-500/40 shadow-gold-sm'
              : 'text-slate-400 hover:text-slate-200 hover:bg-obsidian-900'
          }`}
        >
          <Layers className="w-4 h-4" />
          <span>Catalog Explorer</span>
        </button>

        <button
          onClick={() => setActiveView('tree')}
          className={`px-4 py-2 rounded-xl text-xs font-bold flex items-center space-x-2 transition-all ${
            activeView === 'tree'
              ? 'bg-gold-500/20 text-gold-300 border border-gold-500/40 shadow-gold-sm'
              : 'text-slate-400 hover:text-slate-200 hover:bg-obsidian-900'
          }`}
        >
          <GitFork className="w-4 h-4" />
          <span>Dependency Graph</span>
        </button>

        <button
          onClick={() => setActiveView('json-studio')}
          className={`px-4 py-2 rounded-xl text-xs font-bold flex items-center space-x-2 transition-all ${
            activeView === 'json-studio'
              ? 'bg-gold-500/20 text-gold-300 border border-gold-500/40 shadow-gold-sm'
              : 'text-slate-400 hover:text-slate-200 hover:bg-obsidian-900'
          }`}
        >
          <FileJson className="w-4 h-4" />
          <span>JSON Studio & SDK</span>
        </button>
      </div>

      {/* View Content Body */}
      {activeView === 'explorer' && (
        <div className="space-y-4">
          <CatalogDateFilter
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            dateFilter={dateFilter}
            onDateFilterChange={setDateFilter}
            entityTypeFilter={entityTypeFilter}
            onEntityTypeFilterChange={setEntityTypeFilter}
          />

          <ServiceCardList
            services={filteredServices}
            metrics={overview.metrics}
            plans={overview.plans}
            onOpenServiceModal={() => setIsServiceModalOpen(true)}
            onOpenMetricModal={handleOpenMetricModalForService}
            onOpenPlanModal={handleOpenPlanModalForService}
            onOpenCalculatorModal={(plan) => setSelectedPlanForCalculator(plan)}
          />
        </div>
      )}

      {activeView === 'tree' && (
        <CatalogDependencyTree
          overview={overview}
          onSelectPlan={(plan) => setSelectedPlanForCalculator(plan)}
        />
      )}

      {activeView === 'json-studio' && (
        <CatalogJsonStudio
          overview={overview}
          dateFilter={dateFilter}
          onImportJson={handleBatchImport}
          onShowToast={onShowToast}
        />
      )}

      {/* Service Registration Modal */}
      <ServiceModal
        isOpen={isServiceModalOpen}
        onClose={() => setIsServiceModalOpen(false)}
        onSubmit={handleRegisterService}
      />

      {/* Metric Registration Modal */}
      <MetricModal
        isOpen={isMetricModalOpen}
        onClose={() => setIsMetricModalOpen(false)}
        onSubmit={handleRegisterMetric}
        services={overview.services}
        preselectedServiceId={preselectedServiceId}
      />

      {/* Plan Registration Modal */}
      <PlanModal
        isOpen={isPlanModalOpen}
        onClose={() => setIsPlanModalOpen(false)}
        onSubmit={handleRegisterPlan}
        services={overview.services}
        metrics={overview.metrics}
        preselectedServiceId={preselectedServiceId}
      />

      {/* Live Metering Calculator Modal */}
      <MeteringCalculatorModal
        isOpen={!!selectedPlanForCalculator}
        onClose={() => setSelectedPlanForCalculator(null)}
        plan={selectedPlanForCalculator}
        metrics={overview.metrics}
        onShowToast={onShowToast}
      />
    </div>
  );
};
