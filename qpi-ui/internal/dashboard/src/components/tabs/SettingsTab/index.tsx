import React from "react";
import { ProfileCard } from "./elements/ProfileCard";

interface SettingsTabProps {
  userId: string;
  email: string;
  qpuSeconds: number;
  isAdmin: boolean;
  onLogout: () => void;
}

export const SettingsTab: React.FC<SettingsTabProps> = ({
  userId,
  email,
  qpuSeconds,
  isAdmin,
  onLogout,
}) => {
  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-geist text-gray-900 dark:text-white">
          Profile Settings
        </h1>
        <p className="text-sm text-gray-500 dark:text-zinc-400 mt-1">
          Configure your personal QPI user parameters.
        </p>
      </div>

      <ProfileCard
        userId={userId}
        email={email}
        qpuSeconds={qpuSeconds}
        isAdmin={isAdmin}
        onLogout={onLogout}
      />
    </div>
  );
};
