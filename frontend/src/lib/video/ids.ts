// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export type VideoSource = "local" | "invidious" | "youtube";
export type VideoSearchProvider = "invidious" | "youtube";

export interface LocalVideo {
  id: string;
  title: string;
  artist?: string;
  album?: string;
  duration?: number;
  format: string;
  relPath: string;
  mediaKind: string;
}

export interface VideoSettings {
  enabled: boolean;
  searchProvider: VideoSearchProvider;
  invidiousBaseUrl: string;
  youtubeApiKey: string;
}

export interface VideoSearchHit {
  id: string;
  title: string;
  author?: string;
  lengthSeconds?: number;
  thumbnail?: string;
}

export interface TrackVideoLink {
  trackId: string;
  source: VideoSource;
  videoId: string;
  title?: string;
  playId: string;
}

export interface VideoResolveResult {
  source: VideoSource;
  videoId: string;
  title?: string;
  author?: string;
  embedUrl?: string;
  playId: string;
}

const INVIDIOUS_PREFIX = "ext:invidious:";
const YOUTUBE_PREFIX = "ext:youtube:";
const YOUTUBE_ID_RE = /^[A-Za-z0-9_-]{11}$/;

export function defaultVideoSettings(): VideoSettings {
  return {
    enabled: false,
    searchProvider: "invidious",
    invidiousBaseUrl: "",
    youtubeApiKey: "",
  };
}

export function mergeVideoSettings(
  raw?: Partial<VideoSettings> | null,
): VideoSettings {
  const base = defaultVideoSettings();
  if (!raw) return base;
  return {
    enabled: Boolean(raw.enabled),
    searchProvider: raw.searchProvider === "youtube" ? "youtube" : "invidious",
    invidiousBaseUrl: (raw.invidiousBaseUrl ?? "").trim(),
    youtubeApiKey: (raw.youtubeApiKey ?? "").trim(),
  };
}

export function isInvidiousPlayId(playId: string): boolean {
  return playId.startsWith(INVIDIOUS_PREFIX);
}

export function isYouTubePlayId(playId: string): boolean {
  return playId.startsWith(YOUTUBE_PREFIX);
}

export function isExternalPlayId(playId: string): boolean {
  return isInvidiousPlayId(playId) || isYouTubePlayId(playId);
}

export function invidiousIdFromPlayId(playId: string): string | null {
  if (!isInvidiousPlayId(playId)) return null;
  const id = playId.slice(INVIDIOUS_PREFIX.length).trim();
  return id || null;
}

export function youtubeIdFromPlayId(playId: string): string | null {
  if (!isYouTubePlayId(playId)) return null;
  const id = playId.slice(YOUTUBE_PREFIX.length).trim();
  return id || null;
}

export function playIdForInvidious(videoId: string): string {
  return `${INVIDIOUS_PREFIX}${videoId}`;
}

export function playIdForYouTube(videoId: string): string {
  return `${YOUTUBE_PREFIX}${videoId}`;
}

export function looksLikeYouTubeVideoId(value: string): boolean {
  return YOUTUBE_ID_RE.test(value.trim());
}

/** Prefer a human title. Never surface a bare YouTube-style id as the label. */
export function displayVideoTitle(
  title: string | undefined | null,
  videoId?: string | null,
): string {
  const trimmed = (title ?? "").trim();
  const id = (videoId ?? "").trim();
  if (!trimmed) return "Music video";
  if (id && trimmed === id) return "Music video";
  if (looksLikeYouTubeVideoId(trimmed) && (!id || trimmed === id)) {
    return "Music video";
  }
  return trimmed;
}

export function videoPlayerPath(
  playId: string,
  opts?: { title?: string | null },
): string {
  const base = `/play/${encodeURIComponent(playId)}`;
  const title = opts?.title?.trim();
  if (!title || displayVideoTitle(title) === "Music video") return base;
  return `${base}?title=${encodeURIComponent(title)}`;
}

/** Default Invidious/YouTube search string for a library track. */
export function musicVideoSearchQuery(track: {
  title?: string | null;
  artist?: string | null;
}): string {
  const title = (track.title ?? "").trim();
  const artist = (track.artist ?? "").trim();
  const base = [artist, title].filter(Boolean).join(" ");
  if (!base) return "official music video";
  return `${base} official music video`;
}

/** Build an embed URL without a resolve round-trip when settings are known. */
export function buildEmbedUrl(
  settings: VideoSettings,
  source: VideoSource,
  videoId: string,
): string | null {
  const id = videoId.trim();
  if (!id) return null;
  if (source === "youtube") {
    return `https://www.youtube.com/embed/${encodeURIComponent(id)}`;
  }
  if (source === "invidious") {
    const base = settings.invidiousBaseUrl.trim().replace(/\/+$/, "");
    if (!base) return null;
    return `${base}/embed/${encodeURIComponent(id)}`;
  }
  return null;
}

export function searchReady(settings: VideoSettings): boolean {
  if (!settings.enabled) return false;
  if (settings.searchProvider === "youtube") {
    return settings.youtubeApiKey.trim().length > 0;
  }
  return settings.invidiousBaseUrl.trim().length > 0;
}
