// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { loadPlaybackSettings } from "./playback-settings";
import { loadSavedPlayback } from "./prefs";

export function shouldRestorePlayback(state: {
  playbackRestored: boolean;
  playing: boolean;
  queueLength: number;
  queueIndex: number;
}): boolean {
  if (state.playbackRestored) return false;
  if (state.playing) return false;
  if (state.queueLength > 0 && state.queueIndex >= 0) return false;
  if (!loadPlaybackSettings().continuePlaybackOnLaunch) return false;
  return loadSavedPlayback() !== null;
}
