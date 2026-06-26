import { useState } from "react";
import { X } from "lucide-react";

interface Props {
  onClose: () => void;
  onBook: (startTime: string, endTime: string) => Promise<void>;
}

export function BookSlotModal({ onClose, onBook }: Props) {
  const [startTime, setStartTime] = useState("");
  const [endTime, setEndTime] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const dStart = new Date(startTime);
      const dEnd = new Date(endTime);

      if (dEnd <= dStart) {
        throw new Error("End time must be after start time");
      }

      // Convert browser local datetime-local format (YYYY-MM-DDTHH:MM) to UTC ISO format (YYYY-MM-DD HH:MM:SS.000Z)
      const startIso = dStart.toISOString();
      const endIso = dEnd.toISOString();

      await onBook(startIso, endIso);
      onClose();
      setStartTime("");
      setEndTime("");
    } catch (err: unknown) {
      const message =
        err instanceof Error
          ? `Booking failed: ${err.message}`
          : "Booking failed. Ensure end time is after start time and slots do not overlap.";
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-50 dark:bg-zinc-950/80 backdrop-blur-sm">
      <div className="w-full max-w-md bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 rounded-lg shadow-2xl p-6 space-y-4">
        <div className="flex justify-between items-center border-b border-gray-200 dark:border-zinc-800 pb-3">
          <h3 className="text-lg font-semibold font-geist text-gray-900 dark:text-white">
            Book Time Slot
          </h3>
          <button
            onClick={onClose}
            className="text-gray-500 dark:text-zinc-400 hover:text-gray-900 dark:text-white focus:outline-none"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4 text-sm">
          <div>
            <label className="block text-xs font-medium text-gray-500 dark:text-zinc-400 mb-1.5 uppercase tracking-wider">
              Start Time
            </label>
            <input
              type="datetime-local"
              required
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
              className="w-full bg-gray-50 dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded px-3 py-2 text-gray-900 dark:text-white focus:outline-none focus:border-zinc-500 transition-colors"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-500 dark:text-zinc-400 mb-1.5 uppercase tracking-wider">
              End Time
            </label>
            <input
              type="datetime-local"
              required
              value={endTime}
              onChange={(e) => setEndTime(e.target.value)}
              className="w-full bg-gray-50 dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded px-3 py-2 text-gray-900 dark:text-white focus:outline-none focus:border-zinc-500 transition-colors"
            />
          </div>

          {error && (
            <div className="text-xs text-error font-medium bg-error/10 border border-error/20 p-2.5 rounded">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-white text-zinc-950 font-semibold py-2.5 rounded hover:opacity-90 transition-opacity disabled:opacity-50"
          >
            {loading ? "Booking..." : "Schedule Slot"}
          </button>
        </form>
      </div>
    </div>
  );
}
