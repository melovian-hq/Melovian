// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export interface ConfirmOptions {
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
}

class ConfirmStore {
  open = $state(false);
  options = $state<ConfirmOptions | null>(null);

  private resolver: ((value: boolean) => void) | null = null;

  confirm(options: ConfirmOptions): Promise<boolean> {
    return new Promise((resolve) => {
      this.options = options;
      this.open = true;
      this.resolver = resolve;
    });
  }

  accept() {
    this.open = false;
    this.resolver?.(true);
    this.resolver = null;
    this.options = null;
  }

  cancel() {
    this.open = false;
    this.resolver?.(false);
    this.resolver = null;
    this.options = null;
  }
}

export const confirmDialog = new ConfirmStore();
