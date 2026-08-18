import React, { useState } from 'react';
import {
  Layers,
  Gauge,
  CreditCard,
  Plus,
  ChevronDown,
  ChevronUp,
  Calendar,
  Sparkles,
  Calculator,
  Code2,
  Check,
} from 'lucide-react';
import {
  TenantServiceDescriptor,
  TenantMetricDescriptor,
  TenantPlanDescriptor,
} from '../../types/catalog';

interface ServiceCardListProps {
  services: TenantServiceDescriptor[];
  metrics: TenantMetricDescriptor[];
  plans: TenantPlanDescriptor[];
  onOpenServiceModal: () => void;
  onOpenMetricModal: (serviceId?: string) => void;
  onOpenPlanModal: (serviceId?: string) => void;
  onOpenCalculatorModal: (plan: TenantPlanDescriptor) => void;
}

export const ServiceCardList: React.FC<ServiceCardListProps> = ({
  services,
  metrics,
  plans,
  onOpenServiceModal,
  onOpenMetricModal,
  onOpenPlanModal,
  onOpenCalculatorModal,
}) => {
  const [expandedServiceIds, setExpandedServiceIds] = useState<Record<string, boolean>>({});
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const toggleExpand = (id: string) => {
    setExpandedServiceIds((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  const handleCopyJson = (obj: any, id: string) => {
    navigator.clipboard.writeText(JSON.stringify(obj, null, 2));
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  if (services.length === 0) {
    return (
      <div className="p-12 text-center rounded-2xl glass-panel border border-dashed border-gold-500/30">
        <div className="w-14 h-14 mx-auto rounded-2xl bg-gold-500/10 text-gold-400 flex items-center justify-center border border-gold-500/30 mb-4 shadow-gold-sm">
          <Layers className="w-7 h-7" />
        </div>
        <h3 className="text-base font-bold text-slate-100">No Services Registered Yet</h3>
        <p className="text-xs text-slate-400 max-w-md mx-auto mt-1.5 leading-relaxed">
          Start building your metering architecture by registering your first core service. Metrics and pricing tiers attach directly to registered services.
        </p>
        <button
          onClick={onOpenServiceModal}
          className="gold-button px-5 py-2.5 rounded-xl text-xs font-semibold mt-5 inline-flex items-center space-x-2 shadow-gold-sm"
        >
          <Plus className="w-4 h-4" />
          <span>Register First Service</span>
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {services.map((svc) => {
        const isExpanded = expandedServiceIds[svc.service_id] !== false; // default expanded
        const serviceMetrics = metrics.filter((m) => m.service_id === svc.service_id);
        const servicePlans = plans.filter((p) => p.service_id === svc.service_id);

        return (
          <div
            key={svc.service_id}
            className="rounded-2xl glass-panel border border-gold-500/20 overflow-hidden hover:border-gold-500/40 transition-all duration-300 shadow-lg"
          >
            {/* Service Header Bar */}
            <div className="p-5 flex flex-col lg:flex-row lg:items-center justify-between gap-4 bg-obsidian-900/60">
              <div className="flex items-start space-x-3.5">
                <div className="p-3 rounded-xl bg-gold-500/10 text-gold-400 border border-gold-500/30 shrink-0">
                  <Layers className="w-5 h-5" />
                </div>
                <div>
                  <div className="flex items-center flex-wrap gap-2">
                    <h3 className="text-base font-bold text-slate-100 font-sans">{svc.name}</h3>
                    <span className="px-2.5 py-0.5 rounded-md bg-gold-500/15 text-gold-300 text-xs font-mono font-bold border border-gold-500/30">
                      {svc.service_id}
                    </span>
                    {svc.created_at && (
                      <span className="text-[11px] text-slate-500 flex items-center space-x-1">
                        <Calendar className="w-3 h-3" />
                        <span>{new Date(svc.created_at).toLocaleDateString()}</span>
                      </span>
                    )}
                  </div>
                  {svc.description && (
                    <p className="text-xs text-slate-400 mt-1 max-w-2xl leading-relaxed">
                      {svc.description}
                    </p>
                  )}
                </div>
              </div>

              {/* Action Buttons */}
              <div className="flex items-center flex-wrap gap-2 self-end lg:self-auto">
                <button
                  onClick={() => onOpenMetricModal(svc.service_id)}
                  className="px-3 py-1.5 rounded-lg bg-obsidian-800 hover:bg-obsidian-700 text-slate-200 border border-slate-700 text-xs font-medium flex items-center space-x-1.5 transition-all"
                  title="Add billable metric to this service"
                >
                  <Gauge className="w-3.5 h-3.5 text-gold-400" />
                  <span>+ Metric</span>
                </button>

                <button
                  onClick={() => onOpenPlanModal(svc.service_id)}
                  className="px-3 py-1.5 rounded-lg bg-obsidian-800 hover:bg-obsidian-700 text-slate-200 border border-slate-700 text-xs font-medium flex items-center space-x-1.5 transition-all"
                  title="Add pricing plan to this service"
                >
                  <CreditCard className="w-3.5 h-3.5 text-gold-400" />
                  <span>+ Plan</span>
                </button>

                <button
                  onClick={() => handleCopyJson(svc, svc.service_id)}
                  className="p-2 rounded-lg bg-obsidian-800 hover:bg-obsidian-700 text-slate-400 hover:text-slate-200 border border-slate-700 transition-all"
                  title="Copy Service JSON"
                >
                  {copiedId === svc.service_id ? (
                    <Check className="w-3.5 h-3.5 text-emerald-400" />
                  ) : (
                    <Code2 className="w-3.5 h-3.5" />
                  )}
                </button>

                <button
                  onClick={() => toggleExpand(svc.service_id)}
                  className="p-2 rounded-lg bg-obsidian-800 hover:bg-obsidian-700 text-slate-400 hover:text-slate-200 border border-slate-700 transition-all"
                >
                  {isExpanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                </button>
              </div>
            </div>

            {/* Nested Content: Metrics & Plans */}
            {isExpanded && (
              <div className="p-5 border-t border-gold-500/15 grid grid-cols-1 lg:grid-cols-2 gap-6 bg-obsidian-950/40">
                {/* 1. Billable Metrics Column */}
                <div>
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center space-x-2">
                      <Gauge className="w-4 h-4 text-gold-400" />
                      <h4 className="text-xs font-bold uppercase tracking-wider text-slate-300">
                        Billable Metrics ({serviceMetrics.length})
                      </h4>
                    </div>
                    <button
                      onClick={() => onOpenMetricModal(svc.service_id)}
                      className="text-[11px] text-gold-400 hover:text-gold-300 font-medium"
                    >
                      + Add Metric
                    </button>
                  </div>

                  {serviceMetrics.length === 0 ? (
                    <div className="p-4 rounded-xl bg-obsidian-900/60 border border-slate-800 text-center text-xs text-slate-400">
                      No metrics attached yet.
                    </div>
                  ) : (
                    <div className="space-y-2">
                      {serviceMetrics.map((m) => (
                        <div
                          key={m.metric_id}
                          className="p-3 rounded-xl bg-obsidian-900/80 border border-slate-800/80 hover:border-gold-500/30 transition-all flex items-start justify-between"
                        >
                          <div>
                            <div className="flex items-center space-x-2">
                              <span className="text-xs font-bold font-mono text-gold-300">
                                {m.metric_id}
                              </span>
                              <span className="text-[10px] px-1.5 py-0.5 rounded bg-obsidian-800 text-slate-300 border border-slate-700 font-mono">
                                {m.unit}
                              </span>
                            </div>
                            <p className="text-[11px] font-semibold text-slate-200 mt-1">{m.name}</p>
                            {m.description && (
                              <p className="text-[11px] text-slate-400 mt-0.5">{m.description}</p>
                            )}
                          </div>
                          <button
                            onClick={() => handleCopyJson(m, m.metric_id)}
                            className="text-slate-500 hover:text-slate-300 p-1"
                            title="Copy metric JSON"
                          >
                            {copiedId === m.metric_id ? (
                              <Check className="w-3.5 h-3.5 text-emerald-400" />
                            ) : (
                              <Code2 className="w-3.5 h-3.5" />
                            )}
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                {/* 2. Application Plans Column */}
                <div>
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center space-x-2">
                      <CreditCard className="w-4 h-4 text-gold-400" />
                      <h4 className="text-xs font-bold uppercase tracking-wider text-slate-300">
                        Application Plans ({servicePlans.length})
                      </h4>
                    </div>
                    <button
                      onClick={() => onOpenPlanModal(svc.service_id)}
                      className="text-[11px] text-gold-400 hover:text-gold-300 font-medium"
                    >
                      + Add Plan
                    </button>
                  </div>

                  {servicePlans.length === 0 ? (
                    <div className="p-4 rounded-xl bg-obsidian-900/60 border border-slate-800 text-center text-xs text-slate-400">
                      No pricing plans attached yet.
                    </div>
                  ) : (
                    <div className="space-y-2.5">
                      {servicePlans.map((p) => (
                        <div
                          key={p.plan_id}
                          className="p-3.5 rounded-xl bg-obsidian-900/90 border border-gold-500/20 hover:border-gold-500/40 transition-all space-y-2"
                        >
                          <div className="flex items-center justify-between">
                            <div className="flex items-center space-x-2">
                              <span className="text-xs font-bold text-slate-100">{p.name}</span>
                              <span className="text-[10px] px-1.5 py-0.5 rounded bg-gold-500/10 text-gold-300 border border-gold-500/30 font-mono">
                                v{p.version}
                              </span>
                              <span
                                className={`text-[10px] px-1.5 py-0.5 rounded font-semibold ${
                                  p.active
                                    ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'
                                    : 'bg-slate-800 text-slate-400'
                                }`}
                              >
                                {p.active ? 'Active' : 'Inactive'}
                              </span>
                            </div>

                            <button
                              onClick={() => onOpenCalculatorModal(p)}
                              className="px-2.5 py-1 rounded-md bg-gold-500/10 hover:bg-gold-500/20 text-gold-300 border border-gold-500/30 text-[11px] font-semibold flex items-center space-x-1 transition-all"
                              title="Calculate usage and simulate billing against this plan"
                            >
                              <Calculator className="w-3 h-3" />
                              <span>Simulate</span>
                            </button>
                          </div>

                          {/* Rate Badges */}
                          <div className="flex flex-wrap gap-1.5 pt-1">
                            {Object.entries(p.rates || {}).map(([metricId, rateVal]) => {
                              const quota = p.included_quotas?.[metricId];
                              return (
                                <div
                                  key={metricId}
                                  className="text-[10px] px-2 py-1 rounded-lg bg-obsidian-950 border border-gold-500/15 text-slate-300 flex items-center space-x-1.5 font-mono"
                                >
                                  <span className="text-gold-400">{metricId}:</span>
                                  <span className="text-slate-100 font-bold">${rateVal}</span>
                                  {quota !== undefined && quota > 0 && (
                                    <span className="text-emerald-400">({quota} free)</span>
                                  )}
                                </div>
                              );
                            })}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};
