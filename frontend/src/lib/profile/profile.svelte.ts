// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  loadProfileSettings,
  readAvatarFromFile,
  saveCustomAvatarUrl,
  saveSmileVariants,
} from "./profile-settings";

class ProfileStore {
  customAvatarUrl = $state<string | null>(
    loadProfileSettings().customAvatarUrl,
  );
  smileVariants = $state<Record<string, number>>(
    loadProfileSettings().smileVariants,
  );

  constructor() {
    $effect.root(() => {
      $effect(() => {
        saveCustomAvatarUrl(this.customAvatarUrl);
      });

      $effect(() => {
        saveSmileVariants({ ...this.smileVariants });
      });
    });
  }

  getVariant(seed: string): number {
    return this.smileVariants[seed] ?? 0;
  }

  setVariant(seed: string, variant: number) {
    if (!Number.isFinite(variant) || variant < 0) return;
    this.smileVariants = {
      ...this.smileVariants,
      [seed]: Math.floor(variant),
    };
  }

  incrementVariant(seed: string) {
    this.setVariant(seed, this.getVariant(seed) + 1);
  }

  async setCustomAvatarFromFile(file: File) {
    const url = await readAvatarFromFile(file);
    this.customAvatarUrl = url;
  }

  clearCustomAvatar() {
    this.customAvatarUrl = null;
  }
}

export const profile = new ProfileStore();
