import { Cpu, Power } from "lucide-react";
import type { QPU } from "@/types";

interface Props {
  qpu: QPU;
  isAdmin: boolean;
  onToggle: (id: string, enabled: boolean) => Promise<void>;
}

export function QpuCard({ qpu, isAdmin, onToggle }: Props) {
  const isOnline = qpu.status === "online";

  const handleToggle = async () => {
    try {
      await onToggle(qpu.id, !qpu.enabled);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Toggle failed";
      alert(`Toggle failed: ${message}`);
    }
  };

  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-6 flex flex-col justify-between hover:border-zinc-700 transition-colors">
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
              <p className="text-xs font-mono text-zinc-500 mt-0.5">
                ID: {qpu.id}
              </p>
            </div>
          </div>
          <span
            className={`px-2 py-0.5 border rounded-full text-[10px] uppercase font-semibold flex items-center gap-1 ${
              isOnline
                ? "bg-green-500/10 border-green-500/20 text-green-400"
                : "bg-red-500/10 border-red-500/20 text-red-400"
            }`}
          >
            <span
              className={`w-1.5 h-1.5 rounded-full ${isOnline ? "bg-green-500" : "bg-red-500"}`}
            />
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
              {qpu.nng_command_port > 0
                ? `${qpu.nng_command_port} / ${qpu.nng_result_port}`
                : "offline"}
            </span>
          </div>
        </div>
      </div>

      {isAdmin && (
        <div className="flex justify-between items-center mt-4">
          <span className="text-xs text-zinc-400">Driver Enable Control</span>
          <button
            onClick={handleToggle}
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
}
