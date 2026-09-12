// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { createEmptySmartPlaylistDraft } from "./compile";
import { validateSmartPlaylistDraft } from "./validate";
import type { SmartPlaylistDraft } from "./types";

interface SmartPlaylistEditorCallbacks {
  onclose?: () => void;
  oncreate?: (draft: SmartPlaylistDraft) => Promise<void>;
}

export class SmartPlaylistEditor {
  draft = $state(createEmptySmartPlaylistDraft());
  submitting = $state(false);
  submitError = $state("");

  errorByPath = $derived(
    Object.fromEntries(
      validateSmartPlaylistDraft(this.draft).map((error) => [
        error.path,
        error.message,
      ]),
    ),
  );

  constructor(private readonly callbacks: SmartPlaylistEditorCallbacks) {}

  reset() {
    this.draft = createEmptySmartPlaylistDraft();
    this.submitError = "";
  }

  close() {
    this.reset();
    this.callbacks.onclose?.();
  }

  async submit() {
    this.submitError = "";
    const errors = validateSmartPlaylistDraft(this.draft);
    if (errors.length > 0) {
      this.submitError = errors[0]?.message ?? "Fix the highlighted fields";
      return;
    }
    this.submitting = true;
    try {
      await this.callbacks.oncreate?.(this.draft);
      this.close();
    } catch (err) {
      this.submitError =
        err instanceof Error ? err.message : "Failed to create smart playlist";
    } finally {
      this.submitting = false;
    }
  }
}
