// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export interface SubsonicConfig {
  serverUrl: string;
  clientName: string;
  version: string;
}

export interface SubsonicArtist {
  id: string;
  name: string;
  albumCount?: number;
  coverArt?: string;
  artistImageUrl?: string;
}

export interface SimilarArtist {
  id?: string;
  name: string;
  coverArt?: string;
}

export interface SubsonicArtistInfo {
  biography?: string;
  musicBrainzId?: string;
  lastFmUrl?: string;
  smallImageUrl?: string;
  mediumImageUrl?: string;
  largeImageUrl?: string;
  similarArtists: SimilarArtist[];
}

export interface SubsonicLyrics {
  artist?: string;
  title?: string;
  value: string;
}

export type {
  LyricLine,
  ParsedLyrics,
  LyricsSearchHit,
} from "$lib/music/lyrics";

export interface SubsonicAlbum {
  id: string;
  name: string;
  artist?: string;
  artistId?: string;
  year?: number;
  songCount?: number;
  duration?: number;
  coverArt?: string;
  genre?: string;
}

export interface SubsonicTrackArtist {
  id: string;
  name: string;
}

export interface SubsonicSong {
  id: string;
  title: string;
  album?: string;
  albumId?: string;
  artist?: string;
  artistId?: string;
  artists?: SubsonicTrackArtist[];
  track?: number;
  duration?: number;
  coverArt?: string;
  year?: number;
  genre?: string;
  bitRate?: number;
  contentType?: string;
  suffix?: string;
  transcoded?: boolean;
  /** Channel count when the server provides OpenSubsonic or Navidrome extras. */
  channels?: number;
  channelCount?: number;
  samplingRate?: number;
  bitDepth?: number;
  path?: string;
  starred?: string;
  playCount?: number;
}

export interface SubsonicGenre {
  name: string;
  songCount?: number;
  albumCount?: number;
}

export interface SubsonicSearchResult {
  artists: SubsonicArtist[];
  albums: SubsonicAlbum[];
  songs: SubsonicSong[];
}

export interface ServerPlaylist {
  id: string;
  name: string;
  songCount?: number;
  duration?: number;
  coverArt?: string;
  owner?: string;
  public?: boolean;
  created?: string;
  changed?: string;
}

export interface StarredContent {
  songs: SubsonicSong[];
  albums: SubsonicAlbum[];
  artists: SubsonicArtist[];
}

export interface MusicStatus {
  enabled: boolean;
  connected: boolean;
  serverName?: string;
  version?: string;
  error?: string;
  source?: "local" | "subsonic";
}

export interface LibraryStats {
  songCount: number;
  albumCount: number;
  artistCount: number;
  folderCount: number;
  scanning: boolean;
  lastScan?: string;
}

export interface ListenEntry {
  trackId: string;
  trackTitle: string;
  artistName: string;
  albumId: string;
  albumTitle: string;
  positionMs: number;
  durationMs: number;
  played: boolean;
  playCount: number;
  listenedMs: number;
  lastPlayedAt: string;
  coverArtId: string;
}

export interface ListenEvent {
  id: number;
  trackId: string;
  trackTitle: string;
  artistName: string;
  albumId: string;
  albumTitle: string;
  durationMs: number;
  coverArtId: string;
  playedAt: string;
}

export interface ListenStats {
  totalPlays: number;
  uniqueTracks: number;
  totalListeningMs: number;
  topArtists: { key: string; label: string; count: number }[];
  topTracks: { key: string; label: string; count: number }[];
  topAlbums: { key: string; label: string; count: number }[];
}

export interface MusicPlaylist {
  id: string;
  name: string;
  kind?: "static" | "smart" | string;
  rulesJson?: string;
  createdAt: string;
  updatedAt: string;
  trackCount: number;
  durationMs?: number;
  coverArtIds?: string[];
  tracks?: PlaylistTrack[];
}

export interface PlaylistTrack {
  trackId: string;
  trackTitle: string;
  artistName: string;
  albumId: string;
  albumTitle: string;
  durationMs: number;
  coverArtId: string;
  position?: number;
}

export interface FavoriteTrack {
  trackId: string;
  trackTitle: string;
  artistName: string;
  albumId: string;
  albumTitle: string;
  durationMs: number;
  coverArtId: string;
  favoritedAt: string;
}

export interface InternetRadioStation {
  id: string;
  name: string;
  streamUrl: string;
  homePageUrl?: string;
  coverArt?: string;
}

export interface QueueTrack extends SubsonicSong {
  positionMs?: number;
  streamUrl?: string;
  isInternetRadio?: boolean;
}
