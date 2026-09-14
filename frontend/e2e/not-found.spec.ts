// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { test, expect, waitForAppShell, settle } from "./fixtures";

const BOGUS_PATH = "/definitely/not/a/real-page";

test.describe("not found", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BOGUS_PATH);
    await waitForAppShell(page);
    await settle(page);
  });

  test("unknown route renders the not-found page", async ({ page }) => {
    await expect(
      page.getByRole("heading", { name: "Page not found", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText(`No page exists at ${BOGUS_PATH}.`, { exact: false }),
    ).toBeVisible();
    await expect(page).toHaveTitle(/Page not found/);
  });

  test("Back to music returns to the music home", async ({ page }) => {
    await page.getByRole("link", { name: "Back to music" }).click();
    await settle(page);
    await expect(page).toHaveURL(/\/music\/?(\?|$)/);
  });

  test("Search your library opens search", async ({ page }) => {
    await page.getByRole("link", { name: "Search your library" }).click();
    await settle(page);
    await expect(page).toHaveURL(/\/music\/search/);
    await expect(
      page.getByRole("heading", { name: "Search", exact: true }),
    ).toBeVisible();
  });
});
