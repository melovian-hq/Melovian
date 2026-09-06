// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { randomUUID } from "$lib/utils/uuid";
import { APP_NAME } from "$lib/brand";
import { getSmartField } from "./fields";
import type {
  CreateSmartPlaylistPayload,
  NavidromeSmartPlaylistRules,
  SmartPlaylistDraft,
  SmartPlaylistGroup,
  SmartPlaylistRule,
  SmartRuleValue,
} from "./types";
import { validateSmartPlaylistDraft } from "./validate";

function parseRegexValue(value: string): string {
  const wrapped = value.match(/^\/(.+)\/([gimsuy]*)$/);
  if (wrapped) {
    return wrapped[1] ?? value;
  }
  if (value.startsWith("/") && value.endsWith("/") && value.length > 2) {
    return value.slice(1, -1);
  }
  return value;
}

function compileRuleValue(
  rule: SmartPlaylistRule,
  navidromeField: string,
): Record<string, unknown> {
  const op = rule.operator;

  if (op === "isMissing" || op === "isPresent") {
    return { [op]: { [navidromeField]: rule.value === true } };
  }

  if (op === "inTheLast" || op === "notInTheLast") {
    return { [op]: { [navidromeField]: Number(rule.value) } };
  }

  if (op === "inTheRange") {
    const range = Array.isArray(rule.value) ? rule.value : ["", ""];
    return { [op]: { [navidromeField]: range } };
  }

  let value: string | number | boolean = rule.value as
    string | number | boolean;
  if (typeof value === "string" && op === "contains") {
    value = parseRegexValue(value);
  }

  return { [op]: { [navidromeField]: value } };
}

function compileRule(rule: SmartPlaylistRule): Record<string, unknown> {
  const field = getSmartField(rule.field);
  if (!field) {
    throw new Error(`Unknown field: ${rule.field}`);
  }
  return compileRuleValue(rule, field.navidromeField);
}

function compileGroup(group: SmartPlaylistGroup): Record<string, unknown[]> {
  const compiled: unknown[] = [
    ...group.rules.map((rule) => compileRule(rule)),
    ...group.groups.map((child) => compileGroupNode(child)),
  ];
  return { [group.logic]: compiled };
}

function compileGroupNode(
  group: SmartPlaylistGroup,
): Record<string, unknown[]> {
  return compileGroup(group);
}

export function compileSmartPlaylistDraft(
  draft: SmartPlaylistDraft,
): CreateSmartPlaylistPayload {
  const errors = validateSmartPlaylistDraft(draft);
  if (errors.length > 0) {
    throw new Error(errors[0]?.message ?? "Invalid smart playlist");
  }

  const rules: NavidromeSmartPlaylistRules = {
    ...compileGroup(draft.root),
    sort: draft.sort.trim(),
  };

  if (draft.limit !== null && draft.limit > 0) {
    rules.limit = draft.limit;
  } else if (draft.limitPercent !== null && draft.limitPercent > 0) {
    rules.limitPercent = draft.limitPercent;
  }

  const comment =
    draft.comment.trim() ||
    `SMART - ${draft.name.trim()} (created in ${APP_NAME})`;

  return {
    name: draft.name.trim(),
    comment,
    public: draft.public,
    rules,
  };
}

export function createEmptySmartPlaylistDraft(): SmartPlaylistDraft {
  return {
    name: "",
    comment: "",
    public: false,
    sort: "+random",
    limit: 100,
    limitPercent: null,
    root: {
      id: randomUUID(),
      logic: "all",
      rules: [
        {
          id: randomUUID(),
          field: "genre",
          operator: "contains",
          value: "",
        },
      ],
      groups: [],
    },
  };
}

export function coerceRuleValue(
  raw: string,
  fieldId: string,
  operator: string,
): SmartRuleValue {
  const field = getSmartField(fieldId);
  if (!field) return raw;

  if (operator === "isMissing" || operator === "isPresent") {
    return raw === "true";
  }

  if (
    field.valueType === "number" ||
    field.valueType === "days" ||
    operator === "inTheLast" ||
    operator === "notInTheLast"
  ) {
    const parsed = Number(raw);
    return Number.isFinite(parsed) ? parsed : Number.NaN;
  }

  if (operator === "inTheRange") {
    const [start = "", end = ""] = raw.split(",").map((part) => part.trim());
    if (field.valueType === "range-number") {
      return [Number(start), Number(end)];
    }
    return [start, end];
  }

  if (field.valueType === "boolean") {
    return raw === "true";
  }

  return raw;
}
