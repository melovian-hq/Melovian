// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { SubsonicSong } from "$lib/subsonic/types";

class TrackSelectionStore {
  active = $state(false);
  selected = $state.raw<Map<string, SubsonicSong>>(new Map());

  count = $derived(this.selected.size);
  tracks = $derived([...this.selected.values()]);

  toggle(track: SubsonicSong) {
    const next = new Map(this.selected);
    if (next.has(track.id)) next.delete(track.id);
    else next.set(track.id, track);
    this.selected = next;
  }

  select(track: SubsonicSong) {
    if (this.selected.has(track.id)) return;
    const next = new Map(this.selected);
    next.set(track.id, track);
    this.selected = next;
  }

  isSelected(trackId: string) {
    return this.selected.has(trackId);
  }

  selectAll(tracks: SubsonicSong[]) {
    const next = new Map(this.selected);
    for (const track of tracks) next.set(track.id, track);
    this.selected = next;
  }

  clear() {
    this.selected = new Map();
    this.active = false;
  }

  enable() {
    this.active = true;
  }
}

export const trackSelection = new TrackSelectionStore();
