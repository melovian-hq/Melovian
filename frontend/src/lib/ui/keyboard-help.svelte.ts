// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

class KeyboardHelpStore {
  open = $state(false);

  toggle() {
    this.open = !this.open;
  }

  close() {
    this.open = false;
  }
}

export const keyboardHelp = new KeyboardHelpStore();
