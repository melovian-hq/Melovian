// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

const DEVICE_ID_KEY = "mel-device-id";
const DEVICE_NAME_KEY = "mel-device-name";

function randomId(): string {
  if (typeof crypto !== "undefined" && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return `dev-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

export function getOrCreateDeviceId(): string {
  try {
    const existing = localStorage.getItem(DEVICE_ID_KEY);
    if (existing) return existing;
    const id = randomId();
    localStorage.setItem(DEVICE_ID_KEY, id);
    return id;
  } catch {
    return randomId();
  }
}

export function loadDeviceName(): string {
  try {
    const saved = localStorage.getItem(DEVICE_NAME_KEY);
    if (saved?.trim()) return saved.trim();
  } catch {
    // ignore
  }
  return guessDeviceName(
    typeof navigator !== "undefined" ? navigator.userAgent : "",
  );
}

export function saveDeviceName(name: string): void {
  try {
    localStorage.setItem(DEVICE_NAME_KEY, name.trim());
  } catch {
    // ignore
  }
}

export function guessDeviceName(ua: string): string {
  if (ua.includes("iPhone")) return "iPhone";
  if (ua.includes("iPad")) return "iPad";
  if (ua.includes("Android")) return "Android";
  if (ua.includes("Macintosh") || ua.includes("Mac OS")) return "Mac";
  if (ua.includes("Windows")) return "Windows";
  if (ua.includes("Linux")) return "Linux";
  return "Browser";
}
