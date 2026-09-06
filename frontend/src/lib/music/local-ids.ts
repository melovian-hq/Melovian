// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import XXH from "xxhashjs";

const ARTIST_ID_PREFIX = "art_";

function normalizeName(value: string): string {
  return value.trim().toLowerCase();
}

function hashId(prefix: string, key: string): string {
  const sum = XXH.h64(key, 0);
  const buf = new ArrayBuffer(8);
  const view = new DataView(buf);
  view.setBigUint64(0, BigInt(sum.toString()));
  const bytes = new Uint8Array(buf);
  let hex = "";
  for (const byte of bytes) {
    hex += byte.toString(16).padStart(2, "0");
  }
  return prefix + hex;
}

export function artistIdFromName(name: string): string {
  const normalized = normalizeName(name);
  if (!normalized) return "";
  return hashId(ARTIST_ID_PREFIX, normalized);
}
