package config

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDefaultThemeTokens(t *testing.T) {
	// Verify dark mode colors match exact hard-coded values from tailwind.config.js
	expectedDarkColors := map[string]string{
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

	if !reflect.DeepEqual(DefaultThemeTokens.Colors.Dark, expectedDarkColors) {
		t.Errorf("DefaultThemeTokens.Colors.Dark = %v, want %v", DefaultThemeTokens.Colors.Dark, expectedDarkColors)
	}

	// Verify light mode colors
	expectedLightColors := map[string]string{
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

	if !reflect.DeepEqual(DefaultThemeTokens.Colors.Light, expectedLightColors) {
		t.Errorf("DefaultThemeTokens.Colors.Light = %v, want %v", DefaultThemeTokens.Colors.Light, expectedLightColors)
	}

	// Verify fonts
	if DefaultThemeTokens.Fonts["sans"] != "Inter, sans-serif" {
		t.Errorf("unexpected sans font: %s", DefaultThemeTokens.Fonts["sans"])
	}
	if DefaultThemeTokens.Fonts["mono"] != "JetBrains Mono, monospace" {
		t.Errorf("unexpected mono font: %s", DefaultThemeTokens.Fonts["mono"])
	}
	if DefaultThemeTokens.Fonts["display"] != "Geist, sans-serif" {
		t.Errorf("unexpected display font: %s", DefaultThemeTokens.Fonts["display"])
	}

	// Verify spacing
	if DefaultThemeTokens.Spacing["sidebar-width"] != "240px" {
		t.Errorf("unexpected sidebar-width spacing: %s", DefaultThemeTokens.Spacing["sidebar-width"])
	}
}

func TestDefaultThemeBranding(t *testing.T) {
	if DefaultThemeBranding.SiteName != "QPI Interface" {
		t.Errorf("DefaultThemeBranding.SiteName = %q, want %q", DefaultThemeBranding.SiteName, "QPI Interface")
	}
	if DefaultThemeBranding.Tagline != "Control Hub" {
		t.Errorf("DefaultThemeBranding.Tagline = %q, want %q", DefaultThemeBranding.Tagline, "Control Hub")
	}
}

func TestThemeTokens_JSONSerialization(t *testing.T) {
	bytes, err := json.Marshal(DefaultThemeTokens)
	if err != nil {
		t.Fatalf("failed to marshal DefaultThemeTokens to JSON: %v", err)
	}

	var unmarshaled ThemeTokens
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal JSON into ThemeTokens: %v", err)
	}

	if !reflect.DeepEqual(unmarshaled, DefaultThemeTokens) {
		t.Errorf("unmarshaled ThemeTokens does not match original.\nGot:  %+v\nWant: %+v", unmarshaled, DefaultThemeTokens)
	}
}

func TestThemeBranding_JSONSerialization(t *testing.T) {
	bytes, err := json.Marshal(DefaultThemeBranding)
	if err != nil {
		t.Fatalf("failed to marshal DefaultThemeBranding to JSON: %v", err)
	}

	var unmarshaled ThemeBranding
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal JSON into ThemeBranding: %v", err)
	}

	if unmarshaled != DefaultThemeBranding {
		t.Errorf("unmarshaled ThemeBranding = %+v, want %+v", unmarshaled, DefaultThemeBranding)
	}
}
