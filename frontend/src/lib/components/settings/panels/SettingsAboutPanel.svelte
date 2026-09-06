<script lang="ts">
  import { onMount } from "svelte";
  import { APP_NAME } from "$lib/brand";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { API_VERSION, CLIENT_VERSION, getCompatState } from "$lib/compat";
  import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import "$lib/settings/settings-page.css";

  let buildDate = $state("");
  let dataDir = $state("");
  let loading = $state(true);

  const compat = $derived(getCompatState());
  const capabilityCount = $derived(compat.supported.length);
  const capabilityPreview = $derived(
    compat.supported.slice(0, 8).join(", ") +
      (compat.supported.length > 8 ? "..." : ""),
  );

  function formatLocaleDate(value: string): string {
    const trimmed = value.trim();
    if (!trimmed) return "";
    const parsed = Date.parse(trimmed);
    if (Number.isNaN(parsed)) return trimmed;
    return new Date(parsed).toLocaleDateString();
  }

  const buildDateLabel = $derived(formatLocaleDate(buildDate));

  onMount(() => {
    loading = true;
    fetchWithRetry("/api/config", { headers: apiHeaders() })
      .then((r) => (r.ok ? r.json() : null))
      .then((cfg) => {
        buildDate =
          typeof cfg?.buildDate === "string" ? cfg.buildDate.trim() : "";
        dataDir = typeof cfg?.dataDir === "string" ? cfg.dataDir : "";
      })
      .finally(() => {
        loading = false;
      });
  });
</script>

<SettingsCard
  title="About"
  description={`Client and server identity for this ${APP_NAME} install.`}
>
  {#if loading}
    <div class="about-panel__loading">
      <Spinner />
    </div>
  {:else}
    <dl class="about-panel">
      <div class="about-panel__row">
        <dt>App</dt>
        <dd>{APP_NAME}</dd>
      </div>
      <div class="about-panel__row">
        <dt>Client version</dt>
        <dd>{CLIENT_VERSION}</dd>
      </div>
      <div class="about-panel__row">
        <dt>Server version</dt>
        <dd>{compat.serverVersion || "Unknown"}</dd>
      </div>
      {#if buildDateLabel}
        <div class="about-panel__row">
          <dt>Build date</dt>
          <dd>{buildDateLabel}</dd>
        </div>
      {/if}
      <div class="about-panel__row">
        <dt>API version</dt>
        <dd>{compat.apiVersion || API_VERSION}</dd>
      </div>
      <div class="about-panel__row">
        <dt>Capabilities</dt>
        <dd>
          {capabilityCount} supported
          {#if capabilityCount > 0}
            <span class="about-panel__caps">({capabilityPreview})</span>
          {/if}
        </dd>
      </div>
      {#if dataDir}
        <div class="about-panel__row">
          <dt>Data directory</dt>
          <dd><code class="settings-page__path">{dataDir}</code></dd>
        </div>
      {/if}
    </dl>
  {/if}
</SettingsCard>

<style>
  .about-panel__loading {
    display: grid;
    place-content: center;
    min-height: 6rem;
  }

  .about-panel {
    margin: 0;
    display: grid;
    gap: var(--jb-space-3);
  }

  .about-panel__row {
    display: grid;
    gap: 0.25rem;
  }

  .about-panel__row dt {
    margin: 0;
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-text-muted);
  }

  .about-panel__row dd {
    margin: 0;
    font-size: 0.9375rem;
    color: var(--jb-text);
    overflow-wrap: anywhere;
  }

  .about-panel__caps {
    display: block;
    margin-top: 0.25rem;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    line-height: 1.45;
  }
</style>
