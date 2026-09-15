// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"context"
	"net/http"

	"melovian/internal/store"
	"melovian/internal/subsonic"
)

// subsonicSource is the bundled Subsonic-compatible source. It is a
// thin adapter: the remote server already speaks the protocol, so
// ServeREST just proxies with injected credentials.
type subsonicSource struct{}

// Subsonic returns the bundled Subsonic source.
func Subsonic() Source { return subsonicSource{} }

func (subsonicSource) ID() string          { return store.DefaultSourceID }
func (subsonicSource) DisplayName() string { return "Subsonic" }
func (subsonicSource) Caps() Capabilities  { return Capabilities{} }

func (subsonicSource) Ping(ctx context.Context, inst store.SourceInstance) (string, string, error) {
	client := subsonic.NewClient(inst.ServerURL, inst.Username, inst.Password)
	return client.Ping()
}

func (subsonicSource) ServeREST(w http.ResponseWriter, r *http.Request, inst store.SourceInstance, deps Deps) {
	client := deps.ClientFor(inst)
	proxy := subsonic.NewProxy(func(context.Context) *subsonic.Client {
		return client
	}, deps.Cache, deps.CacheEnabled, deps.CacheScope)
	proxy.ServeHTTP(w, r)
}
