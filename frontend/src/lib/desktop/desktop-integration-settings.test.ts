// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  defaultDesktopIntegrationSettings,
  mergeDesktopIntegrationSettings,
  resolveCloseAction,
} from "./desktop-integration-settings";

describe("desktop integration settings", () => {
  it("merges partial settings with defaults", () => {
    expect(mergeDesktopIntegrationSettings({ taskbarEnabled: false })).toEqual({
      taskbarEnabled: false,
      closeBehavior: "ask",
      nativeTitleBar: false,
    });
  });

  it("rejects invalid close behavior values", () => {
    expect(
      mergeDesktopIntegrationSettings({
        closeBehavior: "invalid" as "ask",
      }),
    ).toEqual(defaultDesktopIntegrationSettings());
  });

  it("maps background close behavior based on tray setting", () => {
    expect(
      resolveCloseAction({
        taskbarEnabled: true,
        closeBehavior: "background",
        nativeTitleBar: false,
      }),
    ).toBe("background");
    expect(
      resolveCloseAction({
        taskbarEnabled: false,
        closeBehavior: "background",
        nativeTitleBar: false,
      }),
    ).toBe("minimize");
  });
});
