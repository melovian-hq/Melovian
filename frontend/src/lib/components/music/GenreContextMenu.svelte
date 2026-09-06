<script lang="ts">
  import ContextMenu from "$lib/components/ui/ContextMenu.svelte";
  import { router } from "$lib/router/router.svelte";
  import { music } from "$lib/config/music.svelte";
  import type { ContextMenuEntry } from "$lib/components/ui/context-menu";

  interface Props {
    name: string;
    x: number;
    y: number;
    onclose: () => void;
  }

  let { name, x, y, onclose }: Props = $props();

  const items = $derived.by((): ContextMenuEntry[] => [
    {
      id: "play",
      label: "Play now",
      icon: "play",
      onclick: () => void music.playGenre(name, false),
    },
    {
      id: "shuffle",
      label: "Shuffle",
      icon: "shuffle",
      onclick: () => void music.playGenre(name, true),
    },
    {
      id: "queue",
      label: "Add to queue",
      icon: "queueAdd",
      onclick: () => void music.addGenreToQueue(name),
    },
    {
      id: "open",
      label: "Go to genre",
      icon: "tag",
      onclick: () =>
        router.navigate(`/music/genre/${encodeURIComponent(name)}`),
    },
  ]);
</script>

<ContextMenu {x} {y} {onclose} label="Genre actions" {items} />
