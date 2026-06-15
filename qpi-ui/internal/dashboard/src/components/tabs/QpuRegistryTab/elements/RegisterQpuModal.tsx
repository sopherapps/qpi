import { useState } from "react";
import { X, Copy, Check } from "lucide-react";
import type { CreateQpuResponse } from "@/types";

interface Props {
  onClose: () => void;
  onRegister: (name: string, executor: string) => Promise<CreateQpuResponse>;
}

export function RegisterQpuModal({ onClose, onRegister }: Props) {
  const [regName, setRegName] = useState("");
  const [regExecutor, setRegExecutor] = useState("mock");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const [createdQpu, setCreatedQpu] = useState<CreateQpuResponse | null>(null);
  const [copiedToken, setCopiedToken] = useState(false);
  const [copiedCommand, setCopiedCommand] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await onRegister(regName, regExecutor);
      if (res && res.access_token) {
        setCreatedQpu(res);
      } else {
        onClose();
      }
      setRegName("");
      setRegExecutor("mock");
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

  const handleCopyToken = () => {
    if (createdQpu) {
      navigator.clipboard.writeText(createdQpu.access_token);
      setCopiedToken(true);
      setTimeout(() => setCopiedToken(false), 2000);
    }
  };

  const handleCopyCommand = () => {
    if (!createdQpu) return;
    const qpiAddr = createdQpu.qpi_addr || window.location.origin;
    const cmd = `QPI_ACCESS_TOKEN=${createdQpu.access_token} qpi-driver start --qpi-addr ${qpiAddr} --name "${createdQpu.name}" --executor "${createdQpu.executor_type}"`;
    navigator.clipboard.writeText(cmd);
    setCopiedCommand(true);
    setTimeout(() => setCopiedCommand(false), 2000);
  };

  if (createdQpu) {
    const qpiAddr = createdQpu.qpi_addr || window.location.origin;
    const commandText = `QPI_ACCESS_TOKEN=${createdQpu.access_token} qpi-driver start --qpi-addr ${qpiAddr} --name "${createdQpu.name}" --executor "${createdQpu.executor_type}"`;

    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm animate-fade-in">
        <div className="w-full max-w-lg bg-zinc-900 border border-zinc-800 rounded-lg shadow-2xl p-6 space-y-5">
          <div className="flex justify-between items-center border-b border-zinc-800 pb-3">
            <h3 className="text-lg font-semibold font-geist text-emerald-400">
              QPU Registered Successfully!
            </h3>
            <button
              onClick={onClose}
              className="text-zinc-400 hover:text-white focus:outline-none"
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          <div className="space-y-4 text-sm text-zinc-300">
            <p>
              Your QPU <strong>{createdQpu.name}</strong> has been registered.
              Copy the credentials below to configure your hardware driver.
            </p>

            <div className="bg-zinc-950 border border-zinc-800 rounded p-4 space-y-4">
              <div>
                <label className="block text-xs font-semibold text-zinc-400 uppercase tracking-wider mb-1">
                  Access Token
                </label>
                <div className="flex items-center gap-2">
                  <span className="flex-1 font-mono bg-zinc-900 border border-zinc-800 px-3 py-1.5 rounded text-white overflow-x-auto whitespace-nowrap">
                    {createdQpu.access_token}
                  </span>
                  <button
                    type="button"
                    onClick={handleCopyToken}
                    className="p-2 bg-zinc-800 hover:bg-zinc-700 text-white rounded transition-colors focus:outline-none flex items-center justify-center min-w-[36px]"
                    title="Copy Token"
                  >
                    {copiedToken ? (
                      <Check className="w-4 h-4 text-emerald-400" />
                    ) : (
                      <Copy className="w-4 h-4" />
                    )}
                  </button>
                </div>
                <span className="text-xs text-amber-400 mt-1 block">
                  ⚠️ This token is only displayed once. Store it securely.
                </span>
              </div>

              <div>
                <label className="block text-xs font-semibold text-zinc-400 uppercase tracking-wider mb-1">
                  Connection Command
                </label>
                <div className="flex items-start gap-2">
                  <pre className="flex-1 font-mono bg-zinc-900 border border-zinc-800 p-3 rounded text-zinc-200 text-xs overflow-x-auto whitespace-pre-wrap break-all leading-relaxed max-h-40">
                    {commandText}
                  </pre>
                  <button
                    type="button"
                    onClick={handleCopyCommand}
                    className="p-2 bg-zinc-800 hover:bg-zinc-700 text-white rounded transition-colors focus:outline-none flex items-center justify-center min-w-[36px] mt-1"
                    title="Copy Connection Command"
                  >
                    {copiedCommand ? (
                      <Check className="w-4 h-4 text-emerald-400" />
                    ) : (
                      <Copy className="w-4 h-4" />
                    )}
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div className="flex justify-end pt-2">
            <button
              type="button"
              onClick={onClose}
              className="bg-white text-zinc-950 font-semibold py-2 px-6 rounded hover:opacity-90 transition-opacity focus:outline-none"
            >
              Done
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
      <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-lg shadow-2xl p-6 space-y-4">
        <div className="flex justify-between items-center border-b border-zinc-800 pb-3">
          <h3 className="text-lg font-semibold font-geist text-white">
            Register QPU
          </h3>
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
              <option value="qblox">qblox (Qblox Driver)</option>
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
