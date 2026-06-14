import React from "react";
import type { QPU, QuantumJob, Notification, TimeSlot } from "@/types";
import { MetricsRow } from "./elements/MetricsRow";
import { RecentJobsTable } from "./elements/RecentJobsTable";
import { NotificationsPanel } from "./elements/NotificationsPanel";

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
      <MetricsRow qpus={qpus} jobs={jobs} qpuSeconds={qpuSeconds} bookings={bookings} />

      {/* Split Table / Notifications */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Recent Jobs Table */}
        <div className="lg:col-span-8">
          <RecentJobsTable jobs={jobs} qpus={qpus} />
        </div>

        {/* Notifications Sidebar */}
        <div className="lg:col-span-4">
          <NotificationsPanel notifications={notifications} onDismiss={onDismissNotification} />
        </div>
      </div>
    </div>
  );
};
