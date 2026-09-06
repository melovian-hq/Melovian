// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export interface ClientErrorView {
  message: string;
  detail?: string;
}

function rawMessage(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (typeof error === "string") return error;
  try {
    return JSON.stringify(error);
  } catch {
    return "An unexpected error occurred";
  }
}

export function mapClientError(error: unknown): ClientErrorView {
  const detail = rawMessage(error).trim();
  const lower = detail.toLowerCase();

  if (!detail) {
    return { message: "Something went wrong. Try again." };
  }

  if (
    lower.includes("401") ||
    lower.includes("unauthorized") ||
    lower.includes("invalid credentials")
  ) {
    return {
      message: "Sign-in failed or your session expired.",
      detail,
    };
  }

  if (
    lower.includes("failed to fetch") ||
    lower.includes("networkerror") ||
    lower.includes("network request failed") ||
    lower.includes("econnrefused") ||
    lower.includes("connection refused")
  ) {
    return {
      message: "Could not reach the server. Check the URL and your network.",
      detail,
    };
  }

  if (lower.includes("certificate") || lower.includes("tls")) {
    return {
      message: "Secure connection failed. Check HTTPS settings for the server.",
      detail,
    };
  }

  if (lower.includes("timeout") || lower.includes("timed out")) {
    return {
      message: "The request timed out. The server may be slow or offline.",
      detail,
    };
  }

  if (lower.includes("not found") && lower.includes("track")) {
    return {
      message: "That track is no longer in the library.",
      detail,
    };
  }

  if (lower.includes("tag") || lower.includes("metadata")) {
    return {
      message: "Could not update file tags. The file may be read-only.",
      detail,
    };
  }

  if (detail.length > 140) {
    return {
      message: detail.slice(0, 140).trimEnd() + "…",
      detail,
    };
  }

  return { message: detail };
}
