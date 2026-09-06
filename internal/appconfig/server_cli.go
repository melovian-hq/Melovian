// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"melovian/internal/brand"
	"melovian/internal/termout"
)

const defaultServerHost = "0.0.0.0"
const defaultServerPort = 8080

type ServerCLI struct {
	ShowHelp bool
	NoAuth   bool

	EnvFile string

	Host   string
	Port   int
	Listen string

	DataDir    string
	NoCache    bool
	AuthSecret string
	PublicURL  string
	DemoMode   bool
	AllowedIPs string
	TrustProxy bool

	NavidromeServer   string
	NavidromeUser     string
	NavidromePassword string

	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string
	OIDCScopes       string

	LocalLibrary     bool
	LocalLibraryPath string
	LocalLibraryOff  bool

	LogLevel   string
	LogFile    string
	Verbose    bool
	Debug      bool
	DebugPprof bool

	ConnMinDelayMs        int
	ConnMaxDelayMs        int
	ConnBackoffMultiplier float64
	ConnHealthCheckMs     int
	ConnOfflinePollMs     int
	ConnSelfHealMs        int
	ConnMaxHistory        int

	Update        bool
	UpdateVersion string
	// Args holds positional arguments left after flag parsing (for example
	// the version in `melovian-server --update v1.2.3`).
	Args []string

	visited map[string]bool
}

// updateFlag accepts an optional value: `--update`, `--update v1.2.3`, and
// `--update=v1.2.3` all work. With IsBoolFlag the bare form calls
// Set("true"), and the space-separated form leaves the version as a
// positional argument captured in ServerCLI.Args.
type updateFlag struct {
	requested *bool
	version   *string
}

func (f updateFlag) String() string { return *f.version }

func (f updateFlag) Set(s string) error {
	*f.requested = true
	if s != "true" && s != "" {
		*f.version = strings.TrimPrefix(strings.TrimSpace(s), "v")
	}
	return nil
}

func (updateFlag) IsBoolFlag() bool { return true }

func NewServerCLI() *ServerCLI {
	return &ServerCLI{visited: make(map[string]bool)}
}

func (c *ServerCLI) Register(fs *flag.FlagSet) {
	fs.BoolVar(&c.ShowHelp, "help", false, "show help")
	fs.BoolVar(&c.ShowHelp, "h", false, "show help")
	fs.BoolVar(&c.NoAuth, "no-auth", false, "disable account authentication")
	fs.StringVar(&c.EnvFile, "env-file", ".env", "path to .env file in the current directory")

	fs.StringVar(&c.Host, "host", "", fmt.Sprintf("listen host (default %s)", defaultServerHost))
	fs.IntVar(&c.Port, "port", 0, fmt.Sprintf("listen port (default %d)", defaultServerPort))
	fs.StringVar(&c.Listen, "listen", "", "listen address host:port (overrides --host and --port)")

	fs.StringVar(&c.DataDir, "data", "", "data directory for database and cache")
	fs.StringVar(&c.DataDir, "data-dir", "", "alias for --data")
	fs.BoolVar(&c.NoCache, "no-cache", false, "disable track cache")
	fs.StringVar(&c.AuthSecret, "auth-secret", "", "session signing secret")
	fs.StringVar(&c.PublicURL, "public-url", "", "public URL for streams and OAuth")
	fs.BoolVar(&c.DemoMode, "demo", false, "read-only demo mode")
	fs.StringVar(&c.AllowedIPs, "allowed-ips", "", "comma-separated IPs/CIDRs allowed to connect")
	fs.BoolVar(&c.TrustProxy, "trust-proxy", false, "trust X-Forwarded-For / X-Real-IP")

	fs.StringVar(&c.NavidromeServer, "navidrome-server", "", "bootstrap Subsonic server URL")
	fs.StringVar(&c.NavidromeServer, "subsonic-server", "", "alias for --navidrome-server")
	fs.StringVar(&c.NavidromeUser, "navidrome-user", "", "bootstrap Subsonic username")
	fs.StringVar(&c.NavidromeUser, "subsonic-user", "", "alias for --navidrome-user")
	fs.StringVar(&c.NavidromePassword, "navidrome-password", "", "bootstrap Subsonic password")
	fs.StringVar(&c.NavidromePassword, "subsonic-password", "", "alias for --navidrome-password")

	fs.StringVar(&c.OIDCIssuer, "oidc-issuer", "", "OIDC issuer URL")
	fs.StringVar(&c.OIDCClientID, "oidc-client-id", "", "OIDC client ID")
	fs.StringVar(&c.OIDCClientSecret, "oidc-client-secret", "", "OIDC client secret")
	fs.StringVar(&c.OIDCRedirectURL, "oidc-redirect-url", "", "OIDC redirect URL")
	fs.StringVar(&c.OIDCScopes, "oidc-scopes", "", "OIDC scopes (space-separated)")

	fs.BoolVar(&c.LocalLibrary, "local-library", false, "enable local folder indexing")
	fs.BoolVar(&c.LocalLibraryOff, "no-local-library", false, "disable local folder indexing")
	fs.StringVar(&c.LocalLibraryPath, "local-library-path", "", "shared local music folder path")

	fs.StringVar(&c.LogLevel, "log-level", "", "log level: debug, info, warn, error")
	fs.StringVar(&c.LogFile, "log-file", "", "main log file path")
	fs.BoolVar(&c.Verbose, "verbose", false, "log level info (includes startup details)")
	fs.BoolVar(&c.Verbose, "v", false, "shorthand for --verbose")
	fs.BoolVar(&c.Debug, "debug", false, "log level debug (includes per-request HTTP logs)")
	fs.BoolVar(&c.DebugPprof, "debug-pprof", false, "enable /debug/pprof heap and CPU profiles")

	fs.IntVar(&c.ConnMinDelayMs, "conn-min-delay-ms", 0, "minimum reconnect delay")
	fs.IntVar(&c.ConnMaxDelayMs, "conn-max-delay-ms", 0, "maximum reconnect delay")
	fs.Float64Var(&c.ConnBackoffMultiplier, "conn-backoff-multiplier", 0, "reconnect backoff multiplier")
	fs.IntVar(&c.ConnHealthCheckMs, "conn-health-check-ms", 0, "connection health check interval")
	fs.IntVar(&c.ConnOfflinePollMs, "conn-offline-poll-ms", 0, "offline poll interval")
	fs.IntVar(&c.ConnSelfHealMs, "conn-self-heal-ms", 0, "self-heal refresh interval")
	fs.IntVar(&c.ConnMaxHistory, "conn-max-history", 0, "connection event history limit")

	fs.Var(updateFlag{requested: &c.Update, version: &c.UpdateVersion},
		"update", "self-update to the latest release (or the given version) and exit")
}

