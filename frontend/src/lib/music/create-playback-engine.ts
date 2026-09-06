// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { nativeDesktopAvailable } from "$lib/config/runtime";
import type { PlaybackEngine } from "./playback-engine";
import { AudioEngine } from "./audio-engine";
import { NativeAudioEngine } from "./native-audio-engine";
import { loadNativeBackendPref, loadNativePlaybackPref } from "./prefs";

export async function createPlaybackEngine(force?: "web" | "native"): Promise<{
  engine: PlaybackEngine;
  native: boolean;
}> {
  if (force === "web" || !nativeDesktopAvailable()) {
    return { engine: new AudioEngine(), native: false };
  }

  try {
    const { AudioService } =
      await import("@bindings/melovian/services/index.js");
    await AudioService.SetPreferredBackend(loadNativeBackendPref());
    const caps = await AudioService.Capabilities();
    if (caps.nativeAvailable && loadNativePlaybackPref()) {
      return { engine: new NativeAudioEngine(), native: true };
    }
  } catch {
    /* Wails bindings unavailable in browser or server mode */
  }
  return { engine: new AudioEngine(), native: false };
}
