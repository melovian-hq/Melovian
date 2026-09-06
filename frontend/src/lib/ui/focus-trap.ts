// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

export function getFocusableElements(root: HTMLElement): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
    (el) => !el.hasAttribute("disabled") && el.tabIndex !== -1,
  );
}

/**
 * Trap Tab focus inside root. Returns a cleanup that removes the listener.
 */
export function trapFocus(root: HTMLElement): () => void {
  function onKeydown(event: KeyboardEvent) {
    if (event.key !== "Tab") return;
    const focusable = getFocusableElements(root);
    if (focusable.length === 0) {
      event.preventDefault();
      return;
    }
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    const active = document.activeElement as HTMLElement | null;
    if (event.shiftKey) {
      if (active === first || !root.contains(active)) {
        event.preventDefault();
        last.focus();
      }
      return;
    }
    if (active === last || !root.contains(active)) {
      event.preventDefault();
      first.focus();
    }
  }

  root.addEventListener("keydown", onKeydown);
  return () => root.removeEventListener("keydown", onKeydown);
}

export function focusInitial(
  root: HTMLElement,
  preferDanger = false,
): HTMLElement | null {
  const focusable = getFocusableElements(root);
  if (focusable.length === 0) return null;
  if (preferDanger) {
    const danger = focusable.find((el) =>
      el.classList.contains("confirm__danger"),
    );
    if (danger) {
      danger.focus();
      return danger;
    }
  }
  const cancel = focusable.find(
    (el) =>
      el.tagName === "BUTTON" && !el.classList.contains("confirm__danger"),
  );
  const target = cancel ?? focusable[0];
  target.focus();
  return target;
}
