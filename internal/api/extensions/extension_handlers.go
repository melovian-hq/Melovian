// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"io"
	"net/http"
	"strings"

	"melovian/internal/appconfig"
	ext "melovian/internal/extensions"
	"melovian/internal/httputil"
)

// Handler serves the extension management and asset routes.
type Handler struct {
	cfg appconfig.Config
}

func New(cfg appconfig.Config) *Handler {
	return &Handler{cfg: cfg}
}

func (h *Handler) Register(mux *http.ServeMux) {
	h.registerExtensionRoutes(mux)
}

func (h *Handler) registerExtensionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/extensions", h.handleListExtensions)
	mux.HandleFunc("POST /api/extensions/install", h.handleInstallExtension)
	mux.HandleFunc("POST /api/extensions/install-dir", h.handleInstallDir)
	mux.HandleFunc("GET /api/extensions/registry", h.handleExtensionRegistry)
	mux.HandleFunc("PUT /api/extensions/registry", h.handlePutRegistry)
	mux.HandleFunc("DELETE /api/extensions/registry", h.handleDeleteRegistry)
	mux.HandleFunc("POST /api/extensions/install-remote", h.handleInstallRemoteExtension)
	mux.HandleFunc("GET /api/extensions/{id}/script", h.handleExtensionScript)
	mux.HandleFunc("GET /api/extensions/{id}/assets/{path...}", h.handleExtensionAsset)
	mux.HandleFunc("GET /api/extensions/{id}/settings", h.handleGetExtensionSettings)
	mux.HandleFunc("PUT /api/extensions/{id}/settings", h.handlePutExtensionSettings)
	mux.HandleFunc("PUT /api/extensions/{id}/enabled", h.handleSetExtensionEnabled)
	mux.HandleFunc("DELETE /api/extensions/{id}", h.handleUninstallExtension)
	mux.HandleFunc("POST /api/extensions/{id}/reinstall", h.handleReinstallExtension)
}

type extensionEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

type extensionListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
	Enabled     bool   `json:"enabled"`
	Installed   bool   `json:"installed"`
	Bundled     bool   `json:"bundled"`
	Dev         bool   `json:"dev,omitempty"`
	HasScript   bool   `json:"hasScript"`
	ScriptSafe  bool   `json:"scriptSafe"`
	HasWasm     bool   `json:"hasWasm"`
	IconURL     string `json:"iconUrl,omitempty"`
	ImageURL    string `json:"imageUrl,omitempty"`
	AppTheme    string `json:"appTheme,omitempty"`
	InstalledAt string `json:"installedAt,omitempty"`
	// Settings carries the current user-configured values merged over
	// manifest defaults. Nil when the extension declares no settings.
	Settings map[string]any `json:"settings,omitempty"`
}

func extensionAssetURL(id, rel string) string {
	rel = strings.TrimSpace(strings.ReplaceAll(rel, "\\", "/"))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return ""
	}
	return "/api/extensions/" + id + "/assets/" + rel
}

