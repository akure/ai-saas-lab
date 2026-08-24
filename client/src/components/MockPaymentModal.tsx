import React, { useState } from 'react';
import {
  CreditCard,
  Building2,
  Wallet,
  Coins,
  ShieldAlert,
  CheckCircle2,
  XCircle,
  Loader2,
  X,
  Lock,
  Sparkles,
} from 'lucide-react';
import {
  CheckoutSession,
  PaymentMethod,
  processPayment,
  Transaction,
} from '../services/paymentApi';

interface MockPaymentModalProps {
  session: CheckoutSession | null;
  onClose: () => void;
  onSuccess: (tx: Transaction) => void;
}

export const MockPaymentModal: React.FC<MockPaymentModalProps> = ({
  session,
  onClose,
  onSuccess,
}) => {
  const [method, setMethod] = useState<PaymentMethod>('card');
  const [simulateMode, setSimulateMode] = useState<'succeed' | 'decline' | 'challenge'>('succeed');
  const [accountHolder, setAccountHolder] = useState('Alex Taylor');
  const [cardNumber, setCardNumber] = useState('4242 •••• •••• 4242');
  const [challengeCode, setChallengeCode] = useState('');
  const [isChallenged, setIsChallenged] = useState(false);
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  if (!session) return null;

  const handlePay = async () => {
    setLoading(true);
    setErrorMsg(null);
    try {
      const tx = await processPayment(session.id, {
        payment_method: method,
        simulate_mode: simulateMode,
        account_holder: accountHolder,
        token_ref: cardNumber,
        challenge_code: isChallenged ? challengeCode : undefined,
      });

      setLoading(false);
      onSuccess(tx);
    } catch (err: any) {
      setLoading(false);
      if (err?.isChallenge) {
        setIsChallenged(true);
        setErrorMsg('Multi-Factor Payment Verification Required. Enter Test Code: 123456');
      } else {
        setErrorMsg(err.message || 'Payment simulation failed');
      }
    }
  };

  const formattedAmount = (session.amount_cents / 100).toLocaleString('en-US', {
    style: 'currency',
    currency: session.currency || 'USD',
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in duration-200">
      <div className="relative w-full max-w-lg overflow-hidden rounded-2xl border border-slate-700 bg-slate-900 text-slate-100 shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-800 bg-slate-950/60 px-6 py-4">
          <div className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-500/20 text-indigo-400">
              <Lock className="h-4 w-4" />
            </div>
            <div>
              <h3 className="font-semibold text-white">Sandbox Payment Gateway</h3>
              <p className="text-xs text-slate-400">Vendor-Neutral Mock Checkout Rails</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-1 text-slate-400 hover:bg-slate-800 hover:text-white transition"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Order Summary */}
        <div className="bg-slate-950/30 px-6 py-4 border-b border-slate-800 flex items-center justify-between">
          <div>
            <span className="text-xs font-mono uppercase tracking-wider text-indigo-400">
              Plan Order
            </span>
            <div className="text-lg font-bold text-white capitalize">{session.plan_id} Plan</div>
            <div className="text-xs text-slate-400 capitalize">{session.billing_cycle} billing</div>
          </div>
          <div className="text-right">
            <div className="text-2xl font-black text-emerald-400">{formattedAmount}</div>
            <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-400">
              <Sparkles className="h-3 w-3" /> Test Sandbox
            </span>
          </div>
        </div>

        {/* Form Body */}
        <div className="p-6 space-y-5">
          {/* Method Tabs */}
          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase mb-2">
              Select Payment Method
            </label>
            <div className="grid grid-cols-4 gap-2">
              <button
                type="button"
                onClick={() => setMethod('card')}
                className={`flex flex-col items-center justify-center p-2.5 rounded-xl border text-xs font-medium transition ${
                  method === 'card'
                    ? 'border-indigo-500 bg-indigo-500/10 text-indigo-300'
                    : 'border-slate-800 bg-slate-950/40 text-slate-400 hover:border-slate-700'
                }`}
              >
                <CreditCard className="h-5 w-5 mb-1 text-indigo-400" />
                Card
              </button>
              <button
                type="button"
                onClick={() => setMethod('bank_transfer')}
                className={`flex flex-col items-center justify-center p-2.5 rounded-xl border text-xs font-medium transition ${
                  method === 'bank_transfer'
                    ? 'border-indigo-500 bg-indigo-500/10 text-indigo-300'
                    : 'border-slate-800 bg-slate-950/40 text-slate-400 hover:border-slate-700'
                }`}
              >
                <Building2 className="h-5 w-5 mb-1 text-emerald-400" />
                Direct Pay
              </button>
              <button
                type="button"
                onClick={() => setMethod('digital_wallet')}
                className={`flex flex-col items-center justify-center p-2.5 rounded-xl border text-xs font-medium transition ${
                  method === 'digital_wallet'
                    ? 'border-indigo-500 bg-indigo-500/10 text-indigo-300'
                    : 'border-slate-800 bg-slate-950/40 text-slate-400 hover:border-slate-700'
                }`}
              >
                <Wallet className="h-5 w-5 mb-1 text-amber-400" />
                Wallet
              </button>
              <button
                type="button"
                onClick={() => setMethod('crypto_token')}
                className={`flex flex-col items-center justify-center p-2.5 rounded-xl border text-xs font-medium transition ${
                  method === 'crypto_token'
                    ? 'border-indigo-500 bg-indigo-500/10 text-indigo-300'
                    : 'border-slate-800 bg-slate-950/40 text-slate-400 hover:border-slate-700'
                }`}
              >
                <Coins className="h-5 w-5 mb-1 text-cyan-400" />
                Crypto
              </button>
            </div>
          </div>

          {/* Instrument Input Mock */}
          <div className="space-y-3">
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">Account Holder</label>
              <input
                type="text"
                value={accountHolder}
                onChange={(e) => setAccountHolder(e.target.value)}
                className="w-full rounded-xl border border-slate-800 bg-slate-950 px-3.5 py-2 text-sm text-slate-200 focus:border-indigo-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">
                {method === 'card'
                  ? 'Card Number'
                  : method === 'bank_transfer'
                  ? 'Account Reference'
                  : method === 'digital_wallet'
                  ? 'Wallet Identifier'
                  : 'Token Address / Chain'}
              </label>
              <input
                type="text"
                value={cardNumber}
                onChange={(e) => setCardNumber(e.target.value)}
                className="w-full rounded-xl border border-slate-800 bg-slate-950 px-3.5 py-2 text-sm font-mono text-slate-200 focus:border-indigo-500 focus:outline-none"
              />
            </div>
          </div>

          {/* Verification Challenge Mode Input */}
          {isChallenged && (
            <div className="p-4 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-300 space-y-2">
              <div className="flex items-center gap-2 font-medium text-xs">
                <ShieldAlert className="h-4 w-4" /> Multi-Factor Verification Challenge
              </div>
              <p className="text-xs text-amber-200/80">
                Enter test passcode <code className="font-bold text-amber-100">123456</code> to complete challenge.
              </p>
              <input
                type="text"
                placeholder="Enter 123456"
                value={challengeCode}
                onChange={(e) => setChallengeCode(e.target.value)}
                className="w-full rounded-lg border border-amber-500/40 bg-slate-950 px-3 py-1.5 text-sm font-mono text-white focus:outline-none"
              />
            </div>
          )}

          {/* Developer Controls Simulator */}
          <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-3 space-y-1.5">
            <span className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
              Developer Simulation Mode
            </span>
            <div className="grid grid-cols-3 gap-1.5">
              <button
                type="button"
                onClick={() => setSimulateMode('succeed')}
                className={`px-2 py-1 rounded-lg text-xs font-medium border transition ${
                  simulateMode === 'succeed'
                    ? 'border-emerald-500 bg-emerald-500/10 text-emerald-300'
                    : 'border-slate-800 text-slate-400 hover:text-slate-200'
                }`}
              >
                Force Pass
              </button>
              <button
                type="button"
                onClick={() => setSimulateMode('decline')}
                className={`px-2 py-1 rounded-lg text-xs font-medium border transition ${
                  simulateMode === 'decline'
                    ? 'border-rose-500 bg-rose-500/10 text-rose-300'
                    : 'border-slate-800 text-slate-400 hover:text-slate-200'
                }`}
              >
                Force Decline
              </button>
              <button
                type="button"
                onClick={() => setSimulateMode('challenge')}
                className={`px-2 py-1 rounded-lg text-xs font-medium border transition ${
                  simulateMode === 'challenge'
                    ? 'border-amber-500 bg-amber-500/10 text-amber-300'
                    : 'border-slate-800 text-slate-400 hover:text-slate-200'
                }`}
              >
                Verification
              </button>
            </div>
          </div>

          {/* Error Message */}
          {errorMsg && (
            <div className="flex items-center gap-2 rounded-xl border border-rose-500/30 bg-rose-500/10 p-3 text-xs text-rose-300">
              <XCircle className="h-4 w-4 shrink-0" />
              <span>{errorMsg}</span>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 border-t border-slate-800 bg-slate-950/60 px-6 py-4">
          <button
            type="button"
            onClick={onClose}
            className="rounded-xl border border-slate-800 bg-slate-900 px-4 py-2 text-xs font-semibold text-slate-300 hover:bg-slate-800 transition"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={handlePay}
            disabled={loading}
            className="flex items-center gap-2 rounded-xl bg-indigo-600 px-5 py-2 text-xs font-semibold text-white shadow-lg shadow-indigo-600/30 hover:bg-indigo-500 disabled:opacity-50 transition"
          >
            {loading ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" /> Processing...
              </>
            ) : (
              <>
                <CheckCircle2 className="h-4 w-4" /> Complete {formattedAmount} Payment
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};
