// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as musicApi from "$lib/music/api";
import {
  loadEqFromLocalStorage,
  saveEqToLocalStorage,
  normalizeEqSettings,
  presetBands,
  type EqBandParam,
  type EqSettings,
} from "$lib/music/eq";
import type { PlaybackEngine } from "$lib/music/playback-engine";

export interface MusicEqContext {
  authEnabled: boolean;
  eq: EqSettings;
  eqAvailable: boolean;
  eqOpen: boolean;
  engine: PlaybackEngine | null;
}

export async function loadEqSettings(
  ctx: MusicEqContext,
  authEnabled: boolean,
) {
  ctx.authEnabled = authEnabled;
  if (authEnabled) {
    const remote = await musicApi.getEqSettings().catch(() => null);
    if (remote) {
      ctx.eq = normalizeEqSettings(remote);
    }
  } else {
    ctx.eq = loadEqFromLocalStorage();
  }
  if (!ctx.eqAvailable || !ctx.eq.enabled) return;
  ctx.engine?.applyEq(ctx.eq);
}

export async function persistEq(ctx: MusicEqContext) {
  if (ctx.authEnabled) {
    try {
      await musicApi.saveEqSettings(ctx.eq);
    } catch {
      /* best effort */
    }
  } else {
    saveEqToLocalStorage(ctx.eq);
  }
}

export function toggleEqPanel(ctx: MusicEqContext) {
  if (!ctx.eqAvailable) return;
  ctx.eqOpen = !ctx.eqOpen;
  if (ctx.eqOpen && !ctx.eq.enabled) {
    void setEqEnabled(ctx, true);
  }
}

export async function setEqEnabled(ctx: MusicEqContext, enabled: boolean) {
  if (!ctx.eqAvailable) return;
  ctx.eq = { ...ctx.eq, enabled };
  await persistEq(ctx);
  ctx.engine?.applyEq(ctx.eq);
}

export function setEqPreset(ctx: MusicEqContext, presetId: string) {
  if (!ctx.eqAvailable) return;
  ctx.eq = { ...ctx.eq, presetId, bands: presetBands(presetId) };
  void persistEq(ctx);
  ctx.engine?.applyEq(ctx.eq);
}

export function setEqBandParam(
  ctx: MusicEqContext,
  index: number,
  param: Partial<EqBandParam>,
) {
  if (!ctx.eqAvailable) return;
  const bands = ctx.eq.bands.map((band, i) =>
    i === index ? { ...band, ...param } : band,
  );
  ctx.eq = { ...ctx.eq, presetId: "custom", bands };
  void persistEq(ctx);
  ctx.engine?.applyEq(ctx.eq);
}

export function toggleEq(ctx: MusicEqContext, enabled?: boolean) {
  void setEqEnabled(ctx, enabled ?? !ctx.eq.enabled);
}

export function createEqOps(ctx: MusicEqContext) {
  return {
    loadEqSettings: (authEnabled: boolean) => loadEqSettings(ctx, authEnabled),
    persistEq: () => persistEq(ctx),
    toggleEqPanel: () => toggleEqPanel(ctx),
    setEqEnabled: (enabled: boolean) => setEqEnabled(ctx, enabled),
    setEqPreset: (presetId: string) => setEqPreset(ctx, presetId),
    setEqBandParam: (index: number, param: Partial<EqBandParam>) =>
      setEqBandParam(ctx, index, param),
    toggleEq: (enabled?: boolean) => toggleEq(ctx, enabled),
  };
}
