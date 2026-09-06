// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/** Absolute cover URL for native Android MediaSession (HttpURLConnection). */
export function absoluteMobileArtworkUrl(
  path: string | null | undefined,
  origin = typeof window !== "undefined" ? window.location.origin : "",
): string {
  if (!path) return "";
  const trimmed = path.trim();
  if (!trimmed) return "";
  if (/^data:/i.test(trimmed)) return "";
  if (/^https?:\/\//i.test(trimmed)) return trimmed;
  if (!origin) return trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  if (trimmed.startsWith("/")) return `${origin}${trimmed}`;
  return `${origin}/${trimmed}`;
}
