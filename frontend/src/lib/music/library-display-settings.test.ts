// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it } from "vitest";
import {
  loadHideUnknownMetadata,
  saveHideUnknownMetadata,
} from "./library-display-settings";

const STORAGE_KEY = "mel-hide-unknown-metadata";

describe("library-display-settings", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it("defaults to showing unknown metadata", () => {
    expect(loadHideUnknownMetadata()).toBe(false);
  });

  it("persists the hide toggle", () => {
    saveHideUnknownMetadata(true);
    expect(localStorage.getItem(STORAGE_KEY)).toBe("true");
    expect(loadHideUnknownMetadata()).toBe(true);
    saveHideUnknownMetadata(false);
    expect(loadHideUnknownMetadata()).toBe(false);
  });
});
