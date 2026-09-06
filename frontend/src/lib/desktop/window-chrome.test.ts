// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import {
  applyWindowChromeDocumentState,
  showCustomWindowControls,
} from "./window-chrome";
import { saveDesktopIntegrationSettings } from "./desktop-integration-settings";

vi.mock("$lib/config/runtime", () => ({
  nativeDesktopAvailable: () => true,
}));

describe("window chrome", () => {
  afterEach(() => {
    localStorage.clear();
    delete document.documentElement.dataset.nativeTitleBar;
    delete document.documentElement.dataset.customWindowChrome;
    document.documentElement.style.removeProperty("--jb-window-controls-inset");
  });

  it("shows custom controls when native title bar is disabled", () => {
    saveDesktopIntegrationSettings({
      taskbarEnabled: true,
      closeBehavior: "ask",
      nativeTitleBar: false,
    });
    expect(showCustomWindowControls()).toBe(true);
  });

  it("hides custom controls when native title bar is enabled", () => {
    saveDesktopIntegrationSettings({
      taskbarEnabled: true,
      closeBehavior: "ask",
      nativeTitleBar: true,
    });
    expect(showCustomWindowControls()).toBe(false);
  });

  it("updates document dataset flags", () => {
    saveDesktopIntegrationSettings({
      taskbarEnabled: true,
      closeBehavior: "ask",
      nativeTitleBar: false,
    });
    applyWindowChromeDocumentState();
    expect(document.documentElement.dataset.customWindowChrome).toBe("true");
    expect(document.documentElement.dataset.nativeTitleBar).toBe("false");
    expect(
      document.documentElement.style.getPropertyValue(
        "--jb-window-controls-inset",
      ),
    ).toBe("var(--jb-window-controls-width, 8.25rem)");
  });
});
