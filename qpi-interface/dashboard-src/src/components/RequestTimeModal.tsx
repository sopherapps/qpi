import React, { useState } from "react";
import { X } from "lucide-react";

interface RequestTimeModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (seconds: number, reason: string) => Promise<void>;
}

export const RequestTimeModal: React.FC<RequestTimeModalProps> = ({
  isOpen,
  onClose,
  onSubmit,
}) => {
  const [seconds, setSeconds] = useState(100);
  const [reason, setReason] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await onSubmit(seconds, reason);
      onClose();
      setSeconds(100);
      setReason("");
      alert("Time request submitted successfully!");
    } catch (err: any) {
      setError(err?.message || "Failed to submit request.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
      <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-lg shadow-2xl p-6 space-y-4">
        <div className="flex justify-between items-center border-b border-zinc-800 pb-3">
          <h3 className="text-lg font-semibold font-geist text-white">Request QPU Time</h3>
          <button
            onClick={onClose}
            className="text-zinc-400 hover:text-white focus:outline-none"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4 text-sm">
          <div>
            <label className="block text-xs font-medium text-zinc-400 mb-1.5 uppercase tracking-wider">
              Requested Seconds
            </label>
            <input
              type="number"
              required
              value={seconds}
              onChange={(e) => setSeconds(parseInt(e.target.value) || 1)}
              className="w-full bg-zinc-950 border border-zinc-800 rounded px-3 py-2 text-white focus:outline-none focus:border-zinc-500 transition-colors font-mono"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-zinc-400 mb-1.5 uppercase tracking-wider">
              Reason / Project Name
            </label>
            <textarea
              required
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              className="w-full bg-zinc-950 border border-zinc-800 rounded px-3 py-2 text-white focus:outline-none focus:border-zinc-500 transition-colors h-20"
              placeholder="Running VQE experiments for chemistry simulations..."
            />
          </div>

          {error && (
            <div className="text-xs text-error font-medium bg-error/10 border border-error/20 p-2.5 rounded">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-white text-zinc-950 font-semibold py-2.5 rounded hover:opacity-90 transition-opacity disabled:opacity-50"
          >
            {loading ? "Submitting..." : "Submit Time Request"}
          </button>
        </form>
      </div>
    </div>
  );
};
