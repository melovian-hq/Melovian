// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { closeApplication } from "$lib/ui/boot-error-screen";

declare global {
  interface Window {
    __melCloseApp?: () => void | Promise<void>;
  }
}

window.__melCloseApp = () => closeApplication();
