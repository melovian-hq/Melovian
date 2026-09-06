// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  appliedEnvSummary,
  defaultGraphicsSettings,
  loadGraphicsEnvironment,
  mergeGraphicsSettings,
  nvExplicitSyncLabel,
  saveGraphicsSettings,
} from "./graphics-settings";

const GetGraphicsEnvironment = vi.fn();
const SaveGraphicsSettings = vi.fn().mockResolvedValue(undefined);

vi.mock("@bindings/melovian/services/index.js", () => ({
  MediaService: {
    GetGraphicsEnvironment,
    SaveGraphicsSettings,
  },
}));

describe("graphics-settings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns linux-safe defaults", () => {
    expect(defaultGraphicsSettings()).toEqual({
      disableDmabufRenderer: true,
      disableCompositingMode: false,
      nvDisableExplicitSync: "auto",
    });
  });

  it("keeps linux defaults when partial settings are empty", () => {
    expect(mergeGraphicsSettings({})).toEqual(defaultGraphicsSettings());
  });

  it("merges explicit overrides", () => {
    expect(
      mergeGraphicsSettings({
        disableDmabufRenderer: false,
        disableCompositingMode: true,
        nvDisableExplicitSync: "off",
      }),
    ).toEqual({
      disableDmabufRenderer: false,
      disableCompositingMode: true,
      nvDisableExplicitSync: "off",
    });
  });

  it("rejects invalid nv explicit sync values", () => {
    expect(
      mergeGraphicsSettings({
        nvDisableExplicitSync: "maybe" as "auto",
      }),
    ).toEqual(defaultGraphicsSettings());
  });

  it("returns null when graphics settings are unsupported", async () => {
    GetGraphicsEnvironment.mockResolvedValue({ supported: false });

    await expect(loadGraphicsEnvironment()).resolves.toBeNull();
  });

  it("merges backend settings and defaults when supported", async () => {
    GetGraphicsEnvironment.mockResolvedValue({
      supported: true,
      platform: "linux",
      wayland: true,
      nvidia: false,
      settings: { nvDisableExplicitSync: "invalid" },
      defaults: { disableDmabufRenderer: false },
      appliedEnv: { WEBKIT_DISABLE_DMABUF_RENDERER: "1" },
    });

    await expect(loadGraphicsEnvironment()).resolves.toEqual({
      supported: true,
      platform: "linux",
      wayland: true,
      nvidia: false,
      settings: {
        disableDmabufRenderer: true,
        disableCompositingMode: false,
        nvDisableExplicitSync: "auto",
      },
      defaults: {
        disableDmabufRenderer: false,
        disableCompositingMode: false,
        nvDisableExplicitSync: "auto",
      },
      appliedEnv: { WEBKIT_DISABLE_DMABUF_RENDERER: "1" },
    });
  });

  it("normalizes settings before saving", async () => {
    await saveGraphicsSettings({
      disableDmabufRenderer: false,
      disableCompositingMode: true,
      nvDisableExplicitSync: "on",
    });

    expect(SaveGraphicsSettings).toHaveBeenCalledWith({
      disableDmabufRenderer: false,
      disableCompositingMode: true,
      nvDisableExplicitSync: "on",
    });
  });

  it("formats nv explicit sync labels for the settings UI", () => {
    expect(nvExplicitSyncLabel("on")).toBe("On");
    expect(nvExplicitSyncLabel("off")).toBe("Off");
    expect(nvExplicitSyncLabel("auto")).toBe("Auto (Wayland + NVIDIA)");
  });

  it("summarizes applied environment variables", () => {
    expect(appliedEnvSummary({})).toBe(
      "No workarounds active in this session.",
    );
    expect(
      appliedEnvSummary({
        WEBKIT_DISABLE_DMABUF_RENDERER: "1",
        __NV_DISABLE_EXPLICIT_SYNC: "1",
      }),
    ).toBe("WEBKIT_DISABLE_DMABUF_RENDERER=1, __NV_DISABLE_EXPLICIT_SYNC=1");
  });
});
