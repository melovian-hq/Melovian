// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  dedupeTrackIdentities,
  dedupeTracksByQuality,
  trackIdentityKey,
  trackQualityScore,
} from "./track-identity";

function track(
  partial: Partial<{
    id: string;
    title: string;
    artist: string;
    bitRate: number;
    samplingRate: number;
    bitDepth: number;
    suffix: string;
    contentType: string;
  }> = {},
) {
  return { id: "t1", title: "Song", artist: "Artist", ...partial };
}

describe("trackIdentityKey", () => {
  it("normalizes case and whitespace", () => {
    const a = trackIdentityKey(track({ title: "  Song  ", artist: "ARTIST" }));
    const b = trackIdentityKey(track({ title: "song", artist: "artist" }));
    expect(a).toBe(b);
  });

  it("distinguishes different songs", () => {
    expect(trackIdentityKey(track({ title: "A" }))).not.toBe(
      trackIdentityKey(track({ title: "B" })),
    );
  });

  it("falls back to id when the title is empty", () => {
    expect(trackIdentityKey({ id: "abc", title: "" })).toBe("id:abc");
    expect(trackIdentityKey({ id: "abc", title: "" })).not.toBe(
      trackIdentityKey({ id: "xyz", title: "" }),
    );
  });

  it("merges encodes of the same song on different releases", () => {
    const single = track({ id: "single-id", title: "Song", artist: "Artist" });
    const remaster = track({
      id: "remaster-id",
      title: "song",
      artist: "Artist",
    });
    expect(trackIdentityKey(single)).toBe(trackIdentityKey(remaster));
  });
});

describe("dedupeTrackIdentities", () => {
  it("keeps the first occurrence of each identity", () => {
    const first = track({ id: "a", title: "Song" });
    const dup = track({ id: "b", title: "song" });
    const other = track({ id: "c", title: "Other" });
    expect(dedupeTrackIdentities([first, dup, other])).toEqual([first, other]);
  });
});

describe("trackQualityScore", () => {
  it("ranks lossless above lossy", () => {
    const flac = trackQualityScore(track({ suffix: "flac", bitRate: 900 }));
    const mp3 = trackQualityScore(track({ suffix: "mp3", bitRate: 320 }));
    expect(flac).toBeGreaterThan(mp3);
  });

  it("ranks higher bitrate above lower within lossy formats", () => {
    const high = trackQualityScore(track({ suffix: "mp3", bitRate: 320 }));
    const low = trackQualityScore(track({ suffix: "mp3", bitRate: 128 }));
    expect(high).toBeGreaterThan(low);
  });

  it("detects lossless from content type", () => {
    const lossless = trackQualityScore(
      track({ contentType: "audio/flac", bitRate: 800 }),
    );
    const lossy = trackQualityScore(
      track({ contentType: "audio/mpeg", bitRate: 800 }),
    );
    expect(lossless).toBeGreaterThan(lossy);
  });

  it("rewards bit depth and sample rate", () => {
    const hires = trackQualityScore(
      track({
        suffix: "flac",
        bitRate: 900,
        bitDepth: 24,
        samplingRate: 96000,
      }),
    );
    const plain = trackQualityScore(
      track({
        suffix: "flac",
        bitRate: 900,
        bitDepth: 16,
        samplingRate: 44100,
      }),
    );
    expect(hires).toBeGreaterThan(plain);
  });
});

describe("dedupeTracksByQuality", () => {
  it("keeps the highest quality variant by default", () => {
    const lossy = track({ id: "mp3", suffix: "mp3", bitRate: 128 });
    const lossless = track({ id: "flac", suffix: "flac", bitRate: 900 });
    const other = track({
      id: "o",
      title: "Other",
      suffix: "mp3",
      bitRate: 320,
    });
    expect(dedupeTracksByQuality([lossy, lossless, other])).toEqual([
      lossless,
      other,
    ]);
  });

  it("keeps the lowest bandwidth variant when preferHighQuality is false", () => {
    const lossy = track({ id: "mp3", suffix: "mp3", bitRate: 128 });
    const lossless = track({ id: "flac", suffix: "flac", bitRate: 900 });
    expect(
      dedupeTracksByQuality([lossless, lossy], false).map((t) => t.id),
    ).toEqual(["mp3"]);
  });

  it("preserves the input order of surviving picks", () => {
    const b = track({ id: "b", title: "B", suffix: "flac", bitRate: 900 });
    const a = track({ id: "a", title: "A", suffix: "mp3", bitRate: 320 });
    const aDup = track({ id: "a2", title: "a", suffix: "flac", bitRate: 900 });
    const result = dedupeTracksByQuality([b, a, aDup]);
    expect(result.map((t) => t.id)).toEqual(["b", "a2"]);
  });
});
