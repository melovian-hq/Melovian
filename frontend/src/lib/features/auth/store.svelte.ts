// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as authApi from "./api";
import type { AuthUser } from "./api";
import { changeUsername as changeUsernameRequest } from "./account";
import { logger } from "$lib/core/logger";

class AuthStore {
  enabled = $state(false);
  authenticated = $state(false);
  setupRequired = $state(false);
  demoMode = $state(false);
  fakeCatalog = $state(false);
  oidcEnabled = $state(false);
  oidcLoginUrl = $state("/api/auth/oidc/login");
  localLoginEnabled = $state(true);
  user = $state<AuthUser | null>(null);
  loading = $state(true);
  error = $state<string | null>(null);
  /** True after a successful /api/auth/status response in this session. */
  statusLoaded = $state(false);

  ready = $derived(!this.loading);
  needsAccountLogin = $derived(
    this.ready &&
      this.statusLoaded &&
      this.enabled &&
      !this.authenticated &&
      !this.demoMode,
  );
  showLocalAccountForm = $derived(
    this.ready &&
      this.statusLoaded &&
      this.enabled &&
      !this.authenticated &&
      (this.setupRequired || this.localLoginEnabled),
  );

  async init() {
    this.loading = true;
    this.error = null;
    try {
      const status = await authApi.getAuthStatus();
      this.enabled = status.enabled;
      this.authenticated = status.authenticated;
      this.setupRequired = status.setupRequired;
      this.demoMode = status.demoMode === true;
      this.fakeCatalog = status.fakeCatalog === true;
      this.oidcEnabled = status.oidcEnabled ?? false;
      this.oidcLoginUrl = status.oidcLoginUrl ?? "/api/auth/oidc/login";
      this.localLoginEnabled = status.localLoginEnabled ?? true;
      this.user = status.user ?? null;
      this.statusLoaded = true;
    } catch (err) {
      this.error =
        err instanceof Error ? err.message : "Failed to load auth status";
      this.statusLoaded = false;
      logger.error(
        "Failed to load auth status",
        err,
        undefined,
        "auth.init",
      );
    } finally {
      this.loading = false;
    }
  }

  async setup(username: string, password: string) {
    const user = await authApi.setupAccount(username, password);
    this.authenticated = true;
    this.setupRequired = false;
    this.user = user;
  }

  async login(username: string, password: string) {
    const user = await authApi.loginAccount(username, password);
    this.authenticated = true;
    this.user = user;
  }

  async logout() {
    await authApi.logoutAccount();
    this.authenticated = false;
    this.user = null;
  }

  async updateUsername(name: string) {
    const user = await changeUsernameRequest(name);
    this.user = user;
    return user;
  }
}

export const auth = new AuthStore();
