import { Cpu, RefreshCw, Timer, Calendar } from "lucide-react";
import type { QPU, QuantumJob, TimeSlot } from "@/types";

interface Props {
  qpus: QPU[];
  jobs: QuantumJob[];
  qpuSeconds: number;
  bookings: TimeSlot[];
}

function formatBookingTime(isoString: string) {
  const d = new Date(isoString);
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function MetricsRow({ qpus, jobs, qpuSeconds, bookings }: Props) {
  const onlineQpus = qpus.filter((q) => q.status === "online").length;
  const pendingJobs = jobs.filter((j) => j.status === "pending").length;
  const runningJobs = jobs.filter((j) => j.status === "running").length;

  const now = new Date();
  const nextBooking = bookings
    .filter((b) => new Date(b.start_time) > now)
    .sort(
      (a, b) =>
        new Date(a.start_time).getTime() - new Date(b.start_time).getTime(),
    )[0];

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <a
        href="#qpus"
        className="bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 p-6 rounded-lg flex flex-col justify-between hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors block cursor-pointer"
      >
        <div className="flex justify-between items-start mb-4">
          <span className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wider">
            Active QPUs
          </span>
          <Cpu className="w-5 h-5 text-gray-400 dark:text-zinc-500" />
        </div>
        <div>
          <div className="text-2xl font-geist font-bold text-gray-900 dark:text-white mb-1">
            {onlineQpus}/{qpus.length}
          </div>
          <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-zinc-400">
            <span className="w-2 h-2 rounded-full bg-green-500"></span> Online
            units
          </div>
        </div>
      </a>

      <a
        href="#jobs"
        className="bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 p-6 rounded-lg flex flex-col justify-between hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors block cursor-pointer"
      >
        <div className="flex justify-between items-start mb-4">
          <span className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wider">
            Queue Status
          </span>
          <RefreshCw className="w-5 h-5 text-gray-400 dark:text-zinc-500" />
        </div>
        <div>
          <div className="text-2xl font-geist font-bold text-gray-900 dark:text-white mb-1">
            {pendingJobs + runningJobs} jobs
          </div>
          <div className="text-xs text-gray-500 dark:text-zinc-400">
            {pendingJobs} pending, {runningJobs} running
          </div>
        </div>
      </a>

      <div className="bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 p-6 rounded-lg flex flex-col justify-between">
        <div className="flex justify-between items-start mb-4">
          <span className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wider">
            Time Credit
          </span>
          <Timer className="w-5 h-5 text-gray-400 dark:text-zinc-500" />
        </div>
        <div>
          <div className="text-2xl font-geist font-bold text-gray-900 dark:text-white mb-1">
            {qpuSeconds}s
          </div>
          <div className="text-xs text-gray-500 dark:text-zinc-400">
            Remaining seconds
          </div>
        </div>
      </div>

      <a
        href="#bookings"
        className="bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 p-6 rounded-lg flex flex-col justify-between hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors block cursor-pointer"
      >
        <div className="flex justify-between items-start mb-4">
          <span className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wider">
            Next Booking
          </span>
          <Calendar className="w-5 h-5 text-gray-400 dark:text-zinc-500" />
        </div>
        <div>
          <div className="text-lg font-geist font-bold text-gray-900 dark:text-white mb-1 truncate">
            {nextBooking
              ? formatBookingTime(nextBooking.start_time)
              : "None Scheduled"}
          </div>
          <div className="text-xs text-gray-500 dark:text-zinc-400">
            {nextBooking ? "Dedicated Window" : "No active reservations"}
          </div>
        </div>
      </a>
    </div>
  );
}
