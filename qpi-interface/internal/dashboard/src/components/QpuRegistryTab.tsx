import React, { useState } from "react";
import { Cpu, Power, Plus, X } from "lucide-react";
import type { QPU } from "../types";

interface QpuRegistryTabProps {
  qpus: QPU[];
  isAdmin: boolean;
  onToggleQpu: (id: string, enabled: boolean) => Promise<void>;
  onRegisterQpu: (name: string, token: string, executor: string) => Promise<void>;
}

export const QpuRegistryTab: React.FC<QpuRegistryTabProps> = ({
  qpus,
  isAdmin,
  onToggleQpu,
  onRegisterQpu,
}) => {
  const [modalOpen, setModalOpen] = useState(false);
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
      await onRegisterQpu(regName, regToken, regExecutor);
      setModalOpen(false);
      setRegName("");
      setRegToken("");
      setRegExecutor("mock");
    } catch (err: any) {
      setError(err?.message || "Registration failed. Check inputs.");
    } finally {
      setLoading(false);
    }
  };

  const handleToggle = async (id: string, currentEnabled: boolean) => {
    try {
      await onToggleQpu(id, !currentEnabled);
    } catch (err: any) {
      alert(`Toggle failed: ${err.message}`);
    }
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-geist text-white">QPU Registry</h1>
          <p className="text-sm text-zinc-400 mt-1">
            Manage and monitor physical/simulator processing units.
          </p>
        </div>
        {isAdmin && (
          <button
            onClick={() => setModalOpen(true)}
            className="w-full md:w-auto bg-white text-zinc-950 font-semibold py-2 px-6 rounded flex items-center justify-center gap-2 hover:opacity-90 transition-opacity focus:outline-none"
          >
            <Plus className="w-4.5 h-4.5" />
            Register QPU
          </button>
        )}
      </div>

      {/* QPU Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {qpus.map((qpu) => {
          const isOnline = qpu.status === "online";
          return (
            <div
              key={qpu.id}
              className="bg-zinc-900 border border-zinc-800 rounded-lg p-6 flex flex-col justify-between hover:border-zinc-700 transition-colors"
            >
              <div>
                <div className="flex justify-between items-start mb-4">
                  <div className="flex items-center gap-3">
                    <div className="p-2 rounded bg-zinc-950 border border-zinc-800">
                      <Cpu className="w-5 h-5 text-white" />
                    </div>
                    <div>
                      <h3 className="font-geist font-bold text-white text-lg leading-tight">
                        {qpu.name}
                      </h3>
                      <p className="text-xs font-mono text-zinc-500 mt-0.5">ID: {qpu.id}</p>
                    </div>
                  </div>
                  <span
                    className={`px-2 py-0.5 border rounded-full text-[10px] uppercase font-semibold flex items-center gap-1 ${
                      isOnline
                        ? "bg-green-500/10 border-green-500/20 text-green-400"
                        : "bg-red-500/10 border-red-500/20 text-red-400"
                    }`}
                  >
                    <span className={`w-1.5 h-1.5 rounded-full ${isOnline ? "bg-green-500" : "bg-red-500"}`} />
                    {qpu.status}
                  </span>
                </div>

                <div className="grid grid-cols-2 gap-4 py-4 my-2 border-t border-b border-zinc-800/50 text-xs">
                  <div>
                    <span className="text-zinc-500 block uppercase tracking-wider text-[10px] mb-1">
                      Executor Driver
                    </span>
                    <span className="font-mono text-zinc-300 bg-zinc-950 px-2 py-0.5 rounded border border-zinc-800">
                      {qpu.executor}
                    </span>
                  </div>
                  <div>
                    <span className="text-zinc-500 block uppercase tracking-wider text-[10px] mb-1">
                      NNG Ports (Cmd/Res)
                    </span>
                    <span className="font-mono text-zinc-300">
                      {qpu.nng_command_port > 0 ? `${qpu.nng_command_port} / ${qpu.nng_result_port}` : "offline"}
                    </span>
                  </div>
                </div>
              </div>

              {isAdmin && (
                <div className="flex justify-between items-center mt-4">
                  <span className="text-xs text-zinc-400">Driver Enable Control</span>
                  <button
                    onClick={() => handleToggle(qpu.id, qpu.enabled)}
                    className={`px-4 py-1.5 rounded text-xs font-semibold flex items-center gap-2 border transition-all focus:outline-none ${
                      qpu.enabled
                        ? "bg-green-500/10 border-green-500/20 text-green-400 hover:bg-green-500/20"
                        : "bg-red-500/10 border-red-500/20 text-red-400 hover:bg-red-500/20"
                    }`}
                  >
                    <Power className="w-3.5 h-3.5" />
                    {qpu.enabled ? "Online (Enabled)" : "Offline (Disabled)"}
                  </button>
                </div>
              )}
            </div>
          );
        })}
      </div>

      {/* Register Modal overlay */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-lg shadow-2xl p-6 space-y-4">
            <div className="flex justify-between items-center border-b border-zinc-800 pb-3">
              <h3 className="text-lg font-semibold font-geist text-white">Register QPU</h3>
              <button
                onClick={() => setModalOpen(false)}
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
      )}
    </div>
  );
};
