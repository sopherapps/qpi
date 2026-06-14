import { AlertTriangle, Info, X } from "lucide-react";
import type { Notification } from "@/types";

interface Props {
  notifications: Notification[];
  onDismiss: (id: string) => void;
}

export function NotificationsPanel({ notifications, onDismiss }: Props) {
  return (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold text-white font-geist">
        System Announcements
      </h3>
      <div className="flex flex-col gap-3">
        {notifications.length === 0 ? (
          <div className="text-zinc-500 text-center py-8 bg-zinc-900/30 border border-zinc-800/50 rounded-lg">
            No active announcements
          </div>
        ) : (
          notifications.map((ann) => {
            const isFail =
              ann.title.toLowerCase().includes("fail") ||
              ann.title.toLowerCase().includes("error");
            return (
              <div
                key={ann.id}
                className={`border p-4 rounded flex justify-between items-start gap-4 hover:border-zinc-700 transition-colors relative group ${
                  isFail
                    ? "bg-red-500/10 border-red-500/20 text-red-200"
                    : "bg-zinc-900 border-zinc-800 text-zinc-300"
                }`}
              >
                <div className="flex items-start gap-3">
                  {isFail ? (
                    <AlertTriangle className="w-5 h-5 text-red-400 mt-0.5" />
                  ) : (
                    <Info className="w-5 h-5 text-zinc-400 mt-0.5" />
                  )}
                  <div>
                    <p className="font-medium text-xs text-white">
                      {ann.title}
                    </p>
                    <p className="text-[11px] text-zinc-400 mt-1">
                      {ann.description}
                    </p>
                  </div>
                </div>
                <button
                  onClick={() => onDismiss(ann.id)}
                  className="text-zinc-500 hover:text-white opacity-0 group-hover:opacity-100 transition-opacity focus:outline-none"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
