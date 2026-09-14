// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { initSentryFromBuildEnv } from "$lib/core/sentry";
import { registerServiceWorker } from "$lib/pwa/register-sw";
import {
  markAppMounted,
  reportFatalError,
  installGlobalErrorHandlers,
} from "$lib/ui/boot-error";
import { installInsecureContextPolyfills } from "$lib/utils/uuid";
import { mount } from "svelte";
import "./app.css";
import App from "./App.svelte";
import "$lib/theme/theme.svelte";
import { setStaticDemo } from "$lib/config/runtime";

installInsecureContextPolyfills();
initSentryFromBuildEnv();
installGlobalErrorHandlers();

// PWA shell + auto-update. The reload is deferred while audio is playing so
// an update can never cut a track mid-stream.
void import("$lib/config/music.svelte")
  .then(({ music }) => {
    registerServiceWorker({
      isPlaying: () => music.playing,
      onUpdateReady: () => {
        // Applied automatically once playback stops, or on next launch
      },
    });
  })
  .catch(() => registerServiceWorker());

async function boot() {
  if (import.meta.env.VITE_STATIC_DEMO === "true") {
    setStaticDemo(true);
    const { installStaticDemoApi } = await import("$lib/demo/static-api");
    await installStaticDemoApi();
  }

  const target = document.getElementById("app");
  if (!target) {
    throw new Error("Missing #app mount target");
  }

  mount(App, { target });
  markAppMounted();
}

boot().catch((error) => {
  reportFatalError(error, "main");
});
