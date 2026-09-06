<script lang="ts">
  interface Props {
    done: number;
    total: number;
    label?: string;
  }

  let { done, total, label = "Working" }: Props = $props();

  const pct = $derived(
    total > 0 ? Math.min(100, Math.round((done / total) * 100)) : 0,
  );
</script>

<div class="batch-progress" role="status" aria-live="polite">
  <div class="batch-progress__head">
    <span>{label}</span>
    <span>{done.toLocaleString()} / {total.toLocaleString()}</span>
  </div>
  <div class="batch-progress__track" aria-hidden="true">
    <div class="batch-progress__fill" style:width="{pct}%"></div>
  </div>
</div>

<style>
  .batch-progress {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
    padding: var(--jb-space-3);
    border-radius: var(--jb-radius-md);
    border: 1px solid var(--jb-border);
    background: var(--jb-bg-muted);
  }

  .batch-progress__head {
    display: flex;
    justify-content: space-between;
    gap: var(--jb-space-2);
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-text-muted);
  }

  .batch-progress__track {
    height: 0.375rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-border);
    overflow: hidden;
  }

  .batch-progress__fill {
    height: 100%;
    border-radius: inherit;
    background: var(--jb-accent);
    transition: width 0.2s ease;
  }
</style>
