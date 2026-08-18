import React from 'react';
import { Layers, Gauge, CreditCard, Database, TrendingUp } from 'lucide-react';
import { CatalogOverview } from '../../types/catalog';

interface CatalogStatsHeaderProps {
  overview: CatalogOverview;
}

export const CatalogStatsHeader: React.FC<CatalogStatsHeaderProps> = ({ overview }) => {
  const serviceCount = overview.services.length;
  const metricCount = overview.metrics.length;
  const planCount = overview.plans.length;
  const activePlansCount = overview.plans.filter((p) => p.active).length;

  // Calculate distinct units
  const unitsCount = new Set(overview.metrics.map((m) => m.unit)).size;

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {/* Services Card */}
      <div className="p-4 rounded-xl glass-panel relative overflow-hidden group hover:border-gold-500/40 transition-all duration-300">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
            Registered Services
          </span>
          <div className="p-2 rounded-lg bg-gold-500/10 text-gold-400 group-hover:bg-gold-500/20 transition-all">
            <Layers className="w-4 h-4" />
          </div>
        </div>
        <div className="mt-2 flex items-baseline space-x-2">
          <span className="text-2xl font-extrabold text-slate-100 font-mono">{serviceCount}</span>
          <span className="text-xs text-emerald-400 font-medium">Core Modules</span>
        </div>
        <p className="text-[11px] text-slate-400 mt-1">APIs & Microservices metered</p>
        <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-gradient-to-r from-gold-500/0 via-gold-500/40 to-gold-500/0" />
      </div>

      {/* Metrics Card */}
      <div className="p-4 rounded-xl glass-panel relative overflow-hidden group hover:border-gold-500/40 transition-all duration-300">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
            Billable Metrics
          </span>
          <div className="p-2 rounded-lg bg-gold-500/10 text-gold-400 group-hover:bg-gold-500/20 transition-all">
            <Gauge className="w-4 h-4" />
          </div>
        </div>
        <div className="mt-2 flex items-baseline space-x-2">
          <span className="text-2xl font-extrabold text-slate-100 font-mono">{metricCount}</span>
          <span className="text-xs text-gold-400 font-medium font-mono">{unitsCount} unit types</span>
        </div>
        <p className="text-[11px] text-slate-400 mt-1">Tokens, requests, seconds & bytes</p>
        <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-gradient-to-r from-gold-500/0 via-gold-500/40 to-gold-500/0" />
      </div>

      {/* Plans Card */}
      <div className="p-4 rounded-xl glass-panel relative overflow-hidden group hover:border-gold-500/40 transition-all duration-300">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
            Application Plans
          </span>
          <div className="p-2 rounded-lg bg-gold-500/10 text-gold-400 group-hover:bg-gold-500/20 transition-all">
            <CreditCard className="w-4 h-4" />
          </div>
        </div>
        <div className="mt-2 flex items-baseline space-x-2">
          <span className="text-2xl font-extrabold text-slate-100 font-mono">{planCount}</span>
          <span className="text-xs text-emerald-400 font-medium">{activePlansCount} Active</span>
        </div>
        <p className="text-[11px] text-slate-400 mt-1">Tiered rates & free quota limits</p>
        <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-gradient-to-r from-gold-500/0 via-gold-500/40 to-gold-500/0" />
      </div>

      {/* Storage & L1/L2 Cache Card */}
      <div className="p-4 rounded-xl glass-panel relative overflow-hidden group hover:border-gold-500/40 transition-all duration-300">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
            Catalog Storage Engine
          </span>
          <div className="p-2 rounded-lg bg-gold-500/10 text-gold-400 group-hover:bg-gold-500/20 transition-all">
            <Database className="w-4 h-4" />
          </div>
        </div>
        <div className="mt-2 flex items-baseline space-x-2">
          <span className="text-lg font-bold text-slate-100 font-mono">L1 + L2 + L3</span>
          <span className="text-xs text-emerald-400 font-medium">WAL Sync</span>
        </div>
        <p className="text-[11px] text-slate-400 mt-1">Multi-tier memory, Redis & Postgres</p>
        <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-gradient-to-r from-gold-500/0 via-gold-500/40 to-gold-500/0" />
      </div>
    </div>
  );
};
