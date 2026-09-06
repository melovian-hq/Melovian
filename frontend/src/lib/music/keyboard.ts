// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * Keyboard shortcut handling for desktop playback control. The pure
 * {@link handleKeydown} maps a keyboard event to a controls action so it can be
 * unit tested without a DOM, while {@link bindKeyboardShortcuts} wires it to the
 * window for runtime use.
 */
export interface PlaybackControls {
  togglePlay(): void;
  next(): void;
  previous(): void;
  adjustVolume(delta: number): void;
  seekBy(seconds: number): void;
}

export interface KeyboardCallbacks extends PlaybackControls {
  toggleHelp?(): void;
  closeHelp?(): void;
  togglePalette?(): void;
  closePalette?(): void;
  toggleQueue?(): void;
}

const VOLUME_STEP = 0.05;
const SEEK_STEP = 5;

function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  const tag = target.tagName;
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return true;
  return target.isContentEditable;
}

/**
 * handleKeydown returns true when the event was handled (and default behavior
 * should be prevented). Modifier-key combinations are ignored so browser and OS
 * shortcuts continue to work.
 */
export function handleKeydown(
  event: KeyboardEvent,
  controls: KeyboardCallbacks,
  options?: { helpOpen?: boolean; paletteOpen?: boolean },
): boolean {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
    controls.togglePalette?.();
    return true;
  }

  if (event.metaKey || event.ctrlKey || event.altKey) return false;
  if (isEditableTarget(event.target) && !options?.paletteOpen) return false;

  if (
    event.code === "Escape" &&
    options?.paletteOpen &&
    controls.closePalette
  ) {
    controls.closePalette();
    return true;
  }

  if (event.code === "Escape" && options?.helpOpen && controls.closeHelp) {
    controls.closeHelp();
    return true;
  }

  if (event.key === "?" && controls.toggleHelp) {
    controls.toggleHelp();
    return true;
  }

  if (event.code === "KeyQ" && controls.toggleQueue) {
    controls.toggleQueue();
    return true;
  }

  switch (event.code) {
    case "Space":
      controls.togglePlay();
      return true;
    case "ArrowUp":
      controls.adjustVolume(VOLUME_STEP);
      return true;
    case "ArrowDown":
      controls.adjustVolume(-VOLUME_STEP);
      return true;
    case "ArrowRight":
      if (event.shiftKey) controls.next();
      else controls.seekBy(SEEK_STEP);
      return true;
    case "ArrowLeft":
      if (event.shiftKey) controls.previous();
      else controls.seekBy(-SEEK_STEP);
      return true;
    default:
      return false;
  }
}

export function bindKeyboardShortcuts(
  controls: KeyboardCallbacks,
  helpOpen: () => boolean = () => false,
  paletteOpen: () => boolean = () => false,
): () => void {
  const listener = (event: KeyboardEvent) => {
    if (
      handleKeydown(event, controls, {
        helpOpen: helpOpen(),
        paletteOpen: paletteOpen(),
      })
    ) {
      event.preventDefault();
    }
  };
  window.addEventListener("keydown", listener);
  return () => window.removeEventListener("keydown", listener);
}
