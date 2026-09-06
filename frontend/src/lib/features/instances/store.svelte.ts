// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as libraryApi from "../local-libraries/api";
import * as instanceApi from "./api";
import { setActiveInstanceId } from "./context";
import type { InstanceInput, SubsonicInstance } from "./types";

class InstanceStore {
  items = $state.raw<SubsonicInstance[]>([]);
  activeId = $state<string | null>(null);
  loading = $state(true);
  switching = $state(false);
  error = $state<string | null>(null);

  active = $derived(
    this.items.find((item) => item.id === this.activeId) ?? null,
  );

  ready = $derived(!this.loading);
  needsLogin = $derived(this.ready && this.activeId === null);

  private initialized = false;

  async init() {
    if (this.initialized) {
      await this.refresh();
      return;
    }
    this.initialized = true;
    this.loading = true;
    this.error = null;
    try {
      const [items, active] = await Promise.all([
        instanceApi.listInstances(),
        instanceApi.getActiveInstance(),
      ]);
      this.items = items;
      this.activeId = active?.id ?? null;
      setActiveInstanceId(this.activeId);
    } catch (err) {
      this.error =
        err instanceof Error ? err.message : "Failed to load instances";
    } finally {
      this.loading = false;
    }
  }

  async refresh() {
    const [items, active] = await Promise.all([
      instanceApi.listInstances(),
      instanceApi.getActiveInstance(),
    ]);
    this.items = items;
    this.activeId = active?.id ?? null;
    setActiveInstanceId(this.activeId);
  }

  async add(input: InstanceInput) {
    const created = await instanceApi.createInstance(input);
    await this.refresh();
    const activeLocal = await libraryApi
      .getActiveLocalLibrary()
      .catch(() => null);
    if (!this.activeId && !activeLocal?.id) {
      await this.activate(created.id);
    }
    return created;
  }

  async update(id: string, input: InstanceInput) {
    const updated = await instanceApi.updateInstance(id, input);
    await this.refresh();
    return updated;
  }

  async test(input: InstanceInput) {
    return instanceApi.testInstance(input);
  }

  async activate(id: string) {
    this.switching = true;
    try {
      await instanceApi.activateInstance(id);
      await this.refresh();
    } finally {
      this.switching = false;
    }
  }

  async remove(id: string) {
    await instanceApi.deleteInstance(id);
    await this.refresh();
  }
}

export const instances = new InstanceStore();
