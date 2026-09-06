// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type {
  GraphicsEnvironment,
  GraphicsSettings,
} from "@bindings/melovian/services/models.js";

export type { GraphicsEnvironment, GraphicsSettings };
export type NvExplicitSyncMode = "auto" | "on" | "off";

export function defaultGraphicsSettings(): GraphicsSettings {
  return {
    disableDmabufRenderer: true,
    disableCompositingMode: false,
    nvDisableExplicitSync: "auto",
  };
}

export function mergeGraphicsSettings(
  partial: Partial<GraphicsSettings> | null | undefined,
): GraphicsSettings {
  const defaults = defaultGraphicsSettings();
  if (!partial) return defaults;
  const nvMode =
    partial.nvDisableExplicitSync === "on" ||
    partial.nvDisableExplicitSync === "off" ||
    partial.nvDisableExplicitSync === "auto"
      ? partial.nvDisableExplicitSync
      : defaults.nvDisableExplicitSync;
  return {
    disableDmabufRenderer:
      typeof partial.disableDmabufRenderer === "boolean"
        ? partial.disableDmabufRenderer
        : defaults.disableDmabufRenderer,
    disableCompositingMode:
      typeof partial.disableCompositingMode === "boolean"
        ? partial.disableCompositingMode
        : defaults.disableCompositingMode,
    nvDisableExplicitSync: nvMode,
  };
}

export async function loadGraphicsEnvironment(): Promise<GraphicsEnvironment | null> {
  const { MediaService } = await import("@bindings/melovian/services/index.js");
  const env = await MediaService.GetGraphicsEnvironment();
  if (!env?.supported) return null;
  return {
    ...env,
    settings: mergeGraphicsSettings(env.settings),
    defaults: mergeGraphicsSettings(env.defaults),
    appliedEnv: env.appliedEnv ?? {},
  };
}

export async function saveGraphicsSettings(
  settings: GraphicsSettings,
): Promise<void> {
  const { MediaService } = await import("@bindings/melovian/services/index.js");
  await MediaService.SaveGraphicsSettings(mergeGraphicsSettings(settings));
}

export function nvExplicitSyncLabel(mode: NvExplicitSyncMode): string {
  switch (mode) {
    case "on":
      return "On";
    case "off":
      return "Off";
    default:
      return "Auto (Wayland + NVIDIA)";
  }
}

export function appliedEnvSummary(
  applied: Record<string, string | undefined>,
): string {
  const entries = Object.entries(applied).filter(
    (entry): entry is [string, string] => typeof entry[1] === "string",
  );
  if (entries.length === 0) return "No workarounds active in this session.";
  return entries.map(([key, value]) => `${key}=${value}`).join(", ");
}
