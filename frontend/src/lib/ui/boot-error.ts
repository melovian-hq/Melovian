// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { formatError, type FatalErrorDetails } from "$lib/ui/boot-error-format";
import { mapClientError } from "$lib/ui/client-error";
import { logClientError, logger } from "$lib/core/logger";
import { toast } from "$lib/ui/toast.svelte";

export type { FatalErrorDetails };

function appMounted(): boolean {
  return document.getElementById("app")?.dataset.mounted === "true";
}

export function reportFatalError(error: unknown, source?: string) {
  logClientError(error, source);
  if (appMounted()) return;
  window.__melReportError?.(error, source);
}

export function logRuntimeError(error: unknown, source?: string) {
  logClientError(error, source);
}

export function markAppMounted() {
  window.__melMarkMounted?.();
}

export function installGlobalErrorHandlers() {
  if (typeof window === "undefined") return;

  window.addEventListener("error", (event) => {
    logger.error(
      "Uncaught error",
      event.error ?? event.message,
      { filename: event.filename, lineno: event.lineno, colno: event.colno },
      "window.error",
    );
    if (appMounted()) {
      event.preventDefault();
      const message = mapClientError(event.error ?? event.message).message;
      toast.error(message);
      return;
    }
    window.__melReportError?.(event.error || event.message, "window.error");
  });

  window.addEventListener("unhandledrejection", (event) => {
    logger.error(
      "Unhandled promise rejection",
      event.reason,
      undefined,
      "unhandledrejection",
    );
    if (appMounted()) {
      event.preventDefault();
      const message = mapClientError(event.reason).message;
      toast.error(message);
      return;
    }
    window.__melReportError?.(event.reason, "unhandledrejection");
  });

  logger.info("frontend logging ready");
}

declare global {
  interface Window {
    __melMarkMounted?: () => void;
    __melReportError?: (error: unknown, source?: string) => void;
    __melCloseApp?: () => void | Promise<void>;
  }
}

export { formatError };
