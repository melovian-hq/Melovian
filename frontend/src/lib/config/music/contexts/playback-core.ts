// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicPlaybackCoreContext } from "../playback-core-ops";
import type { MusicStoreHost } from "../types";

export function createPlaybackCoreContext(
  store: MusicStoreHost,
): MusicPlaybackCoreContext {
  return {
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
}
