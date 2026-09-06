// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export type BottomNavTab = "home" | "search" | "library" | "playlists" | "more";

function startsWithPath(pathname: string, prefix: string): boolean {
  return pathname === prefix || pathname.startsWith(`${prefix}/`);
}

/** Which bottom tab should look active for the current route. */
export function bottomNavActiveTab(
  pathname: string,
  sidebarOpen: boolean,
): BottomNavTab {
  if (sidebarOpen) return "more";

  if (pathname === "/music" || pathname === "/music/") return "home";
  if (startsWithPath(pathname, "/music/search")) return "search";
  if (
    startsWithPath(pathname, "/music/playlists") ||
    startsWithPath(pathname, "/music/playlist") ||
    startsWithPath(pathname, "/music/server-playlist")
  ) {
    return "playlists";
  }
  if (
    startsWithPath(pathname, "/music/albums") ||
    startsWithPath(pathname, "/music/album") ||
    startsWithPath(pathname, "/music/artists") ||
    startsWithPath(pathname, "/music/artist") ||
    startsWithPath(pathname, "/music/genres") ||
    startsWithPath(pathname, "/music/genre")
  ) {
    return "library";
  }
  if (
    startsWithPath(pathname, "/settings") ||
    startsWithPath(pathname, "/music/favorites") ||
    startsWithPath(pathname, "/music/history") ||
    startsWithPath(pathname, "/music/videos") ||
    startsWithPath(pathname, "/music/metadata") ||
    startsWithPath(pathname, "/music/shared") ||
    startsWithPath(pathname, "/music/now-playing") ||
    startsWithPath(pathname, "/music/lyrics") ||
    startsWithPath(pathname, "/music/mix")
  ) {
    return "more";
  }

  return "home";
}
