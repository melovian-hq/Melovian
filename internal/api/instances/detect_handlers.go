// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package instances

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"

	"melovian/internal/httputil"
)

const (
	// detectProbeTimeout caps each individual ping probe.
	detectProbeTimeout = 1500 * time.Millisecond
	// detectTotalBudget caps the whole probe pass across all candidates.
	detectTotalBudget = 2500 * time.Millisecond
	// detectCacheTTL keeps repeated mounts from hammering the network.
	detectCacheTTL = 30 * time.Second
)

// detectedServer is one reachable Subsonic-compatible host.
type detectedServer struct {
	URL        string `json:"url"`
	ServerName string `json:"serverName"`
	Version    string `json:"version"`
	Reachable  bool   `json:"reachable"`
}

// detectCandidateURLs is a var so tests can inject httptest servers.
var detectCandidateURLs = defaultDetectCandidates

// detectProbeClient is a var so tests can swap transports if needed. The
// redirect guard keeps probes on the original host or another loopback
// target so a hostile local service cannot bounce the client at internal
// network hosts.
var detectProbeClient = &http.Client{
	Timeout: detectProbeTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) == 0 {
			return nil
		}
		prev := via[len(via)-1].URL
		if strings.EqualFold(req.URL.Hostname(), prev.Hostname()) || isLoopbackHostname(req.URL.Hostname()) {
			return nil
		}
		return fmt.Errorf("refusing redirect to non-loopback host %q", req.URL.Hostname())
	},
}

// isLoopbackHostname reports whether host is localhost or a loopback IP.
func isLoopbackHostname(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	addr, err := netip.ParseAddr(host)
	return err == nil && addr.IsLoopback()
}

var detectResultCache struct {
	mu      sync.Mutex
	servers []detectedServer
	at      time.Time
}

// defaultDetectCandidates lists loopback addresses where Navidrome (4533)
// and Subsonic (4040) commonly run. Extra URLs come from
// MELOVIAN_DETECT_EXTRA_URLS for admin-managed setups.
func defaultDetectCandidates() []string {
	candidates := []string{
		"http://127.0.0.1:4533",
		"http://localhost:4533",
		"http://[::1]:4533",
	}
	if detectInContainer() {
		candidates = append(candidates, "http://host.docker.internal:4533")
	}
	candidates = append(candidates,
		"http://127.0.0.1:4040",
		"http://localhost:4040",
	)
	candidates = append(candidates, extraDetectCandidates()...)
	return dedupeCandidates(candidates)
}

// extraDetectCandidates parses MELOVIAN_DETECT_EXTRA_URLS, a comma-separated
// list of http(s) base URLs. Entries with credentials or other schemes are
// skipped so the probe list stays free of userinfo.
func extraDetectCandidates() []string {
	raw := strings.TrimSpace(os.Getenv("MELOVIAN_DETECT_EXTRA_URLS"))
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		u, err := httputil.ParseHTTPURL(part)
		if err != nil || u.User != nil {
			continue
		}
		out = append(out, strings.TrimRight(part, "/"))
	}
	return out
}

func dedupeCandidates(candidates []string) []string {
	seen := make(map[string]struct{}, len(candidates))
	out := candidates[:0]
	for _, c := range candidates {
		key := strings.TrimRight(c, "/")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

// detectInContainer mirrors inContainer in the system update handler so
// host.docker.internal is only probed where it resolves.
func detectInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	for _, k := range []string{"MELOVIAN_CONTAINER", "container", "KUBERNETES_SERVICE_HOST"} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}

// probeSubsonic pings base without credentials. Any JSON subsonic-response
// envelope with a status field counts, including auth failures, because
// OpenSubsonic servers still emit type and serverVersion there.
func probeSubsonic(ctx context.Context, client *http.Client, base string) (detectedServer, bool) {
	endpoint := strings.TrimRight(base, "/") + "/rest/ping.view?f=json&u=detect&p=detect"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return detectedServer{}, false
	}
	resp, err := client.Do(req)
	if err != nil {
		return detectedServer{}, false
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := httputil.ReadLimited(resp.Body, 1<<20)
	if err != nil {
		return detectedServer{}, false
	}

	var payload struct {
		SubsonicResponse struct {
			Status        string `json:"status"`
			Version       string `json:"version"`
			Type          string `json:"type"`
			ServerVersion string `json:"serverVersion"`
			OpenSubsonic  bool   `json:"openSubsonic"`
			Server        struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"server"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return detectedServer{}, false
	}
	env := payload.SubsonicResponse
	if env.Status == "" {
		return detectedServer{}, false
	}

	name := env.Type
	if name == "" {
		name = env.Server.Name
	}
	if name == "" {
		name = "Subsonic"
	}
	version := env.ServerVersion
	if version == "" {
		version = env.Server.Version
	}
	if version == "" {
		version = env.Version
	}
	return detectedServer{
		URL:        base,
		ServerName: name,
		Version:    version,
		Reachable:  true,
	}, true
}

// detectSubsonicServers probes all candidates concurrently and returns the
// reachable ones in candidate order. Results are cached briefly.
func detectSubsonicServers(ctx context.Context) []detectedServer {
	detectResultCache.mu.Lock()
	defer detectResultCache.mu.Unlock()
	// Requests queued behind the lock may already be dead; probing for them
	// wastes the budget and would poison the cache with failures.
	if ctx.Err() != nil {
		return nil
	}
	if time.Since(detectResultCache.at) < detectCacheTTL {
		return detectResultCache.servers
	}

	ctx, cancel := context.WithTimeout(ctx, detectTotalBudget)
	defer cancel()

	candidates := detectCandidateURLs()
	results := make([]detectedServer, len(candidates))
	var wg sync.WaitGroup
	for i, base := range candidates {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if server, ok := probeSubsonic(ctx, detectProbeClient, base); ok {
				results[i] = server
			}
		}()
	}
	wg.Wait()

	servers := make([]detectedServer, 0, len(results))
	for _, server := range results {
		if server.Reachable {
			servers = append(servers, server)
		}
	}
	// A cancelled context makes every probe fail, so keep that result out of
	// the cache and let the next live request probe again.
	if ctx.Err() == nil {
		detectResultCache.servers = servers
		detectResultCache.at = time.Now()
	}
	return servers
}

func (h *Handler) handleDetectInstances(w http.ResponseWriter, r *http.Request) {
	servers := detectSubsonicServers(r.Context())
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"servers": servers})
}
