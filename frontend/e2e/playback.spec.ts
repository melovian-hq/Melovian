// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  test,
  expect,
  waitForAppShell,
  settle,
  DEMO_ALBUM_ID,
} from "./fixtures";

test.describe("playback", () => {
  test("playing a demo album shows the player", async ({ page }) => {
    await page.goto(`/music/album/${DEMO_ALBUM_ID}`);
    await waitForAppShell(page);
    await settle(page);

    await expect(
      page.getByRole("button", { name: /^Play$/i }).first(),
    ).toBeVisible({ timeout: 20000 });

    await page
      .getByRole("button", { name: /^Play$/i })
      .first()
      .click();

    const player = page.locator(".player, .mini-player").first();
    await expect(player).toBeVisible({ timeout: 20000 });

    await page.evaluate(() => {
      document.querySelectorAll("audio").forEach((el) => {
        el.muted = true;
        el.volume = 0;
      });
    });

    const pause = page
      .getByRole("button", { name: /^Pause$/i })
      .or(page.locator('button[aria-label="Pause"]'))
      .first();
    await expect(pause).toBeVisible({ timeout: 15000 });
    await pause.click();

    const playControl = page
      .getByRole("button", { name: /^Play$/i })
      .or(page.locator('button[aria-label="Play"]'))
      .first();
    await expect(playControl).toBeVisible({ timeout: 10000 });
  });
});
