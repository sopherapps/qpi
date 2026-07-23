import { useState, useEffect } from "react";
import { pb } from "@/lib/pb";
import type { ThemeRecord } from "@/types";
import { ThemeEditorModal } from "./ThemeEditorModal";

export function AppearanceInnerTab() {
  const [themes, setThemes] = useState<ThemeRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [editingTheme, setEditingTheme] = useState<ThemeRecord | "new" | null>(
    null,
  );

  const fetchThemes = async () => {
    setLoading(true);
    try {
      const records = await pb
        .collection("themes")
        .getFullList<ThemeRecord>({ sort: "-created" });
      setThemes(records);
    } catch (err: unknown) {
      console.error("Failed to fetch themes", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    fetchThemes();
  }, []);

  const handleActivate = async (id: string) => {
    if (
      !confirm(
        "Activating this theme will change the dashboard appearance for all users. Continue?",
      )
    )
      return;
    try {
      await pb.collection("themes").update(id, { is_active: true });
      await fetchThemes();
      // Optionally reload the page so ThemeContext re-fetches active theme immediately
      window.location.reload();
    } catch (err: unknown) {
      alert(`Activation failed: ${(err as Error).message}`);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this theme?")) return;
    try {
      await pb.collection("themes").delete(id);
      await fetchThemes();
    } catch (err: unknown) {
      alert(`Delete failed: ${(err as Error).message}`);
    }
  };

  const closeEditor = (refresh: boolean) => {
    setEditingTheme(null);
    if (refresh) fetchThemes();
  };

  if (loading) {
    return (
      <div className="text-gray-500 dark:text-zinc-400 text-sm">Loading...</div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white font-geist">
          Dashboard Themes
        </h3>
        <button
          onClick={() => setEditingTheme("new")}
          className="bg-white text-zinc-950 font-semibold py-2 px-4 rounded hover:opacity-90 transition-opacity focus:outline-none text-sm"
        >
          Create New Theme
        </button>
      </div>

      <div className="bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 rounded-lg overflow-hidden">
        <table className="w-full text-sm text-left">
          <thead className="bg-gray-50 dark:bg-zinc-950 border-b border-gray-200 dark:border-zinc-800 text-xs uppercase text-gray-500 dark:text-zinc-400">
            <tr>
              <th className="px-6 py-3 font-medium tracking-wider">Name</th>
              <th className="px-6 py-3 font-medium tracking-wider w-32 text-center">
                Active
              </th>
              <th className="px-6 py-3 font-medium tracking-wider text-right w-64">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-zinc-800">
            {themes.length === 0 ? (
              <tr>
                <td
                  colSpan={3}
                  className="px-6 py-8 text-center text-gray-500 dark:text-zinc-400 italic"
                >
                  No themes found.
                </td>
              </tr>
            ) : (
              themes.map((t) => (
                <tr
                  key={t.id}
                  className="hover:bg-gray-50/50 dark:hover:bg-zinc-800/50 transition-colors"
                >
                  <td className="px-6 py-4 text-gray-900 dark:text-white font-medium">
                    {t.name}
                  </td>
                  <td className="px-6 py-4 text-center">
                    {t.is_active && (
                      <span className="inline-block bg-green-500/10 text-green-500 border border-green-500/20 px-2 py-0.5 rounded text-xs font-semibold">
                        ACTIVE
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4 text-right space-x-3">
                    <button
                      onClick={() => setEditingTheme(t)}
                      className="text-gray-500 dark:text-zinc-400 hover:text-gray-900 dark:hover:text-white transition-colors"
                    >
                      Edit
                    </button>
                    {!t.is_active ? (
                      <>
                        <button
                          onClick={() => handleActivate(t.id)}
                          className="text-blue-500 hover:text-blue-400 transition-colors"
                        >
                          Activate
                        </button>
                        <button
                          onClick={() => handleDelete(t.id)}
                          className="text-red-500 hover:text-red-400 transition-colors"
                        >
                          Delete
                        </button>
                      </>
                    ) : (
                      <span className="text-gray-400 dark:text-zinc-600 italic px-2">
                        Deactivate or Delete disabled
                      </span>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {editingTheme && (
        <ThemeEditorModal
          theme={editingTheme === "new" ? null : editingTheme}
          onClose={closeEditor}
        />
      )}
    </div>
  );
}
