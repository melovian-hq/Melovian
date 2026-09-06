// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adrg/xdg"
)

const (
	SettingNavidromeServer = "navidrome_server"
	SettingNavidromeUser   = "navidrome_user"
	// SettingNavidromePassword is the settings table key for the legacy
	// credential migration, not a credential value itself.
	SettingNavidromePassword = "navidrome_password" //#nosec G101 -- settings key name, not a hardcoded secret
	SettingActiveInstanceID  = "active_instance_id"
	SettingSourceViewMode    = "source_view_mode"
	SettingMultiLocalLibrary = "multi_local_library"
)

type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

type LocalLibraryConfig struct {
	Enabled         bool
	DefaultPath     string
	AllowCustomPath bool
}

type Config struct {
	DataDir              string
	DatabasePath         string
	DatabaseURL          string
	ListenAddr           string
	CacheEnabled         bool
	LegacyServer         string
	LegacyUser           string
	LegacyPass           string
	ConnectionDefaults   ConnectionDefaults
	AuthSecret           string
	PublicURL            string
	CORSOrigins          []string
	OIDC                 OIDCConfig
	ServerMode           bool
	LocalLibrary         LocalLibraryConfig
	LocalLibraryOverride bool
	DemoMode             bool
	AllowedIPs           []netip.Prefix
	TrustProxy           bool
	DebugPprof           bool
	SubsonicServer       bool
	DLNAServer           bool
	DLNAHost             string
	DLNAPort             int
	Sentry               SentryConfig
}

type ConnectionDefaults struct {
	MinDelayMs            int     `json:"minDelayMs,omitempty"`
	MaxDelayMs            int     `json:"maxDelayMs,omitempty"`
	BackoffMultiplier     float64 `json:"backoffMultiplier,omitempty"`
	HealthCheckIntervalMs int     `json:"healthCheckIntervalMs,omitempty"`
	OfflinePollMs         int     `json:"offlinePollMs,omitempty"`
	SelfHealIntervalMs    int     `json:"selfHealIntervalMs,omitempty"`
	MaxHistoryEntries     int     `json:"maxHistoryEntries,omitempty"`
}

func DefaultDataDir() string {
	if value := os.Getenv("MELOVIAN_DATA"); value != "" {
		return value
	}
	base := xdg.DataHome
	if base == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return "data"
		}
		return filepath.Join(dir, "melovian")
	}
	return filepath.Join(base, "melovian")
}

func LoadConfig() (Config, error) {
	dataDir := DefaultDataDir()
	allowedIPs, err := parseAllowedIPs(os.Getenv("MELOVIAN_ALLOWED_IPS"))
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		DataDir:            dataDir,
		DatabasePath:       filepath.Join(dataDir, "melovian.db"),
		DatabaseURL:        strings.TrimSpace(os.Getenv("MELOVIAN_DATABASE_URL")),
		ListenAddr:         envOr("MELOVIAN_LISTEN", "127.0.0.1:17337"),
		CacheEnabled:       envOr("CACHE_ENABLED", "true") != "false",
		LegacyServer:       os.Getenv("NAVIDROME_SERVER"),
		LegacyUser:         os.Getenv("NAVIDROME_USER"),
		LegacyPass:         os.Getenv("NAVIDROME_PASSWORD"),
		ConnectionDefaults: loadConnectionDefaults(),
		AuthSecret:         os.Getenv("MELOVIAN_AUTH_SECRET"),
		PublicURL:          strings.TrimRight(os.Getenv("MELOVIAN_PUBLIC_URL"), "/"),
		CORSOrigins:        parseCORSOrigins(os.Getenv("MELOVIAN_CORS_ORIGINS")),
		OIDC:               loadOIDCConfig(),
		LocalLibrary:       loadLocalLibraryConfig(false),
		DemoMode:           envTruthy(os.Getenv("MELOVIAN_DEMO_MODE")),
		AllowedIPs:         allowedIPs,
		TrustProxy:         envTruthy(os.Getenv("MELOVIAN_TRUST_PROXY")),
		DebugPprof:         envTruthy(os.Getenv("MELOVIAN_DEBUG_PPROF")),
		SubsonicServer:     envOr("MELOVIAN_SUBSONIC_SERVER", "true") != "false",
		DLNAServer:         envOr("MELOVIAN_DLNA_SERVER", "false") == "true",
		DLNAHost:           envOr("MELOVIAN_DLNA_HOST", "0.0.0.0"),
		DLNAPort:           envIntOr("MELOVIAN_DLNA_PORT", 8200),
		Sentry:             loadSentryConfig(),
	}
	return cfg, nil
}

func (c Config) LocalLibraryEffective() LocalLibraryConfig {
	if c.LocalLibraryOverride {
		lib := c.LocalLibrary
		if c.ServerMode && c.AuthEnabled() && lib.DefaultPath != "" {
			lib.AllowCustomPath = false
		}
		return lib
	}
	return loadLocalLibraryConfig(c.ServerMode && c.AuthEnabled())
}

func (c Config) DemoModeEffective() bool {
	return c.ServerMode && c.DemoMode
}

// DatabaseDriver returns "postgres" when DatabaseURL uses a Postgres scheme, otherwise "sqlite".
func (c Config) DatabaseDriver() string {
	raw := strings.TrimSpace(c.DatabaseURL)
	if raw == "" {
		return "sqlite"
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "sqlite"
	}
	switch strings.ToLower(u.Scheme) {
	case "postgres", "postgresql":
		return "postgres"
	default:
		return "sqlite"
	}
}

func (c Config) IPAllowlistEnabled() bool {
	return len(c.AllowedIPs) > 0
}

