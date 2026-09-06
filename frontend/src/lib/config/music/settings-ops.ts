// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { PlaybackEngine } from "$lib/music/playback-engine";
import {
  mergeImmersiveAudioSettings,
  saveImmersiveAudioSettings,
  type ImmersiveAudioSettings,
} from "$lib/music/immersive-audio-settings";
import { saveHideUnknownMetadata } from "$lib/music/library-display-settings";
import {
  mergeMetadataEnhancementSettings,
  saveMetadataEnhancementSettings,
  type MetadataEnhancementSettings,
} from "$lib/music/metadata-enhancement-settings";
import {
  mergePlaybackSettings,
  savePlaybackSettings,
  type PlaybackSettings,
} from "$lib/music/playback-settings";
import {
  mergeQueueSettings,
  saveQueueSettings,
  type QueueSettings,
} from "$lib/music/queue-settings";
import {
  mergeTranscodingSettings,
  saveTranscodingSettings,
  type TranscodingSettings,
} from "$lib/music/transcoding-settings";

export interface MusicSettingsContext {
  transcodingSettings: TranscodingSettings;
  immersiveAudioSettings: ImmersiveAudioSettings;
  queueSettings: QueueSettings;
  playbackSettings: PlaybackSettings;
  metadataEnhancementSettings: MetadataEnhancementSettings;
  hideUnknownMetadata: boolean;
  engine: PlaybackEngine | null;
  applyQueueLimit(): void;
}

export function updateTranscodingSettings(
  ctx: MusicSettingsContext,
  settings: Partial<TranscodingSettings>,
) {
  ctx.transcodingSettings = mergeTranscodingSettings(settings);
  saveTranscodingSettings(ctx.transcodingSettings);
}

export function updateImmersiveAudioSettings(
  ctx: MusicSettingsContext,
  settings: Partial<ImmersiveAudioSettings>,
) {
  ctx.immersiveAudioSettings = mergeImmersiveAudioSettings(settings);
  saveImmersiveAudioSettings(ctx.immersiveAudioSettings);
  ctx.engine?.applyImmersiveAudio(ctx.immersiveAudioSettings);
}

export function updateQueueSettings(
  ctx: MusicSettingsContext,
  settings: Partial<QueueSettings>,
) {
  ctx.queueSettings = mergeQueueSettings(settings);
  saveQueueSettings(ctx.queueSettings);
  ctx.applyQueueLimit();
}

export function updatePlaybackSettings(
  ctx: MusicSettingsContext,
  settings: Partial<PlaybackSettings>,
) {
  ctx.playbackSettings = mergePlaybackSettings(settings);
  savePlaybackSettings(ctx.playbackSettings);
}

export function updateMetadataEnhancementSettings(
  ctx: MusicSettingsContext,
  settings: Partial<MetadataEnhancementSettings>,
) {
  ctx.metadataEnhancementSettings = mergeMetadataEnhancementSettings(settings);
  saveMetadataEnhancementSettings(ctx.metadataEnhancementSettings);
}

export function setHideUnknownMetadata(
  ctx: MusicSettingsContext,
  hide: boolean,
) {
  ctx.hideUnknownMetadata = hide;
  saveHideUnknownMetadata(hide);
}

/** Aliases used by characterization tests. */
export const applyTranscodingSettings = updateTranscodingSettings;
export const applyImmersiveAudioSettings = updateImmersiveAudioSettings;
export const applyQueueSettings = updateQueueSettings;
export const applyPlaybackSettings = updatePlaybackSettings;
export const applyMetadataEnhancementSettings =
  updateMetadataEnhancementSettings;
export const applyHideUnknownMetadata = setHideUnknownMetadata;

export function createSettingsOps(ctx: MusicSettingsContext) {
  return {
    updateTranscodingSettings: (settings: Partial<TranscodingSettings>) =>
      updateTranscodingSettings(ctx, settings),
    updateImmersiveAudioSettings: (settings: Partial<ImmersiveAudioSettings>) =>
      updateImmersiveAudioSettings(ctx, settings),
    updateQueueSettings: (settings: Partial<QueueSettings>) =>
      updateQueueSettings(ctx, settings),
    updatePlaybackSettings: (settings: Partial<PlaybackSettings>) =>
      updatePlaybackSettings(ctx, settings),
    updateMetadataEnhancementSettings: (
      settings: Partial<MetadataEnhancementSettings>,
    ) => updateMetadataEnhancementSettings(ctx, settings),
    setHideUnknownMetadata: (hide: boolean) =>
      setHideUnknownMetadata(ctx, hide),
  };
}
