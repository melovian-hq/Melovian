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
	mux.HandleFunc("GET /api/extensions/{id}/script", h.handleExtensionScript)
	mux.HandleFunc("GET /api/extensions/{id}/assets/{path...}", h.handleExtensionAsset)
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
	HasScript   bool   `json:"hasScript"`
	ScriptSafe  bool   `json:"scriptSafe"`
	HasWasm     bool   `json:"hasWasm"`
	IconURL     string `json:"iconUrl,omitempty"`
	ImageURL    string `json:"imageUrl,omitempty"`
	AppTheme    string `json:"appTheme,omitempty"`
	InstalledAt string `json:"installedAt,omitempty"`
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
			HasScript:   strings.TrimSpace(item.Manifest.Script) != "",
			ScriptSafe:  scriptSafe,
			HasWasm:     ext.HasWasm(item.Dir),
			IconURL:     extensionAssetURL(item.Manifest.ID, item.Manifest.Icon),
			ImageURL:    extensionAssetURL(item.Manifest.ID, item.Manifest.Image),
			AppTheme:    strings.TrimSpace(item.Manifest.AppTheme),
			InstalledAt: item.InstalledAt,
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
