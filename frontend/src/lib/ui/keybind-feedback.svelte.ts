// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export const KEYBIND_FEEDBACK_STACK_MS = 900;
export const KEYBIND_FEEDBACK_HOLD_MS = 900;

export type KeybindFeedbackKind =
  "seek" | "volume" | "play" | "pause" | "next" | "prev";

export type KeybindFeedbackDirection = "forward" | "back" | "none";

export interface KeybindFeedbackPulse {
  id: number;
  kind: KeybindFeedbackKind;
  label: string;
  direction: KeybindFeedbackDirection;
  volumePct?: number;
}

export function formatSeekLabel(seconds: number): string {
  const rounded = Math.round(seconds);
  if (rounded > 0) return `+${rounded}s`;
  return `${rounded}s`;
}

export class KeybindFeedbackStore {
  current = $state<KeybindFeedbackPulse | null>(null);

  private hideTimer: ReturnType<typeof setTimeout> | null = null;
  private lastAt = 0;
  private seekTotal = 0;
  private seq = 0;

  seek(seconds: number, now = Date.now()) {
    if (!Number.isFinite(seconds) || seconds === 0) return;
    const sameDirection =
      this.current?.kind === "seek" &&
      now - this.lastAt < KEYBIND_FEEDBACK_STACK_MS &&
      Math.sign(seconds) === Math.sign(this.seekTotal);
    this.seekTotal = sameDirection ? this.seekTotal + seconds : seconds;
    this.lastAt = now;
    this.publish({
      kind: "seek",
      label: formatSeekLabel(this.seekTotal),
      direction: this.seekTotal >= 0 ? "forward" : "back",
    });
  }

  volume(level: number) {
    if (!Number.isFinite(level)) return;
    const clamped = Math.max(0, Math.min(1, level));
    this.seekTotal = 0;
    this.publish({
      kind: "volume",
      label: `${Math.round(clamped * 100)}%`,
      direction: "none",
      volumePct: Math.round(clamped * 100),
    });
  }

  playback(playing: boolean) {
    this.seekTotal = 0;
    this.publish({
      kind: playing ? "play" : "pause",
      label: playing ? "Play" : "Pause",
      direction: "none",
    });
  }

  skip(direction: "next" | "prev") {
    this.seekTotal = 0;
    this.publish({
      kind: direction,
      label: direction === "next" ? "Next" : "Previous",
      direction: direction === "next" ? "forward" : "back",
    });
  }

  clear() {
    if (this.hideTimer) {
      clearTimeout(this.hideTimer);
      this.hideTimer = null;
    }
    this.current = null;
    this.seekTotal = 0;
    this.lastAt = 0;
  }

  private publish(pulse: Omit<KeybindFeedbackPulse, "id">) {
    this.seq += 1;
    this.current = { ...pulse, id: this.seq };
    if (this.hideTimer) clearTimeout(this.hideTimer);
    this.hideTimer = setTimeout(() => {
      this.current = null;
      this.seekTotal = 0;
      this.hideTimer = null;
    }, KEYBIND_FEEDBACK_HOLD_MS);
  }
}

export const keybindFeedback = new KeybindFeedbackStore();
