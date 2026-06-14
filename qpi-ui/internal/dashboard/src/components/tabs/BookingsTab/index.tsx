import React, { useState } from "react";
import { Calendar } from "lucide-react";
import type { TimeSlot } from "@/types";
import { ReservationsTable } from "./elements/ReservationsTable";
import { SchedulingPolicyCard } from "./elements/SchedulingPolicyCard";
import { BookSlotModal } from "./elements/BookSlotModal";

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

  const handleDelete = async (id: string) => {
    if (confirm("Are you sure you want to cancel this reservation?")) {
      try {
        await onCancelSlot(id);
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : "Failed to cancel";
        alert(`Failed to cancel: ${message}`);
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
          <ReservationsTable
            bookings={bookings}
            currentUserId={currentUserId}
            isAdmin={isAdmin}
            onDelete={handleDelete}
          />
        </div>

        {/* Right: Policy details */}
        <div className="lg:col-span-4 space-y-4">
          <SchedulingPolicyCard />
        </div>
      </div>

      {/* Book Slot Modal */}
      {modalOpen && (
        <BookSlotModal
          onClose={() => setModalOpen(false)}
          onBook={onBookSlot}
        />
      )}
    </div>
  );
};
