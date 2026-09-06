// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export interface LocalLibraryScanProgress {
  processed: number;
  phase: "scanning" | "reconciling";
}

export interface LocalLibrary {
  id: string;
  type: "local";
  name: string;
  path: string;
  trackCount: number;
  missingCount: number;
  duplicateCount: number;
  scanStatus: string;
  scanError?: string;
  scanProgress?: LocalLibraryScanProgress;
  createdAt: string;
  updatedAt: string;
  lastScannedAt?: string;
}

export interface LocalLibraryInput {
  name: string;
  path?: string;
}

export interface LocalLibraryScanResult {
  added: number;
  updated: number;
  removed: number;
  duplicates: number;
  unchanged: number;
  total: number;
  errors?: string[];
}

export interface LocalLibraryConfig {
  enabled: boolean;
  defaultPath: string;
  allowCustomPath: boolean;
}
