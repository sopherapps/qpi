import type { QPU, QuantumJob } from "@/types";

interface Props {
  jobs: QuantumJob[];
  qpus: QPU[];
}

function getStatusBadgeClass(status: string) {
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
}

export function RecentJobsTable({ jobs, qpus }: Props) {
  return (
    <div className="space-y-4">
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
  );
}
