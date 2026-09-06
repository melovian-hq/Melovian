// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach } from "vitest";
import {
  AVATAR_STORAGE_KEY,
  SMILE_VARIANTS_STORAGE_KEY,
  defaultAvatarSeed,
  defaultProfileSettings,
  isAcceptedAvatarFile,
  isAcceptedAvatarType,
  loadProfileSettings,
  readAvatarFromFile,
  saveCustomAvatarUrl,
  saveSmileVariants,
} from "./profile-settings";

describe("profile settings", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("validates accepted avatar types", () => {
    expect(isAcceptedAvatarType("image/jpeg")).toBe(true);
    expect(isAcceptedAvatarType("image/png")).toBe(true);
    expect(isAcceptedAvatarType("image/webp")).toBe(true);
    expect(isAcceptedAvatarType("image/gif")).toBe(false);
  });

  it("validates avatar files by mime type", () => {
    expect(
      isAcceptedAvatarFile(new File(["x"], "a.png", { type: "image/png" })),
    ).toBe(true);
    expect(
      isAcceptedAvatarFile(new File(["x"], "a.gif", { type: "image/gif" })),
    ).toBe(false);
  });

  it("loads and saves custom avatar url", () => {
    expect(loadProfileSettings().customAvatarUrl).toBeNull();
    saveCustomAvatarUrl("data:image/png;base64,abc");
    expect(loadProfileSettings().customAvatarUrl).toBe(
      "data:image/png;base64,abc",
    );
    saveCustomAvatarUrl(null);
    expect(localStorage.getItem(AVATAR_STORAGE_KEY)).toBeNull();
  });

  it("loads and saves smile variants", () => {
    saveSmileVariants({ melovian: 2, "instance-1": 4 });
    expect(loadProfileSettings().smileVariants).toEqual({
      melovian: 2,
      "instance-1": 4,
    });
    expect(localStorage.getItem(SMILE_VARIANTS_STORAGE_KEY)).toContain(
      "instance-1",
    );
  });

  it("ignores invalid stored profile data", () => {
    localStorage.setItem(AVATAR_STORAGE_KEY, "not-a-data-url");
    localStorage.setItem(SMILE_VARIANTS_STORAGE_KEY, "not-json");
    expect(loadProfileSettings()).toEqual(defaultProfileSettings());
  });

  it("derives avatar seed from active instance or server", () => {
    expect(defaultAvatarSeed(null, "My Server")).toBe("My Server");
    expect(
      defaultAvatarSeed({ id: "inst-1", username: "alice" }, "My Server"),
    ).toBe("inst-1");
    expect(defaultAvatarSeed({ username: "alice" }, null)).toBe("alice");
    expect(defaultAvatarSeed(null, null)).toBe("melovian");
  });

  it("rejects unsupported avatar uploads", async () => {
    const file = new File(["gif"], "avatar.gif", { type: "image/gif" });
    await expect(readAvatarFromFile(file)).rejects.toThrow(
      "Avatar must be a JPG, PNG, or WebP image",
    );
  });

  it("rejects oversized avatar uploads", async () => {
    const file = new File([new Uint8Array(513 * 1024)], "avatar.png", {
      type: "image/png",
    });
    await expect(readAvatarFromFile(file)).rejects.toThrow(
      "Avatar must be 512 KB or smaller",
    );
  });
});
