// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

type PauseCallback = () => void;

const videoPauseCallbacks = new Set<PauseCallback>();

let pauseMusicHandler: (() => void) | null = null;

/** Wire music.pause without importing the music store (avoids cycles). */
export function setPauseMusicHandler(fn: (() => void) | null): void {
  pauseMusicHandler = fn;
}

/** Register a video pause/unload callback. Returns unsubscribe. */
export function registerVideoPause(cb: PauseCallback): () => void {
  videoPauseCallbacks.add(cb);
  return () => {
    videoPauseCallbacks.delete(cb);
  };
}

/** Pause music when video starts playing. */
export function pauseMusicForVideo(): void {
  pauseMusicHandler?.();
}

/** Pause or unload video when music transport starts. */
export function pauseVideoForMusic(): void {
  for (const cb of [...videoPauseCallbacks]) {
    try {
      cb();
    } catch {
      // Ignore callback failures so one broken player cannot block others.
    }
  }
}
