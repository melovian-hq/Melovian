<script lang="ts">
  import MusicPlayerFull from "./MusicPlayerFull.svelte";
  import MusicMiniPlayer from "./MusicMiniPlayer.svelte";
  import MusicSlimPlayer from "./MusicSlimPlayer.svelte";
  import DevicesPanel from "./DevicesPanel.svelte";
  import EqFloatingPanel from "./EqFloatingPanel.svelte";
  import { music } from "$lib/config/music.svelte";
  import { layout } from "$lib/components/layout/layout.svelte";
  import { isInternetRadioTrack } from "$lib/subsonic";
  import { extensionFeatures } from "$lib/extensions/features.svelte";

  const showLyricsSidebar = $derived(
    extensionFeatures.lyrics &&
      music.lyricsOpen &&
      music.currentTrack &&
      !isInternetRadioTrack(music.currentTrack),
  );
  const tvOverlay = $derived(layout.tvMode && Boolean(music.currentTrack));
  const useSlimPlayer = $derived(layout.isMobileViewport);

  $effect(() => {
    if (!extensionFeatures.lyrics && music.lyricsOpen) {
      music.lyricsOpen = false;
    }
  });
</script>

{#if !tvOverlay}
  {#if useSlimPlayer}
    <MusicSlimPlayer />
  {:else}
    <MusicPlayerFull />
    <MusicMiniPlayer />
  {/if}
  <DevicesPanel />
  <EqFloatingPanel />
{/if}
{#if music.queueOpen && !tvOverlay}
  {@const QueuePanel = import("./MusicQueuePanel.svelte")}
  {#await QueuePanel then { default: MusicQueuePanel }}
    <MusicQueuePanel />
  {/await}
{/if}
{#if showLyricsSidebar && !tvOverlay}
  {@const LyricsPanel = import("./LyricsPanel.svelte")}
  {#await LyricsPanel then { default: LyricsPanel }}
    <LyricsPanel />
  {/await}
{/if}
