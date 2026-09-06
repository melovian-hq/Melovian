// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

class ClosePromptStore {
  open = $state(false);

  show(): void {
    this.open = true;
  }

  hide(): void {
    this.open = false;
  }
}

export const closePrompt = new ClosePromptStore();
