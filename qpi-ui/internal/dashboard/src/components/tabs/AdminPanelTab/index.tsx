import { useCallback, useState } from "react";
import type { User as UserType, TimeRequest } from "@/types";
import { InnerTabBar } from "./elements/InnerTabBar";
import { UserAllocationsInnerTab } from "./elements/UserAllocationsInnerTab";
import { BroadcastAnnouncementInnerTab } from "./elements/BroadcastAnnouncementInnerTab";
import { TimeRequestsInnerTab } from "./elements/TimeRequestsInnerTab";

interface AdminPanelTabProps {
  users: UserType[];
  timeRequests: TimeRequest[];
  onAllocateTime: (userId: string, seconds: number) => Promise<void>;
  onBroadcastAnnouncement: (
    title: string,
    desc: string,
    start: string,
    end: string,
  ) => Promise<void>;
  onApproveRequest: (id: string) => Promise<void>;
  onRejectRequest: (id: string, reason: string) => Promise<void>;
}

type _TabValue = "users" | "announcements" | "requests";

export const AdminPanelTab: React.FC<AdminPanelTabProps> = ({
  users,
  timeRequests,
  onAllocateTime,
  onBroadcastAnnouncement,
  onApproveRequest,
  onRejectRequest,
}) => {
  const [subtab, setSubtab] = useState<_TabValue>("users");

  // Allocate time state
  const [allocateAmounts, setAllocateAmounts] = useState<
    Record<string, string>
  >({});

  const handleAllocate = async (userId: string) => {
    const amount = parseFloat(allocateAmounts[userId] || "");
    if (isNaN(amount) || amount <= 0) {
      alert("Please enter a valid positive number of seconds.");
      return;
    }

    try {
      await onAllocateTime(userId, amount);
      setAllocateAmounts((prev) => ({ ...prev, [userId]: "" }));
      alert("Quota updated successfully!");
    } catch (err: unknown) {
      alert(`Allocation failed: ${(err as Error).message}`);
    }
  };

  const handleApprove = async (id: string) => {
    try {
      await onApproveRequest(id);
    } catch (err: unknown) {
      alert(`Approval failed: ${(err as Error).message}`);
    }
  };

  const handleReject = async (id: string) => {
    const reason = prompt("Enter rejection reason:");
    if (reason === null) return; // cancelled
    try {
      await onRejectRequest(id, reason);
    } catch (err: unknown) {
      alert(`Rejection failed: ${(err as Error).message}`);
    }
  };

  const handleTabClick = useCallback(
    (value: string) => setSubtab(value as _TabValue),
    [setSubtab],
  );

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-geist text-white">Admin Panel</h1>
        <p className="text-sm text-zinc-400 mt-1">
          Superuser controls for quota management and broadcasts.
        </p>
      </div>

      <InnerTabBar onTabClick={handleTabClick} currentTab={subtab} />

      {subtab === "users" && (
        <UserAllocationsInnerTab
          users={users}
          allocateAmounts={allocateAmounts}
          onAmountChange={(userId, value) =>
            setAllocateAmounts((prev) => ({ ...prev, [userId]: value }))
          }
          onAllocate={handleAllocate}
        />
      )}

      {subtab === "announcements" && (
        <BroadcastAnnouncementInnerTab onBroadcast={onBroadcastAnnouncement} />
      )}

      {subtab === "requests" && (
        <TimeRequestsInnerTab
          timeRequests={timeRequests}
          onApprove={handleApprove}
          onReject={handleReject}
        />
      )}
    </div>
  );
};
