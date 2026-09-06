// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package smartplaylist

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"melovian/internal/store"
)

type draftGroup struct {
	Logic  string       `json:"logic"`
	Rules  []draftRule  `json:"rules"`
	Groups []draftGroup `json:"groups"`
}

type draftRule struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

type draftRoot struct {
	Root draftGroup `json:"root"`
}

type EvalMeta struct {
	PlayCount    int
	Rating       int
	Loved        bool
	LastPlayedAt time.Time
	DateAdded    time.Time
	HasLastPlay  bool
	HasDateAdded bool
}

type TrackContext struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	Genre       string
	Language    string
	Comment     string
	FilePath    string
	Codec       string
	Year        int
	DurationSec int
	Bitrate     int
	Meta        EvalMeta
}

func TrackFromLocal(track store.LocalTrack) TrackContext {
	return TrackContext{
		Title:       track.Title,
		Artist:      track.Artist,
		Album:       track.Album,
		AlbumArtist: track.AlbumArtist,
		Genre:       track.Genre,
		FilePath:    track.RelPath,
		Codec:       track.Format,
		Year:        track.Year,
		DurationSec: track.DurationMs / 1000,
		Meta: EvalMeta{
			DateAdded:    track.UpdatedAt,
			HasDateAdded: !track.UpdatedAt.IsZero(),
		},
	}
}

func EnrichMeta(track *TrackContext, progress store.ListenProgress, loved bool) {
	track.Meta.PlayCount = progress.PlayCount
	track.Meta.LastPlayedAt = progress.LastPlayedAt
	track.Meta.HasLastPlay = !progress.LastPlayedAt.IsZero()
	if loved {
		track.Meta.Loved = true
	}
}

func MatchDraftJSON(rulesJSON string, track TrackContext) bool {
	rulesJSON = strings.TrimSpace(rulesJSON)
	if rulesJSON == "" {
		return true
	}
	var draft draftRoot
	if err := json.Unmarshal([]byte(rulesJSON), &draft); err != nil {
		return false
	}
	return matchGroup(draft.Root, track)
}

func matchGroup(group draftGroup, track TrackContext) bool {
	logic := strings.ToLower(strings.TrimSpace(group.Logic))
	if logic == "" {
		logic = "all"
	}
	if logic == "any" {
		for _, rule := range group.Rules {
			if matchRule(rule, track) {
				return true
			}
		}
		for _, child := range group.Groups {
			if matchGroup(child, track) {
				return true
			}
		}
		return false
	}
	for _, rule := range group.Rules {
		if !matchRule(rule, track) {
			return false
		}
	}
	for _, child := range group.Groups {
		if !matchGroup(child, track) {
			return false
		}
	}
	return true
}

func matchRule(rule draftRule, track TrackContext) bool {
	op := strings.TrimSpace(rule.Operator)
	field := strings.ToLower(strings.TrimSpace(rule.Field))

	if op == "isMissing" || op == "isPresent" {
		present := fieldValue(track, field) != ""
		wantPresent := asBool(rule.Value)
		if op == "isPresent" {
			return present == wantPresent
		}
		return present != wantPresent
	}

	if isBoolField(field) {
		actual := boolFieldValue(track, field)
		expected := asBool(rule.Value)
		if op == "is" {
			return actual == expected
		}
		if op == "isNot" {
			return actual != expected
		}
	}

	if op == "inTheLast" || op == "notInTheLast" {
		days := asNumberValue(rule.Value)
		if days <= 0 {
			return false
		}
		elapsed, ok := daysSinceDateField(track, field)
		if !ok {
			return false
		}
		matched := elapsed <= int(days)
		if op == "inTheLast" {
			return matched
		}
		return !matched
	}

	if op == "inTheRange" {
		return matchRange(track, field, rule.Value)
	}

	if op == "before" || op == "after" {
		left, ok := dateFieldValue(track, field)
		if !ok {
			return false
		}
		right := strings.TrimSpace(asString(rule.Value))
		if op == "before" {
			return left < right
		}
		return left > right
	}

	if isNumberField(field) {
		left := numberFieldValue(track, field)
		right := asNumberValue(rule.Value)
		switch op {
		case "is":
			return left == right
		case "isNot":
			return left != right
		case "gt":
			return left > right
		case "lt":
			return left < right
		default:
			return false
		}
	}

	value := fieldValue(track, field)
	needle := asString(rule.Value)
	switch op {
	case "is":
		return strings.EqualFold(value, needle)
	case "isNot":
		return !strings.EqualFold(value, needle)
	case "contains":
		if strings.HasPrefix(needle, "/") && strings.HasSuffix(needle, "/") && len(needle) > 2 {
			return matchRegex(value, needle[1:len(needle)-1])
		}
		return strings.Contains(strings.ToLower(value), strings.ToLower(needle))
	case "notContains":
		return !strings.Contains(strings.ToLower(value), strings.ToLower(needle))
	case "startsWith":
		return strings.HasPrefix(strings.ToLower(value), strings.ToLower(needle))
	case "endsWith":
		return strings.HasSuffix(strings.ToLower(value), strings.ToLower(needle))
	default:
		return false
	}
}

