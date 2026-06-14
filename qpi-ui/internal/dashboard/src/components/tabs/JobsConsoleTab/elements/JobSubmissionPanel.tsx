import React, { useState, useEffect } from "react";
import { Copy, Play, Loader2 } from "lucide-react";
import type { QPU } from "@/types";

interface Props {
  qpus: QPU[];
  onSubmitJob: (qpuId: string, qasm: string, shots: number, measLevel: number) => Promise<string>;
  onJobSubmitted: (jobId: string) => void;
}

export const JobSubmissionPanel: React.FC<Props> = ({ qpus, onSubmitJob, onJobSubmitted }) => {
  const activeQpus = qpus.filter((q) => q.status === "online");

  const [targetQpu, setTargetQpu] = useState("");
  const [qasmCode, setQasmCode] = useState(
    'OPENQASM 3.0;\ninclude "stdgates.inc";\nqubit[2] q;\nbit[2] c;\nh q[0];\ncx q[0], q[1];\nc = measure q;'
  );
  const [shots, setShots] = useState(1000);
  const [measLevel, setMeasLevel] = useState(2);
  const [executing, setExecuting] = useState(false);

  // Set default target QPU once loaded
  useEffect(() => {
    if (activeQpus.length > 0 && !targetQpu) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setTargetQpu(activeQpus[0].id);
    }
  }, [activeQpus, targetQpu]);

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
      onJobSubmitted(newJobId);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Job submission failed";
      alert(`Job submission failed: ${message}`);
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

  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-6 flex flex-col gap-6 h-[650px] overflow-y-auto">
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
  );
};
