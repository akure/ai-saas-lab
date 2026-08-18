import React from 'react';
import { Key, Sparkles, Database, DollarSign, Activity, ArrowUpRight, Zap } from 'lucide-react';
import { ApiKeyItem, PlanTier, SimulationParams } from '../types';
import { TabId } from './Sidebar';

interface OverviewTabProps {
  apiKeys: ApiKeyItem[];
  simParams: SimulationParams;
  selectedPlan: PlanTier;
  onNavigate: (tab: TabId) => void;
}

export const OverviewTab: React.FC<OverviewTabProps> = ({
  apiKeys,
  simParams,
  selectedPlan,
  onNavigate,
}) => {
  const activeKeys = apiKeys.filter((k) => k.status === 'active');
  const totalTokens = apiKeys.reduce((acc, k) => acc + k.totalTokensUsed, 0) + (simParams.concurrentUsers * simParams.inputTokensPerReq * 24);

  // Dynamic calculations
  const projectedReqsPerMonth = simParams.requestsPerSec * 3600 * 24 * 30;
  const estimatedCost = (projectedReqsPerMonth * (simParams.inputTokensPerReq + simParams.outputTokensPerReq) * 0.0000015) + (simParams.vectorStorageGb * 0.15);

  return (
    <div className="space-y-6">
      {/* Top Banner */}
      <div className="p-6 rounded-2xl glass-panel-gold flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <div className="flex items-center space-x-2">
            <Zap className="w-5 h-5 text-gold-400" />
            <h2 className="text-xl font-bold text-slate-100">Welcome to AI SaaS Lab Client Dashboard</h2>
          </div>
          <p className="text-xs text-slate-300 mt-1">
            Production-grade metering simulation, API key provisioning, real-time analytics & JSON data hub.
          </p>
        </div>
        <button
          onClick={() => onNavigate('metering')}
          className="gold-button px-4 py-2 rounded-xl text-xs flex items-center space-x-2"
        >
          <span>Launch Metering Sliders</span>
          <ArrowUpRight className="w-4 h-4" />
        </button>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Card 1: API Keys */}
        <div className="p-5 rounded-xl glass-panel border border-gold-500/20 hover:border-gold-500/40 transition-all">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-400">Active API Keys</span>
            <div className="p-2 rounded-lg bg-gold-500/10 text-gold-400">
              <Key className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-3">
            <span className="text-2xl font-bold text-slate-100 font-mono">{activeKeys.length}</span>
            <span className="text-xs text-slate-400 ml-2">/ {apiKeys.length} total</span>
          </div>
          <div className="mt-3 flex items-center justify-between text-xs">
            <span className="text-slate-400">Plan Tier:</span>
            <span className="font-semibold text-gold-400 capitalize">{selectedPlan}</span>
          </div>
        </div>

        {/* Card 2: Token Volume */}
        <div className="p-5 rounded-xl glass-panel border border-gold-500/20 hover:border-gold-500/40 transition-all">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-400">Token Volume</span>
            <div className="p-2 rounded-lg bg-gold-500/10 text-gold-400">
              <Sparkles className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-3">
            <span className="text-2xl font-bold text-slate-100 font-mono">{Math.round(totalTokens).toLocaleString()}</span>
          </div>
          <div className="mt-3 flex items-center justify-between text-xs">
            <span className="text-slate-400">Rate Limit Peak:</span>
            <span className="font-semibold text-gold-400">{simParams.requestsPerSec} req/s</span>
          </div>
        </div>

        {/* Card 3: Storage */}
        <div className="p-5 rounded-xl glass-panel border border-gold-500/20 hover:border-gold-500/40 transition-all">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-400">Vector Storage</span>
            <div className="p-2 rounded-lg bg-gold-500/10 text-gold-400">
              <Database className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-3">
            <span className="text-2xl font-bold text-slate-100 font-mono">{simParams.vectorStorageGb} GB</span>
          </div>
          <div className="mt-3 flex items-center justify-between text-xs">
            <span className="text-slate-400">Embeddings:</span>
            <span className="font-semibold text-gold-400">{(simParams.vectorStorageGb * 250000).toLocaleString()}</span>
          </div>
        </div>

        {/* Card 4: Est. Monthly Spend */}
        <div className="p-5 rounded-xl glass-panel border border-gold-500/20 hover:border-gold-500/40 transition-all">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-400">Est. Monthly Spend</span>
            <div className="p-2 rounded-lg bg-gold-500/10 text-gold-400">
              <DollarSign className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-3">
            <span className="text-2xl font-bold text-gold-400 font-mono">${estimatedCost.toFixed(2)}</span>
          </div>
          <div className="mt-3 flex items-center justify-between text-xs">
            <span className="text-slate-400">Burn Rate:</span>
            <span className="font-semibold text-emerald-400">${(estimatedCost / 30).toFixed(2)} / day</span>
          </div>
        </div>
      </div>

      {/* Grid: Quick Actions & Live System Telemetry */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Metering Telemetry Gauge */}
        <div className="lg:col-span-2 p-6 rounded-2xl glass-panel border border-gold-500/20 space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="text-base font-bold text-slate-100 flex items-center space-x-2">
              <Activity className="w-4 h-4 text-gold-400" />
              <span>Simulated Metering Telemetry Overview</span>
            </h3>
            <span className="text-xs text-slate-400 font-mono">Live Sync</span>
          </div>

          <div className="space-y-4 pt-2">
            {/* Gauge 1: Request Throughput */}
            <div>
              <div className="flex justify-between text-xs mb-1.5">
                <span className="text-slate-300">Throughput Capacity ({simParams.requestsPerSec} / 500 req/sec)</span>
                <span className="font-semibold text-gold-400">{Math.round((simParams.requestsPerSec / 500) * 100)}%</span>
              </div>
              <div className="w-full h-2 rounded-full bg-obsidian-800 overflow-hidden border border-slate-700">
                <div
                  className="h-full bg-gradient-to-r from-gold-600 via-gold-500 to-gold-300 transition-all duration-300"
                  style={{ width: `${Math.min((simParams.requestsPerSec / 500) * 100, 100)}%` }}
                />
              </div>
            </div>

            {/* Gauge 2: Storage Quota */}
            <div>
              <div className="flex justify-between text-xs mb-1.5">
                <span className="text-slate-300">Vector Index Capacity ({simParams.vectorStorageGb} / 250 GB)</span>
                <span className="font-semibold text-gold-400">{Math.round((simParams.vectorStorageGb / 250) * 100)}%</span>
              </div>
              <div className="w-full h-2 rounded-full bg-obsidian-800 overflow-hidden border border-slate-700">
                <div
                  className="h-full bg-gradient-to-r from-gold-600 via-gold-400 to-amber-300 transition-all duration-300"
                  style={{ width: `${Math.min((simParams.vectorStorageGb / 250) * 100, 100)}%` }}
                />
              </div>
            </div>

            {/* Gauge 3: Cache Hit Ratio */}
            <div>
              <div className="flex justify-between text-xs mb-1.5">
                <span className="text-slate-300">Cache Hit Ratio ({simParams.cacheHitRatioPercent}%)</span>
                <span className="font-semibold text-emerald-400">High Efficiency</span>
              </div>
              <div className="w-full h-2 rounded-full bg-obsidian-800 overflow-hidden border border-slate-700">
                <div
                  className="h-full bg-gradient-to-r from-emerald-600 to-emerald-400 transition-all duration-300"
                  style={{ width: `${simParams.cacheHitRatioPercent}%` }}
                />
              </div>
            </div>
          </div>
        </div>

        {/* Quick Hub Shortcuts */}
        <div className="p-6 rounded-2xl glass-panel border border-gold-500/20 space-y-4">
          <h3 className="text-base font-bold text-slate-100">Quick Actions</h3>
          
          <div className="space-y-2.5">
            <button
              onClick={() => onNavigate('tenant-catalog')}
              className="w-full flex items-center justify-between p-3 rounded-xl bg-obsidian-900 border border-gold-500/30 hover:border-gold-500/60 hover:bg-obsidian-800 transition-all text-xs font-semibold text-gold-300 shadow-gold-sm"
            >
              <div className="flex items-center space-x-2.5">
                <Database className="w-4 h-4 text-gold-400" />
                <span>Tenant Metering Catalog</span>
              </div>
              <ArrowUpRight className="w-3.5 h-3.5 text-gold-400" />
            </button>

            <button
              onClick={() => onNavigate('api-keys')}
              className="w-full flex items-center justify-between p-3 rounded-xl bg-obsidian-900 border border-gold-500/20 hover:border-gold-500/50 hover:bg-obsidian-800 transition-all text-xs font-medium text-slate-200"
            >
              <div className="flex items-center space-x-2.5">
                <Key className="w-4 h-4 text-gold-400" />
                <span>Create New API Key</span>
              </div>
              <ArrowUpRight className="w-3.5 h-3.5 text-slate-400" />
            </button>

            <button
              onClick={() => onNavigate('json-studio')}
              className="w-full flex items-center justify-between p-3 rounded-xl bg-obsidian-900 border border-gold-500/20 hover:border-gold-500/50 hover:bg-obsidian-800 transition-all text-xs font-medium text-slate-200"
            >
              <div className="flex items-center space-x-2.5">
                <Database className="w-4 h-4 text-gold-400" />
                <span>Import / Export JSON Telemetry</span>
              </div>
              <ArrowUpRight className="w-3.5 h-3.5 text-slate-400" />
            </button>

            <button
              onClick={() => onNavigate('api-tester')}
              className="w-full flex items-center justify-between p-3 rounded-xl bg-obsidian-900 border border-gold-500/20 hover:border-gold-500/50 hover:bg-obsidian-800 transition-all text-xs font-medium text-slate-200"
            >
              <div className="flex items-center space-x-2.5">
                <Zap className="w-4 h-4 text-gold-400" />
                <span>Test Live REST Endpoint</span>
              </div>
              <ArrowUpRight className="w-3.5 h-3.5 text-slate-400" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
