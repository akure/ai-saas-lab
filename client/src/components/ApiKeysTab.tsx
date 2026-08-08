import React, { useState } from 'react';
import { Key, Plus, Copy, Check, Eye, EyeOff, Trash2, ShieldCheck, Terminal, AlertCircle } from 'lucide-react';
import { ApiKeyItem, PlanTier } from '../types';
import { createBackendApiKey } from '../services/api';

interface ApiKeysTabProps {
  apiKeys: ApiKeyItem[];
  onAddKey: (key: ApiKeyItem) => void;
  onToggleStatus: (id: string) => void;
  onDeleteKey: (id: string) => void;
  onShowToast: (title: string, msg: string, type?: 'success' | 'error' | 'info') => void;
}

export const ApiKeysTab: React.FC<ApiKeysTabProps> = ({
  apiKeys,
  onAddKey,
  onToggleStatus,
  onDeleteKey,
  onShowToast,
}) => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [keyName, setKeyName] = useState('');
  const [planTier, setPlanTier] = useState<PlanTier>('pro');
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [revealedKeys, setRevealedKeys] = useState<Record<string, boolean>>({});
  const [selectedKeyForSnippet, setSelectedKeyForSnippet] = useState<ApiKeyItem | null>(null);
  const [snippetTab, setSnippetTab] = useState<'curl' | 'powershell'>('curl');
  const [isCreating, setIsCreating] = useState(false);

  const handleCreateKey = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsCreating(true);
    try {
      const newKey = await createBackendApiKey(planTier, keyName || `${planTier.toUpperCase()} API Key`);
      onAddKey(newKey);
      setKeyName('');
      setIsModalOpen(false);
      onShowToast('API Key Created', `Issued new ${planTier.toUpperCase()} API Key: ${newKey.key.substring(0, 10)}...`);
    } catch (err) {
      onShowToast('Key Creation Error', 'Could not issue key', 'error');
    } finally {
      setIsCreating(false);
    }
  };

  const handleCopy = (keyText: string, id: string) => {
    navigator.clipboard.writeText(keyText);
    setCopiedId(id);
    onShowToast('Copied!', 'API key copied to clipboard');
    setTimeout(() => setCopiedId(null), 2000);
  };

  const toggleReveal = (id: string) => {
    setRevealedKeys((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  const activeSnippetKey = selectedKeyForSnippet || apiKeys[0];

  const getCurlSnippet = (key: string) => `curl -N -X POST http://localhost:8080/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -d '{"api_key":"${key}","prompt":"hello from client dashboard"}'`;

  const getPowerShellSnippet = (key: string) => `Invoke-RestMethod -Uri "http://localhost:8080/v1/chat/completions" \`
  -Method Post \`
  -ContentType "application/json" \`
  -Body '{"api_key":"${key}","prompt":"hello from client dashboard"}'`;

  return (
    <div className="space-y-6">
      {/* Header & Create Key Action */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-xl font-bold text-slate-100 flex items-center space-x-2">
            <Key className="w-5 h-5 text-gold-400" />
            <span>API Keys & Credentials</span>
          </h2>
          <p className="text-xs text-slate-400 mt-1">
            Manage production secret keys for authenticating with the AI SaaS Lab completion server.
          </p>
        </div>
        <button
          onClick={() => setIsModalOpen(true)}
          className="gold-button flex items-center space-x-2 px-4 py-2 rounded-xl text-xs"
        >
          <Plus className="w-4 h-4" />
          <span>Create Secret Key</span>
        </button>
      </div>

      {/* Keys Table / List */}
      <div className="rounded-2xl glass-panel border border-gold-500/20 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-obsidian-900/90 text-gold-400 border-b border-gold-500/20 uppercase tracking-wider font-mono">
              <tr>
                <th className="px-5 py-3.5">Name</th>
                <th className="px-5 py-3.5">Secret Key</th>
                <th className="px-5 py-3.5">Plan</th>
                <th className="px-5 py-3.5">Rate Limit</th>
                <th className="px-5 py-3.5">Status</th>
                <th className="px-5 py-3.5 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gold-500/10 text-slate-300">
              {apiKeys.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-5 py-8 text-center text-slate-400">
                    No API keys created yet. Click "Create Secret Key" above to generate your first key.
                  </td>
                </tr>
              ) : (
                apiKeys.map((item) => {
                  const isRevealed = !!revealedKeys[item.id];
                  const displayKey = isRevealed
                    ? item.key
                    : `${item.key.substring(0, 7)}${'•'.repeat(24)}${item.key.slice(-4)}`;

                  return (
                    <tr
                      key={item.id}
                      className={`hover:bg-obsidian-800/40 transition-colors ${
                        selectedKeyForSnippet?.id === item.id ? 'bg-gold-500/5' : ''
                      }`}
                    >
                      <td className="px-5 py-4 font-semibold text-slate-100">
                        {item.name}
                        <div className="text-[10px] font-normal text-slate-400">{new Date(item.createdAt).toLocaleDateString()}</div>
                      </td>
                      <td className="px-5 py-4 font-mono">
                        <div className="flex items-center space-x-2 bg-obsidian-950 px-3 py-1.5 rounded-lg border border-gold-500/20 w-fit">
                          <span className="text-gold-300">{displayKey}</span>
                          <button
                            onClick={() => toggleReveal(item.id)}
                            className="text-slate-400 hover:text-gold-400 transition-colors"
                            title={isRevealed ? 'Mask key' : 'Reveal key'}
                          >
                            {isRevealed ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                          </button>
                          <button
                            onClick={() => handleCopy(item.key, item.id)}
                            className="text-slate-400 hover:text-gold-400 transition-colors"
                            title="Copy key"
                          >
                            {copiedId === item.id ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                          </button>
                        </div>
                      </td>
                      <td className="px-5 py-4 capitalize">
                        <span
                          className={`px-2.5 py-1 rounded-md text-[10px] font-semibold ${
                            item.plan === 'enterprise'
                              ? 'bg-purple-500/10 text-purple-300 border border-purple-500/30'
                              : item.plan === 'pro'
                              ? 'bg-gold-500/10 text-gold-300 border border-gold-500/30'
                              : 'bg-slate-800 text-slate-400 border border-slate-700'
                          }`}
                        >
                          {item.plan}
                        </span>
                      </td>
                      <td className="px-5 py-4 font-mono text-slate-300">
                        {item.rateLimitRpm.toLocaleString()} RPM
                      </td>
                      <td className="px-5 py-4">
                        <button
                          onClick={() => onToggleStatus(item.id)}
                          className={`px-2.5 py-1 rounded-full text-[10px] font-semibold flex items-center space-x-1 transition-all ${
                            item.status === 'active'
                              ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'
                              : 'bg-rose-500/10 text-rose-400 border border-rose-500/30'
                          }`}
                        >
                          <span className={`w-1.5 h-1.5 rounded-full ${item.status === 'active' ? 'bg-emerald-400' : 'bg-rose-400'}`} />
                          <span className="capitalize">{item.status}</span>
                        </button>
                      </td>
                      <td className="px-5 py-4 text-right space-x-2">
                        <button
                          onClick={() => setSelectedKeyForSnippet(item)}
                          className="px-2 py-1 rounded bg-obsidian-800 hover:bg-gold-500/20 text-slate-300 hover:text-gold-300 border border-slate-700 text-[11px]"
                        >
                          Snippets
                        </button>
                        <button
                          onClick={() => onDeleteKey(item.id)}
                          className="p-1 rounded bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 border border-rose-500/30"
                          title="Delete Key"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Code Snippet Generator Box */}
      {activeSnippetKey && (
        <div className="p-6 rounded-2xl glass-panel border border-gold-500/20 space-y-3">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div className="flex items-center space-x-2">
              <Terminal className="w-4 h-4 text-gold-400" />
              <h3 className="text-sm font-bold text-slate-100">
                Integration Snippets for Key: <span className="text-gold-400 font-mono">{activeSnippetKey.name}</span>
              </h3>
            </div>

            <div className="flex items-center space-x-2 bg-obsidian-950 p-1 rounded-lg border border-gold-500/20 text-xs">
              <button
                onClick={() => setSnippetTab('curl')}
                className={`px-3 py-1 rounded-md font-medium transition-all ${
                  snippetTab === 'curl' ? 'bg-gold-500/20 text-gold-300 border border-gold-500/40' : 'text-slate-400'
                }`}
              >
                cURL (Bash)
              </button>
              <button
                onClick={() => setSnippetTab('powershell')}
                className={`px-3 py-1 rounded-md font-medium transition-all ${
                  snippetTab === 'powershell' ? 'bg-gold-500/20 text-gold-300 border border-gold-500/40' : 'text-slate-400'
                }`}
              >
                PowerShell
              </button>
            </div>
          </div>

          <div className="relative">
            <pre className="p-4 rounded-xl bg-obsidian-950 border border-gold-500/20 text-slate-200 font-mono text-xs overflow-x-auto">
              {snippetTab === 'curl' ? getCurlSnippet(activeSnippetKey.key) : getPowerShellSnippet(activeSnippetKey.key)}
            </pre>
            <button
              onClick={() =>
                handleCopy(
                  snippetTab === 'curl'
                    ? getCurlSnippet(activeSnippetKey.key)
                    : getPowerShellSnippet(activeSnippetKey.key),
                  'snippet'
                )
              }
              className="absolute top-3 right-3 p-2 rounded-lg bg-obsidian-800 hover:bg-gold-500/20 text-slate-300 hover:text-gold-300 border border-slate-700"
              title="Copy snippet"
            >
              <Copy className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>
      )}

      {/* Modal: Create Key */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-obsidian-950/80 backdrop-blur-sm">
          <div className="w-full max-w-md p-6 rounded-2xl glass-panel-gold border border-gold-500/40 shadow-2xl space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-base font-bold text-slate-100 flex items-center space-x-2">
                <ShieldCheck className="w-5 h-5 text-gold-400" />
                <span>Create Production API Key</span>
              </h3>
              <button
                onClick={() => setIsModalOpen(false)}
                className="text-slate-400 hover:text-slate-200 text-sm font-bold"
              >
                ✕
              </button>
            </div>

            <form onSubmit={handleCreateKey} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-300 mb-1">Key Label / Name</label>
                <input
                  type="text"
                  placeholder="e.g. Production Billing Microservice"
                  value={keyName}
                  onChange={(e) => setKeyName(e.target.value)}
                  className="w-full px-3.5 py-2.5 rounded-xl bg-obsidian-950 border border-gold-500/20 text-slate-100 text-xs focus:outline-none focus:border-gold-500"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-300 mb-1">Plan Tier</label>
                <select
                  value={planTier}
                  onChange={(e) => setPlanTier(e.target.value as PlanTier)}
                  className="w-full px-3.5 py-2.5 rounded-xl bg-obsidian-950 border border-gold-500/20 text-slate-100 text-xs focus:outline-none focus:border-gold-500"
                >
                  <option value="free">Free Tier (300 RPM limit)</option>
                  <option value="pro">Pro Tier (2,500 RPM limit)</option>
                  <option value="enterprise">Enterprise Tier (10,000 RPM limit)</option>
                </select>
              </div>

              <div className="pt-2 flex justify-end space-x-3">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 rounded-xl text-xs bg-obsidian-900 text-slate-300 hover:bg-obsidian-800"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isCreating}
                  className="gold-button px-4 py-2 rounded-xl text-xs"
                >
                  {isCreating ? 'Issuing Key...' : 'Generate Secret Key'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
