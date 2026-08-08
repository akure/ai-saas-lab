import React from 'react';
import { ShieldCheck, Server, Sparkles, Download, RefreshCw, Key } from 'lucide-react';
import { PlanTier } from '../types';

interface HeaderProps {
  isBackendOnline: boolean;
  selectedPlan: PlanTier;
  onPlanChange: (plan: PlanTier) => void;
  activeKeysCount: number;
  totalTokensUsed: number;
  onQuickExportJson: () => void;
  onRefreshHealth: () => void;
}

export const Header: React.FC<HeaderProps> = ({
  isBackendOnline,
  selectedPlan,
  onPlanChange,
  activeKeysCount,
  totalTokensUsed,
  onQuickExportJson,
  onRefreshHealth,
}) => {
  return (
    <header className="sticky top-0 z-30 bg-obsidian-950/90 backdrop-blur-md border-b border-gold-500/20 px-6 py-4">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        {/* Brand & Status */}
        <div className="flex items-center space-x-4">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-gold-400 via-gold-500 to-gold-700 flex items-center justify-center shadow-gold-glow text-obsidian-950 font-extrabold text-xl">
            ⚡
          </div>
          <div>
            <div className="flex items-center space-x-2">
              <h1 className="text-xl font-bold tracking-tight gold-text-gradient">
                AI SaaS Lab Client
              </h1>
              <span className="px-2 py-0.5 text-xs font-semibold rounded-md bg-gold-500/10 text-gold-400 border border-gold-500/30">
                PROD v1.0
              </span>
            </div>
            <p className="text-xs text-slate-400">Metering Services & Key Management Console</p>
          </div>
        </div>

        {/* Live Metrics Quick Badges & Controls */}
        <div className="flex items-center flex-wrap gap-3">
          {/* Backend Connection Status Badge */}
          <button
            onClick={onRefreshHealth}
            title="Click to re-check Go Backend status"
            className={`flex items-center space-x-2 px-3 py-1.5 rounded-lg border text-xs font-medium transition-all ${
              isBackendOnline
                ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400 hover:bg-emerald-500/20'
                : 'bg-amber-500/10 border-amber-500/30 text-amber-400 hover:bg-amber-500/20'
            }`}
          >
            <Server className={`w-3.5 h-3.5 ${isBackendOnline ? 'text-emerald-400' : 'text-amber-400'}`} />
            <span>{isBackendOnline ? 'Backend Online (Port 8080)' : 'Standalone Simulation Mode'}</span>
            <RefreshCw className="w-3 h-3 ml-1 opacity-70" />
          </button>

          {/* Quick Stats */}
          <div className="hidden lg:flex items-center space-x-4 px-3 py-1.5 rounded-lg bg-obsidian-900 border border-gold-500/20 text-xs">
            <div className="flex items-center space-x-1.5 text-slate-300">
              <Key className="w-3.5 h-3.5 text-gold-500" />
              <span>Keys: <strong className="text-gold-400 font-mono">{activeKeysCount}</strong></span>
            </div>
            <div className="h-3 w-px bg-gold-500/20" />
            <div className="flex items-center space-x-1.5 text-slate-300">
              <Sparkles className="w-3.5 h-3.5 text-gold-500" />
              <span>Tokens: <strong className="text-gold-400 font-mono">{totalTokensUsed.toLocaleString()}</strong></span>
            </div>
          </div>

          {/* Plan Tier Selector */}
          <div className="flex items-center rounded-lg bg-obsidian-900 border border-gold-500/20 p-1">
            {(['free', 'pro', 'enterprise'] as PlanTier[]).map((tier) => (
              <button
                key={tier}
                onClick={() => onPlanChange(tier)}
                className={`px-2.5 py-1 text-xs font-semibold rounded-md capitalize transition-all ${
                  selectedPlan === tier
                    ? 'bg-gradient-to-r from-gold-500 to-gold-600 text-obsidian-950 shadow-sm'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                {tier}
              </button>
            ))}
          </div>

          {/* Download JSON Quick Action */}
          <button
            onClick={onQuickExportJson}
            className="gold-button flex items-center space-x-1.5 px-3 py-1.5 rounded-lg text-xs"
          >
            <Download className="w-3.5 h-3.5" />
            <span>Export JSON</span>
          </button>
        </div>
      </div>
    </header>
  );
};
