import React, { useState, useEffect } from 'react';
import { X, CreditCard, AlertCircle, Plus, Trash2, ShieldCheck } from 'lucide-react';
import {
  TenantPlanDescriptor,
  TenantServiceDescriptor,
  TenantMetricDescriptor,
  CatalogApiError,
} from '../../types/catalog';

interface PlanModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (plan: TenantPlanDescriptor) => Promise<void>;
  services: TenantServiceDescriptor[];
  metrics: TenantMetricDescriptor[];
  preselectedServiceId?: string;
}

export const PlanModal: React.FC<PlanModalProps> = ({
  isOpen,
  onClose,
  onSubmit,
  services,
  metrics,
  preselectedServiceId,
}) => {
  const [planId, setPlanId] = useState('');
  const [serviceId, setServiceId] = useState('');
  const [name, setName] = useState('');
  const [rates, setRates] = useState<Record<string, number>>({});
  const [quotas, setQuotas] = useState<Record<string, number>>({});
  const [version, setVersion] = useState(1);
  const [active, setActive] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [apiError, setApiError] = useState<CatalogApiError | null>(null);

  useEffect(() => {
    if (preselectedServiceId) {
      setServiceId(preselectedServiceId);
    } else if (services.length > 0 && !serviceId) {
      setServiceId(services[0].service_id);
    }
  }, [preselectedServiceId, services, isOpen]);

  // Filter metrics that belong to the selected service
  const availableMetrics = metrics.filter((m) => m.service_id === serviceId);

  // Auto initialize rates for available metrics if empty
  useEffect(() => {
    if (availableMetrics.length > 0 && Object.keys(rates).length === 0) {
      const initialRates: Record<string, number> = {};
      const initialQuotas: Record<string, number> = {};
      availableMetrics.forEach((m) => {
        initialRates[m.metric_id] = m.unit === 'tokens' ? 0.0015 : 0.01;
        initialQuotas[m.metric_id] = m.unit === 'tokens' ? 10000 : 100;
      });
      setRates(initialRates);
      setQuotas(initialQuotas);
    }
  }, [serviceId, availableMetrics.length]);

  if (!isOpen) return null;

  const isPlanIdValid = planId.trim().length > 0;
  const isServiceValid = serviceId.trim().length > 0;

  const handlePlanIdChange = (val: string) => {
    const sanitized = val.toLowerCase().replace(/\s+/g, '_');
    setPlanId(sanitized);
    if (apiError) setApiError(null);
  };

  const handleRateChange = (mId: string, rateVal: number) => {
    setRates((prev) => ({ ...prev, [mId]: rateVal }));
  };

  const handleQuotaChange = (mId: string, quotaVal: number) => {
    setQuotas((prev) => ({ ...prev, [mId]: quotaVal }));
  };

  const removeRateItem = (mId: string) => {
    setRates((prev) => {
      const next = { ...prev };
      delete next[mId];
      return next;
    });
    setQuotas((prev) => {
      const next = { ...prev };
      delete next[mId];
      return next;
    });
  };

  const addRateItem = (mId: string) => {
    setRates((prev) => ({ ...prev, [mId]: 0.001 }));
    setQuotas((prev) => ({ ...prev, [mId]: 1000 }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isPlanIdValid || !isServiceValid) return;

    setIsSubmitting(true);
    setApiError(null);

    try {
      await onSubmit({
        plan_id: planId.trim(),
        service_id: serviceId.trim(),
        name: name.trim() || planId.trim(),
        rates,
        included_quotas: quotas,
        version: Number(version) || 1,
        active,
      });
      // Reset form
      setPlanId('');
      setName('');
      setRates({});
      setQuotas({});
      setVersion(1);
      setActive(true);
      onClose();
    } catch (err: any) {
      setApiError(err);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-obsidian-950/80 backdrop-blur-md animate-fadeIn">
      <div className="bg-obsidian-900 border border-gold-500/30 rounded-2xl max-w-xl w-full p-6 shadow-2xl shadow-black/80 relative max-h-[90vh] overflow-y-auto">
        {/* Modal Header */}
        <div className="flex items-center justify-between pb-4 border-b border-gold-500/20">
          <div className="flex items-center space-x-3">
            <div className="p-2.5 rounded-xl bg-gold-500/10 text-gold-400 border border-gold-500/30">
              <CreditCard className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-base font-bold text-slate-100">Register Application Plan</h3>
              <p className="text-xs text-slate-400">Configure pricing rates and quota thresholds for end users</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-200 transition-colors p-1"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          {/* Target Service Selection */}
          <div>
            <label className="text-xs font-semibold text-slate-300">
              Target Service <span className="text-rose-400">*</span>
            </label>
            <select
              value={serviceId}
              onChange={(e) => {
                setServiceId(e.target.value);
                setRates({});
                setQuotas({});
              }}
              className="mt-1.5 w-full px-3.5 py-2.5 bg-obsidian-950 border border-gold-500/20 rounded-xl text-xs text-slate-100 focus:outline-none focus:border-gold-500/60 font-mono"
            >
              {services.map((s) => (
                <option key={s.service_id} value={s.service_id} className="bg-obsidian-900">
                  {s.name} ({s.service_id})
                </option>
              ))}
            </select>
          </div>

          {/* Plan ID & Display Name */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="text-xs font-semibold text-slate-300">
                Plan ID <span className="text-rose-400">*</span>
              </label>
              <input
                type="text"
                value={planId}
                onChange={(e) => handlePlanIdChange(e.target.value)}
                placeholder="e.g. pro_tier, enterprise_scale"
                className="mt-1.5 w-full px-3.5 py-2.5 bg-obsidian-950 border border-gold-500/20 rounded-xl text-xs font-mono text-slate-100 placeholder-slate-600 focus:outline-none focus:border-gold-500/60"
              />
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-300">Plan Display Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. Pro Growth Developer Tier"
                className="mt-1.5 w-full px-3.5 py-2.5 bg-obsidian-950 border border-gold-500/20 rounded-xl text-xs text-slate-100 placeholder-slate-600 focus:outline-none focus:border-gold-500/60"
              />
            </div>
          </div>

          {/* Version & Active Status */}
          <div className="grid grid-cols-2 gap-3 p-3 bg-obsidian-950/80 rounded-xl border border-gold-500/15">
            <div>
              <label className="text-[11px] font-semibold text-slate-400">Plan Version</label>
              <input
                type="number"
                min={1}
                value={version}
                onChange={(e) => setVersion(parseInt(e.target.value) || 1)}
                className="mt-1 w-full px-3 py-1.5 bg-obsidian-900 border border-gold-500/20 rounded-lg text-xs font-mono text-slate-100 focus:outline-none focus:border-gold-500/50"
              />
            </div>
            <div className="flex items-center justify-between pt-4">
              <span className="text-xs font-semibold text-slate-300">Active Plan</span>
              <button
                type="button"
                onClick={() => setActive(!active)}
                className={`w-11 h-6 flex items-center rounded-full p-1 transition-colors ${
                  active ? 'bg-gold-500' : 'bg-obsidian-800 border border-slate-700'
                }`}
              >
                <div
                  className={`bg-obsidian-950 w-4 h-4 rounded-full shadow-md transform transition-transform ${
                    active ? 'translate-x-5' : 'translate-x-0'
                  }`}
                />
              </button>
            </div>
          </div>

          {/* Dynamic Metric Rates & Free Quotas Matrix */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-xs font-semibold text-slate-300">
                Metric Pricing & Free Quota Matrix
              </label>
              <span className="text-[11px] text-gold-400 font-mono">
                {Object.keys(rates).length} rated metrics
              </span>
            </div>

            {availableMetrics.length === 0 ? (
              <div className="p-3 rounded-xl bg-amber-950/30 border border-amber-500/30 text-amber-300 text-xs">
                No metrics registered under service '{serviceId}' yet. Please register metrics for this service first.
              </div>
            ) : (
              <div className="space-y-2 max-h-48 overflow-y-auto pr-1">
                {availableMetrics.map((m) => {
                  const isRated = rates[m.metric_id] !== undefined;
                  return (
                    <div
                      key={m.metric_id}
                      className={`p-3 rounded-xl border transition-all ${
                        isRated
                          ? 'bg-obsidian-950 border-gold-500/30'
                          : 'bg-obsidian-950/40 border-slate-800 opacity-60'
                      }`}
                    >
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center space-x-2">
                          <span className="text-xs font-bold font-mono text-gold-300">
                            {m.metric_id}
                          </span>
                          <span className="text-[10px] px-1.5 py-0.5 rounded bg-obsidian-800 text-slate-400 border border-slate-700">
                            {m.unit}
                          </span>
                        </div>
                        {isRated ? (
                          <button
                            type="button"
                            onClick={() => removeRateItem(m.metric_id)}
                            className="text-rose-400 hover:text-rose-300 text-xs p-1"
                            title="Exclude metric from plan"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        ) : (
                          <button
                            type="button"
                            onClick={() => addRateItem(m.metric_id)}
                            className="text-gold-400 hover:text-gold-300 text-xs font-semibold flex items-center space-x-1"
                          >
                            <Plus className="w-3 h-3" />
                            <span>Include in Plan</span>
                          </button>
                        )}
                      </div>

                      {isRated && (
                        <div className="grid grid-cols-2 gap-2 mt-2">
                          <div>
                            <span className="text-[10px] text-slate-400 block mb-1">
                              Cost per {m.unit} ($ USD)
                            </span>
                            <input
                              type="number"
                              step="0.0001"
                              min="0"
                              value={rates[m.metric_id] || 0}
                              onChange={(e) =>
                                handleRateChange(m.metric_id, parseFloat(e.target.value) || 0)
                              }
                              className="w-full px-2.5 py-1.5 bg-obsidian-900 border border-gold-500/20 rounded-lg text-xs font-mono text-slate-100 focus:outline-none focus:border-gold-500/60"
                            />
                          </div>
                          <div>
                            <span className="text-[10px] text-slate-400 block mb-1">
                              Included Free Quota ({m.unit})
                            </span>
                            <input
                              type="number"
                              min="0"
                              value={quotas[m.metric_id] || 0}
                              onChange={(e) =>
                                handleQuotaChange(m.metric_id, parseInt(e.target.value) || 0)
                              }
                              className="w-full px-2.5 py-1.5 bg-obsidian-900 border border-gold-500/20 rounded-lg text-xs font-mono text-slate-100 focus:outline-none focus:border-gold-500/60"
                            />
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Error Banner */}
          {apiError && (
            <div className="p-3 rounded-xl bg-rose-950/60 border border-rose-500/50 flex items-start space-x-2.5 text-rose-200">
              <AlertCircle className="w-4 h-4 text-rose-400 shrink-0 mt-0.5" />
              <div className="text-xs">
                <strong className="font-semibold block">
                  {apiError.code === 'CONFLICT'
                    ? '409 Conflict: Plan Already Exists'
                    : apiError.code === 'VALIDATION'
                    ? '422 Validation Error'
                    : 'Registration Failed'}
                </strong>
                <p className="mt-0.5 opacity-90">{apiError.message}</p>
              </div>
            </div>
          )}

          {/* Submit & Cancel Buttons */}
          <div className="flex items-center justify-end space-x-3 pt-3 border-t border-gold-500/20">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-xl text-xs font-medium text-slate-400 hover:text-slate-200 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!isPlanIdValid || !isServiceValid || isSubmitting}
              className="gold-button px-5 py-2 rounded-xl text-xs font-semibold disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-1.5"
            >
              {isSubmitting ? (
                <span>Registering...</span>
              ) : (
                <>
                  <CreditCard className="w-3.5 h-3.5" />
                  <span>Register Plan</span>
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
