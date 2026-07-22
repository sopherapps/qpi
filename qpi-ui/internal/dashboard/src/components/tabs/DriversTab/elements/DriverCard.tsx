import { useState } from "react";
import { Workflow, Power, Trash2, AlertTriangle } from "lucide-react";
import type { Driver } from "@/types";

interface Props {
  driver: Driver;
  isAdmin: boolean;
  onToggle: (id: string, enabled: boolean) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

const statusStyles: Record<Driver["status"], string> = {
  online: "bg-green-500/10 border-green-500/20 text-green-400",
  offline: "bg-red-500/10 border-red-500/20 text-red-400",
  maintenance: "bg-amber-500/10 border-amber-500/20 text-amber-400",
};

const statusDotStyles: Record<Driver["status"], string> = {
  online: "bg-green-500",
  offline: "bg-red-500",
  maintenance: "bg-amber-500",
};

function formatLastSeen(lastSeen?: string) {
  if (!lastSeen) return "never";
  const date = new Date(lastSeen);
  if (Number.isNaN(date.getTime())) return "never";
  return date.toLocaleString();
}

export function DriverCard({ driver, isAdmin, onToggle, onDelete }: Props) {
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const handleToggle = async () => {
    try {
      await onToggle(driver.id, !driver.enabled);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Toggle failed";
      alert(`Toggle failed: ${message}`);
    }
  };

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      await onDelete(driver.id);
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
                <Workflow className="w-5 h-5 text-gray-900 dark:text-white" />
              </div>
              <div>
                <h3 className="font-geist font-bold text-gray-900 dark:text-white text-lg leading-tight">
                  {driver.name}
                </h3>
                <p className="text-xs font-mono text-gray-400 dark:text-zinc-500 mt-0.5">
                  QPU: {driver.expand?.qpu?.name ?? driver.qpu}
                </p>
              </div>
            </div>
            <span
              data-testid="driver-status"
              className={`px-2 py-0.5 border rounded-full text-[10px] uppercase font-semibold flex items-center gap-1 ${statusStyles[driver.status]}`}
            >
              <span
                className={`w-1.5 h-1.5 rounded-full ${statusDotStyles[driver.status]}`}
              />
              {driver.status}
            </span>
          </div>

          <div className="grid grid-cols-2 gap-4 py-4 my-2 border-t border-b border-gray-200 dark:border-zinc-800/50 text-xs">
            <div>
              <span className="text-gray-400 dark:text-zinc-500 block uppercase tracking-wider text-[10px] mb-1">
                Kind / Language
              </span>
              <span
                data-testid="kind-language-value"
                className="font-mono text-gray-600 dark:text-zinc-300 bg-gray-50 dark:bg-zinc-950 px-2 py-0.5 rounded border border-gray-200 dark:border-zinc-800"
              >
                {driver.kind} / {driver.language}
              </span>
            </div>
            <div>
              <span className="text-gray-400 dark:text-zinc-500 block uppercase tracking-wider text-[10px] mb-1">
                Last Seen
              </span>
              <span
                data-testid="last-seen-value"
                className="font-mono text-gray-600 dark:text-zinc-300"
              >
                {formatLastSeen(driver.last_seen)}
              </span>
            </div>
          </div>
        </div>

        {isAdmin && (
          <div className="flex justify-between items-center mt-4">
            <button
              onClick={() => setDeleteModalOpen(true)}
              className="px-3 py-1.5 rounded text-xs font-semibold flex items-center gap-1.5 border border-red-500/20 text-red-400 bg-red-500/10 hover:bg-red-500/20 transition-all focus:outline-none"
              title="Delete Driver"
            >
              <Trash2 className="w-3.5 h-3.5" />
              Delete
            </button>

            <div className="flex items-center gap-3">
              <span className="text-xs text-gray-500 dark:text-zinc-400">
                Driver Enable Control
              </span>
              <button
                onClick={handleToggle}
                className={`px-4 py-1.5 rounded text-xs font-semibold flex items-center gap-2 border transition-all focus:outline-none ${
                  driver.enabled
                    ? "bg-green-500/10 border-green-500/20 text-green-400 hover:bg-green-500/20"
                    : "bg-red-500/10 border-red-500/20 text-red-400 hover:bg-red-500/20"
                }`}
              >
                <Power className="w-3.5 h-3.5" />
                {driver.enabled ? "Enabled" : "Disabled"}
              </button>
            </div>
          </div>
        )}
      </div>

      {deleteModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-50 dark:bg-zinc-950/80 backdrop-blur-sm animate-fade-in">
          <div className="w-full max-w-sm bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 rounded-lg shadow-2xl p-6 space-y-5">
            <div className="flex items-start gap-4">
              <div className="p-3 bg-red-500/10 rounded-full">
                <AlertTriangle className="w-6 h-6 text-red-400" />
              </div>
              <div>
                <h3 className="text-lg font-semibold font-geist text-gray-900 dark:text-white">
                  Delete Driver
                </h3>
                <p className="text-sm text-gray-500 dark:text-zinc-400 mt-1">
                  Are you sure you want to delete{" "}
                  <span className="font-bold text-gray-900 dark:text-white">
                    {driver.name}
                  </span>
                  ? This action cannot be undone and will terminate its
                  connection.
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
                {isDeleting ? "Deleting..." : "Delete Driver"}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
