// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  isInternetRadioTrack,
  radioStreamUrl,
  radioTrackId,
  trackFromRadioStation,
} from "./radio";
import type { InternetRadioStation } from "./types";

const station: InternetRadioStation = {
  id: "42",
  name: "Jazz FM",
  streamUrl: "https://stream.example.com/jazz",
  coverArt: "art-42",
};

describe("radio helpers", () => {
  it("builds stable radio track ids", () => {
    expect(radioTrackId("42")).toBe("ir:42");
  });

  it("maps stations to queue tracks", () => {
    const track = trackFromRadioStation(station);
    expect(track.id).toBe("ir:42");
    expect(track.title).toBe("Jazz FM");
    expect(track.isInternetRadio).toBe(true);
    expect(track.streamUrl).toBe("https://stream.example.com/jazz");
  });

  it("detects internet radio tracks", () => {
    expect(isInternetRadioTrack(trackFromRadioStation(station))).toBe(true);
    expect(isInternetRadioTrack({ id: "song-1" })).toBe(false);
  });

  it("passes through absolute stream urls", () => {
    const track = trackFromRadioStation(station);
    expect(radioStreamUrl(track)).toBe("https://stream.example.com/jazz");
  });
});
