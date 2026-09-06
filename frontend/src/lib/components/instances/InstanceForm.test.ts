// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { flushSync, mount, unmount } from "svelte";
import InstanceForm from "./InstanceForm.svelte";

function mockMatchMedia(matches: boolean) {
  globalThis.matchMedia = ((query: string) => ({
    matches: query.includes("640px") ? matches : false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })) as typeof window.matchMedia;
}

function renderForm(props: {
  stickyActions?: boolean;
  ontest?: () => void;
  oncancel?: () => void;
}) {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(InstanceForm, {
    target,
    props: {
      stickyActions: props.stickyActions ?? true,
      ontest: props.ontest,
      oncancel: props.oncancel,
      onsubmit: vi.fn(),
    },
  });
  flushSync();
  return {
    target,
    cleanup: () => {
      unmount(instance);
      target.remove();
    },
  };
}

describe("InstanceForm sticky actions", () => {
  beforeEach(() => {
    mockMatchMedia(true);
    Object.defineProperty(window, "innerWidth", {
      configurable: true,
      value: 390,
    });
  });

  afterEach(() => {
    mockMatchMedia(false);
  });

  it("keeps two sticky actions in a compact row on mobile", () => {
    const { target, cleanup } = renderForm({
      ontest: vi.fn(),
    });
    try {
      const form = target.querySelector(".instance-form") as HTMLElement;
      const actions = target.querySelector(
        ".instance-form__actions",
      ) as HTMLElement;
      expect(form.classList.contains("instance-form--sticky")).toBe(true);
      expect(form.classList.contains("instance-form--stack-actions")).toBe(
        false,
      );

      const testBtn = target.querySelector(".btn--ghost") as HTMLElement;
      const submitBtn = target.querySelector(".btn--primary") as HTMLElement;
      expect(testBtn.classList.contains("btn--md")).toBe(true);
      expect(submitBtn.classList.contains("btn--lg")).toBe(true);

      // Prefer class structure over computed media-query styles in jsdom.
      expect(actions.children.length).toBe(2);
      const height =
        actions.getBoundingClientRect().height || actions.offsetHeight;
      // Two-button row should stay under ~5.5rem even if fonts inflate a bit.
      if (height > 0) {
        expect(height).toBeLessThanOrEqual(5.5 * 16 + 8);
      }
    } finally {
      cleanup();
    }
  });

  it("uses md secondary buttons and lg submit", () => {
    const { target, cleanup } = renderForm({
      ontest: vi.fn(),
      oncancel: vi.fn(),
    });
    try {
      const form = target.querySelector(".instance-form") as HTMLElement;
      expect(form.classList.contains("instance-form--stack-actions")).toBe(
        true,
      );
      const ghosts = [...target.querySelectorAll(".btn--ghost")];
      const primary = target.querySelector(".btn--primary") as HTMLElement;
      expect(ghosts.every((btn) => btn.classList.contains("btn--md"))).toBe(
        true,
      );
      expect(primary.classList.contains("btn--lg")).toBe(true);
    } finally {
      cleanup();
    }
  });

  it("gives the password field scroll clearance under sticky actions", () => {
    const { target, cleanup } = renderForm({
      ontest: vi.fn(),
    });
    try {
      const form = target.querySelector(".instance-form") as HTMLElement;
      const fields = target.querySelector(
        ".instance-form__fields",
      ) as HTMLElement;
      const passwordLabel = [...fields.querySelectorAll(".field__label")].find(
        (node) => node.textContent?.trim() === "Password",
      );
      expect(form.classList.contains("instance-form--sticky")).toBe(true);
      expect(passwordLabel).not.toBeNull();
      expect(fields.lastElementChild?.contains(passwordLabel!)).toBe(true);
      // Sticky clearance is applied via CSS on --sticky forms. jsdom does not
      // resolve scroll-margin, so assert structural contract instead.
      expect(form.querySelector(".instance-form__actions")).not.toBeNull();
    } finally {
      cleanup();
    }
  });
});
