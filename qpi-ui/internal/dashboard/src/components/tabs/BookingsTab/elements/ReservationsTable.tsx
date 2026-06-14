import { Trash2 } from "lucide-react";
import type { TimeSlot } from "@/types";

interface Props {
  bookings: TimeSlot[];
  currentUserId: string;
  isAdmin: boolean;
  onDelete: (id: string) => void;
}

export function ReservationsTable({
  bookings,
  currentUserId,
  isAdmin,
  onDelete,
}: Props) {
  return (
    <>
      <h3 className="text-lg font-semibold text-white font-geist">
        Scheduled Reservations
      </h3>
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
                  <td
                    colSpan={4}
                    className="py-8 px-4 text-center text-zinc-500 font-medium"
                  >
                    No booking slots scheduled yet.
                  </td>
                </tr>
              ) : (
                bookings.map((slot) => {
                  const isOwner = slot.booked_by === currentUserId;
                  const bookedByName =
                    slot.expand?.booked_by?.email || slot.booked_by;
                  return (
                    <tr
                      key={slot.id}
                      className="hover:bg-zinc-800/20 transition-colors"
                    >
                      <td className="py-3.5 px-4 font-medium text-white">
                        {bookedByName}
                      </td>
                      <td className="py-3.5 px-4 text-zinc-400">
                        {new Date(slot.start_time).toLocaleString()}
                      </td>
                      <td className="py-3.5 px-4 text-zinc-400">
                        {new Date(slot.end_time).toLocaleString()}
                      </td>
                      <td className="py-3.5 px-4 text-right">
                        {isOwner || isAdmin ? (
                          <button
                            onClick={() => onDelete(slot.id)}
                            className="text-red-400 hover:text-red-300 p-1.5 rounded hover:bg-red-500/10 transition-colors focus:outline-none"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        ) : (
                          <span className="text-xs text-zinc-600 font-medium">
                            Read-Only
                          </span>
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
    </>
  );
}
