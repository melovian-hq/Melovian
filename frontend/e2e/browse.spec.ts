// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { test, expect, waitForAppShell, settle } from "./fixtures";

test.describe("browse", () => {
  test("albums list opens an album with Play", async ({ page }) => {
    await page.goto("/music/albums");
    await waitForAppShell(page);
    await settle(page);

    await expect(
      page.getByRole("heading", { name: "Albums", exact: true }),
    ).toBeVisible();

    const albumLink = page.locator('a[href^="/music/album/"]').first();
    await expect(albumLink).toBeVisible({ timeout: 30000 });
    await albumLink.click();
    await settle(page);

    await expect(page).toHaveURL(/\/music\/album\//);
    await expect(
      page.getByRole("button", { name: /^Play$/i }).first(),
    ).toBeVisible({ timeout: 20000 });
  });
});
