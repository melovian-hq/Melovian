// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { getSmartField, SMART_PLAYLIST_OPERATORS } from "./fields";
import type {
  SmartPlaylistDraft,
  SmartPlaylistGroup,
  SmartPlaylistRule,
  SmartPlaylistValidationError,
  SmartRuleValue,
} from "./types";

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;

function isEmptyValue(value: SmartRuleValue): boolean {
  if (typeof value === "boolean") return false;
  if (typeof value === "number") return Number.isNaN(value);
  if (Array.isArray(value)) {
    return value.some((part) => String(part).trim() === "");
  }
  return String(value).trim() === "";
}

function validateRuleValue(
  rule: SmartPlaylistRule,
  path: string,
  errors: SmartPlaylistValidationError[],
): void {
  const field = getSmartField(rule.field);
  if (!field) {
    errors.push({ path, message: `Unknown field "${rule.field}"` });
    return;
  }

  const operator = SMART_PLAYLIST_OPERATORS[rule.operator];
  if (!operator) {
    errors.push({ path, message: `Unknown operator "${rule.operator}"` });
    return;
  }

  if (!field.operators.includes(rule.operator)) {
    errors.push({
      path,
      message: `Operator "${rule.operator}" is not valid for ${field.label}`,
    });
  }

  const operatorTypes = operator.valueTypes;
  const fieldType =
    rule.operator === "inTheRange"
      ? field.valueType === "number"
        ? "range-number"
        : field.valueType === "date"
          ? "range-date"
          : field.valueType
      : rule.operator === "inTheLast" || rule.operator === "notInTheLast"
        ? "days"
        : field.valueType;
  if (!operatorTypes.includes(fieldType)) {
    errors.push({
      path,
      message: `Operator "${rule.operator}" does not support ${field.label}`,
    });
  }

  if (rule.operator === "isMissing" || rule.operator === "isPresent") {
    if (typeof rule.value !== "boolean") {
      errors.push({ path, message: "Presence rules require a boolean value" });
    }
    return;
  }

  if (isEmptyValue(rule.value)) {
    errors.push({ path, message: "Value is required" });
    return;
  }

  if (field.valueType === "number" && typeof rule.value !== "number") {
    errors.push({ path, message: "Enter a valid number" });
  }

  if (
    field.valueType === "days" ||
    rule.operator === "inTheLast" ||
    rule.operator === "notInTheLast"
  ) {
    if (typeof rule.value !== "number" || rule.value <= 0) {
      errors.push({ path, message: "Enter a positive number of days" });
    }
  }

  if (
    field.valueType === "date" &&
    typeof rule.value === "string" &&
    rule.operator !== "inTheLast" &&
    rule.operator !== "notInTheLast"
  ) {
    if (!DATE_RE.test(rule.value)) {
      errors.push({ path, message: "Use YYYY-MM-DD" });
    }
  }

  if (rule.operator === "inTheRange") {
    if (!Array.isArray(rule.value) || rule.value.length !== 2) {
      errors.push({ path, message: "Range requires start and end values" });
      return;
    }
    const [start, end] = rule.value;
    if (field.valueType === "number" || field.valueType === "range-number") {
      if (typeof start !== "number" || typeof end !== "number" || start > end) {
        errors.push({ path, message: "Enter a valid numeric range" });
      }
    } else if (
      typeof start === "string" &&
      typeof end === "string" &&
      (!DATE_RE.test(start) || !DATE_RE.test(end) || start > end)
    ) {
      errors.push({ path, message: "Enter a valid date range" });
    }
  }

  if (rule.operator === "contains" && typeof rule.value === "string") {
    const wrapped = rule.value.match(/^\/(.+)\/([gimsuy]*)$/);
    const pattern = wrapped
      ? wrapped[1]
      : rule.value.startsWith("/") && rule.value.endsWith("/")
        ? rule.value.slice(1, -1)
        : null;
    if (pattern) {
      try {
        void new RegExp(pattern);
      } catch {
        errors.push({ path, message: "Invalid regex pattern" });
      }
    }
  }
}

function validateGroup(
  group: SmartPlaylistGroup,
  path: string,
  errors: SmartPlaylistValidationError[],
): void {
  if (group.rules.length === 0 && group.groups.length === 0) {
    errors.push({
      path,
      message: "Add at least one rule or nested group",
    });
    return;
  }

  group.rules.forEach((rule, index) => {
    validateRuleValue(rule, `${path}.rules[${index}]`, errors);
  });

  group.groups.forEach((child, index) => {
    validateGroup(child, `${path}.groups[${index}]`, errors);
  });
}

export function validateSmartPlaylistDraft(
  draft: SmartPlaylistDraft,
): SmartPlaylistValidationError[] {
  const errors: SmartPlaylistValidationError[] = [];

  if (!draft.name.trim()) {
    errors.push({ path: "name", message: "Playlist name is required" });
  }

  validateGroup(draft.root, "root", errors);

  if (draft.limit !== null && draft.limit <= 0) {
    errors.push({ path: "limit", message: "Limit must be greater than zero" });
  }

  if (
    draft.limitPercent !== null &&
    (draft.limitPercent < 1 || draft.limitPercent > 100)
  ) {
    errors.push({
      path: "limitPercent",
      message: "Limit percent must be between 1 and 100",
    });
  }

  if (draft.limit !== null && draft.limitPercent !== null) {
    errors.push({
      path: "limit",
      message: "Use either a fixed limit or a percentage limit, not both",
    });
  }

  if (!draft.sort.trim()) {
    errors.push({ path: "sort", message: "Choose a sort order" });
  }

  return errors;
}
