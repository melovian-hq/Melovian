// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * offline-gate decides when playback failures mean the server is gone and the
 * queue must stop advancing, versus a single bad track that should be skipped.
 * It reads the connection store's managed/serverOnline/browserOnline signals
 * plus the consecutive failure count as evidence.
 */

export interface OfflineGateState {
  /** Connection health is tracked for the active source (subsonic only). */
  managed: boolean;
  browserOnline: boolean;
  serverOnline: boolean;
}

/**
 * OFFLINE_SUSPECT_SKIP_THRESHOLD is the number of consecutive track failures
 * that counts as outage evidence even when the connection store still reports
 * the server as online. Audio elements often surface a dead server as
 * SRC_NOT_SUPPORTED or decode errors instead of MEDIA_ERR_NETWORK, so a run
 * of failures is the only reliable signal.
 */
export const OFFLINE_SUSPECT_SKIP_THRESHOLD = 3;

/**
 * playbackHoldActive is true when the queue must not advance because the
 * server is known unreachable. Sources that never init the connection store
 * (local-only, demo) never hold: their playback does not depend on the
 * subsonic server. Unified mode is managed when it includes a subsonic
 * instance, so remote-track failures there can hold the queue while local
 * tracks keep playing.
 */
export function playbackHoldActive(state: OfflineGateState): boolean {
  if (!state.managed) return false;
  return !state.browserOnline || !state.serverOnline;
}

/**
 * failuresSuggestOutage is true when consecutive per-track failures are
 * themselves evidence that the server went away, used to flip the connection
 * store into its reconnect path before serverOnline would otherwise update.
 */
export function failuresSuggestOutage(consecutiveFailures: number): boolean {
  return consecutiveFailures >= OFFLINE_SUSPECT_SKIP_THRESHOLD;
}
