<script lang="ts">
  import ContextMenu from "$lib/components/ui/ContextMenu.svelte";
  import PlaylistPicker from "./PlaylistPicker.svelte";
  import { music } from "$lib/config/music.svelte";
  import { router } from "$lib/router/router.svelte";
  import { saveTrackToDevice } from "$lib/music/context-actions";
  import { isLocalTrackId } from "$lib/music/source.svelte";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import type { ContextMenuEntry } from "$lib/components/ui/context-menu";
  import type { SubsonicSong } from "$lib/subsonic";

  interface Props {
    track: SubsonicSong;
    x: number;
    y: number;
    onclose: () => void;
    onRemove?: () => void;
  }

  let { track, x, y, onclose, onRemove }: Props = $props();

  let playlistOpen = $state(false);

  const isFavorite = $derived(music.isFavorite(track.id));
  const artistId = $derived(track.artistId ?? track.artists?.[0]?.id);

  const items = $derived.by((): ContextMenuEntry[] => {
    const entries: ContextMenuEntry[] = [
      {
        id: "play",
        label: "Play now",
        icon: "play",
        onclick: () => music.playTracks([track], 0),
      },
      {
        id: "next",
        label: "Play next",
        icon: "playNext",
        onclick: () => music.playNext(track),
      },
      {
        id: "queue",
        label: "Add to queue",
        icon: "queueAdd",
        onclick: () => music.addToQueue(track),
      },
      {
        id: "playlist",
        label: "Add to playlist",
        icon: "listMusic",
        keepOpen: true,
        onclick: () => {
          playlistOpen = true;
        },
      },
      {
        id: "favorite",
        label: isFavorite ? "Remove favorite" : "Add favorite",
        icon: "star",
        onclick: () => void music.toggleFavorite(track),
      },
      {
        id: "download",
        label: "Download",
        icon: "download",
        onclick: () => void saveTrackToDevice(track),
      },
    ];
    if (track.albumId) {
      entries.push({
        id: "album",
        label: "Go to album",
        icon: "album",
        onclick: () => router.navigate(`/music/album/${track.albumId}`),
      });
    }
    if (artistId) {
      entries.push({
        id: "artist",
        label: "Go to artist",
        icon: "accountMusic",
        onclick: () => router.navigate(`/music/artist/${artistId}`),
      });
    }
    if (isLocalTrackId(track.id) && extensionFeatures.metadata) {
      entries.push({
        id: "metadata",
        label: "Edit metadata",
        icon: "tag",
        onclick: () =>
          router.navigate(
            `/music/metadata?track=${encodeURIComponent(track.id)}`,
          ),
      });
    }
    if (onRemove) {
      entries.push({ id: "remove-sep", separator: true });
      entries.push({
        id: "remove",
        label: "Remove",
        icon: "x",
        danger: true,
        onclick: onRemove,
      });
    }
    return entries;
  });
</script>

{#if playlistOpen}
  <PlaylistPicker
    open
    tracks={[track]}
    onclose={() => {
      playlistOpen = false;
      onclose();
    }}
  />
{:else}
  <ContextMenu {x} {y} {onclose} label="Track actions" {items} />
{/if}
