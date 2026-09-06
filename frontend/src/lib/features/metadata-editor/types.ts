// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export type MetadataIssueFilter =
  "" | "any" | "unknown-artist" | "unknown-album" | "missing-title";

export interface MetadataSummary {
  unknownArtist: number;
  unknownAlbum: number;
  missingTitle: number;
  any: number;
  totalTracks: number;
}

export interface MetadataTrack {
  id: string;
  title: string;
  artist: string;
  album: string;
  albumArtist: string;
  trackNum: number;
  discNum: number;
  year: number;
  genre: string;
  durationMs: number;
  format: string;
  relPath: string;
  issues: string[];
  coverArt: string;
}

export interface MetadataTrackUpdate {
  title?: string;
  artist?: string;
  album?: string;
  albumArtist?: string;
  trackNum?: number;
  discNum?: number;
  year?: number;
  genre?: string;
}

export interface MetadataLookupMatch {
  id: string;
  source: string;
  title: string;
  artist: string;
  album: string;
  albumArtist: string;
  trackNum: number;
  year: number;
  genre: string;
  artworkUrl?: string;
}

export interface MetadataFilenameSuggestion {
  source: string;
  title: string;
  artist: string;
  album: string;
  albumArtist: string;
  trackNum: number;
  discNum: number;
}

export interface MetadataBatchResult {
  trackId: string;
  ok: boolean;
  error?: string;
  track?: MetadataTrack;
}

export interface MetadataSearchResult {
  tracks: MetadataTrack[];
  total: number;
}
