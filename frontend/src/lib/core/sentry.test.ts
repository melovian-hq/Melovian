// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from "vitest";
import * as Sentry from "@sentry/svelte";
import {
  applyRuntimeSentryConfig,
  captureClientError,
  initSentryFromBuildEnv,
  resetSentryForTests,
  sentryEnabled,
} from "./sentry";

vi.mock("@sentry/svelte", () => ({
  init: vi.fn(),
  close: vi.fn().mockResolvedValue(true),
  makeFetchTransport: vi.fn(() => ({
    send: vi.fn().mockRejectedValue(new Error("net::ERR_BLOCKED_BY_CLIENT")),
    flush: vi.fn().mockResolvedValue(true),
  })),
  withScope: vi.fn(
    (fn: (scope: { setTag: typeof vi.fn; setLevel: typeof vi.fn }) => void) => {
      fn({
        setTag: vi.fn(),
        setLevel: vi.fn(),
      });
    },
  ),
  captureException: vi.fn(),
  captureMessage: vi.fn(),
}));

// The SDK loads through a dynamic import, so init resolves a task after the
// config call that requested it.
async function flushSdk() {
  await new Promise((resolve) => setTimeout(resolve, 0));
}

describe("sentry", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetSentryForTests();
  });

  it("does not init without a DSN", async () => {
    applyRuntimeSentryConfig({});
    await flushSdk();
    expect(Sentry.init).not.toHaveBeenCalled();
    expect(sentryEnabled()).toBe(false);
  });

  it("initializes from runtime config", async () => {
    applyRuntimeSentryConfig({
      dsn: "https://glitchtip.example/1",
      environment: "staging",
      release: "melovian@test",
      tracesSampleRate: 0.1,
      clientReporting: true,
    });
    await vi.waitFor(() => expect(Sentry.init).toHaveBeenCalled());
    expect(Sentry.init).toHaveBeenCalledWith(
      expect.objectContaining({
        dsn: "https://glitchtip.example/1",
        environment: "staging",
        release: "melovian@test",
        tracesSampleRate: 0.1,
        sendDefaultPii: false,
      }),
    );
    const opts = vi.mocked(Sentry.init).mock.calls[0][0];
    expect(typeof opts?.beforeSend).toBe("function");
    expect(typeof opts?.beforeBreadcrumb).toBe("function");
    expect(sentryEnabled()).toBe(true);
  });

  it("scrubs credentials and PII from events", async () => {
    applyRuntimeSentryConfig({
      dsn: "https://glitchtip.example/1",
      clientReporting: true,
    });
    await vi.waitFor(() => expect(Sentry.init).toHaveBeenCalled());
    const opts = vi.mocked(Sentry.init).mock.calls[0][0];
    const event = {
      request: {
        url: "https://music.example/rest/stream?id=9&u=alice&t=tok&s=salt",
        query_string: "u=alice&p=hunter2&id=9",
        headers: {
          Authorization: "Bearer secret",
          "Content-Type": "application/json",
        },
        cookies: "session=abc",
        data: { password: "hunter2" },
      },
      user: { id: "u1", username: "alice", email: "a@b.c" },
    };
    const scrubbed = opts?.beforeSend?.(event as never, {} as never) as
      typeof event | null;
    expect(scrubbed?.request?.url).not.toContain("alice");
    expect(scrubbed?.request?.url).not.toContain("tok");
    expect(scrubbed?.request?.url).toContain("id=9");
    expect(scrubbed?.request?.query_string).not.toContain("hunter2");
    expect(scrubbed?.request?.headers?.Authorization).toBe("[redacted]");
    expect(scrubbed?.request?.headers?.["Content-Type"]).toBe(
      "application/json",
    );
    expect(scrubbed?.request).not.toHaveProperty("cookies");
    expect(scrubbed?.request).not.toHaveProperty("data");
    expect(scrubbed?.user).toEqual({ id: "u1" });
  });

  it("scrubs URLs inside breadcrumbs", async () => {
    applyRuntimeSentryConfig({
      dsn: "https://glitchtip.example/1",
      clientReporting: true,
    });
    await vi.waitFor(() => expect(Sentry.init).toHaveBeenCalled());
    const opts = vi.mocked(Sentry.init).mock.calls[0][0];
    const crumb = {
      data: { url: "/rest/ping?u=alice&p=hunter2" },
    };
    const out = opts?.beforeBreadcrumb?.(crumb as never, {} as never);
    const url = (out as typeof crumb | null)?.data.url as string;
    expect(url).not.toContain("alice");
    expect(url).not.toContain("hunter2");
  });

  it("refuses to init from build env without consent", async () => {
    vi.stubEnv("VITE_SENTRY_DSN", "https://glitchtip.example/1");
    try {
      localStorage.removeItem("melovian-telemetry-consent");
      initSentryFromBuildEnv();
      await flushSdk();
      expect(Sentry.init).not.toHaveBeenCalled();
      localStorage.setItem("melovian-telemetry-consent", "accepted");
      initSentryFromBuildEnv();
      await vi.waitFor(() => expect(Sentry.init).toHaveBeenCalledTimes(1));
    } finally {
      vi.unstubAllEnvs();
      localStorage.removeItem("melovian-telemetry-consent");
    }
  });

  it("skips duplicate init for the same DSN", async () => {
    applyRuntimeSentryConfig({
      dsn: "https://glitchtip.example/1",
      clientReporting: true,
    });
    await vi.waitFor(() => expect(Sentry.init).toHaveBeenCalledTimes(1));
    applyRuntimeSentryConfig({
      dsn: "https://glitchtip.example/1",
      clientReporting: true,
    });
    await flushSdk();
    expect(Sentry.init).toHaveBeenCalledTimes(1);
  });

  it("skips init for a dsn whose sends were blocked", async () => {
    localStorage.clear();
    applyRuntimeSentryConfig({
      dsn: "https://glitchtip.example/1",
      clientReporting: true,
    });
    await vi.waitFor(() => expect(Sentry.init).toHaveBeenCalled());
    const opts = vi.mocked(Sentry.init).mock.calls[0][0];
    const transport = opts.transport?.({} as never);
    await transport?.send({} as never);

    resetSentryForTests();
    applyRuntimeSentryConfig({
      dsn: "https://glitchtip.example/1",
      clientReporting: true,
    });
    await flushSdk();
    expect(Sentry.init).toHaveBeenCalledTimes(1);
    expect(sentryEnabled()).toBe(false);
  });

  it("captures client errors when enabled", async () => {
    applyRuntimeSentryConfig({
      dsn: "https://glitchtip.example/1",
      clientReporting: true,
    });
    await vi.waitFor(() => expect(Sentry.init).toHaveBeenCalled());
    captureClientError(new Error("boom"), "test");
    expect(Sentry.captureException).toHaveBeenCalled();
  });

  it("does not init from build env when unset", async () => {
    initSentryFromBuildEnv();
    await flushSdk();
    expect(Sentry.init).not.toHaveBeenCalled();
  });
});
