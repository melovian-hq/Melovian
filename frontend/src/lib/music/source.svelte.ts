// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export type MusicSourceKind = "subsonic" | "local" | "unified";

let sourceKind = $state<MusicSourceKind>("subsonic");

export function getMusicSourceKind(): MusicSourceKind {
  return sourceKind;
}

export function setMusicSourceKind(kind: MusicSourceKind): void {
  sourceKind = kind;
}

export function isUnifiedMusicSource(): boolean {
  return sourceKind === "unified";
}

export function isLocalMusicSource(): boolean {
  return sourceKind === "local";
}

export function isLocalTrackId(id: string): boolean {
  return id.startsWith("trk_");
}
