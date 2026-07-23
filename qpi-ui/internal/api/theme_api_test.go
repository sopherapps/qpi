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
			config.SaveConfigOnApp(app, testConfig())
			_ = db.EnsureSchema(app)
			_ = db.InitActiveThemeCache(app)
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
			config.SaveConfigOnApp(app, testConfig())
			_ = db.EnsureSchema(app)
			_ = db.InitActiveThemeCache(app)
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
			if h := res.Header.Get("Cache-Control"); h != "public, max-age=300" {
				t.Errorf("expected Cache-Control 'public, max-age=300', got %q", h)
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
	if err := db.InitActiveThemeCache(app); err != nil {
		t.Fatalf("failed to init theme cache: %v", err)
	}

	// 1. Initial CSS and JS endpoints return 204 No Content for default theme
	scenarioNoCSS := tests.ApiScenario{
		Name:   "GET /api/theme/css returns 204 when no custom CSS",
		Method: http.MethodGet,
		URL:    "/api/theme/css",
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			config.SaveConfigOnApp(app, cfg)
			_ = db.EnsureSchema(app)
			_ = db.InitActiveThemeCache(app)
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
			_ = db.InitActiveThemeCache(app)
			RegisterRoutes(e, testFS())
		},
		ExpectedStatus: http.StatusNoContent,
	}
	scenarioNoJS.Test(t)

	// 2. Create custom active theme in DB
	theme := &db.Theme{
		Name:      "Neon Quantum",
		IsActive:  true,
		SiteName:  "Neon Lab",
		Tagline:   "Future is Quantum",
		CustomCSS: ":root { --qpi-color-primary: #00ffaa; }",
		CustomJS:  "console.log('Neon JS');",
		Tokens:    config.DefaultThemeTokens,
	}
	rec, err := theme.ToRecord(app)
	if err != nil {
		t.Fatalf("failed to build record: %v", err)
	}

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

	if err := app.Save(rec); err != nil {
		t.Fatalf("failed to save theme record: %v", err)
	}

	// Verify cached active theme is now Neon Quantum
	cached := db.GetActiveThemeFromApp(app)
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
			db.SaveActiveThemeOnApp(app, theme)
			RegisterRoutes(e, testFS())
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"name":"Neon Quantum"`,
			`"site_name":"Neon Lab"`,
			`"tagline":"Future is Quantum"`,
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if h := res.Header.Get("Cache-Control"); h != "public, max-age=300" {
				t.Errorf("expected Cache-Control 'public, max-age=300', got %q", h)
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
			db.SaveActiveThemeOnApp(app, theme)
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
			db.SaveActiveThemeOnApp(app, theme)
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
	theme := &db.Theme{
		Name:     "Temporary Theme",
		IsActive: true,
		SiteName: "Temp Site",
	}
	rec, err := theme.ToRecord(app)
	if err != nil {
		t.Fatalf("failed to build record: %v", err)
	}
	theme.ID = rec.Id

	app.OnRecordDelete().Bind(&hook.Handler[*core.RecordEvent]{
		Func: db.RegisterCollectionHooks(app, db.CollectionHookMap{
			config.DefaultThemesCollection: db.OnThemeDelete,
		}),
	})

	// Save initial active theme to DB and cache
	if err := app.Save(rec); err != nil {
		t.Fatalf("failed to save record: %v", err)
	}
	if err := theme.RefreshFromRecord(rec); err != nil {
		t.Fatalf("failed to refresh theme from record: %v", err)
	}
	db.SaveActiveThemeOnApp(app, theme)
	if db.GetActiveThemeFromApp(app).Name != "Temporary Theme" {
		t.Fatalf("expected active theme in cache")
	}

	// Deleting the active theme should trigger OnThemeDelete hook and revert cache to default theme
	if err := app.Delete(rec); err != nil {
		t.Fatalf("failed to delete theme record: %v", err)
	}

	cached := db.GetActiveThemeFromApp(app)
	if cached.ID != "default" || cached.Name != "Default" {
		t.Fatalf("expected cache to revert to default theme after deletion, got %v", cached)
	}
}

func testFS() fs.FS {
	return fstest.MapFS{
		"internal/dashboard/dist/index.html": &fstest.MapFile{
			Data: []byte("<html></html>"),
		},
	}
}
