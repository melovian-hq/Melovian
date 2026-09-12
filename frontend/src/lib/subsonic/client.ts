// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { createDefaultSubsonicConfig, normalizeSubsonicUrl } from "./auth";
import { parsePayload } from "$lib/core/http/parse";
import { subsonicEnvelopeSchema } from "./schemas";
import type { SubsonicConfig } from "./types";

const SUBSONIC_REQUEST_TIMEOUT_MS = 12_000;

export class SubsonicApiError extends Error {
  status: number;
  body: string;

  constructor(message: string, status: number, body: string) {
    super(message);
    this.name = "SubsonicApiError";
    this.status = status;
    this.body = body;
  }
}

export class SubsonicClient {
  constructor(private config: SubsonicConfig = createDefaultSubsonicConfig()) {}

  updateConfig(partial: Partial<SubsonicConfig>) {
    this.config = { ...this.config, ...partial };
  }

  get configSnapshot(): SubsonicConfig {
    return { ...this.config };
  }

  buildUrl(
    endpoint: string,
    query: Record<
      string,
      string | number | boolean | undefined | Array<string | number>
    > = {},
  ): string {
    const base = normalizeSubsonicUrl(this.config.serverUrl);
    const path = endpoint.startsWith("/") ? endpoint : `/rest/${endpoint}`;
    const params = new URLSearchParams();
    params.set("f", "json");
    params.set("v", this.config.version);
    params.set("c", this.config.clientName);

    for (const [key, value] of Object.entries(query)) {
      if (value === undefined || value === "") continue;
      if (Array.isArray(value)) {
        for (const item of value) {
          if (item !== undefined && item !== "") {
            params.append(key, String(item));
          }
        }
        continue;
      }
      params.set(key, String(value));
    }

    return `${base}${path}?${params.toString()}`;
  }

  private async handleResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
      const body = await response.text();
      throw new SubsonicApiError(
        `Subsonic request failed: ${response.status}`,
        response.status,
        body,
      );
    }

    const text = await response.text();
    if (!text) return undefined as T;

    const payload = parsePayload(
      subsonicEnvelopeSchema,
      JSON.parse(text),
      "subsonic response",
    );
    const inner = payload["subsonic-response"];
    if (inner?.status !== "ok") {
      const msg = inner?.error?.message ?? "Subsonic request failed";
      throw new SubsonicApiError(msg, response.status, text);
    }

    return inner as T;
  }

  async request<T>(
    endpoint: string,
    query: Record<
      string,
      string | number | boolean | undefined | Array<string | number>
    > = {},
    options: { method?: "GET" | "POST" } = {},
  ): Promise<T> {
    const controller = new AbortController();
    const timer = setTimeout(
      () => controller.abort(),
      SUBSONIC_REQUEST_TIMEOUT_MS,
    );
    const method = options.method ?? "GET";
    const url = this.buildUrl(endpoint, query);
    try {
      const response = await fetch(url, {
        method,
        credentials: "include",
        signal: controller.signal,
      });
      return await this.handleResponse<T>(response);
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") {
        throw new SubsonicApiError("Subsonic request timed out", 408, "");
      }
      throw err;
    } finally {
      clearTimeout(timer);
    }
  }

  async requestPost<T>(
    endpoint: string,
    query: Record<
      string,
      string | number | boolean | undefined | Array<string | number>
    > = {},
  ): Promise<T> {
    return this.request<T>(endpoint, query, { method: "POST" });
  }
}
