package db

import (
	"errors"

	"github.com/pocketbase/pocketbase/core"

	"qpi/internal/config"
)

// // GetDefaultThemeRecord constructs a Theme model from compiled-in defaults.
// func GetDefaultThemeRecord() *Theme {
// 	return &Theme{
// 		ID:        "default",
// 		Name:      "Default",
// 		IsActive:  true,
// 		SiteName:  config.DefaultThemeBranding.SiteName,
// 		Tagline:   config.DefaultThemeBranding.Tagline,
// 		Tokens:    config.DefaultThemeTokens,
// 		CustomCSS: "",
// 		CustomJS:  "",
// 	}
// }

// // SaveActiveThemeOnApp caches the active theme globally.
// // If theme is nil, it falls back to caching the default theme record.
// func SaveActiveThemeOnApp(app core.App, theme *Theme) {
// 	if theme == nil {
// 		theme = GetDefaultThemeRecord()
// 	}
// 	cfg, err := config.GetConfigFromApp(app)
// 	if err != nil {
// 		return
// 	}
// 	cfg.UpdateActiveTheme(theme)
// }

// // GetActiveThemeFromApp retrieves the cached active theme.
// // Returns the default theme record if no active theme is cached.
// func GetActiveThemeFromApp(app core.App) *Theme {
// 	cfg, err := config.GetConfigFromApp(app)
// 	if err != nil {
// 		return GetDefaultThemeRecord()
// 	}
// 	theme := cfg.GetActiveTheme()
// 	if t, ok := theme.(*Theme); ok && t != nil {
// 		return t
// 	}
// 	return GetDefaultThemeRecord()
// }

// GetActiveThemeSchema get the active theme schema from the database.
// It returns nil with no error if it is unable to get it.
//
// This is a special query in the database because active theme
// acts like a singleton. It is technically part of config,
// but it is stored in the database.
func GetActiveThemeSchema(app core.App) (*config.ThemeSchema, error) {
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		return nil, err
	}

	var theme Theme
	err = FindOneByFilter(app, cfg.CollectionThemes, &theme, "is_active = true")
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &theme.ThemeSchema, nil
}

// // LoadActiveTheme loads the active theme from the database.
// func LoadActiveTheme(app core.App) error {
// 	cfg, err := config.GetConfigFromApp(app)
// 	if err != nil {
// 		return err
// 	}

// 	var theme Theme
// 	err = FindOneByFilter(app, cfg.CollectionThemes, &theme, "is_active = true")
// 	if err != nil {
// 		return err
// 	}

// 	cfg.UpdateActiveTheme(&theme.ThemeSchema)
// 	return nil
// }
