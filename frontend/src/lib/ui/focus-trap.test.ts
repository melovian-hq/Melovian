// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  focusInitial,
  getFocusableElements,
  trapFocus,
} from "$lib/ui/focus-trap";

describe("focus-trap", () => {
  it("lists focusable elements in order", () => {
    const root = document.createElement("div");
    root.innerHTML = `
      <button type="button">Cancel</button>
      <button type="button" class="confirm__danger">Delete</button>
      <button type="button" disabled>Gone</button>
    `;
    document.body.appendChild(root);
    const list = getFocusableElements(root);
    expect(list).toHaveLength(2);
    expect(list[0].textContent).toBe("Cancel");
    root.remove();
  });

  it("focuses danger button when preferDanger is true", () => {
    const root = document.createElement("div");
    root.innerHTML = `
      <button type="button">Cancel</button>
      <button type="button" class="confirm__danger">Delete</button>
    `;
    document.body.appendChild(root);
    const focused = focusInitial(root, true);
    expect(focused?.classList.contains("confirm__danger")).toBe(true);
    root.remove();
  });

  it("cycles Tab from last to first", () => {
    const root = document.createElement("div");
    root.innerHTML = `
      <button type="button" id="a">A</button>
      <button type="button" id="b">B</button>
    `;
    document.body.appendChild(root);
    const release = trapFocus(root);
    const b = root.querySelector("#b") as HTMLButtonElement;
    b.focus();
    root.dispatchEvent(
      new KeyboardEvent("keydown", { key: "Tab", bubbles: true }),
    );
    expect(document.activeElement?.id).toBe("a");
    release();
    root.remove();
  });
});
