// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import (
	"context"
	"fmt"
	"net/http"
)

type SubsonicFetcher func(ctx context.Context, in FetchInput) (*Document, error)

type SubsonicProvider struct {
	FetchFn SubsonicFetcher
}

func (p SubsonicProvider) ID() string { return "subsonic" }

func (p SubsonicProvider) Name() string { return "Navidrome / Subsonic" }

func (p SubsonicProvider) Fetch(ctx context.Context, _ *http.Client, in FetchInput) (*Document, error) {
	if p.FetchFn == nil {
		return nil, fmt.Errorf("subsonic provider unavailable")
	}
	return p.FetchFn(ctx, in)
}

func SubsonicDocumentFromRaw(doc *Document) *Document {
	if doc == nil {
		return nil
	}
	doc.Source = "subsonic"
	return doc
}
