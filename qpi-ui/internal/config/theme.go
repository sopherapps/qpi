package config

import "time"

// ThemeSchema represents the active theme JSON structure without custom CSS/JS.
type ThemeSchema struct {
	ID        string             `json:"id"         db:"id"`
	Name      string             `json:"name"       db:"name"       required:"true"`
	IsActive  bool               `json:"is_active"  db:"is_active"`
	SiteName  string             `json:"site_name"  db:"site_name"`
	Tagline   string             `json:"tagline"    db:"tagline"`
	Logo      string             `json:"logo"       db:"logo"       type:"file" maxSelect:"1" maxSize:"2097152" mimeTypes:"image/png,image/svg+xml,image/webp"`
	Favicon   string             `json:"favicon"    db:"favicon"     type:"file" maxSelect:"1" maxSize:"524288"  mimeTypes:"image/png,image/svg+xml,image/x-icon,image/webp"`
	Tokens    *ThemeTokensSchema `json:"tokens"     db:"tokens"      type:"json"`
	CustomCSS string             `json:"custom_css" db:"custom_css"`
	CustomJS  string             `json:"custom_js"  db:"custom_js"`
	Created   string             `json:"created"    db:"created"     type:"autodate" onCreate:"true"`
	Updated   string             `json:"updated"    db:"updated"     type:"autodate" onCreate:"true" onUpdate:"true"`
}

// NewDefaultThemeSchema constructs a ThemeSchema with default values.
func NewDefaultThemeSchema() *ThemeSchema {
	ts := &ThemeSchema{}
	ts.SetDefaults()
	return ts
}

func (ts *ThemeSchema) SetDefaults() {
	if ts.ID == "" {
		ts.ID = "default"
	}
	if ts.Name == "" {
		ts.Name = "Default"
	}
	ts.IsActive = true
	if ts.SiteName == "" {
		ts.SiteName = "QPI Interface"
	}
	if ts.Tagline == "" {
		ts.Tagline = "Control Hub"
	}
	if ts.Tokens == nil {
		ts.Tokens = &ThemeTokensSchema{}
		ts.Tokens.SetDefaults()
	}
	if ts.Updated == "" {
		ts.Updated = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	}
}

// ToMap converts the DTO to a map of field values
func (ts *ThemeSchema) ToMap() map[string]any {
	return map[string]any{
		"id":        ts.ID,
		"name":      ts.Name,
		"is_active": ts.IsActive,
		"site_name": ts.SiteName,
		"tagline":   ts.Tagline,
		"logo":      ts.Logo,
		"favicon":   ts.Favicon,
		"tokens":    ts.Tokens.ToMap(),
		"updated":   ts.Updated,
	}
}

// ThemeColorsSchema holds both light and dark colour palettes.
type ThemeColorsSchema struct {
	Light map[string]string `json:"light"`
	Dark  map[string]string `json:"dark"`
}

func (tc *ThemeColorsSchema) SetDefaults() {
	if tc.Light == nil {
		tc.Light = map[string]string{
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
		}
	}
	if tc.Dark == nil {
		tc.Dark = map[string]string{
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
		}
	}
}

// ToMap converts the DTO to a map of field values
func (tc *ThemeColorsSchema) ToMap() map[string]any {
	return map[string]any{
		"light": tc.Light,
		"dark":  tc.Dark,
	}
}

// ThemeTokensSchema holds all design tokens for a theme.
type ThemeTokensSchema struct {
	Colors  *ThemeColorsSchema `json:"colors"`
	Fonts   map[string]string  `json:"fonts"`
	Spacing map[string]string  `json:"spacing"`
	Radius  map[string]string  `json:"radius"`
	Shadows map[string]string  `json:"shadows"`
}

func (tt *ThemeTokensSchema) SetDefaults() {
	if tt.Colors == nil {
		tt.Colors = &ThemeColorsSchema{}
	}
	if tt.Fonts == nil {
		tt.Fonts = map[string]string{
			"sans":    "Inter, sans-serif",
			"mono":    "JetBrains Mono, monospace",
			"display": "Geist, sans-serif",
		}
	}
	if tt.Spacing == nil {
		tt.Spacing = map[string]string{
			"sidebar-width": "240px",
		}
	}
	if tt.Radius == nil {
		tt.Radius = map[string]string{
			"sm":   "0.25rem",
			"md":   "0.375rem",
			"lg":   "0.5rem",
			"full": "9999px",
		}
	}
	if tt.Shadows == nil {
		tt.Shadows = map[string]string{
			"sm": "0 1px 2px rgba(0,0,0,0.05)",
			"md": "0 4px 6px rgba(0,0,0,0.1)",
		}
	}
	if tt.Colors != nil {
		tt.Colors = &ThemeColorsSchema{}
		tt.Colors.SetDefaults()
	}
}

// ToMap converts the DTO to a map of field values
func (tt *ThemeTokensSchema) ToMap() map[string]any {
	return map[string]any{
		"colors":  tt.Colors.ToMap(),
		"fonts":   tt.Fonts,
		"spacing": tt.Spacing,
		"radius":  tt.Radius,
		"shadows": tt.Shadows,
	}
}

// // ThemeBranding holds the default branding values.
// type ThemeBranding struct {
// 	SiteName string `json:"site_name"`
// 	Tagline  string `json:"tagline"`
// }

// // DefaultThemeTokens is the single source of truth for dashboard design tokens.
// var DefaultThemeTokens = ThemeTokens{
// 	Colors: &ThemeColorsSchema{
// 		Light: map[string]string{
// 			"background":        "#f9fafb",
// 			"surface":           "#ffffff",
// 			"surface-dim":       "#f3f4f6",
// 			"surface-container": "#e5e7eb",
// 			"primary":           "#111827",
// 			"secondary":         "#6366f1",
// 			"success":           "#22c55e",
// 			"warning":           "#eab308",
// 			"error":             "#ef4444",
// 			"border":            "#e5e7eb",
// 		},
// 		Dark: map[string]string{
// 			"background":        "#09090b",
// 			"surface":           "#18181b",
// 			"surface-dim":       "#131315",
// 			"surface-container": "#201f22",
// 			"primary":           "#ffffff",
// 			"secondary":         "#6366f1",
// 			"success":           "#22c55e",
// 			"warning":           "#eab308",
// 			"error":             "#ef4444",
// 			"border":            "#27272a",
// 		},
// 	},
// 	Fonts: map[string]string{
// 		"sans":    "Inter, sans-serif",
// 		"mono":    "JetBrains Mono, monospace",
// 		"display": "Geist, sans-serif",
// 	},
// 	Spacing: map[string]string{
// 		"sidebar-width": "240px",
// 	},
// 	Radius: map[string]string{
// 		"sm":   "0.25rem",
// 		"md":   "0.375rem",
// 		"lg":   "0.5rem",
// 		"full": "9999px",
// 	},
// 	Shadows: map[string]string{
// 		"sm": "0 1px 2px rgba(0,0,0,0.05)",
// 		"md": "0 4px 6px rgba(0,0,0,0.1)",
// 	},
// }

// // DefaultThemeBranding holds the built-in brand values.
// var DefaultThemeBranding = ThemeBranding{
// 	SiteName: "QPI Interface",
// 	Tagline:  "Control Hub",
// }
