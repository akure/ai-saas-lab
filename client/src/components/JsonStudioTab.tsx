import React, { useState } from 'react';
import { FileJson, Download, Upload, Copy, Check, Search, Eye, Filter } from 'lucide-react';
import { ApiKeyItem, SimulationParams } from '../types';

interface JsonStudioTabProps {
  apiKeys: ApiKeyItem[];
  simParams: SimulationParams;
  onImportJsonState: (importedData: any) => void;
  onShowToast: (title: string, msg: string, type?: 'success' | 'error' | 'info') => void;
}

export const JsonStudioTab: React.FC<JsonStudioTabProps> = ({
  apiKeys,
  simParams,
  onImportJsonState,
  onShowToast,
}) => {
  const [pasteInput, setPasteInput] = useState('');
  const [activeJsonView, setActiveJsonView] = useState<'telemetry' | 'keys' | 'simulation' | 'custom'>('telemetry');
  const [customParsedJson, setCustomParsedJson] = useState<any>(null);
  const [isCopied, setIsCopied] = useState(false);
  const [searchFilter, setSearchFilter] = useState('');

  // Built-in JSON generators
  const fullTelemetryJson = {
    dashboard_version: '1.0.0-prod',
    timestamp: new Date().toISOString(),
    tenant: {
      plan: 'pro',
      status: 'active',
    },
    metering_simulation: simParams,
    api_keys: apiKeys,
    calculated_metrics: {
      tokens_per_sec: simParams.requestsPerSec * (simParams.inputTokensPerReq + simParams.outputTokensPerReq),
      bandwidth_mbps: ((simParams.requestsPerSec * (simParams.inputTokensPerReq + simParams.outputTokensPerReq) * 4) / 1048576).toFixed(2),
      estimated_monthly_spend_usd: (simParams.requestsPerSec * 86400 * 30 * (simParams.inputTokensPerReq + simParams.outputTokensPerReq) * 0.0000015).toFixed(2),
      vector_storage_embeddings: Math.round(simParams.vectorStorageGb * 250000),
    },
  };

  const getTargetJson = () => {
    if (activeJsonView === 'custom' && customParsedJson) {
      return customParsedJson;
    }
    if (activeJsonView === 'keys') {
      return { api_keys: apiKeys };
    }
    if (activeJsonView === 'simulation') {
      return { simulation_parameters: simParams };
    }
    return fullTelemetryJson;
  };

  const currentJsonString = JSON.stringify(getTargetJson(), null, 2);

  const handleCopy = () => {
    navigator.clipboard.writeText(currentJsonString);
    setIsCopied(true);
    onShowToast('Copied to Clipboard', 'JSON payload copied successfully');
    setTimeout(() => setIsCopied(null as any), 2000);
  };

  const handleDownload = (filename: string, obj: any) => {
    const dataStr = 'data:text/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(obj, null, 2));
    const downloadAnchor = document.createElement('a');
    downloadAnchor.setAttribute('href', dataStr);
    downloadAnchor.setAttribute('download', filename);
    document.body.appendChild(downloadAnchor);
    downloadAnchor.click();
    downloadAnchor.remove();
    onShowToast('Downloaded', `Saved ${filename} to your local files`);
  };

  const handleReceiveJson = () => {
    if (!pasteInput.trim()) return;
    try {
      const parsed = JSON.parse(pasteInput);
      setCustomParsedJson(parsed);
      setActiveJsonView('custom');
      onImportJsonState(parsed);
      onShowToast('JSON Received & Rendered', 'Successfully parsed and applied JSON payload state!');
      setPasteInput('');
    } catch (e) {
      onShowToast('Invalid JSON', 'Please check your JSON format syntax', 'error');
    }
  };

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (event) => {
      try {
        const parsed = JSON.parse(event.target?.result as string);
        setCustomParsedJson(parsed);
        setActiveJsonView('custom');
        onImportJsonState(parsed);
        onShowToast('JSON File Uploaded', `Loaded data from ${file.name}`);
      } catch (err) {
        onShowToast('File Parse Error', 'File does not contain valid JSON', 'error');
      }
    };
    reader.readAsText(file);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-xl font-bold text-slate-100 flex items-center space-x-2">
            <FileJson className="w-5 h-5 text-gold-400" />
            <span>JSON Data Hub (Receive, Render & Download)</span>
          </h2>
          <p className="text-xs text-slate-400 mt-1">
            Import, view, parse, and export telemetry and metering stats as structured JSON.
          </p>
        </div>

        {/* Export Action Buttons */}
        <div className="flex items-center space-x-2">
          <button
            onClick={() => handleDownload('usage_and_metering_stats.json', fullTelemetryJson)}
            className="gold-button flex items-center space-x-1.5 px-3 py-2 rounded-xl text-xs"
          >
            <Download className="w-4 h-4" />
            <span>Download All JSON</span>
          </button>
        </div>
      </div>

      {/* Grid: Receive JSON Panel & Rendered Inspector */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column: Receive / Import JSON Input */}
        <div className="lg:col-span-5 space-y-4 p-6 rounded-2xl glass-panel border border-gold-500/20">
          <h3 className="text-sm font-bold text-slate-100 flex items-center space-x-2">
            <Upload className="w-4 h-4 text-gold-400" />
            <span>Receive / Upload JSON Payload</span>
          </h3>
          <p className="text-xs text-slate-400">
            Paste raw JSON from external metering APIs or upload a `.json` telemetry file.
          </p>

          <textarea
            rows={8}
            placeholder='Paste JSON here e.g. { "concurrentUsers": 250, "requestsPerSec": 45 }'
            value={pasteInput}
            onChange={(e) => setPasteInput(e.target.value)}
            className="w-full p-3 rounded-xl bg-obsidian-950 border border-gold-500/20 text-slate-200 font-mono text-xs focus:outline-none focus:border-gold-500"
          />

          <div className="flex items-center justify-between gap-3">
            <label className="cursor-pointer px-3.5 py-2 rounded-xl bg-obsidian-900 hover:bg-obsidian-800 border border-gold-500/30 text-gold-300 text-xs font-semibold flex items-center space-x-2">
              <Upload className="w-3.5 h-3.5" />
              <span>Choose .json File</span>
              <input type="file" accept=".json" onChange={handleFileUpload} className="hidden" />
            </label>

            <button
              onClick={handleReceiveJson}
              disabled={!pasteInput.trim()}
              className="gold-button px-4 py-2 rounded-xl text-xs flex items-center space-x-1.5 disabled:opacity-50"
            >
              <Check className="w-4 h-4" />
              <span>Process & Render</span>
            </button>
          </div>

          <hr className="border-gold-500/10 my-4" />

          {/* Quick Preset Download Buttons */}
          <div className="space-y-2">
            <label className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider font-mono">
              Download Specific Datasets:
            </label>
            <div className="grid grid-cols-2 gap-2">
              <button
                onClick={() => handleDownload('api_keys_export.json', { api_keys: apiKeys })}
                className="p-2.5 rounded-xl bg-obsidian-950 hover:bg-gold-500/10 border border-gold-500/20 text-slate-200 text-xs flex items-center justify-between"
              >
                <span>API Keys List</span>
                <Download className="w-3.5 h-3.5 text-gold-400" />
              </button>
              <button
                onClick={() => handleDownload('simulation_config.json', { simulation_config: simParams })}
                className="p-2.5 rounded-xl bg-obsidian-950 hover:bg-gold-500/10 border border-gold-500/20 text-slate-200 text-xs flex items-center justify-between"
              >
                <span>Simulator Config</span>
                <Download className="w-3.5 h-3.5 text-gold-400" />
              </button>
            </div>
          </div>
        </div>

        {/* Right Column: Dynamic JSON Renderer / Display Box */}
        <div className="lg:col-span-7 space-y-4 p-6 rounded-2xl glass-panel-gold border border-gold-500/30">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            {/* View Selector Tabs */}
            <div className="flex items-center space-x-1.5 bg-obsidian-950 p-1 rounded-xl border border-gold-500/20">
              <button
                onClick={() => setActiveJsonView('telemetry')}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
                  activeJsonView === 'telemetry' ? 'bg-gold-500 text-obsidian-950 shadow' : 'text-slate-400'
                }`}
              >
                Full Telemetry
              </button>
              <button
                onClick={() => setActiveJsonView('keys')}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
                  activeJsonView === 'keys' ? 'bg-gold-500 text-obsidian-950 shadow' : 'text-slate-400'
                }`}
              >
                API Keys
              </button>
              <button
                onClick={() => setActiveJsonView('simulation')}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
                  activeJsonView === 'simulation' ? 'bg-gold-500 text-obsidian-950 shadow' : 'text-slate-400'
                }`}
              >
                Simulator
              </button>
              {customParsedJson && (
                <button
                  onClick={() => setActiveJsonView('custom')}
                  className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
                    activeJsonView === 'custom' ? 'bg-gold-500 text-obsidian-950 shadow' : 'text-slate-400'
                  }`}
                >
                  Imported Payload
                </button>
              )}
            </div>

            {/* Copy Button */}
            <button
              onClick={handleCopy}
              className="p-2 rounded-xl bg-obsidian-950 hover:bg-gold-500/20 border border-gold-500/20 text-gold-300 text-xs flex items-center space-x-1.5 self-end"
            >
              {isCopied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{isCopied ? 'Copied' : 'Copy JSON'}</span>
            </button>
          </div>

          {/* Formatted JSON Display Container */}
          <div className="relative">
            <pre className="p-4 rounded-xl bg-obsidian-950 border border-gold-500/30 text-gold-200 font-mono text-xs max-h-[480px] overflow-auto leading-relaxed selection:bg-gold-500 selection:text-obsidian-950">
              {currentJsonString}
            </pre>
          </div>
        </div>
      </div>
    </div>
  );
};
