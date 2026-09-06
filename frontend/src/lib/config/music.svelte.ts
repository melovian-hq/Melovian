// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  createDefaultSubsonicConfig,
  SubsonicClient,
  coverArtUrl,
  isInternetRadioTrack,
  radioStationIdFromTrackId,
  trackFromRadioStation,
  type SubsonicAlbum,
  type SubsonicArtist,
  type SubsonicGenre,
  type SubsonicSearchResult,
  type SubsonicSong,
  type ServerPlaylist,
  type InternetRadioStation,
  type ParsedLyrics,
  type LyricsSearchHit,
  type QueueTrack,
} from "$lib/subsonic";
import { absoluteMobileArtworkUrl } from "$lib/music/mobile-media-artwork";
import { createBoundedSet } from "$lib/core/bounded-cache";
import {
  createLocalLibraryAdapter,
  createSubsonicLibraryAdapter,
  createUnifiedLibraryAdapter,
} from "$lib/music/library-adapter";
import {
  loadMetadataEnhancementSettings,
  type MetadataEnhancementSettings,
} from "$lib/music/metadata-enhancement-settings";
import { loadHideUnknownMetadata } from "$lib/music/library-display-settings";
import * as musicApi from "$lib/music/api";
import { toRockskyTrack } from "$lib/music/rocksky";
import type {
  ListenEntry,
  ListenStats,
  MusicPlaylist,
  MusicStatus,
  FavoriteTrack,
  LibraryStats,
} from "$lib/subsonic/types";
import type { PlaybackEngine } from "$lib/music/playback-engine";
import {
  bindMediaSession,
  clearMediaSession,
  updateMediaSessionPosition,
} from "$lib/music/media-session";
import { requestNativeMediaSync } from "$lib/media/native-media-sync";
import {
  defaultEqSettings,
  loadEqFromLocalStorage,
  type EqBandParam,
  type EqSettings,
} from "$lib/music/eq";
import { loadVolume, type NativeBackendPref } from "$lib/music/prefs";
import { nextSequentialIndex, shuffleIndices } from "$lib/music/playback-queue";
import { loadMixSettings, type MixSettings } from "$lib/music/mix-settings";
import {
  enforceQueueLimit,
  loadQueueSettings,
  type QueueSettings,
} from "$lib/music/queue-settings";
import {
  crossfadeActive,
  loadPlaybackSettings,
  type PlaybackSettings,
} from "$lib/music/playback-settings";
import {
  isVideoPlaybackPath,
  isNowPlayingPath,
} from "$lib/music/player-layout";
import { connection } from "$lib/music/connection.svelte";
import { setPauseMusicHandler } from "$lib/video/playback-gate.svelte";
import {
  nativeDesktopAvailable,
  setMobileMediaQueue,
  setMobileMediaState,
} from "$lib/config/runtime";
import {
  defaultCacheSettings,
  type CacheSettings,
} from "$lib/music/cache-settings";
import {
  type LyricsSettings,
  type LyricsSettingsResponse,
} from "$lib/music/lyrics-settings";
import {
  loadTranscodingSettings,
  type TranscodingSettings,
} from "$lib/music/transcoding-settings";
import {
  loadImmersiveAudioSettings,
  type ImmersiveAudioSettings,
} from "$lib/music/immersive-audio-settings";
import { sources } from "$lib/features/sources/store.svelte";
import {
  CONTINUOUS_MODE_LABELS,
  createLibraryPoolState,
  type ContinuousMode,
  type LibraryPoolState,
} from "$lib/music/continuous-pool";
import {
  createPersonalRadioState,
  notePersonalComplete,
  type PersonalRadioOptions,
  type PersonalRadioState,
} from "$lib/music/personal-radio";
import { prefetchNowPlayingCoverArt } from "$lib/music/cover-art-prefetch";
import {
  entryToSong as listenEntryToSong,
  favoriteToSong as favoriteEntryToSong,
} from "./music/helpers";
import { createFavoritesOps } from "./music/favorites-ops";
import { createPlaylistOps } from "./music/playlist-ops";
import { createLibraryBrowseOps } from "./music/library-browse-ops";
import { createPlaybackTransportOps } from "./music/playback-transport-ops";
import { createQueueOps } from "./music/queue-ops";
import { createCacheOps } from "./music/cache-ops";
import { createSettingsOps } from "./music/settings-ops";
import { createLyricsOps } from "./music/lyrics-ops";
import { createEqOps } from "./music/eq-ops";
import { createPlayerChromeOps } from "./music/player-chrome-ops";
import { createMixOps } from "./music/mix-ops";
import { createRadioOps } from "./music/radio-ops";
import { createPlayLaunchOps } from "./music/play-launch-ops";
import { createConnectOps } from "./music/connect-ops";
import { createEngineOps } from "./music/engine-ops";
import { createPlaybackCoreOps } from "./music/playback-core-ops";
import { createLibraryRefreshOps } from "./music/library-refresh-ops";
import { createTrackBoundaryOps } from "./music/track-boundary-ops";
import { MusicContextBinder } from "./music/contexts";
import type { PersonalMix, PlayerLayout } from "./music/types";

export type { PersonalMix, PlayerLayout } from "./music/types";

export { CONTINUOUS_MODE_LABELS };
export type { ContinuousMode };

class MusicStore {
  config = $state(createDefaultSubsonicConfig());
  client = $derived(new SubsonicClient(this.config));
  library = $derived.by(() => {
    if (sources.hasUnifiedMode) {
      return createUnifiedLibraryAdapter(
        createSubsonicLibraryAdapter(new SubsonicClient(this.config)),
        createLocalLibraryAdapter(),
      );
    }
    return sources.hasLocalActive
      ? createLocalLibraryAdapter()
      : createSubsonicLibraryAdapter(new SubsonicClient(this.config));
  });

  status = $state<MusicStatus>({ enabled: false, connected: false });
  serverName = $derived(this.status.serverName ?? "Subsonic");

