// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { getSmartField } from "./fields";
import type { SmartPlaylistGroup, SmartPlaylistRule } from "./types";
import type { SmartTrackContext } from "./context";

function parseRegexValue(value: string): string {
  const wrapped = value.match(/^\/(.+)\/([gimsuy]*)$/);
  if (wrapped) return wrapped[1] ?? value;
  if (value.startsWith("/") && value.endsWith("/") && value.length > 2) {
    return value.slice(1, -1);
  }
  return value;
}

function stringValue(track: SmartTrackContext, fieldId: string): string {
  const value = track[fieldId as keyof SmartTrackContext];
  if (value === null || value === undefined) return "";
  return String(value);
}

function numberValue(track: SmartTrackContext, fieldId: string): number | null {
  const value = track[fieldId as keyof SmartTrackContext];
  if (typeof value === "number" && Number.isFinite(value)) return value;
  return null;
}

function boolValue(track: SmartTrackContext, fieldId: string): boolean | null {
  const value = track[fieldId as keyof SmartTrackContext];
  return typeof value === "boolean" ? value : null;
}

function compareText(
  haystack: string,
  needle: string,
  operator: string,
): boolean {
  const left = haystack.toLowerCase();
  const right = needle.toLowerCase();
  switch (operator) {
    case "is":
      return left === right;
    case "isNot":
      return left !== right;
    case "contains":
      return left.includes(right);
    case "notContains":
      return !left.includes(right);
    case "startsWith":
      return left.startsWith(right);
    case "endsWith":
      return left.endsWith(right);
    default:
      return false;
  }
}

function compareRegex(haystack: string, pattern: string): boolean {
  try {
    return new RegExp(pattern, "i").test(haystack);
  } catch {
    return haystack.toLowerCase().includes(pattern.toLowerCase());
  }
}

function daysAgo(dateValue: string): number | null {
  if (!dateValue) return null;
  const parsed = Date.parse(dateValue);
  if (Number.isNaN(parsed)) return null;
  return Math.floor((Date.now() - parsed) / 86_400_000);
}

function evaluateRule(
  track: SmartTrackContext,
  rule: SmartPlaylistRule,
): boolean {
  const field = getSmartField(rule.field);
  if (!field) return false;

  const op = rule.operator;

  if (op === "isMissing" || op === "isPresent") {
    const present = stringValue(track, rule.field) !== "";
    const wantsPresent = rule.value === true;
    return op === "isPresent"
      ? present === wantsPresent
      : present !== wantsPresent;
  }

  if (field.valueType === "boolean") {
    const actual = boolValue(track, rule.field);
    if (actual === null) return false;
    const expected = rule.value === true;
    return op === "is" ? actual === expected : actual !== expected;
  }

  if (op === "inTheLast" || op === "notInTheLast") {
    const days =
      typeof rule.value === "number" ? rule.value : Number(rule.value);
    if (!Number.isFinite(days) || days <= 0) return false;
    const elapsed = daysAgo(stringValue(track, rule.field));
    if (elapsed === null) return false;
    const matched = elapsed <= days;
    return op === "inTheLast" ? matched : !matched;
  }

  if (op === "inTheRange") {
    if (!Array.isArray(rule.value) || rule.value.length !== 2) return false;
    const [start, end] = rule.value;
    if (field.valueType === "number" || field.valueType === "range-number") {
      const value = numberValue(track, rule.field);
      if (value === null) return false;
      return (
        typeof start === "number" &&
        typeof end === "number" &&
        value >= start &&
        value <= end
      );
    }
    const value = stringValue(track, rule.field);
    return (
      typeof start === "string" &&
      typeof end === "string" &&
      value >= start &&
      value <= end
    );
  }

  if (field.valueType === "number" || field.valueType === "days") {
    const value = numberValue(track, rule.field);
    if (value === null) return false;
    const expected =
      typeof rule.value === "number" ? rule.value : Number(rule.value);
    if (!Number.isFinite(expected)) return false;
    if (op === "is") return value === expected;
    if (op === "isNot") return value !== expected;
    if (op === "gt") return value > expected;
    if (op === "lt") return value < expected;
    return false;
  }

  if (op === "before" || op === "after") {
    const value = stringValue(track, rule.field);
    const expected = String(rule.value ?? "");
    if (!value || !expected) return false;
    return op === "before" ? value < expected : value > expected;
  }

  const text = stringValue(track, rule.field);
  const rawNeedle = String(rule.value ?? "");
  if (op === "contains" && rawNeedle.match(/^\/(.+)\/([gimsuy]*)$/)) {
    return compareRegex(text, parseRegexValue(rawNeedle));
  }
  return compareText(text, rawNeedle, op);
}

function evaluateGroup(
  track: SmartTrackContext,
  group: SmartPlaylistGroup,
): boolean {
  const results = [
    ...group.rules.map((rule) => evaluateRule(track, rule)),
    ...group.groups.map((child) => evaluateGroup(track, child)),
  ];
  if (results.length === 0) return false;
  return group.logic === "all" ? results.every(Boolean) : results.some(Boolean);
}

export function trackMatchesSmartPlaylist(
  track: SmartTrackContext,
  root: SmartPlaylistGroup,
): boolean {
  return evaluateGroup(track, root);
}
