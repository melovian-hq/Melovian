// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { SubsonicSong } from "$lib/subsonic/types";

export type TrackMatch = {
  genreContains?: string;
  artistContains?: string;
  albumContains?: string;
  titleContains?: string;
  titleRegex?: string;
  tagEquals?: string;
  minRating?: number;
  isLocal?: boolean;
};

export type TrackDecoration = {
  progressColor?: string;
  progressGradient?: string;
  progressThumbUrl?: string;
  progressParticleUrl?: string;
  icon?: string;
  iconUrl?: string;
  titlePrefix?: string;
  coverOverlayIcon?: string;
  /** Named player chrome theme. Example: "neon". */
  playerTheme?: string;
};

export type TrackRule = {
  match: TrackMatch;
  decoration: TrackDecoration;
};

export type ExtensionManifest = {
  id: string;
  name: string;
  version: string;
  description?: string;
  author?: string;
  script?: string;
  /** Optional icon path relative to the extension folder. */
  icon?: string;
  /** Optional larger artwork path relative to the extension folder. */
  image?: string;
  /** Named app chrome theme. Example: "neon". */
  appTheme?: string;
  /** Stylesheet paths relative to the extension folder. Injected when enabled. */
  styles?: string[];
  trackRules?: TrackRule[];
  playerHooks?: Array<{
    when: string;
    style?: Record<string, string>;
  }>;
};

export type DecorateTrackContext = {
  track: SubsonicSong;
  isLocal: boolean;
  genre?: string;
  rating?: number;
  tags?: string[];
};

export type ExtensionAPI = {
  registerTrackRule: (rule: TrackRule) => void;
  decorateTrack: (
    ctx: DecorateTrackContext,
  ) => TrackDecoration | null | undefined;
};
