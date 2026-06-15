package api

import (
	"fmt"
	"net"
	"net/http"
	"qpi/internal/config"
	"qpi/internal/db"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// getPaginationParams gets the pagination parameters from the request event, given a default sort
func getPaginationParams(re *core.RequestEvent, defaultSort string) (sort string, skip int, limit int) {
	skip = getQueryParamInt(re, "skip", 0)
	limit = getQueryParamInt(re, "limit", 0)
	sort = getQueryParam(re, "sort", defaultSort)
	return
}

// getQueryParam gets the query parameter from the request event, defaulting to a given value if not present
func getQueryParam(re *core.RequestEvent, key string, defaultValue string) string {
	value := re.Request.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getQueryParamInt gets the query param as an integer, defaulting to a given value if not present or invalid
func getQueryParamInt(re *core.RequestEvent, key string, defaultValue int) int {
	value := re.Request.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	parsedValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return parsedValue
}

// getCurrentUser gets the current logged in user
func getCurrentUser(re *core.RequestEvent) (*db.User, error) {
	if re.Auth != nil && re.Auth.Collection().Name == "users" {
		var user db.User
		err := user.RefreshFromRecord(re.Auth)
		if err != nil {
			return nil, re.Error(http.StatusUnauthorized, "authentication required", err)
		}

		return &user, nil
	}

	token := getToken(re)
	user, err := db.GetUserByToken(re.App, token)
	if err != nil {
		return nil, re.Error(http.StatusUnauthorized, "authentication required", err)
	}

	return user, nil
}

// getToken gets the token from the request
func getToken(re *core.RequestEvent) string {
	token := re.Request.Header.Get("X-API-Token")
	if token != "" {
		return token
	}

	authHeader := re.Request.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return ""
}

// parseBody parses the request body and validates it
func parseBody[T GeneralDTO](cfg *config.AppConfig, re *core.RequestEvent, v T) error {
	err := re.BindBody(v)
	if err != nil {
		return re.Error(http.StatusBadRequest, "invalid request body", err)
	}

	v.SetDefaults()
	err = cfg.Validator.Struct(v)
	if err != nil {
		return re.Error(http.StatusBadRequest, "invalid request body", err)
	}

	return nil
}

// Saves a given model into the database and updates itself based on the db value
func saveToDb[T db.DbModel](app core.App, item T) error {
	record, err := item.ToRecord(app)
	if err != nil {
		return err
	}

	err = app.Save(record)
	if err != nil {
		return err
	}

	err = item.RefreshFromRecord(record)
	if err != nil {
		return err
	}

	return nil
}

// getAddrFromReq extracts the scheme and host from a request to build the full QPI address
func getAddrFromReq(re *core.RequestEvent) string {
	scheme := "http"
	if re.Request.TLS != nil {
		scheme = "https"
	}
	if proto := re.Request.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return fmt.Sprintf("%s://%s", scheme, re.Request.Host)
}

// findFreePorts searches for free TCP ports within the configuration range,
// excluding ports currently reserved/allocated in the QPUs database table.
func findFreePorts(app core.App, count int) ([]int, error) {
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		return nil, err
	}

	allocated := make(map[int]bool)
	filter := fmt.Sprintf("nng_command_port >= %d || nng_result_port >= %d", cfg.PortRangeStart, cfg.PortRangeStart)
	records, err := app.FindRecordsByFilter(cfg.CollectionQPUs, filter, "", 0, 0)
	if err == nil {
		for _, r := range records {
			cmd := r.GetInt("nng_command_port")
			res := r.GetInt("nng_result_port")
			if cmd > 0 {
				allocated[cmd] = true
			}
			if res > 0 {
				allocated[res] = true
			}
		}
	}

	var ports []int
	for port := cfg.PortRangeStart; port < cfg.PortRangeEnd; port++ {
		if allocated[port] {
			continue
		}
		addr := fmt.Sprintf(":%d", port)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ln.Close()
			ports = append(ports, port)
			allocated[port] = true
			if len(ports) == count {
				return ports, nil
			}
		}
	}
	return nil, fmt.Errorf("could not find %d free ports in range %d-%d", count, cfg.PortRangeStart, cfg.PortRangeEnd)
}
