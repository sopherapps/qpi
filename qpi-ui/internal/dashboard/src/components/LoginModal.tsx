import React, { useState } from "react";
import { pb } from "../lib/pb";
import { Lock } from "lucide-react";

interface LoginModalProps {
  isOpen: boolean;
  onLoginSuccess: () => void;
}

export const LoginModal: React.FC<LoginModalProps> = ({
  isOpen,
  onLoginSuccess,
}) => {
  const [role, setRole] = useState<"user" | "admin">("user");
  const [identity, setIdentity] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  if (!isOpen) return null;

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      if (role === "admin") {
        await pb.collection("_superusers").authWithPassword(identity, password);
      } else {
        await pb.collection("users").authWithPassword(identity, password);
      }
      onLoginSuccess();
    } catch {
      setError("Invalid credentials. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-md">
      <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-lg shadow-2xl p-8 space-y-6">
        <div className="text-center">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-zinc-800 text-white mb-4 border border-zinc-700">
            <Lock className="w-6 h-6" />
          </div>
          <h2 className="text-2xl font-geist font-bold text-white">
            Sign in to QPI
          </h2>
          <p className="text-sm text-zinc-400 mt-1">
            Access your quantum computing environment
          </p>
        </div>

        {/* Role tabs */}
        <div className="flex border-b border-zinc-800">
          <button
            type="button"
            onClick={() => {
              setRole("user");
              setError("");
            }}
            className={`flex-1 pb-2 font-geist text-sm text-center border-b-2 font-medium focus:outline-none transition-all ${
              role === "user"
                ? "border-white text-white"
                : "border-transparent text-zinc-400 hover:text-zinc-200"
            }`}
          >
            Regular User
          </button>
          <button
            type="button"
            onClick={() => {
              setRole("admin");
              setError("");
            }}
            className={`flex-1 pb-2 font-geist text-sm text-center border-b-2 font-medium focus:outline-none transition-all ${
              role === "admin"
                ? "border-white text-white"
                : "border-transparent text-zinc-400 hover:text-zinc-200"
            }`}
          >
            Administrator
          </button>
        </div>

        <form onSubmit={handleLogin} className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-zinc-400 mb-1.5 uppercase tracking-wider">
              Email or Username
            </label>
            <input
              type="text"
              required
              value={identity}
              onChange={(e) => setIdentity(e.target.value)}
              className="w-full bg-zinc-950 border border-zinc-800 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-zinc-500 transition-colors"
              placeholder={
                role === "admin" ? "admin@example.com" : "user@example.com"
              }
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-zinc-400 mb-1.5 uppercase tracking-wider">
              Password
            </label>
            <input
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full bg-zinc-950 border border-zinc-800 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-zinc-500 transition-colors"
              placeholder="••••••••"
            />
          </div>

          {error && (
            <div className="text-xs text-error font-medium bg-error/10 border border-error/20 p-2.5 rounded">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-white text-zinc-950 font-geist font-semibold py-2.5 rounded hover:opacity-90 transition-opacity flex justify-center items-center gap-2 disabled:opacity-50"
          >
            {loading ? "Signing In..." : "Sign In"}
          </button>
        </form>
      </div>
    </div>
  );
};
