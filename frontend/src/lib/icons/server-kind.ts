// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export type ServerKind = "navidrome" | "subsonic";

export type SourceKind = ServerKind | "local" | "unified";

export function detectServerKind(
  serverName?: string,
  version?: string,
): ServerKind {
  const name = (serverName ?? "").trim().toLowerCase();
  const ver = (version ?? "").trim().toLowerCase();
  const haystack = `${name} ${ver}`;
  if (
    haystack.includes("navidrome") ||
    name === "nd" ||
    name.startsWith("navidrome/")
  ) {
    return "navidrome";
  }
  return "subsonic";
}

export function serverIconPath(kind: ServerKind): string {
  if (kind === "navidrome") {
    return "/icons/navidrome.svg";
  }
  return "/icons/subsonic.svg";
}
