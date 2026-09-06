// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { randomUUID } from "$lib/utils/uuid";
import type {
  SmartPlaylistDraft,
  SmartPlaylistGroup,
  SmartPlaylistRule,
} from "./types";
import { defaultOperatorForField } from "./fields";

export function newRuleId(): string {
  return randomUUID();
}

export function findGroup(
  root: SmartPlaylistGroup,
  groupId: string,
): SmartPlaylistGroup | null {
  if (root.id === groupId) return root;
  for (const child of root.groups) {
    const found = findGroup(child, groupId);
    if (found) return found;
  }
  return null;
}

export function addRule(
  root: SmartPlaylistGroup,
  groupId: string,
  field = "genre",
): SmartPlaylistGroup {
  const next = structuredClone(root);
  const group = findGroup(next, groupId);
  if (!group) return root;
  group.rules.push({
    id: newRuleId(),
    field,
    operator: defaultOperatorForField(field),
    value: "",
  });
  return next;
}

export function removeRule(
  root: SmartPlaylistGroup,
  groupId: string,
  ruleId: string,
): SmartPlaylistGroup {
  const next = structuredClone(root);
  const group = findGroup(next, groupId);
  if (!group) return root;
  group.rules = group.rules.filter((rule) => rule.id !== ruleId);
  return next;
}

export function updateRule(
  root: SmartPlaylistGroup,
  groupId: string,
  ruleId: string,
  patch: Partial<SmartPlaylistRule>,
): SmartPlaylistGroup {
  const next = structuredClone(root);
  const group = findGroup(next, groupId);
  if (!group) return root;
  const rule = group.rules.find((entry) => entry.id === ruleId);
  if (!rule) return root;
  Object.assign(rule, patch);
  return next;
}

export function addNestedGroup(
  root: SmartPlaylistGroup,
  parentId: string,
  logic: SmartPlaylistGroup["logic"] = "any",
): SmartPlaylistGroup {
  const next = structuredClone(root);
  const parent = findGroup(next, parentId);
  if (!parent) return root;
  parent.groups.push({
    id: newRuleId(),
    logic,
    rules: [
      {
        id: newRuleId(),
        field: "artist",
        operator: defaultOperatorForField("artist"),
        value: "",
      },
    ],
    groups: [],
  });
  return next;
}

export function removeGroup(
  root: SmartPlaylistGroup,
  parentId: string,
  groupId: string,
): SmartPlaylistGroup {
  const next = structuredClone(root);
  const parent = findGroup(next, parentId);
  if (!parent) return root;
  parent.groups = parent.groups.filter((group) => group.id !== groupId);
  return next;
}

export function setGroupLogic(
  root: SmartPlaylistGroup,
  groupId: string,
  logic: SmartPlaylistGroup["logic"],
): SmartPlaylistGroup {
  const next = structuredClone(root);
  const group = findGroup(next, groupId);
  if (!group) return root;
  group.logic = logic;
  return next;
}

function patchDraftRoot(
  draft: SmartPlaylistDraft,
  root: SmartPlaylistGroup,
): SmartPlaylistDraft {
  return { ...draft, root };
}

export function addDraftRule(
  draft: SmartPlaylistDraft,
  groupId: string,
  field = "genre",
): SmartPlaylistDraft {
  return patchDraftRoot(draft, addRule(draft.root, groupId, field));
}

export function removeDraftRule(
  draft: SmartPlaylistDraft,
  groupId: string,
  ruleId: string,
): SmartPlaylistDraft {
  return patchDraftRoot(draft, removeRule(draft.root, groupId, ruleId));
}

export function updateDraftRule(
  draft: SmartPlaylistDraft,
  groupId: string,
  ruleId: string,
  patch: Partial<SmartPlaylistRule>,
): SmartPlaylistDraft {
  return patchDraftRoot(draft, updateRule(draft.root, groupId, ruleId, patch));
}

export function addDraftNestedGroup(
  draft: SmartPlaylistDraft,
  parentId: string,
  logic: SmartPlaylistGroup["logic"] = "any",
): SmartPlaylistDraft {
  return patchDraftRoot(draft, addNestedGroup(draft.root, parentId, logic));
}

export function removeDraftGroup(
  draft: SmartPlaylistDraft,
  parentId: string,
  groupId: string,
): SmartPlaylistDraft {
  return patchDraftRoot(draft, removeGroup(draft.root, parentId, groupId));
}

export function setDraftGroupLogic(
  draft: SmartPlaylistDraft,
  groupId: string,
  logic: SmartPlaylistGroup["logic"],
): SmartPlaylistDraft {
  return patchDraftRoot(draft, setGroupLogic(draft.root, groupId, logic));
}