func (c *ServerCLI) Parse(args []string) error {
	fs := flag.NewFlagSet(brand.Slug+"-server", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	c.Register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	c.Args = fs.Args()
	fs.Visit(func(f *flag.Flag) {
		c.visited[f.Name] = true
	})
	if c.Update && c.UpdateVersion == "" && len(c.Args) > 0 {
		// `melovian-server --update v1.2.3` leaves the version positional.
		c.UpdateVersion = strings.TrimPrefix(strings.TrimSpace(c.Args[0]), "v")
		c.Args = c.Args[1:]
	}
	return nil
}

func (c *ServerCLI) Visited(name string) bool {
	return c.visited[name]
}

func (c *ServerCLI) LoadEnvFile() (bool, error) {
	if c.Visited("env-file") {
		if c.EnvFile == "" {
			return false, nil
		}
		return LoadDotEnv(c.EnvFile)
	}
	return LoadDotEnv(".env")
}

func (c *ServerCLI) Apply(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	cfg.ServerMode = true

	if c.Visited("data") || c.Visited("data-dir") {
		if strings.TrimSpace(c.DataDir) == "" {
			return fmt.Errorf("--data requires a path")
		}
		cfg.DataDir = c.DataDir
		cfg.DatabasePath = filepath.Join(c.DataDir, "melovian.db")
	}

	if err := c.applyListen(cfg); err != nil {
		return err
	}

	if c.Visited("no-cache") {
		cfg.CacheEnabled = false
	}
	if c.Visited("auth-secret") {
		cfg.AuthSecret = c.AuthSecret
	}
	if c.Visited("public-url") {
		cfg.PublicURL = strings.TrimRight(c.PublicURL, "/")
	}
	if c.Visited("demo") {
		cfg.DemoMode = c.DemoMode
	}
	if c.Visited("allowed-ips") {
		prefixes, err := parseAllowedIPs(c.AllowedIPs)
		if err != nil {
			return err
		}
		cfg.AllowedIPs = prefixes
	}
	if c.Visited("trust-proxy") {
		cfg.TrustProxy = c.TrustProxy
	}

	if c.Visited("navidrome-server") || c.Visited("subsonic-server") {
		cfg.LegacyServer = c.NavidromeServer
	}
	if c.Visited("navidrome-user") || c.Visited("subsonic-user") {
		cfg.LegacyUser = c.NavidromeUser
	}
	if c.Visited("navidrome-password") || c.Visited("subsonic-password") {
		cfg.LegacyPass = c.NavidromePassword
	}

	if c.Visited("oidc-issuer") {
		cfg.OIDC.Issuer = strings.TrimSpace(c.OIDCIssuer)
	}
	if c.Visited("oidc-client-id") {
		cfg.OIDC.ClientID = strings.TrimSpace(c.OIDCClientID)
	}
	if c.Visited("oidc-client-secret") {
		cfg.OIDC.ClientSecret = c.OIDCClientSecret
	}
	if c.Visited("oidc-redirect-url") {
		cfg.OIDC.RedirectURL = strings.TrimSpace(c.OIDCRedirectURL)
	}
	if c.Visited("oidc-scopes") {
		scopes := strings.Fields(c.OIDCScopes)
		if len(scopes) == 0 {
			scopes = []string{"openid", "profile", "email"}
		}
		cfg.OIDC.Scopes = scopes
	}

	if c.Visited("local-library") || c.Visited("no-local-library") || c.Visited("local-library-path") {
		lib := cfg.LocalLibrary
		if c.Visited("local-library") {
			lib.Enabled = c.LocalLibrary
		}
		if c.Visited("no-local-library") && c.LocalLibraryOff {
			lib.Enabled = false
		}
		if c.Visited("local-library-path") {
			lib.DefaultPath = strings.TrimSpace(c.LocalLibraryPath)
			lib.AllowCustomPath = lib.DefaultPath == ""
		}
		cfg.LocalLibrary = lib
		cfg.LocalLibraryOverride = true
	}

	if c.Visited("conn-min-delay-ms") && c.ConnMinDelayMs > 0 {
		cfg.ConnectionDefaults.MinDelayMs = c.ConnMinDelayMs
	}
	if c.Visited("conn-max-delay-ms") && c.ConnMaxDelayMs > 0 {
		cfg.ConnectionDefaults.MaxDelayMs = c.ConnMaxDelayMs
	}
	if c.Visited("conn-backoff-multiplier") && c.ConnBackoffMultiplier > 0 {
		cfg.ConnectionDefaults.BackoffMultiplier = c.ConnBackoffMultiplier
	}
	if c.Visited("conn-health-check-ms") && c.ConnHealthCheckMs > 0 {
		cfg.ConnectionDefaults.HealthCheckIntervalMs = c.ConnHealthCheckMs
	}
	if c.Visited("conn-offline-poll-ms") && c.ConnOfflinePollMs > 0 {
		cfg.ConnectionDefaults.OfflinePollMs = c.ConnOfflinePollMs
	}
	if c.Visited("conn-self-heal-ms") && c.ConnSelfHealMs > 0 {
		cfg.ConnectionDefaults.SelfHealIntervalMs = c.ConnSelfHealMs
	}
	if c.Visited("conn-max-history") && c.ConnMaxHistory > 0 {
		cfg.ConnectionDefaults.MaxHistoryEntries = c.ConnMaxHistory
	}

	if c.Visited("log-level") {
		if err := os.Setenv("MELOVIAN_LOG_LEVEL", c.LogLevel); err != nil {
			return err
		}
	}
	if c.Visited("log-file") {
		if err := os.Setenv("MELOVIAN_LOG_FILE", c.LogFile); err != nil {
			return err
		}
	}
	if c.Visited("debug-pprof") {
		cfg.DebugPprof = c.DebugPprof
	}

	return nil
}

func (c *ServerCLI) applyListen(cfg *Config) error {
	switch {
	case c.Visited("listen"):
		addr := strings.TrimSpace(c.Listen)
		if addr == "" {
			return fmt.Errorf("--listen requires host:port")
		}
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return fmt.Errorf("invalid --listen address %q: %w", addr, err)
		}
		if host == "" || port == "" {
			return fmt.Errorf("invalid --listen address %q", addr)
		}
		cfg.ListenAddr = addr
	case c.Visited("host") || c.Visited("port"):
		host := defaultServerHost
		if c.Visited("host") && strings.TrimSpace(c.Host) != "" {
			host = strings.TrimSpace(c.Host)
		}
		port := defaultServerPort
		if c.Visited("port") && c.Port > 0 {
			port = c.Port
		}
		cfg.ListenAddr = net.JoinHostPort(host, strconv.Itoa(port))
	case cfg.ListenAddr == "127.0.0.1:17337":
		cfg.ListenAddr = net.JoinHostPort(defaultServerHost, strconv.Itoa(defaultServerPort))
	}
	return nil
}

