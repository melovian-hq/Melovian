// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
)

// configFileEnvKeys maps dotted TOML keys to the environment variable the
// existing loaders already read. Values are applied with os.Setenv only
// when the variable is unset, which keeps the precedence chain
// flags > environment > .env > config.toml > built-in defaults.
var configFileEnvKeys = map[string]string{
	"data_dir":        "MELOVIAN_DATA",
	"database_url":    "MELOVIAN_DATABASE_URL",
	"listen":          "MELOVIAN_LISTEN",
	"cache_enabled":   "CACHE_ENABLED",
	"auth_secret":     "MELOVIAN_AUTH_SECRET",
	"no_auth":         "MELOVIAN_NO_AUTH",
	"public_url":      "MELOVIAN_PUBLIC_URL",
	"demo_mode":       "MELOVIAN_DEMO_MODE",
	"allowed_ips":     "MELOVIAN_ALLOWED_IPS",
	"trust_proxy":     "MELOVIAN_TRUST_PROXY",
	"debug_pprof":     "MELOVIAN_DEBUG_PPROF",
	"subsonic_server": "MELOVIAN_SUBSONIC_SERVER",
	"cors_origins":    "MELOVIAN_CORS_ORIGINS",
	"log_level":       "MELOVIAN_LOG_LEVEL",
	"log_file":        "MELOVIAN_LOG_FILE",

	"oidc.issuer":        "MELOVIAN_OIDC_ISSUER",
	"oidc.client_id":     "MELOVIAN_OIDC_CLIENT_ID",
	"oidc.client_secret": "MELOVIAN_OIDC_CLIENT_SECRET",
	"oidc.redirect_url":  "MELOVIAN_OIDC_REDIRECT_URL",
	"oidc.provider_name": "MELOVIAN_OIDC_PROVIDER_NAME",
	"oidc.scopes":        "MELOVIAN_OIDC_SCOPES",
	"oidc.auth_url":      "MELOVIAN_OIDC_AUTH_URL",
	"oidc.token_url":     "MELOVIAN_OIDC_TOKEN_URL",
	"oidc.userinfo_url":  "MELOVIAN_OIDC_USERINFO_URL",

	"local_library.enabled": "MELOVIAN_LOCAL_LIBRARY",
	"local_library.path":    "MELOVIAN_LOCAL_LIBRARY_PATH",
	"local_library.roots":   "MELOVIAN_LOCAL_LIBRARY_ROOTS",

	"dlna.enabled": "MELOVIAN_DLNA_SERVER",
	"dlna.host":    "MELOVIAN_DLNA_HOST",
	"dlna.port":    "MELOVIAN_DLNA_PORT",

	"subsonic.server_url": "NAVIDROME_SERVER",
	"subsonic.username":   "NAVIDROME_USER",
	"subsonic.password":   "NAVIDROME_PASSWORD",

	"connection.min_delay_ms":             "MELOVIAN_CONN_MIN_DELAY_MS",
	"connection.max_delay_ms":             "MELOVIAN_CONN_MAX_DELAY_MS",
	"connection.backoff_multiplier":       "MELOVIAN_CONN_BACKOFF_MULTIPLIER",
	"connection.health_check_interval_ms": "MELOVIAN_CONN_HEALTH_CHECK_MS",
	"connection.offline_poll_ms":          "MELOVIAN_CONN_OFFLINE_POLL_MS",
	"connection.self_heal_interval_ms":    "MELOVIAN_CONN_SELF_HEAL_MS",
	"connection.max_history_entries":      "MELOVIAN_CONN_MAX_HISTORY",

	"sentry.dsn":                "MELOVIAN_SENTRY_DSN",
	"sentry.frontend_dsn":       "MELOVIAN_SENTRY_FRONTEND_DSN",
	"sentry.environment":        "MELOVIAN_SENTRY_ENVIRONMENT",
	"sentry.release":            "MELOVIAN_SENTRY_RELEASE",
	"sentry.traces_sample_rate": "MELOVIAN_SENTRY_TRACES_SAMPLE_RATE",
}

// ResolveConfigPath picks the config file to load. An explicit path wins
// over the default search order: ./melovian.toml, ./config.toml,
// $XDG_CONFIG_HOME/melovian/config.toml, then {data dir}/config.toml.
// Returns the path and true when the file exists and is a regular file.
func ResolveConfigPath(explicit string) (string, bool) {
	var candidates []string
	if explicit != "" {
		candidates = []string{explicit}
	} else {
		candidates = []string{
			"melovian.toml",
			"config.toml",
			filepath.Join(xdg.ConfigHome, "melovian", "config.toml"),
			filepath.Join(DefaultDataDir(), "config.toml"),
		}
	}
	for _, path := range candidates {
		info, err := os.Stat(path)
		if err == nil && info.Mode().IsRegular() {
			return path, true
		}
	}
	return "", false
}

