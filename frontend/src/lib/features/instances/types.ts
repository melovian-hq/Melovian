// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export interface SubsonicInstance {
  id: string;
  name: string;
  serverUrl: string;
  username: string;
  serverName: string;
  createdAt: string;
  updatedAt: string;
  lastUsedAt?: string;
}

export interface InstanceInput {
  name: string;
  serverUrl: string;
  username: string;
  password: string;
}