func (c *ServerCLI) ApplyDefaultLogging() error {
	switch {
	case c.Visited("log-level"):
		return nil
	case c.Visited("debug") && c.Debug:
		return os.Setenv("MELOVIAN_LOG_LEVEL", "debug")
	case c.Visited("verbose") && c.Verbose:
		return os.Setenv("MELOVIAN_LOG_LEVEL", "info")
	case os.Getenv("MELOVIAN_LOG_LEVEL") != "":
		return nil
	default:
		return os.Setenv("MELOVIAN_LOG_LEVEL", "warn")
	}
}

func (c *ServerCLI) StartupOptions() ServerStartupOptions {
	return ServerStartupOptions{NoAuth: c.NoAuth}
}

func (c *ServerCLI) PrintHelp() {
	termout.HelpTitle("Usage:")
	termout.HelpText("  " + brand.Slug + "-server [flags]")
	termout.HelpSection("Configuration precedence:")
	termout.HelpText("  flags > environment variables > .env file in the current directory")
	termout.HelpSection("Flags:")
	helpFlags := []struct {
		name string
		desc string
	}{
		{"-h, -help", "Show this help"},
		{"--host", fmt.Sprintf("Listen host (default %s)", defaultServerHost)},
		{"--port", fmt.Sprintf("Listen port (default %d)", defaultServerPort)},
		{"--listen", "Listen address host:port (overrides --host and --port)"},
		{"--env-file", "Path to .env file (default .env, skipped if missing)"},
		{"--data, --data-dir", "Data directory for database and cache"},
		{"--no-cache", "Disable track cache"},
		{"--no-auth", "Disable account authentication"},
		{"--auth-secret", "Session signing secret (auto-generated if unset)"},
		{"--public-url", "Public URL for streams and OAuth behind a reverse proxy"},
		{"--demo", "Read-only demo mode"},
		{"--allowed-ips", "Comma-separated IPs/CIDRs allowed to connect"},
		{"--trust-proxy", "Trust X-Forwarded-For / X-Real-IP for allowlist checks"},
		{"--navidrome-server, --subsonic-server", "Bootstrap Subsonic server URL"},
		{"--navidrome-user, --subsonic-user", "Bootstrap Subsonic username"},
		{"--navidrome-password, --subsonic-password", "Bootstrap Subsonic password"},
		{"--oidc-issuer", "OIDC issuer URL"},
		{"--oidc-client-id", "OIDC client ID"},
		{"--oidc-client-secret", "OIDC client secret"},
		{"--oidc-redirect-url", "OIDC redirect URL"},
		{"--oidc-scopes", "OIDC scopes (space-separated)"},
		{"--local-library", "Enable local folder indexing"},
		{"--no-local-library", "Disable local folder indexing"},
		{"--local-library-path", "Shared local music folder path"},
		{"--log-level", "Log level: debug, info, warn, error (default warn)"},
		{"--log-file", "Main log file path (default {data}/logs/melovian.log)"},
		{"-v, --verbose", "Log level info"},
		{"--debug", "Log level debug (includes per-request HTTP logs)"},
		{"--debug-pprof", "Enable /debug/pprof for heap and CPU profiles"},
		{"--conn-min-delay-ms", "Minimum reconnect delay"},
		{"--conn-max-delay-ms", "Maximum reconnect delay"},
		{"--conn-backoff-multiplier", "Reconnect backoff multiplier"},
		{"--conn-health-check-ms", "Connection health check interval"},
		{"--conn-offline-poll-ms", "Offline poll interval"},
		{"--conn-self-heal-ms", "Self-heal refresh interval"},
		{"--conn-max-history", "Connection event history limit"},
		{"--update [version]", "Self-update to the given version (default latest) and exit"},
	}
	for _, item := range helpFlags {
		termout.HelpFlag(item.name, item.desc)
	}
	termout.HelpSection("Authentication:")
	termout.HelpText(`  If --auth-secret is unset and auth is not disabled, a secure secret is
  generated at startup and printed once. Save it to persist sessions across
  restarts.`)
	termout.HelpSection("Environment variables:")
	termout.HelpText("  All flags have MELOVIAN_* or NAVIDROME_* equivalents. See README.md.")
	_, _ = fmt.Fprintln(os.Stdout)
}
