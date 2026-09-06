<script lang="ts">
  import DeviceAvatar from "$lib/components/ui/DeviceAvatar.svelte";

  interface Member {
    seed: string;
    label: string;
    self?: boolean;
    active?: boolean;
  }

  interface Props {
    members: Member[];
    size?: number;
    max?: number;
    class?: string;
    overlap?: number;
    showTitles?: boolean;
  }

  let {
    members,
    size = 22,
    max = 3,
    class: className = "",
    overlap,
    showTitles = false,
  }: Props = $props();

  const overlapPx = $derived(overlap ?? Math.round(size * 0.35));
  const visible = $derived(members.slice(0, max));
  const overflow = $derived(Math.max(0, members.length - max));
</script>

<span
  class={["avatar-stack", className]}
  style:--avatar-stack-overlap="-{overlapPx}px"
  aria-hidden={showTitles ? undefined : "true"}
>
  {#each visible as member, index (member.seed)}
    <span class="avatar-stack__item" style:z-index={index + 1}>
      <DeviceAvatar
        seed={member.seed}
        label={member.label}
        self={member.self}
        active={member.active}
        {size}
        interactive={false}
      />
    </span>
  {/each}
  {#if overflow > 0}
    <span
      class="avatar-stack__more"
      style:z-index={visible.length + 1}
      style:width="{size}px"
      style:height="{size}px"
      style:font-size="{Math.max(9, Math.round(size * 0.45))}px"
      title={showTitles ? `+${overflow} more` : undefined}
    >
      +{overflow}
    </span>
  {/if}
</span>

<style>
  .avatar-stack {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
    --avatar-stack-ring: var(--jb-bg-elevated);
  }

  .avatar-stack__item {
    display: inline-flex;
    position: relative;
  }

  .avatar-stack__item:not(:first-child),
  .avatar-stack__more {
    margin-inline-start: var(--avatar-stack-overlap);
  }

  .avatar-stack__more {
    display: inline-grid;
    place-content: center;
    position: relative;
    border-radius: var(--jb-radius-md);
    background: color-mix(
      in srgb,
      var(--jb-surface) 88%,
      var(--jb-accent-muted)
    );
    color: var(--jb-text-muted);
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    line-height: 1;
    box-shadow: 0 0 0 2px var(--avatar-stack-ring, var(--jb-bg-elevated));
    flex-shrink: 0;
  }
</style>
