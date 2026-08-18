import React, { useState } from 'react';
import { X, Layers, AlertCircle, CheckCircle2, Sparkles } from 'lucide-react';
import { TenantServiceDescriptor, CatalogApiError } from '../../types/catalog';

interface ServiceModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (svc: TenantServiceDescriptor) => Promise<void>;
}

const SERVICE_ID_REGEX = /^[a-z0-9][a-z0-9_-]{0,62}[a-z0-9]$|^[a-z0-9]$/;

export const ServiceModal: React.FC<ServiceModalProps> = ({ isOpen, onClose, onSubmit }) => {
  const [serviceId, setServiceId] = useState('');
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [apiError, setApiError] = useState<CatalogApiError | null>(null);

  if (!isOpen) return null;

  // Real-time client validation
  const isIdValid = serviceId.length > 0 && SERVICE_ID_REGEX.test(serviceId);
  const isNameValid = name.trim().length > 0 && name.trim().length <= 128;

  const handleServiceIdChange = (val: string) => {
    // Auto convert uppercase to lowercase and replace spaces with hyphens
    const sanitized = val.toLowerCase().replace(/\s+/g, '-');
    setServiceId(sanitized);
    if (apiError) setApiError(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isIdValid || !isNameValid) return;

    setIsSubmitting(true);
    setApiError(null);

    try {
      await onSubmit({
        service_id: serviceId.trim(),
        name: name.trim(),
        description: description.trim() || undefined,
      });
      // Reset form and close
      setServiceId('');
      setName('');
      setDescription('');
      onClose();
    } catch (err: any) {
      setApiError(err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Preset suggestions for rapid onboarding
  const applyPreset = (presetId: string, presetName: string, presetDesc: string) => {
    setServiceId(presetId);
    setName(presetName);
    setDescription(presetDesc);
    setApiError(null);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-obsidian-950/80 backdrop-blur-md animate-fadeIn">
      <div className="bg-obsidian-900 border border-gold-500/30 rounded-2xl max-w-lg w-full p-6 shadow-2xl shadow-black/80 relative">
        {/* Modal Header */}
        <div className="flex items-center justify-between pb-4 border-b border-gold-500/20">
          <div className="flex items-center space-x-3">
            <div className="p-2.5 rounded-xl bg-gold-500/10 text-gold-400 border border-gold-500/30">
              <Layers className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-base font-bold text-slate-100">Register New Service</h3>
              <p className="text-xs text-slate-400">Define a service module for metering & telemetry</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-200 transition-colors p-1"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Quick Presets */}
        <div className="mt-4 p-3 rounded-xl bg-obsidian-800/80 border border-gold-500/15">
          <div className="flex items-center space-x-1.5 text-xs text-gold-400 font-semibold mb-2">
            <Sparkles className="w-3.5 h-3.5" />
            <span>Quick Start Templates</span>
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() =>
                applyPreset(
                  'ai-voice-agent',
                  'AI Real-Time Voice Agent',
                  'Ultra-low latency streaming voice inference & TTS conversion'
                )
              }
              className="text-[11px] px-2.5 py-1 rounded-md bg-obsidian-900 text-slate-300 hover:text-gold-300 hover:border-gold-500/40 border border-slate-700 transition-all font-mono"
            >
              + ai-voice-agent
            </button>
            <button
              type="button"
              onClick={() =>
                applyPreset(
                  'ocr-vision-pipeline',
                  'OCR Document Vision Pipeline',
                  'Multimodal document page analysis & token extraction engine'
                )
              }
              className="text-[11px] px-2.5 py-1 rounded-md bg-obsidian-900 text-slate-300 hover:text-gold-300 hover:border-gold-500/40 border border-slate-700 transition-all font-mono"
            >
              + ocr-vision-pipeline
            </button>
            <button
              type="button"
              onClick={() =>
                applyPreset(
                  'custom-api-gateway',
                  'Tenant API Gateway',
                  'Unified request proxy with rate limits and quota enforcement'
                )
              }
              className="text-[11px] px-2.5 py-1 rounded-md bg-obsidian-900 text-slate-300 hover:text-gold-300 hover:border-gold-500/40 border border-slate-700 transition-all font-mono"
            >
              + custom-api-gateway
            </button>
          </div>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          {/* Service ID Field */}
          <div>
            <div className="flex items-center justify-between">
              <label className="text-xs font-semibold text-slate-300">
                Service ID <span className="text-rose-400">*</span>
              </label>
              <span className="text-[10px] text-slate-400 font-mono">
                {serviceId ? `${serviceId.length}/64` : 'Lowercase, alphanumeric, hyphens'}
              </span>
            </div>
            <div className="relative mt-1.5">
              <input
                type="text"
                value={serviceId}
                onChange={(e) => handleServiceIdChange(e.target.value)}
                placeholder="e.g. ai-completion, vector-search"
                className={`w-full px-3.5 py-2.5 bg-obsidian-950 border rounded-xl text-xs font-mono text-slate-100 placeholder-slate-600 focus:outline-none transition-all ${
                  serviceId.length > 0
                    ? isIdValid
                      ? 'border-emerald-500/60 focus:border-emerald-400'
                      : 'border-rose-500/60 focus:border-rose-400'
                    : 'border-gold-500/20 focus:border-gold-500/60'
                }`}
              />
              {serviceId.length > 0 && (
                <div className="absolute right-3 top-1/2 -translate-y-1/2">
                  {isIdValid ? (
                    <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                  ) : (
                    <AlertCircle className="w-4 h-4 text-rose-400" />
                  )}
                </div>
              )}
            </div>
            {serviceId.length > 0 && !isIdValid && (
              <p className="text-[11px] text-rose-400 mt-1">
                Must be 1-64 lowercase alphanumeric chars with hyphens/underscores. Cannot start/end with a hyphen.
              </p>
            )}
          </div>

          {/* Service Name Field */}
          <div>
            <label className="text-xs font-semibold text-slate-300">
              Display Name <span className="text-rose-400">*</span>
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. AI Completion Engine"
              className="mt-1.5 w-full px-3.5 py-2.5 bg-obsidian-950 border border-gold-500/20 rounded-xl text-xs text-slate-100 placeholder-slate-600 focus:outline-none focus:border-gold-500/60 transition-all font-sans"
            />
          </div>

          {/* Description Field */}
          <div>
            <label className="text-xs font-semibold text-slate-300">Description (Optional)</label>
            <textarea
              rows={2}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Brief summary of what this service meters or stores..."
              className="mt-1.5 w-full px-3.5 py-2 bg-obsidian-950 border border-gold-500/20 rounded-xl text-xs text-slate-100 placeholder-slate-600 focus:outline-none focus:border-gold-500/60 transition-all resize-none"
            />
          </div>

          {/* Error Banner */}
          {apiError && (
            <div className="p-3 rounded-xl bg-rose-950/60 border border-rose-500/50 flex items-start space-x-2.5 text-rose-200">
              <AlertCircle className="w-4 h-4 text-rose-400 shrink-0 mt-0.5" />
              <div className="text-xs">
                <strong className="font-semibold block">
                  {apiError.code === 'CONFLICT'
                    ? '409 Conflict: Service Already Exists'
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
              disabled={!isIdValid || !isNameValid || isSubmitting}
              className="gold-button px-5 py-2 rounded-xl text-xs font-semibold disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-1.5"
            >
              {isSubmitting ? (
                <span>Registering...</span>
              ) : (
                <>
                  <Layers className="w-3.5 h-3.5" />
                  <span>Register Service</span>
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