  recentAlbums = $state.raw<SubsonicAlbum[]>([]);
  frequentAlbums = $state.raw<SubsonicAlbum[]>([]);
  recommendations = $state.raw<SubsonicAlbum[]>([]);
  personalMixes = $state.raw<PersonalMix[]>([]);
  randomTracks = $state.raw<SubsonicSong[]>([]);
  resumeTracks = $state.raw<ListenEntry[]>([]);
  listenHistory = $state.raw<ListenEntry[]>([]);
  stats = $state<ListenStats | null>(null);
  libraryStats = $state<LibraryStats | null>(null);
  libraryRefreshing = $state(false);
  libraryRevision = $state(0);
  playlists = $state.raw<MusicPlaylist[]>([]);
  serverPlaylists = $state.raw<ServerPlaylist[]>([]);
  internetRadios = $state.raw<InternetRadioStation[]>([]);
  favoriteTracks = $state.raw<FavoriteTrack[]>([]);
  favoriteIds = $state.raw<Set<string>>(new Set());
  favoriteAlbums = $state.raw<SubsonicAlbum[]>([]);
  favoriteAlbumIds = $state.raw<Set<string>>(new Set());
  favoriteArtists = $state.raw<SubsonicArtist[]>([]);
  favoriteArtistIds = $state.raw<Set<string>>(new Set());
  genres = $state.raw<SubsonicGenre[]>([]);
  allArtists = $state.raw<SubsonicArtist[]>([]);
  private artistsLoaded = false;
  private artistsLoadedAt = 0;
  loading = $state(false);
  error = $state<string | null>(null);
  connected = $state(false);
  libraryReady = $derived(
    sources.needsSetup ||
      this.connected ||
      ((sources.hasLocalActive || sources.hasUnifiedMode) &&
        this.status.enabled &&
        this.status.connected),
  );
  reconnecting = $derived(
    connection.phase === "reconnecting" || connection.phase === "retrying",
  );

  queue = $state.raw<QueueTrack[]>([]);
  queueIndex = $state(-1);
  playing = $state(false);
  volume = $state(loadVolume());
  shuffle = $state(false);
  autoplay = $state(true);
  continuousMode = $state<ContinuousMode>("off");
  repeat = $state<"off" | "all" | "one">("off");
  queueOpen = $state(false);
  downloadedIds = $state.raw<Set<string>>(new Set());
  cacheSettings = $state<CacheSettings>(defaultCacheSettings());
  cacheUsedBytes = $state(0);
  cacheTrackCount = $state(0);
  mixSettings = $state<MixSettings>(loadMixSettings());
  mixesRegenerating = $state(false);
  continuousBusy = $state<"off" | ContinuousMode | "internet">("off");
  currentTime = $state(0);
  smoothProgress = $state(0);
  duration = $state(0);

  playerLayout = $state<PlayerLayout>("full");
  routePath = $state("");
  eqOpen = $state(false);
  lyricsOpen = $state(false);
  currentLyrics = $state<ParsedLyrics | null>(null);
  lyricsLoading = $state(false);
  lyricsFetching = $state(false);
  lyricsRequestId = 0;
  pendingLyricsReload = false;
  lyricsSettings = $state<LyricsSettingsResponse | null>(null);
  transcodingSettings = $state<TranscodingSettings>(loadTranscodingSettings());
  immersiveAudioSettings = $state<ImmersiveAudioSettings>(
    loadImmersiveAudioSettings(),
  );
  queueSettings = $state<QueueSettings>(loadQueueSettings());
  playbackSettings = $state<PlaybackSettings>(loadPlaybackSettings());
  metadataEnhancementSettings = $state<MetadataEnhancementSettings>(
    loadMetadataEnhancementSettings(),
  );
  hideUnknownMetadata = $state(loadHideUnknownMetadata());
  eq = $state<EqSettings>(defaultEqSettings());
  nativePlayback = $state(false);
  nativeAvailable = $state(false);
  mpvAvailable = $state(false);
  vlcAvailable = $state(false);
  nativeBackend = $state("");
  nativeInitError = $state("");
  mpvLoadError = $state("");

  private applyNativeCaps(caps: {
    nativeAvailable: boolean;
    backend: string;
    mpvAvailable: boolean;
    vlcAvailable: boolean;
    mpvLoadError?: string;
    initError?: string;
  }) {
    this.engineOps.applyNativeCaps(caps);
  }

  eqAvailable = $derived(!this.nativePlayback);

  private authEnabled = false;
  private progressRaf = 0;
  private playbackEpoch = 0;
  private playCurrentTail: Promise<void> = Promise.resolve();
  private trackBoundaryBusy = false;
  private failedTrackSkips = 0;
  private continuousRefillInFlight: Promise<boolean> | null = null;
  private libraryPool: LibraryPoolState = createLibraryPoolState();
  private personalRadio: PersonalRadioState = createPersonalRadioState();
  private shuffleUpcoming: number[] = [];
  private shuffleHistory: number[] = [];
  private static readonly HOME_STALE_MS = 2 * 60 * 1000;
  private readonly contexts = new MusicContextBinder(
    this,
    MusicStore.HOME_STALE_MS,
  );
  private readonly queueOps = createQueueOps(this.contexts.queueCtx());
  private readonly favoritesOps = createFavoritesOps(
    this.contexts.favoritesCtx(),
  );
  private readonly playlistOps = createPlaylistOps(this.contexts.playlistCtx());
  private readonly libraryBrowseOps = createLibraryBrowseOps(
    this.contexts.libraryBrowseCtx(),
  );
  private readonly playbackTransportOps = createPlaybackTransportOps(
    this.contexts.playbackTransportCtx(),
  );
  private readonly cacheOps = createCacheOps(this.contexts.cacheCtx());
  private readonly settingsOps = createSettingsOps(this.contexts.settingsCtx());
  private readonly lyricsOps = createLyricsOps(this.contexts.lyricsCtx());
  private readonly eqOps = createEqOps(this.contexts.eqCtx());
  private readonly playerChromeOps = createPlayerChromeOps(
    this.contexts.playerChromeCtx(),
  );
  private readonly mixOps = createMixOps(this.contexts.mixCtx());
  private readonly radioOps = createRadioOps(this.contexts.radioCtx());
  private readonly playLaunchOps = createPlayLaunchOps(
    this.contexts.playLaunchCtx(),
  );
  private readonly connectOps = createConnectOps(this.contexts.connectCtx());
  private readonly engineOps = createEngineOps(this.contexts.engineCtx());
  private readonly playbackCoreOps = createPlaybackCoreOps(
    this.contexts.playbackCoreCtx(),
  );
  private readonly libraryRefreshOps = createLibraryRefreshOps(
    this.contexts.libraryRefreshCtx(),
  );
  private readonly trackBoundaryOps = createTrackBoundaryOps(
    this.contexts.trackBoundaryCtx(),
  );

  currentTrack = $derived(
    this.queueIndex >= 0
      ? this.queue[Math.min(this.queueIndex, this.queue.length - 1)]
      : null,
  );

