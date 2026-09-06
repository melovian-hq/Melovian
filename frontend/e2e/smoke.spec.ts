// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { test, expect, waitForAppShell, settle } from "./fixtures";

test.describe("smoke @smoke", () => {
  test("home loads with main navigation", async ({ page }) => {
    await page.goto("/music");
    await waitForAppShell(page);
    await settle(page);

    await expect(
      page.getByRole("link", { name: "Melovian home" }),
    ).toBeVisible();
    await expect(
      page.getByRole("navigation", { name: "Main navigation" }),
    ).toBeVisible();

    const shellHtml = await page.locator("body").innerHTML();
    expect(shellHtml.length).toBeGreaterThan(200);
  });
});
