import React, { useState, useEffect } from "react";
import { pb } from "../lib/pb";
import { Lock } from "lucide-react";
import type { AuthMethodsList } from "pocketbase";

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
  const [authMethods, setAuthMethods] = useState<AuthMethodsList | null>(null);

  useEffect(() => {
    let isMounted = true;
    if (isOpen) {
      const fetchAuthMethods = async () => {
        try {
          const methods = await pb.collection("users").listAuthMethods();
          if (isMounted) {
            setAuthMethods(methods);
          }
        } catch (err) {
          console.error(err);
        }
      };
      fetchAuthMethods();
    }
    return () => {
      isMounted = false;
    };
  }, [isOpen]);

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
      setRole("user");
      setIdentity("");
      setPassword("");
      setError("");
    } catch {
      setError("Invalid credentials. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const handleOAuth2Login = async (providerName: string) => {
    setError("");
    setLoading(true);
    try {
      await pb.collection("users").authWithOAuth2({ provider: providerName });
      onLoginSuccess();
      setRole("user");
      setIdentity("");
      setPassword("");
      setError("");
    } catch (err) {
      console.error("OAuth2 error:", err);
      setError(`Failed to sign in with ${providerName}.`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      data-testid="login-modal"
      className="fixed inset-0 z-[100] flex items-center justify-center p-4 bg-gray-50 dark:bg-zinc-950/80 backdrop-blur-md"
    >
      <div className="w-full max-w-md bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 rounded-lg shadow-2xl p-8 space-y-6">
        <div className="text-center">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-gray-100 dark:bg-zinc-800 text-gray-900 dark:text-white mb-4 border border-gray-300 dark:border-zinc-700">
            <Lock className="w-6 h-6" />
          </div>
          <h2 className="text-2xl font-geist font-bold text-gray-900 dark:text-white">
            Sign in to QPI
          </h2>
          <p className="text-sm text-gray-500 dark:text-zinc-400 mt-1">
            Access your quantum computing environment
          </p>
        </div>

        {/* Role tabs */}
        <div className="flex border-b border-gray-200 dark:border-zinc-800">
          <button
            type="button"
            onClick={() => {
              setRole("user");
              setError("");
            }}
            className={`flex-1 pb-2 font-geist text-sm text-center border-b-2 font-medium focus:outline-none transition-all ${
              role === "user"
                ? "border-white text-gray-900 dark:text-white"
                : "border-transparent text-gray-500 dark:text-zinc-400 hover:text-gray-700 dark:text-zinc-200"
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
                ? "border-white text-gray-900 dark:text-white"
                : "border-transparent text-gray-500 dark:text-zinc-400 hover:text-gray-700 dark:text-zinc-200"
            }`}
          >
            Administrator
          </button>
        </div>

        {role === "user" &&
          authMethods?.oauth2?.enabled &&
          authMethods.oauth2.providers.length > 0 && (
            <div className="space-y-3">
              <div className="flex flex-col gap-2">
                {authMethods.oauth2.providers.map((provider) => (
                  <button
                    key={provider.name}
                    type="button"
                    disabled={loading}
                    onClick={() => handleOAuth2Login(provider.name)}
                    className="w-full bg-gray-100 dark:bg-zinc-800 hover:bg-gray-200 dark:hover:bg-zinc-700 text-gray-900 dark:text-white font-medium py-2 px-4 rounded transition-colors flex items-center justify-center gap-2 border border-gray-300 dark:border-zinc-700 disabled:opacity-50"
                  >
                    <span className="capitalize">
                      Continue with {provider.name}
                    </span>
                  </button>
                ))}
              </div>

              {(!authMethods || authMethods.password?.enabled !== false) && (
                <div className="relative pt-4 pb-2">
                  <div className="absolute inset-0 flex items-center pt-2">
                    <span className="w-full border-t border-gray-200 dark:border-zinc-800" />
                  </div>
                  <div className="relative flex justify-center text-xs uppercase">
                    <span className="bg-white dark:bg-zinc-900 px-2 text-gray-400 dark:text-zinc-500 font-medium tracking-wider">
                      Or use credentials
                    </span>
                  </div>
                </div>
              )}
            </div>
          )}

        {(!authMethods || role === "admin" || authMethods.password?.enabled !== false) && (
          <form onSubmit={handleLogin} className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-zinc-400 mb-1.5 uppercase tracking-wider">
                Email or Username
              </label>
              <input
                type="text"
                required
                value={identity}
                onChange={(e) => setIdentity(e.target.value)}
                className="w-full bg-gray-50 dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded px-3 py-2 text-gray-900 dark:text-white text-sm focus:outline-none focus:border-zinc-500 transition-colors"
                placeholder={
                  role === "admin" ? "admin@example.com" : "user@example.com"
                }
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-zinc-400 mb-1.5 uppercase tracking-wider">
                Password
              </label>
              <input
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full bg-gray-50 dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 rounded px-3 py-2 text-gray-900 dark:text-white text-sm focus:outline-none focus:border-zinc-500 transition-colors"
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
        )}
      </div>
    </div>
  );
};
