// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { RouteContentLayout } from "./router.svelte";

/**
 * Layout of the page the outlet has actually committed. The route match
 * flips the moment navigation starts, but lazy pages keep the previous
 * view mounted while their chunk loads. Shell chrome keys off this state
 * so compact/fill padding only changes when the new page is ready, which
 * is what stops the visible jump before the crossfade starts.
 */
export const outletState = $state<{
  content: RouteContentLayout;
  bare: boolean;
}>({ content: "default", bare: false });
