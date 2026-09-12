// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicSettingsContext } from "../settings-ops";
import type { MusicStoreHost } from "../types";

export function createSettingsContext(
  store: MusicStoreHost,
): MusicSettingsContext {
  return {
    get transcodingSettings() {
      return store.transcodingSettings;
    },
    set transcodingSettings(v) {
      store.transcodingSettings = v;
    },
    get immersiveAudioSettings() {
      return store.immersiveAudioSettings;
    },
    set immersiveAudioSettings(v) {
      store.immersiveAudioSettings = v;
    },
    get queueSettings() {
      return store.queueSettings;
    },
    set queueSettings(v) {
      store.queueSettings = v;
    },
    get playbackSettings() {
      return store.playbackSettings;
    },
    set playbackSettings(v) {
      store.playbackSettings = v;
    },
    get metadataEnhancementSettings() {
      return store.metadataEnhancementSettings;
    },
    set metadataEnhancementSettings(v) {
      store.metadataEnhancementSettings = v;
    },
    get hideUnknownMetadata() {
      return store.hideUnknownMetadata;
    },
    set hideUnknownMetadata(v) {
      store.hideUnknownMetadata = v;
    },
    get engine() {
      return store.engine;
    },
    applyQueueLimit: () => store.applyQueueLimit(),
  };
}
