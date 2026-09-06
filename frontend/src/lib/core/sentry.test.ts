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

describe("sentry", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetSentryForTests();
  });

  it("does not init without a DSN", () => {
    applyRuntimeSentryConfig({});
    expect(Sentry.init).not.toHaveBeenCalled();
    expect(sentryEnabled()).toBe(false);
  });

  it("initializes from runtime config", () => {
    applyRuntimeSentryConfig({
      dsn: "https://glitchtip.example/1",
      environment: "staging",
      release: "melovian@test",
      tracesSampleRate: 0.1,
      clientReporting: true,
    });
    expect(Sentry.init).toHaveBeenCalledWith({
      dsn: "https://glitchtip.example/1",
      environment: "staging",
      release: "melovian@test",
      tracesSampleRate: 0.1,
    });
    expect(sentryEnabled()).toBe(true);
  });

  it("skips duplicate init for the same DSN", () => {
    applyRuntimeSentryConfig({
      dsn: "https://glitchtip.example/1",
      clientReporting: true,
    });
    applyRuntimeSentryConfig({
      dsn: "https://glitchtip.example/1",
      clientReporting: true,
    });
    expect(Sentry.init).toHaveBeenCalledTimes(1);
  });

  it("captures client errors when enabled", () => {
    applyRuntimeSentryConfig({
      dsn: "https://glitchtip.example/1",
      clientReporting: true,
    });
    captureClientError(new Error("boom"), "test");
    expect(Sentry.captureException).toHaveBeenCalled();
  });

  it("does not init from build env when unset", () => {
    initSentryFromBuildEnv();
    expect(Sentry.init).not.toHaveBeenCalled();
  });
});
