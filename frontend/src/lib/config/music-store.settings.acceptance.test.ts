// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import {
  defaultPlaybackSettings,
  mergePlaybackSettings,
} from "$lib/music/playback-settings";
import {
  defaultQueueSettings,
  mergeQueueSettings,
} from "$lib/music/queue-settings";
import {
  defaultTranscodingSettings,
  mergeTranscodingSettings,
} from "$lib/music/transcoding-settings";
import {
  defaultImmersiveAudioSettings,
  mergeImmersiveAudioSettings,
} from "$lib/music/immersive-audio-settings";
import {
  defaultMetadataEnhancementSettings,
  mergeMetadataEnhancementSettings,
} from "$lib/music/metadata-enhancement-settings";
import {
  applyHideUnknownMetadata,
  applyImmersiveAudioSettings,
  applyMetadataEnhancementSettings,
  applyPlaybackSettings,
  applyQueueSettings,
  applyTranscodingSettings,
  type MusicSettingsContext,
} from "./music/settings-ops";

function makeCtx(
  overrides: Partial<MusicSettingsContext> = {},
): MusicSettingsContext {
  return {
    transcodingSettings: defaultTranscodingSettings(),
    immersiveAudioSettings: defaultImmersiveAudioSettings(),
    queueSettings: defaultQueueSettings(),
    playbackSettings: defaultPlaybackSettings(),
    metadataEnhancementSettings: defaultMetadataEnhancementSettings(),
    hideUnknownMetadata: false,
    engine: null,
    applyQueueLimit: vi.fn(),
    ...overrides,
  };
}

describe("music settings ops (characterization)", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("merges and assigns transcoding settings", () => {
    const ctx = makeCtx();
    applyTranscodingSettings(ctx, { maxBitRate: 128 });
    expect(ctx.transcodingSettings).toEqual(
      mergeTranscodingSettings({ maxBitRate: 128 }),
    );
  });

  it("applies immersive settings to engine when present", () => {
    const applyImmersiveAudio = vi.fn();
    const ctx = makeCtx({
      engine: {
        applyImmersiveAudio,
      } as unknown as MusicSettingsContext["engine"],
    });
    applyImmersiveAudioSettings(ctx, { mode: "surround" });
    expect(ctx.immersiveAudioSettings).toEqual(
      mergeImmersiveAudioSettings({ mode: "surround" }),
    );
    expect(applyImmersiveAudio).toHaveBeenCalledWith(
      ctx.immersiveAudioSettings,
    );
  });

  it("updates queue settings and enforces limit", () => {
    const applyQueueLimit = vi.fn();
    const ctx = makeCtx({ applyQueueLimit });
    applyQueueSettings(ctx, { maxQueueSize: 50 });
    expect(ctx.queueSettings).toEqual(mergeQueueSettings({ maxQueueSize: 50 }));
    expect(applyQueueLimit).toHaveBeenCalledOnce();
  });

  it("updates playback settings", () => {
    const ctx = makeCtx();
    applyPlaybackSettings(ctx, { crossfadeEnabled: true });
    expect(ctx.playbackSettings).toEqual(
      mergePlaybackSettings({ crossfadeEnabled: true }),
    );
  });

  it("updates metadata enhancement settings", () => {
    const ctx = makeCtx();
    applyMetadataEnhancementSettings(ctx, { enabled: true });
    expect(ctx.metadataEnhancementSettings).toEqual(
      mergeMetadataEnhancementSettings({ enabled: true }),
    );
  });

  it("updates hide-unknown flag", () => {
    const ctx = makeCtx();
    applyHideUnknownMetadata(ctx, true);
    expect(ctx.hideUnknownMetadata).toBe(true);
  });
});
