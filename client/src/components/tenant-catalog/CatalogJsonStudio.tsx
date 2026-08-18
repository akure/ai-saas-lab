import React, { useState } from 'react';
import {
  FileJson,
  Download,
  Upload,
  Copy,
  Check,
  Code2,
  Terminal,
  AlertCircle,
  CheckCircle2,
  Calendar,
  Sparkles,
} from 'lucide-react';
import {
  CatalogOverview,
  CatalogDateFilterState,
  BatchImportResult,
} from '../../types/catalog';

interface CatalogJsonStudioProps {
  overview: CatalogOverview;
  dateFilter: CatalogDateFilterState;
  onImportJson: (jsonPayload: any) => Promise<BatchImportResult>;
  onShowToast: (title: string, message: string, type?: 'success' | 'error' | 'info' | 'warning') => void;
}

export const CatalogJsonStudio: React.FC<CatalogJsonStudioProps> = ({
  overview,
  dateFilter,
  onImportJson,
  onShowToast,
}) => {
  const [importJsonText, setImportJsonText] = useState('');
  const [isImporting, setIsImporting] = useState(false);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [activeSnippetTab, setActiveSnippetTab] = useState<'json' | 'ts' | 'curl'>('json');

  // Filter items by date if range is selected
  const filterByDate = <T extends { created_at?: string }>(items: T[]): T[] => {
    if (dateFilter.rangeType === 'all') return items;

    const now = Date.now();
    return items.filter((item) => {
      if (!item.created_at) return true;
      const itemTime = new Date(item.created_at).getTime();

      if (dateFilter.rangeType === 'today') {
        return now - itemTime <= 86400000;
      }
      if (dateFilter.rangeType === '7d') {
        return now - itemTime <= 86400000 * 7;
      }
      if (dateFilter.rangeType === '30d') {
        return now - itemTime <= 86400000 * 30;
      }
      if (dateFilter.rangeType === 'custom' && dateFilter.startDate && dateFilter.endDate) {
        const start = new Date(dateFilter.startDate).getTime();
        const end = new Date(dateFilter.endDate).getTime() + 86400000;
        return itemTime >= start && itemTime <= end;
      }
      return true;
    });
  };

  const filteredOverview: CatalogOverview = {
    tenant_key: overview.tenant_key,
    services: filterByDate(overview.services),
    metrics: filterByDate(overview.metrics),
    plans: filterByDate(overview.plans),
  };

  const jsonString = JSON.stringify(
    {
      maas_catalog_version: '1.0.0',
      exported_at: new Date().toISOString(),
      date_filter_applied: dateFilter.rangeType,
      ...filteredOverview,
    },
    null,
    2
  );

  // Generate TypeScript Definitions for tenant SDK
  const generateTypescriptSnippet = () => {
    return `// ==========================================
// MAAS Platform Tenant Catalog SDK Definition
// Generated for Tenant: ${overview.tenant_key}
// ==========================================

export interface MaasServices {
${overview.services
  .map(
    (s) => `  /** ${s.description || s.name} */
  "${s.service_id}": {
    name: "${s.name}";
    metrics: {
${overview.metrics
  .filter((m) => m.service_id === s.service_id)
  .map((m) => `      "${m.metric_id}": { unit: "${m.unit}"; name: "${m.name}" };`)
  .join('\n')}
    };
  };`
  )
  .join('\n')}
}

export interface MaasPlans {
${overview.plans
  .map(
    (p) => `  "${p.plan_id}": {
    serviceId: "${p.service_id}";
    name: "${p.name}";
    version: ${p.version};
    rates: ${JSON.stringify(p.rates)};
  };`
  )
  .join('\n')}
}
`;
  };

  // Generate cURL Commands
  const generateCurlSnippet = () => {
    const sampleService = overview.services[0] || {
      service_id: 'ai-completion',
      name: 'AI Completion Engine',
      description: 'LLM Text Generation',
    };
    const sampleMetric = overview.metrics[0] || {
      metric_id: 'tokens',
      service_id: sampleService.service_id,
      name: 'Tokens',
      unit: 'tokens',
    };
    const samplePlan = overview.plans[0] || {
      plan_id: 'starter_plan',
      service_id: sampleService.service_id,
      name: 'Starter Tier',
      rates: { tokens: 0.0015 },
      included_quotas: { tokens: 10000 },
    };

    return `# 1. Register Service via REST API
curl -X POST http://localhost:8080/v1/tenant/catalog/services \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: ${overview.tenant_key}" \\
  -d '${JSON.stringify(sampleService)}'

# 2. Register Billable Metric
curl -X POST http://localhost:8080/v1/tenant/catalog/metrics \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: ${overview.tenant_key}" \\
  -d '${JSON.stringify(sampleMetric)}'

# 3. Register Application Plan
curl -X POST http://localhost:8080/v1/tenant/catalog/plans \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: ${overview.tenant_key}" \\
  -d '${JSON.stringify(samplePlan)}'

# 4. Fetch Full Catalog Overview
curl -X GET http://localhost:8080/v1/tenant/catalog/overview \\
  -H "X-API-Key: ${overview.tenant_key}"
`;
  };

  const handleCopy = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    onShowToast('Copied to Clipboard', `Copied ${key} to clipboard`);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const handleDownloadJson = () => {
    const blob = new Blob([jsonString], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `maas_catalog_${overview.tenant_key}_${Date.now()}.json`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
    onShowToast('Catalog Downloaded', 'Exported tenant catalog JSON schema');
  };

  const handleImportSubmit = async () => {
    if (!importJsonText.trim()) return;

    try {
      const parsed = JSON.parse(importJsonText);
      setIsImporting(true);
      const result = await onImportJson(parsed);

      if (result.errors.length === 0) {
        onShowToast(
          'Import Successful',
          `Added ${result.servicesAdded} services, ${result.metricsAdded} metrics, and ${result.plansAdded} plans`
        );
        setImportJsonText('');
      } else {
        onShowToast(
          'Import Finished with Warnings',
          `Added ${result.servicesAdded} services, ${result.metricsAdded} metrics, ${result.plansAdded} plans. ${result.errors.length} failed.`,
          'warning'
        );
      }
    } catch (e: any) {
      onShowToast('Invalid JSON Syntax', e.message || 'Please verify JSON syntax', 'error');
    } finally {
      setIsImporting(false);
    }
  };

  const handleLoadSampleJson = () => {
    const sample = {
      services: [
        {
          service_id: 'document-ocr',
          name: 'Document OCR & Parsing Engine',
          description: 'Extracts tabular & structured key-values from PDF documents',
        },
      ],
      metrics: [
        {
          metric_id: 'pages_processed',
          service_id: 'document-ocr',
          name: 'Pages Processed',
          unit: 'pages',
          description: 'Document sheets converted & parsed',
        },
        {
          metric_id: 'table_extractions',
          service_id: 'document-ocr',
          name: 'Table Extractions',
          unit: 'requests',
          description: 'Complex financial table parsed',
        },
      ],
      plans: [
        {
          plan_id: 'ocr_starter',
          service_id: 'document-ocr',
          name: 'OCR Starter Pack',
          rates: { pages_processed: 0.05, table_extractions: 0.1 },
          included_quotas: { pages_processed: 100 },
          version: 1,
          active: true,
        },
      ],
    };
    setImportJsonText(JSON.stringify(sample, null, 2));
  };

  return (
    <div className="space-y-6">
      {/* Studio Header Bar */}
      <div className="p-5 rounded-2xl glass-panel flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="flex items-center space-x-3.5">
          <div className="p-3 rounded-xl bg-gold-500/10 text-gold-400 border border-gold-500/30">
            <FileJson className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-base font-bold text-slate-100">Tenant Catalog JSON Studio</h3>
            <p className="text-xs text-slate-400">
              Download schemas, import batch catalogs, and generate SDK snippets for apps
            </p>
          </div>
        </div>

        <div className="flex items-center space-x-2">
          <button
            onClick={handleDownloadJson}
            className="gold-button px-4 py-2 rounded-xl text-xs font-semibold flex items-center space-x-1.5 shadow-gold-sm"
          >
            <Download className="w-3.5 h-3.5" />
            <span>Download Catalog JSON</span>
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Side: Code Inspector (7 Cols) */}
        <div className="lg:col-span-7 space-y-3">
          {/* Tab Switcher for Code Views */}
          <div className="flex items-center justify-between bg-obsidian-900/90 p-1.5 rounded-xl border border-gold-500/20">
            <div className="flex items-center space-x-1">
              <button
                onClick={() => setActiveSnippetTab('json')}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center space-x-1.5 transition-all ${
                  activeSnippetTab === 'json'
                    ? 'bg-gold-500/20 text-gold-300 border border-gold-500/40 shadow-gold-sm'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                <FileJson className="w-3.5 h-3.5" />
                <span>Catalog JSON</span>
              </button>

              <button
                onClick={() => setActiveSnippetTab('ts')}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center space-x-1.5 transition-all ${
                  activeSnippetTab === 'ts'
                    ? 'bg-gold-500/20 text-gold-300 border border-gold-500/40 shadow-gold-sm'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                <Code2 className="w-3.5 h-3.5" />
                <span>TypeScript SDK</span>
              </button>

              <button
                onClick={() => setActiveSnippetTab('curl')}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center space-x-1.5 transition-all ${
                  activeSnippetTab === 'curl'
                    ? 'bg-gold-500/20 text-gold-300 border border-gold-500/40 shadow-gold-sm'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                <Terminal className="w-3.5 h-3.5" />
                <span>cURL API</span>
              </button>
            </div>

            <button
              onClick={() => {
                const text =
                  activeSnippetTab === 'json'
                    ? jsonString
                    : activeSnippetTab === 'ts'
                    ? generateTypescriptSnippet()
                    : generateCurlSnippet();
                handleCopy(text, activeSnippetTab);
              }}
              className="px-2.5 py-1 rounded-lg bg-obsidian-800 hover:bg-obsidian-700 text-slate-300 border border-slate-700 text-xs flex items-center space-x-1 transition-all"
            >
              {copiedKey === activeSnippetTab ? (
                <Check className="w-3.5 h-3.5 text-emerald-400" />
              ) : (
                <Copy className="w-3.5 h-3.5" />
              )}
              <span>Copy</span>
            </button>
          </div>

          {/* Code Viewer Panel */}
          <div className="relative rounded-xl border border-gold-500/20 bg-obsidian-950 p-4 font-mono text-xs text-slate-300 max-h-[500px] overflow-y-auto shadow-inner">
            <pre className="whitespace-pre">
              {activeSnippetTab === 'json' && jsonString}
              {activeSnippetTab === 'ts' && generateTypescriptSnippet()}
              {activeSnippetTab === 'curl' && generateCurlSnippet()}
            </pre>
          </div>
        </div>

        {/* Right Side: Batch JSON Importer (5 Cols) */}
        <div className="lg:col-span-5 space-y-4">
          <div className="p-5 rounded-2xl glass-panel space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <Upload className="w-4 h-4 text-gold-400" />
                <h4 className="text-xs font-bold uppercase tracking-wider text-slate-200">
                  Batch Catalog Importer
                </h4>
              </div>
              <button
                onClick={handleLoadSampleJson}
                className="text-[11px] text-gold-400 hover:text-gold-300 font-semibold flex items-center space-x-1"
              >
                <Sparkles className="w-3 h-3" />
                <span>Load Sample</span>
              </button>
            </div>

            <p className="text-xs text-slate-400 leading-relaxed">
              Paste JSON containing <code>services</code>, <code>metrics</code>, and <code>plans</code> arrays to bulk-register your architecture into MAAS in one go.
            </p>

            <textarea
              rows={12}
              value={importJsonText}
              onChange={(e) => setImportJsonText(e.target.value)}
              placeholder={`{\n  "services": [...],\n  "metrics": [...],\n  "plans": [...]\n}`}
              className="w-full p-3.5 bg-obsidian-950 border border-gold-500/20 rounded-xl text-xs font-mono text-slate-200 placeholder-slate-600 focus:outline-none focus:border-gold-500/60 resize-none font-mono"
            />

            <button
              onClick={handleImportSubmit}
              disabled={!importJsonText.trim() || isImporting}
              className="gold-button w-full py-2.5 rounded-xl text-xs font-semibold disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center space-x-2"
            >
              <Upload className="w-4 h-4" />
              <span>{isImporting ? 'Importing Schema...' : 'Run Batch Import'}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
