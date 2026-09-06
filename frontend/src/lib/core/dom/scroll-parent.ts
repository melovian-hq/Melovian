// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export function findScrollParent(el: HTMLElement | null): HTMLElement {
  if (!el) return document.documentElement;
  let node: HTMLElement | null = el.parentElement;
  while (node) {
    const style = getComputedStyle(node);
    if (/(auto|scroll|overlay)/.test(style.overflowY)) {
      return node;
    }
    node = node.parentElement;
  }
  return document.documentElement;
}

export function offsetWithin(el: HTMLElement, ancestor: HTMLElement): number {
  let top = 0;
  let node: HTMLElement | null = el;
  while (node && node !== ancestor) {
    top += node.offsetTop;
    node = node.offsetParent as HTMLElement | null;
    if (node && !ancestor.contains(node)) {
      break;
    }
  }
  return top;
}
