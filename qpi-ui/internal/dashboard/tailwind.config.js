/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        background: "#09090b",
        surface: "#18181b",
        "surface-dim": "#131315",
        "surface-container": "#201f22",
        primary: "#ffffff",
        secondary: "#6366f1",
        success: "#22c55e",
        warning: "#eab308",
        error: "#ef4444",
        border: "#27272a",
      },
      spacing: {
        "sidebar-width": "240px",
      },
    },
  },
  plugins: [],
};
