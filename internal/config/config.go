package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt64(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		slog.Error("invalid env var, must be an integer", "key", key, "value", v)
		os.Exit(1)
	}
	return n
}

// A comma-separated list, empty entries are dropped
func envList(key string) []string {
	var out []string
	for _, v := range strings.Split(os.Getenv(key), ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func envBool(key string) bool {
	v := os.Getenv(key)
	return v == "1" || v == "true"
}

type Config struct {
	// Server
	Addr       string // listen address
	PublicURL  string // how browsers reach the app, https:// sets Secure cookies
	BaseDomain string // sites live at <subdomain>.BaseDomain

	// Storage
	SitesDir string // uploaded sites
	DataDir  string // Database

	// Upload limits
	MaxUpload   int64 // bytes per request
	MaxUnzipped int64 // bytes after decompression
	MaxFiles    int64 // entries per archive

	// Auth
	OIDCIssuer       string   // empty disables OIDC login
	OIDCClientID     string   // provider client ID
	OIDCClientSecret string   // provider client secret
	OIDCAdminGroups  []string // "groups" claim values that make an admin
	DevLogin         bool     // passwordless admin login, never in production
}

// Load reads the config from the environment. Defaults are for local dev,
// the Docker image and compose file override them for production
func Load() Config {
	return Config{
		Addr:       env("ADDR", "127.0.0.1:8000"),
		PublicURL:  strings.TrimRight(env("PUBLIC_URL", "http://localhost:8000"), "/"),
		BaseDomain: env("BASE_DOMAIN", "localhost"),

		SitesDir: env("SITES_DIR", ".sites"),
		DataDir:  env("DATA_DIR", ".data"),

		MaxUpload:   envInt64("MAX_UPLOAD", 50<<20),
		MaxUnzipped: envInt64("MAX_UNZIPPED", 200<<20),
		MaxFiles:    envInt64("MAX_FILES", 10_000),

		OIDCIssuer:       os.Getenv("OIDC_ISSUER"),
		OIDCClientID:     os.Getenv("OIDC_CLIENT_ID"),
		OIDCClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		OIDCAdminGroups:  envList("OIDC_ADMIN_GROUPS"),
		DevLogin:         envBool("DEV_LOGIN"),
	}
}
