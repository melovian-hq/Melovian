// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  runExtensionScript,
  SHADOWED_GLOBALS,
} from "$lib/extensions/registry";
import type { ExtensionAPI } from "$lib/extensions/types";

const api: ExtensionAPI = {
  registerTrackRule() {},
  decorateTrack: () => null,
  settings: {},
};

// Every identifier the runner shadows must be undefined inside extension
// code. This is the runtime counterpart to the repo audit's static
// blocklist: if a global gets reintroduced (browser API surface grows),
// this matrix fails.
describe("extension sandbox", () => {
  for (const name of SHADOWED_GLOBALS) {
    it(`shadows ${name}`, () => {
      const probe = `function register() { return typeof ${name}; }`;
      expect(runExtensionScript(probe, api)).toBe("undefined");
    });
  }

  it("keeps api reachable", () => {
    const probe = `function register(a) { return typeof a.registerTrackRule; }`;
    expect(runExtensionScript(probe, api)).toBe("function");
  });

  it("keeps builtins reachable", () => {
    const probe = `function register() { return typeof JSON.stringify + typeof Math.floor; }`;
    expect(runExtensionScript(probe, api)).toBe("functionfunction");
  });

  it("runs in strict mode so this is not the global object", () => {
    const probe = `function register() { return this === undefined; }`;
    expect(runExtensionScript(probe, api)).toBe(true);
  });

  it("exposes frozen settings to the script", () => {
    const withSettings: ExtensionAPI = {
      ...api,
      settings: Object.freeze({ accent: "red" }),
    };
    const reader = `function register(a) { return a.settings.accent; }`;
    expect(runExtensionScript(reader, withSettings)).toBe("red");
    const writer = `function register(a) { a.settings.accent = "blue"; }`;
    expect(() => runExtensionScript(writer, withSettings)).toThrow(TypeError);
  });

  it("a real register call reaches the api", () => {
    let seen = 0;
    const counting: ExtensionAPI = {
      registerTrackRule: () => {
        seen += 1;
      },
      decorateTrack: () => null,
      settings: {},
    };
    runExtensionScript(
      `function register(a) { a.registerTrackRule({ match: {}, decoration: {} }); }`,
      counting,
    );
    expect(seen).toBe(1);
  });
});
