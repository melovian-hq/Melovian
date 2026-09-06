// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { APP_SLUG, StorageKeys } from "$lib/brand";

export const AVATAR_STORAGE_KEY = StorageKeys.profileAvatar;
export const SMILE_VARIANTS_STORAGE_KEY = StorageKeys.profileSmileVariants;

export const ACCEPTED_AVATAR_TYPES = [
  "image/jpeg",
  "image/png",
  "image/webp",
] as const;

export type AcceptedAvatarType = (typeof ACCEPTED_AVATAR_TYPES)[number];

export const ACCEPTED_AVATAR_EXTENSIONS = ".jpg,.jpeg,.png,.webp";
export const MAX_AVATAR_FILE_BYTES = 512 * 1024;
export const AVATAR_MAX_DIMENSION = 256;

export interface ProfileSettings {
  customAvatarUrl: string | null;
  smileVariants: Record<string, number>;
}

export function defaultProfileSettings(): ProfileSettings {
  return {
    customAvatarUrl: null,
    smileVariants: {},
  };
}

export function isAcceptedAvatarType(type: string): type is AcceptedAvatarType {
  return (ACCEPTED_AVATAR_TYPES as readonly string[]).includes(type);
}

export function isAcceptedAvatarFile(file: File): boolean {
  return isAcceptedAvatarType(file.type);
}

export function defaultAvatarSeed(
  activeInstance: { id?: string; username?: string } | null | undefined,
  serverName: string | null | undefined,
): string {
  return (
    activeInstance?.id ?? activeInstance?.username ?? serverName ?? APP_SLUG
  );
}

export function loadProfileSettings(): ProfileSettings {
  if (typeof localStorage === "undefined") {
    return defaultProfileSettings();
  }

  const settings = defaultProfileSettings();

  try {
    const avatar = localStorage.getItem(AVATAR_STORAGE_KEY)?.trim();
    if (avatar?.startsWith("data:image/")) {
      settings.customAvatarUrl = avatar;
    }

    const rawVariants = localStorage.getItem(SMILE_VARIANTS_STORAGE_KEY);
    if (rawVariants) {
      const parsed = JSON.parse(rawVariants) as unknown;
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        for (const [seed, value] of Object.entries(parsed)) {
          if (
            typeof value === "number" &&
            Number.isFinite(value) &&
            value >= 0
          ) {
            settings.smileVariants[seed] = Math.floor(value);
          }
        }
      }
    }
  } catch {
    return defaultProfileSettings();
  }

  return settings;
}

export function saveCustomAvatarUrl(url: string | null): void {
  if (typeof localStorage === "undefined") return;
  if (url) {
    localStorage.setItem(AVATAR_STORAGE_KEY, url);
    return;
  }
  localStorage.removeItem(AVATAR_STORAGE_KEY);
}

export function saveSmileVariants(variants: Record<string, number>): void {
  if (typeof localStorage === "undefined") return;
  localStorage.setItem(SMILE_VARIANTS_STORAGE_KEY, JSON.stringify(variants));
}

function loadImageFromFile(file: File): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file);
    const image = new Image();
    image.onload = () => {
      URL.revokeObjectURL(url);
      resolve(image);
    };
    image.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error("Could not read image file"));
    };
    image.src = url;
  });
}

function encodeCanvas(
  canvas: HTMLCanvasElement,
  type: AcceptedAvatarType,
): string {
  const quality = type === "image/jpeg" ? 0.9 : undefined;
  return canvas.toDataURL(type, quality);
}

export async function resizeAvatarImage(
  file: File,
  maxDimension = AVATAR_MAX_DIMENSION,
): Promise<string> {
  const image = await loadImageFromFile(file);
  const scale = Math.min(1, maxDimension / Math.max(image.width, image.height));
  const width = Math.max(1, Math.round(image.width * scale));
  const height = Math.max(1, Math.round(image.height * scale));

  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;

  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("Could not process image");
  }

  context.drawImage(image, 0, 0, width, height);
  return encodeCanvas(canvas, file.type as AcceptedAvatarType);
}

export async function readAvatarFromFile(file: File): Promise<string> {
  if (!isAcceptedAvatarFile(file)) {
    throw new Error("Avatar must be a JPG, PNG, or WebP image");
  }
  if (file.size > MAX_AVATAR_FILE_BYTES) {
    throw new Error("Avatar must be 512 KB or smaller");
  }

  return resizeAvatarImage(file);
}
