// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export type SmartPlaylistLogic = "all" | "any";

export type SmartRuleValue =
  string | number | boolean | [number, number] | [string, string];

export interface SmartPlaylistRule {
  id: string;
  field: string;
  operator: string;
  value: SmartRuleValue;
}

export interface SmartPlaylistGroup {
  id: string;
  logic: SmartPlaylistLogic;
  rules: SmartPlaylistRule[];
  groups: SmartPlaylistGroup[];
}

export interface SmartPlaylistDraft {
  name: string;
  comment: string;
  public: boolean;
  root: SmartPlaylistGroup;
  sort: string;
  limit: number | null;
  limitPercent: number | null;
}

export interface NavidromeSmartPlaylistRules {
  all?: unknown[];
  any?: unknown[];
  sort?: string;
  order?: string;
  limit?: number;
  limitPercent?: number;
}

export interface CreateSmartPlaylistPayload {
  name: string;
  comment?: string;
  public?: boolean;
  rules: NavidromeSmartPlaylistRules;
}

export interface SmartPlaylistValidationError {
  path: string;
  message: string;
}
