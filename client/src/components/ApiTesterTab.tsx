import React, { useState } from 'react';
import { Terminal, Send, Zap, Clock, Key, CheckCircle, AlertCircle } from 'lucide-react';
import { ApiKeyItem } from '../types';
import { executeAiCompletion } from '../services/api';

interface ApiTesterTabProps {
  apiKeys: ApiKeyItem[];
  onShowToast: (title: string, msg: string, type?: 'success' | 'error' | 'info') => void;
}

export const ApiTesterTab: React.FC<ApiTesterTabProps> = ({ apiKeys, onShowToast }) => {
  const [selectedKey, setSelectedKey] = useState<string>(apiKeys[0]?.key || '');
  const [prompt, setPrompt] = useState('Write a high-performance metering middleware strategy in Go.');
  const [response, setResponse] = useState<string | null>(null);
  const [latencyMs, setLatencyMs] = useState<number | null>(null);
  const [tokenCount, setTokenCount] = useState<number | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const handleSendRequest = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedKey) {
      onShowToast('Missing Key', 'Please select or create an API Key first', 'error');
      return;
    }

    setIsLoading(true);
    setResponse(null);
    try {
      const res = await executeAiCompletion(selectedKey, prompt);
      setResponse(res.answer);
      setLatencyMs(res.latencyMs);
      setTokenCount(res.tokens);
      onShowToast('Request Successful', `Completed in ${res.latencyMs}ms (${res.tokens} tokens)`);
    } catch (err) {
      onShowToast('Request Failed', 'Could not complete request', 'error');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-xl font-bold text-slate-100 flex items-center space-x-2">
          <Terminal className="w-5 h-5 text-gold-400" />
          <span>Live API Request Tester</span>
        </h2>
        <p className="text-xs text-slate-400 mt-1">
          Execute live completion requests against the backend `/v1/chat/completions` endpoint and measure response metrics.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Request Form Panel */}
        <form onSubmit={handleSendRequest} className="lg:col-span-6 space-y-4 p-6 rounded-2xl glass-panel border border-gold-500/20">
          <h3 className="text-sm font-bold text-slate-100 flex items-center space-x-2">
            <Zap className="w-4 h-4 text-gold-400" />
            <span>Request Payload Configuration</span>
          </h3>

          <div>
            <label className="block text-xs font-semibold text-slate-300 mb-1 flex items-center space-x-1.5">
              <Key className="w-3.5 h-3.5 text-gold-400" />
              <span>Select Active API Key</span>
            </label>
            <select
              value={selectedKey}
              onChange={(e) => setSelectedKey(e.target.value)}
              className="w-full px-3.5 py-2.5 rounded-xl bg-obsidian-950 border border-gold-500/20 text-slate-100 font-mono text-xs focus:outline-none focus:border-gold-500"
            >
              {apiKeys.length === 0 ? (
                <option value="">No keys available (Create one in API Keys tab)</option>
              ) : (
                apiKeys.map((k) => (
                  <option key={k.id} value={k.key}>
                    {k.name} ({k.key.substring(0, 10)}...)
                  </option>
                ))
              )}
            </select>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-300 mb-1">Prompt Text</label>
            <textarea
              rows={5}
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              className="w-full p-3.5 rounded-xl bg-obsidian-950 border border-gold-500/20 text-slate-100 font-sans text-xs focus:outline-none focus:border-gold-500"
            />
          </div>

          <div className="pt-2 flex justify-end">
            <button
              type="submit"
              disabled={isLoading || !selectedKey}
              className="gold-button px-5 py-2.5 rounded-xl text-xs flex items-center space-x-2 disabled:opacity-50"
            >
              <Send className="w-4 h-4" />
              <span>{isLoading ? 'Executing Request...' : 'Send Completion Request'}</span>
            </button>
          </div>
        </form>

        {/* Response Panel */}
        <div className="lg:col-span-6 space-y-4 p-6 rounded-2xl glass-panel-gold border border-gold-500/30">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-bold text-slate-100 flex items-center space-x-2">
              <CheckCircle className="w-4 h-4 text-emerald-400" />
              <span>Response Inspector</span>
            </h3>

            {latencyMs !== null && (
              <div className="flex items-center space-x-3 text-xs font-mono">
                <span className="text-emerald-400 flex items-center space-x-1">
                  <Clock className="w-3.5 h-3.5" />
                  <span>{latencyMs}ms</span>
                </span>
                <span className="text-gold-400">{tokenCount} tokens</span>
              </div>
            )}
          </div>

          <div className="min-h-[220px] p-4 rounded-xl bg-obsidian-950 border border-gold-500/20 text-slate-200 font-mono text-xs leading-relaxed overflow-auto">
            {isLoading ? (
              <div className="flex items-center justify-center h-40 text-gold-400 animate-pulse">
                Sending HTTP POST request to Go server...
              </div>
            ) : response ? (
              <div className="whitespace-pre-wrap">{response}</div>
            ) : (
              <div className="text-slate-500 text-center py-12">
                Click "Send Completion Request" to test live API response.
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
