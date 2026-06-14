import { useState } from "react";

interface Props {
  onBroadcast: (title: string, desc: string, start: string, end: string) => Promise<void>;
}

export function BroadcastAnnouncementInnerTab({ onBroadcast }: Props) {
  const [annTitle, setAnnTitle] = useState("");
  const [annDesc, setAnnDesc] = useState("");
  const [annStart, setAnnStart] = useState("");
  const [annEnd, setAnnEnd] = useState("");
  const [annLoading, setAnnLoading] = useState(false);
  const [annSuccess, setAnnSuccess] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setAnnLoading(true);
    setAnnSuccess(false);

    try {
      const startIso = annStart ? new Date(annStart).toISOString() : "";
      const endIso = annEnd ? new Date(annEnd).toISOString() : "";
      await onBroadcast(annTitle, annDesc, startIso, endIso);

      setAnnTitle("");
      setAnnDesc("");
      setAnnStart("");
      setAnnEnd("");
      setAnnSuccess(true);
      setTimeout(() => setAnnSuccess(false), 3000);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Broadcast failed.";
      alert(`Broadcast failed: ${message}`);
    } finally {
      setAnnLoading(false);
    }
  };

  return (
    <div className="max-w-xl bg-zinc-900 border border-zinc-800 rounded-lg p-6 space-y-4">
      <h3 className="text-lg font-semibold text-white font-geist">Compose Announcement</h3>
      <form onSubmit={handleSubmit} className="space-y-4 text-sm">
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
  );
}