// LoadConfigFileIfResolved resolves the config file path and loads it.
// explicit is the --config flag value; MELOVIAN_CONFIG is consulted when
// empty. An explicit path that does not exist is an error. Returns the
// loaded path, or "" when no config file was found.
func LoadConfigFileIfResolved(explicit string) (string, error) {
	if explicit == "" {
		explicit = strings.TrimSpace(os.Getenv("MELOVIAN_CONFIG"))
	}
	path, found := ResolveConfigPath(explicit)
	if !found {
		if explicit != "" {
			return "", fmt.Errorf("config file %q does not exist", explicit)
		}
		return "", nil
	}
	if err := LoadConfigFile(path); err != nil {
		return path, err
	}
	return path, nil
}

// LoadConfigFile reads a TOML config file and applies each known key as an
// environment variable, but only when that variable is currently unset.
// Unknown keys are rejected so typos surface at startup.
func LoadConfigFile(path string) error {
	var doc map[string]any
	if _, err := toml.DecodeFile(path, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	flat := flattenConfigDoc(doc)
	keys := make([]string, 0, len(flat))
	for key := range flat {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := flat[key]
		if _, empty := value.(emptyConfigTable); empty {
			if !knownConfigTable(key) {
				return fmt.Errorf("%s: unknown config table %q", path, key)
			}
			continue
		}
		env, ok := configFileEnvKeys[key]
		if !ok {
			return fmt.Errorf("%s: unknown config key %q", path, key)
		}
		if err := checkConfigValueType(key, value); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		rendered, err := renderConfigValue(key, value)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if os.Getenv(env) == "" {
			if err := os.Setenv(env, rendered); err != nil {
				return fmt.Errorf("set %s: %w", env, err)
			}
		}
	}
	return nil
}

// emptyConfigTable marks a TOML table that holds no keys so a misspelled
// empty table like [bogus] is still rejected instead of flattening to
// nothing.
type emptyConfigTable struct{}

// flattenConfigDoc collapses nested TOML tables into dotted keys so
// [oidc] issuer and oidc.issuer both resolve to the same mapping entry.
func flattenConfigDoc(doc map[string]any) map[string]any {
	out := make(map[string]any)
	var walk func(prefix string, table map[string]any)
	walk = func(prefix string, table map[string]any) {
		for key, value := range table {
			dotted := key
			if prefix != "" {
				dotted = prefix + "." + key
			}
			if nested, ok := value.(map[string]any); ok {
				if len(nested) == 0 {
					out[dotted] = emptyConfigTable{}
					continue
				}
				walk(dotted, nested)
				continue
			}
			out[dotted] = value
		}
	}
	walk("", doc)
	return out
}

// knownConfigTable reports whether name prefixes at least one known key,
// for example "oidc" or "connection". An empty but recognized table is
// allowed even though it sets nothing.
func knownConfigTable(name string) bool {
	for key := range configFileEnvKeys {
		if strings.HasPrefix(key, name+".") {
			return true
		}
	}
	return false
}

// configFileValueKinds lists the TOML value kinds accepted by each
// non-string key. Keys not listed must be strings so a typo like
// listen = 8080 fails at load instead of at bind.
var configFileValueKinds = map[string]string{
	"cache_enabled":         "bool",
	"no_auth":               "bool",
	"demo_mode":             "bool",
	"trust_proxy":           "bool",
	"debug_pprof":           "bool",
	"dlna.enabled":          "bool",
	"dlna.port":             "int",
	"local_library.enabled": "bool",
	"cors_origins":          "list",
	"oidc.scopes":           "list",
	"local_library.roots":   "list",

	"connection.min_delay_ms":             "int",
	"connection.max_delay_ms":             "int",
	"connection.health_check_interval_ms": "int",
	"connection.offline_poll_ms":          "int",
	"connection.self_heal_interval_ms":    "int",
	"connection.max_history_entries":      "int",
	"connection.backoff_multiplier":       "number",

	"sentry.traces_sample_rate": "number",
}

func checkConfigValueType(key string, value any) error {
	switch configFileValueKinds[key] {
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("config key %q must be a boolean", key)
		}
	case "int":
		if _, ok := value.(int64); !ok {
			return fmt.Errorf("config key %q must be an integer", key)
		}
	case "number":
		switch value.(type) {
		case int64, float64:
		default:
			return fmt.Errorf("config key %q must be a number", key)
		}
	case "list":
		switch value.(type) {
		case string, []any:
		default:
			return fmt.Errorf("config key %q must be a string or an array of strings", key)
		}
	default:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("config key %q must be a string", key)
		}
	}
	return nil
}

func renderConfigValue(key string, value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case bool:
		return strconv.FormatBool(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64), nil
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return "", fmt.Errorf("config key %q must be an array of strings", key)
			}
			parts = append(parts, s)
		}
		return strings.Join(parts, configFileArraySeparator(key)), nil
	default:
		return "", fmt.Errorf("config key %q has unsupported value type %T", key, value)
	}
}

// configFileArraySeparator returns the separator used to flatten a TOML
// string array into the env var form each loader expects. OIDC scopes are
// space-separated, local library roots use the OS path list separator, and
// everything else joins with a comma.
func configFileArraySeparator(key string) string {
	switch key {
	case "oidc.scopes":
		return " "
	case "local_library.roots":
		return string(filepath.ListSeparator)
	default:
		return ","
	}
}
