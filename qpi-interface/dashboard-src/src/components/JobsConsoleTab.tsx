import React, { useState, useEffect } from "react";
import { Copy, Play, Loader2 } from "lucide-react";
import type { QPU, QuantumJob } from "../types";
import { pb } from "../lib/pb";

interface JobsConsoleTabProps {
  qpus: QPU[];
  selectedJobId: string | null;
  setSelectedJobId: (id: string | null) => void;
  onSubmitJob: (qpuId: string, qasm: string, shots: number, measLevel: number) => Promise<string>;
}

export const JobsConsoleTab: React.FC<JobsConsoleTabProps> = ({
  qpus,
  selectedJobId,
  setSelectedJobId,
  onSubmitJob,
}) => {
  const activeQpus = qpus.filter((q) => q.status === "online");

  const [targetQpu, setTargetQpu] = useState("");
  const [qasmCode, setQasmCode] = useState(
    'OPENQASM 3.0;\ninclude "stdgates.inc";\nqubit[2] q;\nbit[2] c;\nh q[0];\ncx q[0], q[1];\nc = measure q;'
  );
  const [shots, setShots] = useState(1000);
  const [measLevel, setMeasLevel] = useState(2);
  const [activeVisualizerTab, setActiveVisualizerTab] = useState<"counts" | "iq" | "trace">("counts");
  const [executing, setExecuting] = useState(false);
  const [viewedJob, setViewedJob] = useState<QuantumJob | null>(null);

  // Set default target QPU once loaded
  useEffect(() => {
    if (activeQpus.length > 0 && !targetQpu) {
      setTargetQpu(activeQpus[0].id);
    }
  }, [activeQpus, targetQpu]);

  // Load viewed job details when selectedJobId changes
  useEffect(() => {
    if (!selectedJobId) {
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

  const loadExample = () => {
    setQasmCode(
      'OPENQASM 3.0;\ninclude "stdgates.inc";\nqubit[2] q;\nbit[2] c;\nh q[0];\ncx q[0], q[1];\nc = measure q;'
    );
  };

  const handleExecute = async () => {
    if (!targetQpu) {
      alert("Please select a target QPU first.");
      return;
    }
    setExecuting(true);
    try {
      const newJobId = await onSubmitJob(targetQpu, qasmCode, shots, measLevel);
      setSelectedJobId(newJobId);
    } catch (err: any) {
      alert(`Job submission failed: ${err.message}`);
    } finally {
      setExecuting(false);
    }
  };

  const getMeasLevelLabel = (lvl: number) => {
    switch (lvl) {
      case 0:
        return "0 (Trace)";
      case 1:
        return "1 (IQ Memory)";
      case 2:
      default:
        return "2 (Counts)";
    }
  };

  // Chart rendering functions
  const renderChart = () => {
    if (!viewedJob) {
      return (
        <div className="text-zinc-500 text-sm">
          Select a completed job to view results visualization.
        </div>
      );
    }

    if (viewedJob.status !== "completed") {
      return (
        <div className="text-zinc-500 text-sm">
          Job status is currently <span className="text-warning capitalize font-semibold">{viewedJob.status}</span>. Waiting for completion...
        </div>
      );
    }

    const results = viewedJob.results || {};

    if (activeVisualizerTab === "counts") {
      const counts = results.counts || results.hex_counts || {};
      const keys = Object.keys(counts);
      if (keys.length === 0) {
        return <div className="text-zinc-500 text-sm">No counts data available.</div>;
      }

      const maxVal = Math.max(...Object.values(counts) as number[]);
      return (
        <div className="w-full h-full flex flex-col justify-between pt-8 pb-6 px-4 gap-4 relative">
          <div className="flex-1 flex items-end justify-around gap-4 w-full h-56">
            {keys.map((k) => {
              const count = counts[k] as number;
              const percent = maxVal > 0 ? (count / maxVal) * 100 : 0;
              return (
                <div key={k} className="w-full max-w-[60px] flex flex-col items-center gap-2 group h-full justify-end">
                  <span className="font-mono text-zinc-400 opacity-0 group-hover:opacity-100 transition-opacity text-[10px]">
                    {count}
                  </span>
                  <div
                    className="w-full bg-indigo-500 hover:bg-indigo-400 transition-colors rounded-t-sm"
                    style={{ height: `${percent * 0.8}%` }}
                  />
                  <span className="font-mono text-white text-xs mt-2">{k}</span>
                </div>
              );
            })}
          </div>
          <div
            className="absolute inset-0 pointer-events-none"
            style={{
              backgroundImage: "linear-gradient(to bottom, #27272a 1px, transparent 1px)",
              backgroundSize: "100% 20%",
              opacity: 0.1,
            }}
          />
        </div>
      );
    }

    if (activeVisualizerTab === "iq") {
      const memory = results.memory || [];
      if (memory.length === 0) {
        return (
          <div className="text-zinc-500 text-sm text-center">
            No IQ memory data available. <br />
            <span className="text-xs text-zinc-600 mt-1 block">
              (Must submit with Meas Level = 1)
            </span>
          </div>
        );
      }

      const mapVal = (v: number) => {
        const min = -0.5, max = 1.5;
        return ((v - min) / (max - min)) * 200;
      };

      // Plot first qubit points (limited to 200)
      const points: React.ReactNode[] = [];
      memory.forEach((shot: any, idx: number) => {
        if (idx > 200) return;
        const qPoint = shot[0]; // first qubit [I, Q]
        if (qPoint) {
          const cx = mapVal(qPoint[0]);
          const cy = 200 - mapVal(qPoint[1]);
          const color = qPoint[0] > 0.5 ? "#6366f1" : "#22c55e"; // color by cluster threshold
          points.push(
            <circle
              key={idx}
              cx={cx}
              cy={cy}
              r="3"
              fill={color}
              opacity="0.7"
            />
          );
        }
      });

      return (
        <div className="flex flex-col items-center justify-center">
          <svg viewBox="0 0 200 200" className="w-64 h-64 border border-zinc-800 rounded bg-zinc-950/50">
            <line x1="0" y1="100" x2="200" y2="100" stroke="#27272a" strokeDasharray="2" />
            <line x1="100" y1="0" x2="100" y2="200" stroke="#27272a" strokeDasharray="2" />
            {points}
          </svg>
          <div className="text-[10px] text-zinc-500 mt-2 font-mono">
            I component vs Q component scatter plot
          </div>
        </div>
      );
    }

    if (activeVisualizerTab === "trace") {
      const memory = results.memory || [];
      if (memory.length === 0) {
        return (
          <div className="text-zinc-500 text-sm text-center">
            No raw trace data available. <br />
            <span className="text-xs text-zinc-600 mt-1 block">
              (Must submit with Meas Level = 0)
            </span>
          </div>
        );
      }

      // Plot first qubit trace
      const qubitTrace = memory[0] || [];
      if (qubitTrace.length === 0) {
        return <div className="text-zinc-500 text-sm">No trace points.</div>;
      }

      let pathPoints = "";
      const maxLen = Math.min(qubitTrace.length, 100);
      for (let i = 0; i < maxLen; i++) {
        const val = qubitTrace[i][0]; // real component
        const x = (i / (maxLen - 1)) * 200;
        const y = 100 - val * 40; // scale factor
        pathPoints += `${i === 0 ? "M" : "L"} ${x} ${y} `;
      }

      return (
        <div className="w-full flex flex-col items-center px-4">
          <svg viewBox="0 0 200 100" className="w-full h-48 border border-zinc-800 rounded bg-zinc-950/50 px-2">
            <line x1="0" y1="50" x2="200" y2="50" stroke="#27272a" strokeDasharray="2" />
            <path d={pathPoints} fill="none" stroke="#6366f1" strokeWidth="1.5" />
          </svg>
          <div className="text-[10px] text-zinc-500 mt-2 font-mono">Voltage signal trace vs time</div>
        </div>
      );
    }
  };

  return (
    <div className="space-y-6 h-full flex flex-col">
      <div>
        <h1 className="text-3xl font-geist text-white">Jobs Console</h1>
        <p className="text-sm text-zinc-400 mt-1">Configure, write QASM, and execute circuits.</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 flex-1 min-h-[500px]">
        {/* Left submissions pane */}
        <div className="lg:col-span-5 bg-zinc-900 border border-zinc-800 rounded-lg p-6 flex flex-col gap-6 h-[650px] overflow-y-auto">
          <div>
            <label className="block text-xs font-semibold text-zinc-400 uppercase tracking-wider mb-2">
              Target QPU
            </label>
            <select
              value={targetQpu}
              onChange={(e) => setTargetQpu(e.target.value)}
              className="w-full bg-zinc-950 border border-zinc-800 text-white rounded px-3 py-2 focus:outline-none focus:border-indigo-500 transition-colors"
            >
              {activeQpus.length === 0 ? (
                <option value="">No online QPUs available</option>
              ) : (
                activeQpus.map((q) => (
                  <option key={q.id} value={q.id}>
                    {q.name}
                  </option>
                ))
              )}
            </select>
          </div>

          <div className="flex-1 flex flex-col min-h-[220px]">
            <div className="flex justify-between items-center bg-zinc-800/30 border border-zinc-800 border-b-0 rounded-t px-4 py-2">
              <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">
                OpenQASM 3.0
              </span>
              <button
                onClick={loadExample}
                className="text-zinc-500 hover:text-white text-xs transition-colors flex items-center gap-1 focus:outline-none"
              >
                <Copy className="w-3.5 h-3.5" /> Load Example
              </button>
            </div>
            <textarea
              value={qasmCode}
              onChange={(e) => setQasmCode(e.target.value)}
              className="flex-1 w-full bg-zinc-950 border border-zinc-800 rounded-b p-4 text-white font-mono text-xs focus:outline-none focus:border-indigo-500 transition-all resize-none"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-zinc-400 uppercase tracking-wider mb-2">
                Shots Count
              </label>
              <input
                type="number"
                value={shots}
                onChange={(e) => setShots(parseInt(e.target.value) || 1)}
                className="w-full bg-zinc-950 border border-zinc-800 text-white rounded px-3 py-2 focus:outline-none focus:border-indigo-500 transition-colors font-mono text-sm"
              />
            </div>
            <div>
              <label className="block text-xs font-semibold text-zinc-400 uppercase tracking-wider mb-2 flex justify-between">
                Meas Level
                <span className="text-indigo-400 font-medium">{getMeasLevelLabel(measLevel)}</span>
              </label>
              <input
                type="range"
                min="0"
                max="2"
                value={measLevel}
                onChange={(e) => setMeasLevel(parseInt(e.target.value))}
                className="w-full h-2 bg-zinc-800 rounded-lg appearance-none cursor-pointer mt-3"
              />
            </div>
          </div>

          <button
            onClick={handleExecute}
            disabled={executing || !targetQpu}
            className="w-full bg-white text-zinc-950 font-geist font-semibold py-3 rounded hover:opacity-90 transition-opacity flex justify-center items-center gap-2 disabled:opacity-50 focus:outline-none"
          >
            {executing ? (
              <>
                <Loader2 className="w-5 h-5 animate-spin" />
                Submitting...
              </>
            ) : (
              <>
                <Play className="w-5 h-5 fill-current" />
                Execute Job
              </>
            )}
          </button>
        </div>

        {/* Right results pane */}
        <div className="lg:col-span-7 bg-zinc-900 border border-zinc-800 rounded-lg p-6 flex flex-col gap-6 h-[650px] overflow-hidden">
          <div className="flex items-start justify-between border-b border-zinc-800 pb-4">
            <div>
              <div className="flex items-center gap-3 mb-1">
                <h3 className="text-xl font-bold font-geist text-white">
                  {viewedJob ? `#${viewedJob.id.substring(0, 8)}` : "Select or run a job"}
                </h3>
                {viewedJob && (
                  <div
                    className={`flex items-center gap-1.5 px-2 py-0.5 rounded-full border text-[10px] uppercase font-semibold ${
                      viewedJob.status === "completed"
                        ? "border-green-500/30 bg-green-500/10 text-green-400"
                        : viewedJob.status === "pending"
                        ? "border-yellow-500/30 bg-yellow-500/10 text-yellow-400"
                        : viewedJob.status === "running"
                        ? "border-indigo-500/30 bg-indigo-500/10 text-indigo-400"
                        : "border-red-500/30 bg-red-500/10 text-red-400"
                    }`}
                  >
                    {viewedJob.status}
                  </div>
                )}
              </div>
              <p className="text-xs text-zinc-400">
                {viewedJob
                  ? `Executed on ${qpus.find((q) => q.id === viewedJob.qpu_target)?.name || viewedJob.qpu_target}`
                  : "No active job selected"}
              </p>
            </div>
            {viewedJob && (
              <div className="flex gap-4 text-right">
                <div>
                  <span className="block text-[10px] font-semibold text-zinc-500 uppercase tracking-wider">
                    Duration
                  </span>
                  <span className="font-mono text-sm text-white font-semibold">
                    {viewedJob.results?.duration !== undefined ? `${viewedJob.results.duration}ms` : "--"}
                  </span>
                </div>
                <div>
                  <span className="block text-[10px] font-semibold text-zinc-500 uppercase tracking-wider">
                    Created
                  </span>
                  <span className="font-mono text-sm text-white font-semibold">
                    {new Date(viewedJob.created).toLocaleTimeString()}
                  </span>
                </div>
              </div>
            )}
          </div>

          {/* Tab view options */}
          <div className="flex border-b border-zinc-800">
            <button
              onClick={() => setActiveVisualizerTab("counts")}
              className={`px-4 py-2 font-geist text-sm transition-all -mb-[1px] ${
                activeVisualizerTab === "counts"
                  ? "text-white border-b-2 border-white"
                  : "text-zinc-500 hover:text-zinc-300"
              }`}
            >
              Counts Histogram
            </button>
            <button
              onClick={() => setActiveVisualizerTab("iq")}
              className={`px-4 py-2 font-geist text-sm transition-all -mb-[1px] ${
                activeVisualizerTab === "iq"
                  ? "text-white border-b-2 border-white"
                  : "text-zinc-500 hover:text-zinc-300"
              }`}
            >
              IQ Clusters
            </button>
            <button
              onClick={() => setActiveVisualizerTab("trace")}
              className={`px-4 py-2 font-geist text-sm transition-all -mb-[1px] ${
                activeVisualizerTab === "trace"
                  ? "text-white border-b-2 border-white"
                  : "text-zinc-500 hover:text-zinc-300"
              }`}
            >
              Raw Trace
            </button>
          </div>

          {/* Chart body */}
          <div className="flex-1 bg-zinc-950 border border-zinc-800 rounded p-6 flex flex-col items-center justify-center relative overflow-hidden">
            {renderChart()}
          </div>
        </div>
      </div>
    </div>
  );
};
