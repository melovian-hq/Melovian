// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";
import { trackRuleSchema } from "./schemas";
import type { TrackRule } from "./types";

export type SandboxRunResult = {
  rules: TrackRule[];
  error?: string;
};

export type SandboxOptions = {
  timeoutMs?: number;
  iframeSrc?: string;
};

const DEFAULT_TIMEOUT_MS = 5_000;

function parseRules(raw: unknown): TrackRule[] {
  if (!Array.isArray(raw)) return [];
  const rules: TrackRule[] = [];
  for (const candidate of raw) {
    const parsed = v.safeParse(trackRuleSchema, candidate);
    if (parsed.success) rules.push(parsed.output as TrackRule);
  }
  return rules;
}

/**
 * Run an extension script inside a sandboxed iframe loaded from
 * ext-sandbox.html. The iframe uses sandbox="allow-scripts" without
 * allow-same-origin, so the script runs in an opaque origin: no
 * localStorage, no cookies, no parent DOM, and credentialed fetches
 * to the app API fail CORS. Escaping the Function wrapper only
 * lands the code in the null origin.
 */
export function runExtensionScript(
  source: string,
  settings: Record<string, unknown>,
  seedRules: TrackRule[] = [],
  options: SandboxOptions = {},
): Promise<SandboxRunResult> {
  const timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  const iframeSrc = options.iframeSrc ?? "/ext-sandbox.html";

  return new Promise((resolve) => {
    if (typeof document === "undefined") {
      resolve({ rules: [], error: "extension sandbox unavailable" });
      return;
    }

    const iframe = document.createElement("iframe");
    iframe.setAttribute("sandbox", "allow-scripts");
    iframe.setAttribute("aria-hidden", "true");
    iframe.style.display = "none";

    const finish = (result: SandboxRunResult) => {
      cleanup();
      resolve(result);
    };

    const timer = setTimeout(
      () => finish({ rules: [], error: "extension script timed out" }),
      timeoutMs,
    );

    const onMessage = (event: MessageEvent) => {
      if (event.source !== iframe.contentWindow) return;
      const data = event.data as {
        type?: string;
        rules?: unknown;
        error?: unknown;
      };
      if (!data || typeof data !== "object") return;
      if (data.type === "ready") {
        iframe.contentWindow?.postMessage(
          { type: "run", source, settings, rules: seedRules },
          "*",
        );
        return;
      }
      if (data.type === "done") {
        const error =
          typeof data.error === "string" && data.error ? data.error : undefined;
        finish({ rules: parseRules(data.rules), error });
      }
    };

    const cleanup = () => {
      clearTimeout(timer);
      window.removeEventListener("message", onMessage);
      iframe.remove();
    };

    window.addEventListener("message", onMessage);
    iframe.src = iframeSrc;
    document.body.appendChild(iframe);
  });
}
