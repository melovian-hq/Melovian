// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach } from "vitest";
import {
  applyCompatFromConfig,
  Cap,
  CLIENT_VERSION,
  compareSemver,
  LEGACY_CAPABILITIES,
  resetCompatForTests,
  supports,
  versionScheme,
  versionsEqual,
} from "./index";

describe("compat", () => {
  beforeEach(() => {
    resetCompatForTests();
  });

  it("compares semver", () => {
    expect(compareSemver("0.1.0", "0.1.0")).toBe(0);
    expect(compareSemver("v0.1.0", "0.1.0")).toBe(0);
    expect(compareSemver("0.1.0", "0.2.0")).toBe(-1);
    expect(compareSemver("0.9.9", "0.10.0")).toBe(-1);
  });

  it("compares version labels exactly", () => {
    expect(versionsEqual("5c9f54d", "5c9f54d")).toBe(true);
    expect(versionsEqual("v0.1.0", "0.1.0")).toBe(true);
    expect(versionsEqual("0.1.0", "5c9f54d")).toBe(false);
  });

  it("classifies version schemes", () => {
    expect(versionScheme("416368d")).toBe("git");
    expect(versionScheme("0.1.0")).toBe("semver");
    expect(versionScheme("v1.2.3")).toBe("semver");
  });

  it("does not flag mismatch when version schemes differ", () => {
    const state = applyCompatFromConfig({
      version: "416368d",
      apiVersion: 1,
      capabilities: [Cap.browse, Cap.playback],
    });
    if (versionScheme(CLIENT_VERSION) !== "git") {
      expect(state.versionMismatch).toBe(false);
    }
  });

  it("flags mismatch for same-scheme differing versions", () => {
    const peer = versionScheme(CLIENT_VERSION) === "git" ? "deadbeef" : "9.9.9";
    const state = applyCompatFromConfig({
      version: peer,
      apiVersion: 1,
      capabilities: [Cap.browse, Cap.playback],
    });
    expect(state.versionMismatch).toBe(true);
    expect(state.blocked).toBe(false);
  });

  it("uses legacy capabilities when server omits fields", () => {
    const state = applyCompatFromConfig({});
    expect(state.legacy).toBe(true);
    expect(state.blocked).toBe(false);
    expect(supports(Cap.browse)).toBe(true);
    expect(supports(Cap.party)).toBe(false);
    for (const cap of LEGACY_CAPABILITIES) {
      expect(supports(cap)).toBe(true);
    }
  });

  it("intersects advertised capabilities", () => {
    applyCompatFromConfig({
      version: CLIENT_VERSION,
      apiVersion: 1,
      capabilities: [Cap.browse, Cap.playback, Cap.party],
    });
    expect(supports(Cap.browse)).toBe(true);
    expect(supports(Cap.party)).toBe(true);
    expect(supports(Cap.downloads)).toBe(false);
  });

  it("blocks when server is below min", () => {
    const state = applyCompatFromConfig({
      version: "0.0.1",
      apiVersion: 1,
      capabilities: [Cap.browse],
    });
    expect(state.serverTooOld).toBe(true);
    expect(state.blocked).toBe(true);
    expect(supports(Cap.browse)).toBe(false);
  });

  it("does not flag mismatch for matching versions", () => {
    const state = applyCompatFromConfig({
      version: CLIENT_VERSION,
      apiVersion: 1,
      capabilities: [Cap.browse],
    });
    expect(state.versionMismatch).toBe(false);
  });
});
