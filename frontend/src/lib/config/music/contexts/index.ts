// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicStore } from "../../music.svelte";
import type { MusicStoreHost } from "../types";
import type { MusicQueueContext } from "../queue-ops";
import type { MusicFavoritesContext } from "../favorites-ops";
import type { MusicPlaylistContext } from "../playlist-ops";
import type { MusicLibraryBrowseContext } from "../library-browse-ops";
import type { MusicPlaybackTransportContext } from "../playback-transport-ops";
import type { MusicCacheContext } from "../cache-ops";
import type { MusicSettingsContext } from "../settings-ops";
import type { MusicLyricsContext } from "../lyrics-ops";
import type { MusicEqContext } from "../eq-ops";
import type { MusicPlayerChromeContext } from "../player-chrome-ops";
import type { MusicMixContext } from "../mix-ops";
import type { MusicRadioContext } from "../radio-ops";
import type { MusicPlayLaunchContext } from "../play-launch-ops";
import type { MusicConnectContext } from "../connect-ops";
import type { MusicEngineContext } from "../engine-ops";
import type { MusicPlaybackCoreContext } from "../playback-core-ops";
import type { MusicLibraryRefreshContext } from "../library-refresh-ops";
import type { MusicTrackBoundaryContext } from "../track-boundary-ops";
import { createQueueContext } from "./queue";
import { createFavoritesContext } from "./favorites";
import { createPlaylistContext } from "./playlist";
import { createLibraryBrowseContext } from "./library-browse";
import { createPlaybackTransportContext } from "./playback-transport";
import { createCacheContext } from "./cache";
import { createSettingsContext } from "./settings";
import { createLyricsContext } from "./lyrics";
import { createEqContext } from "./eq";
import { createPlayerChromeContext } from "./player-chrome";
import { createMixContext } from "./mix";
import { createRadioContext } from "./radio";
import { createPlayLaunchContext } from "./play-launch";
import { createConnectContext } from "./connect";
import { createEngineContext } from "./engine";
import { createPlaybackCoreContext } from "./playback-core";
import { createLibraryRefreshContext } from "./library-refresh";
import { createTrackBoundaryContext } from "./track-boundary";

export type { MusicStoreHost } from "../types";

export class MusicContextBinder {
  private readonly store: MusicStoreHost;
  private readonly homeStaleMs: number;

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

  constructor(store: MusicStore, homeStaleMs: number) {
    // MusicStore keeps most of the consumed members private, so the class is
    // not assignable to the MusicStoreHost view. The bridge happens once here.
    this.store = store as unknown as MusicStoreHost;
    this.homeStaleMs = homeStaleMs;
  }

  queueCtx(): MusicQueueContext {
    if (this.queueContext) return this.queueContext;
    this.queueContext = createQueueContext(this.store);
    return this.queueContext;
  }

  favoritesCtx(): MusicFavoritesContext {
    if (this.favoritesContext) return this.favoritesContext;
    this.favoritesContext = createFavoritesContext(this.store);
    return this.favoritesContext;
  }

  playlistCtx(): MusicPlaylistContext {
    if (this.playlistContext) return this.playlistContext;
    this.playlistContext = createPlaylistContext(this.store);
    return this.playlistContext;
  }

  libraryBrowseCtx(): MusicLibraryBrowseContext {
    if (this.libraryBrowseContext) return this.libraryBrowseContext;
    this.libraryBrowseContext = createLibraryBrowseContext(this.store);
    return this.libraryBrowseContext;
  }

  playbackTransportCtx(): MusicPlaybackTransportContext {
    if (this.playbackTransportContext) return this.playbackTransportContext;
    this.playbackTransportContext = createPlaybackTransportContext(this.store);
    return this.playbackTransportContext;
  }

  cacheCtx(): MusicCacheContext {
    if (this.cacheContext) return this.cacheContext;
    this.cacheContext = createCacheContext(this.store);
    return this.cacheContext;
  }

  settingsCtx(): MusicSettingsContext {
    if (this.settingsContext) return this.settingsContext;
    this.settingsContext = createSettingsContext(this.store);
    return this.settingsContext;
  }

  lyricsCtx(): MusicLyricsContext {
    if (this.lyricsContext) return this.lyricsContext;
    this.lyricsContext = createLyricsContext(this.store);
    return this.lyricsContext;
  }

  eqCtx(): MusicEqContext {
    if (this.eqContext) return this.eqContext;
    this.eqContext = createEqContext(this.store);
    return this.eqContext;
  }

  playerChromeCtx(): MusicPlayerChromeContext {
    if (this.playerChromeContext) return this.playerChromeContext;
    this.playerChromeContext = createPlayerChromeContext(this.store);
    return this.playerChromeContext;
  }

  mixCtx(): MusicMixContext {
    if (this.mixContext) return this.mixContext;
    this.mixContext = createMixContext(this.store);
    return this.mixContext;
  }

  radioCtx(): MusicRadioContext {
    if (this.radioContext) return this.radioContext;
    this.radioContext = createRadioContext(this.store);
    return this.radioContext;
  }

  playLaunchCtx(): MusicPlayLaunchContext {
    if (this.playLaunchContext) return this.playLaunchContext;
    this.playLaunchContext = createPlayLaunchContext(this.store);
    return this.playLaunchContext;
  }

  connectCtx(): MusicConnectContext {
    if (this.connectContext) return this.connectContext;
    this.connectContext = createConnectContext(this.store, this.homeStaleMs);
    return this.connectContext;
  }

  engineCtx(): MusicEngineContext {
    if (this.engineContext) return this.engineContext;
    this.engineContext = createEngineContext(this.store);
    return this.engineContext;
  }

  playbackCoreCtx(): MusicPlaybackCoreContext {
    if (this.playbackCoreContext) return this.playbackCoreContext;
    this.playbackCoreContext = createPlaybackCoreContext(this.store);
    return this.playbackCoreContext;
  }

  libraryRefreshCtx(): MusicLibraryRefreshContext {
    if (this.libraryRefreshContext) return this.libraryRefreshContext;
    this.libraryRefreshContext = createLibraryRefreshContext(this.store);
    return this.libraryRefreshContext;
  }

  trackBoundaryCtx(): MusicTrackBoundaryContext {
    if (this.trackBoundaryContext) return this.trackBoundaryContext;
    this.trackBoundaryContext = createTrackBoundaryContext(this.store);
    return this.trackBoundaryContext;
  }
}
