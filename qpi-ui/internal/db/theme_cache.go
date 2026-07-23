package db

import (
	"github.com/pocketbase/pocketbase/core"

	"qpi/internal/config"
)

const appStoreActiveThemeKey = "active_theme"

// GetDefaultThemeRecord constructs a Theme model from compiled-in defaults.
func GetDefaultThemeRecord() *Theme {
	return &Theme{
		ID:        "default",
		Name:      "Default",
		IsActive:  true,
		SiteName:  config.DefaultThemeBranding.SiteName,
		Tagline:   config.DefaultThemeBranding.Tagline,
		Tokens:    config.DefaultThemeTokens,
		CustomCSS: "",
		CustomJS:  "",
	}
}

// SaveActiveThemeOnApp caches the active theme in the app store.
// If theme is nil, it falls back to caching the default theme record.
func SaveActiveThemeOnApp(app core.App, theme *Theme) {
	if theme == nil {
		theme = GetDefaultThemeRecord()
	}
	app.Store().Set(appStoreActiveThemeKey, theme)
}

// GetActiveThemeFromApp retrieves the cached active theme.
// Returns the default theme record if no active theme is cached.
func GetActiveThemeFromApp(app core.App) *Theme {
	value := app.Store().Get(appStoreActiveThemeKey)
	if theme, ok := value.(*Theme); ok && theme != nil {
		return theme
	}
	return GetDefaultThemeRecord()
}

// InitActiveThemeCache initializes the active theme cache on server startup.
func InitActiveThemeCache(app core.App) error {
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		return err
	}

	record, err := app.FindFirstRecordByFilter(
		cfg.CollectionThemes,
		"is_active = true",
	)
	if err != nil {
		SaveActiveThemeOnApp(app, nil)
		return nil
	}

	var theme Theme
	if err := theme.RefreshFromRecord(record); err != nil {
		return err
	}
	SaveActiveThemeOnApp(app, &theme)
	return nil
}
