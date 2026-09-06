// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

type FetchInput struct {
	TrackID     string
	Artist      string
	Title       string
	Album       string
	DurationSec int
}

type Line struct {
	Text    string `json:"text"`
	StartMs *int   `json:"startMs,omitempty"`
}

type Document struct {
	Source   string `json:"source"`
	Artist   string `json:"artist,omitempty"`
	Title    string `json:"title,omitempty"`
	Synced   bool   `json:"synced"`
	OffsetMs int    `json:"offsetMs"`
	RawValue string `json:"rawValue"`
	Lines    []Line `json:"lines"`
}

type ProviderInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Builtin bool   `json:"builtin"`
	Custom  bool   `json:"custom,omitempty"`
}
