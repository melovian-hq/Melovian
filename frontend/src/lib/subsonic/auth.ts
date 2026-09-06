// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { SubsonicConfig } from "./types";
import { APP_SLUG } from "$lib/brand";

export function createDefaultSubsonicConfig(): SubsonicConfig {
  return {
    serverUrl: "/api/subsonic",
    clientName: APP_SLUG,
    version: "1.16.1",
  };
}

export function normalizeSubsonicUrl(serverUrl: string): string {
  return serverUrl.replace(/\/+$/, "");
}