func (c Config) AuthEnabled() bool {
	if c.DemoModeEffective() {
		return false
	}
	return strings.TrimSpace(c.AuthSecret) != ""
}

func (c Config) SubsonicServerEffective() bool {
	return c.SubsonicServer && c.LocalLibraryEffective().Enabled
}

func (c Config) DLNAServerEffective() bool {
	return c.DLNAServer && c.LocalLibraryEffective().Enabled
}

func ValidateServerConfig(cfg Config) error {
	if !cfg.ServerMode {
		return nil
	}
	if !cfg.DemoModeEffective() {
		return nil
	}
	server := strings.TrimSpace(cfg.LegacyServer)
	// Empty or fake:// uses the built-in demo catalog. No Navidrome required.
	if server == "" || strings.HasPrefix(strings.ToLower(server), "fake://") {
		return nil
	}
	if strings.TrimSpace(cfg.LegacyUser) == "" {
		return fmt.Errorf("MELOVIAN_DEMO_MODE with a real server requires NAVIDROME_USER")
	}
	if strings.TrimSpace(cfg.LegacyPass) == "" {
		return fmt.Errorf("MELOVIAN_DEMO_MODE with a real server requires NAVIDROME_PASSWORD")
	}
	return nil
}

func (c Config) OIDCEnabled() bool {
	return c.AuthEnabled() &&
		strings.TrimSpace(c.OIDC.Issuer) != "" &&
		strings.TrimSpace(c.OIDC.ClientID) != ""
}

func loadOIDCConfig() OIDCConfig {
	scopesRaw := envOr("MELOVIAN_OIDC_SCOPES", "openid profile email")
	scopes := strings.Fields(scopesRaw)
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}
	return OIDCConfig{
		Issuer:       strings.TrimSpace(os.Getenv("MELOVIAN_OIDC_ISSUER")),
		ClientID:     strings.TrimSpace(os.Getenv("MELOVIAN_OIDC_CLIENT_ID")),
		ClientSecret: os.Getenv("MELOVIAN_OIDC_CLIENT_SECRET"),
		RedirectURL:  strings.TrimSpace(os.Getenv("MELOVIAN_OIDC_REDIRECT_URL")),
		Scopes:       scopes,
	}
}

func loadConnectionDefaults() ConnectionDefaults {
	defaults := ConnectionDefaults{
		MinDelayMs:            2000,
		MaxDelayMs:            120_000,
		BackoffMultiplier:     1.6,
		HealthCheckIntervalMs: 60_000,
		OfflinePollMs:         5000,
		SelfHealIntervalMs:    300_000,
		MaxHistoryEntries:     32,
	}
	if v := envInt("MELOVIAN_CONN_MIN_DELAY_MS", 0); v > 0 {
		defaults.MinDelayMs = v
	}
	if v := envInt("MELOVIAN_CONN_MAX_DELAY_MS", 0); v > 0 {
		defaults.MaxDelayMs = v
	}
	if v := envFloat("MELOVIAN_CONN_BACKOFF_MULTIPLIER", 0); v > 0 {
		defaults.BackoffMultiplier = v
	}
	if v := envInt("MELOVIAN_CONN_HEALTH_CHECK_MS", 0); v > 0 {
		defaults.HealthCheckIntervalMs = v
	}
	if v := envInt("MELOVIAN_CONN_OFFLINE_POLL_MS", 0); v > 0 {
		defaults.OfflinePollMs = v
	}
	if v := envInt("MELOVIAN_CONN_SELF_HEAL_MS", 0); v > 0 {
		defaults.SelfHealIntervalMs = v
	}
	if v := envInt("MELOVIAN_CONN_MAX_HISTORY", 0); v > 0 {
		defaults.MaxHistoryEntries = v
	}
	return defaults
}

func envInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	var value int
	if _, err := fmt.Sscanf(raw, "%d", &value); err != nil {
		return fallback
	}
	return value
}

func envFloat(key string, fallback float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	var value float64
	if _, err := fmt.Sscanf(raw, "%f", &value); err != nil {
		return fallback
	}
	return value
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envTruthy(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseCORSOrigins(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == ';'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		origin := strings.TrimRight(strings.TrimSpace(part), "/")
		if origin == "" {
			continue
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		seen[origin] = struct{}{}
		out = append(out, origin)
	}
	return out
}

func parseAllowedIPs(raw string) ([]netip.Prefix, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == ';'
	})
	prefixes := make([]netip.Prefix, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "/") {
			prefix, err := netip.ParsePrefix(part)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR in MELOVIAN_ALLOWED_IPS %q: %w", part, err)
			}
			prefixes = append(prefixes, prefix)
			continue
		}
		addr, err := netip.ParseAddr(part)
		if err != nil {
			return nil, fmt.Errorf("invalid IP in MELOVIAN_ALLOWED_IPS %q: %w", part, err)
		}
		bits := 32
		if addr.Is6() {
			bits = 128
		}
		prefixes = append(prefixes, netip.PrefixFrom(addr, bits))
	}
	return prefixes, nil
}

func loadLocalLibraryConfig(multiUser bool) LocalLibraryConfig {
	raw := strings.TrimSpace(os.Getenv("MELOVIAN_LOCAL_LIBRARY"))
	enabled := raw == "" || (raw != "false" && !strings.EqualFold(raw, "disabled"))
	if multiUser && raw == "" {
		enabled = false
	}

	defaultPath := strings.TrimSpace(os.Getenv("MELOVIAN_LOCAL_LIBRARY_PATH"))
	allowCustom := defaultPath == ""
	if defaultPath != "" && multiUser {
		allowCustom = false
	}

	return LocalLibraryConfig{
		Enabled:         enabled,
		DefaultPath:     defaultPath,
		AllowCustomPath: allowCustom,
	}
}
