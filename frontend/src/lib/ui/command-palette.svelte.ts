// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

class CommandPaletteStore {
  open = $state(false);

  toggle() {
    this.open = !this.open;
  }

  openPalette() {
    this.open = true;
  }

  close() {
    this.open = false;
  }
}

export const commandPalette = new CommandPaletteStore();
