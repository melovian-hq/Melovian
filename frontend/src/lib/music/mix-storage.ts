// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { GeneratedMix } from "./mix-generator";
import type { SubsonicSong } from "$lib/subsonic/types";

export interface SlimPersonalMix {
  id: string;
  title: string;
  subtitle: string;
  tracks: SubsonicSong[];
  trackIds?: string[];
  coverArtId?: string;
  gradient: string;
}

export interface CachedMix {
  id: string;
  title: string;
  subtitle: string;
  coverArtId?: string;
  gradient: string;
  trackIds: string[];
}

export function mixTrackCount(mix: {
  tracks?: readonly { id: string }[];
  trackIds?: readonly string[];
}): number {
  if (mix.tracks && mix.tracks.length > 0) return mix.tracks.length;
  return mix.trackIds?.length ?? 0;
}

export function slimPersonalMix(
  mix: GeneratedMix | SlimPersonalMix,
): SlimPersonalMix {
  const trackIds =
    ("trackIds" in mix && mix.trackIds?.length
      ? mix.trackIds
      : mix.tracks.map((track) => track.id).filter((id) => id.length > 0)) ??
    [];
  return {
    id: mix.id,
    title: mix.title,
    subtitle: mix.subtitle,
    tracks: [],
    trackIds,
    coverArtId: mix.coverArtId,
    gradient: mix.gradient,
  };
}

export function toCachedMix(mix: GeneratedMix | SlimPersonalMix): CachedMix {
  const slim = slimPersonalMix(mix);
  return {
    id: slim.id,
    title: slim.title,
    subtitle: slim.subtitle,
    coverArtId: slim.coverArtId,
    gradient: slim.gradient,
    trackIds: slim.trackIds ?? [],
  };
}

export function fromCachedMix(cached: CachedMix): SlimPersonalMix {
  return {
    id: cached.id,
    title: cached.title,
    subtitle: cached.subtitle,
    tracks: [],
    trackIds: cached.trackIds,
    coverArtId: cached.coverArtId,
    gradient: cached.gradient,
  };
}

export function normalizeLoadedMix(
  mix: GeneratedMix | SlimPersonalMix | CachedMix,
): SlimPersonalMix {
  if (
    "trackIds" in mix &&
    Array.isArray(mix.trackIds) &&
    mix.trackIds.length > 0
  ) {
    const tracks = "tracks" in mix ? mix.tracks : [];
    if (!tracks || tracks.length === 0) {
      return fromCachedMix(mix as CachedMix);
    }
  }
  if ("tracks" in mix && mix.tracks.length > 0) {
    return slimPersonalMix(mix);
  }
  return slimPersonalMix(mix as GeneratedMix);
}
