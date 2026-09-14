// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  test,
  expect,
  waitForAppShell,
  settle,
  DEMO_ALBUM_ID,
} from "./fixtures";

test.describe("demo mode", () => {
  test("demo banner announces the sample library", async ({ page }) => {
    await page.goto("/music");
    await waitForAppShell(page);
    await settle(page);

    const banner = page.locator(".demo-banner");
    await expect(banner).toBeVisible();
    await expect(banner).toContainText("Demo mode");
    await expect(banner).toContainText("changes are not saved");
  });

  test("demo catalog album opens from the albums grid", async ({ page }) => {
    await page.goto("/music/albums");
    await waitForAppShell(page);
    await settle(page);

    const albumLink = page
      .locator(`a[href="/music/album/${DEMO_ALBUM_ID}"]`)
      .first();
    await expect(albumLink).toBeVisible({ timeout: 30000 });
    await albumLink.click();
    await settle(page);

    await expect(page).toHaveURL(
      new RegExp(`/music/album/${DEMO_ALBUM_ID}($|\\?)`),
    );
    await expect(
      page.getByRole("heading", { name: "Debí Tirar Más Fotos" }),
    ).toBeVisible({ timeout: 20000 });
  });

  test("sidebar hides settings and instance switcher in demo mode", async ({
    page,
  }) => {
    await page.goto("/music");
    await waitForAppShell(page);
    await settle(page);

    const sidebar = page.getByRole("navigation", { name: "Main navigation" });
    await expect(sidebar.getByRole("link", { name: "Settings" })).toBeHidden();
    await expect(sidebar.locator(".sidebar__instance")).toHaveCount(0);
  });
});