  isFavorite = (trackId: string) => this.favoriteIds.has(trackId);
  isFavoriteAlbum = (albumId: string) => this.favoriteAlbumIds.has(albumId);
  isFavoriteArtist = (artistId: string) => this.favoriteArtistIds.has(artistId);

  onPlayRoute = $derived(isVideoPlaybackPath(this.routePath));
  onNowPlayingRoute = $derived(isNowPlayingPath(this.routePath));

  playerVisible = $derived(
    this.currentTrack !== null &&
      this.playerLayout !== "dismissed" &&
      !this.onPlayRoute,
  );

  private engine: PlaybackEngine | null = null;
  private engineInitPromise: Promise<void> | null = null;
  private transcodedTrackIds = $state.raw<Set<string>>(
    createBoundedSet<string>(64),
  );
  private progressTimer: ReturnType<typeof setInterval> | undefined;
  private lastSavedPosition = 0;
  private cleanupListeners: (() => void)[] = [];
  private resumeOnPlay = false;
  private pendingStartAt: number | null = null;
  private pendingStartPaused = false;
  private cacheInFlight = new Set<string>();
  private cacheStrategyRunning = false;
  private homeCoreFetchedAt = 0;
  private personalizationFetchedAt = 0;
  private mixesRefreshInFlight: Promise<void> | null = null;
  private static readonly PERSONALIZATION_STALE_MS = 10 * 60 * 1000;
  private libraryWatchTimer: ReturnType<typeof setTimeout> | undefined;
  private lastLibrarySongCount = 0;
  private lastLibraryScanning = false;

  constructor() {
    if (typeof window !== "undefined") {
      this.eq = loadEqFromLocalStorage();
      this.volume = loadVolume();
      this.transcodingSettings = loadTranscodingSettings();
    }
  }

  updateTranscodingSettings(settings: TranscodingSettings) {
    this.settingsOps.updateTranscodingSettings(settings);
  }

  updateImmersiveAudioSettings(settings: ImmersiveAudioSettings) {
    this.settingsOps.updateImmersiveAudioSettings(settings);
  }

  updateQueueSettings(settings: QueueSettings) {
    this.settingsOps.updateQueueSettings(settings);
  }

  updatePlaybackSettings(settings: PlaybackSettings) {
    this.settingsOps.updatePlaybackSettings(settings);
  }

  updateMetadataEnhancementSettings(settings: MetadataEnhancementSettings) {
    this.settingsOps.updateMetadataEnhancementSettings(settings);
  }

  setHideUnknownMetadata(hide: boolean) {
    this.settingsOps.setHideUnknownMetadata(hide);
  }

  private applyQueueLimit() {
    const max = this.queueSettings.maxQueueSize;
    if (max <= 0 || this.queue.length === 0) return;
    const { queue, queueIndex } = enforceQueueLimit(
      this.queue,
      this.queueIndex,
      max,
    );
    this.queue = queue;
    this.queueIndex = queueIndex;
  }

  toggleLyricsPanel() {
    this.lyricsOps.toggleLyricsPanel();
  }

  async loadCurrentLyrics() {
    await this.lyricsOps.loadCurrentLyrics();
  }

  async fetchCurrentLyrics(): Promise<ParsedLyrics | null> {
    return this.lyricsOps.fetchCurrentLyrics();
  }

  async generateWhisperLyrics(): Promise<ParsedLyrics | null> {
    return this.lyricsOps.generateWhisperLyrics();
  }

  async loadLyricsSettings() {
    await this.lyricsOps.loadLyricsSettings();
  }

  async updateLyricsSettings(settings: LyricsSettings) {
    await this.lyricsOps.updateLyricsSettings(settings);
  }

  seekToLyricLine(startMs: number) {
    this.lyricsOps.seekToLyricLine(startMs);
  }

  async searchLyrics(query: string): Promise<LyricsSearchHit[]> {
    return this.lyricsOps.searchLyrics(query);
  }

  async loadEqSettings(authEnabled: boolean) {
    await this.eqOps.loadEqSettings(authEnabled);
  }

  private async persistEq() {
    await this.eqOps.persistEq();
  }

  toggleEqPanel() {
    this.eqOps.toggleEqPanel();
  }

  async setEqEnabled(enabled: boolean) {
    await this.eqOps.setEqEnabled(enabled);
  }

  initEngine(): Promise<void> {
    return this.engineOps.initEngine();
  }

  async refreshNativeCapabilities(): Promise<void> {
    await this.syncNativeCaps();
  }

  private async setupEngine() {
    await this.buildEngine();
  }

  private async buildEngine(force?: "web" | "native") {
    return this.engineOps.buildEngine(force);
  }

  private async syncNativeCaps() {
    return this.engineOps.syncNativeCaps();
  }

  private attachEngineListeners() {
    this.engineOps.attachEngineListeners();
  }

  private teardownEngine() {
    this.engineOps.teardownEngine();
  }

  private recordPlaybackError(message: string) {
    this.engineOps.recordPlaybackError(message);
  }

  private async fallbackToWebPlayback(reason: string): Promise<boolean> {
    return this.engineOps.fallbackToWebPlayback(reason);
  }

  private async resumeCurrentAt(positionSec: number, wasPlaying: boolean) {
    return this.engineOps.resumeCurrentAt(positionSec, wasPlaying);
  }

  private async applyEngineChange() {
    return this.engineOps.applyEngineChange();
  }

  private connectInFlight: Promise<boolean> | null = null;
  private libraryWarmup = false;
  private playbackRestored = false;
  private reconnectResumePending = false;
  private reconnectPositionMs = 0;
  private nextTrackPrepared = false;
  private crossfadeHandled = false;
  private crossfadeFinishingTrack: QueueTrack | null = null;
  private nativeFallbackAttempted = false;

  setRoutePath(pathname: string) {
    this.playerChromeOps.setRoutePath(pathname);
  }

  dismissPlayer() {
    this.playerChromeOps.dismissPlayer();
  }

  restorePlayer() {
    this.playerChromeOps.restorePlayer();
  }

  expandPlayer() {
    this.playerChromeOps.expandPlayer();
  }

  async connect(
    options: { quiet?: boolean; authEnabled?: boolean; force?: boolean } = {},
  ): Promise<boolean> {
    return this.connectOps.connect(options);
  }

  private async doConnect(
    options: { quiet?: boolean; authEnabled?: boolean } = {},
  ): Promise<boolean> {
    return this.connectOps.doConnect(options);
  }

