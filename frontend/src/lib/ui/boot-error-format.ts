// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { APP_NAME } from "$lib/brand";

export interface FatalErrorDetails {
  title: string;
  message: string;
  stack?: string;
  source?: string;
  url: string;
  time: string;
}

export function formatError(
  error: unknown,
  source?: string,
): FatalErrorDetails {
  const time = new Date().toISOString();
  const url = typeof location !== "undefined" ? location.href : "";

  if (error instanceof Error) {
    return {
      title: `${APP_NAME} crashed`,
      message: error.message || "Unknown error",
      stack: error.stack,
      source,
      url,
      time,
    };
  }

  if (typeof error === "string") {
    return {
      title: `${APP_NAME} crashed`,
      message: error,
      source,
      url,
      time,
    };
  }

  return {
    title: `${APP_NAME} crashed`,
    message: "An unknown error occurred",
    source,
    url,
    time,
  };
}
