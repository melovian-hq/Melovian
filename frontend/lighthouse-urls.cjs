// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * Paths audited by Lighthouse CI against the static demo build
 * (VITE_STATIC_DEMO=true). Demo mode redirects /account/login and
 * /settings* to /music, so those are omitted. Dynamic segments use
 * known democatalog IDs.
 */

const BASE = "http://127.0.0.1:4173";

/** Demo catalog fixtures (keep in sync with e2e/fixtures.ts). */
const DEMO = {
  albumId: "al-006-01",
  artistId: "ar-006",
  playlistId: "pl-hits",
  mixId: "daily-mix-1",
  genre: "Pop",
  shareToken: "demo-share",
  listenToken: "demo-listen",
  videoId: "demo-video",
};

/**
 * Static app routes that render distinct page modules in demo mode.
 * Keep in sync with frontend/src/routes.ts (demo-reachable paths only).
 */
const PATHS = [
  "/music",
  "/music/search",
  "/music/artists",
  "/music/albums",
  "/music/genres",
  `/music/genre/${encodeURIComponent(DEMO.genre)}`,
  `/music/album/${DEMO.albumId}`,
  `/music/artist/${DEMO.artistId}`,
  `/music/mix/${DEMO.mixId}`,
  "/music/playlists",
  `/music/server-playlist/${DEMO.playlistId}`,
  `/music/playlist/${DEMO.playlistId}`,
  "/music/shared",
  "/music/favorites",
  "/music/history",
  "/music/videos",
  `/play/${DEMO.videoId}`,
  "/music/now-playing",
  "/music/lyrics",
  "/music/metadata",
  `/share/${DEMO.shareToken}`,
  `/listen/${DEMO.listenToken}`,
];

/** Paths that redirect away under demo mode (not audited). */
const DEMO_REDIRECT_PATHS = [
  "/",
  "/login",
  "/account/login",
  "/settings",
  "/settings/profile",
  "/instances",
];

module.exports = {
  BASE,
  DEMO,
  PATHS,
  DEMO_REDIRECT_PATHS,
  urls: PATHS.map((path) => `${BASE}${path}`),
};