  private async finishConnectSetup(options: {
    quiet?: boolean;
    authEnabled?: boolean;
  }) {
    return this.connectOps.finishConnectSetup(options);
  }

  private scheduleBackgroundHydration() {
    this.connectOps.scheduleBackgroundHydration();
  }

  async pingServer(): Promise<boolean> {
    return this.connectOps.pingServer();
  }

  private handleServerReachabilityLoss() {
    this.connectOps.handleServerReachabilityLoss();
  }

  private async ensureOfflinePlaybackSource(token: number) {
    return this.connectOps.ensureOfflinePlaybackSource(token);
  }

  private canPlayCurrentOffline(): boolean {
    return this.connectOps.canPlayCurrentOffline();
  }

  private async bootstrapOfflinePlayback() {
    await Promise.all([this.loadDownloads(), this.initEngine()]).catch(
      () => {},
    );
  }

  private async resolveOfflineTrack(
    trackId: string,
  ): Promise<SubsonicSong | null> {
    if (!trackId) return null;
    try {
      const song = await this.library.getSong(trackId);
      if (song) return song;
    } catch {
      /* library lookup may fail while server is down */
    }
    const items = await musicApi.listDownloads().catch(() => []);
    const entry = items.find((item) => item.trackId === trackId);
    if (!entry) return null;
    return {
      id: trackId,
      title: entry.trackTitle || "Unknown track",
      artist: entry.artistName || "",
    };
  }

  private async handlePlaybackNetworkFailure(
    token: number,
    options: { resume?: boolean } = {},
  ) {
    return this.trackBoundaryOps.handlePlaybackNetworkFailure(token, options);
  }

  suspendForReconnect() {
    this.connectOps.suspendForReconnect();
  }

  private markPendingReconnectResume() {
    this.connectOps.markPendingReconnectResume();
  }

  async resumeAfterReconnect() {
    return this.connectOps.resumeAfterReconnect();
  }

  async selfHeal() {
    return this.connectOps.selfHeal();
  }

  disconnect() {
    this.connectOps.disconnect();
    this.allArtists = [];
    this.genres = [];
  }

  private personalRadioOptions(coldStart = false): PersonalRadioOptions {
    const settings = this.mixSettings;
    return {
      recencyCooldownMs: settings.personalRadioRecencyHours * 60 * 60 * 1000,
      exploreBonus: settings.radioExploreBonus,
      albumLookback: settings.flowAlbumLookback,
      coldStart,
    };
  }

  async refreshHome(forcePersonalization = false) {
    await this.refreshHomeCore();
    this.schedulePersonalizationRefresh(forcePersonalization);
  }

  async refreshHomeCore() {
    return this.libraryBrowseOps.refreshHomeCore();
  }

  private schedulePersonalizationRefresh(force = false) {
    if (
      !force &&
      Date.now() - this.personalizationFetchedAt <
        MusicStore.PERSONALIZATION_STALE_MS
    ) {
      return;
    }
    const run = () => {
      void this.refreshPersonalization().catch(() => {});
    };
    if (force) {
      queueMicrotask(run);
      return;
    }
    if (typeof requestIdleCallback !== "undefined") {
      requestIdleCallback(run, { timeout: 1500 });
    } else {
      queueMicrotask(run);
    }
  }

  private restoreCachedMixes() {
    this.mixOps.restoreCachedMixes();
  }

  private async refreshPersonalization() {
    await this.refreshMixes();
    void this.refreshRecommendations().catch(() => {});
    this.personalizationFetchedAt = Date.now();
  }

  async refreshMixes() {
    return this.mixOps.refreshMixes();
  }

  private async doRefreshMixes(force = false) {
    return this.mixOps.doRefreshMixes(force);
  }

  updateMixSettings(settings: MixSettings) {
    this.mixOps.updateMixSettings(settings);
  }

  async regenerateMixes() {
    return this.mixOps.regenerateMixes();
  }

  async regenerateMix(mixId: string) {
    return this.mixOps.regenerateMix(mixId);
  }

  async refreshRecommendations() {
    return this.mixOps.refreshRecommendations();
  }

  async refreshHistory(limit = 50) {
    return this.libraryBrowseOps.refreshHistory(limit);
  }

  async clearListenHistory() {
    return this.libraryBrowseOps.clearListenHistory();
  }

  async refreshStats(limit = 8) {
    return this.libraryBrowseOps.refreshStats(limit);
  }

  async refreshLibraryStats(options?: { bypassCache?: boolean }) {
    return this.libraryBrowseOps.refreshLibraryStats(options);
  }

  async refreshLibrary(options: { quiet?: boolean } = {}) {
    return this.libraryRefreshOps.refreshLibrary(options);
  }

  private startLibraryWatch() {
    this.libraryRefreshOps.startLibraryWatch();
  }

  private stopLibraryWatch() {
    this.libraryRefreshOps.stopLibraryWatch();
  }

  private scheduleLibraryWatch(delayMs: number) {
    this.libraryRefreshOps.scheduleLibraryWatch(delayMs);
  }

  private async pollLibraryChanges() {
    return this.libraryRefreshOps.pollLibraryChanges();
  }

  async playArtistAlbums(albums: SubsonicAlbum[], shuffle = false) {
    return this.libraryBrowseOps.playArtistAlbums(albums, shuffle);
  }

  async refreshPlaylists() {
    await this.playlistOps.refreshPlaylists();
  }

  async refreshServerPlaylists() {
    await this.playlistOps.refreshServerPlaylists();
  }

  async refreshInternetRadios() {
    await this.playlistOps.refreshInternetRadios();
  }

  async refreshFavorites() {
    await this.favoritesOps.refreshFavorites();
  }

  async toggleFavoriteAlbum(album: SubsonicAlbum) {
    await this.favoritesOps.toggleFavoriteAlbum(album);
  }

  async toggleFavoriteArtist(artist: SubsonicArtist) {
    await this.favoritesOps.toggleFavoriteArtist(artist);
  }

  async playFavoriteAlbums(albums: SubsonicAlbum[]) {
    await this.favoritesOps.playFavoriteAlbums(albums);
  }

  async playFavoriteArtists(artists: SubsonicArtist[]) {
    await this.favoritesOps.playFavoriteArtists(artists);
  }

  async toggleFavorite(track: SubsonicSong) {
    await this.favoritesOps.toggleFavorite(track);
  }

  async fetchServerPlaylist(id: string) {
    return this.playlistOps.fetchServerPlaylist(id);
  }

