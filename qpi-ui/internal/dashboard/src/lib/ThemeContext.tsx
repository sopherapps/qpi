import React, { createContext, useContext, useEffect, useState } from "react";

import type { ThemeRecord } from "@/types";

interface ThemeContextValue {
  theme: ThemeRecord | null;
  siteName: string;
  tagline: string;
  logoUrl: string | null;
  faviconUrl: string | null;
  isDark: boolean;
  toggleMode: () => void;
  isLoading: boolean;
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

function applyTokens(tokens: ThemeRecord["tokens"], isDark: boolean) {
  const root = document.documentElement;

  // Clear previously set --qpi-* properties
  for (const prop of [...root.style]) {
    if (prop.startsWith("--qpi-")) {
      root.style.removeProperty(prop);
    }
  }

  if (!tokens) return;

  // Apply mode-specific colours
  const colors = isDark ? tokens.colors.dark : tokens.colors.light;
  if (colors) {
    for (const [key, value] of Object.entries(colors)) {
      root.style.setProperty(`--qpi-color-${key}`, value);
    }
  }

  // Apply shared tokens
  const sections = {
    fonts: "font",
    spacing: "spacing",
    radius: "radius",
    shadows: "shadow",
  };
  for (const [section, prefix] of Object.entries(sections)) {
    const values = tokens[section as keyof typeof tokens];
    if (values && typeof values === "object") {
      for (const [key, value] of Object.entries(
        values as Record<string, string>,
      )) {
        root.style.setProperty(`--qpi-${prefix}-${key}`, value);
      }
    }
  }
}

export const ThemeProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [theme, setTheme] = useState<ThemeRecord | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const [isDark, setIsDark] = useState(() => {
    const savedTheme = localStorage.getItem("theme");
    return savedTheme !== "light";
  });

  useEffect(() => {
    fetch("/api/theme/active")
      .then((res) => {
        if (!res.ok) throw new Error("Failed to fetch theme");
        return res.json();
      })
      .then((data: ThemeRecord) => {
        setTheme(data);
      })
      .catch((err) => {
        console.error("Error fetching theme:", err);
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, []);

  useEffect(() => {
    if (isDark) {
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.classList.remove("dark");
    }

    if (theme && theme.tokens) {
      applyTokens(theme.tokens, isDark);
    }
  }, [isDark, theme]);

  const toggleMode = () => {
    setIsDark((prev) => {
      const next = !prev;
      localStorage.setItem("theme", next ? "dark" : "light");
      return next;
    });
  };

  const siteName = theme?.site_name || "QPI Interface";
  const tagline = theme?.tagline || "Control Hub";

  const logoUrl = theme?.logo
    ? `/api/files/themes/${theme.id}/${theme.logo}`
    : null;
  const faviconUrl = theme?.favicon
    ? `/api/files/themes/${theme.id}/${theme.favicon}`
    : null;

  useEffect(() => {
    document.title = siteName
      ? `${siteName} Dashboard`
      : "QPI Dashboard — Obsidian Precision";
  }, [siteName]);

  useEffect(() => {
    if (!faviconUrl) return;
    let link = document.querySelector("link[rel='icon']") as HTMLLinkElement;
    if (!link) {
      link = document.createElement("link");
      link.rel = "icon";
      document.head.appendChild(link);
    }
    link.href = faviconUrl;
  }, [faviconUrl]);

  useEffect(() => {
    if (isLoading) return;

    const loadThemeCSS = async () => {
      try {
        const res = await fetch("/api/theme/css");
        let styleEl = document.getElementById(
          "qpi-theme-css",
        ) as HTMLStyleElement | null;

        if (res.status === 204) {
          if (styleEl) {
            styleEl.textContent = "";
          }
          return;
        }

        const cssText = await res.text();
        if (!styleEl) {
          styleEl = document.createElement("style");
          styleEl.id = "qpi-theme-css";
          document.head.appendChild(styleEl);
        }
        styleEl.textContent = cssText;
      } catch (err) {
        console.error("Error loading theme CSS:", err);
      }
    };

    const loadThemeJS = async () => {
      try {
        const oldScript = document.getElementById("qpi-theme-js");
        if (oldScript) {
          oldScript.remove();
        }

        const res = await fetch("/api/theme/js");
        if (res.status === 204) {
          return;
        }

        const jsText = await res.text();
        const scriptEl = document.createElement("script");
        scriptEl.id = "qpi-theme-js";
        scriptEl.textContent = jsText;
        document.body.appendChild(scriptEl);
      } catch (err) {
        console.error("Error loading theme JS:", err);
      }
    };

    loadThemeCSS();
    loadThemeJS();
  }, [theme?.id, isLoading]);

  const value: ThemeContextValue = {
    theme,
    siteName,
    tagline,
    logoUrl,
    faviconUrl,
    isDark,
    toggleMode,
    isLoading,
  };

  return (
    <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export function useTheme() {
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error("useTheme must be used within a ThemeProvider");
  }
  return context;
}
