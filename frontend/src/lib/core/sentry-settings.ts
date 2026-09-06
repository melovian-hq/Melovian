// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { readAPIError } from "$lib/core/http/errors";

export type SentryEnvLocks = {
  dsn?: boolean;
  frontendDsn?: boolean;
  environment?: boolean;
  release?: boolean;
  tracesSampleRate?: boolean;
};

export type StoredSentrySettings = {
  enabled: boolean;
  dsn?: string;
  frontendDsn?: string;
  environment?: string;
  release?: string;
  tracesSampleRate?: number;
  clientReportingAllowed: boolean;
};

export type EffectiveSentrySettings = {
  enabled: boolean;
  dsnConfigured: boolean;
  frontendDsnConfigured: boolean;
  environment?: string;
  release?: string;
  tracesSampleRate?: number;
  clientReportingAllowed: boolean;
};

export type SentryServerSettingsResponse = {
  stored: StoredSentrySettings;
  effective: EffectiveSentrySettings;
  envLocks: SentryEnvLocks;
};

export type SentryClientSettings = {
  enabled: boolean;
};

export type SentryTestEventResponse = {
  ok: true;
  eventId: string;
  message?: string;
};

export function defaultStoredSentrySettings(): StoredSentrySettings {
  return {
    enabled: false,
    dsn: "",
    frontendDsn: "",
    environment: "",
    release: "",
    tracesSampleRate: 0,
    clientReportingAllowed: true,
  };
}

export function defaultSentryClientSettings(): SentryClientSettings {
  return { enabled: false };
}

export function mergeStoredSentrySettings(
  partial: Partial<StoredSentrySettings> | null | undefined,
): StoredSentrySettings {
  const defaults = defaultStoredSentrySettings();
  if (!partial) return defaults;
  return {
    enabled: partial.enabled ?? defaults.enabled,
    dsn: partial.dsn ?? defaults.dsn,
    frontendDsn: partial.frontendDsn ?? defaults.frontendDsn,
    environment: partial.environment ?? defaults.environment,
    release: partial.release ?? defaults.release,
    tracesSampleRate: partial.tracesSampleRate ?? defaults.tracesSampleRate,
    clientReportingAllowed:
      partial.clientReportingAllowed ?? defaults.clientReportingAllowed,
  };
}

export async function getSentryServerSettings(): Promise<SentryServerSettingsResponse> {
  const response = await fetchWithRetry("/api/settings/sentry", {
    headers: apiHeaders(),
  });
  if (!response.ok) {
    throw new Error(
      `Failed to load error tracking settings (${response.status})`,
    );
  }
  return (await response.json()) as SentryServerSettingsResponse;
}

export async function saveSentryServerSettings(
  settings: StoredSentrySettings,
): Promise<SentryServerSettingsResponse> {
  const response = await fetchWithRetry("/api/settings/sentry", {
    method: "PUT",
    headers: {
      ...apiHeaders(),
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify(settings),
  });
  if (!response.ok) {
    const text = await response.text().catch(() => "");
    throw new Error(
      text || `Failed to save error tracking settings (${response.status})`,
    );
  }
  return (await response.json()) as SentryServerSettingsResponse;
}

export async function getSentryClientSettings(): Promise<SentryClientSettings> {
  const response = await fetchWithRetry("/api/music/settings/sentry-client", {
    headers: apiHeaders(),
  });
  if (!response.ok) {
    throw new Error(
      `Failed to load client error reporting (${response.status})`,
    );
  }
  return (await response.json()) as SentryClientSettings;
}

export async function saveSentryClientSettings(
  settings: SentryClientSettings,
): Promise<SentryClientSettings> {
  const response = await fetchWithRetry("/api/music/settings/sentry-client", {
    method: "PUT",
    headers: {
      ...apiHeaders(),
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify(settings),
  });
  if (!response.ok) {
    const text = await response.text().catch(() => "");
    throw new Error(
      text || `Failed to save client error reporting (${response.status})`,
    );
  }
  return (await response.json()) as SentryClientSettings;
}

export async function sendSentryTestEvent(): Promise<SentryTestEventResponse> {
  const response = await fetchWithRetry("/api/settings/sentry/test", {
    method: "POST",
    headers: {
      ...apiHeaders(),
      Accept: "application/json",
    },
  });
  if (!response.ok) {
    throw new Error(
      await readAPIError(
        response,
        `Failed to send test event (${response.status})`,
      ),
    );
  }

  const payload = (await response.json()) as {
    ok?: boolean;
    eventId?: string;
    message?: string;
    error?: string;
  };
  if (!payload.ok || !payload.eventId) {
    throw new Error(
      payload.error ||
        payload.message ||
        "Test event failed: missing event id",
    );
  }

  return {
    ok: true,
    eventId: payload.eventId,
    message: payload.message,
  };
}
