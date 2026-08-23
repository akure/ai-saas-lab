import React, { useState, useEffect } from 'react';
import {
  ShieldCheck,
  ShieldAlert,
  ShieldX,
  Clock,
  Globe,
  Zap,
  RefreshCw,
  Plus,
  ArrowRight,
  Check,
  AlertTriangle,
  Activity,
  Layers,
  Sparkles,
  Database,
  X,
} from 'lucide-react';
import {
  SubscriptionPlan,
  TenantSubscriptionDetails,
  SubscriptionState,
} from '../types/subscription';
import {
  fetchSubscriptionPlans,
  fetchTenantSubscription,
  createSubscriptionContract,
  fireSubscriptionEvent,
  registerSubscriptionPlan,
} from '../services/subscriptionApi';

interface SubscriptionTabProps {
  currentTenantKey: string;
  onTenantKeyChange: (key: string) => void;
  addToast: (title: string, message: string, type?: 'success' | 'info' | 'warning' | 'error') => void;
}

export const SubscriptionTab: React.FC<SubscriptionTabProps> = ({
  currentTenantKey,
  onTenantKeyChange,
  addToast,
}) => {
  const [plans, setPlans] = useState<SubscriptionPlan[]>([]);
  const [contract, setContract] = useState<TenantSubscriptionDetails | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [tenantInput, setTenantInput] = useState<string>(currentTenantKey);
  const [simulatedUsage, setSimulatedUsage] = useState<number>(450000);
  const [eventLogs, setEventLogs] = useState<Array<{ id: string; time: string; text: string; type: 'success' | 'warn' | 'info' | 'danger' }>>([]);
  const [isPlanModalOpen, setIsPlanModalOpen] = useState<boolean>(false);

  // New plan form state
  const [newPlanId, setNewPlanId] = useState<string>('');
  const [newPlanName, setNewPlanName] = useState<string>('');
  const [newQuota, setNewQuota] = useState<number>(25000000);

  const loadData = async (key: string) => {
    setIsLoading(true);
    try {
      const [fetchedPlans, fetchedContract] = await Promise.all([
        fetchSubscriptionPlans(),
        fetchTenantSubscription(key),
      ]);
      setPlans(fetchedPlans);
      setContract(fetchedContract);
    } catch (err: any) {
      addToast('Fetch Error', err.message || 'Failed loading subscription state', 'error');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadData(currentTenantKey);
  }, [currentTenantKey]);

  const addLog = (text: string, type: 'success' | 'warn' | 'info' | 'danger') => {
    const time = new Date().toLocaleTimeString();
    setEventLogs((prev) => [{ id: `log_${Date.now()}_${Math.random()}`, time, text, type }, ...prev.slice(0, 19)]);
  };

  const handleTenantSwitch = (e: React.FormEvent) => {
    e.preventDefault();
    if (!tenantInput.trim()) return;
    onTenantKeyChange(tenantInput.trim());
    addToast('Tenant Selected', `Inspecting subscription for "${tenantInput.trim()}"`, 'info');
  };

  const handleSelectPlan = async (planId: string) => {
    try {
      const updated = await createSubscriptionContract(currentTenantKey, planId);
      setContract(updated);
      addToast('Plan Upgraded', `Tenant "${currentTenantKey}" bound to plan "${planId}"`, 'success');
      addLog(`Updated contract to plan "${planId}"`, 'success');
    } catch (err: any) {
      addToast('Upgrade Failed', err.message || 'Could not update contract plan', 'error');
      addLog(`Failed switching plan: ${err.message}`, 'danger');
    }
  };

  const handleFireFsmEvent = async (event: string, label: string) => {
    try {
      const res = await fireSubscriptionEvent(currentTenantKey, event);
      addToast('FSM State Transition', `${label}: ${res.from} ➔ ${res.to}`, 'info');
      addLog(`Fired FSM Event "${event}": ${res.from} ➔ ${res.to}`, res.to === 'active' ? 'success' : res.to === 'past_due' ? 'warn' : 'danger');
      await loadData(currentTenantKey);
    } catch (err: any) {
      addToast('Transition Failed', err.message || 'FSM transition rejected', 'error');
      addLog(`Rejected FSM event "${event}": ${err.message}`, 'danger');
    }
  };

  const handleCreatePlanSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPlanId.trim() || !newPlanName.trim()) {
      addToast('Validation Error', 'Plan ID and Name are required', 'warning');
      return;
    }
    try {
      const newPlan: SubscriptionPlan = {
        id: newPlanId.trim().toLowerCase(),
        name: newPlanName.trim(),
        entitlements: {
          total_tokens: {
            metric_id: 'total_tokens',
            allowed: true,
            quota: newQuota,
          },
        },
      };
      await registerSubscriptionPlan(newPlan);
      addToast('Plan Registered', `Successfully created "${newPlan.name}" tier`, 'success');
      addLog(`Registered subscription plan "${newPlan.id}"`, 'success');
      setIsPlanModalOpen(false);
      setNewPlanId('');
      setNewPlanName('');
      loadData(currentTenantKey);
    } catch (err: any) {
      addToast('Registration Error', err.message || 'Failed creating plan tier', 'error');
    }
  };

  const getStateBadge = (state: SubscriptionState = 'trial') => {
    switch (state) {
      case 'active':
        return (
          <span className="inline-flex items-center space-x-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-emerald-500/20 text-emerald-300 border border-emerald-500/40">
            <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
            <span className="uppercase tracking-wider">Active</span>
          </span>
        );
      case 'trial':
        return (
          <span className="inline-flex items-center space-x-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-sky-500/20 text-sky-300 border border-sky-500/40">
            <Sparkles className="w-3.5 h-3.5 text-sky-400" />
            <span className="uppercase tracking-wider">Trial</span>
          </span>
        );
      case 'past_due':
        return (
          <span className="inline-flex items-center space-x-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-amber-500/20 text-amber-300 border border-amber-500/40">
            <ShieldAlert className="w-3.5 h-3.5 text-amber-400" />
            <span className="uppercase tracking-wider">Past Due</span>
          </span>
        );
      case 'cancelled':
        return (
          <span className="inline-flex items-center space-x-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-rose-500/20 text-rose-300 border border-rose-500/40">
            <ShieldX className="w-3.5 h-3.5 text-rose-400" />
            <span className="uppercase tracking-wider">Cancelled</span>
          </span>
        );
      default:
        return null;
    }
  };

  // Quota usage calculation
  const activePlan = plans.find((p) => p.id === contract?.plan_id) || plans[0];
  const tokenQuota = activePlan?.entitlements?.total_tokens?.quota || 1000000;
  const usagePercentage = Math.min(100, Math.round((simulatedUsage / tokenQuota) * 100));
  const isOverQuota = simulatedUsage >= tokenQuota;

  return (
    <div className="space-y-8 animate-fadeIn">
      {/* Header Banner */}
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4 p-6 rounded-2xl glass-panel-gold">
        <div>
          <div className="flex items-center space-x-3">
            <div className="p-2.5 rounded-xl bg-gold-500/20 text-gold-400 border border-gold-500/40">
              <Layers className="w-6 h-6" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-slate-100 flex items-center gap-2">
                Subscription & Entitlement Engine
                <span className="text-xs px-2 py-0.5 rounded bg-gold-500/20 text-gold-300 border border-gold-500/40 font-mono">
                  MaaS FSM
                </span>
              </h1>
              <p className="text-xs text-slate-400 mt-0.5">
                Deterministic Finite State Machine (FSM), plan catalog governance, and metric quota evaluator.
              </p>
            </div>
          </div>
        </div>

        {/* Tenant Identity Selector Form */}
        <form onSubmit={handleTenantSwitch} className="flex items-center space-x-2">
          <div className="relative">
            <input
              type="text"
              value={tenantInput}
              onChange={(e) => setTenantInput(e.target.value)}
              placeholder="Tenant Key / API Key"
              className="px-3.5 py-2 pl-9 bg-obsidian-950/90 border border-gold-500/30 rounded-xl text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-gold-400 font-mono w-56 shadow-inner"
            />
            <Activity className="w-4 h-4 text-gold-400 absolute left-3 top-2.5" />
          </div>
          <button
            type="submit"
            className="px-3.5 py-2 rounded-xl text-xs font-semibold bg-gradient-to-r from-gold-500 to-gold-600 hover:from-gold-400 hover:to-gold-500 text-obsidian-950 transition-all duration-200 shadow-gold-sm flex items-center space-x-1.5"
          >
            <span>Inspect</span>
            <ArrowRight className="w-3.5 h-3.5" />
          </button>
        </form>
      </div>

      {/* Main KPI Cards Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-5">
        {/* KPI 1: Subscription Status */}
        <div className="p-5 rounded-2xl bg-obsidian-900/80 border border-gold-500/20 space-y-3 relative overflow-hidden">
          <div className="flex items-center justify-between text-xs text-slate-400">
            <span>FSM State</span>
            <Activity className="w-4 h-4 text-gold-400" />
          </div>
          <div className="flex items-center justify-between">
            {getStateBadge(contract?.state)}
            <span
              className={`text-xs font-mono font-semibold px-2 py-0.5 rounded ${
                contract?.is_usable
                  ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'
                  : 'bg-rose-500/10 text-rose-400 border border-rose-500/30'
              }`}
            >
              {contract?.is_usable ? 'Usable (200)' : 'Blocked (403)'}
            </span>
          </div>
          <p className="text-[11px] text-slate-500">
            ID: <span className="font-mono text-slate-300">{contract?.subscription_id || 'sub_trial'}</span>
          </p>
        </div>

        {/* KPI 2: Active Plan */}
        <div className="p-5 rounded-2xl bg-obsidian-900/80 border border-gold-500/20 space-y-3">
          <div className="flex items-center justify-between text-xs text-slate-400">
            <span>Active Plan Tier</span>
            <Zap className="w-4 h-4 text-amber-400" />
          </div>
          <div className="flex items-baseline space-x-2">
            <span className="text-2xl font-bold text-gold-300 capitalize">
              {contract?.plan_id || 'starter'}
            </span>
            <span className="text-xs text-slate-400">Tier</span>
          </div>
          <p className="text-[11px] text-slate-500">
            Metric Limit: <span className="font-mono text-slate-300">{(tokenQuota / 1000000).toFixed(1)}M Tokens/Cycle</span>
          </p>
        </div>

        {/* KPI 3: Quota Usage */}
        <div className="p-5 rounded-2xl bg-obsidian-900/80 border border-gold-500/20 space-y-3">
          <div className="flex items-center justify-between text-xs text-slate-400">
            <span>Cycle Quota Consumption</span>
            <Database className="w-4 h-4 text-sky-400" />
          </div>
          <div className="flex items-baseline justify-between">
            <span className="text-xl font-bold text-slate-100 font-mono">
              {(simulatedUsage / 1000).toFixed(0)}k / {(tokenQuota / 1000).toFixed(0)}k
            </span>
            <span className={`text-xs font-semibold ${isOverQuota ? 'text-rose-400' : 'text-emerald-400'}`}>
              {usagePercentage}%
            </span>
          </div>
          <div className="w-full bg-obsidian-950 rounded-full h-2 overflow-hidden border border-slate-800">
            <div
              className={`h-full transition-all duration-300 ${
                isOverQuota
                  ? 'bg-rose-500 shadow-[0_0_10px_#F43F5E]'
                  : usagePercentage > 80
                  ? 'bg-amber-500'
                  : 'bg-gradient-to-r from-emerald-500 to-gold-400'
              }`}
              style={{ width: `${usagePercentage}%` }}
            />
          </div>
        </div>

        {/* KPI 4: Anchor & Timezone */}
        <div className="p-5 rounded-2xl bg-obsidian-900/80 border border-gold-500/20 space-y-3">
          <div className="flex items-center justify-between text-xs text-slate-400">
            <span>Billing Cycle Anchor</span>
            <Globe className="w-4 h-4 text-purple-400" />
          </div>
          <div className="flex items-center space-x-2">
            <Clock className="w-4 h-4 text-slate-400" />
            <span className="text-xs font-mono text-slate-200">
              {contract?.anchor_time ? new Date(contract.anchor_time).toLocaleDateString() : 'Active'}
            </span>
          </div>
          <p className="text-[11px] text-slate-500">
            Timezone: <span className="font-mono text-slate-300">{contract?.timezone || 'UTC'}</span>
          </p>
        </div>
      </div>

      {/* Row 2: FSM Lifecycle Control Panel & Event Logs */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left 2 Cols: FSM Action Panel */}
        <div className="lg:col-span-2 p-6 rounded-2xl bg-obsidian-900/80 border border-gold-500/20 space-y-5">
          <div className="flex items-center justify-between">
            <h2 className="text-base font-semibold text-slate-100 flex items-center space-x-2">
              <Activity className="w-4 h-4 text-gold-400" />
              <span>FSM Dunning & Lifecycle Controls</span>
            </h2>
            <button
              onClick={() => loadData(currentTenantKey)}
              className="p-1.5 rounded-lg text-slate-400 hover:text-gold-300 hover:bg-obsidian-800 transition-colors"
              title="Refresh State"
            >
              <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
            </button>
          </div>

          <p className="text-xs text-slate-400">
            Simulate real-time billing webhooks and automated dunning transitions for tenant{' '}
            <span className="font-mono text-gold-300">{currentTenantKey}</span>.
          </p>

          <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
            {/* Activate */}
            <button
              onClick={() => handleFireFsmEvent('activate', 'Payment Success / Activate')}
              className="p-3.5 rounded-xl bg-emerald-500/10 hover:bg-emerald-500/20 border border-emerald-500/30 text-emerald-300 text-xs font-medium transition-all text-left space-y-1 hover:border-emerald-500/60"
            >
              <div className="flex items-center justify-between">
                <span className="font-semibold">Activate Tier</span>
                <Check className="w-3.5 h-3.5 text-emerald-400" />
              </div>
              <p className="text-[10px] text-slate-400">event: "activate"</p>
            </button>

            {/* Payment Failed */}
            <button
              onClick={() => handleFireFsmEvent('payment_failed', 'Payment Failure')}
              className="p-3.5 rounded-xl bg-amber-500/10 hover:bg-amber-500/20 border border-amber-500/30 text-amber-300 text-xs font-medium transition-all text-left space-y-1 hover:border-amber-500/60"
            >
              <div className="flex items-center justify-between">
                <span className="font-semibold">Payment Failure</span>
                <AlertTriangle className="w-3.5 h-3.5 text-amber-400" />
              </div>
              <p className="text-[10px] text-slate-400">event: "payment_failed"</p>
            </button>

            {/* Expire Trial */}
            <button
              onClick={() => handleFireFsmEvent('trial_expired', 'Trial Expiration')}
              className="p-3.5 rounded-xl bg-sky-500/10 hover:bg-sky-500/20 border border-sky-500/30 text-sky-300 text-xs font-medium transition-all text-left space-y-1 hover:border-sky-500/60"
            >
              <div className="flex items-center justify-between">
                <span className="font-semibold">Expire Trial</span>
                <Clock className="w-3.5 h-3.5 text-sky-400" />
              </div>
              <p className="text-[10px] text-slate-400">event: "trial_expired"</p>
            </button>

            {/* Cancel Subscription */}
            <button
              onClick={() => handleFireFsmEvent('cancel', 'Cancel Contract')}
              className="p-3.5 rounded-xl bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/30 text-rose-300 text-xs font-medium transition-all text-left space-y-1 hover:border-rose-500/60"
            >
              <div className="flex items-center justify-between">
                <span className="font-semibold font-mono">Cancel Access</span>
                <ShieldX className="w-3.5 h-3.5 text-rose-400" />
              </div>
              <p className="text-[10px] text-slate-400">event: "cancel"</p>
            </button>

            {/* Reactivate */}
            <button
              onClick={() => handleFireFsmEvent('reactivate', 'Reactivate Contract')}
              className="p-3.5 rounded-xl bg-purple-500/10 hover:bg-purple-500/20 border border-purple-500/30 text-purple-300 text-xs font-medium transition-all text-left space-y-1 hover:border-purple-500/60"
            >
              <div className="flex items-center justify-between">
                <span className="font-semibold font-mono">Reactivate</span>
                <RefreshCw className="w-3.5 h-3.5 text-purple-400" />
              </div>
              <p className="text-[10px] text-slate-400">event: "reactivate"</p>
            </button>

            {/* Payment Succeeded */}
            <button
              onClick={() => handleFireFsmEvent('payment_succeeded', 'Payment Succeeded')}
              className="p-3.5 rounded-xl bg-teal-500/10 hover:bg-teal-500/20 border border-teal-500/30 text-teal-300 text-xs font-medium transition-all text-left space-y-1 hover:border-teal-500/60"
            >
              <div className="flex items-center justify-between">
                <span className="font-semibold">Payment Success</span>
                <ShieldCheck className="w-3.5 h-3.5 text-teal-400" />
              </div>
              <p className="text-[10px] text-slate-400">event: "payment_succeeded"</p>
            </button>
          </div>

          {/* Interactive Quota Simulator Slider */}
          <div className="p-4 rounded-xl bg-obsidian-950/80 border border-slate-800 space-y-3">
            <div className="flex items-center justify-between text-xs">
              <span className="font-semibold text-slate-200">Simulate API Usage (Tokens/Cycle)</span>
              <span className="font-mono text-gold-300">{simulatedUsage.toLocaleString()} tokens</span>
            </div>
            <input
              type="range"
              min="0"
              max={tokenQuota * 1.5}
              step="50000"
              value={simulatedUsage}
              onChange={(e) => setSimulatedUsage(Number(e.target.value))}
              className="w-full accent-gold-400 cursor-pointer"
            />
            <div className="flex items-center justify-between text-[11px] text-slate-400 font-mono">
              <span>0 Tokens</span>
              <span>Quota: {(tokenQuota / 1000000).toFixed(1)}M</span>
              <span>Overage Limit: {((tokenQuota * 1.5) / 1000000).toFixed(1)}M</span>
            </div>
          </div>
        </div>

        {/* Right 1 Col: Event Audit Log */}
        <div className="p-6 rounded-2xl bg-obsidian-900/80 border border-gold-500/20 space-y-4 flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold text-slate-200">FSM Event Audit Log</h3>
              <span className="text-[10px] font-mono text-slate-400">{eventLogs.length} events</span>
            </div>
            <div className="space-y-2 max-h-64 overflow-y-auto pr-1">
              {eventLogs.length === 0 ? (
                <p className="text-xs text-slate-500 italic py-4 text-center">
                  No state transitions recorded yet. Click any FSM control above to simulate events.
                </p>
              ) : (
                eventLogs.map((log) => (
                  <div
                    key={log.id}
                    className={`p-2.5 rounded-lg border text-xs font-mono space-y-0.5 ${
                      log.type === 'success'
                        ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-300'
                        : log.type === 'warn'
                        ? 'bg-amber-500/10 border-amber-500/30 text-amber-300'
                        : log.type === 'danger'
                        ? 'bg-rose-500/10 border-rose-500/30 text-rose-300'
                        : 'bg-slate-800/60 border-slate-700 text-slate-300'
                    }`}
                  >
                    <div className="flex items-center justify-between text-[10px] opacity-75">
                      <span>{log.time}</span>
                      <span>TopicSubscriptionUpdated</span>
                    </div>
                    <p className="leading-tight">{log.text}</p>
                  </div>
                ))
              )}
            </div>
          </div>

          <div className="pt-2 border-t border-slate-800 text-[11px] text-slate-400">
            Publishes <code className="text-gold-300 font-mono">TopicSubscriptionUpdated</code> events on kernel event bus.
          </div>
        </div>
      </div>

      {/* Row 3: Product Pricing Plans Catalog Tiers */}
      <div className="space-y-5">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-base font-semibold text-slate-100">Product Plan Catalog Tiers</h2>
            <p className="text-xs text-slate-400 mt-0.5">
              Available self-service plan tiers and feature entitlement limits.
            </p>
          </div>
          <button
            onClick={() => setIsPlanModalOpen(true)}
            className="px-3.5 py-2 rounded-xl text-xs font-semibold bg-obsidian-800 hover:bg-obsidian-700 text-gold-300 border border-gold-500/40 transition-colors flex items-center space-x-1.5"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>Register Plan</span>
          </button>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {plans.map((plan) => {
            const isCurrent = contract?.plan_id === plan.id;
            const tokenQuotaItem = plan.entitlements?.total_tokens?.quota || 0;
            return (
              <div
                key={plan.id}
                className={`p-6 rounded-2xl transition-all duration-300 flex flex-col justify-between space-y-6 ${
                  isCurrent
                    ? 'bg-gradient-to-b from-gold-500/15 via-obsidian-900/90 to-obsidian-900 border-2 border-gold-400 shadow-gold-md'
                    : 'bg-obsidian-900/80 border border-slate-800 hover:border-gold-500/40'
                }`}
              >
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold text-gold-400 uppercase tracking-wider font-mono">
                      {plan.id}
                    </span>
                    {isCurrent && (
                      <span className="px-2.5 py-0.5 rounded-full text-[10px] font-bold bg-gold-400 text-obsidian-950">
                        Active Contract
                      </span>
                    )}
                  </div>

                  <div>
                    <h3 className="text-lg font-bold text-slate-100">{plan.name}</h3>
                    <p className="text-xs text-slate-400 mt-1">
                      Entitlement limit:{' '}
                      <span className="font-mono text-gold-300">
                        {tokenQuotaItem === 0 ? 'Unlimited' : `${(tokenQuotaItem / 1000000).toFixed(0)}M Tokens/mo`}
                      </span>
                    </p>
                  </div>

                  <div className="space-y-2 pt-2 border-t border-slate-800/80 text-xs">
                    <p className="font-semibold text-slate-300 text-[11px] uppercase tracking-wider">
                      Included Entitlements:
                    </p>
                    <div className="space-y-1.5">
                      <div className="flex items-center justify-between text-slate-300">
                        <span className="flex items-center space-x-2">
                          <Check className="w-3.5 h-3.5 text-emerald-400" />
                          <span>Token Quota</span>
                        </span>
                        <span className="font-mono text-slate-400">{tokenQuotaItem.toLocaleString()}</span>
                      </div>
                      <div className="flex items-center justify-between text-slate-300">
                        <span className="flex items-center space-x-2">
                          <Check className="w-3.5 h-3.5 text-emerald-400" />
                          <span>AI Chat Completion</span>
                        </span>
                        <span className="text-emerald-400 text-[11px]">Permitted</span>
                      </div>
                    </div>
                  </div>
                </div>

                <button
                  disabled={isCurrent}
                  onClick={() => handleSelectPlan(plan.id)}
                  className={`w-full py-2.5 rounded-xl text-xs font-semibold transition-all duration-200 ${
                    isCurrent
                      ? 'bg-gold-500/20 text-gold-300 border border-gold-500/30 cursor-default'
                      : 'bg-gradient-to-r from-gold-500 to-gold-600 hover:from-gold-400 hover:to-gold-500 text-obsidian-950 shadow-gold-sm'
                  }`}
                >
                  {isCurrent ? 'Current Plan' : `Switch to ${plan.name}`}
                </button>
              </div>
            );
          })}
        </div>
      </div>

      {/* Plan Registration Modal */}
      {isPlanModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-fadeIn">
          <div className="w-full max-w-md p-6 rounded-2xl bg-obsidian-900 border border-gold-500/40 shadow-2xl space-y-5">
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <h3 className="text-base font-bold text-slate-100 flex items-center space-x-2">
                <Plus className="w-4 h-4 text-gold-400" />
                <span>Register Custom Subscription Plan</span>
              </h3>
              <button
                onClick={() => setIsPlanModalOpen(false)}
                className="text-slate-400 hover:text-slate-200"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <form onSubmit={handleCreatePlanSubmit} className="space-y-4 text-xs">
              <div className="space-y-1">
                <label className="text-slate-300 font-medium">Plan ID (Slug)</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. enterprise_plus"
                  value={newPlanId}
                  onChange={(e) => setNewPlanId(e.target.value)}
                  className="w-full px-3 py-2 bg-obsidian-950 border border-slate-700 rounded-xl text-slate-200 focus:outline-none focus:border-gold-400 font-mono"
                />
              </div>

              <div className="space-y-1">
                <label className="text-slate-300 font-medium">Plan Name</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. Enterprise Plus Scale"
                  value={newPlanName}
                  onChange={(e) => setNewPlanName(e.target.value)}
                  className="w-full px-3 py-2 bg-obsidian-950 border border-slate-700 rounded-xl text-slate-200 focus:outline-none focus:border-gold-400"
                />
              </div>

              <div className="space-y-1">
                <label className="text-slate-300 font-medium">Monthly Token Quota</label>
                <input
                  type="number"
                  step="1000000"
                  required
                  value={newQuota}
                  onChange={(e) => setNewQuota(Number(e.target.value))}
                  className="w-full px-3 py-2 bg-obsidian-950 border border-slate-700 rounded-xl text-slate-200 focus:outline-none focus:border-gold-400 font-mono"
                />
              </div>

              <div className="flex items-center justify-end space-x-3 pt-3 border-t border-slate-800">
                <button
                  type="button"
                  onClick={() => setIsPlanModalOpen(false)}
                  className="px-4 py-2 rounded-xl text-slate-400 hover:text-slate-200 font-medium"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-xl bg-gradient-to-r from-gold-500 to-gold-600 hover:from-gold-400 hover:to-gold-500 text-obsidian-950 font-semibold shadow-gold-sm"
                >
                  Register Plan
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
