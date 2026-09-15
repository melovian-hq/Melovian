// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"context"

	"melovian/internal/cache"
	"melovian/internal/store"
	"melovian/internal/subsonic"
)

// Deps carries shared infrastructure handed to a source while it serves
// one request. ClientFor resolves (and caches) the upstream client for
// the instance; sources that speak Subsonic use it for all traffic.
type Deps struct {
	Cache        *cache.ResponseCache
	CacheEnabled bool
	CacheScope   func(context.Context) string
	ClientFor    func(inst store.SourceInstance) *subsonic.Client
}
