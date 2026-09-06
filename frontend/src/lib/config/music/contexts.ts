// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { sources } from "$lib/features/sources/store.svelte";
import type { MusicQueueContext } from "./queue-ops";
import type { MusicFavoritesContext } from "./favorites-ops";
import type { MusicPlaylistContext } from "./playlist-ops";
import type { MusicLibraryBrowseContext } from "./library-browse-ops";
import type { MusicPlaybackTransportContext } from "./playback-transport-ops";
import type { MusicCacheContext } from "./cache-ops";
import type { MusicSettingsContext } from "./settings-ops";
import type { MusicLyricsContext } from "./lyrics-ops";
import type { MusicEqContext } from "./eq-ops";
import type { MusicPlayerChromeContext } from "./player-chrome-ops";
import type { MusicMixContext } from "./mix-ops";
import type { MusicRadioContext } from "./radio-ops";
import type { MusicPlayLaunchContext } from "./play-launch-ops";
import type { MusicConnectContext } from "./connect-ops";
import type { MusicEngineContext } from "./engine-ops";
import type { MusicPlaybackCoreContext } from "./playback-core-ops";
import type { MusicLibraryRefreshContext } from "./library-refresh-ops";
import type { MusicTrackBoundaryContext } from "./track-boundary-ops";

