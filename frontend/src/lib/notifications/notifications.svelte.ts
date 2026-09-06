// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { eventSocket } from "$lib/core/events/ws.svelte";
import { router } from "$lib/router/router.svelte";
import { toast } from "$lib/ui/toast.svelte";
import { deviceSync } from "$lib/music/device-sync.svelte";
import * as api from "./api";
import type { AppNotification } from "./api";

function listenInviteToken(href: string): string | null {
  const match = href.trim().match(/^\/listen\/([^/?#]+)/);
  return match?.[1] ? decodeURIComponent(match[1]) : null;
}

class NotificationsStore {
  items = $state<AppNotification[]>([]);
  unread = $state(0);
  loading = $state(false);
  panelOpen = $state(false);
  started = $state(false);

  #unsubscribers: Array<() => void> = [];

  async start() {
    if (this.started) return;
    this.started = true;
    await this.refresh();
    this.#unsubscribers.push(
      eventSocket.on("notification.created", (event) => {
        const item = event.payload as AppNotification;
        if (!item?.id) return;
        this.items = [item, ...this.items.filter((n) => n.id !== item.id)];
        if (!item.read) this.unread += 1;
        this.toastForNotification(item);
      }),
      eventSocket.on("notification.updated", (event) => {
        const payload = event.payload as {
          action?: string;
          id?: string;
          notification?: AppNotification;
        };
        if (payload.action === "read_all") {
          this.items = this.items.map((n) => ({ ...n, read: true }));
          this.unread = 0;
          return;
        }
        if (payload.action === "deleted" && payload.id) {
          this.items = this.items.filter((n) => n.id !== payload.id);
          void this.refreshUnread();
          return;
        }
        if (payload.notification) {
          const next = payload.notification;
          this.items = this.items.map((n) => (n.id === next.id ? next : n));
          void this.refreshUnread();
        }
      }),
    );
  }

  private toastForNotification(item: AppNotification) {
    if (item.kind === "party_invite" && item.href) {
      toast.info(item.body || "Open the invite to join their session.", {
        title: item.title,
        duration: 15000,
        action: {
          label: "Join",
          onClick: () => {
            void this.acceptPartyInvite(item);
          },
        },
      });
      return;
    }
    if (
      item.kind === "party_joined" ||
      item.kind === "party_left" ||
      item.kind === "party_ended"
    ) {
      // Live session.updated already toasts connected members.
      return;
    }
    toast.info(item.body || item.title, {
      title: item.title,
      action: item.href
        ? {
            label: "Open",
            onClick: () => {
              void this.openNotification(item);
            },
          }
        : undefined,
    });
  }

  async acceptPartyInvite(item: AppNotification) {
    if (!item.read) await this.markRead(item.id);
    this.closePanel();
    const token = listenInviteToken(item.href);
    if (!token) {
      if (item.href) router.navigate(item.href);
      return;
    }
    if (!eventSocket.connected) {
      eventSocket.connect();
    }
    deviceSync.start();
    const ok = await deviceSync.joinByInviteToken(token);
    if (!ok && item.href) router.navigate(item.href);
  }

  stop() {
    for (const unsub of this.#unsubscribers) unsub();
    this.#unsubscribers = [];
    this.started = false;
    this.items = [];
    this.unread = 0;
    this.panelOpen = false;
  }

  async refresh() {
    this.loading = true;
    try {
      const page = await api.listNotifications(50, 0);
      this.items = page.items;
      this.unread = page.unread;
    } catch {
      // Auth may be off or session expired. Keep prior state.
    } finally {
      this.loading = false;
    }
  }

  async refreshUnread() {
    try {
      this.unread = await api.getUnreadCount();
    } catch {
      // ignore
    }
  }

  togglePanel() {
    this.panelOpen = !this.panelOpen;
    if (this.panelOpen) void this.refresh();
  }

  closePanel() {
    this.panelOpen = false;
  }

  async markRead(id: string) {
    const item = this.items.find((n) => n.id === id);
    if (item && !item.read) {
      item.read = true;
      this.unread = Math.max(0, this.unread - 1);
    }
    try {
      const updated = await api.markNotificationRead(id);
      this.items = this.items.map((n) => (n.id === updated.id ? updated : n));
    } catch {
      void this.refresh();
    }
  }

  async markAllRead() {
    this.items = this.items.map((n) => ({ ...n, read: true }));
    this.unread = 0;
    try {
      await api.markAllNotificationsRead();
    } catch {
      void this.refresh();
    }
  }

  async remove(id: string) {
    this.items = this.items.filter((n) => n.id !== id);
    void this.refreshUnread();
    try {
      await api.deleteNotification(id);
    } catch {
      void this.refresh();
    }
  }

  async openNotification(item: AppNotification) {
    if (!item.read) await this.markRead(item.id);
    this.closePanel();
    if (item.href) router.navigate(item.href);
  }
}

export const notifications = new NotificationsStore();
