// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { isMixEnabled } from "../mix-settings";
import { mixBuilders } from "./builders";
import { createMixBuildState } from "./context";
import { createCachedFetchers, hydrateStarred } from "./fetch";
import { MIX_PRIORITY, sortMixesByPriority } from "./priority";
import type { GeneratedMix, MixBuildContext, MixFetchers } from "./types";

export {
  createSeededRandom,
  dedupeTracks,
  interleaveByArtist,
  orderForFlow,
  selectMixTracks,
  shuffleWithSeed,
  clusterArtistFeatures,
  emptyMixSeed,
} from "./types";
export type {
  FlowOptions,
  MixSeedProfile,
  MixSelectPolicy,
  ArtistCluster,
  ArtistFeature,
} from "./types";
export type {
  GeneratedMix,
  MixBuildContext,
  MixBuildState,
  MixFetchers,
} from "./types";
export { pickUniqueMixCover } from "./helpers";
export { sortMixesByPriority, upsertMix } from "./priority";
export {
  createMixBuildState,
  buildMixContextInput,
  isInferredSkip,
} from "./context";

export async function buildPersonalMixes(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  onMix?: (mix: GeneratedMix) => void,
): Promise<GeneratedMix[]> {
  const state = createMixBuildState(ctx);
  const cachedFetchers = createCachedFetchers(fetchers);
  await hydrateStarred(state, cachedFetchers);
  const mixes: GeneratedMix[] = [];
  const seenIds = new Set<string>();

  const add = (mix: GeneratedMix | null) => {
    if (!mix || seenIds.has(mix.id)) return;
    seenIds.add(mix.id);
    mixes.push(mix);
    onMix?.(mix);
  };

  for (const [index, build] of mixBuilders(
    ctx,
    cachedFetchers,
    state,
  ).entries()) {
    const id = MIX_PRIORITY[index];
    if (!isMixEnabled(id, ctx.settings)) continue;
    add(await build());
  }

  return sortMixesByPriority(mixes);
}

export async function buildSingleMix(
  mixId: string,
  ctx: MixBuildContext,
  fetchers: MixFetchers,
): Promise<GeneratedMix | null> {
  const state = createMixBuildState(ctx);
  const cachedFetchers = createCachedFetchers(fetchers);
  await hydrateStarred(state, cachedFetchers);
  const index = MIX_PRIORITY.indexOf(mixId);
  if (index < 0) return null;
  if (!isMixEnabled(mixId, ctx.settings)) return null;
  const builders = mixBuilders(ctx, cachedFetchers, state);
  const build = builders[index];
  if (!build) return null;
  return (await build()) ?? null;
}
