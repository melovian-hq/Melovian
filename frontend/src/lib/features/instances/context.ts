// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

let activeInstanceId: string | null = null;

export function setActiveInstanceId(id: string | null) {
  activeInstanceId = id;
}

export function getActiveInstanceId(): string | null {
  return activeInstanceId;
}
