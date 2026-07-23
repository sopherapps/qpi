package config

// ThemeColors holds both light and dark colour palettes.
type ThemeColors struct {
	Light map[string]string `json:"light"`
	Dark  map[string]string `json:"dark"`
}

// ThemeTokens holds all design tokens for a theme.
type ThemeTokens struct {
	Colors  ThemeColors       `json:"colors"`
	Fonts   map[string]string `json:"fonts"`
	Spacing map[string]string `json:"spacing"`
	Radius  map[string]string `json:"radius"`
	Shadows map[string]string `json:"shadows"`
}

// ThemeBranding holds the default branding values.
type ThemeBranding struct {
	SiteName string `json:"site_name"`
	Tagline  string `json:"tagline"`
}

// DefaultThemeTokens is the single source of truth for dashboard design tokens.
var DefaultThemeTokens = ThemeTokens{
	Colors: ThemeColors{
		Light: map[string]string{
			"background":        "#f9fafb",
			"surface":           "#ffffff",
			"surface-dim":       "#f3f4f6",
			"surface-container": "#e5e7eb",
			"primary":           "#111827",
			"secondary":         "#6366f1",
			"success":           "#22c55e",
			"warning":           "#eab308",
			"error":             "#ef4444",
			"border":            "#e5e7eb",
		},
		Dark: map[string]string{
			"background":        "#09090b",
			"surface":           "#18181b",
			"surface-dim":       "#131315",
			"surface-container": "#201f22",
			"primary":           "#ffffff",
			"secondary":         "#6366f1",
			"success":           "#22c55e",
			"warning":           "#eab308",
			"error":             "#ef4444",
			"border":            "#27272a",
		},
	},
	Fonts: map[string]string{
		"sans":    "Inter, sans-serif",
		"mono":    "JetBrains Mono, monospace",
		"display": "Geist, sans-serif",
	},
	Spacing: map[string]string{
		"sidebar-width": "240px",
	},
	Radius: map[string]string{
		"sm":   "0.25rem",
		"md":   "0.375rem",
		"lg":   "0.5rem",
		"full": "9999px",
	},
	Shadows: map[string]string{
		"sm": "0 1px 2px rgba(0,0,0,0.05)",
		"md": "0 4px 6px rgba(0,0,0,0.1)",
	},
}

// DefaultThemeBranding holds the built-in brand values.
var DefaultThemeBranding = ThemeBranding{
	SiteName: "QPI Interface",
	Tagline:  "Control Hub",
}
