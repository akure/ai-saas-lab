import React, { useState, useEffect } from 'react';
import { Header } from './components/Header';
import { Sidebar, TabId } from './components/Sidebar';
import { OverviewTab } from './components/OverviewTab';
import { SubscriptionTab } from './components/SubscriptionTab';
import { TenantCatalogTab } from './components/tenant-catalog/TenantCatalogTab';
import { ApiKeysTab } from './components/ApiKeysTab';
import { MeteringSimulatorTab } from './components/MeteringSimulatorTab';
import { AnalyticsStorageTab } from './components/AnalyticsStorageTab';
import { JsonStudioTab } from './components/JsonStudioTab';
import { ApiTesterTab } from './components/ApiTesterTab';
import { ApiKeyItem, PlanTier, SimulationParams, ToastMessage } from './types';
import { checkBackendHealth } from './services/api';
import { X, CheckCircle, AlertCircle, Info } from 'lucide-react';

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<TabId>('overview');
  const [selectedPlan, setSelectedPlan] = useState<PlanTier>('pro');
  const [isBackendOnline, setIsBackendOnline] = useState<boolean>(false);
  const [toasts, setToasts] = useState<ToastMessage[]>([]);
  const [currentTenantKey, setCurrentTenantKey] = useState<string>('demo-key-starter');

  // Default API Keys state
  const [apiKeys, setApiKeys] = useState<ApiKeyItem[]>([
    {
      id: 'key_prod_01',
      name: 'Primary Production Key',
      key: 'sk_lab_pro_8f92a1c4e7b3091d',
      plan: 'pro',
      createdAt: new Date(Date.now() - 86400000 * 3).toISOString(),
      status: 'active',
      rateLimitRpm: 2500,
      totalTokensUsed: 142500,
      lastUsedAt: '12 mins ago',
    },
    {
      id: 'key_demo_02',
      name: 'Development Sandbox Key',
      key: 'sk_lab_free_3c7d91e8b2a5',
      plan: 'free',
      createdAt: new Date(Date.now() - 86400000 * 10).toISOString(),
      status: 'active',
      rateLimitRpm: 300,
      totalTokensUsed: 18400,
      lastUsedAt: '2 hours ago',
    },
  ]);

  // Metering Simulator Flexible Sliders Default State
  const [simParams, setSimParams] = useState<SimulationParams>({
    concurrentUsers: 150,
    requestsPerSec: 45,
    inputTokensPerReq: 1200,
    outputTokensPerReq: 450,
    vectorStorageGb: 12.5,
    modelTier: 'pro-reasoning',
    cacheHitRatioPercent: 65,
  });

  // Check Go Backend health on mount
  const verifyHealth = async () => {
    const isOnline = await checkBackendHealth();
    setIsBackendOnline(isOnline);
  };

  useEffect(() => {
    verifyHealth();
    const interval = setInterval(verifyHealth, 15000);
    return () => clearInterval(interval);
  }, []);

  const addToast = (title: string, message: string, type: ToastMessage['type'] = 'success') => {
    const id = `toast_${Date.now()}`;
    setToasts((prev) => [...prev, { id, title, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 4000);
  };

  const removeToast = (id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  };

  // Handlers for API Key manipulation
  const handleAddKey = (newKey: ApiKeyItem) => {
    setApiKeys((prev) => [newKey, ...prev]);
  };

  const handleToggleKeyStatus = (id: string) => {
    setApiKeys((prev) =>
      prev.map((k) => (k.id === id ? { ...k, status: k.status === 'active' ? 'revoked' : 'active' } : k))
    );
    addToast('Status Updated', 'Toggled API Key operational status', 'info');
  };

  const handleDeleteKey = (id: string) => {
    setApiKeys((prev) => prev.filter((k) => k.id !== id));
    addToast('Key Removed', 'API key has been revoked and removed', 'warning');
  };

  // Handler for receiving imported JSON state
  const handleImportJsonState = (importedData: any) => {
    if (importedData.metering_simulation || importedData.simulation_parameters) {
      const simData = importedData.metering_simulation || importedData.simulation_parameters;
      setSimParams((prev) => ({ ...prev, ...simData }));
    }
    if (importedData.api_keys && Array.isArray(importedData.api_keys)) {
      setApiKeys(importedData.api_keys);
    }
  };

  // Quick export action
  const handleQuickExportJson = () => {
    const dataObj = {
      version: '1.0.0-prod',
      exported_at: new Date().toISOString(),
      simulated_plan: selectedPlan,
      api_keys: apiKeys,
      simulation_config: simParams,
    };
    const dataStr = 'data:text/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(dataObj, null, 2));
    const a = document.createElement('a');
    a.setAttribute('href', dataStr);
    a.setAttribute('download', `ai_saas_telemetry_${Date.now()}.json`);
    document.body.appendChild(a);
    a.click();
    a.remove();
    addToast('Export Successful', 'Downloaded client telemetry JSON file');
  };

  return (
    <div className="min-h-screen flex flex-col bg-obsidian-950 text-slate-100">
      {/* Top Header */}
      <Header
        isBackendOnline={isBackendOnline}
        selectedPlan={selectedPlan}
        onPlanChange={setSelectedPlan}
        activeKeysCount={apiKeys.filter((k) => k.status === 'active').length}
        totalTokensUsed={apiKeys.reduce((acc, k) => acc + k.totalTokensUsed, 0)}
        onQuickExportJson={handleQuickExportJson}
        onRefreshHealth={verifyHealth}
      />

      {/* Main Layout Area */}
      <div className="flex-1 flex flex-col md:flex-row">
        {/* Sidebar */}
        <Sidebar activeTab={activeTab} onTabChange={setActiveTab} />

        {/* Tab Content Body */}
        <main className="flex-1 p-6 md:p-8 max-w-7xl mx-auto w-full overflow-y-auto">
          {activeTab === 'overview' && (
            <OverviewTab
              apiKeys={apiKeys}
              simParams={simParams}
              selectedPlan={selectedPlan}
              onNavigate={setActiveTab}
            />
          )}

          {activeTab === 'subscription' && (
            <SubscriptionTab
              currentTenantKey={currentTenantKey}
              onTenantKeyChange={setCurrentTenantKey}
              addToast={addToast}
            />
          )}

          {activeTab === 'tenant-catalog' && (
            <TenantCatalogTab
              apiKeys={apiKeys}
              currentTenantKey={currentTenantKey}
              onSelectTenantKey={setCurrentTenantKey}
              onShowToast={addToast}
            />
          )}

          {activeTab === 'api-keys' && (
            <ApiKeysTab
              apiKeys={apiKeys}
              onAddKey={handleAddKey}
              onToggleStatus={handleToggleKeyStatus}
              onDeleteKey={handleDeleteKey}
              onShowToast={addToast}
            />
          )}

          {activeTab === 'metering' && (
            <MeteringSimulatorTab
              simParams={simParams}
              onSimParamsChange={setSimParams}
              onRunBatchSimulation={() => addToast('Batch Simulation Triggered', 'Processed 500 simulated token metering events!')}
              onExportJson={() => handleQuickExportJson()}
              onResetParams={() => {
                setSimParams({
                  concurrentUsers: 150,
                  requestsPerSec: 45,
                  inputTokensPerReq: 1200,
                  outputTokensPerReq: 450,
                  vectorStorageGb: 12.5,
                  modelTier: 'pro-reasoning',
                  cacheHitRatioPercent: 65,
                });
                addToast('Sliders Reset', 'Reset metering parameters to default baseline');
              }}
            />
          )}

          {activeTab === 'analytics' && <AnalyticsStorageTab simParams={simParams} />}

          {activeTab === 'json-studio' && (
            <JsonStudioTab
              apiKeys={apiKeys}
              simParams={simParams}
              onImportJsonState={handleImportJsonState}
              onShowToast={addToast}
            />
          )}

          {activeTab === 'api-tester' && <ApiTesterTab apiKeys={apiKeys} onShowToast={addToast} />}
        </main>
      </div>

      {/* Floating Toast Notification Engine */}
      <div className="fixed bottom-5 right-5 z-50 space-y-2 max-w-sm w-full pointer-events-none">
        {toasts.map((t) => (
          <div
            key={t.id}
            className={`pointer-events-auto p-4 rounded-xl shadow-2xl backdrop-blur-md border flex items-start justify-between space-x-3 transition-all duration-300 ${
              t.type === 'error'
                ? 'bg-rose-950/90 border-rose-500/40 text-rose-200'
                : t.type === 'warning'
                ? 'bg-amber-950/90 border-amber-500/40 text-amber-200'
                : t.type === 'info'
                ? 'bg-blue-950/90 border-blue-500/40 text-blue-200'
                : 'bg-obsidian-900/95 border-gold-500/50 text-gold-300 shadow-gold-sm'
            }`}
          >
            <div className="flex items-start space-x-2.5">
              {t.type === 'error' ? (
                <AlertCircle className="w-5 h-5 text-rose-400 shrink-0 mt-0.5" />
              ) : (
                <CheckCircle className="w-5 h-5 text-gold-400 shrink-0 mt-0.5" />
              )}
              <div>
                <h4 className="text-xs font-bold">{t.title}</h4>
                <p className="text-[11px] opacity-90 mt-0.5 leading-snug">{t.message}</p>
              </div>
            </div>
            <button onClick={() => removeToast(t.id)} className="text-slate-400 hover:text-slate-200">
              <X className="w-4 h-4" />
            </button>
          </div>
        ))}
      </div>
    </div>
  );
};
