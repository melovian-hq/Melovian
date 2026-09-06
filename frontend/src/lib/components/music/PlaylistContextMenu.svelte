<script lang="ts">
  import ContextMenu from "$lib/components/ui/ContextMenu.svelte";
  import { homeHidden } from "$lib/music/home-hidden.svelte";
  import {
    playPlaylistNow,
    queuePlaylist,
    savePlaylistFilesToDevice,
  } from "$lib/music/context-actions";
  import type { ContextMenuEntry } from "$lib/components/ui/context-menu";
  import type { PlaylistKind } from "$lib/music/playlist-display";
  import { toast } from "$lib/ui/toast.svelte";

  interface Props {
    playlistId: string;
    name: string;
    kind: PlaylistKind;
    x: number;
    y: number;
    onclose: () => void;
    hideable?: boolean;
    onExport?: () => void;
    onDelete?: () => void;
    onShare?: () => void;
  }

  let {
    playlistId,
    name,
    kind,
    x,
    y,
    onclose,
    hideable = false,
    onExport,
    onDelete,
    onShare,
  }: Props = $props();

  const items = $derived.by((): ContextMenuEntry[] => {
    const entries: ContextMenuEntry[] = [
      {
        id: "play",
        label: "Play now",
        icon: "play",
        onclick: () => void playPlaylistNow(kind, playlistId),
      },
      {
        id: "queue",
        label: "Add to queue",
        icon: "queueAdd",
        onclick: () => void queuePlaylist(kind, playlistId),
      },
      {
        id: "download",
        label: "Download",
        icon: "download",
        onclick: () => void savePlaylistFilesToDevice(kind, playlistId, name),
      },
    ];
    if (onShare) {
      entries.push({
        id: "share",
        label: "Share",
        icon: "share",
        onclick: onShare,
      });
    }
    if (onExport) {
      entries.push({
        id: "export",
        label: "Export M3U",
        icon: "export",
        onclick: onExport,
      });
    }
    if (hideable) {
      entries.push({ id: "hide-sep", separator: true });
      entries.push({
        id: "hide",
        label: "Hide",
        icon: "eyeOff",
        onclick: () => {
          homeHidden.hide("playlist", `${kind}:${playlistId}`);
          toast.info(`Hidden ${name} from home`);
        },
      });
    }
    if (onDelete) {
      entries.push({ id: "delete-sep", separator: true });
      entries.push({
        id: "delete",
        label: "Delete",
        icon: "trash2",
        danger: true,
        onclick: onDelete,
      });
    }
    return entries;
  });
</script>

<ContextMenu {x} {y} {onclose} label="Playlist actions" {items} />