  async createServerPlaylist(name: string, songIds: string[] = []) {
    return this.playlistOps.createServerPlaylist(name, songIds);
  }

  async createSmartPlaylist(
    draft: import("$lib/music/smart-playlist/types").SmartPlaylistDraft,
    target: "local" | "server" = "server",
  ) {
    return this.playlistOps.createSmartPlaylist(draft, target);
  }

  async refreshSmartPlaylist(playlistId: string) {
    return this.playlistOps.refreshSmartPlaylist(playlistId);
  }

  async deleteServerPlaylist(id: string) {
    await this.playlistOps.deleteServerPlaylist(id);
  }

  async renameServerPlaylist(id: string, name: string) {
    await this.playlistOps.renameServerPlaylist(id, name);
  }

  async addToServerPlaylist(playlistId: string, tracks: SubsonicSong[]) {
    await this.playlistOps.addToServerPlaylist(playlistId, tracks);
  }

  async addTracksToServerPlaylist(playlistId: string, tracks: SubsonicSong[]) {
    await this.playlistOps.addTracksToServerPlaylist(playlistId, tracks);
  }

  async removeFromServerPlaylist(playlistId: string, songIndex: number) {
    await this.playlistOps.removeFromServerPlaylist(playlistId, songIndex);
  }

  async moveServerPlaylistSong(
    playlistId: string,
    currentSongIds: readonly string[],
    fromIndex: number,
    toIndex: number,
  ) {
    await this.playlistOps.moveServerPlaylistSong(
      playlistId,
      currentSongIds,
      fromIndex,
      toIndex,
    );
  }

  playFavorites(startIndex = 0) {
    this.favoritesOps.playFavorites(startIndex);
  }

  playAllFavorites(shuffle = false) {
    this.favoritesOps.playAllFavorites(shuffle);
  }

  async search(term: string) {
    return this.libraryBrowseOps.search(term);
  }

  async searchAll(term: string): Promise<{
    query: string;
    result: SubsonicSearchResult;
    suggestion: string | null;
    similar: SubsonicSong[];
  }> {
    return this.libraryBrowseOps.searchAll(term);
  }

  searchSimilarTracks(trackId: string, count = 12): Promise<SubsonicSong[]> {
    return this.libraryBrowseOps.searchSimilarTracks(trackId, count);
  }

  async loadGenres() {
    return this.libraryBrowseOps.loadGenres();
  }

  async loadArtists(options?: { force?: boolean }) {
    return this.libraryBrowseOps.loadArtists(options);
  }

  async getGenreSongs(
    genre: string,
    count = 200,
    offset = 0,
  ): Promise<SubsonicSong[]> {
    return this.libraryBrowseOps.getGenreSongs(genre, count, offset);
  }

  async playGenre(genre: string, shuffle = true) {
    return this.libraryBrowseOps.playGenre(genre, shuffle);
  }

  async addGenreToQueue(genre: string) {
    return this.libraryBrowseOps.addGenreToQueue(genre);
  }

  async playGenreNext(genre: string) {
    return this.libraryBrowseOps.playGenreNext(genre);
  }

  playTracks(
    tracks: SubsonicSong[],
    startIndex = 0,
    resume = false,
    continuousMode: ContinuousMode | boolean = "off",
    options: {
      feedback?: "play" | "shuffle" | "queue" | false;
      preservePersonalRadio?: boolean;
      preserveLibraryPool?: boolean;
    } = {},
  ) {
    this.playLaunchOps.playTracks(
      tracks,
      startIndex,
      resume,
      continuousMode,
      options,
    );
  }

  getMix(id: string): PersonalMix | undefined {
    return this.mixOps.getMix(id);
  }

  async ensureMix(id: string): Promise<PersonalMix | undefined> {
    return this.mixOps.ensureMix(id);
  }

  private replaceMixWithHydrated(hydrated: PersonalMix): PersonalMix[] {
    return this.mixOps.replaceMixWithHydrated(hydrated);
  }

  private async hydrateMixTracks(mix: PersonalMix): Promise<PersonalMix> {
    return this.mixOps.hydrateMixTracks(mix);
  }

  async playMix(mix: PersonalMix) {
    return this.mixOps.playMix(mix);
  }

  slimStoredMix(mixId: string) {
    this.mixOps.slimStoredMix(mixId);
  }

  removeFromQueue(index: number) {
    this.queueOps.removeFromQueue(index);
  }

  /** playNext inserts a track right after the current one. */
  playNext(track: SubsonicSong) {
    this.queueOps.playNext(track);
  }

  /** addToQueue appends a track to the end of the queue. */
  addToQueue(track: SubsonicSong) {
    this.queueOps.addToQueue(track);
  }

  /** addTracksToQueue appends multiple tracks to the end of the queue. */
  addTracksToQueue(tracks: SubsonicSong[]) {
    this.queueOps.addTracksToQueue(tracks);
  }

  /** playTracksNext inserts multiple tracks right after the current one. */
  playTracksNext(tracks: SubsonicSong[]) {
    this.queueOps.playTracksNext(tracks);
  }

  moveInQueue(from: number, to: number) {
    this.queueOps.moveInQueue(from, to);
  }

  /** clearQueue stops playback and empties the queue. */
  clearQueue() {
    this.queueOps.clearQueue();
  }

  playQueueIndex(index: number) {
    this.queueOps.playQueueIndex(index);
  }

  armStartPosition(seconds: number, paused = false) {
    this.queueOps.armStartPosition(seconds, paused);
  }

  async applyRemoteQueue(trackIds: string[], startIndex = 0) {
    const tracks = await this.resolveTracksForRestore(trackIds);
    if (tracks.length === 0) return;
    this.playTracks(tracks, startIndex);
  }

  toggleQueue() {
    this.queueOps.toggleQueue();
  }

  playAlbum(songs: SubsonicSong[], startIndex = 0) {
    this.playTracks(songs, startIndex);
  }

  async playTrackById(trackId: string) {
    return this.playLaunchOps.playTrackById(trackId);
  }

  async playOpenUri(uri: string) {
    return this.playLaunchOps.playOpenUri(uri);
  }

  async playRandomRadio(count = 25) {
    return this.radioOps.playRandomRadio(count);
  }

  private async startRandomRadio(count = 25) {
    return this.radioOps.startRandomRadio(count);
  }

