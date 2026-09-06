// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { music } from "$lib/config/music.svelte";
import { nativeDesktopAvailable, resolveMediaUrl } from "$lib/config/runtime";
import { logger } from "$lib/core/logger";
import { setNativeMediaSync } from "$lib/media/native-media-sync";
import { canAdvanceQueue } from "$lib/music/playback-queue";
import { coverArtUrl } from "$lib/subsonic";
import type { PendingMediaAction } from "@bindings/melovian/services/models.js";

type MediaServiceModule = typeof import("@bindings/melovian/services/index.js");

function applyMediaAction(
  MediaService: MediaServiceModule["MediaService"],
  pending: PendingMediaAction,
) {
  switch (pending.action) {
    case "play":
      if (!music.playing) void music.togglePlay();
      break;
    case "pause":
      music.pause();
      break;
    case "next":
      music.next();
      break;
    case "previous":
      music.previous();
      break;
    case "seek":
      if (pending.seekMs >= 0) music.seek(pending.seekMs / 1000);
      break;
    case "openUri":
      if (pending.openUri) {
        void MediaService.RaiseWindow();
        void music.playOpenUri(pending.openUri);
      }
      break;
  }
}

export function bindNativeMedia(): () => void {
  if (!nativeDesktopAvailable()) {
    return () => {};
  }

  let syncInterval = 0;
  let pollInterval = 0;
  let lastSyncKey = "";
  let disposed = false;
  let syncImpl: (() => void) | null = null;

  void import("@bindings/melovian/services/index.js")
    .then(({ MediaService }) => {
      if (disposed) return;

      const sync = () => {
        const track = music.currentTrack;
        if (!track) {
          if (lastSyncKey === "idle") return;
          lastSyncKey = "idle";
          void MediaService.UpdatePlayback({
            title: "",
            artist: "",
            album: "",
            trackId: "",
            coverArtId: "",
            coverArtUrl: "",
            durationMs: 0,
            positionMs: 0,
            playing: false,
            canPlay: false,
            canPause: false,
            canGoNext: false,
            canGoPrevious: false,
          });
          return;
        }

        const positionBucket = music.playing
          ? Math.floor(music.currentTime)
          : Math.floor(music.currentTime / 5) * 5;
        const key = [
          track.id,
          music.playing ? "1" : "0",
          positionBucket,
          music.queue.length,
          music.queueIndex,
        ].join("|");
        if (key === lastSyncKey) return;
        lastSyncKey = key;

        void MediaService.UpdatePlayback({
          title: track.title,
          artist: track.artist ?? "",
          album: track.album ?? "",
          trackId: track.id,
          coverArtId: track.coverArt ?? track.albumId ?? track.id,
          coverArtUrl: resolveMediaUrl(
            coverArtUrl(
              music.config,
              track.coverArt ?? track.albumId ?? track.id,
              128,
              track.id,
            ) ?? "",
          ),
          durationMs: Math.floor(
            (music.duration || track.duration || 0) * 1000,
          ),
          positionMs: Math.floor(music.currentTime * 1000),
          playing: music.playing,
          canPlay: true,
          canPause: true,
          canGoNext: canAdvanceQueue(
            music.queueIndex,
            music.queue.length,
            music.repeat,
            music.shuffle,
          ),
          canGoPrevious: music.queue.length > 0,
        });
      };

      syncImpl = () => {
        lastSyncKey = "";
        sync();
      };
      setNativeMediaSync(syncImpl);

      const poll = () => {
        void MediaService.PollMediaAction().then(
          (pending: PendingMediaAction) => {
            if (pending?.action) {
              applyMediaAction(MediaService, pending);
              lastSyncKey = "";
              sync();
            }
          },
        );
      };

      syncInterval = window.setInterval(sync, 500);
      pollInterval = window.setInterval(poll, 100);
      sync();
      poll();
    })
    .catch((err) => {
      logger.warn(
        "Native media bindings unavailable",
        err instanceof Error ? { err: err.message } : { err: String(err) },
        "media.native",
      );
    });

  return () => {
    disposed = true;
    if (syncImpl) {
      setNativeMediaSync(null);
      syncImpl = null;
    }
    if (syncInterval) clearInterval(syncInterval);
    if (pollInterval) clearInterval(pollInterval);
  };
}
