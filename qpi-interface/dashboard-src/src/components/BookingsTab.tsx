import React, { useState } from "react";
import { Calendar, Trash2, X } from "lucide-react";
import type { TimeSlot } from "../types";

interface BookingsTabProps {
  bookings: TimeSlot[];
  currentUserId: string;
  isAdmin: boolean;
  onBookSlot: (startTime: string, endTime: string) => Promise<void>;
  onCancelSlot: (id: string) => Promise<void>;
}

export const BookingsTab: React.FC<BookingsTabProps> = ({
  bookings,
  currentUserId,
  isAdmin,
  onBookSlot,
  onCancelSlot,
}) => {
  const [modalOpen, setModalOpen] = useState(false);
  const [startTime, setStartTime] = useState("");
  const [endTime, setEndTime] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      // Convert browser local datetime-local format (YYYY-MM-DDTHH:MM) to UTC ISO format (YYYY-MM-DD HH:MM:SS.000Z)
      const startIso = new Date(startTime).toISOString();
      const endIso = new Date(endTime).toISOString();

      await onBookSlot(startIso, endIso);
      setModalOpen(false);
      setStartTime("");
      setEndTime("");
    } catch (err: any) {
      setError(err?.message || "Booking failed. Ensure end time is after start time and slots do not overlap.");
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (confirm("Are you sure you want to cancel this reservation?")) {
      try {
        await onCancelSlot(id);
      } catch (err: any) {
        alert(`Failed to cancel: ${err.message}`);
      }
    }
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <h1 className="text-3xl font-geist text-white">Bookings</h1>
          <p className="text-sm text-zinc-400 mt-1">
            Reserve QPU execution blocks to run priority jobs.
          </p>
        </div>
        <button
          onClick={() => setModalOpen(true)}
          className="w-full md:w-auto bg-white text-zinc-950 font-semibold py-2 px-6 rounded flex items-center justify-center gap-2 hover:opacity-90 transition-opacity focus:outline-none"
        >
          <Calendar className="w-4.5 h-4.5" />
          Book Time Slot
        </button>
      </div>

      {/* Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left: Scheduled Reservations */}
        <div className="lg:col-span-8 space-y-4">
          <h3 className="text-lg font-semibold text-white font-geist">Scheduled Reservations</h3>
          <div className="bg-zinc-900 border border-zinc-800 rounded-lg overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b border-zinc-800 text-zinc-400 text-xs font-semibold uppercase tracking-wider bg-zinc-900/50">
                    <th className="py-3 px-4">Booked By</th>
                    <th className="py-3 px-4">Start Time</th>
                    <th className="py-3 px-4">End Time</th>
                    <th className="py-3 px-4 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="text-sm text-zinc-300 divide-y divide-zinc-800/50">
                  {bookings.length === 0 ? (
                    <tr>
                      <td colSpan={4} className="py-8 px-4 text-center text-zinc-500 font-medium">
                        No booking slots scheduled yet.
                      </td>
                    </tr>
                  ) : (
                    bookings.map((slot) => {
                      const isOwner = slot.booked_by === currentUserId;
                      const bookedByName = slot.expand?.booked_by?.email || slot.booked_by;
                      return (
                        <tr key={slot.id} className="hover:bg-zinc-800/20 transition-colors">
                          <td className="py-3.5 px-4 font-medium text-white">{bookedByName}</td>
                          <td className="py-3.5 px-4 text-zinc-400">
                            {new Date(slot.start_time).toLocaleString()}
                          </td>
                          <td className="py-3.5 px-4 text-zinc-400">
                            {new Date(slot.end_time).toLocaleString()}
                          </td>
                          <td className="py-3.5 px-4 text-right">
                            {(isOwner || isAdmin) ? (
                              <button
                                onClick={() => handleDelete(slot.id)}
                                className="text-red-400 hover:text-red-300 p-1.5 rounded hover:bg-red-500/10 transition-colors focus:outline-none"
                              >
                                <Trash2 className="w-4 h-4" />
                              </button>
                            ) : (
                              <span className="text-xs text-zinc-600 font-medium">Read-Only</span>
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
        </div>

        {/* Right: Policy details */}
        <div className="lg:col-span-4 space-y-4">
          <div className="bg-zinc-900 border border-zinc-800 p-6 rounded-lg space-y-4">
            <h3 className="text-base font-bold text-white font-geist">Scheduling Policy</h3>
            <div className="space-y-3 text-xs text-zinc-400 leading-relaxed">
              <p>
                <strong>Dedicated Window:</strong> When you book a slot, you have exclusive priority to
                submit jobs to the active QPUs.
              </p>
              <p>
                <strong>Opportunistic FIFO:</strong> If the slot owner is idle (has not submitted a job)
                for more than 5 seconds, other pending jobs from the queue will automatically run to
                optimize machine usage.
              </p>
              <p>
                <strong>Releasing Slots:</strong> You can cancel/delete your booked slots at any time
                prior to the start time.
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Book Slot Modal */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-lg shadow-2xl p-6 space-y-4">
            <div className="flex justify-between items-center border-b border-zinc-800 pb-3">
              <h3 className="text-lg font-semibold font-geist text-white">Book Time Slot</h3>
              <button
                onClick={() => setModalOpen(false)}
                className="text-zinc-400 hover:text-white focus:outline-none"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleSubmit} className="space-y-4 text-sm">
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1.5 uppercase tracking-wider">
                  Start Time
                </label>
                <input
                  type="datetime-local"
                  required
                  value={startTime}
                  onChange={(e) => setStartTime(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded px-3 py-2 text-white focus:outline-none focus:border-zinc-500 transition-colors"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1.5 uppercase tracking-wider">
                  End Time
                </label>
                <input
                  type="datetime-local"
                  required
                  value={endTime}
                  onChange={(e) => setEndTime(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded px-3 py-2 text-white focus:outline-none focus:border-zinc-500 transition-colors"
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
      )}
    </div>
  );
};
