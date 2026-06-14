import { LogOut } from "lucide-react";

interface Props {
  userId: string;
  email: string;
  qpuSeconds: number;
  isAdmin: boolean;
  onLogout: () => void;
}

function getInitials(emailStr: string) {
  return emailStr ? emailStr.charAt(0).toUpperCase() : "-";
}

export function ProfileCard({
  userId,
  email,
  qpuSeconds,
  isAdmin,
  onLogout,
}: Props) {
  return (
    <div className="max-w-xl bg-zinc-900 border border-zinc-800 rounded-lg p-6 space-y-6">
      <div className="flex items-center gap-4 border-b border-zinc-800 pb-6">
        <div className="w-16 h-16 rounded-full bg-zinc-800 border border-zinc-700 flex items-center justify-center text-white text-2xl font-semibold uppercase">
          {getInitials(email)}
        </div>
        <div>
          <h3 className="text-lg font-bold text-white font-geist">
            {email.split("@")[0]}
          </h3>
          <p className="text-sm text-zinc-400">{email}</p>
        </div>
      </div>

      <div className="space-y-4">
        <div className="flex justify-between items-center text-sm py-2 border-b border-zinc-800/50">
          <span className="text-zinc-400">Account ID</span>
          <span className="font-mono text-white font-medium">{userId}</span>
        </div>
        <div className="flex justify-between items-center text-sm py-2 border-b border-zinc-800/50">
          <span className="text-zinc-400">Account Type</span>
          <span className="text-white font-medium">
            {isAdmin ? "Administrator" : "Standard User"}
          </span>
        </div>
        <div className="flex justify-between items-center text-sm py-2 border-b border-zinc-800/50">
          <span className="text-zinc-400">Allocated QPU Seconds</span>
          <span className="font-mono text-white font-medium">
            {qpuSeconds}s
          </span>
        </div>
      </div>

      <button
        onClick={onLogout}
        className="w-full bg-red-500/10 border border-red-500/20 text-red-400 font-geist font-semibold py-2.5 rounded hover:bg-red-500 hover:text-zinc-950 transition-colors flex justify-center items-center gap-2 focus:outline-none"
      >
        <LogOut className="w-4.5 h-4.5" />
        Sign Out
      </button>
    </div>
  );
}
