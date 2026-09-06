// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { absoluteMediaUri } from "$lib/music/open-uri";

const LOCAL_STREAM_RE =
  /\/api\/local-music\/tracks\/([^/?#]+)\/stream(?:[/?#]|$)/;
const DOWNLOAD_STREAM_RE = /\/api\/downloads\/([^/?#]+)\/stream(?:[/?#]|$)/;

export function isDownloadStreamUrl(uri: string): boolean {
  return DOWNLOAD_STREAM_RE.test(absoluteMediaUri(uri));
}

export function isOfflinePlaybackUrl(uri: string): boolean {
  const absolute = absoluteMediaUri(uri);
  return LOCAL_STREAM_RE.test(absolute) || DOWNLOAD_STREAM_RE.test(absolute);
}
