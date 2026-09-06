// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  decorateTrack,
  registerTrackRuleForTests,
  resetExtensionsForTests,
} from "./registry";
import type { SubsonicSong } from "$lib/subsonic/types";

describe("decorateTrack", () => {
  it("returns empty decoration by default", () => {
    resetExtensionsForTests();
    const track = {
      id: "trk_1",
      title: "Song",
      artist: "Artist",
    } as SubsonicSong;
    expect(decorateTrack(track)).toEqual({});
  });

  it("applies matching artist and album rules", () => {
    resetExtensionsForTests();
    registerTrackRuleForTests({
      match: { artistContains: "C418" },
      decoration: {
        progressThumbUrl: "/extensions/demo-pack/thumb.png",
        progressParticleUrl: "/extensions/demo-pack/particle.png",
        progressColor: "#8b5a2b",
        coverOverlayIcon: "/extensions/demo-pack/note.png",
        playerTheme: "neon",
      },
    });
    registerTrackRuleForTests({
      match: { albumContains: "Demo" },
      decoration: {
        progressThumbUrl: "/extensions/demo-pack/thumb.png",
        progressParticleUrl: "/extensions/demo-pack/particle.png",
        playerTheme: "neon",
      },
    });

    const byArtist = decorateTrack({
      id: "trk_artist",
      title: "Sweden",
      artist: "C418",
      album: "Volume Alpha",
    } as SubsonicSong);
    expect(byArtist.progressThumbUrl).toContain("thumb.png");
    expect(byArtist.progressParticleUrl).toContain("particle.png");
    expect(byArtist.coverOverlayIcon).toContain("note.png");
    expect(byArtist.playerTheme).toBe("neon");

    const byAlbum = decorateTrack({
      id: "trk_album",
      title: "Otherside",
      artist: "Someone",
      album: "Demo: The Album",
    } as SubsonicSong);
    expect(byAlbum.progressThumbUrl).toContain("thumb.png");
    expect(byAlbum.playerTheme).toBe("neon");

    const unrelated = decorateTrack({
      id: "trk_other",
      title: "Helplessness Blues",
      artist: "Fleet Foxes",
      album: "Helplessness Blues",
    } as SubsonicSong);
    expect(unrelated).toEqual({});
  });

  it("merges titlePrefix and coverOverlayIcon", () => {
    resetExtensionsForTests();
    registerTrackRuleForTests({
      match: { titleContains: "Episode" },
      decoration: {
        titlePrefix: "[Pod] ",
        coverOverlayIcon: "mic",
      },
    });
    const track = decorateTrack({
      id: "trk_pod",
      title: "Episode 12",
      artist: "Host",
    } as SubsonicSong);
    expect(track.titlePrefix).toBe("[Pod] ");
    expect(track.coverOverlayIcon).toBe("mic");
  });

  it("lets later rules override earlier decoration", () => {
    resetExtensionsForTests();
    registerTrackRuleForTests({
      match: { artistContains: "Lena" },
      decoration: {
        playerTheme: "neon",
      },
    });
    registerTrackRuleForTests({
      match: {
        albumContains: "Nether",
        artistContains: "Lena",
      },
      decoration: {
        playerTheme: "neon-alt",
        progressGradient: "url(/extensions/demo-pack/lava.png)",
      },
    });
    const alt = decorateTrack({
      id: "trk_alt",
      title: "Pigstep",
      artist: "Lena Raine",
      album: "Demo: Nether Update",
    } as SubsonicSong);
    expect(alt.playerTheme).toBe("neon-alt");
    expect(alt.progressGradient).toContain("lava.png");

    const other = decorateTrack({
      id: "trk_other_lena",
      title: "Otherside",
      artist: "Lena Raine",
      album: "Demo: Caves & Cliffs",
    } as SubsonicSong);
    expect(other.playerTheme).toBe("neon");
  });
});
