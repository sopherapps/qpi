package api

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"qpi/internal/config"
	"qpi/internal/db"
)

// ThemeActiveResponse represents the active theme JSON structure without custom CSS/JS.
type ThemeActiveResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
	SiteName string `json:"site_name"`
	Tagline  string `json:"tagline"`
	Logo     string `json:"logo"`
	Favicon  string `json:"favicon"`
	Tokens   any    `json:"tokens"`
}

func handleThemeActive(re *core.RequestEvent) error {
	re.Response.Header().Set("Cache-Control", "public, max-age=300")
	theme := db.GetActiveThemeFromApp(re.App)

	resp := ThemeActiveResponse{
		ID:       theme.ID,
		Name:     theme.Name,
		IsActive: theme.IsActive,
		SiteName: theme.SiteName,
		Tagline:  theme.Tagline,
		Logo:     theme.Logo,
		Favicon:  theme.Favicon,
		Tokens:   theme.Tokens,
	}
	return re.JSON(http.StatusOK, resp)
}

func handleThemeDefaults(re *core.RequestEvent) error {
	re.Response.Header().Set("Cache-Control", "public, max-age=300")
	return re.JSON(http.StatusOK, map[string]any{
		"tokens":   config.DefaultThemeTokens,
		"branding": config.DefaultThemeBranding,
	})
}

func handleThemeCSS(re *core.RequestEvent) error {
	re.Response.Header().Set("Cache-Control", "public, max-age=300")
	theme := db.GetActiveThemeFromApp(re.App)
	if theme == nil || theme.CustomCSS == "" {
		return re.NoContent(http.StatusNoContent)
	}

	return re.Blob(http.StatusOK, "text/css", []byte(theme.CustomCSS))
}

func handleThemeJS(re *core.RequestEvent) error {
	re.Response.Header().Set("Cache-Control", "public, max-age=300")
	theme := db.GetActiveThemeFromApp(re.App)
	if theme == nil || theme.CustomJS == "" {
		return re.NoContent(http.StatusNoContent)
	}

	return re.Blob(http.StatusOK, "text/javascript", []byte(theme.CustomJS))
}
