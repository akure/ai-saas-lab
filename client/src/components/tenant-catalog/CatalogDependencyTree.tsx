import React, { useState } from 'react';
import {
  Layers,
  Gauge,
  CreditCard,
  ChevronRight,
  Sparkles,
  Search,
  ExternalLink,
  Shield,
} from 'lucide-react';
import {
  CatalogOverview,
  TenantServiceDescriptor,
  TenantMetricDescriptor,
  TenantPlanDescriptor,
} from '../../types/catalog';

interface CatalogDependencyTreeProps {
  overview: CatalogOverview;
  onSelectPlan: (plan: TenantPlanDescriptor) => void;
}

export const CatalogDependencyTree: React.FC<CatalogDependencyTreeProps> = ({
  overview,
  onSelectPlan,
}) => {
  const [selectedServiceId, setSelectedServiceId] = useState<string>(
    overview.services[0]?.service_id || ''
  );

  const selectedService = overview.services.find((s) => s.service_id === selectedServiceId);
  const relatedMetrics = overview.metrics.filter((m) => m.service_id === selectedServiceId);
  const relatedPlans = overview.plans.filter((p) => p.service_id === selectedServiceId);

  if (overview.services.length === 0) {
    return (
      <div className="p-8 text-center glass-panel rounded-2xl">
        <p className="text-xs text-slate-400">Register at least one service to view dependency tree.</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Root Overview Banner */}
      <div className="p-5 rounded-2xl glass-panel-gold border border-gold-500/30 flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="flex items-center space-x-3.5">
          <div className="w-10 h-10 rounded-xl bg-gold-500/20 text-gold-300 border border-gold-500/40 flex items-center justify-center font-mono font-bold">
            <Shield className="w-5 h-5 text-gold-400" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-slate-100">Tenant Architecture Graph</h3>
            <p className="text-xs text-slate-400 font-mono">
              Tenant Key: <strong className="text-gold-300">{overview.tenant_key}</strong>
            </p>
          </div>
        </div>

        <div className="flex items-center space-x-2 text-xs font-mono text-slate-300 bg-obsidian-950/80 px-3 py-1.5 rounded-xl border border-gold-500/20">
          <span>{overview.services.length} Services</span>
          <span className="text-gold-500">→</span>
          <span>{overview.metrics.length} Metrics</span>
          <span className="text-gold-500">→</span>
          <span>{overview.plans.length} Plans</span>
        </div>
      </div>

      {/* Interactive Relational Tree Explorer */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Step 1: Select Service Node (4 Cols) */}
        <div className="lg:col-span-4 space-y-3">
          <div className="flex items-center space-x-2 text-xs font-bold uppercase tracking-wider text-gold-400">
            <Layers className="w-4 h-4" />
            <span>1. Services (Parent Node)</span>
          </div>

          <div className="space-y-2">
            {overview.services.map((s) => {
              const isSelected = s.service_id === selectedServiceId;
              const countM = overview.metrics.filter((m) => m.service_id === s.service_id).length;
              const countP = overview.plans.filter((p) => p.service_id === s.service_id).length;

              return (
                <button
                  key={s.service_id}
                  onClick={() => setSelectedServiceId(s.service_id)}
                  className={`w-full text-left p-4 rounded-xl border transition-all flex items-center justify-between ${
                    isSelected
                      ? 'bg-gradient-to-r from-gold-500/20 to-gold-700/10 border-gold-500/60 shadow-gold-sm text-gold-200'
                      : 'bg-obsidian-900/80 border-slate-800 text-slate-300 hover:border-gold-500/30'
                  }`}
                >
                  <div>
                    <div className="text-xs font-bold">{s.name}</div>
                    <div className="text-[11px] font-mono text-slate-400 mt-0.5">{s.service_id}</div>
                  </div>
                  <div className="flex items-center space-x-2">
                    <span className="text-[10px] px-2 py-0.5 rounded-full bg-obsidian-950 font-mono text-gold-400 border border-gold-500/20">
                      {countM}m / {countP}p
                    </span>
                    <ChevronRight className={`w-4 h-4 ${isSelected ? 'text-gold-400' : 'text-slate-600'}`} />
                  </div>
                </button>
              );
            })}
          </div>
        </div>

        {/* Step 2 & 3: Dependent Metrics & Plans (8 Cols) */}
        <div className="lg:col-span-8 space-y-6">
          {selectedService && (
            <>
              {/* Connected Metrics Branch */}
              <div className="space-y-3">
                <div className="flex items-center space-x-2 text-xs font-bold uppercase tracking-wider text-gold-400">
                  <Gauge className="w-4 h-4" />
                  <span>2. Attached Metrics for '{selectedService.name}'</span>
                </div>

                {relatedMetrics.length === 0 ? (
                  <div className="p-4 rounded-xl bg-obsidian-900 border border-slate-800 text-xs text-slate-400">
                    No metrics linked to this service.
                  </div>
                ) : (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    {relatedMetrics.map((m) => (
                      <div
                        key={m.metric_id}
                        className="p-3.5 rounded-xl bg-obsidian-900/90 border border-gold-500/20 hover:border-gold-500/40 transition-all"
                      >
                        <div className="flex items-center justify-between">
                          <span className="text-xs font-bold font-mono text-gold-300">{m.metric_id}</span>
                          <span className="text-[10px] px-1.5 py-0.5 rounded bg-obsidian-950 text-slate-300 font-mono border border-slate-700">
                            unit: {m.unit}
                          </span>
                        </div>
                        <p className="text-xs font-semibold text-slate-200 mt-1">{m.name}</p>
                        {m.description && (
                          <p className="text-[11px] text-slate-400 mt-0.5 line-clamp-2">{m.description}</p>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Connected Plans Branch */}
              <div className="space-y-3">
                <div className="flex items-center space-x-2 text-xs font-bold uppercase tracking-wider text-gold-400">
                  <CreditCard className="w-4 h-4" />
                  <span>3. Pricing Plans Configured for '{selectedService.name}'</span>
                </div>

                {relatedPlans.length === 0 ? (
                  <div className="p-4 rounded-xl bg-obsidian-900 border border-slate-800 text-xs text-slate-400">
                    No plans linked to this service.
                  </div>
                ) : (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    {relatedPlans.map((p) => (
                      <div
                        key={p.plan_id}
                        onClick={() => onSelectPlan(p)}
                        className="p-4 rounded-xl bg-obsidian-900/90 border border-gold-500/30 hover:border-gold-500/60 transition-all cursor-pointer group shadow-md"
                      >
                        <div className="flex items-center justify-between">
                          <span className="text-xs font-bold text-slate-100 group-hover:text-gold-300 transition-colors">
                            {p.name}
                          </span>
                          <span className="text-[10px] px-1.5 py-0.5 rounded bg-gold-500/15 text-gold-300 font-mono">
                            v{p.version}
                          </span>
                        </div>

                        <div className="mt-3 space-y-1.5">
                          {Object.entries(p.rates || {}).map(([metricId, rateVal]) => (
                            <div
                              key={metricId}
                              className="text-[11px] flex items-center justify-between text-slate-300 font-mono bg-obsidian-950 px-2 py-1 rounded-md"
                            >
                              <span>{metricId}</span>
                              <span className="text-gold-400 font-bold">${rateVal}</span>
                            </div>
                          ))}
                        </div>

                        <div className="mt-3 pt-2 border-t border-slate-800 flex items-center justify-between text-[11px] text-gold-400 font-medium">
                          <span>Click to Simulate Metering</span>
                          <ExternalLink className="w-3 h-3" />
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
};