  /**
   * Start continuous random playback from a seed set of tracks. The provided
   * tracks become the initial queue and the queue is refilled with more random
   * songs from the library as playback approaches the end.
   */
  playRandomTracks(tracks: SubsonicSong[]) {
    if (tracks.length === 0) return;
    this.shuffle = true;
    this.autoplay = true;
    this.playTracks(tracks, 0, false, "random");
  }

  async playLibraryShuffle() {
    return this.radioOps.playLibraryShuffle();
  }

  async playPersonalRadio(count = 25) {
    return this.radioOps.playPersonalRadio(count);
  }

  clearContinuousMode() {
    this.continuousMode = "off";
    this.persistPlaybackState();
  }

  async playRandomInternetRadio() {
    return this.radioOps.playRandomInternetRadio();
  }

  playInternetRadio(station: InternetRadioStation) {
    this.shuffle = false;
    this.autoplay = false;
    this.repeat = "off";
    this.playTracks([trackFromRadioStation(station)], 0);
  }

  playInternetRadios(stations: InternetRadioStation[], startIndex = 0) {
    this.radioOps.playInternetRadios(stations, startIndex);
  }

  pause() {
    this.playbackTransportOps.pause();
  }

  async togglePlay() {
    return this.playbackTransportOps.togglePlay();
  }

  requestPlayCurrent() {
    this.playbackCoreOps.requestPlayCurrent();
  }

  private markTrackTranscoded(trackId: string) {
    this.playbackCoreOps.markTrackTranscoded(trackId);
  }

  async playCurrent(epoch?: number, resume = false) {
    return this.playbackCoreOps.playCurrent(epoch, resume);
  }

  private isSupersededPlaybackError(message: string): boolean {
    return this.playbackCoreOps.isSupersededPlaybackError(message);
  }

  private onPlaybackStarted(
    track: QueueTrack,
    token: number,
    options: { incrementPlay?: boolean } = {},
  ) {
    this.playbackCoreOps.onPlaybackStarted(track, token, options);
  }

  private async playCurrentWork(
    token: number,
    resume = false,
    startAt: number | null = null,
    startPaused = false,
  ) {
    return this.playbackCoreOps.playCurrentWork(
      token,
      resume,
      startAt,
      startPaused,
    );
  }

  private async loadTrackSource(
    token: number,
    track: QueueTrack,
    url: string,
  ): Promise<void> {
    return this.trackBoundaryOps.loadTrackSource(token, track, url);
  }

  private async enrichCurrentTrack(trackId: string, epoch?: number) {
    return this.trackBoundaryOps.enrichCurrentTrack(trackId, epoch);
  }

  private async recordNowPlaying(track: SubsonicSong) {
    return this.trackBoundaryOps.recordNowPlaying(track);
  }

  trackStreamUrl(track: QueueTrack): string {
    return this.playbackCoreOps.trackStreamUrl(track);
  }

  isDownloaded(trackId: string): boolean {
    return this.cacheOps.isDownloaded(trackId);
  }

  async loadDownloads() {
    await this.cacheOps.loadDownloads();
  }

  async loadCacheSettings() {
    await this.cacheOps.loadCacheSettings();
  }

  async updateCacheSettings(settings: CacheSettings) {
    await this.cacheOps.updateCacheSettings(settings);
  }

  async clearDownloadCache() {
    await this.cacheOps.clearDownloadCache();
  }

  private maybeCacheTrack(track: SubsonicSong) {
    this.cacheOps.maybeCacheTrack(track);
  }

  private async runCacheStrategy() {
    await this.cacheOps.runCacheStrategy();
  }

  private async collectStrategyTracks(): Promise<SubsonicSong[]> {
    return this.cacheOps.collectStrategyTracks();
  }

  private prefetchCache(track: SubsonicSong) {
    this.cacheOps.prefetchCache(track);
  }

  async downloadCurrentOr(track: SubsonicSong) {
    await this.cacheOps.downloadCurrentOr(track);
  }

  async removeDownload(trackId: string) {
    await this.cacheOps.removeDownload(trackId);
  }

  async toggleDownload(track: SubsonicSong) {
    await this.cacheOps.toggleDownload(track);
  }

  offlineDownloadProgress = $state<{
    total: number;
    completed: number;
    failed: number;
    active: boolean;
  } | null>(null);
  private offlineDownloadAbort: AbortController | null = null;

  cancelOfflineDownload() {
    this.cacheOps.cancelOfflineDownload();
  }

  async downloadTracks(
    tracks: SubsonicSong[],
    options: { concurrency?: number } = {},
  ): Promise<{ downloaded: number; skipped: number; failed: number }> {
    return this.cacheOps.downloadTracks(tracks, options);
  }

  private sequentialNextIndex(): number {
    if (this.queue.length === 0) return -1;
    const last = this.queue.length - 1;
    if (this.queueIndex >= last) {
      return this.repeat === "all" ? 0 : -1;
    }
    return this.queueIndex + 1;
  }

  private prefetchAround() {
    if (!this.engine || this.queue.length <= 1 || this.queueIndex < 0) return;
    const len = this.queue.length;

    const prevIdx = (this.queueIndex - 1 + len) % len;
    const prev = this.queue[prevIdx];
    const nextIdx = this.sequentialNextIndex();
    const next = nextIdx >= 0 ? this.queue[nextIdx] : undefined;
    if (next) {
      prefetchNowPlayingCoverArt(this.config, next);
    }
    if (this.cacheSettings.enabled || this.shuffle || this.repeat === "one") {
      this.engine.prefetch([]);
    } else {
      this.engine.prefetch(prev ? [this.trackStreamUrl(prev)] : []);
    }
  }

  private maybePrepareNext() {
    if (
      this.nextTrackPrepared ||
      this.crossfadeHandled ||
      !this.engine ||
      !this.playing ||
      this.shuffle ||
      this.repeat === "one"
    ) {
      return;
    }
    if (this.queue.length <= 1 || this.queueIndex < 0) return;
    if (this.currentTrack && isInternetRadioTrack(this.currentTrack)) return;

    const duration = this.engine.duration || (this.currentTrack?.duration ?? 0);
    if (duration <= 0) return;

    const fade = crossfadeActive(this.playbackSettings, this.nativePlayback)
      ? this.playbackSettings.crossfadeDurationSec
      : 0;
    const threshold =
      fade > 0 ? Math.max(0, 1 - Math.min(duration, fade + 3) / duration) : 0.7;
    if (this.engine.currentTime / duration < threshold) return;

    const nextIdx = this.sequentialNextIndex();
    const nextTrack = nextIdx >= 0 ? this.queue[nextIdx] : undefined;
    this.engine.prepareNext(nextTrack ? this.trackStreamUrl(nextTrack) : "");
    this.nextTrackPrepared = true;
  }