func (h *Handler) buildExtensionList() ([]extensionListItem, []ext.Manifest, error) {
	items, err := ext.List(h.cfg.DataDir)
	if err != nil {
		return nil, nil, err
	}
	out := make([]extensionListItem, 0, len(items)+4)
	seen := map[string]bool{}
	manifests := make([]ext.Manifest, 0, len(items))

	for _, item := range items {
		seen[item.Manifest.ID] = true
		scriptSafe := false
		if strings.TrimSpace(item.Manifest.Script) != "" {
			if data, _, err := ext.ReadScript(h.cfg.DataDir, item.Manifest.ID); err == nil {
				scriptSafe = ext.ScriptAllowed(string(data))
			}
		}
		out = append(out, extensionListItem{
			ID:          item.Manifest.ID,
			Name:        item.Manifest.Name,
			Version:     item.Manifest.Version,
			Description: item.Manifest.Description,
			Author:      item.Manifest.Author,
			Enabled:     item.Enabled,
			Installed:   true,
			Bundled:     ext.IsBundled(item.Manifest.ID),
			Dev:         item.Dev,
			HasScript:   strings.TrimSpace(item.Manifest.Script) != "",
			ScriptSafe:  scriptSafe,
			HasWasm:     ext.HasWasm(item.Dir),
			IconURL:     extensionAssetURL(item.Manifest.ID, item.Manifest.Icon),
			ImageURL:    extensionAssetURL(item.Manifest.ID, item.Manifest.Image),
			AppTheme:    strings.TrimSpace(item.Manifest.AppTheme),
			InstalledAt: item.InstalledAt,
			Settings:    ext.LoadSettings(h.cfg.DataDir, item.Manifest),
		})
		if !item.Enabled {
			continue
		}
		manifest := item.Manifest
		if strings.TrimSpace(manifest.Script) != "" {
			if data, _, err := ext.ReadScript(h.cfg.DataDir, manifest.ID); err != nil || !ext.ScriptAllowed(string(data)) {
				manifest.Script = ""
			}
		}
		manifests = append(manifests, manifest)
	}

	bundledIDs, err := ext.BundledIDs()
	if err != nil {
		return nil, nil, err
	}
	for _, id := range bundledIDs {
		if seen[id] {
			continue
		}
		manifest, err := ext.ReadBundledManifest(id)
		if err != nil {
			continue
		}
		out = append(out, extensionListItem{
			ID:          manifest.ID,
			Name:        manifest.Name,
			Version:     manifest.Version,
			Description: manifest.Description,
			Author:      manifest.Author,
			Enabled:     false,
			Installed:   false,
			Bundled:     true,
			HasScript:   strings.TrimSpace(manifest.Script) != "",
			ScriptSafe:  false,
			HasWasm:     false,
			IconURL:     "",
			ImageURL:    "",
			AppTheme:    strings.TrimSpace(manifest.AppTheme),
		})
	}
	return out, manifests, nil
}

func (h *Handler) writeExtensionList(w http.ResponseWriter, r *http.Request) {
	out, manifests, err := h.buildExtensionList()
	if err != nil {
		httputil.WriteInternalError(w, r, "list extensions", err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"items":     out,
		"manifests": manifests,
		"dir":       ext.ExtensionsDir(h.cfg.DataDir),
	})
}

func (h *Handler) handleListExtensions(w http.ResponseWriter, r *http.Request) {
	h.writeExtensionList(w, r)
}

func (h *Handler) handleExtensionScript(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing_id", "missing id")
		return
	}
	data, contentType, err := ext.ReadScript(h.cfg.DataDir, id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	if err := ext.ValidateScript(string(data)); err != nil {
		httputil.WriteError(w, http.StatusForbidden, "script_blocked_by_sandbox_policy", "script blocked by sandbox policy")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data) //#nosec G705 -- validated extension script with explicit JS content type
}

func (h *Handler) handleExtensionAsset(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	rel := strings.TrimSpace(r.PathValue("path"))
	if id == "" || rel == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing_path", "missing path")
		return
	}
	full, err := ext.ResolveAssetPath(h.cfg.DataDir, id, rel)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, full) //#nosec G304 G703 -- path from ResolveAssetPath under extension dir
}

func (h *Handler) handleSetExtensionEnabled(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing_id", "missing id")
		return
	}
	var req extensionEnabledRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := ext.SetEnabled(h.cfg.DataDir, id, req.Enabled); err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	h.writeExtensionList(w, r)
}

func (h *Handler) handleInstallExtension(w http.ResponseWriter, r *http.Request) {
	const maxMemory = 32 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(strings.ToLower(contentType), "multipart/") {
		if err := r.ParseMultipartForm(maxMemory); err != nil { //#nosec G120 -- body bounded by MaxBytesReader above
			httputil.WriteError(w, http.StatusBadRequest, "invalid_package", "could not parse multipart upload")
			return
		}
		file, header, err := r.FormFile("package")
		if err != nil {
			file, header, err = r.FormFile("file")
		}
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "missing_file", "upload a zip as package or file")
			return
		}
		defer file.Close()
		name := strings.ToLower(header.Filename)
		if !strings.HasSuffix(name, ".zip") {
			httputil.WriteError(w, http.StatusBadRequest, "invalid_package", "only .zip packages are supported")
			return
		}
		data, err := io.ReadAll(io.LimitReader(file, maxMemory+1))
		if err != nil || len(data) == 0 || len(data) > maxMemory {
			httputil.WriteError(w, http.StatusBadRequest, "invalid_package", "could not read zip")
			return
		}
		if _, err := ext.InstallFromZip(h.cfg.DataDir, data); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "install_failed", err.Error())
			return
		}
		h.writeExtensionList(w, r)
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil || len(data) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_package", "expected multipart file or zip body")
		return
	}
	if _, err := ext.InstallFromZip(h.cfg.DataDir, data); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "install_failed", err.Error())
		return
	}
	h.writeExtensionList(w, r)
}

