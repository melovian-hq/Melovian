// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { APP_NAME } from "$lib/brand";

export interface BootErrorPayload {
  title: string;
  message: string;
  source?: string;
  stack?: string;
  url?: string;
  time?: string;
}

export function buildBootErrorDetail(payload: BootErrorPayload): string {
  const url =
    payload.url ?? (typeof location !== "undefined" ? location.href : "");
  const time = payload.time ?? new Date().toISOString();

  return [
    payload.source ? `Source: ${payload.source}` : "",
    url ? `URL: ${url}` : "",
    `Time: ${time}`,
    payload.stack ? `\n${payload.stack}` : "",
  ]
    .filter(Boolean)
    .join("\n");
}

export function buildBootErrorLog(
  title: string,
  message: string,
  detail: string,
): string {
  return [title, message, detail].filter(Boolean).join("\n\n");
}

export function bootErrorPayloadFromUnknown(
  error: unknown,
  source?: string,
): BootErrorPayload {
  const message =
    error instanceof Error
      ? error.message || "Unknown error"
      : typeof error === "string"
        ? error
        : "Unknown error";
  const stack = error instanceof Error ? error.stack : undefined;

  return {
    title: `${APP_NAME} crashed`,
    message,
    source,
    stack,
  };
}

export async function copyTextToClipboard(text: string): Promise<boolean> {
  if (!text) return false;

  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return true;
    }
  } catch {
    // fall through to legacy copy
  }

  try {
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.setAttribute("readonly", "");
    textarea.style.position = "fixed";
    textarea.style.left = "-9999px";
    document.body.appendChild(textarea);
    textarea.select();
    const copied = document.execCommand("copy");
    document.body.removeChild(textarea);
    return copied;
  } catch {
    return false;
  }
}

export function isNativeDesktopRuntime(): boolean {
  if (typeof window === "undefined") return false;
  const w = window as Window & {
    chrome?: { webview?: { postMessage?: (message: unknown) => void } };
    webkit?: {
      messageHandlers?: {
        external?: { postMessage?: (message: unknown) => void };
      };
    };
    wails?: { invokeAsync?: (id: string, payload: string) => void };
  };
  if (window.location?.protocol === "wails:") return true;
  if (w.chrome?.webview?.postMessage) return true;
  if (w.webkit?.messageHandlers?.external?.postMessage) return true;
  if (typeof w.wails?.invokeAsync === "function") return true;
  return false;
}

/** MediaService.QuitApp binding id from generated Wails bindings. */
export const QUIT_APP_CALL_ID = 4076836223;

export async function closeApplication(): Promise<void> {
  if (isNativeDesktopRuntime()) {
    try {
      const { Call } = await import("@wailsio/runtime");
      await Call.ByID(QUIT_APP_CALL_ID);
      return;
    } catch {
      // fall through to window.close()
    }
  }

  window.close();
}
