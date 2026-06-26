import React, { useState, useEffect } from "react";
import type { QPU, QuantumJob } from "@/types";
import { pb } from "@/lib/pb";
import { JobSubmissionPanel } from "./elements/JobSubmissionPanel";
import { JobResultsPanel } from "./elements/JobResultsPanel";

interface JobsConsoleTabProps {
  qpus: QPU[];
  selectedJobId: string | null;
  setSelectedJobId: (id: string | null) => void;
  onSubmitJob: (
    qpuId: string,
    qasm: string,
    shots: number,
    measLevel: number,
  ) => Promise<string>;
}

export const JobsConsoleTab: React.FC<JobsConsoleTabProps> = ({
  qpus,
  selectedJobId,
  setSelectedJobId,
  onSubmitJob,
}) => {
  const [viewedJob, setViewedJob] = useState<QuantumJob | null>(null);

  // Load viewed job details when selectedJobId changes
  useEffect(() => {
    if (!selectedJobId) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setViewedJob(null);
      return;
    }

    const fetchJob = async () => {
      try {
        const job = await pb.collection("quantum_jobs").getOne(selectedJobId);
        setViewedJob(job as unknown as QuantumJob);

        // Auto-refresh if pending/running
        if (job.status === "pending" || job.status === "running") {
          const timeout = setTimeout(fetchJob, 1500);
          return () => clearTimeout(timeout);
        }
      } catch (err) {
        console.error("Error fetching job details:", err);
      }
    };

    fetchJob();
  }, [selectedJobId]);

  return (
    <div className="space-y-6 h-full flex flex-col">
      <div>
        <h1 className="text-3xl font-geist text-gray-900 dark:text-white">Jobs Console</h1>
        <p className="text-sm text-gray-500 dark:text-zinc-400 mt-1">
          Configure, write QASM, and execute circuits.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 flex-1 min-h-[500px]">
        {/* Left submissions pane */}
        <div className="lg:col-span-5">
          <JobSubmissionPanel
            qpus={qpus}
            onSubmitJob={onSubmitJob}
            onJobSubmitted={setSelectedJobId}
          />
        </div>

        {/* Right results pane */}
        <div className="lg:col-span-7">
          <JobResultsPanel viewedJob={viewedJob} qpus={qpus} />
        </div>
      </div>
    </div>
  );
};
