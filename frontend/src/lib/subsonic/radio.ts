// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { resolveMediaUrl } from "$lib/config/runtime";
import type { InternetRadioStation, QueueTrack } from "./types";

export const RADIO_TRACK_ID_PREFIX = "ir:";

export function radioTrackId(stationId: string): string {
  return `${RADIO_TRACK_ID_PREFIX}${stationId}`;
}

export function isInternetRadioTrack(track: {
  id?: string;
  isInternetRadio?: boolean;
}): boolean {
  return (
    track.isInternetRadio === true ||
    (track.id?.startsWith(RADIO_TRACK_ID_PREFIX) ?? false)
  );
}

export function trackFromRadioStation(
  station: InternetRadioStation,
): QueueTrack {
  return {
    id: radioTrackId(station.id),
    title: station.name,
    artist: "Internet Radio",
    coverArt: station.coverArt,
    streamUrl: station.streamUrl,
    isInternetRadio: true,
  };
}

export function radioStationIdFromTrackId(trackId: string): string | null {
  if (!trackId.startsWith(RADIO_TRACK_ID_PREFIX)) return null;
  return trackId.slice(RADIO_TRACK_ID_PREFIX.length);
}

export function radioStreamUrl(track: QueueTrack): string {
  if (!track.streamUrl) return "";
  return resolveMediaUrl(track.streamUrl);
}
