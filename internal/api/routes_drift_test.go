// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type apiRoute struct {
	Method   string `json:"method,omitempty"`
	Pattern  string `json:"pattern"`
	Prefix   bool   `json:"prefix,omitempty"`
	Requires string `json:"requires,omitempty"`
}

type apiRouteManifest struct {
	Routes []apiRoute `json:"routes"`
}

var (
	routeRegistrationRE = regexp.MustCompile(`\.HandleFunc\("([A-Z]+) ([^"]+)"|\.Handle\("([A-Z]+) ([^"]+)"`)
	bareHandleRE        = regexp.MustCompile(`\.Handle\("(/[^"]+)"`)
)

func loadAPIRouteManifest(t *testing.T) apiRouteManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "api-routes.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest apiRouteManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	return manifest
}

func collectRoutesFromGoSource(t *testing.T) []apiRoute {
	t.Helper()
	dir := "."

	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk api dir: %v", err)
	}
	sort.Strings(files)

	var routes []apiRoute
	seen := make(map[string]struct{})
	for _, file := range files {
		content, readErr := os.ReadFile(file)
		if readErr != nil {
			t.Fatalf("read %s: %v", file, readErr)
		}
		source := string(content)
		for _, match := range routeRegistrationRE.FindAllStringSubmatch(source, -1) {
			method := match[1]
			pattern := match[2]
			if method == "" {
				method = match[3]
				pattern = match[4]
			}
			addRoute(&routes, seen, apiRoute{Method: method, Pattern: pattern})
		}
		for _, match := range bareHandleRE.FindAllStringSubmatch(source, -1) {
			pattern := match[1]
			addRoute(&routes, seen, apiRoute{Pattern: pattern, Prefix: true})
		}
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Pattern == routes[j].Pattern {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Pattern < routes[j].Pattern
	})
	return routes
}

func addRoute(routes *[]apiRoute, seen map[string]struct{}, route apiRoute) {
	key := routeKey(route)
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*routes = append(*routes, route)
}

func routeKey(r apiRoute) string {
	if r.Prefix {
		return "prefix " + r.Pattern
	}
	return r.Method + " " + r.Pattern
}

func TestAPIRoutesMatchGoSource(t *testing.T) {
	manifest := loadAPIRouteManifest(t)
	sourceRoutes := collectRoutesFromGoSource(t)

	manifestByKey := make(map[string]apiRoute, len(manifest.Routes))
	for _, r := range manifest.Routes {
		manifestByKey[routeKey(r)] = r
	}
	sourceByKey := make(map[string]apiRoute, len(sourceRoutes))
	for _, r := range sourceRoutes {
		sourceByKey[routeKey(r)] = r
	}

	var missingInManifest []string
	for key := range sourceByKey {
		if _, ok := manifestByKey[key]; !ok {
			missingInManifest = append(missingInManifest, key)
		}
	}
	var missingInSource []string
	for key := range manifestByKey {
		if _, ok := sourceByKey[key]; !ok {
			missingInSource = append(missingInSource, key)
		}
	}
	sort.Strings(missingInManifest)
	sort.Strings(missingInSource)

	if len(missingInManifest) > 0 || len(missingInSource) > 0 {
		var b strings.Builder
		b.WriteString("API route manifest drift detected.\n")
		if len(missingInManifest) > 0 {
			b.WriteString("Registered in Go but missing from testdata/api-routes.json:\n")
			for _, line := range missingInManifest {
				b.WriteString("  - " + line + "\n")
			}
		}
		if len(missingInSource) > 0 {
			b.WriteString("In testdata/api-routes.json but not registered in Go handlers:\n")
			for _, line := range missingInSource {
				b.WriteString("  - " + line + "\n")
			}
		}
		b.WriteString("Update internal/api/testdata/api-routes.json to match handler registrations.")
		t.Fatal(b.String())
	}
}

func substitutePathParams(pattern string) string {
	re := regexp.MustCompile(`\{[^}]+\}`)
	return re.ReplaceAllString(pattern, "test-id")
}

func TestAPIRoutesAreRegistered(t *testing.T) {
	manifest := loadAPIRouteManifest(t)
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	for _, route := range manifest.Routes {
		if route.Prefix || route.Requires == "oidc" || route.Requires == "debug-pprof" {
			continue
		}
		path := substitutePathParams(route.Pattern)
		req := httptest.NewRequest("PROPFIND", path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Errorf("route %s %s is not registered (path %s returned 404)", route.Method, route.Pattern, path)
		}
	}
}
