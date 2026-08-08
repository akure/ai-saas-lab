import React from 'react';
import { Sliders, Zap, Database, DollarSign, Activity, RotateCcw, Download, Sparkles, Layers } from 'lucide-react';
import { SimulationParams } from '../types';

interface MeteringSimulatorTabProps {
  simParams: SimulationParams;
  onSimParamsChange: (params: SimulationParams) => void;
  onRunBatchSimulation: () => void;
  onExportJson: () => void;
  onResetParams: () => void;
}

export const MeteringSimulatorTab: React.FC<MeteringSimulatorTabProps> = ({
  simParams,
  onSimParamsChange,
  onRunBatchSimulation,
  onExportJson,
  onResetParams,
}) => {
  const updateField = <K extends keyof SimulationParams>(field: K, value: SimulationParams[K]) => {
    onSimParamsChange({ ...simParams, [field]: value });
  };

  // Dynamic calculations
  const totalTokensPerReq = simParams.inputTokensPerReq + simParams.outputTokensPerReq;
  const totalTokensPerSec = simParams.requestsPerSec * totalTokensPerReq;
  const bandwidthMbps = ((totalTokensPerSec * 4) / (1024 * 1024)).toFixed(2);

  // Model multiplier
  const modelRateMultiplier =
    simParams.modelTier === 'ultra-vision' ? 3.5 : simParams.modelTier === 'pro-reasoning' ? 1.8 : 0.8;

  const reqsPerDay = simParams.requestsPerSec * 86400;
  const dailyTokens = reqsPerDay * totalTokensPerReq;
  const effectiveDailyTokens = dailyTokens * (1 - (simParams.cacheHitRatioPercent / 100) * 0.7);

  const dailyCost = (effectiveDailyTokens / 1_000_000) * 1.5 * modelRateMultiplier + simParams.vectorStorageGb * 0.005;
  const monthlyCost = dailyCost * 30;

  const vectorEmbeddingsCount = Math.round(simParams.vectorStorageGb * 250_000);

  return (
    <div className="space-y-6">
      {/* Header Banner */}
      <div className="p-6 rounded-2xl glass-panel-gold flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <div className="flex items-center space-x-2">
            <Sliders className="w-5 h-5 text-gold-400" />
            <h2 className="text-xl font-bold text-slate-100">Flexible Metering & Usage Simulator</h2>
          </div>
          <p className="text-xs text-slate-300 mt-1">
            Adjust real-time load sliders to simulate token consumption, vector storage allocation, latency, and costs.
          </p>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={onResetParams}
            className="p-2 rounded-xl bg-obsidian-900 hover:bg-obsidian-800 border border-gold-500/20 text-slate-300 text-xs flex items-center space-x-1"
            title="Reset sliders"
          >
            <RotateCcw className="w-3.5 h-3.5 text-gold-400" />
            <span>Reset</span>
          </button>
          <button
            onClick={onExportJson}
            className="px-3 py-2 rounded-xl bg-obsidian-900 hover:bg-gold-500/10 border border-gold-500/30 text-gold-300 text-xs flex items-center space-x-1.5"
          >
            <Download className="w-3.5 h-3.5" />
            <span>Export Config JSON</span>
          </button>
          <button
            onClick={onRunBatchSimulation}
            className="gold-button px-4 py-2 rounded-xl text-xs flex items-center space-x-2"
          >
            <Zap className="w-4 h-4" />
            <span>Run Batch Test</span>
          </button>
        </div>
      </div>

      {/* Main Grid: Sliders Controls & Live Calculated Output */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column: Interactive Flexible Sliders */}
        <div className="lg:col-span-7 space-y-5 p-6 rounded-2xl glass-panel border border-gold-500/20">
          <h3 className="text-sm font-bold text-gold-400 uppercase tracking-wider font-mono flex items-center space-x-2">
            <Sliders className="w-4 h-4" />
            <span>Simulation Parameters (Flexible Sliders)</span>
          </h3>

          {/* Slider 1: Concurrent Active Users */}
          <div className="space-y-2 pt-2">
            <div className="flex justify-between items-center text-xs">
              <label className="font-semibold text-slate-200">Concurrent Active Users</label>
              <span className="font-mono text-gold-400 font-bold bg-obsidian-950 px-2.5 py-1 rounded border border-gold-500/20">
                {simParams.concurrentUsers.toLocaleString()} Users
              </span>
            </div>
            <input
              type="range"
              min="1"
              max="1000"
              step="1"
              value={simParams.concurrentUsers}
              onChange={(e) => updateField('concurrentUsers', parseInt(e.target.value))}
              className="gold-slider"
            />
            <div className="flex justify-between text-[10px] text-slate-400 font-mono">
              <span>1 User</span>
              <span>500 Users</span>
              <span>1,000 Users</span>
            </div>
          </div>

          {/* Slider 2: Requests Per Second */}
          <div className="space-y-2">
            <div className="flex justify-between items-center text-xs">
              <label className="font-semibold text-slate-200">Request Rate (Req / Sec)</label>
              <span className="font-mono text-gold-400 font-bold bg-obsidian-950 px-2.5 py-1 rounded border border-gold-500/20">
                {simParams.requestsPerSec} Req/s
              </span>
            </div>
            <input
              type="range"
              min="1"
              max="500"
              step="1"
              value={simParams.requestsPerSec}
              onChange={(e) => updateField('requestsPerSec', parseInt(e.target.value))}
              className="gold-slider"
            />
            <div className="flex justify-between text-[10px] text-slate-400 font-mono">
              <span>1 req/s</span>
              <span>250 req/s</span>
              <span>500 req/s</span>
            </div>
          </div>

          {/* Slider 3: Input Tokens */}
          <div className="space-y-2">
            <div className="flex justify-between items-center text-xs">
              <label className="font-semibold text-slate-200">Input Prompt Tokens per Request</label>
              <span className="font-mono text-gold-400 font-bold bg-obsidian-950 px-2.5 py-1 rounded border border-gold-500/20">
                {simParams.inputTokensPerReq.toLocaleString()} Tokens
              </span>
            </div>
            <input
              type="range"
              min="100"
              max="16000"
              step="100"
              value={simParams.inputTokensPerReq}
              onChange={(e) => updateField('inputTokensPerReq', parseInt(e.target.value))}
              className="gold-slider"
            />
            <div className="flex justify-between text-[10px] text-slate-400 font-mono">
              <span>100 Tokens</span>
              <span>8,000 Tokens</span>
              <span>16,000 Tokens</span>
            </div>
          </div>

          {/* Slider 4: Output Tokens */}
          <div className="space-y-2">
            <div className="flex justify-between items-center text-xs">
              <label className="font-semibold text-slate-200">Completion Output Tokens per Request</label>
              <span className="font-mono text-gold-400 font-bold bg-obsidian-950 px-2.5 py-1 rounded border border-gold-500/20">
                {simParams.outputTokensPerReq.toLocaleString()} Tokens
              </span>
            </div>
            <input
              type="range"
              min="50"
              max="8000"
              step="50"
              value={simParams.outputTokensPerReq}
              onChange={(e) => updateField('outputTokensPerReq', parseInt(e.target.value))}
              className="gold-slider"
            />
            <div className="flex justify-between text-[10px] text-slate-400 font-mono">
              <span>50 Tokens</span>
              <span>4,000 Tokens</span>
              <span>8,000 Tokens</span>
            </div>
          </div>

          {/* Slider 5: Vector Storage GB */}
          <div className="space-y-2">
            <div className="flex justify-between items-center text-xs">
              <label className="font-semibold text-slate-200">Vector Storage Index Size (GB)</label>
              <span className="font-mono text-gold-400 font-bold bg-obsidian-950 px-2.5 py-1 rounded border border-gold-500/20">
                {simParams.vectorStorageGb} GB
              </span>
            </div>
            <input
              type="range"
              min="0.5"
              max="250"
              step="0.5"
              value={simParams.vectorStorageGb}
              onChange={(e) => updateField('vectorStorageGb', parseFloat(e.target.value))}
              className="gold-slider"
            />
            <div className="flex justify-between text-[10px] text-slate-400 font-mono">
              <span>0.5 GB</span>
              <span>125 GB</span>
              <span>250 GB</span>
            </div>
          </div>

          {/* Slider 6: Cache Hit Ratio % */}
          <div className="space-y-2">
            <div className="flex justify-between items-center text-xs">
              <label className="font-semibold text-slate-200">Cache Hit Efficiency Ratio</label>
              <span className="font-mono text-emerald-400 font-bold bg-obsidian-950 px-2.5 py-1 rounded border border-emerald-500/30">
                {simParams.cacheHitRatioPercent}%
              </span>
            </div>
            <input
              type="range"
              min="0"
              max="95"
              step="5"
              value={simParams.cacheHitRatioPercent}
              onChange={(e) => updateField('cacheHitRatioPercent', parseInt(e.target.value))}
              className="gold-slider"
            />
          </div>

          {/* Model Selection */}
          <div className="pt-2">
            <label className="block text-xs font-semibold text-slate-200 mb-2">Simulated Model Architecture</label>
            <div className="grid grid-cols-3 gap-3">
              {(
                [
                  { id: 'fast-lite', name: 'Fast-Lite', desc: '0.8x Cost' },
                  { id: 'pro-reasoning', name: 'Pro-Reasoning', desc: '1.8x Cost' },
                  { id: 'ultra-vision', name: 'Ultra-Vision', desc: '3.5x Cost' },
                ] as const
              ).map((m) => (
                <button
                  key={m.id}
                  onClick={() => updateField('modelTier', m.id)}
                  className={`p-3 rounded-xl border text-left transition-all ${
                    simParams.modelTier === m.id
                      ? 'bg-gold-500/20 border-gold-500 text-gold-300 shadow-gold-sm'
                      : 'bg-obsidian-950 border-slate-800 text-slate-400 hover:border-slate-700'
                  }`}
                >
                  <div className="text-xs font-bold">{m.name}</div>
                  <div className="text-[10px] text-slate-400 font-mono mt-0.5">{m.desc}</div>
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Right Column: Real-Time Calculated Telemetry Cards */}
        <div className="lg:col-span-5 space-y-4">
          <div className="p-6 rounded-2xl glass-panel-gold border border-gold-500/30 space-y-4">
            <h3 className="text-sm font-bold text-slate-100 flex items-center space-x-2">
              <Activity className="w-4 h-4 text-gold-400" />
              <span>Real-Time Output Telemetry</span>
            </h3>

            <div className="grid grid-cols-2 gap-3">
              {/* Stat 1 */}
              <div className="p-3.5 rounded-xl bg-obsidian-950/80 border border-gold-500/20">
                <span className="text-[10px] font-medium text-slate-400 uppercase font-mono">Token Velocity</span>
                <div className="text-lg font-bold text-gold-300 font-mono mt-1">
                  {totalTokensPerSec.toLocaleString()} / sec
                </div>
              </div>

              {/* Stat 2 */}
              <div className="p-3.5 rounded-xl bg-obsidian-950/80 border border-gold-500/20">
                <span className="text-[10px] font-medium text-slate-400 uppercase font-mono">Bandwidth Payload</span>
                <div className="text-lg font-bold text-slate-100 font-mono mt-1">
                  {bandwidthMbps} MB/s
                </div>
              </div>

              {/* Stat 3 */}
              <div className="p-3.5 rounded-xl bg-obsidian-950/80 border border-gold-500/20">
                <span className="text-[10px] font-medium text-slate-400 uppercase font-mono">Est. Daily Spend</span>
                <div className="text-lg font-bold text-gold-400 font-mono mt-1">
                  ${dailyCost.toFixed(2)}
                </div>
              </div>

              {/* Stat 4 */}
              <div className="p-3.5 rounded-xl bg-obsidian-950/80 border border-gold-500/20">
                <span className="text-[10px] font-medium text-slate-400 uppercase font-mono">Est. Monthly Spend</span>
                <div className="text-lg font-bold text-gold-400 font-mono mt-1">
                  ${monthlyCost.toFixed(2)}
                </div>
              </div>
            </div>

            {/* Storage Vector Card */}
            <div className="p-4 rounded-xl bg-obsidian-950/90 border border-gold-500/20 space-y-2">
              <div className="flex items-center justify-between text-xs">
                <span className="text-slate-300 font-semibold flex items-center space-x-1.5">
                  <Database className="w-3.5 h-3.5 text-gold-400" />
                  <span>Vector Index Count</span>
                </span>
                <span className="font-mono text-gold-400 font-bold">
                  {vectorEmbeddingsCount.toLocaleString()} Vectors
                </span>
              </div>
              <p className="text-[11px] text-slate-400 leading-relaxed">
                Allocating {simParams.vectorStorageGb} GB handles approx. {vectorEmbeddingsCount.toLocaleString()} document embeddings at 1536 dimensions.
              </p>
            </div>

            {/* Simulated Load Status Alert */}
            <div
              className={`p-3.5 rounded-xl border text-xs flex items-center space-x-3 ${
                simParams.requestsPerSec > 350
                  ? 'bg-rose-500/10 border-rose-500/30 text-rose-300'
                  : simParams.requestsPerSec > 150
                  ? 'bg-amber-500/10 border-amber-500/30 text-amber-300'
                  : 'bg-emerald-500/10 border-emerald-500/30 text-emerald-300'
              }`}
            >
              <Layers className="w-5 h-5 shrink-0" />
              <div>
                <strong className="block font-bold">
                  {simParams.requestsPerSec > 350
                    ? 'High Load (Rate Throttling Expected)'
                    : simParams.requestsPerSec > 150
                    ? 'Moderate Production Load'
                    : 'Optimal Service Load'}
                </strong>
                <span className="text-[11px] opacity-80">
                  {simParams.requestsPerSec > 350
                    ? 'Auto-scaling policies will queue requests exceeding tier limits.'
                    : 'Metering service operates well within quota guidelines.'}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
