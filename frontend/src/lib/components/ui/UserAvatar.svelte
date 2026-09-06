<script lang="ts">
  import PixelSmile from "$lib/components/ui/PixelSmile.svelte";
  import { profile } from "$lib/profile/profile.svelte";

  interface Props {
    seed: string;
    size?: number;
    class?: string;
    interactive?: boolean;
  }

  let {
    seed,
    size = 32,
    class: className = "",
    interactive = true,
  }: Props = $props();

  const variant = $derived(profile.smileVariants[seed] ?? 0);

  function handleVariantChange(next: number) {
    profile.setVariant(seed, next);
  }
</script>

{#if profile.customAvatarUrl}
  <img
    src={profile.customAvatarUrl}
    alt="Profile avatar"
    class="user-avatar user-avatar--custom {className}"
    style:width="{size}px"
    style:height="{size}px"
  />
{:else}
  <PixelSmile
    {seed}
    {size}
    class={className}
    {variant}
    {interactive}
    onvariantchange={interactive ? handleVariantChange : undefined}
  />
{/if}

<style>
  .user-avatar {
    display: block;
    flex-shrink: 0;
    border-radius: var(--jb-radius-md);
    object-fit: cover;
    box-shadow: var(--jb-shadow-sm);
  }
</style>
