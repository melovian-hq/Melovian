// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { router } from "$lib/router/router.svelte";
import { music } from "$lib/config/music.svelte";
import { layout } from "$lib/components/layout/layout.svelte";
import { theme } from "$lib/theme/theme.svelte";
import { keyboardHelp } from "$lib/ui/keyboard-help.svelte";
import { commandPalette } from "$lib/ui/command-palette.svelte";
import { extensionFeatures } from "$lib/extensions/features.svelte";

export interface PaletteCommand {
  id: string;
  label: string;
  group: string;
  keywords?: string[];
  icon?: string;
  run: () => void | Promise<void>;
}

const NAV_COMMANDS: PaletteCommand[] = [
  {
    id: "nav-home",
    label: "Go to Home",
    group: "Navigate",
    icon: "music",
    keywords: ["browse"],
    run: () => router.navigate("/music"),
  },
  {
    id: "nav-search",
    label: "Go to Search",
    group: "Navigate",
    icon: "search",
    run: () => router.navigate("/music/search"),
  },
  {
    id: "nav-artists",
    label: "Go to Artists",
    group: "Navigate",
    icon: "accountMusic",
    run: () => router.navigate("/music/artists"),
  },
  {
    id: "nav-albums",
    label: "Go to Albums",
    group: "Navigate",
    icon: "album",
    run: () => router.navigate("/music/albums"),
  },
  {
    id: "nav-genres",
    label: "Go to Genres",
    group: "Navigate",
    icon: "tag",
    run: () => router.navigate("/music/genres"),
  },
  {
    id: "nav-playlists",
    label: "Go to Playlists",
    group: "Navigate",
    icon: "listMusic",
    run: () => router.navigate("/music/playlists"),
  },
  {
    id: "nav-favorites",
    label: "Go to Favorites",
    group: "Navigate",
    icon: "star",
    run: () => router.navigate("/music/favorites"),
  },
  {
    id: "nav-now-playing",
    label: "Go to Now Playing",
    group: "Navigate",
    icon: "disc",
    keywords: ["queue", "lyrics", "artwork"],
    run: () => router.navigate("/music/now-playing"),
  },
  {
    id: "tv-mode",
    label: "TV / fullscreen now playing",
    group: "Playback",
    icon: "fullscreen",
    keywords: ["tv", "cinema", "fullscreen"],
    run: () => {
      router.navigate("/music/now-playing");
      if (music.currentTrack) layout.enterTvMode();
    },
  },
  {
    id: "nav-lyrics",
    label: "Go to Lyrics",
    group: "Navigate",
    icon: "lyrics",
    run: () => router.navigate("/music/lyrics"),
  },
  {
    id: "nav-settings",
    label: "Go to Settings",
    group: "Navigate",
    icon: "settings",
    run: () => router.navigate("/settings/profile"),
  },
];

const ACTION_COMMANDS: PaletteCommand[] = [
  {
    id: "play-pause",
    label: "Play / Pause",
    group: "Playback",
    icon: "play",
    keywords: ["toggle"],
    run: () => void music.togglePlay(),
  },
  {
    id: "next-track",
    label: "Next track",
    group: "Playback",
    icon: "skipForward",
    run: () => music.next(),
  },
  {
    id: "prev-track",
    label: "Previous track",
    group: "Playback",
    icon: "skipBack",
    run: () => music.previous(),
  },
  {
    id: "toggle-queue",
    label: "Toggle queue",
    group: "Playback",
    icon: "listMusic",
    run: () => music.toggleQueue(),
  },
  {
    id: "add-current-to-queue",
    label: "Add current track to queue",
    group: "Playback",
    icon: "queueAdd",
    keywords: ["queue", "append"],
    run: () => {
      const track = music.currentTrack;
      if (!track) return;
      music.addToQueue(track);
    },
  },
  {
    id: "play-current-next",
    label: "Play current track next",
    group: "Playback",
    icon: "playNext",
    keywords: ["queue", "next"],
    run: () => {
      const track = music.currentTrack;
      if (!track) return;
      music.playNext(track);
    },
  },
  {
    id: "theme-light",
    label: "Use light theme",
    group: "Appearance",
    icon: "sun",
    run: () => theme.setMode("light"),
  },
  {
    id: "theme-dark",
    label: "Use dark theme",
    group: "Appearance",
    icon: "moon",
    run: () => theme.setMode("dark"),
  },
  {
    id: "theme-system",
    label: "Use system theme",
    group: "Appearance",
    icon: "monitor",
    run: () => theme.setMode("system"),
  },
  {
    id: "keyboard-help",
    label: "Show keyboard shortcuts",
    group: "Help",
    icon: "slidersHorizontal",
    keywords: ["shortcuts", "help"],
    run: () => {
      commandPalette.close();
      keyboardHelp.toggle();
    },
  },
];

export function basePaletteCommands(): PaletteCommand[] {
  const nav = NAV_COMMANDS.filter((command) => {
    if (command.id === "nav-lyrics") return extensionFeatures.lyrics;
    return true;
  });
  return [...nav, ...ACTION_COMMANDS];
}

function normalize(value: string): string {
  return value.trim().toLowerCase();
}

export function scorePaletteCommand(
  command: PaletteCommand,
  query: string,
): number {
  const q = normalize(query);
  if (!q) return 1;

  const haystack = [command.label, command.group, ...(command.keywords ?? [])]
    .join(" ")
    .toLowerCase();

  if (haystack.startsWith(q)) return 100;
  if (command.label.toLowerCase().includes(q)) return 80;
  if (haystack.includes(q)) return 60;
  return 0;
}

export function filterPaletteCommands(
  commands: readonly PaletteCommand[],
  query: string,
): PaletteCommand[] {
  const q = normalize(query);
  if (!q) return [...commands];

  return commands
    .map((command) => ({ command, score: scorePaletteCommand(command, q) }))
    .filter((entry) => entry.score > 0)
    .sort(
      (a, b) =>
        b.score - a.score || a.command.label.localeCompare(b.command.label),
    )
    .map((entry) => entry.command);
}

export function searchPaletteCommands(query: string): PaletteCommand[] {
  return filterPaletteCommands(basePaletteCommands(), query);
}

export async function searchPaletteWithMusic(
  query: string,
): Promise<PaletteCommand[]> {
  const trimmed = query.trim();
  const staticMatches = searchPaletteCommands(trimmed);
  if (!trimmed || !music.connected) return staticMatches;

  const result = await music.search(trimmed);
  const dynamic: PaletteCommand[] = [];

  for (const artist of result.artists.slice(0, 5)) {
    dynamic.push({
      id: `artist-${artist.id}`,
      label: artist.name,
      group: "Artists",
      icon: "accountMusic",
      run: () => router.navigate(`/music/artist/${artist.id}`),
    });
  }

  for (const album of result.albums.slice(0, 5)) {
    dynamic.push({
      id: `album-${album.id}`,
      label: album.name,
      group: "Albums",
      icon: "album",
      run: () => router.navigate(`/music/album/${album.id}`),
    });
  }

  for (const song of result.songs.slice(0, 5)) {
    dynamic.push({
      id: `song-${song.id}`,
      label: `${song.title} · ${song.artist ?? "Unknown"}`,
      group: "Tracks",
      icon: "music",
      run: () => void music.playTrackById(song.id),
    });
  }

  return [...dynamic, ...staticMatches];
}
