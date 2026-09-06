// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as Sentry from "@sentry/svelte";

export type SentryRuntimeConfig = {
  dsn?: string;
  environment?: string;
  release?: string;
  tracesSampleRate?: number;
  clientReporting?: boolean;
};

let activeDSN = "";
let clientReportingEnabled = false;

export function resetSentryForTests(): void {
  activeDSN = "";
  clientReportingEnabled = false;
}

function buildOptions(cfg: SentryRuntimeConfig): Sentry.BrowserOptions | null {
  const dsn = cfg.dsn?.trim() ?? "";
  if (!dsn || !cfg.clientReporting) return null;
  return {
    dsn,
    environment: cfg.environment?.trim() || undefined,
    release: cfg.release?.trim() || undefined,
    tracesSampleRate: cfg.tracesSampleRate ?? 0,
  };
}

function initOptions(opts: Sentry.BrowserOptions, reporting: boolean): void {
  const dsn = opts.dsn ?? "";
  clientReportingEnabled = reporting && dsn !== "";
  if (!clientReportingEnabled) {
    activeDSN = "";
    return;
  }
  if (activeDSN === dsn) return;
  Sentry.init(opts);
  activeDSN = dsn;
}

export function initSentryFromBuildEnv(): void {
  const dsn = import.meta.env.VITE_SENTRY_DSN?.trim();
  if (!dsn) return;
  const tracesRaw = import.meta.env.VITE_SENTRY_TRACES_SAMPLE_RATE;
  const tracesSampleRate =
    tracesRaw === undefined || tracesRaw === "" ? 0 : Number(tracesRaw);
  const opts = buildOptions({
    dsn,
    environment: import.meta.env.VITE_SENTRY_ENVIRONMENT,
    release: import.meta.env.VITE_SENTRY_RELEASE,
    tracesSampleRate: Number.isFinite(tracesSampleRate) ? tracesSampleRate : 0,
    clientReporting: true,
  });
  if (!opts) return;
  initOptions(opts, true);
}

export function applyRuntimeSentryConfig(cfg?: SentryRuntimeConfig): void {
  const reporting = cfg?.clientReporting === true;
  const opts = buildOptions({ ...cfg, clientReporting: reporting });
  if (!opts) {
    disableClientSentry();
    return;
  }
  initOptions(opts, reporting);
}

export function disableClientSentry(): void {
  clientReportingEnabled = false;
  activeDSN = "";
  try {
    void Sentry.close(2000);
  } catch {
    // close is best-effort when the SDK was never initialized
  }
}

export function sentryEnabled(): boolean {
  return clientReportingEnabled && activeDSN !== "";
}

export function captureClientError(error: unknown, source?: string): void {
  if (!sentryEnabled()) return;
  Sentry.withScope((scope) => {
    if (source) {
      scope.setTag("source", source);
    }
    if (error instanceof Error) {
      Sentry.captureException(error);
      return;
    }
    if (typeof error === "string") {
      Sentry.captureMessage(error);
      return;
    }
    scope.setExtra("value", error);
    Sentry.captureMessage(String(error ?? "Unknown client error"));
  });
}

export function captureClientMessage(
  message: string,
  level: Sentry.SeverityLevel = "error",
  source?: string,
): void {
  if (!sentryEnabled()) return;
  Sentry.withScope((scope) => {
    if (source) {
      scope.setTag("source", source);
    }
    scope.setLevel(level);
    Sentry.captureMessage(message);
  });
}
