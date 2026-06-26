import { useState } from "react";
import { Cpu, Power, Trash2, AlertTriangle } from "lucide-react";
import type { QPU } from "@/types";

interface Props {
  qpu: QPU;
  isAdmin: boolean;
  onToggle: (id: string, enabled: boolean) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

export function QpuCard({ qpu, isAdmin, onToggle, onDelete }: Props) {
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const isOnline = qpu.status === "online";

  const handleToggle = async () => {
    try {
      await onToggle(qpu.id, !qpu.enabled);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Toggle failed";
      alert(`Toggle failed: ${message}`);
    }
  };

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      await onDelete(qpu.id);
      setDeleteModalOpen(false);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Delete failed";
      alert(`Delete failed: ${message}`);
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <>
    <div className="bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 rounded-lg p-6 flex flex-col justify-between hover:border-gray-300 dark:border-zinc-700 transition-colors">
      <div>
        <div className="flex justify-between items-start mb-4">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded bg-gray-50 dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800">
              <Cpu className="w-5 h-5 text-gray-900 dark:text-white" />
            </div>
            <div>
              <h3 className="font-geist font-bold text-gray-900 dark:text-white text-lg leading-tight">
                {qpu.name}
              </h3>
              <p className="text-xs font-mono text-gray-400 dark:text-zinc-500 mt-0.5">
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

        <div className="grid grid-cols-2 gap-4 py-4 my-2 border-t border-b border-gray-200 dark:border-zinc-800/50 text-xs">
          <div>
            <span className="text-gray-400 dark:text-zinc-500 block uppercase tracking-wider text-[10px] mb-1">
              Executor Driver
            </span>
            <span data-testid="executor-value" className="font-mono text-gray-600 dark:text-zinc-300 bg-gray-50 dark:bg-zinc-950 px-2 py-0.5 rounded border border-gray-200 dark:border-zinc-800">
              {qpu.executor_type}
            </span>
          </div>
          <div>
            <span className="text-gray-400 dark:text-zinc-500 block uppercase tracking-wider text-[10px] mb-1">
              NNG Ports (Cmd/Res)
            </span>
            <span className="font-mono text-gray-600 dark:text-zinc-300">
              {qpu.nng_command_port > 0
                ? `${qpu.nng_command_port} / ${qpu.nng_result_port}`
                : "offline"}
            </span>
          </div>
        </div>
      </div>

      {isAdmin && (
        <div className="flex justify-between items-center mt-4">
          <button
            onClick={() => setDeleteModalOpen(true)}
            className="px-3 py-1.5 rounded text-xs font-semibold flex items-center gap-1.5 border border-red-500/20 text-red-400 bg-red-500/10 hover:bg-red-500/20 transition-all focus:outline-none"
            title="Delete QPU"
          >
            <Trash2 className="w-3.5 h-3.5" />
            Delete
          </button>
          
          <div className="flex items-center gap-3">
            <span className="text-xs text-gray-500 dark:text-zinc-400">Driver Enable Control</span>
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
        </div>
      )}
    </div>

    {/* Delete Confirmation Modal */}
    {deleteModalOpen && (
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-50 dark:bg-zinc-950/80 backdrop-blur-sm animate-fade-in">
        <div className="w-full max-w-sm bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 rounded-lg shadow-2xl p-6 space-y-5">
          <div className="flex items-start gap-4">
            <div className="p-3 bg-red-500/10 rounded-full">
              <AlertTriangle className="w-6 h-6 text-red-400" />
            </div>
            <div>
              <h3 className="text-lg font-semibold font-geist text-gray-900 dark:text-white">Delete QPU</h3>
              <p className="text-sm text-gray-500 dark:text-zinc-400 mt-1">
                Are you sure you want to delete <span className="font-bold text-gray-900 dark:text-white">{qpu.name}</span>? This action cannot be undone and will terminate any active driver connection.
              </p>
            </div>
          </div>
          <div className="flex justify-end gap-3 pt-2">
            <button
              onClick={() => setDeleteModalOpen(false)}
              disabled={isDeleting}
              className="px-4 py-2 text-sm font-semibold text-gray-600 dark:text-zinc-300 hover:text-gray-900 dark:text-white transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              onClick={handleDelete}
              disabled={isDeleting}
              className="px-4 py-2 bg-red-500 text-zinc-950 text-sm font-semibold rounded hover:bg-red-400 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              {isDeleting ? "Deleting..." : "Delete QPU"}
            </button>
          </div>
        </div>
      </div>
    )}
    </>
  );
}
