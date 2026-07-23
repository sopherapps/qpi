package api

import (
	"io/fs"
	"net/http"
	"testing"
	"testing/fstest"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/hook"

	"qpi/internal/config"
	"qpi/internal/db"
)

func TestThemeAPI_Defaults(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:   "GET /api/theme/defaults returns compiled-in defaults",
		Method: http.MethodGet,
		URL:    "/api/theme/defaults",
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			cfg := testConfig()
			config.SaveConfigOnApp(app, cfg)
			_ = db.EnsureSchema(app)
			theme, _ := db.GetActiveThemeSchema(app)
			cfg.UpdateActiveTheme(theme)
			RegisterRoutes(e, testFS())
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"site_name":"QPI Interface"`,
			`"tagline":"Control Hub"`,
			`"colors"`,
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if h := res.Header.Get("Cache-Control"); h != "public, max-age=300" {
				t.Errorf("expected Cache-Control 'public, max-age=300', got %q", h)
			}
		},
	}
	scenario.Test(t)
}

func TestThemeAPI_ActiveDefault(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:   "GET /api/theme/active returns default theme when no active theme in DB",
		Method: http.MethodGet,
		URL:    "/api/theme/active",
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			cfg := testConfig()
			config.SaveConfigOnApp(app, cfg)
			_ = db.EnsureSchema(app)
			theme, _ := db.GetActiveThemeSchema(app)
			cfg.UpdateActiveTheme(theme)
			RegisterRoutes(e, testFS())
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"id":"default"`,
			`"name":"Default"`,
			`"site_name":"QPI Interface"`,
			`"tagline":"Control Hub"`,
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if h := res.Header.Get("Cache-Control"); h != "" {
				t.Errorf("expected no Cache-Control header, got %q", h)
			}
		},
	}
	scenario.Test(t)
}

