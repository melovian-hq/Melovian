// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  test,
  expect,
  waitForAppShell,
  settle,
  DEMO_SEARCH_QUERY,
} from "./fixtures";

test.describe("search", () => {
  test("search returns demo catalog results", async ({ page }) => {
    await page.goto("/music/search");
    await waitForAppShell(page);
    await settle(page);

    await expect(
      page.getByRole("heading", { name: "Search", exact: true }),
    ).toBeVisible();

    const input = page.getByPlaceholder("Search your library");
    await expect(input).toBeVisible();
    await input.fill(DEMO_SEARCH_QUERY);

    const results = page.locator(".search-results");
    await expect(results).toBeVisible({ timeout: 20000 });
    await expect(results).not.toHaveClass(/search-results--pending/);

    const hit = results.getByText(DEMO_SEARCH_QUERY, { exact: false }).first();
    await expect(hit).toBeVisible({ timeout: 20000 });
  });
});
