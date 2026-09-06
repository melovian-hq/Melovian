<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import { sources } from "$lib/features/sources/store.svelte";
  import { music } from "$lib/config/music.svelte";
  import {
    detectServerKind,
    serverIconPath,
    type ServerKind,
    type SourceKind,
  } from "$lib/icons/server-kind";

  interface Props {
    kind?: SourceKind | "auto" | "server";
    serverName?: string;
    version?: string;
    size?: number;
    class?: string;
    "aria-hidden"?: boolean | "true" | "false";
  }

  let {
    kind = "auto",
    serverName,
    version,
    size = 20,
    class: className = "",
    "aria-hidden": ariaHidden = true,
  }: Props = $props();

  // Prefer live ping identity so a custom instance label does not hide Navidrome.
  const activeServerName = $derived(
    serverName ?? music.serverName ?? instances.active?.serverName,
  );
  const activeVersion = $derived(version ?? music.status.version);

  const resolvedKind = $derived.by((): SourceKind => {
    if (kind === "server") {
      return detectServerKind(activeServerName, activeVersion);
    }
    if (kind !== "auto") {
      return kind;
    }
    if (sources.hasUnifiedMode) {
      return "unified";
    }
    if (sources.hasLocalActive && !sources.hasSubsonicActive) {
      return "local";
    }
    return detectServerKind(activeServerName, activeVersion);
  });

  const serverKind = $derived.by((): ServerKind => {
    if (resolvedKind === "navidrome" || resolvedKind === "subsonic") {
      return resolvedKind;
    }
    return detectServerKind(activeServerName, activeVersion);
  });

  const iconSrc = $derived(serverIconPath(serverKind));
</script>

{#if resolvedKind === "local"}
  <MdiIcon
    name="folderOpen"
    {size}
    class={className}
    aria-hidden={ariaHidden}
  />
{:else if resolvedKind === "unified"}
  <MdiIcon name="album" {size} class={className} aria-hidden={ariaHidden} />
{:else}
  {#key iconSrc}
    <img
      src={iconSrc}
      alt=""
      width={size}
      height={size}
      class="source-icon {className}"
      aria-hidden={ariaHidden}
    />
  {/key}
{/if}

<style>
  .source-icon {
    display: block;
    flex-shrink: 0;
    object-fit: contain;
  }
</style>