/**
 * Host passed into MusicContextBinder.
 * Typed as any because MusicStore private fields are not assignable to a public interface.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type MusicStoreHost = any;

export class MusicContextBinder {
  private queueContext: MusicQueueContext | null = null;
  private favoritesContext: MusicFavoritesContext | null = null;
  private playlistContext: MusicPlaylistContext | null = null;
  private libraryBrowseContext: MusicLibraryBrowseContext | null = null;
  private playbackTransportContext: MusicPlaybackTransportContext | null = null;
  private cacheContext: MusicCacheContext | null = null;
  private settingsContext: MusicSettingsContext | null = null;
  private lyricsContext: MusicLyricsContext | null = null;
  private eqContext: MusicEqContext | null = null;
  private playerChromeContext: MusicPlayerChromeContext | null = null;
  private mixContext: MusicMixContext | null = null;
  private radioContext: MusicRadioContext | null = null;
  private playLaunchContext: MusicPlayLaunchContext | null = null;
  private connectContext: MusicConnectContext | null = null;
  private engineContext: MusicEngineContext | null = null;
  private playbackCoreContext: MusicPlaybackCoreContext | null = null;
  private libraryRefreshContext: MusicLibraryRefreshContext | null = null;
  private trackBoundaryContext: MusicTrackBoundaryContext | null = null;

  constructor(
    private readonly store: MusicStoreHost,
    private readonly homeStaleMs: number,
  ) {}

  queueCtx(): MusicQueueContext {
    if (this.queueContext) return this.queueContext;
    const store = this.store;
    this.queueContext = {
      get queue() {
        return store.queue;
      },
      set queue(v) {
        store.queue = v;
      },
      get queueIndex() {
        return store.queueIndex;
      },
      set queueIndex(v) {
        store.queueIndex = v;
      },
      get shuffle() {
        return store.shuffle;
      },
      set shuffle(v) {
        store.shuffle = v;
      },
      get shuffleUpcoming() {
        return store.shuffleUpcoming;
      },
      set shuffleUpcoming(v) {
        store.shuffleUpcoming = v;
      },
      get shuffleHistory() {
        return store.shuffleHistory;
      },
      set shuffleHistory(v) {
        store.shuffleHistory = v;
      },
      get queueSettings() {
        return store.queueSettings;
      },
      set queueSettings(v) {
        store.queueSettings = v;
      },
      get engine() {
        return store.engine;
      },
      get playing() {
        return store.playing;
      },
      set playing(v) {
        store.playing = v;
      },
      get continuousMode() {
        return store.continuousMode;
      },
      set continuousMode(v) {
        store.continuousMode = v;
      },
      get queueOpen() {
        return store.queueOpen;
      },
      set queueOpen(v) {
        store.queueOpen = v;
      },
      get currentTrack() {
        return store.currentTrack;
      },
      get playerLayout() {
        return store.playerLayout;
      },
      set playerLayout(v) {
        store.playerLayout = v;
      },
      get pendingStartAt() {
        return store.pendingStartAt;
      },
      set pendingStartAt(v) {
        store.pendingStartAt = v;
      },
      get pendingStartPaused() {
        return store.pendingStartPaused;
      },
      set pendingStartPaused(v) {
        store.pendingStartPaused = v;
      },
      playTracks: (tracks, startIndex) => store.playTracks(tracks, startIndex),
      prefetchAround: () => store.prefetchAround(),
      persistPlaybackState: () => store.persistPlaybackState(),
      requestPlayCurrent: () => store.requestPlayCurrent(),
      seedShuffleUpcoming: () => store.seedShuffleUpcoming(),
      stopSmoothProgress: () => store.stopSmoothProgress(),
    };
    return this.queueContext;
  }

  favoritesCtx(): MusicFavoritesContext {
    if (this.favoritesContext) return this.favoritesContext;
    const store = this.store;
    this.favoritesContext = {
      get connected() {
        return store.connected;
      },
      get library() {
        return store.library;
      },
      get favoriteTracks() {
        return store.favoriteTracks;
      },
      set favoriteTracks(v) {
        store.favoriteTracks = v;
      },
      get favoriteIds() {
        return store.favoriteIds;
      },
      set favoriteIds(v) {
        store.favoriteIds = v;
      },
      get favoriteAlbums() {
        return store.favoriteAlbums;
      },
      set favoriteAlbums(v) {
        store.favoriteAlbums = v;
      },
      get favoriteAlbumIds() {
        return store.favoriteAlbumIds;
      },
      set favoriteAlbumIds(v) {
        store.favoriteAlbumIds = v;
      },
      get favoriteArtists() {
        return store.favoriteArtists;
      },
      set favoriteArtists(v) {
        store.favoriteArtists = v;
      },
      get favoriteArtistIds() {
        return store.favoriteArtistIds;
      },
      set favoriteArtistIds(v) {
        store.favoriteArtistIds = v;
      },
      get shuffle() {
        return store.shuffle;
      },
      set shuffle(v) {
        store.shuffle = v;
      },
      isFavorite: (trackId) => store.isFavorite(trackId),
      isFavoriteAlbum: (albumId) => store.isFavoriteAlbum(albumId),
      isFavoriteArtist: (artistId) => store.isFavoriteArtist(artistId),
      playArtistAlbums: (albums, shuffle) =>
        store.libraryBrowseOps.playArtistAlbums(albums, shuffle),
      playTracks: (tracks, startIndex) => store.playTracks(tracks, startIndex),
    };
    return this.favoritesContext;
  }

  playlistCtx(): MusicPlaylistContext {
    if (this.playlistContext) return this.playlistContext;
    const store = this.store;
    this.playlistContext = {
      get connected() {
        return store.connected;
      },
      get hasSubsonicActive() {
        return sources.hasSubsonicActive;
      },
      get client() {
        return store.client;
      },
      get library() {
        return store.library;
      },
      get playlists() {
        return store.playlists;
      },
      set playlists(v) {
        store.playlists = v;
      },
      get serverPlaylists() {
        return store.serverPlaylists;
      },
      set serverPlaylists(v) {
        store.serverPlaylists = v;
      },
      get internetRadios() {
        return store.internetRadios;
      },
      set internetRadios(v) {
        store.internetRadios = v;
      },
    };
    return this.playlistContext;
  }

  libraryBrowseCtx(): MusicLibraryBrowseContext {
    if (this.libraryBrowseContext) return this.libraryBrowseContext;
    const store = this.store;
    this.libraryBrowseContext = {
      get library() {
        return store.library;
      },
      get recentAlbums() {
        return store.recentAlbums;
      },
      set recentAlbums(v) {
        store.recentAlbums = v;
      },
      get frequentAlbums() {
        return store.frequentAlbums;
      },
      set frequentAlbums(v) {
        store.frequentAlbums = v;
      },
      get randomTracks() {
        return store.randomTracks;
      },
      set randomTracks(v) {
        store.randomTracks = v;
      },
      get resumeTracks() {
        return store.resumeTracks;
      },
      set resumeTracks(v) {
        store.resumeTracks = v;
      },
      get homeCoreFetchedAt() {
        return store.homeCoreFetchedAt;
      },
      set homeCoreFetchedAt(v) {
        store.homeCoreFetchedAt = v;
      },
      get listenHistory() {
        return store.listenHistory;
      },
      set listenHistory(v) {
        store.listenHistory = v;
      },
      get stats() {
        return store.stats;
      },
      set stats(v) {
        store.stats = v;
      },
      get libraryStats() {
        return store.libraryStats;
      },
      set libraryStats(v) {
        store.libraryStats = v;
      },
      get genres() {
        return store.genres;
      },
      set genres(v) {
        store.genres = v;
      },
      get allArtists() {
        return store.allArtists;
      },
      set allArtists(v) {
        store.allArtists = v;
      },
      get artistsLoaded() {
        return store.artistsLoaded;
      },
      set artistsLoaded(v) {
        store.artistsLoaded = v;
      },
      get artistsLoadedAt() {
        return store.artistsLoadedAt;
      },
      set artistsLoadedAt(v) {
        store.artistsLoadedAt = v;
      },
      get shuffle() {
        return store.shuffle;
      },
      set shuffle(v) {
        store.shuffle = v;
      },
      playTracks: (tracks, startIndex) => store.playTracks(tracks, startIndex),
      addTracksToQueue: (tracks) => store.addTracksToQueue(tracks),
      playTracksNext: (tracks) => store.playTracksNext(tracks),
    };
    return this.libraryBrowseContext;
  }

  playbackTransportCtx(): MusicPlaybackTransportContext {
    if (this.playbackTransportContext) return this.playbackTransportContext;
    const store = this.store;
    this.playbackTransportContext = {
      get engine() {
        return store.engine;
      },
      get playing() {
        return store.playing;
      },
      set playing(v) {
        store.playing = v;
      },
      get volume() {
        return store.volume;
      },
      set volume(v) {
        store.volume = v;
      },
      get repeat() {
        return store.repeat;
      },
      set repeat(v) {
        store.repeat = v;
      },
      get shuffle() {
        return store.shuffle;
      },
      set shuffle(v) {
        store.shuffle = v;
      },
      get queue() {
        return store.queue;
      },
      get queueIndex() {
        return store.queueIndex;
      },
      set queueIndex(v) {
        store.queueIndex = v;
      },
      get currentTrack() {
        return store.currentTrack;
      },
      get currentTime() {
        return store.currentTime;
      },
      set currentTime(v) {
        store.currentTime = v;
      },
      get duration() {
        return store.duration;
      },
      set duration(v) {
        store.duration = v;
      },
      get smoothProgress() {
        return store.smoothProgress;
      },
      set smoothProgress(v) {
        store.smoothProgress = v;
      },
      get continuousMode() {
        return store.continuousMode;
      },
      get shuffleUpcoming() {
        return store.shuffleUpcoming;
      },
      set shuffleUpcoming(v) {
        store.shuffleUpcoming = v;
      },
      get shuffleHistory() {
        return store.shuffleHistory;
      },
      set shuffleHistory(v) {
        store.shuffleHistory = v;
      },
      get personalRadio() {
        return store.personalRadio;
      },
      get lastSavedPosition() {
        return store.lastSavedPosition;
      },
      set lastSavedPosition(v) {
        store.lastSavedPosition = v;
      },
      stopSmoothProgress: () => store.stopSmoothProgress(),
      syncMediaSession: () => store.syncMediaSession(),
      initEngine: () => store.initEngine(),
      playCurrent: () => store.playCurrent(),
      requestPlayCurrent: () => store.requestPlayCurrent(),
      advanceShuffleIndex: () => store.advanceShuffleIndex(),
      stopAtQueueEnd: () => store.stopAtQueueEnd(),
      maybeRefillContinuousQueue: () => store.maybeRefillContinuousQueue(),
      persistPlaybackState: () => store.persistPlaybackState(),
      saveProgress: (positionMs, options) =>
        store.saveProgress(positionMs, options),
      prefetchAround: () => store.prefetchAround(),
      startSmoothProgress: () => store.startSmoothProgress(),
    };
    return this.playbackTransportContext;
  }

  cacheCtx(): MusicCacheContext {
    if (this.cacheContext) return this.cacheContext;
    const store = this.store;
    this.cacheContext = {
      get downloadedIds() {
        return store.downloadedIds;
      },
      set downloadedIds(v) {
        store.downloadedIds = v;
      },
      get cacheSettings() {
        return store.cacheSettings;
      },
      set cacheSettings(v) {
        store.cacheSettings = v;
      },
      get cacheUsedBytes() {
        return store.cacheUsedBytes;
      },
      set cacheUsedBytes(v) {
        store.cacheUsedBytes = v;
      },
      get cacheTrackCount() {
        return store.cacheTrackCount;
      },
      set cacheTrackCount(v) {
        store.cacheTrackCount = v;
      },
      get cacheInFlight() {
        return store.cacheInFlight;
      },
      get cacheStrategyRunning() {
        return store.cacheStrategyRunning;
      },
      set cacheStrategyRunning(v) {
        store.cacheStrategyRunning = v;
      },
      get connected() {
        return store.connected;
      },
      get shuffle() {
        return store.shuffle;
      },
      get repeat() {
        return store.repeat;
      },
      get queue() {
        return store.queue;
      },
      get listenHistory() {
        return store.listenHistory;
      },
      get frequentAlbums() {
        return store.frequentAlbums;
      },
      get playlists() {
        return store.playlists;
      },
      get library() {
        return store.library;
      },
      get offlineDownloadProgress() {
        return store.offlineDownloadProgress;
      },
      set offlineDownloadProgress(v) {
        store.offlineDownloadProgress = v;
      },
      get offlineDownloadAbort() {
        return store.offlineDownloadAbort;
      },
      set offlineDownloadAbort(v) {
        store.offlineDownloadAbort = v;
      },
      sequentialNextIndex: () => store.sequentialNextIndex(),
      runCacheStrategy: () => store.cacheOps.runCacheStrategy(),
      downloadCurrentOr: (track) => store.cacheOps.downloadCurrentOr(track),
      loadCacheSettings: () => store.cacheOps.loadCacheSettings(),
      collectStrategyTracks: () => store.cacheOps.collectStrategyTracks(),
      prefetchCache: (track) => store.cacheOps.prefetchCache(track),
      removeDownload: (trackId) => store.cacheOps.removeDownload(trackId),
      cancelOfflineDownload: () => store.cacheOps.cancelOfflineDownload(),
    };
    return this.cacheContext;
  }

  settingsCtx(): MusicSettingsContext {
    if (this.settingsContext) return this.settingsContext;
    const store = this.store;
    this.settingsContext = {
      get transcodingSettings() {
        return store.transcodingSettings;
      },
      set transcodingSettings(v) {
        store.transcodingSettings = v;
      },
      get immersiveAudioSettings() {
        return store.immersiveAudioSettings;
      },
      set immersiveAudioSettings(v) {
        store.immersiveAudioSettings = v;
      },
      get queueSettings() {
        return store.queueSettings;
      },
      set queueSettings(v) {
        store.queueSettings = v;
      },
      get playbackSettings() {
        return store.playbackSettings;
      },
      set playbackSettings(v) {
        store.playbackSettings = v;
      },
      get metadataEnhancementSettings() {
        return store.metadataEnhancementSettings;
      },
      set metadataEnhancementSettings(v) {
        store.metadataEnhancementSettings = v;
      },
      get hideUnknownMetadata() {
        return store.hideUnknownMetadata;
      },
      set hideUnknownMetadata(v) {
        store.hideUnknownMetadata = v;
      },
      get engine() {
        return store.engine;
      },
      applyQueueLimit: () => store.applyQueueLimit(),
    };
    return this.settingsContext;
  }

  lyricsCtx(): MusicLyricsContext {
    if (this.lyricsContext) return this.lyricsContext;
    const store = this.store;
    this.lyricsContext = {
      get lyricsOpen() {
        return store.lyricsOpen;
      },
      set lyricsOpen(v) {
        store.lyricsOpen = v;
      },
      get currentTrack() {
        return store.currentTrack;
      },
      get currentLyrics() {
        return store.currentLyrics;
      },
      set currentLyrics(v) {
        store.currentLyrics = v;
      },
      get lyricsLoading() {
        return store.lyricsLoading;
      },
      set lyricsLoading(v) {
        store.lyricsLoading = v;
      },
      get lyricsFetching() {
        return store.lyricsFetching;
      },
      set lyricsFetching(v) {
        store.lyricsFetching = v;
      },
      get lyricsRequestId() {
        return store.lyricsRequestId;
      },
      set lyricsRequestId(v) {
        store.lyricsRequestId = v;
      },
      get pendingLyricsReload() {
        return store.pendingLyricsReload;
      },
      set pendingLyricsReload(v) {
        store.pendingLyricsReload = v;
      },
      get lyricsSettings() {
        return store.lyricsSettings;
      },
      set lyricsSettings(v) {
        store.lyricsSettings = v;
      },
      get favoriteTracks() {
        return store.favoriteTracks;
      },
      get listenHistory() {
        return store.listenHistory;
      },
      get resumeTracks() {
        return store.resumeTracks;
      },
      get randomTracks() {
        return store.randomTracks;
      },
      get library() {
        return store.library;
      },
      get config() {
        return store.config;
      },
      seek: (seconds) => store.seek(seconds),
      favoriteToSong: (entry) => store.favoriteToSong(entry),
      entryToSong: (entry) => store.entryToSong(entry),
    };
    return this.lyricsContext;
  }

  eqCtx(): MusicEqContext {
    if (this.eqContext) return this.eqContext;
    const store = this.store;
    this.eqContext = {
      get authEnabled() {
        return store.authEnabled;
      },
      set authEnabled(v) {
        store.authEnabled = v;
      },
      get eq() {
        return store.eq;
      },
      set eq(v) {
        store.eq = v;
      },
      get eqAvailable() {
        return store.eqAvailable;
      },
      get eqOpen() {
        return store.eqOpen;
      },
      set eqOpen(v) {
        store.eqOpen = v;
      },
      get engine() {
        return store.engine;
      },
    };
    return this.eqContext;
  }

  playerChromeCtx(): MusicPlayerChromeContext {
    if (this.playerChromeContext) return this.playerChromeContext;
    const store = this.store;
    this.playerChromeContext = {
      get routePath() {
        return store.routePath;
      },
      set routePath(v) {
        store.routePath = v;
      },
      get playing() {
        return store.playing;
      },
      set playing(v) {
        store.playing = v;
      },
      get playerLayout() {
        return store.playerLayout;
      },
      set playerLayout(v) {
        store.playerLayout = v;
      },
      get currentTrack() {
        return store.currentTrack;
      },
      get queueOpen() {
        return store.queueOpen;
      },
      set queueOpen(v) {
        store.queueOpen = v;
      },
      get engine() {
        return store.engine;
      },
      initEngine: () => store.initEngine(),
      stopSmoothProgress: () => store.stopSmoothProgress(),
      syncMediaSession: () => store.syncMediaSession(),
    };
    return this.playerChromeContext;
  }

  mixCtx(): MusicMixContext {
    if (this.mixContext) return this.mixContext;
    const store = this.store;
    this.mixContext = {
      get config() {
        return store.config;
      },
      get personalMixes() {
        return store.personalMixes;
      },
      set personalMixes(v) {
        store.personalMixes = v;
      },
      get mixSettings() {
        return store.mixSettings;
      },
      set mixSettings(v) {
        store.mixSettings = v;
      },
      get mixesRegenerating() {
        return store.mixesRegenerating;
      },
      set mixesRegenerating(v) {
        store.mixesRegenerating = v;
      },
      get mixesRefreshInFlight() {
        return store.mixesRefreshInFlight;
      },
      set mixesRefreshInFlight(v) {
        store.mixesRefreshInFlight = v;
      },
      get personalizationFetchedAt() {
        return store.personalizationFetchedAt;
      },
      set personalizationFetchedAt(v) {
        store.personalizationFetchedAt = v;
      },
      get connected() {
        return store.connected;
      },
      get stats() {
        return store.stats;
      },
      set stats(v) {
        store.stats = v;
      },
      get listenHistory() {
        return store.listenHistory;
      },
      set listenHistory(v) {
        store.listenHistory = v;
      },
      get frequentAlbums() {
        return store.frequentAlbums;
      },
      get recommendations() {
        return store.recommendations;
      },
      set recommendations(v) {
        store.recommendations = v;
      },
      get library() {
        return store.library;
      },
      get shuffle() {
        return store.shuffle;
      },
      set shuffle(v) {
        store.shuffle = v;
      },
      entryToSong: (entry) => store.entryToSong(entry),
      resolveTracksForRestore: (ids) => store.resolveTracksForRestore(ids),
      playTracks: (tracks, startIndex) => store.playTracks(tracks, startIndex),
      refreshMixes: () => store.mixOps.refreshMixes(),
      doRefreshMixes: (force) => store.mixOps.doRefreshMixes(force),
      refreshRecommendations: () => store.mixOps.refreshRecommendations(),
      hydrateMixTracks: (mix) => store.mixOps.hydrateMixTracks(mix),
      replaceMixWithHydrated: (hydrated) =>
        store.mixOps.replaceMixWithHydrated(hydrated),
    };
    return this.mixContext;
  }

  radioCtx(): MusicRadioContext {
    if (this.radioContext) return this.radioContext;
    const store = this.store;
    this.radioContext = {
      get continuousBusy() {
        return store.continuousBusy;
      },
      set continuousBusy(v) {
        store.continuousBusy = v;
      },
      get continuousMode() {
        return store.continuousMode;
      },
      set continuousMode(v) {
        store.continuousMode = v;
      },
      get shuffle() {
        return store.shuffle;
      },
      set shuffle(v) {
        store.shuffle = v;
      },
      get autoplay() {
        return store.autoplay;
      },
      set autoplay(v) {
        store.autoplay = v;
      },
      get repeat() {
        return store.repeat;
      },
      set repeat(v) {
        store.repeat = v;
      },
      get library() {
        return store.library;
      },
      get libraryPool() {
        return store.libraryPool;
      },
      set libraryPool(v) {
        store.libraryPool = v;
      },
      get personalRadio() {
        return store.personalRadio;
      },
      set personalRadio(v) {
        store.personalRadio = v;
      },
      get queueSettings() {
        return store.queueSettings;
      },
      get listenHistory() {
        return store.listenHistory;
      },
      get stats() {
        return store.stats;
      },
      get mixSettings() {
        return store.mixSettings;
      },
      get internetRadios() {
        return store.internetRadios;
      },
      playTracks: (tracks, startIndex, resume, continuousMode, options) =>
        store.playTracks(tracks, startIndex, resume, continuousMode, options),
      startRandomRadio: (count) => store.radioOps.startRandomRadio(count),
      playInternetRadio: (station) => store.radioOps.playInternetRadio(station),
      refreshInternetRadios: () => store.refreshInternetRadios(),
      persistPlaybackState: () => store.persistPlaybackState(),
      entryToSong: (entry) => store.entryToSong(entry),
      personalRadioOptions: (coldStart) =>
        store.personalRadioOptions(coldStart),
    };
    return this.radioContext;
  }

  playLaunchCtx(): MusicPlayLaunchContext {
    if (this.playLaunchContext) return this.playLaunchContext;
    const store = this.store;
    this.playLaunchContext = {
      get continuousMode() {
        return store.continuousMode;
      },
      set continuousMode(v) {
        store.continuousMode = v;
      },
      get libraryPool() {
        return store.libraryPool;
      },
      set libraryPool(v) {
        store.libraryPool = v;
      },
      get personalRadio() {
        return store.personalRadio;
      },
      set personalRadio(v) {
        store.personalRadio = v;
      },
      get queueSettings() {
        return store.queueSettings;
      },
      get queue() {
        return store.queue;
      },
      set queue(v) {
        store.queue = v;
      },
      get queueIndex() {
        return store.queueIndex;
      },
      set queueIndex(v) {
        store.queueIndex = v;
      },
      get shuffleHistory() {
        return store.shuffleHistory;
      },
      set shuffleHistory(v) {
        store.shuffleHistory = v;
      },
      get shuffleUpcoming() {
        return store.shuffleUpcoming;
      },
      set shuffleUpcoming(v) {
        store.shuffleUpcoming = v;
      },
      get shuffle() {
        return store.shuffle;
      },
      set shuffle(v) {
        store.shuffle = v;
      },
      get autoplay() {
        return store.autoplay;
      },
      set autoplay(v) {
        store.autoplay = v;
      },
      get repeat() {
        return store.repeat;
      },
      set repeat(v) {
        store.repeat = v;
      },
      get playerLayout() {
        return store.playerLayout;
      },
      set playerLayout(v) {
        store.playerLayout = v;
      },
      get resumeOnPlay() {
        return store.resumeOnPlay;
      },
      set resumeOnPlay(v) {
        store.resumeOnPlay = v;
      },
      get connected() {
        return store.connected;
      },
      get library() {
        return store.library;
      },
      seedShuffleUpcoming: () => store.seedShuffleUpcoming(),
      persistPlaybackState: () => store.persistPlaybackState(),
      requestPlayCurrent: () => store.requestPlayCurrent(),
      resolveTracksForRestore: (trackIds) =>
        store.resolveTracksForRestore(trackIds),
      playTracks: (tracks, startIndex, resume, continuousMode, options) =>
        store.playLaunchOps.playTracks(
          tracks,
          startIndex,
          resume,
          continuousMode,
          options,
        ),
      bootstrapOfflinePlayback: () => store.bootstrapOfflinePlayback(),
      isDownloaded: (trackId) => store.isDownloaded(trackId),
      resolveOfflineTrack: (trackId) => store.resolveOfflineTrack(trackId),
      connect: (options) => store.connect(options),
      playTrackById: (trackId) => store.playLaunchOps.playTrackById(trackId),
    };
    return this.playLaunchContext;
  }

  connectCtx(): MusicConnectContext {
    if (this.connectContext) return this.connectContext;
    const store = this.store;
    const homeStaleMs = this.homeStaleMs;
    this.connectContext = {
      get connectInFlight() {
        return store.connectInFlight;
      },
      set connectInFlight(v) {
        store.connectInFlight = v;
      },
      get connected() {
        return store.connected;
      },
      set connected(v) {
        store.connected = v;
      },
      get loading() {
        return store.loading;
      },
      set loading(v) {
        store.loading = v;
      },
      get error() {
        return store.error;
      },
      set error(v) {
        store.error = v;
      },
      get status() {
        return store.status;
      },
      set status(v) {
        store.status = v;
      },
      get libraryWarmup() {
        return store.libraryWarmup;
      },
      set libraryWarmup(v) {
        store.libraryWarmup = v;
      },
      get playbackRestored() {
        return store.playbackRestored;
      },
      set playbackRestored(v) {
        store.playbackRestored = v;
      },
      get playing() {
        return store.playing;
      },
      set playing(v) {
        store.playing = v;
      },
      get queue() {
        return store.queue;
      },
      get queueIndex() {
        return store.queueIndex;
      },
      get reconnectResumePending() {
        return store.reconnectResumePending;
      },
      set reconnectResumePending(v) {
        store.reconnectResumePending = v;
      },
      get reconnectPositionMs() {
        return store.reconnectPositionMs;
      },
      set reconnectPositionMs(v) {
        store.reconnectPositionMs = v;
      },
      get currentTime() {
        return store.currentTime;
      },
      get currentTrack() {
        return store.currentTrack;
      },
      get engine() {
        return store.engine;
      },
      get playbackEpoch() {
        return store.playbackEpoch;
      },
      get homeCoreFetchedAt() {
        return store.homeCoreFetchedAt;
      },
      get homeStaleMs() {
        return homeStaleMs;
      },
      restoreCachedMixes: () => store.restoreCachedMixes(),
      initEngine: () => store.initEngine(),
      loadEqSettings: (authEnabled) => store.loadEqSettings(authEnabled),
      refreshHomeCore: () => store.refreshHomeCore(),
      loadCacheSettings: () => store.loadCacheSettings(),
      loadLyricsSettings: () => store.loadLyricsSettings(),
      restorePlayback: () => store.restorePlayback(),
      startLibraryWatch: () => store.startLibraryWatch(),
      stopLibraryWatch: () => store.stopLibraryWatch(),
      refreshHistory: (limit) => store.refreshHistory(limit),
      refreshStats: (limit) => store.refreshStats(limit),
      refreshPlaylists: () => store.refreshPlaylists(),
      refreshServerPlaylists: () => store.refreshServerPlaylists(),
      refreshInternetRadios: () => store.refreshInternetRadios(),
      refreshFavorites: () => store.refreshFavorites(),
      refreshLibraryStats: () => store.refreshLibraryStats(),
      loadDownloads: () => store.loadDownloads(),
      schedulePersonalizationRefresh: (force) =>
        store.schedulePersonalizationRefresh(force),
      runCacheStrategy: () => store.runCacheStrategy(),
      isDownloaded: (trackId) => store.isDownloaded(trackId),
      loadTrackSource: (token, track, url) =>
        store.loadTrackSource(token, track, url),
      trackStreamUrl: (track) => store.trackStreamUrl(track),
      startProgressTracking: () => store.startProgressTracking(),
      startSmoothProgress: () => store.startSmoothProgress(),
      stopSmoothProgress: () => store.stopSmoothProgress(),
      syncMediaSession: () => store.syncMediaSession(),
      handlePlaybackNetworkFailure: (token, options) =>
        store.handlePlaybackNetworkFailure(token, options),
      playCurrent: () => store.playCurrent(),
      seek: (seconds) => store.seek(seconds),
    };
    return this.connectContext;
  }

  engineCtx(): MusicEngineContext {
    if (this.engineContext) return this.engineContext;
    const store = this.store;
    this.engineContext = {
      get engine() {
        return store.engine;
      },
      set engine(v) {
        store.engine = v;
      },
      get engineInitPromise() {
        return store.engineInitPromise;
      },
      set engineInitPromise(v) {
        store.engineInitPromise = v;
      },
      get nativePlayback() {
        return store.nativePlayback;
      },
      set nativePlayback(v) {
        store.nativePlayback = v;
      },
      get nativeAvailable() {
        return store.nativeAvailable;
      },
      set nativeAvailable(v) {
        store.nativeAvailable = v;
      },
      get mpvAvailable() {
        return store.mpvAvailable;
      },
      set mpvAvailable(v) {
        store.mpvAvailable = v;
      },
      get vlcAvailable() {
        return store.vlcAvailable;
      },
      set vlcAvailable(v) {
        store.vlcAvailable = v;
      },
      get nativeBackend() {
        return store.nativeBackend;
      },
      set nativeBackend(v) {
        store.nativeBackend = v;
      },
      get nativeInitError() {
        return store.nativeInitError;
      },
      set nativeInitError(v) {
        store.nativeInitError = v;
      },
      get mpvLoadError() {
        return store.mpvLoadError;
      },
      set mpvLoadError(v) {
        store.mpvLoadError = v;
      },
      get volume() {
        return store.volume;
      },
      get immersiveAudioSettings() {
        return store.immersiveAudioSettings;
      },
      get eq() {
        return store.eq;
      },
      get eqOpen() {
        return store.eqOpen;
      },
      set eqOpen(v) {
        store.eqOpen = v;
      },
      get cleanupListeners() {
        return store.cleanupListeners;
      },
      set cleanupListeners(v) {
        store.cleanupListeners = v;
      },
      get currentTime() {
        return store.currentTime;
      },
      set currentTime(v) {
        store.currentTime = v;
      },
      get duration() {
        return store.duration;
      },
      set duration(v) {
        store.duration = v;
      },
      get currentTrack() {
        return store.currentTrack;
      },
      get playing() {
        return store.playing;
      },
      set playing(v) {
        store.playing = v;
      },
      get lastSavedPosition() {
        return store.lastSavedPosition;
      },
      set lastSavedPosition(v) {
        store.lastSavedPosition = v;
      },
      get playbackEpoch() {
        return store.playbackEpoch;
      },
      set playbackEpoch(v) {
        store.playbackEpoch = v;
      },
      get nextTrackPrepared() {
        return store.nextTrackPrepared;
      },
      set nextTrackPrepared(v) {
        store.nextTrackPrepared = v;
      },
      get nativeFallbackAttempted() {
        return store.nativeFallbackAttempted;
      },
      set nativeFallbackAttempted(v) {
        store.nativeFallbackAttempted = v;
      },
      get transcodedTrackIds() {
        return store.transcodedTrackIds;
      },
      onTrackEnded: () => store.onTrackEnded(),
      maybePrepareNext: () => store.maybePrepareNext(),
      maybeStartCrossfade: () => store.maybeStartCrossfade(),
      saveProgress: (positionMs, options) =>
        store.saveProgress(positionMs, options),
      persistPlaybackState: () => store.persistPlaybackState(),
      handlePlaybackNetworkFailure: (token) =>
        store.handlePlaybackNetworkFailure(token),
      skipFailedTrack: (epoch, reason) => store.skipFailedTrack(epoch, reason),
      markTrackTranscoded: (trackId) => store.markTrackTranscoded(trackId),
      playCurrent: (epoch, resume) => store.playCurrent(epoch, resume),
      stopSmoothProgress: () => store.stopSmoothProgress(),
      syncMediaSession: () => store.syncMediaSession(),
      trackStreamUrl: (track) => store.trackStreamUrl(track),
      loadTrackSource: (token, track, url) =>
        store.loadTrackSource(token, track, url),
      prefetchAround: () => store.prefetchAround(),
      startProgressTracking: () => store.startProgressTracking(),
      startSmoothProgress: () => store.startSmoothProgress(),
    };
    return this.engineContext;
  }

  playbackCoreCtx(): MusicPlaybackCoreContext {
    if (this.playbackCoreContext) return this.playbackCoreContext;
    const store = this.store;
    this.playbackCoreContext = {
      get playbackEpoch() {
        return store.playbackEpoch;
      },
      set playbackEpoch(v) {
        store.playbackEpoch = v;
      },
      get resumeOnPlay() {
        return store.resumeOnPlay;
      },
      set resumeOnPlay(v) {
        store.resumeOnPlay = v;
      },
      get playCurrentTail() {
        return store.playCurrentTail;
      },
      set playCurrentTail(v) {
        store.playCurrentTail = v;
      },
      get pendingStartAt() {
        return store.pendingStartAt;
      },
      set pendingStartAt(v) {
        store.pendingStartAt = v;
      },
      get pendingStartPaused() {
        return store.pendingStartPaused;
      },
      set pendingStartPaused(v) {
        store.pendingStartPaused = v;
      },
      get transcodedTrackIds() {
        return store.transcodedTrackIds;
      },
      set transcodedTrackIds(v) {
        store.transcodedTrackIds = v;
      },
      get failedTrackSkips() {
        return store.failedTrackSkips;
      },
      set failedTrackSkips(v) {
        store.failedTrackSkips = v;
      },
      get playing() {
        return store.playing;
      },
      set playing(v) {
        store.playing = v;
      },
      get currentTrack() {
        return store.currentTrack;
      },
      get currentTime() {
        return store.currentTime;
      },
      set currentTime(v) {
        store.currentTime = v;
      },
      get duration() {
        return store.duration;
      },
      get smoothProgress() {
        return store.smoothProgress;
      },
      set smoothProgress(v) {
        store.smoothProgress = v;
      },
      get engine() {
        return store.engine;
      },
      get nextTrackPrepared() {
        return store.nextTrackPrepared;
      },
      set nextTrackPrepared(v) {
        store.nextTrackPrepared = v;
      },
      get playbackSettings() {
        return store.playbackSettings;
      },
      get nativePlayback() {
        return store.nativePlayback;
      },
      get error() {
        return store.error;
      },
      set error(v) {
        store.error = v;
      },
      get resumeTracks() {
        return store.resumeTracks;
      },
      get listenHistory() {
        return store.listenHistory;
      },
      get downloadedIds() {
        return store.downloadedIds;
      },
      get config() {
        return store.config;
      },
      get transcodingSettings() {
        return store.transcodingSettings;
      },
      get immersiveAudioSettings() {
        return store.immersiveAudioSettings;
      },
      get lyricsOpen() {
        return store.lyricsOpen;
      },
      get queue() {
        return store.queue;
      },
      get queueIndex() {
        return store.queueIndex;
      },
      get shuffle() {
        return store.shuffle;
      },
      get autoplay() {
        return store.autoplay;
      },
      get continuousMode() {
        return store.continuousMode;
      },
      get lastSavedPosition() {
        return store.lastSavedPosition;
      },
      set lastSavedPosition(v) {
        store.lastSavedPosition = v;
      },
      syncMediaSession: () => store.syncMediaSession(),
      initEngine: () => store.initEngine(),
      loadTrackSource: (token, track, url) =>
        store.loadTrackSource(token, track, url),
      handlePlaybackNetworkFailure: (token, options) =>
        store.handlePlaybackNetworkFailure(token, options),
      skipFailedTrack: (token, msg) => store.skipFailedTrack(token, msg),
      prefetchAround: () => store.prefetchAround(),
      startProgressTracking: () => store.startProgressTracking(),
      startSmoothProgress: () => store.startSmoothProgress(),
      recordNowPlaying: (track) => store.recordNowPlaying(track),
      enrichCurrentTrack: (trackId, token) =>
        store.enrichCurrentTrack(trackId, token),
      maybeCacheTrack: (track) => store.maybeCacheTrack(track),
      loadCurrentLyrics: () => store.loadCurrentLyrics(),
    };
    return this.playbackCoreContext;
  }

  libraryRefreshCtx(): MusicLibraryRefreshContext {
    if (this.libraryRefreshContext) return this.libraryRefreshContext;
    const store = this.store;
    this.libraryRefreshContext = {
      get libraryRefreshing() {
        return store.libraryRefreshing;
      },
      set libraryRefreshing(v) {
        store.libraryRefreshing = v;
      },
      get connected() {
        return store.connected;
      },
      get homeCoreFetchedAt() {
        return store.homeCoreFetchedAt;
      },
      set homeCoreFetchedAt(v) {
        store.homeCoreFetchedAt = v;
      },
      get personalizationFetchedAt() {
        return store.personalizationFetchedAt;
      },
      set personalizationFetchedAt(v) {
        store.personalizationFetchedAt = v;
      },
      get libraryStats() {
        return store.libraryStats;
      },
      set libraryStats(v) {
        store.libraryStats = v;
      },
      get libraryRevision() {
        return store.libraryRevision;
      },
      set libraryRevision(v) {
        store.libraryRevision = v;
      },
      get lastLibrarySongCount() {
        return store.lastLibrarySongCount;
      },
      set lastLibrarySongCount(v) {
        store.lastLibrarySongCount = v;
      },
      get lastLibraryScanning() {
        return store.lastLibraryScanning;
      },
      set lastLibraryScanning(v) {
        store.lastLibraryScanning = v;
      },
      get libraryWatchTimer() {
        return store.libraryWatchTimer;
      },
      set libraryWatchTimer(v) {
        store.libraryWatchTimer = v;
      },
      invalidateArtistIndex: () =>
        store.libraryBrowseOps.invalidateArtistIndex(),
      refreshHomeCore: () => store.refreshHomeCore(),
      refreshLibraryStats: (options) => store.refreshLibraryStats(options),
      refreshFavorites: () => store.refreshFavorites(),
      refreshServerPlaylists: () => store.refreshServerPlaylists(),
      loadArtists: (options) => store.loadArtists(options),
      schedulePersonalizationRefresh: (force) =>
        store.schedulePersonalizationRefresh(force),
      refreshLibrary: (options) => store.refreshLibrary(options),
      stopLibraryWatch: () => store.stopLibraryWatch(),
      scheduleLibraryWatch: (delayMs) => store.scheduleLibraryWatch(delayMs),
      pollLibraryChanges: () => store.pollLibraryChanges(),
    };
    return this.libraryRefreshContext;
  }

  trackBoundaryCtx(): MusicTrackBoundaryContext {
    if (this.trackBoundaryContext) return this.trackBoundaryContext;
    const store = this.store;
    this.trackBoundaryContext = {
      get trackBoundaryBusy() {
        return store.trackBoundaryBusy;
      },
      set trackBoundaryBusy(v) {
        store.trackBoundaryBusy = v;
      },
      get crossfadeHandled() {
        return store.crossfadeHandled;
      },
      set crossfadeHandled(v) {
        store.crossfadeHandled = v;
      },
      get currentTrack() {
        return store.currentTrack;
      },
      get playing() {
        return store.playing;
      },
      set playing(v) {
        store.playing = v;
      },
      get engine() {
        return store.engine;
      },
      get currentTime() {
        return store.currentTime;
      },
      set currentTime(v) {
        store.currentTime = v;
      },
      get playbackEpoch() {
        return store.playbackEpoch;
      },
      get smoothProgress() {
        return store.smoothProgress;
      },
      set smoothProgress(v) {
        store.smoothProgress = v;
      },
      get lastSavedPosition() {
        return store.lastSavedPosition;
      },
      set lastSavedPosition(v) {
        store.lastSavedPosition = v;
      },
      get playbackRestored() {
        return store.playbackRestored;
      },
      set playbackRestored(v) {
        store.playbackRestored = v;
      },
      get queue() {
        return store.queue;
      },
      set queue(v) {
        store.queue = v;
      },
      get queueIndex() {
        return store.queueIndex;
      },
      set queueIndex(v) {
        store.queueIndex = v;
      },
      get shuffle() {
        return store.shuffle;
      },
      set shuffle(v) {
        store.shuffle = v;
      },
      get autoplay() {
        return store.autoplay;
      },
      set autoplay(v) {
        store.autoplay = v;
      },
      get continuousMode() {
        return store.continuousMode;
      },
      set continuousMode(v) {
        store.continuousMode = v;
      },
      get repeat() {
        return store.repeat;
      },
      get shuffleUpcoming() {
        return store.shuffleUpcoming;
      },
      set shuffleUpcoming(v) {
        store.shuffleUpcoming = v;
      },
      get shuffleHistory() {
        return store.shuffleHistory;
      },
      set shuffleHistory(v) {
        store.shuffleHistory = v;
      },
      get queueSettings() {
        return store.queueSettings;
      },
      get continuousRefillInFlight() {
        return store.continuousRefillInFlight;
      },
      set continuousRefillInFlight(v) {
        store.continuousRefillInFlight = v;
      },
      get libraryPool() {
        return store.libraryPool;
      },
      get personalRadio() {
        return store.personalRadio;
      },
      get listenHistory() {
        return store.listenHistory;
      },
      get stats() {
        return store.stats;
      },
      get library() {
        return store.library;
      },
      get nativePlayback() {
        return store.nativePlayback;
      },
      get transcodedTrackIds() {
        return store.transcodedTrackIds;
      },
      get failedTrackSkips() {
        return store.failedTrackSkips;
      },
      set failedTrackSkips(v) {
        store.failedTrackSkips = v;
      },
      get error() {
        return store.error;
      },
      set error(v) {
        store.error = v;
      },
      get playerLayout() {
        return store.playerLayout;
      },
      set playerLayout(v) {
        store.playerLayout = v;
      },
      get internetRadios() {
        return store.internetRadios;
      },
      isDownloaded: (trackId) => store.isDownloaded(trackId),
      trackStreamUrl: (track) => store.trackStreamUrl(track),
      loadTrackSource: (token, track, url) =>
        store.loadTrackSource(token, track, url),
      startProgressTracking: () => store.startProgressTracking(),
      startSmoothProgress: () => store.startSmoothProgress(),
      stopSmoothProgress: () => store.stopSmoothProgress(),
      syncMediaSession: () => store.syncMediaSession(),
      suspendForReconnect: () => store.suspendForReconnect(),
      isSupersededPlaybackError: (message) =>
        store.isSupersededPlaybackError(message),
      markTrackTranscoded: (trackId) => store.markTrackTranscoded(trackId),
      maybeRefillContinuousQueue: () => store.maybeRefillContinuousQueue(),
      refillLibraryQueue: (count) => store.refillLibraryQueue(count),
      refillPersonalQueue: (count) => store.refillPersonalQueue(count),
      appendRandomSongsToQueue: (count) =>
        store.appendRandomSongsToQueue(count),
      appendTracksToQueue: (tracks) => store.appendTracksToQueue(tracks),
      advanceTrack: () => store.advanceTrack(),
      stopAtQueueEnd: () => store.stopAtQueueEnd(),
      recordPlayCompletion: (track) => store.recordPlayCompletion(track),
      personalRadioOptions: (coldStart) =>
        store.personalRadioOptions(coldStart),
      entryToSong: (entry) => store.entryToSong(entry),
      persistPlaybackState: () => store.persistPlaybackState(),
      seedShuffleUpcoming: () => store.seedShuffleUpcoming(),
      initEngine: () => store.initEngine(),
      resolveTracksForRestore: (trackIds) =>
        store.resolveTracksForRestore(trackIds),
      refreshInternetRadios: () => store.refreshInternetRadios(),
    };
    return this.trackBoundaryContext;
  }
}