func matchRange(track TrackContext, field string, raw any) bool {
	start, end, ok := rangeValues(raw)
	if !ok {
		return false
	}
	if isNumberField(field) {
		value := numberFieldValue(track, field)
		return value >= start && value <= end
	}
	text, ok := dateFieldValue(track, field)
	if !ok {
		return false
	}
	startText := strconv.FormatFloat(start, 'f', -1, 64)
	endText := strconv.FormatFloat(end, 'f', -1, 64)
	return text >= startText && text <= endText
}

func rangeValues(raw any) (float64, float64, bool) {
	items, ok := raw.([]any)
	if !ok || len(items) != 2 {
		return 0, 0, false
	}
	start := asNumberValue(items[0])
	end := asNumberValue(items[1])
	return start, end, true
}

func fieldValue(track TrackContext, field string) string {
	switch field {
	case "title":
		return track.Title
	case "artist":
		return track.Artist
	case "album":
		return track.Album
	case "albumartist":
		return track.AlbumArtist
	case "genre":
		return track.Genre
	case "language":
		return track.Language
	case "comment":
		return track.Comment
	case "filepath":
		return track.FilePath
	case "codec", "filetype", "suffix":
		return track.Codec
	case "year":
		if track.Year == 0 {
			return ""
		}
		return strconv.Itoa(track.Year)
	default:
		return ""
	}
}

func numberFieldValue(track TrackContext, field string) float64 {
	switch field {
	case "year":
		return float64(track.Year)
	case "duration":
		return float64(track.DurationSec)
	case "bitrate":
		return float64(track.Bitrate)
	case "rating":
		return float64(track.Meta.Rating)
	case "playcount":
		return float64(track.Meta.PlayCount)
	default:
		return asNumberValue(fieldValue(track, field))
	}
}

func boolFieldValue(track TrackContext, field string) bool {
	switch field {
	case "loved":
		return track.Meta.Loved
	case "missing":
		return false
	case "compilation":
		return strings.TrimSpace(track.AlbumArtist) != "" &&
			!strings.EqualFold(track.AlbumArtist, track.Artist)
	default:
		return false
	}
}

func dateFieldValue(track TrackContext, field string) (string, bool) {
	switch field {
	case "dateadded":
		if !track.Meta.HasDateAdded {
			return "", false
		}
		return track.Meta.DateAdded.UTC().Format("2006-01-02"), true
	case "lastplayed":
		if !track.Meta.HasLastPlay {
			return "", false
		}
		return track.Meta.LastPlayedAt.UTC().Format("2006-01-02T15:04:05"), true
	default:
		return "", false
	}
}

func daysSinceDateField(track TrackContext, field string) (int, bool) {
	switch field {
	case "dateadded":
		if !track.Meta.HasDateAdded {
			return 0, false
		}
		return daysSince(track.Meta.DateAdded), true
	case "lastplayed":
		if !track.Meta.HasLastPlay {
			return 0, false
		}
		return daysSince(track.Meta.LastPlayedAt), true
	default:
		return 0, false
	}
}

func daysSince(value time.Time) int {
	if value.IsZero() {
		return 0
	}
	elapsed := time.Since(value)
	if elapsed < 0 {
		return 0
	}
	return int(elapsed.Hours() / 24)
}

func isNumberField(field string) bool {
	switch field {
	case "year", "duration", "bitrate", "rating", "playcount", "bpm", "bitdepth", "samplerate":
		return true
	default:
		return false
	}
}

func isBoolField(field string) bool {
	switch field {
	case "loved", "missing", "compilation":
		return true
	default:
		return false
	}
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.Itoa(int(typed))
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		if value == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func asNumberValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case string:
		out, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0
		}
		return out
	default:
		return 0
	}
}

func asBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func matchRegex(value, pattern string) bool {
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return strings.Contains(strings.ToLower(value), strings.ToLower(pattern))
	}
	return re.MatchString(value)
}
