import { useState } from "react";
import { X } from "lucide-react";
import type { CreateQpuResponse } from "@/types";

interface Props {
  onClose: () => void;
  onRegister: (name: string) => Promise<CreateQpuResponse>;
}

export function RegisterQpuModal({ onClose, onRegister }: Props) {
  const [regName, setRegName] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await onRegister(regName);
      setRegName("");
      onClose();
    } catch (err: unknown) {
      const message =
        err instanceof Error
          ? err.message
          : "Registration failed. Check inputs.";
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-50 dark:bg-zinc-950/80 backdrop-blur-sm">
      <div className="w-full max-w-md bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 rounded-lg shadow-2xl p-6 space-y-4 max-h-[90vh] overflow-y-auto">
        <div className="flex justify-between items-center border-b border-gray-200 dark:border-zinc-800 pb-3">
          <h3 className="text-lg font-semibold font-geist text-gray-900 dark:text-white">
            Register QPU
          </h3>
          <button
            onClick={onClose}
            className="text-gray-500 dark:text-zinc-400 hover:text-gray-900 dark:text-white focus:outline-none"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4 text-sm">
          <div>
            <label className="block text-xs font-medium text-gray-500 dark:text-zinc-400 mb-1.5 uppercase tracking-wider">
              QPU ID/Name
            </label>
            <input
              type="text"
              required
              value={regName}
              onChange={(e) => setRegName(e.target.value)}
              className="w-full bg-gray-50 dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded px-3 py-2 text-gray-900 dark:text-white focus:outline-none focus:border-zinc-500 transition-colors"
              placeholder="rigetti-aspen-9"
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
            {loading ? "Registering..." : "Register Unit"}
          </button>
        </form>
      </div>
    </div>
  );
}
