// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * Central brand identity. Rebrand the app by setting VITE_APP_NAME and
 * VITE_APP_SLUG at build time, or by editing the fallbacks here.
 *
 * APP_NAME is the display name users see. APP_SLUG is the lowercase
 * identifier used for file names, storage prefixes, and server product names.
 *
 * Warning: changing APP_SLUG or STORAGE_PREFIX orphans existing user data
 * (localStorage keys, download file names, extension manifests). Keep them
 * stable unless the rebrand intends a clean break.
 */

export const APP_NAME: string =
  (typeof import.meta !== "undefined" &&
    (import.meta.env?.VITE_APP_NAME as string | undefined)) ||
  "Melovian";

export const APP_SLUG: string =
  (typeof import.meta !== "undefined" &&
    (import.meta.env?.VITE_APP_SLUG as string | undefined)) ||
  "melovian";

/** One-line product blurb for meta tags, Open Graph, and installers. */
export const APP_DESCRIPTION: string =
  (typeof import.meta !== "undefined" &&
    (import.meta.env?.VITE_APP_DESCRIPTION as string | undefined)) ||
  "Music player for local libraries and Subsonic-compatible servers";

/** Default Open Graph / Twitter card image (site-root path). */
export const DEFAULT_OG_IMAGE = "/og.png";

/** Local and session storage key prefix. Changing this orphans saved settings. */
export const STORAGE_PREFIX = APP_SLUG;

/** Build a localStorage key under the app prefix. */
export function storageKey(name: string): string {
  return `${STORAGE_PREFIX}-${name}`;
}

/** Well-known storage keys, kept here so a rebrand sees them in one place. */
export const StorageKeys = {
  theme: storageKey("theme"),
  themeAccent: storageKey("theme-accent"),
  customCss: storageKey("custom-css"),
  sidebarCollapsed: storageKey("sidebar-collapsed"),
  eq: storageKey("eq"),
  profileAvatar: storageKey("profile-avatar"),
  profileSmileVariants: storageKey("profile-smile-variants"),
  /** Dot-separated legacy keys kept for compatibility. */
  remoteServerUrl: `${APP_SLUG}.remoteServerUrl`,
  playlistKind: `${APP_SLUG}.playlists.kind`,
  playlistView: `${APP_SLUG}.playlists.view.v2`,
  /**
   * Legacy "mel-" keys kept for compatibility. These predate STORAGE_PREFIX
   * and must stay literal so existing installs keep their saved settings.
   */
  musicVolume: "mel-music-volume",
  musicPlayback: "mel-music-playback",
  nativePlayback: "mel-native-playback",
  nativeBackend: "mel-native-backend",
  queuePanelPosition: "mel-queue-panel-position",
  lyricsPanelPosition: "mel-lyrics-panel-position",
  lyricsPanelSize: "mel-lyrics-panel-size",
  homeHidden: "mel-home-hidden",
  musicMixes: "mel-music-mixes",
  playbackSettings: "mel-playback-settings",
  queueSettings: "mel-queue-settings",
  mixSettings: "mel-mix-settings",
  mixDisplay: "mel-mix-display",
  connectionSettings: "mel-connection-settings",
  connectionHistory: "mel-connection-history",
  transcodingSettings: "mel-transcoding-settings",
  immersiveAudioSettings: "mel-immersive-audio-settings",
  metadataEnhancementSettings: "mel-metadata-enhancement-settings",
  hideUnknownMetadata: "mel-hide-unknown-metadata",
  searchHistory: "mel-search-history",
  deviceId: "mel-device-id",
  deviceName: "mel-device-name",
  desktopIntegration: "mel-desktop-integration",
  homeTipsDismissed: "mel-home-tips-dismissed",
  /** Legacy "mel-" key prefixes used for per-item cache entries. */
  metaArtPrefix: "mel-meta-art:",
  artistInfoPrefix: "mel-artist-info:",
  compatMismatchPrefix: "mel-compat-mismatch-dismissed:",
  spaReloadPrefix: "mel-spa-reload:",
} as const;

/**
 * Storage prefixes swept by backup export and restore. "mel-" and "melovian-"
 * are kept for legacy installs.
 */
export const STORAGE_BACKUP_PREFIXES = Array.from(
  new Set(["mel-", "melovian-", `${STORAGE_PREFIX}-`]),
);

/** Extension manifest file name, mirrored from internal/extensions. */
export const EXTENSION_MANIFEST = `${APP_SLUG}-extension.json`;

/** Base name for backup file downloads. */
export const BACKUP_FILE_BASENAME = `${APP_SLUG}-backup`;

/** Base name for the downloads zip export. */
export const DOWNLOADS_ZIP_BASENAME = `${APP_SLUG}-downloads`;

/** Product name reported as the Subsonic server identity. */
export const SERVER_PRODUCT_NAME = APP_NAME;

/** Product name used for listen submission metadata (scrobbling). */
export const SUBMISSION_CLIENT_NAME = APP_NAME;
