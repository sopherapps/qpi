import React, { useState } from "react";
import { Plus } from "lucide-react";
import type {
  CreateDriverRequest,
  CreateDriverResponse,
  Driver,
  QPU,
} from "@/types";
import { DriverCard } from "./elements/DriverCard";
import { RegisterDriverModal } from "./elements/RegisterDriverModal";

interface DriversTabProps {
  drivers: Driver[];
  qpus: QPU[];
  isAdmin: boolean;
  onToggleDriver: (id: string, enabled: boolean) => Promise<void>;
  onRegisterDriver: (req: CreateDriverRequest) => Promise<CreateDriverResponse>;
  onDeleteDriver: (id: string) => Promise<void>;
}

export const DriversTab: React.FC<DriversTabProps> = ({
  drivers,
  qpus,
  isAdmin,
  onToggleDriver,
  onRegisterDriver,
  onDeleteDriver,
}) => {
  const [modalOpen, setModalOpen] = useState(false);

  return (
    <div className="space-y-8">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-geist text-gray-900 dark:text-white">
            Drivers
          </h1>
          <p className="text-sm text-gray-500 dark:text-zinc-400 mt-1">
            Register and manage the external driver processes that exchange
            events with your QPUs.
          </p>
        </div>
        {isAdmin && (
          <button
            onClick={() => setModalOpen(true)}
            className="w-full md:w-auto bg-white text-zinc-950 font-semibold py-2 px-6 rounded flex items-center justify-center gap-2 hover:opacity-90 transition-opacity focus:outline-none"
          >
            <Plus className="w-4.5 h-4.5" />
            Register Driver
          </button>
        )}
      </div>

      {drivers.length === 0 ? (
        <div className="text-sm text-gray-500 dark:text-zinc-400 border border-dashed border-gray-200 dark:border-zinc-800 rounded-lg p-8 text-center">
          No drivers registered yet.
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {drivers.map((driver) => (
            <DriverCard
              key={driver.id}
              driver={driver}
              isAdmin={isAdmin}
              onToggle={onToggleDriver}
              onDelete={onDeleteDriver}
            />
          ))}
        </div>
      )}

      {modalOpen && (
        <RegisterDriverModal
          qpus={qpus}
          onClose={() => setModalOpen(false)}
          onRegister={onRegisterDriver}
        />
      )}
    </div>
  );
};