  private maybeStartCrossfade() {
    if (
      this.crossfadeHandled ||
      !this.engine ||
      !this.playing ||
      !crossfadeActive(this.playbackSettings, this.nativePlayback)
    ) {
      return;
    }
    if (this.shuffle || this.repeat === "one") return;
    if (this.queue.length <= 1 || this.queueIndex < 0) return;
    if (this.currentTrack && isInternetRadioTrack(this.currentTrack)) return;

    const duration = this.engine.duration || (this.currentTrack?.duration ?? 0);
    if (duration <= 0) return;

    const fade = this.playbackSettings.crossfadeDurationSec;
    const remaining = duration - this.engine.currentTime;
    if (remaining > fade || remaining <= 0) return;

    const nextIdx = this.sequentialNextIndex();
    if (nextIdx < 0) return;
    const nextTrack = this.queue[nextIdx];
    if (!nextTrack) return;
    const nextUrl = this.trackStreamUrl(nextTrack);
    if (!this.engine.hasPrepared(nextUrl)) return;

    this.crossfadeHandled = true;
    this.crossfadeFinishingTrack = this.currentTrack;
    const token = this.playbackEpoch;

    void this.engine.activatePrepared(nextUrl, fade).then((started) => {
      if (!started || token !== this.playbackEpoch) {
        this.crossfadeHandled = false;
        this.crossfadeFinishingTrack = null;
        return;
      }
      this.queueIndex = nextIdx;
      const finished = this.crossfadeFinishingTrack;
      this.crossfadeFinishingTrack = null;
      this.nextTrackPrepared = false;
      this.engine?.prepareNext("");
      this.prefetchAround();
      if (finished) void this.recordPlayCompletion(finished);
      const track = this.currentTrack;
      if (track) {
        this.onPlaybackStarted(track, token, { incrementPlay: true });
      }
    });
  }

  cycleRepeat() {
    this.playbackTransportOps.cycleRepeat();
  }

  next() {
    this.playbackTransportOps.next();
  }

  previous() {
    this.playbackTransportOps.previous();
  }

  setVolume(value: number) {
    this.playbackTransportOps.setVolume(value);
  }

  adjustVolume(delta: number) {
    this.playbackTransportOps.adjustVolume(delta);
  }

  seekBy(seconds: number) {
    this.playbackTransportOps.seekBy(seconds);
  }

  seek(seconds: number) {
    this.playbackTransportOps.seek(seconds);
  }

  setEqPreset(presetId: string) {
    this.eqOps.setEqPreset(presetId);
  }

  setEqBandParam(index: number, param: Partial<EqBandParam>) {
    this.eqOps.setEqBandParam(index, param);
  }

  toggleEq(enabled?: boolean) {
    this.eqOps.toggleEq(enabled);
  }

  async setNativePlaybackEnabled(enabled: boolean) {
    return this.engineOps.setNativePlaybackEnabled(enabled);
  }

  async setNativeBackend(backend: NativeBackendPref) {
    return this.engineOps.setNativeBackend(backend);
  }

  private startSmoothProgress() {
    cancelAnimationFrame(this.progressRaf);
    const tick = () => {
      if (this.engine && this.playing) {
        this.currentTime = this.engine.currentTime;
        this.duration =
          this.engine.duration || (this.currentTrack?.duration ?? 0);
        const target =
          this.duration > 0 ? (this.currentTime / this.duration) * 100 : 0;
        const delta = target - this.smoothProgress;
        if (Math.abs(delta) > 2.5) {
          this.smoothProgress = target;
        } else {
          this.smoothProgress += delta * 0.35;
        }
        this.progressRaf = requestAnimationFrame(tick);
      }
    };
    this.progressRaf = requestAnimationFrame(tick);
  }

  private stopSmoothProgress() {
    cancelAnimationFrame(this.progressRaf);
    this.progressRaf = 0;
  }

  private startProgressTracking() {
    clearInterval(this.progressTimer);
    this.progressTimer = setInterval(() => {
      if (!this.engine || !this.playing) return;
      const pos = Math.floor(this.engine.currentTime * 1000);
      if (Math.abs(pos - this.lastSavedPosition) > 8000) {
        const deltaMs = Math.max(
          0,
          Math.min(15000, pos - this.lastSavedPosition),
        );
        this.lastSavedPosition = pos;
        void this.saveProgress(pos, { deltaMs });
      }
    }, 4000);
  }

  private syncMediaSession() {
    const track = this.currentTrack;
    if (!track) {
      clearMediaSession();
      setMobileMediaState(false);
      setMobileMediaQueue([], -1);
      if (nativeDesktopAvailable()) requestNativeMediaSync();
      return;
    }
    bindMediaSession(track, this.config, this.playing, {
      onPlay: async () => {
        await this.togglePlay();
      },
      onPause: () => {
        this.pause();
      },
      onPrevious: () => {
        this.previous();
      },
      onNext: () => {
        this.next();
      },
      onSeek: (seconds) => {
        this.seek(seconds);
      },
    });
    const duration = this.duration || track.duration || 0;
    updateMediaSessionPosition(duration, this.currentTime);
    const coverPath =
      coverArtUrl(
        this.config,
        track.coverArt ?? track.albumId ?? track.id,
        512,
      ) ?? "";
    setMobileMediaState(this.playing, {
      title: track.title,
      artist: track.artist ?? "Unknown artist",
      album: track.album ?? "",
      artwork: absoluteMobileArtworkUrl(coverPath),
      duration,
      position: this.currentTime,
    });
    setMobileMediaQueue(
      this.queue.map((q) => ({
        id: q.id,
        title: q.title,
        artist: q.artist ?? "",
        album: q.album ?? "",
        artwork: absoluteMobileArtworkUrl(
          coverArtUrl(this.config, q.coverArt ?? q.albumId ?? q.id, 256) ?? "",
        ),
      })),
      this.queueIndex,
    );
    if (nativeDesktopAvailable()) requestNativeMediaSync();
  }

  private async onTrackEnded() {
    return this.trackBoundaryOps.onTrackEnded();
  }

  private async maybeRefillContinuousQueue(): Promise<boolean> {
    return this.trackBoundaryOps.maybeRefillContinuousQueue();
  }

  private async refillLibraryQueue(count: number): Promise<boolean> {
    return this.trackBoundaryOps.refillLibraryQueue(count);
  }

