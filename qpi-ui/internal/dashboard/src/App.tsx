import React, { useState, useEffect, useCallback } from "react";
import { pb } from "./lib/pb";
import { Sidebar } from "./components/Sidebar";
import { Header } from "./components/Header";
import { OverviewTab } from "./components/tabs/OverviewTab";
import { QpuRegistryTab } from "./components/tabs/QpuRegistryTab";
import { JobsConsoleTab } from "./components/tabs/JobsConsoleTab";
import { BookingsTab } from "./components/tabs/BookingsTab";
import { AdminPanelTab } from "./components/tabs/AdminPanelTab";
import { SettingsTab } from "./components/tabs/SettingsTab";
import { LoginModal } from "./components/LoginModal";
import { RequestTimeModal } from "./components/RequestTimeModal";
import type {
  QPU,
  QuantumJob,
  Notification,
  TimeSlot,
  User,
  TimeRequest,
} from "./types";

export const App: React.FC = () => {
  const [authValid, setAuthValid] = useState(pb.authStore.isValid);
  const [userEmail, setUserEmail] = useState("");
  const [userId, setUserId] = useState("");
  const [isAdmin, setIsAdmin] = useState(false);
  const [qpuSeconds, setQpuSeconds] = useState(0);

  // Tab routing
  const [activeTab, setActiveTab] = useState("overview");
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);

  // Data collections
  const [qpus, setQpus] = useState<QPU[]>([]);
  const [jobs, setJobs] = useState<QuantumJob[]>([]);
  const [bookings, setBookings] = useState<TimeSlot[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [usersList, setUsersList] = useState<User[]>([]);
  const [timeRequests, setTimeRequests] = useState<TimeRequest[]>([]);

  // Modal overlays
  const [isRequestTimeOpen, setIsRequestTimeOpen] = useState(false);

  // Synced tab hash router
  useEffect(() => {
    const handleHashChange = () => {
      const hash = window.location.hash.replace("#", "");
      if (
        ["overview", "qpus", "jobs", "bookings", "admin", "settings"].includes(
          hash,
        )
      ) {
        setActiveTab(hash);
      }
    };
    window.addEventListener("hashchange", handleHashChange);
    handleHashChange(); // initial routing
    return () => window.removeEventListener("hashchange", handleHashChange);
  }, []);

  const changeTab = (tab: string) => {
    setActiveTab(tab);
    window.location.hash = tab;
  };

  // Data Loading Helpers
  const loadUserQuota = useCallback(async () => {
    if (!pb.authStore.isValid || !pb.authStore.model) return;
    const isSuper = pb.authStore.model.collectionName === "_superusers";
    if (isSuper) {
      setQpuSeconds(999999);
      return;
    }
    try {
      const user = await pb.collection("users").getOne(pb.authStore.model.id);
      setQpuSeconds((user as unknown as User).qpu_seconds || 0);
    } catch (err) {
      console.error("Failed to load user quota:", err);
    }
  }, []);

  const loadQpus = useCallback(async () => {
    try {
      const records = await pb
        .collection("qpus")
        .getFullList({ sort: "+name" });
      setQpus(records as unknown as QPU[]);
    } catch (err) {
      console.error("Failed to load QPUs:", err);
    }
  }, []);

  const loadJobs = useCallback(async () => {
    try {
      const records = await pb.collection("quantum_jobs").getFullList({
        sort: "-created",
      });
      setJobs(records as unknown as QuantumJob[]);
    } catch (err) {
      console.error("Failed to load quantum jobs:", err);
    }
  }, []);

  const loadBookings = useCallback(async () => {
    try {
      const records = await pb.collection("time_slots").getFullList({
        sort: "start_time",
        expand: "booked_by",
      });
      setBookings(records as unknown as TimeSlot[]);
    } catch (err) {
      console.error("Failed to load bookings:", err);
    }
  }, []);

  const loadNotifications = useCallback(async () => {
    try {
      const records = await pb.collection("notifications").getFullList({
        sort: "-created",
      });
      setNotifications(records as unknown as Notification[]);
    } catch (err) {
      console.error("Failed to load notifications:", err);
    }
  }, []);

  const loadAdminUsers = useCallback(async () => {
    try {
      const records = await pb
        .collection("users")
        .getFullList({ sort: "+email" });
      setUsersList(records as unknown as User[]);
    } catch (err) {
      console.error("Failed to load users for admin:", err);
    }
  }, []);

  const loadTimeRequests = useCallback(async () => {
    try {
      const records = await pb.collection("time_requests").getFullList({
        sort: "-created",
        expand: "user",
      });
      setTimeRequests(records as unknown as TimeRequest[]);
    } catch (err) {
      console.error("Failed to load time requests:", err);
    }
  }, []);

  // Main data loader
  const loadAllData = useCallback(async () => {
    if (!pb.authStore.isValid || !pb.authStore.model) return;
    const isSuper = pb.authStore.model.collectionName === "_superusers";

    await Promise.all([
      loadUserQuota(),
      loadQpus(),
      loadJobs(),
      loadBookings(),
      loadNotifications(),
    ]);

    if (isSuper) {
      await Promise.all([loadAdminUsers(), loadTimeRequests()]);
    }
  }, [
    loadUserQuota,
    loadQpus,
    loadJobs,
    loadBookings,
    loadNotifications,
    loadAdminUsers,
    loadTimeRequests,
  ]);

  // Real-time Subscriptions setup
  useEffect(() => {
    if (!authValid) return;

    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadAllData();

    // Subscribe to database changes
    pb.collection("notifications").subscribe("*", () => {
      loadNotifications();
    });
    pb.collection("quantum_jobs").subscribe("*", () => {
      loadJobs();
      loadUserQuota();
    });
    pb.collection("time_slots").subscribe("*", () => {
      loadBookings();
    });
    pb.collection("qpus").subscribe("*", () => {
      loadQpus();
    });

    const isSuper = pb.authStore.model?.collectionName === "_superusers";
    if (isSuper) {
      pb.collection("users").subscribe("*", () => {
        loadAdminUsers();
      });
      pb.collection("time_requests").subscribe("*", () => {
        loadTimeRequests();
      });
    }

    return () => {
      pb.collection("notifications").unsubscribe();
      pb.collection("quantum_jobs").unsubscribe();
      pb.collection("time_slots").unsubscribe();
      pb.collection("qpus").unsubscribe();
      if (isSuper) {
        pb.collection("users").unsubscribe();
        pb.collection("time_requests").unsubscribe();
      }
    };
  }, [
    authValid,
    loadAllData,
    loadNotifications,
    loadJobs,
    loadUserQuota,
    loadBookings,
    loadQpus,
    loadAdminUsers,
    loadTimeRequests,
  ]);

  // Initialize auth credentials from session storage
  useEffect(() => {
    if (pb.authStore.isValid && pb.authStore.model) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setUserEmail(pb.authStore.model.email || "admin@qpi.org");
      setUserId(pb.authStore.model.id);
      setIsAdmin(pb.authStore.model.collectionName === "_superusers");
      setAuthValid(true);
    }
  }, []);

  const handleLoginSuccess = () => {
    if (pb.authStore.isValid && pb.authStore.model) {
      setUserEmail(pb.authStore.model.email || "admin@qpi.org");
      setUserId(pb.authStore.model.id);
      setIsAdmin(pb.authStore.model.collectionName === "_superusers");
      setAuthValid(true);
    }
  };

  const handleLogout = () => {
    pb.authStore.clear();
    setUserEmail("");
    setUserId("");
    setIsAdmin(false);
    setAuthValid(false);
    setSelectedJobId(null);
    setQpus([]);
    setJobs([]);
    setBookings([]);
    setNotifications([]);
    setUsersList([]);
    setTimeRequests([]);
  };

  // Actions & Custom Endpoints handlers
  const handleDismissNotification = async (id: string) => {
    try {
      await pb.send(`/api/notifications/${encodeURIComponent(id)}/dismiss`, {
        method: "POST",
      });
      loadNotifications();
    } catch (err: unknown) {
      alert(`Dismiss failed: ${(err as Error).message}`);
    }
  };

  const handleDismissAllNotifications = async () => {
    try {
      for (const ann of notifications) {
        await pb.send(
          `/api/notifications/${encodeURIComponent(ann.id)}/dismiss`,
          {
            method: "POST",
          },
        );
      }
      loadNotifications();
    } catch (err: unknown) {
      alert(`Clear failed: ${(err as Error).message}`);
    }
  };

  const handleToggleQpu = async (id: string, enabled: boolean) => {
    await pb.send("/api/op/qpu/toggle", {
      method: "POST",
      body: JSON.stringify({ id: id, enabled: enabled }),
      headers: { "Content-Type": "application/json" },
    });
    loadQpus();
  };

  const handleCreateQpu = async (
    name: string,
    executor: string,
  ) => {
    const res = await pb.send<{ access_token: string }>("/api/op/qpus/create", {
      method: "POST",
      body: JSON.stringify({
        name: name,
        executor_type: executor,
      }),
      headers: { "Content-Type": "application/json" },
    });
    if (res && res.access_token) {
      alert(`QPU created successfully!\n\nHere is your QPU Access Token (copy this, it will NOT be shown again):\n\n${res.access_token}`);
    } else {
      alert("QPU created successfully!");
    }
    loadQpus();
  };

  const handleSubmitQuantumJob = async (
    qpuId: string,
    qasm: string,
    shots: number,
    measLevel: number,
  ) => {
    const res = await pb.send("/api/jobs", {
      method: "POST",
      body: JSON.stringify({
        circuits: [{ circuit: qasm }],
        shots: shots,
        meas_level: measLevel,
        meas_return: "single",
        qpu_target: qpuId,
      }),
      headers: { "Content-Type": "application/json" },
    });
    loadJobs();
    loadUserQuota();
    return res.job_id;
  };

  const handleBookSlot = async (startTime: string, endTime: string) => {
    await pb.collection("time_slots").create({
      start_time: startTime,
      end_time: endTime,
      booked_by: userId,
    });
    loadBookings();
  };

  const handleCancelSlot = async (id: string) => {
    await pb.collection("time_slots").delete(id);
    loadBookings();
  };

  const handleRequestTime = async (seconds: number, reason: string) => {
    await pb.collection("time_requests").create({
      user: userId,
      seconds: seconds,
      reason: reason,
      status: "pending",
    });
    if (isAdmin) loadTimeRequests();
  };

  const handleAllocateTime = async (targetUserId: string, seconds: number) => {
    await pb.send(`/api/admin/users/${encodeURIComponent(targetUserId)}`, {
      method: "PATCH",
      body: JSON.stringify({ qpu_seconds: seconds }),
      headers: { "Content-Type": "application/json" },
    });
    loadAdminUsers();
  };

  const handleBroadcastAnnouncement = async (
    title: string,
    desc: string,
    start: string,
    end: string,
  ) => {
    await pb.collection("notifications").create({
      title: title,
      description: desc,
      start_time: start || null,
      end_time: end || null,
    });
    loadNotifications();
  };

  const handleApproveRequest = async (id: string) => {
    await pb.collection("time_requests").update(id, {
      status: "approved",
    });
    loadTimeRequests();
    loadAdminUsers();
  };

  const handleRejectRequest = async (id: string, reason: string) => {
    await pb.collection("time_requests").update(id, {
      status: "rejected",
      rejection_reason: reason,
    });
    loadTimeRequests();
  };

  const renderActiveTab = () => {
    switch (activeTab) {
      case "qpus":
        return (
          <QpuRegistryTab
            qpus={qpus}
            isAdmin={isAdmin}
            onToggleQpu={handleToggleQpu}
            onRegisterQpu={handleCreateQpu}
          />
        );
      case "jobs":
        return (
          <JobsConsoleTab
            qpus={qpus}
            selectedJobId={selectedJobId}
            setSelectedJobId={setSelectedJobId}
            onSubmitJob={handleSubmitQuantumJob}
          />
        );
      case "bookings":
        return (
          <BookingsTab
            bookings={bookings}
            currentUserId={userId}
            isAdmin={isAdmin}
            onBookSlot={handleBookSlot}
            onCancelSlot={handleCancelSlot}
          />
        );
      case "admin":
        return isAdmin ? (
          <AdminPanelTab
            users={usersList}
            timeRequests={timeRequests}
            onAllocateTime={handleAllocateTime}
            onBroadcastAnnouncement={handleBroadcastAnnouncement}
            onApproveRequest={handleApproveRequest}
            onRejectRequest={handleRejectRequest}
          />
        ) : (
          <div className="text-zinc-500">Access Denied.</div>
        );
      case "settings":
        return (
          <SettingsTab
            userId={userId}
            email={userEmail}
            qpuSeconds={qpuSeconds}
            isAdmin={isAdmin}
            onLogout={handleLogout}
          />
        );
      case "overview":
      default:
        return (
          <OverviewTab
            qpus={qpus}
            jobs={jobs}
            qpuSeconds={qpuSeconds}
            bookings={bookings}
            notifications={notifications}
            onDismissNotification={handleDismissNotification}
            switchTab={(tab) => {
              changeTab(tab);
              if (tab !== "jobs") setSelectedJobId(null);
            }}
          />
        );
    }
  };

  return (
    <>
      <LoginModal isOpen={!authValid} onLoginSuccess={handleLoginSuccess} />

      {authValid && (
        <div className="flex w-full h-full">
          <Sidebar
            activeTab={activeTab}
            setActiveTab={changeTab}
            isAdmin={isAdmin}
            qpuSeconds={qpuSeconds}
            onRequestTimeClick={() => setIsRequestTimeOpen(true)}
          />

          <div className="flex-1 flex flex-col min-w-0 h-full">
            <Header
              pageTitle={activeTab}
              userEmail={userEmail}
              notifications={notifications}
              onDismissNotification={handleDismissNotification}
              onDismissAllNotifications={handleDismissAllNotifications}
            />

            <main className="flex-1 overflow-y-auto p-8 bg-background">
              <div className="max-w-7xl mx-auto">{renderActiveTab()}</div>
            </main>
          </div>
        </div>
      )}

      <RequestTimeModal
        isOpen={isRequestTimeOpen}
        onClose={() => setIsRequestTimeOpen(false)}
        onSubmit={handleRequestTime}
      />
    </>
  );
};
