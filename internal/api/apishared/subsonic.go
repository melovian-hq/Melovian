// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package apishared

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"sync"

	"melovian/internal/httputil"
	"melovian/internal/store"
	"melovian/internal/subsonic"
)

// Resolver owns the active Subsonic client and the per-instance client
// cache. It replaces the client fields that used to live on api.Server.
type Resolver struct {
	mu        sync.RWMutex
	instances *store.InstanceStore
	active    *subsonic.Client
	cache     map[string]*subsonic.Client
}

func NewResolver(instances *store.InstanceStore) *Resolver {
	return &Resolver{
		instances: instances,
		cache:     make(map[string]*subsonic.Client),
	}
}

func (r *Resolver) ResolveInstanceID(req *http.Request) (string, error) {
	headerValue := InstanceIDFromRequest(req)
	userID := UserIDFromContext(req.Context())

	if headerValue != "" {
		if _, err := r.instances.GetForUser(userID, headerValue); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", errors.New("unknown instance id")
			}
			return "", err
		}
		return headerValue, nil
	}

	activeID, err := r.instances.GetActiveIDForUser(userID)
	if err != nil {
		return "", err
	}
	if activeID == "" {
		return "", nil
	}
	if _, err := r.instances.GetForUser(userID, activeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = r.instances.ClearActiveForUser(userID)
			return "", nil
		}
		return "", err
	}
	return activeID, nil
}

func (r *Resolver) ReloadActive() error {
	inst, err := r.instances.GetActive()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.mu.Lock()
			r.active = subsonic.NewClient("", "", "")
			r.mu.Unlock()
			slog.Debug("no active subsonic instance configured")
			return nil
		}
		return err
	}

	client := subsonic.NewClient(inst.ServerURL, inst.Username, inst.Password)
	r.mu.Lock()
	r.active = client
	r.cache[inst.ID] = client
	r.mu.Unlock()
	slog.Info("active subsonic instance loaded",
		"instance_id", inst.ID,
		"url", httputil.RedactURLUserinfo(inst.ServerURL),
		"name", inst.Name,
	)
	return nil
}

func (r *Resolver) CachedClient(instanceID, serverURL, username, password string) *subsonic.Client {
	r.mu.RLock()
	if client, ok := r.cache[instanceID]; ok {
		r.mu.RUnlock()
		return client
	}
	r.mu.RUnlock()

	client := subsonic.NewClient(serverURL, username, password)
	r.mu.Lock()
	r.cache[instanceID] = client
	r.mu.Unlock()
	return client
}

func (r *Resolver) Invalidate(instanceID string) {
	r.mu.Lock()
	delete(r.cache, instanceID)
	r.mu.Unlock()
}

func (r *Resolver) Active() *subsonic.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.active
}

func (r *Resolver) ClientCacheSize() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.cache)
}

func (r *Resolver) ForContext(ctx context.Context) *subsonic.Client {
	instanceID := InstanceIDFromContext(ctx)
	userID := UserIDFromContext(ctx)
	if instanceID == "" {
		if userID != "" {
			return subsonic.NewClient("", "", "")
		}
		return r.Active()
	}

	activeID, _ := r.instances.GetActiveID()
	r.mu.RLock()
	if userID == "" && activeID == instanceID && r.active != nil && r.active.Enabled() {
		client := r.active
		r.mu.RUnlock()
		return client
	}
	r.mu.RUnlock()

	inst, err := r.instances.GetForUser(userID, instanceID)
	if err != nil {
		return subsonic.NewClient("", "", "")
	}
	return r.CachedClient(instanceID, inst.ServerURL, inst.Username, inst.Password)
}