  private async refillPersonalQueue(count: number): Promise<boolean> {
    return this.trackBoundaryOps.refillPersonalQueue(count);
  }

  private appendTracksToQueue(tracks: SubsonicSong[]): boolean {
    return this.trackBoundaryOps.appendTracksToQueue(tracks);
  }

  private async appendRandomSongsToQueue(count: number): Promise<boolean> {
    return this.trackBoundaryOps.appendRandomSongsToQueue(count);
  }

  private seedShuffleUpcoming() {
    if (!this.shuffle || this.queue.length <= 1) {
      this.shuffleUpcoming = [];
      this.shuffleHistory = [];
      return;
    }
    this.shuffleUpcoming = shuffleIndices(this.queue.length, this.queueIndex);
  }

  private ensureShuffleUpcoming() {
    if (!this.shuffle) {
      this.shuffleUpcoming = [];
      this.shuffleHistory = [];
      return;
    }
    if (this.queue.length <= 1) {
      this.shuffleUpcoming = [];
      this.shuffleHistory = [];
      return;
    }
    if (this.shuffleUpcoming.length === 0) {
      this.seedShuffleUpcoming();
    }
  }

  private advanceShuffleIndex(): boolean {
    if (this.queue.length <= 1) return false;
    const current = this.queueIndex;
    this.ensureShuffleUpcoming();
    if (this.shuffleUpcoming.length === 0) {
      if (this.repeat === "all") {
        this.seedShuffleUpcoming();
      } else {
        return false;
      }
    }
    const next = this.shuffleUpcoming.shift();
    if (next === undefined) return false;
    if (current >= 0) this.shuffleHistory.push(current);
    this.queueIndex = next;
    this.syncMediaSession();
    this.requestPlayCurrent();
    return true;
  }

  private advanceTrack(): boolean {
    if (this.queue.length === 0) return false;
    if (this.shuffle && this.queue.length > 1) {
      return this.advanceShuffleIndex();
    }
    const nextIdx = nextSequentialIndex(
      this.queueIndex,
      this.queue.length,
      this.repeat,
    );
    if (nextIdx === null) return false;
    this.queueIndex = nextIdx;
    this.syncMediaSession();
    this.requestPlayCurrent();
    return true;
  }

  private stopAtQueueEnd() {
    this.engine?.pause();
    this.playing = false;
    this.stopSmoothProgress();
    this.syncMediaSession();
    this.persistPlaybackState();
  }

  private skipFailedTrack(epoch: number, reason?: string) {
    this.trackBoundaryOps.skipFailedTrack(epoch, reason);
  }

  private async recordPlayCompletion(track: SubsonicSong) {
    if (isInternetRadioTrack(track)) return;
    if (this.continuousMode === "personal") {
      notePersonalComplete(this.personalRadio, track);
    }
    await this.saveProgress(
      (track.duration ?? 0) * 1000,
      {
        played: true,
        deltaMs: Math.max(
          0,
          Math.min(
            15000,
            (track.duration ?? 0) * 1000 - this.lastSavedPosition,
          ),
        ),
      },
      track,
    );
    await musicApi.markTrackPlayed(track.id).catch(() => {});
    await this.library.scrobble(track.id, true).catch(() => {});
    await musicApi.rockskyScrobble(toRockskyTrack(track)).catch(() => {});
    await musicApi.listenbrainzScrobble(toRockskyTrack(track)).catch(() => {});
    await musicApi.lastfmScrobble(toRockskyTrack(track)).catch(() => {});
  }

  async saveProgress(
    positionMs: number,
    options: {
      played?: boolean;
      incrementPlay?: boolean;
      deltaMs?: number;
    } = {},
    forTrack?: SubsonicSong,
  ) {
    return this.playbackCoreOps.saveProgress(positionMs, options, forTrack);
  }

  async addToPlaylist(playlistId: string, track: SubsonicSong) {
    await this.playlistOps.addToPlaylist(playlistId, track);
  }

  async addTracksToPlaylist(playlistId: string, tracks: SubsonicSong[]) {
    await this.playlistOps.addTracksToPlaylist(playlistId, tracks);
  }

  async createPlaylist(name: string) {
    return this.playlistOps.createPlaylist(name);
  }

  entryToSong(entry: ListenEntry): SubsonicSong {
    return listenEntryToSong(entry);
  }

  favoriteToSong(entry: FavoriteTrack): SubsonicSong {
    return favoriteEntryToSong(entry);
  }

  persistPlaybackSnapshot(): void {
    this.persistPlaybackState();
  }

  async flushPlaybackState(): Promise<void> {
    return this.playbackCoreOps.flushPlaybackState();
  }

  private persistPlaybackState() {
    this.playbackCoreOps.persistPlaybackState();
  }

  private async restorePlayback() {
    return this.trackBoundaryOps.restorePlayback();
  }

  private async resolveTracksForRestore(
    trackIds: string[],
  ): Promise<SubsonicSong[]> {
    const progress = await musicApi
      .getListenProgressBatch(trackIds)
      .catch(() => new Map<string, ListenEntry>());

    const byId = new Map<string, SubsonicSong>();
    const missing: string[] = [];
    const radioMissing: string[] = [];

    for (const id of trackIds) {
      if (radioStationIdFromTrackId(id)) {
        radioMissing.push(id);
        continue;
      }
      const fromHistory = this.listenHistory.find((e) => e.trackId === id);
      const entry = progress.get(id) ?? fromHistory;
      if (entry) {
        byId.set(id, this.entryToSong(entry));
      } else {
        missing.push(id);
      }
    }

    if (radioMissing.length > 0) {
      if (this.internetRadios.length === 0) {
        await this.refreshInternetRadios();
      }
      const stationsById = new Map(
        this.internetRadios.map((station) => [station.id, station]),
      );
      for (const id of radioMissing) {
        const stationId = radioStationIdFromTrackId(id);
        if (!stationId) continue;
        const station = stationsById.get(stationId);
        if (station) byId.set(id, trackFromRadioStation(station));
      }
    }

    if (missing.length > 0) {
      const fetched = await Promise.all(
        missing.map((id) => this.library.getSong(id).catch(() => null)),
      );
      missing.forEach((id, index) => {
        const song = fetched[index];
        if (song) byId.set(id, song);
      });
    }

    return trackIds
      .map((id) => byId.get(id))
      .filter((track): track is SubsonicSong => track !== undefined);
  }
}

export const music = new MusicStore();

setPauseMusicHandler(() => {
  if (music.playing) music.pause();
});
