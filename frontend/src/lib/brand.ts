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
