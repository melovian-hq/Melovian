// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export type SmartFieldValueType =
  | "string"
  | "number"
  | "boolean"
  | "date"
  | "days"
  | "range-number"
  | "range-date";

export interface SmartFieldDefinition {
  id: string;
  label: string;
  valueType: SmartFieldValueType;
  navidromeField: string;
  operators: string[];
}

export const SMART_PLAYLIST_OPERATORS: Record<
  string,
  { label: string; valueTypes: SmartFieldValueType[] }
> = {
  is: { label: "is", valueTypes: ["string", "number", "boolean", "date"] },
  isNot: {
    label: "is not",
    valueTypes: ["string", "number", "boolean", "date"],
  },
  gt: { label: "greater than", valueTypes: ["number", "date"] },
  lt: { label: "less than", valueTypes: ["number", "date"] },
  contains: { label: "contains", valueTypes: ["string"] },
  notContains: { label: "does not contain", valueTypes: ["string"] },
  startsWith: { label: "starts with", valueTypes: ["string"] },
  endsWith: { label: "ends with", valueTypes: ["string"] },
  inTheRange: {
    label: "in range",
    valueTypes: ["range-number", "range-date"],
  },
  before: { label: "before", valueTypes: ["date"] },
  after: { label: "after", valueTypes: ["date"] },
  inTheLast: { label: "in the last (days)", valueTypes: ["days"] },
  notInTheLast: {
    label: "not in the last (days)",
    valueTypes: ["days"],
  },
  isMissing: { label: "is missing", valueTypes: ["boolean"] },
  isPresent: { label: "is present", valueTypes: ["boolean"] },
};

export const SMART_PLAYLIST_FIELDS: SmartFieldDefinition[] = [
  {
    id: "title",
    label: "Title",
    valueType: "string",
    navidromeField: "title",
    operators: [
      "is",
      "isNot",
      "contains",
      "notContains",
      "startsWith",
      "endsWith",
    ],
  },
  {
    id: "artist",
    label: "Artist",
    valueType: "string",
    navidromeField: "artist",
    operators: [
      "is",
      "isNot",
      "contains",
      "notContains",
      "startsWith",
      "endsWith",
    ],
  },
  {
    id: "album",
    label: "Album",
    valueType: "string",
    navidromeField: "album",
    operators: [
      "is",
      "isNot",
      "contains",
      "notContains",
      "startsWith",
      "endsWith",
    ],
  },
  {
    id: "albumartist",
    label: "Album artist",
    valueType: "string",
    navidromeField: "albumartist",
    operators: [
      "is",
      "isNot",
      "contains",
      "notContains",
      "startsWith",
      "endsWith",
    ],
  },
  {
    id: "genre",
    label: "Genre",
    valueType: "string",
    navidromeField: "genre",
    operators: ["is", "isNot", "contains", "notContains"],
  },
  {
    id: "language",
    label: "Language",
    valueType: "string",
    navidromeField: "language",
    operators: [
      "is",
      "isNot",
      "contains",
      "notContains",
      "isMissing",
      "isPresent",
    ],
  },
  {
    id: "comment",
    label: "Comment",
    valueType: "string",
    navidromeField: "comment",
    operators: ["contains", "notContains", "startsWith", "endsWith"],
  },
  {
    id: "filepath",
    label: "File path",
    valueType: "string",
    navidromeField: "filepath",
    operators: ["contains", "notContains", "startsWith", "endsWith"],
  },
  {
    id: "codec",
    label: "Codec",
    valueType: "string",
    navidromeField: "codec",
    operators: ["is", "isNot", "contains"],
  },
  {
    id: "filetype",
    label: "File type",
    valueType: "string",
    navidromeField: "filetype",
    operators: ["is", "isNot", "contains"],
  },
  {
    id: "year",
    label: "Year",
    valueType: "number",
    navidromeField: "year",
    operators: ["is", "isNot", "gt", "lt", "inTheRange"],
  },
  {
    id: "date",
    label: "Recording date",
    valueType: "date",
    navidromeField: "date",
    operators: [
      "is",
      "before",
      "after",
      "inTheRange",
      "inTheLast",
      "notInTheLast",
    ],
  },
  {
    id: "dateadded",
    label: "Date added",
    valueType: "date",
    navidromeField: "dateadded",
    operators: ["before", "after", "inTheRange", "inTheLast", "notInTheLast"],
  },
  {
    id: "releasedate",
    label: "Release date",
    valueType: "date",
    navidromeField: "releasedate",
    operators: ["before", "after", "inTheRange", "inTheLast", "notInTheLast"],
  },
  {
    id: "lastplayed",
    label: "Last played",
    valueType: "date",
    navidromeField: "lastplayed",
    operators: ["before", "after", "inTheRange", "inTheLast", "notInTheLast"],
  },
  {
    id: "bitrate",
    label: "Bitrate (kbps)",
    valueType: "number",
    navidromeField: "bitrate",
    operators: ["is", "gt", "lt", "inTheRange"],
  },
  {
    id: "bitdepth",
    label: "Bit depth",
    valueType: "number",
    navidromeField: "bitdepth",
    operators: ["is", "gt", "lt", "inTheRange"],
  },
  {
    id: "samplerate",
    label: "Sample rate",
    valueType: "number",
    navidromeField: "samplerate",
    operators: ["is", "gt", "lt", "inTheRange"],
  },
  {
    id: "duration",
    label: "Duration (sec)",
    valueType: "number",
    navidromeField: "duration",
    operators: ["gt", "lt", "inTheRange"],
  },
  {
    id: "bpm",
    label: "BPM",
    valueType: "number",
    navidromeField: "bpm",
    operators: ["is", "gt", "lt", "inTheRange"],
  },
  {
    id: "rating",
    label: "Rating",
    valueType: "number",
    navidromeField: "rating",
    operators: ["is", "isNot", "gt", "lt", "inTheRange"],
  },
  {
    id: "playcount",
    label: "Play count",
    valueType: "number",
    navidromeField: "playcount",
    operators: ["is", "gt", "lt", "inTheRange"],
  },
  {
    id: "loved",
    label: "Loved",
    valueType: "boolean",
    navidromeField: "loved",
    operators: ["is", "isNot"],
  },
  {
    id: "compilation",
    label: "Compilation",
    valueType: "boolean",
    navidromeField: "compilation",
    operators: ["is", "isNot"],
  },
  {
    id: "missing",
    label: "File missing",
    valueType: "boolean",
    navidromeField: "missing",
    operators: ["is", "isNot"],
  },
];

const fieldById = new Map(
  SMART_PLAYLIST_FIELDS.map((field) => [field.id, field]),
);

export function getSmartField(id: string): SmartFieldDefinition | undefined {
  return fieldById.get(id);
}

export function operatorsForField(fieldId: string): string[] {
  return getSmartField(fieldId)?.operators ?? ["contains"];
}

export function defaultOperatorForField(fieldId: string): string {
  return operatorsForField(fieldId)[0] ?? "contains";
}

export const SMART_PLAYLIST_SORT_OPTIONS = [
  { id: "+random", label: "Random" },
  { id: "+title", label: "Title (A-Z)" },
  { id: "-title", label: "Title (Z-A)" },
  { id: "+artist", label: "Artist (A-Z)" },
  { id: "-year", label: "Year (newest)" },
  { id: "+year", label: "Year (oldest)" },
  { id: "-rating", label: "Rating (high)" },
  { id: "-playcount", label: "Most played" },
  { id: "-lastplayed", label: "Recently played" },
  { id: "+dateadded", label: "Recently added" },
] as const;
