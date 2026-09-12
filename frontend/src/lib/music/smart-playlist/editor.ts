// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  SMART_PLAYLIST_FIELDS,
  SMART_PLAYLIST_OPERATORS,
  SMART_PLAYLIST_SORT_OPTIONS,
  getSmartField,
} from "./fields";
import type {
  SmartPlaylistGroup,
  SmartPlaylistRule,
  SmartRuleValue,
} from "./types";

export const FIELD_ITEMS = SMART_PLAYLIST_FIELDS.map((field) => ({
  value: field.id,
  label: field.label,
}));

export const SORT_ITEMS = SMART_PLAYLIST_SORT_OPTIONS.map((option) => ({
  value: option.id as string,
  label: option.label,
}));

export const PRESENCE_ITEMS = [
  { value: "present", label: "Present" },
  { value: "missing", label: "Missing" },
];

export const BOOLEAN_ITEMS = [
  { value: "true", label: "Yes" },
  { value: "false", label: "No" },
];

export function findGroupPath(
  group: SmartPlaylistGroup,
  targetId: string,
  prefix: string,
): string | null {
  for (let index = 0; index < group.groups.length; index += 1) {
    const child = group.groups[index]!;
    const childPath = `${prefix}.groups[${index}]`;
    if (child.id === targetId) return childPath;
    const nested = findGroupPath(child, targetId, childPath);
    if (nested) return nested;
  }
  return null;
}

export function groupPathFor(
  root: SmartPlaylistGroup,
  groupId: string,
): string {
  if (root.id === groupId) return "root";
  return findGroupPath(root, groupId, "root") ?? "root";
}

export function rulePath(
  root: SmartPlaylistGroup,
  groupId: string,
  index: number,
): string {
  const groupPath = groupPathFor(root, groupId);
  return `${groupPath}.rules[${index}]`;
}

export function defaultValueFor(
  fieldId: string,
  operator: string,
): SmartRuleValue {
  const field = getSmartField(fieldId);
  if (operator === "isMissing" || operator === "isPresent") return true;
  if (operator === "inTheRange") {
    return field?.valueType === "number" ? [0, 0] : ["", ""];
  }
  if (field?.valueType === "boolean") return true;
  if (field?.valueType === "days" || field?.valueType === "number") return 0;
  return "";
}

export function valueInputType(rule: SmartPlaylistRule): string {
  const field = getSmartField(rule.field);
  if (!field) return "text";
  if (field.valueType === "number" || field.valueType === "days")
    return "number";
  if (field.valueType === "date") return "date";
  return "text";
}

export function operatorLabel(operator: string): string {
  return SMART_PLAYLIST_OPERATORS[operator]?.label ?? operator;
}

export function rangeValues(rule: SmartPlaylistRule): [string, string] {
  if (!Array.isArray(rule.value)) return ["", ""];
  return [String(rule.value[0] ?? ""), String(rule.value[1] ?? "")];
}

export function nextRangeValue(
  rule: SmartPlaylistRule,
  index: 0 | 1,
  raw: string,
): SmartRuleValue {
  const field = getSmartField(rule.field);
  const current = rangeValues(rule);
  current[index] = raw;
  return field?.valueType === "range-number"
    ? [Number(current[0]), Number(current[1])]
    : current;
}
