// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import "strings"

import "fmt"

func NormalizeDocument(doc *Document) (*Document, error) {
	if doc == nil {
		return nil, fmt.Errorf("lyrics document is nil")
	}
	if len(doc.Lines) == 0 && doc.RawValue != "" {
		parsed := ParseText(doc.RawValue, FetchInput{
			Artist: doc.Artist,
			Title:  doc.Title,
		})
		doc.Lines = parsed.Lines
		doc.Synced = parsed.Synced
		doc.OffsetMs = parsed.OffsetMs
	}

	hasTimedLine := false
	for _, line := range doc.Lines {
		if line.StartMs != nil {
			hasTimedLine = true
			break
		}
	}
	// Plain Subsonic responses often embed LRC in RawValue with no StartMs.
	if !doc.Synced && !hasTimedLine && strings.TrimSpace(doc.RawValue) != "" {
		parsed := ParseText(doc.RawValue, FetchInput{
			Artist: doc.Artist,
			Title:  doc.Title,
		})
		if parsed.Synced {
			doc.Lines = parsed.Lines
			doc.Synced = true
			doc.OffsetMs = parsed.OffsetMs
			hasTimedLine = true
		}
	}

	lines := make([]Line, 0, len(doc.Lines))
	for _, line := range doc.Lines {
		text := trimSpace(line.Text)
		if text == "" {
			continue
		}
		entry := Line{Text: text}
		if line.StartMs != nil {
			start := *line.StartMs
			entry.StartMs = &start
			hasTimedLine = true
		}
		lines = append(lines, entry)
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("lyrics document has no lines")
	}
	doc.Lines = lines
	if hasTimedLine {
		doc.Synced = true
	}
	if doc.RawValue == "" {
		parts := make([]string, 0, len(lines))
		for _, line := range lines {
			parts = append(parts, line.Text)
		}
		doc.RawValue = joinLineTexts(parts)
	}
	return doc, nil
}

func joinLineTexts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString(parts[0])
	for i := 1; i < len(parts); i++ {
		out.WriteString("\n" + parts[i])
	}
	return out.String()
}

func trimSpace(value string) string {
	start := 0
	for start < len(value) && (value[start] == ' ' || value[start] == '\t' || value[start] == '\n' || value[start] == '\r') {
		start++
	}
	end := len(value)
	for end > start && (value[end-1] == ' ' || value[end-1] == '\t' || value[end-1] == '\n' || value[end-1] == '\r') {
		end--
	}
	return value[start:end]
}
