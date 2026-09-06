// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export interface APIErrorBody {
  error: string;
  code: string;
}

export async function readAPIError(
  response: Response,
  fallback = "Request failed",
): Promise<string> {
  const text = await response.text().catch(() => "");
  const contentType = response.headers.get("Content-Type") ?? "";
  if (contentType.includes("application/json") || text.startsWith("{")) {
    try {
      const payload = JSON.parse(text) as Partial<APIErrorBody>;
      if (payload.error) return payload.error;
    } catch {
      // fall through to plain text
    }
  }
  const trimmed = text.trim();
  return trimmed || response.statusText || fallback;
}

export async function requireOk(
  response: Response,
  fallback: string,
): Promise<void> {
  if (response.ok) return;
  throw new Error(await readAPIError(response, fallback));
}
