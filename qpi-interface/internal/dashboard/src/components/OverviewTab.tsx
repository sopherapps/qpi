import React from "react";
import { Cpu, RefreshCw, Timer, Calendar, AlertTriangle, Info, X } from "lucide-react";
import type { QPU, QuantumJob, Notification, TimeSlot } from "../types";

interface OverviewTabProps {
  qpus: QPU[];
  jobs: QuantumJob[];
  qpuSeconds: number;
  bookings: TimeSlot[];
  notifications: Notification[];
  onDismissNotification: (id: string) => void;
  switchTab: (tab: string) => void;
}

export const OverviewTab: React.FC<OverviewTabProps> = ({
  qpus,
  jobs,
  qpuSeconds,
  bookings,
  notifications,
  onDismissNotification,
  switchTab,
}) => {
  // Metrics calculations
  const onlineQpus = qpus.filter((q) => q.status === "online").length;
  const pendingJobs = jobs.filter((j) => j.status === "pending").length;
  const runningJobs = jobs.filter((j) => j.status === "running").length;

  // Find next upcoming booking for current user
  const now = new Date();
  const nextBooking = bookings
    .filter((b) => new Date(b.start_time) > now)
    .sort((a, b) => new Date(a.start_time).getTime() - new Date(b.start_time).getTime())[0];

  const formatBookingTime = (isoString: string) => {
    const d = new Date(isoString);
    return d.toLocaleString(undefined, {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const getStatusBadgeClass = (status: string) => {
    switch (status) {
      case "completed":
        return "bg-green-500/10 border-green-500/20 text-green-400";
      case "pending":
        return "bg-yellow-500/10 border-yellow-500/20 text-yellow-400";
      case "running":
        return "bg-indigo-500/10 border-indigo-500/20 text-indigo-400";
      case "failed":
        return "bg-red-500/10 border-red-500/20 text-red-400";
      case "cancelled":
        return "bg-zinc-500/10 border-zinc-500/20 text-zinc-400";
      default:
        return "bg-zinc-500/10 border-zinc-500/20 text-zinc-400";
    }
  };

  return (
    <div className="space-y-8">
      {/* Title & Actions */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <h1 className="text-3xl font-geist text-white">Overview</h1>
          <p className="text-sm text-zinc-400 mt-1">System status and resource consumption.</p>
        </div>
        <div className="flex gap-3 w-full md:w-auto">
          <button
            onClick={() => switchTab("bookings")}
            className="flex-1 md:flex-none border border-zinc-800 bg-transparent text-white px-4 py-2 rounded font-medium text-sm hover:bg-zinc-800 transition-colors focus:outline-none"
          >
            Book Slot
          </button>
          <button
            onClick={() => switchTab("jobs")}
            className="flex-1 md:flex-none bg-white text-zinc-950 px-4 py-2 rounded font-medium text-sm hover:opacity-90 transition-opacity focus:outline-none"
          >
            Submit Job
          </button>
        </div>
      </div>

      {/* Metrics Row */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-zinc-900 border border-zinc-800 p-6 rounded-lg flex flex-col justify-between">
          <div className="flex justify-between items-start mb-4">
            <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">
              Active QPUs
            </span>
            <Cpu className="w-5 h-5 text-zinc-500" />
          </div>
          <div>
            <div className="text-2xl font-geist font-bold text-white mb-1">
              {onlineQpus}/{qpus.length}
            </div>
            <div className="flex items-center gap-2 text-xs text-zinc-400">
              <span className="w-2 h-2 rounded-full bg-green-500"></span> Online units
            </div>
          </div>
        </div>

        <div className="bg-zinc-900 border border-zinc-800 p-6 rounded-lg flex flex-col justify-between">
          <div className="flex justify-between items-start mb-4">
            <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">
              Queue Status
            </span>
            <RefreshCw className="w-5 h-5 text-zinc-500" />
          </div>
          <div>
            <div className="text-2xl font-geist font-bold text-white mb-1">
              {pendingJobs + runningJobs} jobs
            </div>
            <div className="text-xs text-zinc-400">
              {pendingJobs} pending, {runningJobs} running
            </div>
          </div>
        </div>

        <div className="bg-zinc-900 border border-zinc-800 p-6 rounded-lg flex flex-col justify-between">
          <div className="flex justify-between items-start mb-4">
            <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">
              Time Credit
            </span>
            <Timer className="w-5 h-5 text-zinc-500" />
          </div>
          <div>
            <div className="text-2xl font-geist font-bold text-white mb-1">{qpuSeconds}s</div>
            <div className="text-xs text-zinc-400">Remaining seconds</div>
          </div>
        </div>

        <div className="bg-zinc-900 border border-zinc-800 p-6 rounded-lg flex flex-col justify-between">
          <div className="flex justify-between items-start mb-4">
            <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">
              Next Booking
            </span>
            <Calendar className="w-5 h-5 text-zinc-500" />
          </div>
          <div>
            <div className="text-lg font-geist font-bold text-white mb-1 truncate">
              {nextBooking ? formatBookingTime(nextBooking.start_time) : "None Scheduled"}
            </div>
            <div className="text-xs text-zinc-400">
              {nextBooking ? "Dedicated Window" : "No active reservations"}
            </div>
          </div>
        </div>
      </div>

      {/* Split Table / Notifications */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Recent Jobs Table */}
        <div className="lg:col-span-8 space-y-4">
          <h3 className="text-lg font-semibold text-white font-geist">Recent Job Executions</h3>
          <div className="bg-zinc-900 border border-zinc-800 rounded-lg overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b border-zinc-800 text-zinc-400 text-xs font-semibold uppercase tracking-wider bg-zinc-900/50">
                    <th className="py-3 px-4">Job ID</th>
                    <th className="py-3 px-4">QPU Target</th>
                    <th className="py-3 px-4">Status</th>
                    <th className="py-3 px-4">Created</th>
                    <th className="py-3 px-4 text-right">Finished</th>
                  </tr>
                </thead>
                <tbody className="text-sm text-zinc-300 divide-y divide-zinc-800/50">
                  {jobs.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="py-8 px-4 text-center text-zinc-500">
                        No jobs submitted yet.
                      </td>
                    </tr>
                  ) : (
                    jobs.slice(0, 10).map((job) => {
                      const targetQpu = qpus.find((q) => q.id === job.qpu_target)?.name || job.qpu_target;
                      return (
                        <tr key={job.id} className="hover:bg-zinc-800/20 transition-colors">
                          <td className="py-3.5 px-4 font-mono text-xs text-white">{job.id}</td>
                          <td className="py-3.5 px-4 text-zinc-400">{targetQpu}</td>
                          <td className="py-3.5 px-4">
                            <span
                              className={`inline-flex px-2 py-0.5 rounded-full border text-[10px] uppercase font-semibold ${getStatusBadgeClass(
                                job.status
                              )}`}
                            >
                              {job.status}
                            </span>
                          </td>
                          <td className="py-3.5 px-4 text-zinc-400 text-xs">
                            {new Date(job.created).toLocaleString()}
                          </td>
                          <td className="py-3.5 px-4 text-zinc-400 text-xs text-right">
                            {job.finished_at ? new Date(job.finished_at).toLocaleString() : "-"}
                          </td>
                        </tr>
                      );
                    })
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>

        {/* Notifications Sidebar */}
        <div className="lg:col-span-4 space-y-4">
          <h3 className="text-lg font-semibold text-white font-geist">System Announcements</h3>
          <div className="flex flex-col gap-3">
            {notifications.length === 0 ? (
              <div className="text-zinc-500 text-center py-8 bg-zinc-900/30 border border-zinc-800/50 rounded-lg">
                No active announcements
              </div>
            ) : (
              notifications.map((ann) => {
                const isFail =
                  ann.title.toLowerCase().includes("fail") ||
                  ann.title.toLowerCase().includes("error");
                return (
                  <div
                    key={ann.id}
                    className={`border p-4 rounded flex justify-between items-start gap-4 hover:border-zinc-700 transition-colors relative group ${
                      isFail
                        ? "bg-red-500/10 border-red-500/20 text-red-200"
                        : "bg-zinc-900 border-zinc-800 text-zinc-300"
                    }`}
                  >
                    <div className="flex items-start gap-3">
                      {isFail ? (
                        <AlertTriangle className="w-5 h-5 text-red-400 mt-0.5" />
                      ) : (
                        <Info className="w-5 h-5 text-zinc-400 mt-0.5" />
                      )}
                      <div>
                        <p className="font-medium text-xs text-white">{ann.title}</p>
                        <p className="text-[11px] text-zinc-400 mt-1">{ann.description}</p>
                      </div>
                    </div>
                    <button
                      onClick={() => onDismissNotification(ann.id)}
                      className="text-zinc-500 hover:text-white opacity-0 group-hover:opacity-100 transition-opacity focus:outline-none"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                );
              })
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
