// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import * as v from "valibot";
import { authStatusSchema, authUserSchema } from "./schemas";
import { ssoProviderLabel } from "./api";

const baseStatus = {
  enabled: true,
  authenticated: false,
  setupRequired: false,
};

describe("authStatusSchema", () => {
  it("parses oidcProviderName and hasPassword", () => {
    const result = v.safeParse(authStatusSchema, {
      ...baseStatus,
      authenticated: true,
      oidcEnabled: true,
      oidcProviderName: "Pocket ID",
      user: { id: "u1", username: "alice", hasPassword: false },
    });
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.output.oidcProviderName).toBe("Pocket ID");
      expect(result.output.user?.hasPassword).toBe(false);
    }
  });

  it("accepts a status without the optional oidc fields", () => {
    const result = v.safeParse(authStatusSchema, baseStatus);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.output.oidcProviderName).toBeUndefined();
      expect(result.output.user).toBeUndefined();
    }
  });
});

describe("authUserSchema", () => {
  it("accepts a user without hasPassword", () => {
    const result = v.safeParse(authUserSchema, { id: "u1", username: "a" });
    expect(result.success).toBe(true);
  });
});

describe("ssoProviderLabel", () => {
  it("falls back to SSO", () => {
    expect(ssoProviderLabel(undefined)).toBe("SSO");
    expect(ssoProviderLabel(null)).toBe("SSO");
    expect(ssoProviderLabel("")).toBe("SSO");
    expect(ssoProviderLabel("   ")).toBe("SSO");
  });

  it("returns the trimmed provider name", () => {
    expect(ssoProviderLabel("Pocket ID")).toBe("Pocket ID");
    expect(ssoProviderLabel("  Authelia  ")).toBe("Authelia");
  });
});
