// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package notify

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"melovian/internal/api/apishared"
	"melovian/internal/api/realtime"
	"melovian/internal/appconfig"
	"melovian/internal/httputil"
	"melovian/internal/store"
)

const (
	EventNotificationCreated = "notification.created"
	EventNotificationUpdated = "notification.updated"
)

// Handler serves notification routes and pushes notification events.
type Handler struct {
	notifications *store.NotificationStore
	events        *realtime.EventHub
	cfg           appconfig.Config
}

func New(notifications *store.NotificationStore, events *realtime.EventHub, cfg appconfig.Config) *Handler {
	return &Handler{notifications: notifications, events: events, cfg: cfg}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/notifications", h.handleListNotifications)
	mux.HandleFunc("GET /api/notifications/unread-count", h.handleNotificationUnreadCount)
	mux.HandleFunc("POST /api/notifications/{id}/read", h.handleMarkNotificationRead)
	mux.HandleFunc("POST /api/notifications/read-all", h.handleMarkAllNotificationsRead)
	mux.HandleFunc("DELETE /api/notifications/{id}", h.handleDeleteNotification)
}

func (h *Handler) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	userID := apishared.UserIDFromContext(r.Context())
	if userID == "" || !h.cfg.AuthEnabled() {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"items": []any{}, "unread": 0})
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := h.notifications.List(userID, limit, offset)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleListNotifications", err)
		return
	}
	unread, err := h.notifications.UnreadCount(userID)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleListNotifications", err)
		return
	}
	out := make([]map[string]any, len(items))
	for i, item := range items {
		out[i] = notificationJSON(item)
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"items": out, "unread": unread})
}

func (h *Handler) handleNotificationUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := apishared.UserIDFromContext(r.Context())
	if userID == "" || !h.cfg.AuthEnabled() {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"unread": 0})
		return
	}
	unread, err := h.notifications.UnreadCount(userID)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleNotificationUnreadCount", err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"unread": unread})
}

func (h *Handler) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	userID := apishared.UserIDFromContext(r.Context())
	if userID == "" || !h.cfg.AuthEnabled() {
		httputil.WriteError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return
	}
	item, err := h.notifications.MarkRead(userID, r.PathValue("id"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		httputil.WriteInternalError(w, r, "handleMarkNotificationRead", err)
		return
	}
	h.pushNotificationUpdated(userID, item)
	httputil.WriteJSON(w, http.StatusOK, notificationJSON(item))
}

func (h *Handler) handleMarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	userID := apishared.UserIDFromContext(r.Context())
	if userID == "" || !h.cfg.AuthEnabled() {
		httputil.WriteError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return
	}
	n, err := h.notifications.MarkAllRead(userID)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleMarkAllNotificationsRead", err)
		return
	}
	h.events.BroadcastToUser(userID, realtime.Event{
		Type: EventNotificationUpdated,
		Payload: map[string]any{
			"action": "read_all",
			"count":  n,
		},
	})
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"updated": n})
}

func (h *Handler) handleDeleteNotification(w http.ResponseWriter, r *http.Request) {
	userID := apishared.UserIDFromContext(r.Context())
	if userID == "" || !h.cfg.AuthEnabled() {
		httputil.WriteError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return
	}
	if err := h.notifications.Delete(userID, r.PathValue("id")); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		httputil.WriteInternalError(w, r, "handleDeleteNotification", err)
		return
	}
	h.events.BroadcastToUser(userID, realtime.Event{
		Type: EventNotificationUpdated,
		Payload: map[string]any{
			"action": "deleted",
			"id":     r.PathValue("id"),
		},
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Notify(input store.CreateNotificationInput) (store.Notification, error) {
	if h.notifications == nil {
		return store.Notification{}, errors.New("notifications unavailable")
	}
	item, err := h.notifications.Create(input)
	if err != nil {
		return store.Notification{}, err
	}
	h.pushNotificationCreated(item)
	return item, nil
}

func (h *Handler) pushNotificationCreated(item store.Notification) {
	if h.events == nil {
		return
	}
	h.events.BroadcastToUser(item.UserID, realtime.Event{
		Type:    EventNotificationCreated,
		Payload: notificationJSON(item),
	})
}

func (h *Handler) pushNotificationUpdated(userID string, item store.Notification) {
	if h.events == nil {
		return
	}
	h.events.BroadcastToUser(userID, realtime.Event{
		Type: EventNotificationUpdated,
		Payload: map[string]any{
			"action":       "read",
			"notification": notificationJSON(item),
		},
	})
}

func notificationJSON(item store.Notification) map[string]any {
	out := map[string]any{
		"id":        item.ID,
		"kind":      item.Kind,
		"title":     item.Title,
		"body":      item.Body,
		"href":      item.Href,
		"createdAt": item.CreatedAt.UTC().Format(time.RFC3339),
		"read":      item.ReadAt != nil,
	}
	if item.ReadAt != nil {
		out["readAt"] = item.ReadAt.UTC().Format(time.RFC3339)
	}
	if strings.TrimSpace(item.Payload) != "" {
		var payload any
		if err := json.Unmarshal([]byte(item.Payload), &payload); err == nil {
			out["payload"] = payload
		} else {
			out["payload"] = item.Payload
		}
	}
	return out
}
