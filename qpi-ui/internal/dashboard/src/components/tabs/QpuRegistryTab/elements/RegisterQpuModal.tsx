import { useState } from "react";
import { X } from "lucide-react";

interface Props {
  onClose: () => void;
  onRegister: (name: string, token: string, executor: string) => Promise<void>;
}

export function RegisterQpuModal({ onClose, onRegister }: Props) {
  const [regName, setRegName] = useState("");
  const [regToken, setRegToken] = useState("");
  const [regExecutor, setRegExecutor] = useState("mock");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await onRegister(regName, regToken, regExecutor);
      onClose();
      setRegName("");
      setRegToken("");
      setRegExecutor("mock");
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Registration failed. Check inputs.";
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
      <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-lg shadow-2xl p-6 space-y-4">
        <div className="flex justify-between items-center border-b border-zinc-800 pb-3">
          <h3 className="text-lg font-semibold font-geist text-white">Register QPU</h3>
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
              QPU ID/Name
            </label>
            <input
              type="text"
              required
              value={regName}
              onChange={(e) => setRegName(e.target.value)}
              className="w-full bg-zinc-950 border border-zinc-800 rounded px-3 py-2 text-white focus:outline-none focus:border-zinc-500 transition-colors"
              placeholder="rigetti-aspen-9"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-zinc-400 mb-1.5 uppercase tracking-wider">
              Registration Token
            </label>
            <input
              type="text"
              required
              value={regToken}
              onChange={(e) => setRegToken(e.target.value)}
              className="w-full bg-zinc-950 border border-zinc-800 rounded px-3 py-2 text-white font-mono focus:outline-none focus:border-zinc-500 transition-colors"
              placeholder="Enter QPU secret token"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-zinc-400 mb-1.5 uppercase tracking-wider">
              Executor Type
            </label>
            <select
              value={regExecutor}
              onChange={(e) => setRegExecutor(e.target.value)}
              className="w-full bg-zinc-950 border border-zinc-800 text-white rounded px-3 py-2 focus:outline-none focus:border-zinc-500 transition-colors"
            >
              <option value="mock">mock (Local Simulator)</option>
              <option value="qiskit_aer">qiskit_aer (Aer Simulator)</option>
              <option value="quantify">quantify (Quantify Driver)</option>
            </select>
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
