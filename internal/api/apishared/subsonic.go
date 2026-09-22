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
	inst, err := r.ResolveInstance(req)
	if err != nil {
		return "", err
	}
	return inst.ID, nil
}

// ResolveInstance resolves the request's target instance once, returning the
// full row. A zero-value instance means no instance is configured.
func (r *Resolver) ResolveInstance(req *http.Request) (store.SourceInstance, error) {
	headerValue := InstanceIDFromRequest(req)
	userID := UserIDFromContext(req.Context())

	if headerValue != "" {
		inst, err := r.instances.GetForUser(userID, headerValue)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return store.SourceInstance{}, errors.New("unknown instance id")
			}
			return store.SourceInstance{}, err
		}
		return inst, nil
	}

	activeID, err := r.instances.GetActiveIDForUser(userID)
	if err != nil {
		return store.SourceInstance{}, err
	}
	if activeID == "" {
		return store.SourceInstance{}, nil
	}
	inst, err := r.instances.GetForUser(userID, activeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = r.instances.ClearActiveForUser(userID)
			return store.SourceInstance{}, nil
		}
		return store.SourceInstance{}, err
	}
	return inst, nil
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
	userID := UserIDFromContext(ctx)
	if inst, ok := ResolvedInstanceFromContext(ctx); ok {
		if inst.ID == "" {
			if userID != "" {
				return subsonic.NewClient("", "", "")
			}
			return r.Active()
		}
		return r.CachedClient(inst.ID, inst.ServerURL, inst.Username, inst.Password)
	}

	instanceID := InstanceIDFromContext(ctx)
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
