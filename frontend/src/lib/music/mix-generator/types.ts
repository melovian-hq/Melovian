// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MixSettings } from "../mix-settings";
import type { ArtistCluster } from "../mix-select";
import type { TasteProfile } from "../taste-score";
import type {
  ListenEntry,
  ListenStats,
  SubsonicAlbum,
  SubsonicGenre,
  SubsonicSong,
} from "$lib/subsonic/types";

export {
  createSeededRandom,
  dedupeTracks,
  interleaveByArtist,
  orderForFlow,
  selectMixTracks,
  shuffleWithSeed,
  clusterArtistFeatures,
  emptyMixSeed,
} from "../mix-select";
export type {
  FlowOptions,
  MixSeedProfile,
  MixSelectPolicy,
  ArtistCluster,
  ArtistFeature,
} from "../mix-select";

export interface GeneratedMix {
  id: string;
  title: string;
  subtitle: string;
  tracks: SubsonicSong[];
  coverArtId?: string;
  gradient: string;
}

export interface MixBuildContext {
  stats: ListenStats | null;
  history: readonly ListenEntry[];
  frequentAlbums: readonly SubsonicAlbum[];
  recentlyPlayedIds: ReadonlySet<string>;
  playCountByTrack: ReadonlyMap<string, number>;
  listenedMsByTrack: ReadonlyMap<string, number>;
  skippedTrackIds: ReadonlySet<string>;
  daySeed: string;
  settings: MixSettings;
  entryToSong: (entry: ListenEntry) => SubsonicSong;
  recentMixTrackIds: ReadonlySet<string>;
  tasteProfile: TasteProfile;
}

export interface MixBuildState {
  usedTrackIds: Set<string>;
  usedCoverArtIds: Set<string>;
  languageWeights: Map<string, number>;
  starredTrackIds: Set<string>;
  artistClusters: ArtistCluster[] | null;
}

export interface MixFetchers {
  searchArtistSongs(artist: string, limit: number): Promise<SubsonicSong[]>;
  searchArtistAlbums(artist: string, limit: number): Promise<SubsonicAlbum[]>;
  getAlbumSongs(albumId: string): Promise<SubsonicSong[]>;
  getSimilarSongs(trackId: string, count: number): Promise<SubsonicSong[]>;
  getRandomSongs(count: number): Promise<SubsonicSong[]>;
  getRandomAlbums(count: number): Promise<SubsonicAlbum[]>;
  getNewestAlbums(count: number): Promise<SubsonicAlbum[]>;
  getGenreSongs(genre: string, count: number): Promise<SubsonicSong[]>;
  getGenres(): Promise<SubsonicGenre[]>;
  getStarredSongs(): Promise<SubsonicSong[]>;
}
