// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { router } from "$lib/router/router.svelte";

// Emitted by main.go when a second instance arrives with a
// melovian://install-extension/<id> argument.
export const EXTENSION_INSTALL_EVENT = "melovian:extension:install";

const EXTENSION_ID = /^[a-z0-9][a-z0-9-]{0,62}$/;

export function extensionInstallUrl(id: string): string {
  return `/settings/extensions?install-extension=${encodeURIComponent(id)}`;
}

// Reads the install-extension query param from the current URL. Returns ""
// when absent or malformed.
export function extensionDeepLinkId(search: string): string {
  const id = new URLSearchParams(search).get("install-extension") ?? "";
  return EXTENSION_ID.test(id) ? id : "";
}

// Binds the wails event so a melovian:// deep link lands on the extensions
// settings tab regardless of the current page. Returns an unbind function.
export function bindExtensionDeepLinks(): () => void {
  let off: (() => void) | undefined;
  void import("@wailsio/runtime")
    .then(({ Events }) => {
      off = Events.On(EXTENSION_INSTALL_EVENT, (event) => {
        const data = event.data as { id?: string } | undefined;
        const id = typeof data?.id === "string" ? data.id : "";
        if (EXTENSION_ID.test(id)) {
          router.navigate(extensionInstallUrl(id));
        }
      });
    })
    .catch(() => {});
  return () => off?.();
}
