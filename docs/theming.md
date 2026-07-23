# Dashboard Theming

QPI features a powerful, built-in theming engine that allows superusers to customize the appearance of the web dashboard directly from the PocketBase Admin Panel.

No CSS or recompilation is required for basic theming.

## Features

- **Custom Branding:** Change the site name, tagline, logo, and favicon.
- **Design Tokens:** Customize colors, typography, spacing, radius, and shadows using a JSON-based design token system.
- **Dark/Light Modes:** Each theme supports distinct color palettes for dark and light modes, seamlessly integrated with the dashboard's mode toggle.
- **Custom Injection:** Inject arbitrary CSS and JavaScript directly into the dashboard for advanced customizations.

## Managing Themes

Themes are managed via the **Appearance** tab in the Admin Panel.

1. Log in to the dashboard as a superuser.
2. Click **Admin Panel** in the sidebar.
3. Navigate to the **Appearance** tab.
4. Click **Create New Theme** to open the Theme Editor.

The Theme Editor allows you to configure all aspects of a theme. You can **Preview** a theme before saving it to see how your changes will look live.

When a theme is **Activated**, it immediately takes effect for all users across the dashboard.

## Design Tokens

Design tokens are defined using JSON. The default structure looks like this:

```json
{
  "colors": {
    "light": {
      "background": "#f9fafb",
      "surface": "#ffffff",
      "surface-dim": "#f3f4f6",
      "surface-container": "#e5e7eb",
      "primary": "#111827",
      "secondary": "#6366f1",
      "success": "#22c55e",
      "warning": "#eab308",
      "error": "#ef4444",
      "border": "#e5e7eb"
    },
    "dark": {
      "background": "#09090b",
      "surface": "#18181b",
      "surface-dim": "#131315",
      "surface-container": "#201f22",
      "primary": "#ffffff",
      "secondary": "#6366f1",
      "success": "#22c55e",
      "warning": "#eab308",
      "error": "#ef4444",
      "border": "#27272a"
    }
  },
  "fonts": {
    "sans": "Inter, sans-serif",
    "mono": "JetBrains Mono, monospace",
    "display": "Geist, sans-serif"
  },
  "spacing": {
    "sidebar-width": "240px"
  },
  "radius": {
    "sm": "0.25rem",
    "md": "0.375rem",
    "lg": "0.5rem",
    "full": "9999px"
  },
  "shadows": {
    "sm": "0 1px 2px rgba(0,0,0,0.05)",
    "md": "0 4px 6px rgba(0,0,0,0.1)"
  }
}
```

### Example: High Contrast Theme

To create a high-contrast theme, you could modify the colors like so:

```json
{
  "colors": {
    "light": {
      "background": "#ffffff",
      "surface": "#ffffff",
      "surface-dim": "#eeeeee",
      "surface-container": "#dddddd",
      "primary": "#000000",
      "secondary": "#0000ff",
      "success": "#008000",
      "warning": "#806000",
      "error": "#ff0000",
      "border": "#000000"
    },
    "dark": {
      "background": "#000000",
      "surface": "#000000",
      "surface-dim": "#111111",
      "surface-container": "#222222",
      "primary": "#ffffff",
      "secondary": "#00ffff",
      "success": "#00ff00",
      "warning": "#ffff00",
      "error": "#ff0000",
      "border": "#ffffff"
    }
  }
}
```

## Custom CSS and JS

For advanced use cases, you can inject Custom CSS and Custom JS.
- **Custom CSS:** Useful for hiding specific elements or tweaking layouts that aren't exposed via design tokens.
- **Custom JS:** Useful for adding custom analytics or third-party support chat widgets.

> **Warning:** Custom JS runs with full access to the dashboard context. Only inject trusted code.
