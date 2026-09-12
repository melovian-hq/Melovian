// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicPlayerChromeContext } from "../player-chrome-ops";
import type { MusicStoreHost } from "../types";

export function createPlayerChromeContext(
  store: MusicStoreHost,
): MusicPlayerChromeContext {
  return {
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
}
