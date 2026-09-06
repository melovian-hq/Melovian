// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import type { SubsonicSong } from "$lib/subsonic";
import {
  playAlbum,
  type MusicPlayLaunchContext,
} from "./music/play-launch-ops";

describe("music play-launch ops (characterization)", () => {
  it("playAlbum forwards songs to playTracks", () => {
    const playTracks = vi.fn();
    const ctx = {
      playTracks,
    } as unknown as MusicPlayLaunchContext;
    const songs = [{ id: "1", title: "A" }] as SubsonicSong[];
    playAlbum(ctx, songs, 2);
    expect(playTracks).toHaveBeenCalledWith(songs, 2);
  });
});
