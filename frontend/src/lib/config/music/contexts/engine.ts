// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicEngineContext } from "../engine-ops";
import type { MusicStoreHost } from "../types";

export function createEngineContext(store: MusicStoreHost): MusicEngineContext {
  return {
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
}
