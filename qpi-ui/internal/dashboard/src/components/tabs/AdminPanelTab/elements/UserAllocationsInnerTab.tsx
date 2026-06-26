import type { User as UserType } from "@/types";

interface Props {
  users: UserType[];
  allocateAmounts: Record<string, string>;
  onAmountChange: (userId: string, value: string) => void;
  onAllocate: (userId: string) => void;
}

export function UserAllocationsInnerTab({
  users,
  allocateAmounts,
  onAmountChange,
  onAllocate,
}: Props) {
  return (
    <div className="bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 rounded-lg overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="border-b border-gray-200 dark:border-zinc-800 text-gray-500 dark:text-zinc-400 text-xs font-semibold uppercase tracking-wider bg-white dark:bg-zinc-900/50">
              <th className="py-3 px-4">User ID / Username</th>
              <th className="py-3 px-4">Email</th>
              <th className="py-3 px-4">QPU Balance</th>
              <th className="py-3 px-4 text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="text-sm text-gray-600 dark:text-zinc-300 divide-y divide-zinc-800/50">
            {users.length === 0 ? (
              <tr>
                <td
                  colSpan={4}
                  className="py-8 px-4 text-center text-gray-400 dark:text-zinc-500"
                >
                  Loading registered users...
                </td>
              </tr>
            ) : (
              users.map((u) => (
                <tr
                  key={u.id}
                  className="hover:bg-gray-100 dark:bg-zinc-800/20 transition-colors"
                >
                  <td className="py-3.5 px-4 font-mono text-xs text-gray-900 dark:text-white">
                    {u.id}
                  </td>
                  <td className="py-3.5 px-4 text-gray-500 dark:text-zinc-400">
                    {u.email}
                  </td>
                  <td className="py-3.5 px-4 font-mono text-gray-600 dark:text-zinc-300">
                    {u.qpu_seconds}s
                  </td>
                  <td className="py-3.5 px-4 text-right">
                    <div className="inline-flex items-center gap-2">
                      <input
                        type="number"
                        placeholder="Add sec"
                        value={allocateAmounts[u.id] || ""}
                        onChange={(e) => onAmountChange(u.id, e.target.value)}
                        className="bg-gray-50 dark:bg-zinc-950 border border-gray-200 dark:border-zinc-800 text-gray-900 dark:text-white rounded px-2 py-1 text-xs w-24 focus:outline-none focus:border-zinc-500 font-mono"
                      />
                      <button
                        onClick={() => onAllocate(u.id)}
                        className="bg-white text-zinc-950 px-3 py-1 rounded text-xs font-semibold hover:opacity-90 transition-all focus:outline-none"
                      >
                        Grant
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
