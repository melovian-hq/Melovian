// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { test, expect, waitForAppShell, settle } from "./fixtures";

async function clickSidebarLink(
  page: import("@playwright/test").Page,
  href: string,
) {
  await page
    .locator(`.app-shell__sidebar a.sidebar__link[href="${href}"]`)
    .first()
    .click();
}

test.describe("navigation", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/music");
    await waitForAppShell(page);
  });

  test("sidebar links reach major pages", async ({ page }) => {
    await clickSidebarLink(page, "/music/albums");
    await settle(page);
    await expect(page).toHaveURL(/\/music\/albums/);
    await expect(
      page.getByRole("heading", { name: "Albums", exact: true }),
    ).toBeVisible();

    await clickSidebarLink(page, "/music/artists");
    await settle(page);
    await expect(page).toHaveURL(/\/music\/artists/);
    await expect(
      page.getByRole("heading", { name: "Artists", exact: true }),
    ).toBeVisible();

    await clickSidebarLink(page, "/music/playlists");
    await settle(page);
    await expect(page).toHaveURL(/\/music\/playlists/);
    await expect(
      page.getByRole("heading", { name: "Playlists", exact: true }),
    ).toBeVisible();

    await clickSidebarLink(page, "/music/search");
    await settle(page);
    await expect(page).toHaveURL(/\/music\/search/);
    await expect(
      page.getByRole("heading", { name: "Search", exact: true }),
    ).toBeVisible();

    await clickSidebarLink(page, "/music");
    await settle(page);
    await expect(page).toHaveURL(/\/music\/?(\?|$)/);
  });
});
