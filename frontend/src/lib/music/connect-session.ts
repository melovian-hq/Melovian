// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export function shouldSkipMusicConnect(
  connected: boolean,
  force = false,
): boolean {
  return connected && !force;
}

export function shouldAwaitConnectInFlight(
  inFlight: Promise<boolean> | null,
  force = false,
): inFlight is Promise<boolean> {
  return inFlight !== null && !force;
}
