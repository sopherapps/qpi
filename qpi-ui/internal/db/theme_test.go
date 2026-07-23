package db

import (
	"testing"

	"qpi/internal/config"

	"github.com/pocketbase/pocketbase/tests"
)

func TestTheme_ToRecordAndRefreshFromRecord(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	defer app.Cleanup()

	cfg := testConfig()
	config.SaveConfigOnApp(app, cfg)
	if err := EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	theme := &Theme{
		Name:      "Dark Cyberpunk",
		IsActive:  true,
		SiteName:  "CyberQPI",
		Tagline:   "Quantum Matrix",
		CustomCSS: ":root { --qpi-color-primary: #00ffcc; }",
		CustomJS:  "console.log('Cyberpunk loaded');",
		Tokens:    config.DefaultThemeTokens,
	}

	rec, err := theme.ToRecord(app)
	if err != nil {
		t.Fatalf("failed to convert theme to record: %v", err)
	}

	if err := app.Save(rec); err != nil {
		t.Fatalf("failed to save theme record: %v", err)
	}

	if rec.Id == "" {
		t.Errorf("expected non-empty ID after save")
	}

	var loadedTheme Theme
	if err := loadedTheme.RefreshFromRecord(rec); err != nil {
		t.Fatalf("failed to refresh theme from record: %v", err)
	}

	if loadedTheme.ID != rec.Id {
		t.Errorf("loadedTheme.ID = %q, want %q", loadedTheme.ID, rec.Id)
	}
	if loadedTheme.Name != "Dark Cyberpunk" {
		t.Errorf("loadedTheme.Name = %q, want %q", loadedTheme.Name, "Dark Cyberpunk")
	}
	if !loadedTheme.IsActive {
		t.Errorf("loadedTheme.IsActive = false, want true")
	}
	if loadedTheme.SiteName != "CyberQPI" {
		t.Errorf("loadedTheme.SiteName = %q, want %q", loadedTheme.SiteName, "CyberQPI")
	}
	if loadedTheme.Tagline != "Quantum Matrix" {
		t.Errorf("loadedTheme.Tagline = %q, want %q", loadedTheme.Tagline, "Quantum Matrix")
	}
	if loadedTheme.CustomCSS != ":root { --qpi-color-primary: #00ffcc; }" {
		t.Errorf("loadedTheme.CustomCSS = %q", loadedTheme.CustomCSS)
	}
	if loadedTheme.CustomJS != "console.log('Cyberpunk loaded');" {
		t.Errorf("loadedTheme.CustomJS = %q", loadedTheme.CustomJS)
	}
	if loadedTheme.Tokens == nil {
		t.Errorf("expected loadedTheme.Tokens to be non-nil")
	}
}
