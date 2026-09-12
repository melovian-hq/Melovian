// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// LookupQuery describes a catalog lookup. Query is a freeform search string
// while Artist, Title, and Album narrow providers that support structured
// fields, such as TheAudioDB and MusicBrainz.
type LookupQuery struct {
	Query  string
	Artist string
	Title  string
	Album  string
}

func (q LookupQuery) term() string {
	if strings.TrimSpace(q.Query) != "" {
		return strings.TrimSpace(q.Query)
	}
	return strings.TrimSpace(strings.Join([]string{q.Artist, q.Title, q.Album}, " "))
}

// LookupSourceIDs lists the metadata catalog sources accepted by Lookup.
var LookupSourceIDs = []string{"itunes", "musicbrainz", "deezer", "theaudiodb", "all"}

// maxLookupCap bounds the result capacity so a caller-supplied limit cannot
// trigger a huge allocation.
const maxLookupCap = 500

// Lookup queries one or more metadata catalogs. An empty source falls back to
// iTunes so existing callers keep working. "all" fans out to every provider
// and merges the results.
func Lookup(ctx context.Context, source string, q LookupQuery, limit int) ([]LookupMatch, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" || source == "lookup" {
		source = "itunes"
	}
	term := q.term()
	if term == "" && source != "theaudiodb" {
		return nil, nil
	}
	switch source {
	case "itunes":
		return LookupITunes(ctx, term, limit)
	case "musicbrainz":
		return LookupMusicBrainz(ctx, q, limit)
	case "deezer":
		return LookupDeezer(ctx, term, limit)
	case "theaudiodb":
		return LookupTheAudioDB(ctx, q, limit)
	case "all":
		return lookupAll(ctx, q, limit)
	default:
		return nil, fmt.Errorf("unsupported metadata source %q", source)
	}
}

func lookupAll(ctx context.Context, q LookupQuery, limit int) ([]LookupMatch, error) {
	type result struct {
		matches []LookupMatch
		err     error
	}
	fns := []func(context.Context) ([]LookupMatch, error){
		func(ctx context.Context) ([]LookupMatch, error) {
			return LookupITunes(ctx, q.term(), limit)
		},
		func(ctx context.Context) ([]LookupMatch, error) {
			return LookupMusicBrainz(ctx, q, limit)
		},
		func(ctx context.Context) ([]LookupMatch, error) {
			return LookupDeezer(ctx, q.term(), limit)
		},
		func(ctx context.Context) ([]LookupMatch, error) {
			return LookupTheAudioDB(ctx, q, limit)
		},
	}
	results := make([]result, len(fns))
	var wg sync.WaitGroup
	for i, fn := range fns {
		wg.Add(1)
		go func(i int, fn func(context.Context) ([]LookupMatch, error)) {
			defer wg.Done()
			matches, err := fn(ctx)
			results[i] = result{matches: matches, err: err}
		}(i, fn)
	}
	wg.Wait()

	capHint := limit
	if capHint < 0 || capHint > maxLookupCap {
		capHint = maxLookupCap
	}
	out := make([]LookupMatch, 0, capHint)
	seen := map[string]bool{}
	var firstErr error
	for _, res := range results {
		if res.err != nil {
			if firstErr == nil {
				firstErr = res.err
			}
			continue
		}
		for _, match := range res.matches {
			key := strings.ToLower(match.Source + "|" + match.Title + "|" + match.Artist + "|" + match.Album)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, match)
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
	}
	if len(out) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}
