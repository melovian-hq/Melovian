// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"melovian/internal/httputil"
	"melovian/internal/store"
)

const (
	EventNotificationCreated = "notification.created"
	EventNotificationUpdated = "notification.updated"
)

func (s *Server) registerNotificationRoutes() {
	s.mux.HandleFunc("GET /api/notifications", s.handleListNotifications)
	s.mux.HandleFunc("GET /api/notifications/unread-count", s.handleNotificationUnreadCount)
	s.mux.HandleFunc("POST /api/notifications/{id}/read", s.handleMarkNotificationRead)
	s.mux.HandleFunc("POST /api/notifications/read-all", s.handleMarkAllNotificationsRead)
	s.mux.HandleFunc("DELETE /api/notifications/{id}", s.handleDeleteNotification)
}

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" || !s.cfg.AuthEnabled() {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"items": []any{}, "unread": 0})
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := s.notifications.List(userID, limit, offset)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleListNotifications", err)
		return
	}
	unread, err := s.notifications.UnreadCount(userID)
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

func (s *Server) handleNotificationUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" || !s.cfg.AuthEnabled() {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"unread": 0})
		return
	}
	unread, err := s.notifications.UnreadCount(userID)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleNotificationUnreadCount", err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"unread": unread})
}

func (s *Server) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" || !s.cfg.AuthEnabled() {
		httputil.WriteError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return
	}
	item, err := s.notifications.MarkRead(userID, r.PathValue("id"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		httputil.WriteInternalError(w, r, "handleMarkNotificationRead", err)
		return
	}
	s.pushNotificationUpdated(userID, item)
	httputil.WriteJSON(w, http.StatusOK, notificationJSON(item))
}

func (s *Server) handleMarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" || !s.cfg.AuthEnabled() {
		httputil.WriteError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return
	}
	n, err := s.notifications.MarkAllRead(userID)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleMarkAllNotificationsRead", err)
		return
	}
	s.events.BroadcastToUser(userID, Event{
		Type: EventNotificationUpdated,
		Payload: map[string]any{
			"action": "read_all",
			"count":  n,
		},
	})
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"updated": n})
}

func (s *Server) handleDeleteNotification(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" || !s.cfg.AuthEnabled() {
		httputil.WriteError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return
	}
	if err := s.notifications.Delete(userID, r.PathValue("id")); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		httputil.WriteInternalError(w, r, "handleDeleteNotification", err)
		return
	}
	s.events.BroadcastToUser(userID, Event{
		Type: EventNotificationUpdated,
		Payload: map[string]any{
			"action": "deleted",
			"id":     r.PathValue("id"),
		},
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) notifyUser(input store.CreateNotificationInput) (store.Notification, error) {
	if s.notifications == nil {
		return store.Notification{}, errors.New("notifications unavailable")
	}
	item, err := s.notifications.Create(input)
	if err != nil {
		return store.Notification{}, err
	}
	s.pushNotificationCreated(item)
	return item, nil
}

func (s *Server) pushNotificationCreated(item store.Notification) {
	if s.events == nil {
		return
	}
	s.events.BroadcastToUser(item.UserID, Event{
		Type:    EventNotificationCreated,
		Payload: notificationJSON(item),
	})
}

func (s *Server) pushNotificationUpdated(userID string, item store.Notification) {
	if s.events == nil {
		return
	}
	s.events.BroadcastToUser(userID, Event{
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
