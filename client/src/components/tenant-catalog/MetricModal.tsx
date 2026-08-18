import React, { useState, useEffect } from 'react';
import { X, Gauge, AlertCircle, CheckCircle2, Sparkles } from 'lucide-react';
import {
  TenantMetricDescriptor,
  TenantServiceDescriptor,
  CatalogApiError,
} from '../../types/catalog';

interface MetricModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (metric: TenantMetricDescriptor) => Promise<void>;
  services: TenantServiceDescriptor[];
  preselectedServiceId?: string;
}

const METRIC_ID_REGEX = /^[a-z0-9][a-z0-9_-]{0,62}[a-z0-9]$|^[a-z0-9]$/;

const UNIT_PRESETS = [
  { value: 'tokens', label: 'tokens (LLM Tokens)' },
  { value: 'requests', label: 'requests (API Invocations)' },
  { value: 'seconds', label: 'seconds (Execution Duration)' },
  { value: 'bytes', label: 'bytes (Storage Footprint)' },
  { value: 'pages', label: 'pages (Document Sheets)' },
  { value: 'custom', label: 'custom (Enter custom unit)' },
];

export const MetricModal: React.FC<MetricModalProps> = ({
  isOpen,
  onClose,
  onSubmit,
  services,
  preselectedServiceId,
}) => {
  const [metricId, setMetricId] = useState('');
  const [serviceId, setServiceId] = useState('');
  const [name, setName] = useState('');
  const [unit, setUnit] = useState('tokens');
  const [customUnit, setCustomUnit] = useState('');
  const [description, setDescription] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [apiError, setApiError] = useState<CatalogApiError | null>(null);

  useEffect(() => {
    if (preselectedServiceId) {
      setServiceId(preselectedServiceId);
    } else if (services.length > 0 && !serviceId) {
      setServiceId(services[0].service_id);
    }
  }, [preselectedServiceId, services, isOpen]);

  if (!isOpen) return null;

  const isIdValid = metricId.length > 0 && METRIC_ID_REGEX.test(metricId);
  const effectiveUnit = unit === 'custom' ? customUnit.trim() : unit;
  const isUnitValid = effectiveUnit.length > 0;
  const isServiceValid = serviceId.trim().length > 0;

  const handleMetricIdChange = (val: string) => {
    const sanitized = val.toLowerCase().replace(/\s+/g, '_');
    setMetricId(sanitized);
    if (apiError) setApiError(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isIdValid || !isServiceValid || !isUnitValid) return;

    setIsSubmitting(true);
    setApiError(null);

    try {
      await onSubmit({
        metric_id: metricId.trim(),
        service_id: serviceId.trim(),
        name: name.trim() || metricId.trim(),
        unit: effectiveUnit,
        description: description.trim() || undefined,
      });
      // Reset form and close
      setMetricId('');
      setName('');
      setDescription('');
      setUnit('tokens');
      setCustomUnit('');
      onClose();
    } catch (err: any) {
      setApiError(err);
    } finally {
      setIsSubmitting(false);
    }
  };

  const applyPreset = (mId: string, mName: string, mUnit: string, mDesc: string) => {
    setMetricId(mId);
    setName(mName);
    setUnit(mUnit);
    setDescription(mDesc);
    setApiError(null);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-obsidian-950/80 backdrop-blur-md animate-fadeIn">
      <div className="bg-obsidian-900 border border-gold-500/30 rounded-2xl max-w-lg w-full p-6 shadow-2xl shadow-black/80 relative">
        {/* Modal Header */}
        <div className="flex items-center justify-between pb-4 border-b border-gold-500/20">
          <div className="flex items-center space-x-3">
            <div className="p-2.5 rounded-xl bg-gold-500/10 text-gold-400 border border-gold-500/30">
              <Gauge className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-base font-bold text-slate-100">Register Billable Metric</h3>
              <p className="text-xs text-slate-400">Attach a measurable resource dimension to a service</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-200 transition-colors p-1"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Quick Metric Presets */}
        <div className="mt-4 p-3 rounded-xl bg-obsidian-800/80 border border-gold-500/15">
          <div className="flex items-center space-x-1.5 text-xs text-gold-400 font-semibold mb-2">
            <Sparkles className="w-3.5 h-3.5" />
            <span>Standard Metering Templates</span>
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() =>
                applyPreset(
                  'prompt_tokens',
                  'Input Prompt Tokens',
                  'tokens',
                  'Incoming token count metered before model generation'
                )
              }
              className="text-[11px] px-2.5 py-1 rounded-md bg-obsidian-900 text-slate-300 hover:text-gold-300 hover:border-gold-500/40 border border-slate-700 transition-all font-mono"
            >
              + prompt_tokens
            </button>
            <button
              type="button"
              onClick={() =>
                applyPreset(
                  'completion_tokens',
                  'Output Completion Tokens',
                  'tokens',
                  'Generated token count produced by AI model'
                )
              }
              className="text-[11px] px-2.5 py-1 rounded-md bg-obsidian-900 text-slate-300 hover:text-gold-300 hover:border-gold-500/40 border border-slate-700 transition-all font-mono"
            >
              + completion_tokens
            </button>
            <button
              type="button"
              onClick={() =>
                applyPreset(
                  'search_queries',
                  'Semantic Vector Queries',
                  'requests',
                  'Top-K vector similarity index searches'
                )
              }
              className="text-[11px] px-2.5 py-1 rounded-md bg-obsidian-900 text-slate-300 hover:text-gold-300 hover:border-gold-500/40 border border-slate-700 transition-all font-mono"
            >
              + search_queries
            </button>
          </div>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          {/* Target Service Selection */}
          <div>
            <label className="text-xs font-semibold text-slate-300">
              Attached Service <span className="text-rose-400">*</span>
            </label>
            {services.length === 0 ? (
              <div className="mt-1.5 p-3 rounded-xl bg-amber-950/40 border border-amber-500/30 text-amber-300 text-xs">
                No services registered yet. Please register a service first!
              </div>
            ) : (
              <select
                value={serviceId}
                onChange={(e) => setServiceId(e.target.value)}
                className="mt-1.5 w-full px-3.5 py-2.5 bg-obsidian-950 border border-gold-500/20 rounded-xl text-xs text-slate-100 focus:outline-none focus:border-gold-500/60 font-mono"
              >
                {services.map((s) => (
                  <option key={s.service_id} value={s.service_id} className="bg-obsidian-900">
                    {s.name} ({s.service_id})
                  </option>
                ))}
              </select>
            )}
          </div>

          {/* Metric ID */}
          <div>
            <div className="flex items-center justify-between">
              <label className="text-xs font-semibold text-slate-300">
                Metric ID <span className="text-rose-400">*</span>
              </label>
              <span className="text-[10px] text-slate-400 font-mono">
                {metricId ? `${metricId.length}/64` : 'e.g. tokens, audio_seconds'}
              </span>
            </div>
            <div className="relative mt-1.5">
              <input
                type="text"
                value={metricId}
                onChange={(e) => handleMetricIdChange(e.target.value)}
                placeholder="e.g. prompt_tokens, api_invocations"
                className={`w-full px-3.5 py-2.5 bg-obsidian-950 border rounded-xl text-xs font-mono text-slate-100 placeholder-slate-600 focus:outline-none transition-all ${
                  metricId.length > 0
                    ? isIdValid
                      ? 'border-emerald-500/60 focus:border-emerald-400'
                      : 'border-rose-500/60 focus:border-rose-400'
                    : 'border-gold-500/20 focus:border-gold-500/60'
                }`}
              />
              {metricId.length > 0 && (
                <div className="absolute right-3 top-1/2 -translate-y-1/2">
                  {isIdValid ? (
                    <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                  ) : (
                    <AlertCircle className="w-4 h-4 text-rose-400" />
                  )}
                </div>
              )}
            </div>
          </div>

          {/* Unit Selector */}
          <div>
            <label className="text-xs font-semibold text-slate-300">
              Unit of Measurement <span className="text-rose-400">*</span>
            </label>
            <select
              value={unit}
              onChange={(e) => setUnit(e.target.value)}
              className="mt-1.5 w-full px-3.5 py-2.5 bg-obsidian-950 border border-gold-500/20 rounded-xl text-xs text-slate-100 focus:outline-none focus:border-gold-500/60"
            >
              {UNIT_PRESETS.map((u) => (
                <option key={u.value} value={u.value} className="bg-obsidian-900">
                  {u.label}
                </option>
              ))}
            </select>
            {unit === 'custom' && (
              <input
                type="text"
                value={customUnit}
                onChange={(e) => setCustomUnit(e.target.value)}
                placeholder="Enter custom unit name (e.g. gigabytes, queries)"
                className="mt-2 w-full px-3.5 py-2 bg-obsidian-950 border border-gold-500/20 rounded-xl text-xs text-slate-100 placeholder-slate-600 focus:outline-none focus:border-gold-500/60"
              />
            )}
          </div>

          {/* Display Name */}
          <div>
            <label className="text-xs font-semibold text-slate-300">Display Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Input Tokens Processed (defaults to metric ID)"
              className="mt-1.5 w-full px-3.5 py-2.5 bg-obsidian-950 border border-gold-500/20 rounded-xl text-xs text-slate-100 placeholder-slate-600 focus:outline-none focus:border-gold-500/60"
            />
          </div>

          {/* Error Banner */}
          {apiError && (
            <div className="p-3 rounded-xl bg-rose-950/60 border border-rose-500/50 flex items-start space-x-2.5 text-rose-200">
              <AlertCircle className="w-4 h-4 text-rose-400 shrink-0 mt-0.5" />
              <div className="text-xs">
                <strong className="font-semibold block">
                  {apiError.code === 'CONFLICT'
                    ? '409 Conflict: Metric Already Exists'
                    : apiError.code === 'VALIDATION'
                    ? '422 Validation Error'
                    : 'Registration Failed'}
                </strong>
                <p className="mt-0.5 opacity-90">{apiError.message}</p>
              </div>
            </div>
          )}

          {/* Submit & Cancel Buttons */}
          <div className="flex items-center justify-end space-x-3 pt-3 border-t border-gold-500/20">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-xl text-xs font-medium text-slate-400 hover:text-slate-200 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!isIdValid || !isServiceValid || !isUnitValid || isSubmitting}
              className="gold-button px-5 py-2 rounded-xl text-xs font-semibold disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-1.5"
            >
              {isSubmitting ? (
                <span>Registering...</span>
              ) : (
                <>
                  <Gauge className="w-3.5 h-3.5" />
                  <span>Register Metric</span>
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
