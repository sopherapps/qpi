import { useState } from "react";
import { Check, X } from "lucide-react";
import type { User as UserType, TimeRequest } from "../types";

interface AdminPanelTabProps {
  users: UserType[];
  timeRequests: TimeRequest[];
  onAllocateTime: (userId: string, seconds: number) => Promise<void>;
  onBroadcastAnnouncement: (title: string, desc: string, start: string, end: string) => Promise<void>;
  onApproveRequest: (id: string) => Promise<void>;
  onRejectRequest: (id: string, reason: string) => Promise<void>;
}

export const AdminPanelTab: React.FC<AdminPanelTabProps> = ({
  users,
  timeRequests,
  onAllocateTime,
  onBroadcastAnnouncement,
  onApproveRequest,
  onRejectRequest,
}) => {
  const [subtab, setSubtab] = useState<"users" | "announcements" | "requests">("users");

  // Allocate time state
  const [allocateAmounts, setAllocateAmounts] = useState<Record<string, string>>({});

  // Announcement state
  const [annTitle, setAnnTitle] = useState("");
  const [annDesc, setAnnDesc] = useState("");
  const [annStart, setAnnStart] = useState("");
  const [annEnd, setAnnEnd] = useState("");
  const [annLoading, setAnnLoading] = useState(false);
  const [annSuccess, setAnnSuccess] = useState(false);

  const handleAllocate = async (userId: string) => {
    const amount = parseFloat(allocateAmounts[userId] || "");
    if (isNaN(amount) || amount <= 0) {
      alert("Please enter a valid positive number of seconds.");
      return;
    }

    try {
      await onAllocateTime(userId, amount);
      setAllocateAmounts((prev) => ({ ...prev, [userId]: "" }));
      alert("Quota updated successfully!");
    } catch (err: any) {
      alert(`Allocation failed: ${err.message}`);
    }
  };

  const handleBroadcast = async (e: React.FormEvent) => {
    e.preventDefault();
    setAnnLoading(true);
    setAnnSuccess(false);

    try {
      // announcements/notifications start/end time are optional in PB
      const startIso = annStart ? new Date(annStart).toISOString() : "";
      const endIso = annEnd ? new Date(annEnd).toISOString() : "";
      await onBroadcastAnnouncement(annTitle, annDesc, startIso, endIso);

      setAnnTitle("");
      setAnnDesc("");
      setAnnStart("");
      setAnnEnd("");
      setAnnSuccess(true);
      setTimeout(() => setAnnSuccess(false), 3000);
    } catch (err: any) {
      alert(`Broadcast failed: ${err.message}`);
    } finally {
      setAnnLoading(false);
    }
  };

  const handleApprove = async (id: string) => {
    try {
      await onApproveRequest(id);
    } catch (err: any) {
      alert(`Approval failed: ${err.message}`);
    }
  };

  const handleReject = async (id: string) => {
    const reason = prompt("Enter rejection reason:");
    if (reason === null) return; // cancelled
    try {
      await onRejectRequest(id, reason);
    } catch (err: any) {
      alert(`Rejection failed: ${err.message}`);
    }
  };

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-geist text-white">Admin Panel</h1>
        <p className="text-sm text-zinc-400 mt-1">
          Superuser controls for quota management and broadcasts.
        </p>
      </div>

      {/* Inner Tabs */}
      <div className="flex border-b border-zinc-800 mb-6">
        <button
          onClick={() => setSubtab("users")}
          className={`px-4 py-2 font-geist text-sm transition-all -mb-[1px] ${
            subtab === "users"
              ? "text-white border-b-2 border-white font-medium"
              : "text-zinc-500 hover:text-zinc-300"
          }`}
        >
          User Time Allocations
        </button>
        <button
          onClick={() => setSubtab("announcements")}
          className={`px-4 py-2 font-geist text-sm transition-all -mb-[1px] ${
            subtab === "announcements"
              ? "text-white border-b-2 border-white font-medium"
              : "text-zinc-500 hover:text-zinc-300"
          }`}
        >
          Broadcast Announcement
        </button>
        <button
          onClick={() => setSubtab("requests")}
          className={`px-4 py-2 font-geist text-sm transition-all -mb-[1px] ${
            subtab === "requests"
              ? "text-white border-b-2 border-white font-medium"
              : "text-zinc-500 hover:text-zinc-300"
          }`}
        >
          Time Requests
        </button>
      </div>

      {/* Panel 1: User allocations */}
      {subtab === "users" && (
        <div className="bg-zinc-900 border border-zinc-800 rounded-lg overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-zinc-800 text-zinc-400 text-xs font-semibold uppercase tracking-wider bg-zinc-900/50">
                  <th className="py-3 px-4">User ID / Username</th>
                  <th className="py-3 px-4">Email</th>
                  <th className="py-3 px-4">QPU Balance</th>
                  <th className="py-3 px-4 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="text-sm text-zinc-300 divide-y divide-zinc-800/50">
                {users.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="py-8 px-4 text-center text-zinc-500">
                      Loading registered users...
                    </td>
                  </tr>
                ) : (
                  users.map((u) => (
                    <tr key={u.id} className="hover:bg-zinc-800/20 transition-colors">
                      <td className="py-3.5 px-4 font-mono text-xs text-white">{u.id}</td>
                      <td className="py-3.5 px-4 text-zinc-400">{u.email}</td>
                      <td className="py-3.5 px-4 font-mono text-zinc-300">{u.qpu_seconds}s</td>
                      <td className="py-3.5 px-4 text-right">
                        <div className="inline-flex items-center gap-2">
                          <input
                            type="number"
                            placeholder="Add sec"
                            value={allocateAmounts[u.id] || ""}
                            onChange={(e) =>
                              setAllocateAmounts((prev) => ({ ...prev, [u.id]: e.target.value }))
                            }
                            className="bg-zinc-950 border border-zinc-800 text-white rounded px-2 py-1 text-xs w-24 focus:outline-none focus:border-zinc-500 font-mono"
                          />
                          <button
                            onClick={() => handleAllocate(u.id)}
                            className="bg-white text-zinc-950 px-3 py-1 rounded text-xs font-semibold hover:opacity-90 transition-all focus:outline-none"
                          >
                            Grant
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Panel 2: Compose announcement */}
      {subtab === "announcements" && (
        <div className="max-w-xl bg-zinc-900 border border-zinc-800 rounded-lg p-6 space-y-4">
          <h3 className="text-lg font-semibold text-white font-geist">Compose Announcement</h3>
          <form onSubmit={handleBroadcast} className="space-y-4 text-sm">
            <div>
              <label className="block text-xs font-medium text-zinc-400 mb-1.5 uppercase tracking-wider">
                Announcement Title
              </label>
              <input
                type="text"
                required
                value={annTitle}
                onChange={(e) => setAnnTitle(e.target.value)}
                className="w-full bg-zinc-950 border border-zinc-800 rounded px-3 py-2 text-white focus:outline-none focus:border-zinc-500 transition-colors"
                placeholder="QPU Maintenance Schedule"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-400 mb-1.5 uppercase tracking-wider">
                Description
              </label>
              <textarea
                required
                value={annDesc}
                onChange={(e) => setAnnDesc(e.target.value)}
                className="w-full bg-zinc-950 border border-zinc-800 rounded px-3 py-2 text-white focus:outline-none focus:border-zinc-500 transition-colors h-24"
                placeholder="Rigetti Aspen-9 will be offline for calibration tomorrow..."
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1.5 uppercase tracking-wider">
                  Start Time
                </label>
                <input
                  type="datetime-local"
                  value={annStart}
                  onChange={(e) => setAnnStart(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded px-3 py-2 text-white focus:outline-none focus:border-zinc-500 transition-colors"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1.5 uppercase tracking-wider">
                  End Time
                </label>
                <input
                  type="datetime-local"
                  value={annEnd}
                  onChange={(e) => setAnnEnd(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded px-3 py-2 text-white focus:outline-none focus:border-zinc-500 transition-colors"
                />
              </div>
            </div>

            {annSuccess && (
              <div className="text-xs text-green-400 bg-green-500/10 border border-green-500/20 p-2.5 rounded font-medium">
                Announcement broadcasted successfully!
              </div>
            )}

            <button
              type="submit"
              disabled={annLoading}
              className="bg-white text-zinc-950 font-semibold py-2 px-6 rounded hover:opacity-90 transition-opacity focus:outline-none disabled:opacity-50"
            >
              {annLoading ? "Broadcasting..." : "Broadcast Announcement"}
            </button>
          </form>
        </div>
      )}

      {/* Panel 3: Time requests */}
      {subtab === "requests" && (
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
                                onClick={() => handleApprove(req.id)}
                                className="bg-green-600 hover:bg-green-500 text-white p-1 rounded transition-colors focus:outline-none"
                              >
                                <Check className="w-4 h-4" />
                              </button>
                              <button
                                onClick={() => handleReject(req.id)}
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
      )}
    </div>
  );
};
