// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { RockskyTrack } from "./api";

export function toRockskyTrack(track: {
  id: string;
  title: string;
  artist?: string;
  album?: string;
  duration?: number;
}): RockskyTrack {
  return {
    id: track.id,
    title: track.title,
    artist: track.artist ?? "",
    album: track.album ?? "",
    duration: track.duration ?? 0,
  };
}
