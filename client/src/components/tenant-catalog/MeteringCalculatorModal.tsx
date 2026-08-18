import React, { useState, useEffect } from 'react';
import {
  X,
  Calculator,
  CreditCard,
  Layers,
  Sparkles,
  Zap,
  TrendingUp,
  CheckCircle2,
  Send,
} from 'lucide-react';
import { TenantPlanDescriptor, TenantMetricDescriptor } from '../../types/catalog';

interface MeteringCalculatorModalProps {
  isOpen: boolean;
  onClose: () => void;
  plan: TenantPlanDescriptor | null;
  metrics: TenantMetricDescriptor[];
  onShowToast: (title: string, message: string, type?: 'success' | 'error' | 'info' | 'warning') => void;
}

export const MeteringCalculatorModal: React.FC<MeteringCalculatorModalProps> = ({
  isOpen,
  onClose,
  plan,
  metrics,
  onShowToast,
}) => {
  const [usageInputs, setUsageInputs] = useState<Record<string, number>>({});
  const [isSimulatingApi, setIsSimulatingApi] = useState(false);

  useEffect(() => {
    if (plan) {
      const initial: Record<string, number> = {};
      Object.keys(plan.rates || {}).forEach((mId) => {
        initial[mId] = mId === 'tokens' ? 75000 : 500;
      });
      setUsageInputs(initial);
    }
  }, [plan, isOpen]);

  if (!isOpen || !plan) return null;

  const handleUsageChange = (mId: string, val: number) => {
    setUsageInputs((prev) => ({ ...prev, [mId]: Math.max(0, val) }));
  };

  // Compute breakdown per metric
  let totalCost = 0;
  const breakdown = Object.entries(plan.rates || {}).map(([mId, rate]) => {
    const rawUsage = usageInputs[mId] || 0;
    const quota = plan.included_quotas?.[mId] || 0;
    const billableUsage = Math.max(0, rawUsage - quota);
    const cost = billableUsage * rate;
    totalCost += cost;

    const metricObj = metrics.find((m) => m.metric_id === mId);
    const unit = metricObj?.unit || 'units';

    return {
      metricId: mId,
      unit,
      rawUsage,
      quota,
      billableUsage,
      rate,
      cost,
    };
  });

  const handleSimulateApiEmission = () => {
    setIsSimulatingApi(true);
    setTimeout(() => {
      setIsSimulatingApi(false);
      onShowToast(
        'Usage Ingestion Recorded',
        `Emitted ${Object.keys(usageInputs).length} metering events to MAAS pipeline (Total cost calculated: $${totalCost.toFixed(
          4
        )})`
      );
    }, 600);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-obsidian-950/80 backdrop-blur-md animate-fadeIn">
      <div className="bg-obsidian-900 border border-gold-500/30 rounded-2xl max-w-2xl w-full p-6 shadow-2xl shadow-black/80 relative max-h-[90vh] overflow-y-auto">
        {/* Modal Header */}
        <div className="flex items-center justify-between pb-4 border-b border-gold-500/20">
          <div className="flex items-center space-x-3">
            <div className="p-2.5 rounded-xl bg-gold-500/10 text-gold-400 border border-gold-500/30">
              <Calculator className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center space-x-2">
                <h3 className="text-base font-bold text-slate-100">Live Metering & Cost Calculator</h3>
                <span className="text-[10px] px-2 py-0.5 rounded bg-gold-500/15 text-gold-300 font-mono">
                  {plan.plan_id}
                </span>
              </div>
              <p className="text-xs text-slate-400">
                Simulate customer usage volume against plan '{plan.name}' (v{plan.version})
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-200 transition-colors p-1"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Live Simulation Inputs */}
        <div className="mt-5 space-y-4">
          <div className="flex items-center justify-between">
            <h4 className="text-xs font-bold uppercase tracking-wider text-gold-400">
              Input Simulated Monthly Usage
            </h4>
            <span className="text-[11px] text-slate-400 font-mono">Adjust values below</span>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {breakdown.map((item) => (
              <div key={item.metricId} className="p-3.5 rounded-xl bg-obsidian-950 border border-gold-500/20">
                <div className="flex items-center justify-between mb-1.5">
                  <span className="text-xs font-bold font-mono text-gold-300">{item.metricId}</span>
                  <span className="text-[10px] text-slate-400 font-mono">
                    ${item.rate} / {item.unit}
                  </span>
                </div>

                <div className="flex items-center space-x-2">
                  <input
                    type="number"
                    min={0}
                    value={item.rawUsage}
                    onChange={(e) => handleUsageChange(item.metricId, parseFloat(e.target.value) || 0)}
                    className="w-full px-3 py-2 bg-obsidian-900 border border-gold-500/30 rounded-lg text-xs font-mono text-slate-100 focus:outline-none focus:border-gold-500/70"
                  />
                  <span className="text-[11px] text-slate-400 font-mono shrink-0">{item.unit}</span>
                </div>

                {item.quota > 0 && (
                  <p className="text-[10px] text-emerald-400 mt-1">
                    Free quota: {item.quota.toLocaleString()} {item.unit}
                  </p>
                )}
              </div>
            ))}
          </div>

          {/* Detailed Invoice & Breakdown Card */}
          <div className="mt-6 p-5 rounded-2xl glass-panel-gold border border-gold-500/40 space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <Sparkles className="w-4 h-4 text-gold-400" />
                <h4 className="text-xs font-bold uppercase tracking-wider text-slate-100">
                  Billing Calculation Summary
                </h4>
              </div>
              <div className="text-right">
                <span className="text-[10px] text-slate-400 block uppercase tracking-wider">
                  Total Estimated Bill
                </span>
                <span className="text-2xl font-black gold-text-gradient font-mono">
                  ${totalCost.toFixed(4)}
                </span>
              </div>
            </div>

            {/* Table breakdown */}
            <div className="border-t border-gold-500/20 pt-3 space-y-2">
              {breakdown.map((item) => (
                <div
                  key={item.metricId}
                  className="flex items-center justify-between text-xs font-mono text-slate-300 py-1 border-b border-gold-500/10 last:border-0"
                >
                  <div>
                    <span className="text-gold-300 font-bold">{item.metricId}</span>
                    <span className="text-[10px] text-slate-400 ml-1">
                      ({item.rawUsage.toLocaleString()} - {item.quota.toLocaleString()} free ={' '}
                      {item.billableUsage.toLocaleString()} billable)
                    </span>
                  </div>
                  <div className="text-right font-bold text-slate-100">
                    ${item.cost.toFixed(4)}
                  </div>
                </div>
              ))}
            </div>

            {/* Simulate API Ingestion Button */}
            <div className="pt-2">
              <button
                onClick={handleSimulateApiEmission}
                disabled={isSimulatingApi}
                className="gold-button w-full py-2.5 rounded-xl text-xs font-semibold flex items-center justify-center space-x-2 shadow-gold-sm"
              >
                <Zap className="w-4 h-4" />
                <span>
                  {isSimulatingApi
                    ? 'Recording Metering Telemetry...'
                    : 'Simulate API Metering Event Emission'}
                </span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
