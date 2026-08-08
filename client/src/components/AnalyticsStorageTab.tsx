import React from 'react';
import { BarChart3, Database, HardDrive, Cpu, ShieldAlert } from 'lucide-react';
import { SimulationParams } from '../types';

interface AnalyticsStorageTabProps {
  simParams: SimulationParams;
}

export const AnalyticsStorageTab: React.FC<AnalyticsStorageTabProps> = ({ simParams }) => {
  // Simulated hourly data points for SVG line chart
  const hourlyData = [
    { hour: '00:00', reqs: 42, tokens: 42000 },
    { hour: '04:00', reqs: 18, tokens: 18000 },
    { hour: '08:00', reqs: 120, tokens: 120000 },
    { hour: '12:00', reqs: 340, tokens: 340000 },
    { hour: '16:00', reqs: simParams.requestsPerSec * 1.2, tokens: simParams.requestsPerSec * 1200 },
    { hour: '20:00', reqs: simParams.requestsPerSec * 0.9, tokens: simParams.requestsPerSec * 900 },
  ];

  const maxReq = Math.max(...hourlyData.map((d) => d.reqs), 10);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-xl font-bold text-slate-100 flex items-center space-x-2">
          <BarChart3 className="w-5 h-5 text-gold-400" />
          <span>Usage Stats & Storage Analytics</span>
        </h2>
        <p className="text-xs text-slate-400 mt-1">
          Detailed metrics breakdown for token throughput, vector database allocation, and quota usage.
        </p>
      </div>

      {/* SVG Traffic & Token Volume Chart */}
      <div className="p-6 rounded-2xl glass-panel border border-gold-500/20 space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-bold text-slate-100">Hourly Request Traffic (Last 24 Hours)</h3>
          <span className="text-xs font-mono text-gold-400">Unit: Requests / min</span>
        </div>

        {/* SVG Chart */}
        <div className="h-48 w-full pt-4 relative">
          <div className="absolute inset-0 flex flex-col justify-between pointer-events-none opacity-20">
            <div className="border-b border-gold-500 w-full" />
            <div className="border-b border-gold-500 w-full" />
            <div className="border-b border-gold-500 w-full" />
          </div>

          <div className="h-full flex items-end justify-between gap-4 relative z-10 px-2">
            {hourlyData.map((point, idx) => {
              const heightPercent = Math.min((point.reqs / maxReq) * 100, 100);
              return (
                <div key={idx} className="flex-1 flex flex-col items-center group">
                  <div className="text-[10px] font-mono text-gold-300 opacity-0 group-hover:opacity-100 transition-opacity mb-1">
                    {Math.round(point.reqs)} req
                  </div>
                  <div
                    className="w-full max-w-[40px] rounded-t-lg bg-gradient-to-t from-gold-700 via-gold-500 to-gold-300 transition-all duration-300 hover:shadow-gold-glow"
                    style={{ height: `${Math.max(heightPercent, 8)}%` }}
                  />
                  <span className="text-[10px] font-mono text-slate-400 mt-2">{point.hour}</span>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* Storage Breakdown Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Card 1: Vector Storage */}
        <div className="p-5 rounded-xl glass-panel border border-gold-500/20 space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-slate-300 flex items-center space-x-2">
              <Database className="w-4 h-4 text-gold-400" />
              <span>Vector Database</span>
            </span>
            <span className="text-xs font-mono font-bold text-gold-400">{simParams.vectorStorageGb} GB</span>
          </div>
          <p className="text-[11px] text-slate-400">
            HNSW vector indexes for fast approximate nearest neighbor similarity searches.
          </p>
          <div className="w-full h-1.5 rounded-full bg-obsidian-800 overflow-hidden">
            <div className="h-full bg-gold-400" style={{ width: `${(simParams.vectorStorageGb / 250) * 100}%` }} />
          </div>
        </div>

        {/* Card 2: Document Embeddings */}
        <div className="p-5 rounded-xl glass-panel border border-gold-500/20 space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-slate-300 flex items-center space-x-2">
              <HardDrive className="w-4 h-4 text-gold-400" />
              <span>Document Cache</span>
            </span>
            <span className="text-xs font-mono font-bold text-gold-400">
              {(simParams.vectorStorageGb * 0.4).toFixed(1)} GB
            </span>
          </div>
          <p className="text-[11px] text-slate-400">
            Raw text chunks & tokenized context cache for retrieval augmented generation (RAG).
          </p>
          <div className="w-full h-1.5 rounded-full bg-obsidian-800 overflow-hidden">
            <div className="h-full bg-gold-500" style={{ width: '40%' }} />
          </div>
        </div>

        {/* Card 3: Key Index & Audit Store */}
        <div className="p-5 rounded-xl glass-panel border border-gold-500/20 space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-slate-300 flex items-center space-x-2">
              <Cpu className="w-4 h-4 text-gold-400" />
              <span>Audit Log Store</span>
            </span>
            <span className="text-xs font-mono font-bold text-gold-400">128 MB</span>
          </div>
          <p className="text-[11px] text-slate-400">
            Encrypted API request audit logs & tenant metering transactions log.
          </p>
          <div className="w-full h-1.5 rounded-full bg-obsidian-800 overflow-hidden">
            <div className="h-full bg-emerald-400" style={{ width: '15%' }} />
          </div>
        </div>
      </div>
    </div>
  );
};
