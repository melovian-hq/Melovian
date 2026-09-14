// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { test, expect, waitForAppShell, settle } from "./fixtures";

const THEME_KEY = "melovian-theme";

async function storedTheme(
  page: import("@playwright/test").Page,
): Promise<string | null> {
  return page.evaluate((key) => localStorage.getItem(key), THEME_KEY);
}

test.describe("theme", () => {
  test("toggle switches data-theme and writes the storage key", async ({
    page,
  }) => {
    await page.goto("/music");
    await waitForAppShell(page);
    await settle(page);

    const html = page.locator("html");
    await expect(html).toHaveAttribute("data-theme", "dark");

    const group = page.getByRole("group", { name: "Theme" });
    const lightBtn = group.getByRole("button", { name: "Light" });
    const darkBtn = group.getByRole("button", { name: "Dark" });
    const systemBtn = group.getByRole("button", { name: "System" });

    await lightBtn.click();
    await expect(html).toHaveAttribute("data-theme", "light");
    await expect(lightBtn).toHaveAttribute("aria-pressed", "true");
    expect(await storedTheme(page)).toBe("light");

    await darkBtn.click();
    await expect(html).toHaveAttribute("data-theme", "dark");
    await expect(darkBtn).toHaveAttribute("aria-pressed", "true");
    expect(await storedTheme(page)).toBe("dark");

    // System resolves against prefers-color-scheme, which the config pins
    // to dark.
    await systemBtn.click();
    await expect(html).toHaveAttribute("data-theme", "dark");
    expect(await storedTheme(page)).toBe("system");
  });

  test("theme query param overrides stored mode on load", async ({ page }) => {
    // The shared fixture stores "dark" in localStorage. The URL param wins.
    await page.goto("/music?theme=light");
    await waitForAppShell(page);
    await settle(page);

    await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
    expect(await storedTheme(page)).toBe("light");
  });

  test("theme choice persists across reload", async ({ browser, baseURL }) => {
    // The shared context fixture rewrites melovian-theme on every load, so
    // reload persistence needs a context without that init script.
    const context = await browser.newContext({
      baseURL,
      colorScheme: "dark",
    });
    try {
      const page = await context.newPage();
      await page.goto("/music");
      await waitForAppShell(page);
      await settle(page);

      await page
        .getByRole("group", { name: "Theme" })
        .getByRole("button", { name: "Light" })
        .click();
      await expect(page.locator("html")).toHaveAttribute("data-theme", "light");

      await page.reload();
      await waitForAppShell(page);
      await settle(page);

      await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
      expect(await storedTheme(page)).toBe("light");
    } finally {
      await context.close();
    }
  });
});
