import { useState, useRef, useEffect } from "react";
import { Bell, X, Info, AlertTriangle } from "lucide-react";
import type { Notification } from "../types";

interface HeaderProps {
  pageTitle: string;
  userEmail: string;
  notifications: Notification[];
  onDismissNotification: (id: string) => void;
  onDismissAllNotifications: () => void;
}

export const Header: React.FC<HeaderProps> = ({
  pageTitle,
  userEmail,
  notifications,
  onDismissNotification,
  onDismissAllNotifications,
}) => {
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown on click outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setDropdownOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const getInitials = (email: string) => {
    return email ? email.charAt(0).toUpperCase() : "U";
  };

  return (
    <header className="h-16 border-b border-zinc-800 flex items-center justify-between px-8 bg-zinc-950/20 z-40">
      <h2 className="font-geist text-lg font-bold text-white capitalize">
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
            onClick={() => setDropdownOpen(!dropdownOpen)}
            className="w-9 h-9 rounded-full bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 text-zinc-400 hover:text-white transition-all flex items-center justify-center relative focus:outline-none"
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
            <div className="absolute right-0 mt-2 w-80 bg-zinc-900 border border-zinc-800 rounded-lg shadow-2xl p-4 space-y-3 z-[100]">
              <div className="flex justify-between items-center border-b border-zinc-800 pb-2">
                <span className="text-xs font-semibold text-white uppercase tracking-wider">
                  Announcements
                </span>
                {notifications.length > 0 && (
                  <button
                    onClick={() => {
                      onDismissAllNotifications();
                      setDropdownOpen(false);
                    }}
                    className="text-[10px] text-zinc-500 hover:text-white transition-colors focus:outline-none"
                  >
                    Clear All
                  </button>
                )}
              </div>
              <div className="max-h-60 overflow-y-auto space-y-2 text-xs">
                {notifications.length === 0 ? (
                  <div className="text-zinc-500 text-center py-4">
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
                            : "bg-zinc-900 border-zinc-800 text-zinc-300"
                        }`}
                      >
                        <div className="flex items-start gap-2.5">
                          {isFail ? (
                            <AlertTriangle className="w-4 h-4 text-red-400 mt-0.5" />
                          ) : (
                            <Info className="w-4 h-4 text-zinc-400 mt-0.5" />
                          )}
                          <div>
                            <p className="font-semibold text-xs text-white">
                              {ann.title}
                            </p>
                            <p className="text-[10px] text-zinc-400 mt-1">
                              {ann.description}
                            </p>
                          </div>
                        </div>
                        <button
                          onClick={() => onDismissNotification(ann.id)}
                          className="text-zinc-500 hover:text-white transition-colors focus:outline-none"
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

        {/* User menu */}
        <div className="flex items-center gap-3">
          <span className="text-xs text-zinc-400 font-medium">{userEmail}</span>
          <div className="w-8 h-8 rounded-full bg-zinc-800 border border-zinc-700 flex items-center justify-center text-white font-semibold uppercase text-xs">
            {getInitials(userEmail)}
          </div>
        </div>
      </div>
    </header>
  );
};
