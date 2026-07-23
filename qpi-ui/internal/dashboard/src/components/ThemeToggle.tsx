import React from "react";
import { Sun, Moon } from "lucide-react";
import { useTheme } from "../lib/ThemeContext";

export const ThemeToggle: React.FC = () => {
  const { isDark, toggleMode } = useTheme();

  return (
    <button
      data-testid="theme-toggle"
      onClick={toggleMode}
      className="w-9 h-9 rounded-full bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 hover:bg-gray-100 dark:hover:bg-zinc-800 text-gray-500 hover:text-gray-900 dark:text-zinc-400 dark:hover:text-white transition-all flex items-center justify-center focus:outline-none"
      title={`Switch to ${isDark ? "Light" : "Dark"} Mode`}
    >
      {isDark ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
    </button>
  );
};
