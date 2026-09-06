// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { test as base, expect, type Page } from "@playwright/test";

/** Known demo catalog album (Bad Bunny / Debi Tirar Mas Fotos). */
export const DEMO_ALBUM_ID = "al-006-01";
export const DEMO_SEARCH_QUERY = "Bad Bunny";

async function muteAudio(page: Page): Promise<void> {
  await page.evaluate(() => {
    document.querySelectorAll("audio").forEach((el) => {
      el.muted = true;
      el.volume = 0;
    });
  });
}

export const test = base.extend({
  context: async ({ context }, use) => {
    await context.addInitScript(() => {
      localStorage.setItem("melovian-theme", "dark");
      localStorage.setItem("mel-home-tips-dismissed", "1");
    });
    await use(context);
  },
  page: async ({ page }, use) => {
    page.on("load", () => {
      void muteAudio(page);
    });
    await use(page);
  },
});

export { expect };

/** Wait until the app shell nav is visible (library chrome ready). */
export async function waitForAppShell(page: Page): Promise<void> {
  await expect(
    page.getByRole("navigation", { name: "Main navigation" }),
  ).toBeVisible({ timeout: 30000 });
}

/** Soft settle after client-side navigation. */
export async function settle(page: Page): Promise<void> {
  await page.waitForLoadState("domcontentloaded");
  await muteAudio(page);
}
