import { useState, useRef, useEffect } from "react";
import { Bell, X, Info, AlertTriangle, Settings, LogOut } from "lucide-react";
import type { Notification } from "../types";
import { ThemeToggle } from "./ThemeToggle";

interface HeaderProps {
  pageTitle: string;
  userEmail: string;
  notifications: Notification[];
  onDismissNotification: (id: string) => void;
  onDismissAllNotifications: () => void;
  onLogout: () => void;
  onGoToSettings: () => void;
}

export const Header: React.FC<HeaderProps> = ({
  pageTitle,
  userEmail,
  notifications,
  onDismissNotification,
  onDismissAllNotifications,
  onLogout,
  onGoToSettings,
}) => {
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const [profileDropdownOpen, setProfileDropdownOpen] = useState(false);
  const profileDropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown on click outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setDropdownOpen(false);
      }
      if (
        profileDropdownRef.current &&
        !profileDropdownRef.current.contains(event.target as Node)
      ) {
        setProfileDropdownOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const getInitials = (email: string) => {
    return email ? email.charAt(0).toUpperCase() : "U";
  };

  return (
    <header className="h-16 border-b border-gray-200 dark:border-zinc-800 flex items-center justify-between px-8 bg-gray-50 dark:bg-zinc-950/20 z-40">
      <h2 className="font-geist text-lg font-bold text-gray-900 dark:text-white capitalize">
        {pageTitle === "qpus"
          ? "QPU Registry"
          : pageTitle === "jobs"
            ? "Jobs Console"
            : `${pageTitle} Overview`}
      </h2>
      <div className="flex items-center gap-4">
        {/* Notification Bell Dropdown */}
        <div className="relative" ref={dropdownRef}>
          <button
            aria-label="Notifications"
            onClick={() => setDropdownOpen(!dropdownOpen)}
            className="w-9 h-9 rounded-full bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 hover:bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400 hover:text-gray-900 dark:text-white transition-all flex items-center justify-center relative focus:outline-none"
          >
            <Bell className="w-5 h-5" />
            {notifications.length > 0 && (
              <span className="absolute -top-1.5 -right-1.5 bg-red-500 text-zinc-950 text-[10px] font-bold h-4 w-4 rounded-full flex items-center justify-center">
                {notifications.length}
              </span>
            )}
          </button>

          {/* Dropdown panel */}
          {dropdownOpen && (
            <div className="absolute right-0 mt-2 w-80 bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 rounded-lg shadow-2xl p-4 space-y-3 z-[100]">
              <div className="flex justify-between items-center border-b border-gray-200 dark:border-zinc-800 pb-2">
                <span className="text-xs font-semibold text-gray-900 dark:text-white uppercase tracking-wider">
                  Announcements
                </span>
                {notifications.length > 0 && (
                  <button
                    onClick={() => {
                      onDismissAllNotifications();
                      setDropdownOpen(false);
                    }}
                    className="text-[10px] text-gray-400 dark:text-zinc-500 hover:text-gray-900 dark:text-white transition-colors focus:outline-none"
                  >
                    Clear All
                  </button>
                )}
              </div>
              <div className="max-h-60 overflow-y-auto space-y-2 text-xs">
                {notifications.length === 0 ? (
                  <div className="text-gray-400 dark:text-zinc-500 text-center py-4">
                    No new notifications
                  </div>
                ) : (
                  notifications.map((ann) => {
                    const isFail =
                      ann.title.toLowerCase().includes("fail") ||
                      ann.title.toLowerCase().includes("error");
                    return (
                      <div
                        key={ann.id}
                        className={`border p-3 rounded flex justify-between items-start gap-4 transition-colors relative group ${
                          isFail
                            ? "bg-red-500/10 border-red-500/20 text-red-200"
                            : "bg-white dark:bg-zinc-900 border-gray-200 dark:border-zinc-800 text-gray-600 dark:text-zinc-300"
                        }`}
                      >
                        <div className="flex items-start gap-2.5">
                          {isFail ? (
                            <AlertTriangle className="w-4 h-4 text-red-400 mt-0.5" />
                          ) : (
                            <Info className="w-4 h-4 text-gray-500 dark:text-zinc-400 mt-0.5" />
                          )}
                          <div>
                            <p className="font-semibold text-xs text-gray-900 dark:text-white">
                              {ann.title}
                            </p>
                            <p className="text-[10px] text-gray-500 dark:text-zinc-400 mt-1">
                              {ann.description}
                            </p>
                          </div>
                        </div>
                        <button
                          onClick={() => onDismissNotification(ann.id)}
                          className="text-gray-400 dark:text-zinc-500 hover:text-gray-900 dark:text-white transition-colors focus:outline-none"
                        >
                          <X className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    );
                  })
                )}
              </div>
            </div>
          )}
        </div>

        {/* Theme Toggle */}
        <ThemeToggle />

        {/* User menu */}
        <div
          className="relative flex items-center gap-3"
          ref={profileDropdownRef}
        >
          <span className="text-xs text-gray-500 dark:text-zinc-400 font-medium">
            {userEmail}
          </span>
          <button
            data-testid="user-avatar"
            onClick={() => setProfileDropdownOpen(!profileDropdownOpen)}
            className="w-8 h-8 rounded-full bg-gray-100 dark:bg-zinc-800 border border-gray-300 dark:border-zinc-700 flex items-center justify-center text-gray-900 dark:text-white font-semibold uppercase text-xs hover:border-gray-400 dark:hover:border-zinc-500 transition-colors focus:outline-none"
          >
            {getInitials(userEmail)}
          </button>

          {profileDropdownOpen && (
            <div className="absolute right-0 top-12 mt-2 w-48 bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 rounded-lg shadow-2xl p-2 space-y-1 z-[100]">
              <button
                onClick={() => {
                  setProfileDropdownOpen(false);
                  onGoToSettings();
                }}
                className="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-600 dark:text-zinc-300 hover:text-gray-900 dark:text-white hover:bg-gray-100 dark:bg-zinc-800 rounded transition-colors focus:outline-none"
              >
                <Settings className="w-4 h-4" />
                Settings
              </button>
              <button
                onClick={() => {
                  setProfileDropdownOpen(false);
                  onLogout();
                }}
                className="w-full flex items-center gap-2 px-3 py-2 text-sm text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded transition-colors focus:outline-none"
              >
                <LogOut className="w-4 h-4" />
                Sign Out
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
};
