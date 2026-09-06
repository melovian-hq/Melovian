// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonic

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type LyricsLine struct {
	Start *int   `json:"start"`
	Value string `json:"value"`
}

type LyricsDocument struct {
	Source   string
	Artist   string
	Title    string
	Synced   bool
	OffsetMs int
	RawValue string
	Lines    []LyricsLine
}

func (s *Client) GetLyricsBySongID(id string) (*LyricsDocument, error) {
	body, err := s.getJSON("getLyricsBySongId.view", url.Values{"id": {id}})
	if err != nil {
		return nil, err
	}
	return parseLyricsResponse(body)
}

func (s *Client) GetLyrics(artist, title string) (*LyricsDocument, error) {
	query := url.Values{}
	if artist != "" {
		query.Set("artist", artist)
	}
	if title != "" {
		query.Set("title", title)
	}
	body, err := s.getJSON("getLyrics.view", query)
	if err != nil {
		return nil, err
	}
	return parseLyricsResponse(body)
}

func parseLyricsResponse(body []byte) (*LyricsDocument, error) {
	var payload struct {
		SubsonicResponse struct {
			Status string `json:"status"`
			Lyrics *struct {
				Artist string `json:"artist"`
				Title  string `json:"title"`
				Value  string `json:"value"`
			} `json:"lyrics"`
			LyricsList *struct {
				StructuredLyrics json.RawMessage `json:"structuredLyrics"`
			} `json:"lyricsList"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.SubsonicResponse.Status != "ok" {
		msg := "subsonic lyrics request failed"
		if payload.SubsonicResponse.Error != nil {
			msg = payload.SubsonicResponse.Error.Message
		}
		return nil, fmt.Errorf("%s", msg)
	}

	if payload.SubsonicResponse.LyricsList != nil {
		doc, err := parseStructuredLyrics(payload.SubsonicResponse.LyricsList.StructuredLyrics)
		if err == nil && doc != nil {
			return doc, nil
		}
	}

	if payload.SubsonicResponse.Lyrics != nil && strings.TrimSpace(payload.SubsonicResponse.Lyrics.Value) != "" {
		raw := payload.SubsonicResponse.Lyrics.Value
		doc := parsePlainLyrics(raw)
		doc.Source = "subsonic"
		doc.Artist = payload.SubsonicResponse.Lyrics.Artist
		doc.Title = payload.SubsonicResponse.Lyrics.Title
		return doc, nil
	}

	return nil, fmt.Errorf("subsonic: lyrics not found")
}

func parseStructuredLyrics(raw json.RawMessage) (*LyricsDocument, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("structured lyrics missing")
	}
	var entries []map[string]any
	if err := json.Unmarshal(raw, &entries); err != nil {
		var single map[string]any
		if errSingle := json.Unmarshal(raw, &single); errSingle != nil {
			return nil, err
		}
		entries = []map[string]any{single}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("structured lyrics empty")
	}

	entry := pickStructuredEntry(entries)
	linesRaw, ok := entry["line"]
	if !ok {
		return nil, fmt.Errorf("structured lyrics lines missing")
	}

	lineItems := unwrapJSONObjects(linesRaw)
	doc := &LyricsDocument{Source: "subsonic"}
	if artist, ok := entry["displayArtist"].(string); ok {
		doc.Artist = artist
	}
	if title, ok := entry["displayTitle"].(string); ok {
		doc.Title = title
	}
	if offset, ok := entry["offset"].(float64); ok {
		doc.OffsetMs = int(offset)
	}
	if synced, ok := entry["synced"].(bool); ok {
		doc.Synced = synced
	}

	for _, item := range lineItems {
		text, _ := item["value"].(string)
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		line := LyricsLine{Value: text}
		if start, ok := item["start"].(float64); ok {
			startMs := int(start)
			line.Start = &startMs
			doc.Synced = true
		}
		doc.Lines = append(doc.Lines, line)
	}
	if len(doc.Lines) == 0 {
		return nil, fmt.Errorf("structured lyrics empty")
	}
	doc.RawValue = joinLyricLines(doc.Lines)
	return doc, nil
}

func pickStructuredEntry(entries []map[string]any) map[string]any {
	var mainEntry, syncedMain, syncedAny map[string]any
	for _, entry := range entries {
		kind, _ := entry["kind"].(string)
		isMain := kind == "" || kind == "main"
		synced := entryLooksSynced(entry)
		if isMain && mainEntry == nil {
			mainEntry = entry
		}
		if synced {
			if isMain && syncedMain == nil {
				syncedMain = entry
			}
			if syncedAny == nil {
				syncedAny = entry
			}
		}
	}
	if syncedMain != nil {
		return syncedMain
	}
	if syncedAny != nil {
		return syncedAny
	}
	if mainEntry != nil {
		return mainEntry
	}
	return entries[0]
}

func entryLooksSynced(entry map[string]any) bool {
	if synced, ok := entry["synced"].(bool); ok && synced {
		return true
	}
	for _, item := range unwrapJSONObjects(entry["line"]) {
		if _, ok := item["start"].(float64); ok {
			return true
		}
	}
	return false
}

func unwrapJSONObjects(value any) []map[string]any {
	switch typed := value.(type) {
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if obj, ok := item.(map[string]any); ok {
				out = append(out, obj)
			}
		}
		return out
	case map[string]any:
		return []map[string]any{typed}
	default:
		return nil
	}
}

func parsePlainLyrics(raw string) *LyricsDocument {
	doc := &LyricsDocument{RawValue: raw}
	for line := range strings.SplitSeq(raw, "\n") {
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}
		doc.Lines = append(doc.Lines, LyricsLine{Value: text})
	}
	return doc
}

func joinLyricLines(lines []LyricsLine) string {
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		parts = append(parts, line.Value)
	}
	return strings.Join(parts, "\n")
}

func ToLyricsDocument(doc *LyricsDocument) map[string]any {
	if doc == nil {
		return nil
	}
	lines := make([]map[string]any, 0, len(doc.Lines))
	for _, line := range doc.Lines {
		entry := map[string]any{"text": line.Value}
		if line.Start != nil {
			entry["startMs"] = *line.Start
		}
		lines = append(lines, entry)
	}
	return map[string]any{
		"source":   doc.Source,
		"artist":   doc.Artist,
		"title":    doc.Title,
		"synced":   doc.Synced,
		"offsetMs": doc.OffsetMs,
		"rawValue": doc.RawValue,
		"lines":    lines,
	}
}