func TestThemeAPI_ActiveCustomAndCSSJS(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	defer app.Cleanup()

	cfg := testConfig()
	config.SaveConfigOnApp(app, cfg)
	if err := db.EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}
	theme, err := db.GetActiveThemeSchema(app)
	if err != nil {
		t.Fatalf("failed to get active theme schema: %v", err)
	}
	cfg.UpdateActiveTheme(theme)

	// 1. Initial CSS and JS endpoints return 204 No Content for default theme
	scenarioNoCSS := tests.ApiScenario{
		Name:   "GET /api/theme/css returns 204 when no custom CSS",
		Method: http.MethodGet,
		URL:    "/api/theme/css",
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			config.SaveConfigOnApp(app, cfg)
			_ = db.EnsureSchema(app)
			theme, _ := db.GetActiveThemeSchema(app)
			cfg.UpdateActiveTheme(theme)
			RegisterRoutes(e, testFS())
		},
		ExpectedStatus: http.StatusNoContent,
	}
	scenarioNoCSS.Test(t)

	scenarioNoJS := tests.ApiScenario{
		Name:   "GET /api/theme/js returns 204 when no custom JS",
		Method: http.MethodGet,
		URL:    "/api/theme/js",
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			config.SaveConfigOnApp(app, cfg)
			_ = db.EnsureSchema(app)
			theme, _ := db.GetActiveThemeSchema(app)
			cfg.UpdateActiveTheme(theme)
			RegisterRoutes(e, testFS())
		},
		ExpectedStatus: http.StatusNoContent,
	}
	scenarioNoJS.Test(t)

	setupCustomTheme := func(app *tests.TestApp) {
		app.OnRecordCreate().Bind(&hook.Handler[*core.RecordEvent]{
			Func: db.RegisterCollectionHooks(app, db.CollectionHookMap{
				config.DefaultThemesCollection: db.OnThemeUpsert,
			}),
		})
		app.OnRecordUpdate().Bind(&hook.Handler[*core.RecordEvent]{
			Func: db.RegisterCollectionHooks(app, db.CollectionHookMap{
				config.DefaultThemesCollection: db.OnThemeUpsert,
			}),
		})

		themeRec := &db.Theme{
			ThemeSchema: config.ThemeSchema{
				Name:      "Neon Quantum",
				IsActive:  true,
				SiteName:  "Neon Lab",
				Tagline:   "Future is Quantum",
				CustomCSS: ":root { --qpi-color-primary: #00ffaa; }",
				CustomJS:  "console.log('Neon JS');",
			},
		}
		rec, _ := themeRec.ToRecord(app)
		_ = app.Save(rec)
	}

	// Create custom active theme in the outer app (to test db logic directly)
	setupCustomTheme(app)

	// Verify cached active theme is now Neon Quantum
	cached, err := config.GetActiveThemeFromApp(app)
	if err != nil {
		t.Fatalf("failed to get cached active theme: %v", err)
	}

	if cached == nil || cached.Name != "Neon Quantum" {
		t.Fatalf("expected cached theme to be Neon Quantum, got %v", cached)
	}

	// 3. GET /api/theme/active with custom active theme
	scenarioActive := tests.ApiScenario{
		Name:   "GET /api/theme/active returns custom active theme",
		Method: http.MethodGet,
		URL:    "/api/theme/active",
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			config.SaveConfigOnApp(app, cfg)
			_ = db.EnsureSchema(app)
			setupCustomTheme(app)
			theme, _ := db.GetActiveThemeSchema(app)
			cfg.UpdateActiveTheme(theme)
			RegisterRoutes(e, testFS())
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"name":"Neon Quantum"`,
			`"site_name":"Neon Lab"`,
			`"tagline":"Future is Quantum"`,
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if h := res.Header.Get("Cache-Control"); h != "" {
				t.Errorf("expected no Cache-Control header, got %q", h)
			}
		},
	}
	scenarioActive.Test(t)

	// 4. GET /api/theme/css returns custom CSS content
	scenarioCSS := tests.ApiScenario{
		Name:   "GET /api/theme/css returns custom CSS",
		Method: http.MethodGet,
		URL:    "/api/theme/css",
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			config.SaveConfigOnApp(app, cfg)
			_ = db.EnsureSchema(app)
			setupCustomTheme(app)
			theme, _ := db.GetActiveThemeSchema(app)
			cfg.UpdateActiveTheme(theme)
			RegisterRoutes(e, testFS())
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{":root { --qpi-color-primary: #00ffaa; }"},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if h := res.Header.Get("Cache-Control"); h != "public, max-age=300" {
				t.Errorf("expected Cache-Control 'public, max-age=300', got %q", h)
			}
			if h := res.Header.Get("Content-Type"); h != "text/css" {
				t.Errorf("expected Content-Type 'text/css', got %q", h)
			}
		},
	}
	scenarioCSS.Test(t)

	// 5. GET /api/theme/js returns custom JS content
	scenarioJS := tests.ApiScenario{
		Name:   "GET /api/theme/js returns custom JS",
		Method: http.MethodGet,
		URL:    "/api/theme/js",
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			config.SaveConfigOnApp(app, cfg)
			_ = db.EnsureSchema(app)
			setupCustomTheme(app)
			theme, _ := db.GetActiveThemeSchema(app)
			cfg.UpdateActiveTheme(theme)
			RegisterRoutes(e, testFS())
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{"console.log('Neon JS');"},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if h := res.Header.Get("Cache-Control"); h != "public, max-age=300" {
				t.Errorf("expected Cache-Control 'public, max-age=300', got %q", h)
			}
			if h := res.Header.Get("Content-Type"); h != "text/javascript" {
				t.Errorf("expected Content-Type 'text/javascript', got %q", h)
			}
		},
	}
	scenarioJS.Test(t)
}

func TestThemeAPI_ThemeDeactivationAndDeletionRevertsToDefault(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	defer app.Cleanup()

	cfg := testConfig()
	config.SaveConfigOnApp(app, cfg)
	if err := db.EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	// Create custom theme record
	themeRec := &db.Theme{
		ThemeSchema: config.ThemeSchema{
			Name:     "Temporary Theme",
			IsActive: true,
			SiteName: "Temp Site",
		},
	}
	rec, err := themeRec.ToRecord(app)
	if err != nil {
		t.Fatalf("failed to build record: %v", err)
	}
	themeRec.ID = rec.Id

	app.OnRecordDelete().Bind(&hook.Handler[*core.RecordEvent]{
		Func: db.RegisterCollectionHooks(app, db.CollectionHookMap{
			config.DefaultThemesCollection: db.OnThemeDelete,
		}),
	})

	// Save initial active theme to DB and cache
	if err := app.Save(rec); err != nil {
		t.Fatalf("failed to save record: %v", err)
	}
	if err := themeRec.RefreshFromRecord(rec); err != nil {
		t.Fatalf("failed to refresh theme from record: %v", err)
	}
	cfg.UpdateActiveTheme(&themeRec.ThemeSchema)
	appTheme, err := config.GetActiveThemeFromApp(app)
	if err != nil {
		t.Fatalf("failed to get active theme from app: %v", err)
	}
	if appTheme.Name != "Temporary Theme" {
		t.Fatalf("expected active theme in cache")
	}

	// Deleting the active theme should trigger OnThemeDelete hook and revert cache to default theme
	if err := app.Delete(rec); err != nil {
		t.Fatalf("failed to delete theme record: %v", err)
	}

	updatedAppTheme, err := config.GetActiveThemeFromApp(app)
	if err != nil {
		t.Fatalf("failed to get active theme from app: %v", err)
	}

	if updatedAppTheme.ID != "default" || updatedAppTheme.Name != "Default" {
		t.Fatalf("expected cache to revert to default theme after deletion, got %v", updatedAppTheme)
	}
}

func testFS() fs.FS {
	return fstest.MapFS{
		"internal/dashboard/dist/index.html": &fstest.MapFile{
			Data: []byte("<html></html>"),
		},
	}
}
