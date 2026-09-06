// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, vi } from "vitest";

// jsdom does not implement media playback. Stub the methods so engine code can
// run in tests without emitting "Not implemented" noise.
for (const method of ["play", "pause", "load"] as const) {
  Object.defineProperty(HTMLMediaElement.prototype, method, {
    configurable: true,
    value:
      method === "play"
        ? () => Promise.resolve()
        : () => {
            /* no-op in jsdom */
          },
  });
}

// Svelte MediaQuery (prefersReducedMotion) calls matchMedia at module load.
// Stub before any component imports, not only in beforeEach.
function stubMatchMedia(query: string) {
  return {
    matches: false,
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  };
}
vi.stubGlobal("matchMedia", stubMatchMedia);

const storage = new Map<string, string>();

beforeEach(() => {
  storage.clear();
  vi.stubGlobal("matchMedia", stubMatchMedia);
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => {
      storage.set(key, value);
    },
    removeItem: (key: string) => {
      storage.delete(key);
    },
    clear: () => {
      storage.clear();
    },
    key: (index: number) => Array.from(storage.keys())[index] ?? null,
    get length() {
      return storage.size;
    },
  });
});
