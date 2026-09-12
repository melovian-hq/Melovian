// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicEqContext } from "../eq-ops";
import type { MusicStoreHost } from "../types";

export function createEqContext(store: MusicStoreHost): MusicEqContext {
  return {
    get authEnabled() {
      return store.authEnabled;
    },
    set authEnabled(v) {
      store.authEnabled = v;
    },
    get eq() {
      return store.eq;
    },
    set eq(v) {
      store.eq = v;
    },
    get eqAvailable() {
      return store.eqAvailable;
    },
    get eqOpen() {
      return store.eqOpen;
    },
    set eqOpen(v) {
      store.eqOpen = v;
    },
    get engine() {
      return store.engine;
    },
  };
}
