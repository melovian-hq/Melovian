// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from "vitest";
import { flushSync, mount, unmount } from "svelte";
import ExtensionsSettings from "./ExtensionsSettings.svelte";
import {
  fetchExtensions,
  fetchExtensionRegistry,
  type ExtensionListItem,
} from "$lib/extensions/api";

vi.mock("$lib/extensions/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("$lib/extensions/api")>();
  return {
    ...actual,
    fetchExtensions: vi.fn(),
    fetchExtensionRegistry: vi.fn(),
    installExtension: vi.fn(),
    installExtensionDir: vi.fn(),
    installRemoteExtension: vi.fn(),
    reinstallExtension: vi.fn(),
    saveExtensionSettings: vi.fn(),
    setExtensionEnabled: vi.fn(),
    uninstallExtension: vi.fn(),
    saveRegistryConfig: vi.fn(),
    clearRegistryConfig: vi.fn(),
  };
});

vi.mock("$lib/extensions/registry", () => ({
  loadExtensions: vi.fn().mockResolvedValue(undefined),
}));

const fetchExtensionsMock = vi.mocked(fetchExtensions);
const fetchRegistryMock = vi.mocked(fetchExtensionRegistry);

const itemWithSettings: ExtensionListItem = {
  id: "visualizations",
  name: "Visualizations",
  version: "0.1.0",
  enabled: true,
  installed: true,
  bundled: false,
  hasScript: false,
  scriptSafe: false,
  hasWasm: false,
  settings: {},
};

function renderPage() {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(ExtensionsSettings, { target });
  flushSync();
  return {
    target,
    cleanup: () => {
      unmount(instance);
      target.remove();
    },
  };
}

describe("ExtensionsSettings", () => {
  beforeEach(() => {
    fetchExtensionsMock.mockReset().mockResolvedValue({
      items: [itemWithSettings],
      dir: "/data/extensions",
      manifests: [
        {
          id: "visualizations",
          name: "Visualizations",
          version: "0.1.0",
          settings: [
            {
              key: "preset",
              type: "choice",
              label: "Preset",
              options: ["bars", "wave"],
              default: "bars",
            },
            {
              key: "overlay",
              type: "boolean",
              label: "Overlay",
              default: false,
            },
          ],
        },
      ],
    });
    fetchRegistryMock.mockReset().mockResolvedValue({
      items: [],
      signed: false,
      custom: false,
      url: "",
      generatedAt: "",
    });
  });

  it("renders settings fields for installed extensions without state_unsafe_mutation", async () => {
    const errSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const unhandled: unknown[] = [];
    const onRejection = (e: PromiseRejectionEvent) => unhandled.push(e.reason);
    window.addEventListener("unhandledrejection", onRejection);
    const { target, cleanup } = renderPage();
    await vi.waitFor(() => {
      expect(target.textContent).toContain("Preset");
    });
    await flushSync();
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(target.textContent).toContain("Visualizations");
    expect(target.textContent).toContain("Overlay");
    const logged = errSpy.mock.calls.flat().join(" ");
    const rejected = unhandled.map(String).join(" ");
    expect(logged + rejected).not.toContain("state_unsafe_mutation");
    window.removeEventListener("unhandledrejection", onRejection);
    cleanup();
  });
});