type installDirRequest struct {
	Path string `json:"path"`
}

// handleInstallDir links a local extension directory for development.
// The path is local machine state, not remote input, so no signature is
// involved; scripts still face the sandbox and the scriptSafe gate.
func (h *Handler) handleInstallDir(w http.ResponseWriter, r *http.Request) {
	var req installDirRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if _, err := ext.InstallFromDir(h.cfg.DataDir, req.Path); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "install_failed", err.Error())
		return
	}
	h.writeExtensionList(w, r)
}

type registryListItem struct {
	ID              string                  `json:"id"`
	Name            string                  `json:"name"`
	Version         string                  `json:"version"`
	Description     string                  `json:"description,omitempty"`
	Author          string                  `json:"author,omitempty"`
	Homepage        string                  `json:"homepage,omitempty"`
	License         string                  `json:"license,omitempty"`
	Tags            []string                `json:"tags,omitempty"`
	Risk            string                  `json:"risk,omitempty"`
	ExternalURLs    []string                `json:"externalUrls,omitempty"`
	Permissions     []string                `json:"permissions,omitempty"`
	MinAppVersion   string                  `json:"minAppVersion,omitempty"`
	Requires        []string                `json:"requires,omitempty"`
	Delisted        *ext.RegistryDelisted   `json:"delisted,omitempty"`
	Versions        []ext.RegistryVersion   `json:"versions,omitempty"`
	IconURL         string                  `json:"iconUrl,omitempty"`
	ImageURL        string                  `json:"imageUrl,omitempty"`
	PackageURL      string                  `json:"packageUrl,omitempty"`
	SHA256          string                  `json:"sha256,omitempty"`
	Bytes           int64                   `json:"bytes,omitempty"`
	HasScript       bool                    `json:"hasScript"`
	HasWasm         bool                    `json:"hasWasm"`
	Styles          int                     `json:"styles,omitempty"`
	AppTheme        bool                    `json:"appTheme,omitempty"`
	TrackRules      int                     `json:"trackRules,omitempty"`
	PlayerHooks     int                     `json:"playerHooks,omitempty"`
	AuditStatus     string                  `json:"auditStatus,omitempty"`
	AuditWarnings   []string                `json:"auditWarnings,omitempty"`
	Changelog       []ext.RegistryChangelog `json:"changelog,omitempty"`
	Installed       bool                    `json:"installed"`
	InstalledVer    string                  `json:"installedVersion,omitempty"`
	Enabled         bool                    `json:"enabled"`
	UpdateAvailable bool                    `json:"updateAvailable"`
}

type installRemoteRequest struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

func (h *Handler) handleExtensionRegistry(w http.ResponseWriter, r *http.Request) {
	index, indexURL, verified, err := ext.FetchRegistry(r.Context(), h.cfg.DataDir)
	if err != nil {
		httputil.WriteError(w, http.StatusBadGateway, "registry_unavailable", err.Error())
		return
	}
	installed := map[string]ext.Entry{}
	if items, err := ext.List(h.cfg.DataDir); err == nil {
		for _, item := range items {
			installed[item.Manifest.ID] = item
		}
	}
	items := make([]registryListItem, 0, len(index.Extensions))
	for _, entry := range index.Extensions {
		item := registryListItem{
			ID:            entry.ID,
			Name:          entry.Name,
			Version:       entry.Version,
			Description:   entry.Description,
			Author:        entry.Author,
			Homepage:      entry.Homepage,
			License:       entry.License,
			Tags:          entry.Tags,
			Risk:          entry.Risk,
			ExternalURLs:  entry.ExternalURLs,
			Permissions:   entry.Permissions,
			MinAppVersion: entry.MinAppVersion,
			Requires:      entry.Requires,
			Delisted:      entry.Delisted,
			Versions:      entry.Versions,
			IconURL:       ext.ResolveRegistryAsset(indexURL, entry.Icon),
			ImageURL:      ext.ResolveRegistryAsset(indexURL, entry.Image),
			PackageURL:    entry.Package.URL,
			SHA256:        entry.Package.SHA256,
			Bytes:         entry.Package.Bytes,
			HasScript:     entry.Capabilities.Script,
			HasWasm:       entry.Capabilities.Wasm,
			Styles:        entry.Capabilities.Styles,
			AppTheme:      entry.Capabilities.AppTheme,
			TrackRules:    entry.Capabilities.TrackRules,
			PlayerHooks:   entry.Capabilities.PlayerHooks,
			AuditStatus:   entry.Audit.Status,
			AuditWarnings: entry.Audit.Warnings,
			Changelog:     entry.Changelog,
		}
		if cur, ok := installed[entry.ID]; ok {
			item.Installed = true
			item.Enabled = cur.Enabled
			item.InstalledVer = cur.Manifest.Version
			item.UpdateAvailable = ext.CompareVersions(entry.Version, cur.Manifest.Version) > 0
		}
		items = append(items, item)
	}
	custom := false
	if o, ok := ext.LoadRegistryOverride(h.cfg.DataDir); ok && o.URL == indexURL {
		custom = true
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"url":         indexURL,
		"generatedAt": index.GeneratedAt,
		"signed":      verified,
		"custom":      custom,
		"items":       items,
	})
}

