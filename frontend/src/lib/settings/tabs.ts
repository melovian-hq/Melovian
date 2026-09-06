// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export const SETTINGS_TAB_IDS = [
  "profile",
  "general",
  "servers",
  "playback",
  "downloads",
  "mixes",
  "connection",
  "lyrics",
  "video",
  "rocksky",
  "listenbrainz",
  "lastfm",
  "extensions",
  "about",
  "tasks",
] as const;

export type SettingsTabId = (typeof SETTINGS_TAB_IDS)[number];

export interface SettingsTab {
  id: SettingsTabId;
  label: string;
  description: string;
  keywords?: string[];
  tier?: "recommended" | "advanced";
}

export const SETTINGS_TABS: SettingsTab[] = [
  {
    id: "profile",
    label: "Profile",
    description: "Avatar and identity",
    keywords: ["avatar", "photo", "name"],
    tier: "recommended",
  },
  {
    id: "general",
    label: "General",
    description: "Account and storage",
    keywords: [
      "theme",
      "backup",
      "account",
      "appearance",
      "accent",
      "css",
      "dark",
      "light",
      "storage",
      "layout",
    ],
    tier: "recommended",
  },
  {
    id: "servers",
    label: "Sources",
    description: "Servers and local folders",
    keywords: ["navidrome", "subsonic", "folder", "library", "scan", "servers"],
    tier: "recommended",
  },
  {
    id: "playback",
    label: "Playback",
    description: "Audio engine and streaming",
    keywords: [
      "audio",
      "mpv",
      "crossfade",
      "volume",
      "transcode",
      "atmos",
      "dolby",
      "surround",
      "spatial",
      "binaural",
      "passthrough",
    ],
    tier: "recommended",
  },
  {
    id: "downloads",
    label: "Downloads",
    description: "Offline cache",
    keywords: ["cache", "offline", "download"],
    tier: "recommended",
  },
  {
    id: "about",
    label: "About",
    description: "Version and capabilities",
    keywords: ["version", "build", "server", "api", "info"],
    tier: "recommended",
  },
  {
    id: "mixes",
    label: "Mixes",
    description: "Personalized playlists",
    keywords: ["radio", "discover", "personal"],
    tier: "advanced",
  },
  {
    id: "connection",
    label: "Connection",
    description: "Reconnect behavior",
    keywords: ["offline", "retry", "network"],
    tier: "advanced",
  },
  {
    id: "lyrics",
    label: "Lyrics",
    description: "Providers and cache",
    keywords: ["lrclib", "synced"],
    tier: "advanced",
  },
  {
    id: "video",
    label: "Video",
    description: "Invidious music videos",
    keywords: ["invidious", "youtube", "music video", "embed"],
    tier: "advanced",
  },
  {
    id: "rocksky",
    label: "Rocksky",
    description: "Rocksky scrobbling",
    keywords: ["rocksky", "scrobble", "last.fm", "listenbrainz"],
    tier: "advanced",
  },
  {
    id: "listenbrainz",
    label: "ListenBrainz",
    description: "ListenBrainz scrobbling",
    keywords: ["listenbrainz", "scrobble", "maloja", "self-hosted"],
    tier: "advanced",
  },
  {
    id: "lastfm",
    label: "Last.fm",
    description: "Last.fm scrobbling",
    keywords: ["lastfm", "scrobble", "last.fm"],
    tier: "advanced",
  },
  {
    id: "extensions",
    label: "Extensions",
    description: "Bundled features and player rules",
    keywords: ["plugin", "script", "decoration", "lyrics", "metadata"],
    tier: "advanced",
  },
  {
    id: "tasks",
    label: "Tasks",
    description: "Background jobs",
    keywords: ["scan", "mix", "lyrics", "progress", "jobs"],
    tier: "advanced",
  },
];

export function isSettingsTabId(value: string): value is SettingsTabId {
  return (SETTINGS_TAB_IDS as readonly string[]).includes(value);
}

export function settingsTabFromPath(pathname: string): string {
  if (pathname === "/settings") return "";
  if (!pathname.startsWith("/settings/")) return "";
  return pathname.slice("/settings/".length).split("/")[0] ?? "";
}

export function isSettingsPath(pathname: string): boolean {
  return pathname === "/settings" || pathname.startsWith("/settings/");
}

export function settingsTabPath(tab: SettingsTabId): string {
  return `/settings/${tab}`;
}

/** Tabs visible for the current extension feature set and server capabilities. */
export function visibleSettingsTabs(opts: {
  lyrics: boolean;
  rocksky?: boolean;
  listenbrainz?: boolean;
  lastfm?: boolean;
  supports?: (cap: string) => boolean;
}): SettingsTab[] {
  const can = opts.supports ?? (() => true);
  return SETTINGS_TABS.filter((tab) => {
    if (tab.id === "lyrics") return opts.lyrics && can("lyrics");
    if (tab.id === "rocksky") return opts.rocksky ?? false;
    if (tab.id === "listenbrainz") return opts.listenbrainz ?? false;
    if (tab.id === "lastfm") return opts.lastfm ?? false;
    if (tab.id === "downloads") return can("downloads");
    if (tab.id === "mixes") return can("mixes");
    if (tab.id === "extensions") return can("extensions");
    if (tab.id === "video") return can("videos");
    if (tab.id === "servers") return can("instances") || can("local_library");
    return true;
  });
}

/** Filter tabs by label, description, or keywords. Empty query returns all tabs. */
export function filterSettingsTabs(
  tabs: readonly SettingsTab[],
  query: string,
): SettingsTab[] {
  const q = query.trim().toLowerCase();
  if (!q) return [...tabs];
  return tabs.filter((tab) => {
    if (tab.label.toLowerCase().includes(q)) return true;
    if (tab.description.toLowerCase().includes(q)) return true;
    return (tab.keywords ?? []).some((keyword) =>
      keyword.toLowerCase().includes(q),
    );
  });
}
