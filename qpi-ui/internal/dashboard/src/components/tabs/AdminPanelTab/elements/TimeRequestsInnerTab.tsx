import { Check, X } from "lucide-react";
import type { TimeRequest } from "../../types";

interface Props {
  timeRequests: TimeRequest[];
  onApprove: (id: string) => void;
  onReject: (id: string) => void;
}

export function TimeRequestsInnerTab({ timeRequests, onApprove, onReject }: Props) {
  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="border-b border-zinc-800 text-zinc-400 text-xs font-semibold uppercase tracking-wider bg-zinc-900/50">
              <th className="py-3 px-4">User</th>
              <th className="py-3 px-4">Requested (seconds)</th>
              <th className="py-3 px-4">Reason</th>
              <th className="py-3 px-4">Status</th>
              <th className="py-3 px-4 text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="text-sm text-zinc-300 divide-y divide-zinc-800/50">
            {timeRequests.length === 0 ? (
              <tr>
                <td colSpan={5} className="py-8 px-4 text-center text-zinc-500">
                  No time requests found.
                </td>
              </tr>
            ) : (
              timeRequests.map((req) => {
                const userName = req.expand?.user?.email || req.user;
                const isPending = req.status === "pending";
                return (
                  <tr key={req.id} className="hover:bg-zinc-800/20 transition-colors">
                    <td className="py-3.5 px-4 font-medium text-white">{userName}</td>
                    <td className="py-3.5 px-4 font-mono text-zinc-300">{req.seconds}s</td>
                    <td className="py-3.5 px-4 text-zinc-400 text-xs">{req.reason}</td>
                    <td className="py-3.5 px-4">
                      <span
                        className={`inline-flex px-2 py-0.5 rounded-full border text-[10px] uppercase font-semibold ${
                          req.status === "approved"
                            ? "border-green-500/30 bg-green-500/10 text-green-400"
                            : req.status === "pending"
                            ? "border-yellow-500/30 bg-yellow-500/10 text-yellow-400"
                            : "border-red-500/30 bg-red-500/10 text-red-400"
                        }`}
                      >
                        {req.status}
                      </span>
                    </td>
                    <td className="py-3.5 px-4 text-right">
                      {isPending ? (
                        <div className="inline-flex gap-2">
                          <button
                            onClick={() => onApprove(req.id)}
                            className="bg-green-600 hover:bg-green-500 text-white p-1 rounded transition-colors focus:outline-none"
                          >
                            <Check className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => onReject(req.id)}
                            className="bg-red-600 hover:bg-red-500 text-white p-1 rounded transition-colors focus:outline-none"
                          >
                            <X className="w-4 h-4" />
                          </button>
                        </div>
                      ) : (
                        <span className="text-xs text-zinc-500 font-medium">Processed</span>
                      )}
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