type registryConfigRequest struct {
	URL  string   `json:"url"`
	Keys []string `json:"keys"`
}

func (h *Handler) handlePutRegistry(w http.ResponseWriter, r *http.Request) {
	var req registryConfigRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := ext.SaveRegistryOverride(h.cfg.DataDir, req.URL, req.Keys); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_registry", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleDeleteRegistry(w http.ResponseWriter, r *http.Request) {
	if err := ext.ClearRegistryOverride(h.cfg.DataDir); err != nil {
		httputil.WriteInternalError(w, r, "clear registry override", err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleInstallRemoteExtension(w http.ResponseWriter, r *http.Request) {
	var req installRemoteRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	id := strings.TrimSpace(req.ID)
	if !ext.IsValidExtensionID(id) {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid extension id")
		return
	}
	// The package URL comes from the trusted index, not the request, so
	// clients cannot turn this endpoint into an arbitrary fetch. Version
	// only selects which published entry to install.
	if _, err := ext.InstallFromRegistry(r.Context(), h.cfg.DataDir, id, strings.TrimSpace(req.Version)); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "install_failed", err.Error())
		return
	}
	h.writeExtensionList(w, r)
}

func (h *Handler) extensionManifestFor(w http.ResponseWriter, r *http.Request) (ext.Manifest, bool) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing_id", "missing id")
		return ext.Manifest{}, false
	}
	entry, err := ext.FindInstalled(h.cfg.DataDir, id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "extension not installed")
		return ext.Manifest{}, false
	}
	return entry.Manifest, true
}

func (h *Handler) handleGetExtensionSettings(w http.ResponseWriter, r *http.Request) {
	manifest, ok := h.extensionManifestFor(w, r)
	if !ok {
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"schema":   manifest.Settings,
		"settings": ext.LoadSettings(h.cfg.DataDir, manifest),
	})
}

func (h *Handler) handlePutExtensionSettings(w http.ResponseWriter, r *http.Request) {
	manifest, ok := h.extensionManifestFor(w, r)
	if !ok {
		return
	}
	var req struct {
		Settings map[string]any `json:"settings"`
	}
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := ext.SaveSettings(h.cfg.DataDir, manifest, req.Settings); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_settings", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"settings": ext.LoadSettings(h.cfg.DataDir, manifest),
	})
}

func (h *Handler) handleUninstallExtension(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing_id", "missing id")
		return
	}
	if err := ext.Uninstall(h.cfg.DataDir, id); err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	h.writeExtensionList(w, r)
}

func (h *Handler) handleReinstallExtension(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing_id", "missing id")
		return
	}
	if !ext.IsBundled(id) {
		httputil.WriteError(w, http.StatusBadRequest, "not_bundled", "only bundled extensions can be reinstalled this way")
		return
	}
	if err := ext.ReinstallBundled(h.cfg.DataDir, id); err != nil {
		httputil.WriteInternalError(w, r, "reinstall bundled extension", err)
		return
	}
	h.writeExtensionList(w, r)
}
