// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

type SaveStatusState = "idle" | "saving" | "saved" | "error";

const SAVED_VISIBLE_MS = 1500;

/**
 * Tracks the save lifecycle of a settings card so the inline
 * SettingsSaveStatus indicator can show Saving, Saved, or the concrete
 * failure. The Saved state clears itself after a short flash.
 */
export class SaveStatus {
  state = $state<SaveStatusState>("idle");
  error = $state("");

  private timer: ReturnType<typeof setTimeout> | undefined;

  begin() {
    this.clearTimer();
    this.error = "";
    this.state = "saving";
  }

  saved() {
    this.clearTimer();
    this.error = "";
    this.state = "saved";
    this.timer = setTimeout(() => {
      if (this.state === "saved") this.state = "idle";
    }, SAVED_VISIBLE_MS);
  }

  failed(message: string) {
    this.clearTimer();
    this.state = "error";
    this.error = message;
  }

  private clearTimer() {
    if (this.timer === undefined) return;
    clearTimeout(this.timer);
    this.timer = undefined;
  }
}

export function createSaveStatus(): SaveStatus {
  return new SaveStatus();
}

/**
 * Runs a settings save against a SaveStatus: begin, saved on success,
 * failed with the message on error. Returns whether the save succeeded.
 * onFailure runs after failed() with the resolved message so callers can
 * toast or log.
 */
export async function saveWithStatus(
  status: SaveStatus,
  failureMessage: string | ((err: unknown) => string),
  work: () => void | Promise<void>,
  onFailure?: (message: string) => void,
): Promise<boolean> {
  status.begin();
  try {
    await work();
    status.saved();
    return true;
  } catch (err) {
    const message =
      typeof failureMessage === "function"
        ? failureMessage(err)
        : failureMessage;
    status.failed(message);
    onFailure?.(message);
    return false;
  }
}
