// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Telemetry consent state. The backend persists the per-user choice under
// PrefKeySentryClient. localStorage mirrors the answer so boot code can gate
// any build-embedded Sentry init before the API round-trip finishes.

import { StorageKeys } from "$lib/brand";
import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { ApiPaths } from "$lib/core/http/api-paths";
import { parseJson } from "$lib/core/http/parse";
import { runtimeConfigSchema } from "$lib/config/schemas";
import {
  applyRuntimeSentryConfig,
  disableClientSentry,
} from "$lib/core/sentry";
import {
  getSentryClientSettings,
  saveSentryClientSettings,
  type SentryConsentChoice,
} from "$lib/core/sentry-settings";
import { isStaticDemo } from "$lib/config/runtime";

let loaded = $state(false);
let choice = $state<SentryConsentChoice>("unset");
let promptOpen = $state(false);
let saving = $state(false);

function cacheChoice(value: SentryConsentChoice): void {
  try {
    if (value === "unset") {
      localStorage.removeItem(StorageKeys.telemetryConsent);
    } else {
      localStorage.setItem(StorageKeys.telemetryConsent, value);
    }
  } catch {
    // Storage can be unavailable in private contexts
  }
}

function normalizeChoice(
  raw: string | undefined,
  enabled: boolean,
): SentryConsentChoice {
  if (raw === "accepted" || raw === "declined") return raw;
  // Legacy installs that enabled reporting before the prompt existed keep it
  if (enabled) return "accepted";
  return "unset";
}

async function refreshSentryRuntime(): Promise<void> {
  try {
    const response = await fetchWithRetry(ApiPaths.config, {
      headers: apiHeaders(),
    });
    if (!response.ok) {
      disableClientSentry();
      return;
    }
    const cfg = await parseJson(
      runtimeConfigSchema,
      response,
      "runtime config",
    );
    if (cfg.sentry?.clientReporting) {
      applyRuntimeSentryConfig(cfg.sentry);
    } else {
      disableClientSentry();
    }
  } catch {
    disableClientSentry();
  }
}

export const telemetryConsent = {
  get loaded() {
    return loaded;
  },
  get choice() {
    return choice;
  },
  get promptOpen() {
    return promptOpen;
  },
  get saving() {
    return saving;
  },

  /** Load the stored choice and open the modal when the user never answered. */
  async init(): Promise<void> {
    if (loaded) return;
    loaded = true;
    if (isStaticDemo()) {
      choice = "unset";
      promptOpen = false;
      return;
    }
    try {
      const settings = await getSentryClientSettings();
      choice = normalizeChoice(settings.choice, settings.enabled);
    } catch {
      choice = "unset";
    }
    cacheChoice(choice);
    promptOpen = choice === "unset";
  },

  /** Reopen the prompt from settings without changing the stored choice. */
  reopenPrompt(): void {
    promptOpen = true;
  },

  async answer(next: "accepted" | "declined"): Promise<void> {
    if (saving) return;
    saving = true;
    try {
      await saveSentryClientSettings({
        enabled: next === "accepted",
        choice: next,
      });
    } catch {
      // The choice is opt-out by default. A failed save still honors the
      // local answer so no events leave this device.
    } finally {
      saving = false;
    }
    choice = next;
    cacheChoice(choice);
    promptOpen = false;
    if (next === "accepted") {
      await refreshSentryRuntime();
    } else {
      disableClientSentry();
    }
  },
};
