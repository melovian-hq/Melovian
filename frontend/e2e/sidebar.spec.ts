// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { test, expect, waitForAppShell, settle } from "./fixtures";

const COLLAPSED_KEY = "melovian-sidebar-collapsed";

test.describe("sidebar collapse", () => {
  test("collapse hides nav labels and persists across reload", async ({
    page,
  }) => {
    await page.goto("/music");
    await waitForAppShell(page);
    await settle(page);

    const sidebar = page.getByRole("navigation", { name: "Main navigation" });
    const firstLabel = sidebar.locator(".sidebar__label").first();
    await expect(firstLabel).toBeVisible();

    await page.getByRole("button", { name: "Collapse sidebar" }).click();

    await expect(sidebar).toHaveClass(/sidebar--collapsed/);
    await expect(firstLabel).toBeHidden();
    await expect(
      page.getByRole("button", { name: "Expand sidebar" }),
    ).toBeVisible();
    expect(
      await page.evaluate((key) => localStorage.getItem(key), COLLAPSED_KEY),
    ).toBe("true");

    await page.reload();
    await waitForAppShell(page);
    await settle(page);

    const sidebarAfter = page.getByRole("navigation", {
      name: "Main navigation",
    });
    await expect(sidebarAfter).toHaveClass(/sidebar--collapsed/);
    await expect(sidebarAfter.locator(".sidebar__label").first()).toBeHidden();

    await page.getByRole("button", { name: "Expand sidebar" }).click();

    await expect(sidebarAfter).not.toHaveClass(/sidebar--collapsed/);
    await expect(sidebarAfter.locator(".sidebar__label").first()).toBeVisible();
    expect(
      await page.evaluate((key) => localStorage.getItem(key), COLLAPSED_KEY),
    ).toBe("false");
  });
});
