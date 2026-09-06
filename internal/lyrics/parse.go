// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	lrcLinePattern   = regexp.MustCompile(`^\[(\d+):(\d{2})(?:[.:](\d{1,3}))?\]\s*(.*)$`)
	lrcOffsetPattern = regexp.MustCompile(`(?i)^\[offset:\s*([+-]?\d+)\s*\]$`)
	lrcMetaPattern   = regexp.MustCompile(`(?i)^\[(?:ti|ar|al|by|au|length|re|ve|la|tool|key|bpm|sign|language|lang):`)
)

func ParseText(value string, meta FetchInput) *Document {
	rawValue := strings.ReplaceAll(value, "\r\n", "\n")
	lines := make([]Line, 0)
	offsetMs := 0
	sawTimestamp := false

	for rawLine := range strings.SplitSeq(rawValue, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if matches := lrcOffsetPattern.FindStringSubmatch(line); len(matches) == 2 {
			if parsed, err := strconv.Atoi(matches[1]); err == nil {
				offsetMs = parsed
			}
			continue
		}
		if lrcMetaPattern.MatchString(line) {
			continue
		}
		if matches := lrcLinePattern.FindStringSubmatch(line); len(matches) >= 5 {
			start := parseTimestamp(matches[1], matches[2], matches[3])
			text := strings.TrimSpace(matches[4])
			if text == "" {
				continue
			}
			sawTimestamp = true
			lines = append(lines, Line{Text: text, StartMs: &start})
			continue
		}
		lines = append(lines, Line{Text: line})
	}

	synced := sawTimestamp
	for _, entry := range lines {
		if entry.StartMs != nil {
			synced = true
			break
		}
	}

	return &Document{
		Artist:   meta.Artist,
		Title:    meta.Title,
		Synced:   synced,
		OffsetMs: offsetMs,
		RawValue: rawValue,
		Lines:    lines,
	}
}

func parseTimestamp(minutes, seconds, fraction string) int {
	mins, _ := strconv.Atoi(minutes)
	secs, _ := strconv.Atoi(seconds)
	ms := mins*60_000 + secs*1000
	if fraction != "" {
		digits := fraction
		for len(digits) < 3 {
			digits += "0"
		}
		if len(digits) > 3 {
			digits = digits[:3]
		}
		extra, _ := strconv.Atoi(digits)
		ms += extra
	}
	return ms
}
