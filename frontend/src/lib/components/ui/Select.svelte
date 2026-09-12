<script lang="ts" generics="T extends string">
  import { Select } from "bits-ui";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";

  interface SelectOption {
    value: T;
    label: string;
    disabled?: boolean;
  }

  interface Props {
    value?: T;
    options: SelectOption[];
    placeholder?: string;
    disabled?: boolean;
    ariaLabel?: string;
    id?: string;
    class?: string;
    onchange?: (value: T) => void;
  }

  let {
    value = $bindable(),
    options,
    placeholder = "",
    disabled = false,
    ariaLabel,
    id,
    class: className = "",
    onchange,
  }: Props = $props();

  // The portaled listbox needs an accessible name too. Label it by the
  // trigger so it inherits the ariaLabel or visible value text.
  const uid = $props.id();
  const triggerId = $derived(id ?? `jb-select-trigger-${uid}`);
</script>

<Select.Root
  type="single"
  {disabled}
  items={options}
  bind:value={value as never}
  onValueChange={(next: string) => onchange?.(next as T)}
>
  <Select.Trigger
    id={triggerId}
    class="jb-select__trigger {className}"
    aria-label={ariaLabel}
  >
    <Select.Value {placeholder} class="jb-select__value" />
    <MdiIcon name="chevronDown" size={16} class="jb-select__chevron" />
  </Select.Trigger>
  <Select.Portal>
    <Select.Content
      class="jb-select__content"
      sideOffset={4}
      aria-labelledby={triggerId}
    >
      <Select.ScrollUpButton class="jb-select__scroll">
        <MdiIcon name="chevronUp" size={14} />
      </Select.ScrollUpButton>
      <Select.Viewport class="jb-select__viewport">
        {#each options as option (option.value)}
          <Select.Item
            class="jb-select__item"
            value={option.value}
            label={option.label}
            disabled={option.disabled}
          >
            {#snippet children({ selected })}
              <span class="jb-select__item-label">{option.label}</span>
              {#if selected}
                <MdiIcon name="check" size={16} class="jb-select__check" />
              {/if}
            {/snippet}
          </Select.Item>
        {/each}
      </Select.Viewport>
      <Select.ScrollDownButton class="jb-select__scroll">
        <MdiIcon name="chevronDown" size={14} />
      </Select.ScrollDownButton>
    </Select.Content>
  </Select.Portal>
</Select.Root>

<style>
  /* Bits UI renders trigger and portaled content inside its own components,
     so these rules must be global. The jb-select prefix avoids collisions. */
  :global(.jb-select__trigger) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-2);
    width: 100%;
    padding: var(--jb-space-3);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-surface);
    color: var(--jb-text);
    font: inherit;
    font-size: 0.875rem;
    text-align: left;
    cursor: pointer;
  }

  :global(.jb-select__trigger:focus-visible) {
    outline: none;
    box-shadow: var(--jb-focus-ring);
    border-color: var(--jb-accent);
  }

  :global(.jb-select__trigger[data-state="open"]) {
    border-color: var(--jb-accent);
  }

  :global(.jb-select__trigger[data-placeholder]) {
    color: var(--jb-text-subtle);
  }

  :global(.jb-select__trigger[data-disabled]) {
    opacity: 0.6;
    cursor: not-allowed;
  }

  :global(.jb-select__value) {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :global(.jb-select__chevron) {
    flex-shrink: 0;
    color: var(--jb-text-muted);
    transition: transform var(--jb-transition);
  }

  :global(.jb-select__trigger[data-state="open"] .jb-select__chevron) {
    transform: rotate(180deg);
  }

  :global(.jb-select__content) {
    z-index: calc(var(--jb-z-dialog) + 1);
    min-width: var(--bits-select-anchor-width);
    padding: var(--jb-space-1);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-bg-elevated);
    box-shadow: var(--jb-shadow-lg);
    display: flex;
    flex-direction: column;
    user-select: none;
    outline: none;
  }

  :global(.jb-select__viewport) {
    max-height: var(--bits-select-content-available-height);
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  :global(.jb-select__item) {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    width: 100%;
    padding: 0.55rem 0.75rem;
    border-radius: var(--jb-radius-sm);
    color: var(--jb-text);
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    outline: none;
  }

  :global(.jb-select__item[data-highlighted]) {
    background: var(--jb-surface-hover);
  }

  :global(.jb-select__item[data-disabled]) {
    opacity: 0.5;
    cursor: not-allowed;
  }

  :global(.jb-select__item-label) {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :global(.jb-select__check) {
    flex-shrink: 0;
    color: var(--jb-accent);
  }

  :global(.jb-select__scroll) {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--jb-space-1) 0;
    color: var(--jb-text-muted);
  }
</style>
