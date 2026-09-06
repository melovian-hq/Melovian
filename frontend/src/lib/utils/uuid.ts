// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

function generateUUIDBytes(): Uint8Array {
  const bytes = new Uint8Array(16);
  if (typeof globalThis.crypto?.getRandomValues === "function") {
    try {
      globalThis.crypto.getRandomValues(bytes);
      return bytes;
    } catch {
      // Fall back when getRandomValues is unavailable in insecure contexts.
    }
  }

  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = Math.floor(Math.random() * 256);
  }
  return bytes;
}

function formatUUIDFromBytes(bytes: Uint8Array): string {
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;

  const hex = [...bytes].map((b) => b.toString(16).padStart(2, "0")).join("");
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

function melovianRandomUUID(): `${string}-${string}-${string}-${string}-${string}` {
  return formatUUIDFromBytes(
    generateUUIDBytes(),
  ) as `${string}-${string}-${string}-${string}-${string}`;
}

function tryNativeRandomUUID(): string | undefined {
  if (globalThis.isSecureContext === false) {
    return undefined;
  }

  try {
    const crypto = globalThis.crypto;
    if (crypto && typeof crypto.randomUUID === "function") {
      return crypto.randomUUID();
    }
  } catch {
    return undefined;
  }

  return undefined;
}

export function randomUUID(): string {
  return tryNativeRandomUUID() ?? melovianRandomUUID();
}

export function installInsecureContextPolyfills(): void {
  if (typeof globalThis === "undefined") {
    return;
  }

  if (typeof globalThis.crypto === "undefined") {
    const pseudoCrypto = {
      getRandomValues<T extends ArrayBufferView>(array: T): T {
        const view = new Uint8Array(
          array.buffer,
          array.byteOffset,
          array.byteLength,
        );
        for (let i = 0; i < view.length; i++) {
          view[i] = Math.floor(Math.random() * 256);
        }
        return array;
      },
      randomUUID: () => melovianRandomUUID(),
    };
    Object.defineProperty(globalThis, "crypto", {
      value: pseudoCrypto,
      configurable: true,
      writable: true,
    });
    return;
  }

  if (typeof globalThis.crypto.randomUUID !== "function") {
    globalThis.crypto.randomUUID = () => melovianRandomUUID();
  }
}
