// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  isVideoPlaybackPath,
  playerLayoutForPath,
} from "$lib/music/player-layout";
import type { PlaybackEngine } from "$lib/music/playback-engine";
import type { QueueTrack } from "$lib/subsonic";
import type { PlayerLayout } from "./types";

export interface MusicPlayerChromeContext {
  routePath: string;
  playing: boolean;
  playerLayout: PlayerLayout;
  currentTrack: QueueTrack | null;
  queueOpen: boolean;
  engine: PlaybackEngine | null;
  initEngine(): Promise<void>;
  stopSmoothProgress(): void;
  syncMediaSession(): void;
}

export function setRoutePath(ctx: MusicPlayerChromeContext, pathname: string) {
  const enteringPlayback =
    isVideoPlaybackPath(pathname) && !isVideoPlaybackPath(ctx.routePath);

  if (enteringPlayback && ctx.playing) {
    void ctx.initEngine().then(() => {
      ctx.engine?.pause();
    });
    ctx.playing = false;
    ctx.stopSmoothProgress();
    ctx.syncMediaSession();
    ctx.queueOpen = false;
  }

  ctx.routePath = pathname;

  if (!ctx.currentTrack || ctx.playerLayout === "dismissed") return;
  ctx.playerLayout = playerLayoutForPath(pathname, ctx.playerLayout);
}

export function dismissPlayer(ctx: MusicPlayerChromeContext) {
  ctx.playerLayout = "dismissed";
}

export function restorePlayer(ctx: MusicPlayerChromeContext) {
  if (!ctx.currentTrack) return;
  ctx.playerLayout = "full";
}

export function expandPlayer(ctx: MusicPlayerChromeContext) {
  if (!ctx.currentTrack) return;
  ctx.playerLayout = "full";
}

export function createPlayerChromeOps(ctx: MusicPlayerChromeContext) {
  return {
    setRoutePath: (pathname: string) => setRoutePath(ctx, pathname),
    dismissPlayer: () => dismissPlayer(ctx),
    restorePlayer: () => restorePlayer(ctx),
    expandPlayer: () => expandPlayer(ctx),
  };
}
