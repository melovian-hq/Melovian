<script lang="ts">
  import { onMount } from "svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import Toggle from "$lib/components/ui/Toggle.svelte";
  import { loadExtensions } from "$lib/extensions/registry";
  import { APP_NAME, EXTENSION_MANIFEST } from "$lib/brand";
  import {
    fetchExtensions,
    installExtension,
    reinstallExtension,
    setExtensionEnabled,
    uninstallExtension,
    type ExtensionListItem,
  } from "$lib/extensions/api";
  import { toast } from "$lib/ui/toast.svelte";

  let items = $state<ExtensionListItem[]>([]);
  let extensionsDir = $state("");
  let loading = $state(true);
  let loadError = $state<string | null>(null);
  let busyId = $state<string | null>(null);
  let uploading = $state(false);
  let fileInput = $state<HTMLInputElement | undefined>();
  let dragOver = $state(false);

  async function applyPayload(payload: {
    items: ExtensionListItem[];
    dir: string;
  }) {
    items = payload.items;
    extensionsDir = payload.dir;
    await loadExtensions();
  }

  async function refresh() {
    loading = true;
    loadError = null;
    try {
      const payload = await fetchExtensions();
      items = payload.items;
      extensionsDir = payload.dir;
    } catch (err) {
      items = [];
      extensionsDir = "";
      loadError =
        err instanceof Error ? err.message : "Could not load extensions";
    } finally {
      loading = false;
    }
  }

  async function toggle(item: ExtensionListItem, enabled: boolean) {
    busyId = item.id;
    try {
      const payload = await setExtensionEnabled(item.id, enabled);
      await applyPayload(payload);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not update extension",
      );
    } finally {
      busyId = null;
    }
  }

  async function handleInstallBundled(item: ExtensionListItem) {
    busyId = item.id;
    try {
      const payload = await reinstallExtension(item.id);
      await applyPayload(payload);
      toast.success(
        item.installed ? `Reinstalled ${item.name}` : `Installed ${item.name}`,
      );
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not install extension",
      );
    } finally {
      busyId = null;
    }
  }

  async function handleUninstall(item: ExtensionListItem) {
    busyId = item.id;
    try {
      const payload = await uninstallExtension(item.id);
      await applyPayload(payload);
      toast.success(`Uninstalled ${item.name}`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not uninstall extension",
      );
    } finally {
      busyId = null;
    }
  }

  function isZipPackage(file: File): boolean {
    const name = file.name.toLowerCase();
    return (
      name.endsWith(".zip") ||
      file.type === "application/zip" ||
      file.type === "application/x-zip-compressed"
    );
  }

  function isLoneWasm(file: File): boolean {
    const name = file.name.toLowerCase();
    return name.endsWith(".wasm") || file.type === "application/wasm";
  }

  async function installPackageFile(file: File) {
    if (isLoneWasm(file)) {
      toast.error(
        `Upload a .zip package that includes the .wasm and ${EXTENSION_MANIFEST}`,
      );
      return;
    }
    if (!isZipPackage(file)) {
      toast.error("Upload a .zip extension package");
      return;
    }

    uploading = true;
    try {
      const payload = await installExtension(file);
      await applyPayload(payload);
      toast.success("Extension installed");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not install extension",
      );
    } finally {
      uploading = false;
    }
  }

  async function handleUpload(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    input.value = "";
    if (!file) return;
    await installPackageFile(file);
  }

  function onDragOver(event: DragEvent) {
    event.preventDefault();
    dragOver = true;
  }

  function onDragLeave(event: DragEvent) {
    event.preventDefault();
    const next = event.relatedTarget;
    if (next instanceof Node && event.currentTarget instanceof Node) {
      if (event.currentTarget.contains(next)) return;
    }
    dragOver = false;
  }

  async function onDrop(event: DragEvent) {
    event.preventDefault();
    dragOver = false;
    const file = event.dataTransfer?.files?.[0];
    if (!file) return;
    await installPackageFile(file);
  }

  function extensionMeta(item: ExtensionListItem): string {
    const parts: string[] = [];
    if (item.author) parts.push(item.author);
    if (item.hasScript && !item.scriptSafe) parts.push("script blocked");
    if (item.hasWasm) parts.push("WASM");
    if (item.bundled && item.installed) parts.push("bundled · installed");
    else if (item.bundled && !item.installed)
      parts.push("bundled · not installed");
    else if (item.installed) parts.push("installed");
    return parts.join(" · ");
  }

  function formatLocaleDate(value: string): string {
    const trimmed = value.trim();
    if (!trimmed) return "";
    const parsed = Date.parse(trimmed);
    if (Number.isNaN(parsed)) return trimmed;
    return new Date(parsed).toLocaleDateString();
  }

  function installedDateLabel(item: ExtensionListItem): string {
    if (!item.installed || !item.installedAt) return "";
    const date = formatLocaleDate(item.installedAt);
    return date ? `Installed ${date}` : "";
  }

  function letterAvatar(name: string): string {
    const trimmed = name.trim();
    return trimmed ? trimmed[0]!.toUpperCase() : "?";
  }

  onMount(() => {
    void refresh();
  });
</script>

{#if loading}
  <Spinner class="extensions-settings__spinner" />
{:else if loadError}
  <EmptyState
    title="Could not load extensions"
    message={loadError}
    icon="alertCircle"
    embedded
  />
{:else}
  <div class="extensions-settings">
    <div
      class="extensions-settings__drop"
      class:extensions-settings__drop--active={dragOver}
      role="region"
      aria-label="Extension upload drop zone"
      ondragover={onDragOver}
      ondragleave={onDragLeave}
      ondrop={(event) => void onDrop(event)}
    >
      <input
        bind:this={fileInput}
        type="file"
        accept=".zip,.wasm,application/zip,application/wasm"
        class="extensions-settings__file-input"
        onchange={(event) => void handleUpload(event)}
      />
      <p class="extensions-settings__drop-copy">
        Drop a .zip package here, or upload one. Packages need
        {EXTENSION_MANIFEST} (and .wasm when used).
      </p>
      <Button
        size="sm"
        disabled={uploading || busyId !== null}
        onclick={() => fileInput?.click()}
      >
        {uploading ? "Installing..." : "Upload"}
      </Button>
    </div>

    {#if items.length === 0}
      <EmptyState
        title="No extensions installed"
        message={extensionsDir
          ? `Add a folder with ${EXTENSION_MANIFEST} under ${extensionsDir} and restart ${APP_NAME}, or upload a .zip package.`
          : `Add a folder with ${EXTENSION_MANIFEST} and restart ${APP_NAME}, or upload a .zip package.`}
        icon="puzzle"
        embedded
      />
    {:else}
      <div class="extensions-settings__list">
        {#each items as item (item.id)}
          <div class="extensions-settings__row">
            {#if item.imageUrl && item.imageUrl !== item.iconUrl}
              <img
                class="extensions-settings__banner"
                src={item.imageUrl}
                alt=""
                draggable="false"
              />
            {/if}
            <div class="extensions-settings__body">
              {#if item.iconUrl}
                <img
                  class="extensions-settings__icon"
                  src={item.iconUrl}
                  alt=""
                  draggable="false"
                />
              {:else}
                <span class="extensions-settings__letter" aria-hidden="true">
                  {letterAvatar(item.name)}
                </span>
              {/if}
              <div class="extensions-settings__copy">
                <div class="extensions-settings__name-row">
                  <p class="extensions-settings__name">{item.name}</p>
                  <span class="extensions-settings__version"
                    >v{item.version}</span
                  >
                </div>
                {#if item.description}
                  <p class="extensions-settings__description">
                    {item.description}
                  </p>
                {:else}
                  <p class="extensions-settings__description">{item.id}</p>
                {/if}
                {#if extensionMeta(item)}
                  <p class="extensions-settings__meta">{extensionMeta(item)}</p>
                {/if}
                {#if installedDateLabel(item)}
                  <p class="extensions-settings__installed">
                    {installedDateLabel(item)}
                  </p>
                {/if}
                <div class="extensions-settings__actions">
                  {#if item.installed}
                    <Button
                      size="sm"
                      variant="ghost"
                      disabled={busyId === item.id || uploading}
                      onclick={() => void handleUninstall(item)}
                    >
                      Uninstall
                    </Button>
                  {/if}
                  {#if item.bundled && !item.installed}
                    <Button
                      size="sm"
                      disabled={busyId === item.id || uploading}
                      onclick={() => void handleInstallBundled(item)}
                    >
                      Install
                    </Button>
                  {:else if item.bundled && item.installed}
                    <Button
                      size="sm"
                      variant="ghost"
                      disabled={busyId === item.id || uploading}
                      onclick={() => void handleInstallBundled(item)}
                    >
                      Reinstall
                    </Button>
                  {/if}
                </div>
              </div>
              {#if item.installed}
                <Toggle
                  checked={item.enabled}
                  disabled={busyId === item.id ||
                    uploading ||
                    (item.hasScript && !item.scriptSafe)}
                  ariaLabel={item.name}
                  onchange={(enabled) => void toggle(item, enabled)}
                />
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
{/if}

<style>
  :global(.extensions-settings__spinner .spinner__ring) {
    width: 1.125rem;
    height: 1.125rem;
  }

  .extensions-settings {
    display: grid;
    gap: var(--jb-space-3);
  }

  .extensions-settings__drop {
    display: grid;
    gap: var(--jb-space-3);
    justify-items: start;
    padding: var(--jb-space-4);
    border: 1px dashed var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-bg-muted);
    transition:
      border-color var(--jb-transition),
      background var(--jb-transition);
  }

  .extensions-settings__drop--active {
    border-color: var(--jb-accent);
    background: color-mix(in srgb, var(--jb-accent) 10%, transparent);
  }

  .extensions-settings__drop-copy {
    margin: 0;
    font-size: 0.8125rem;
    line-height: 1.5;
    color: var(--jb-text-muted);
  }

  .extensions-settings__file-input {
    display: none;
  }

  .extensions-settings__list {
    display: grid;
    gap: 0.5rem;
  }

  .extensions-settings__row {
    display: grid;
    gap: 0.5rem;
    padding: 0.5rem 0;
    border-bottom: 1px solid
      color-mix(in srgb, var(--jb-border, currentColor) 35%, transparent);
  }

  .extensions-settings__row:last-child {
    border-bottom: none;
  }

  .extensions-settings__banner {
    width: 100%;
    max-height: 3.5rem;
    object-fit: cover;
    border-radius: var(--jb-radius-md, 6px);
    image-rendering: pixelated;
  }

  .extensions-settings__body {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
  }

  .extensions-settings__icon,
  .extensions-settings__letter {
    width: 2.25rem;
    height: 2.25rem;
    flex-shrink: 0;
    border-radius: 2px;
  }

  .extensions-settings__icon {
    object-fit: contain;
    image-rendering: pixelated;
  }

  .extensions-settings__letter {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.9375rem;
    font-weight: 700;
    background: var(--jb-bg-muted);
    color: var(--jb-text);
    border: 1px solid var(--jb-border);
  }

  .extensions-settings__copy {
    flex: 1;
    min-width: 0;
    display: grid;
    gap: 0.25rem;
  }

  .extensions-settings__name-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
  }

  .extensions-settings__name {
    margin: 0;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--jb-text);
  }

  .extensions-settings__version {
    display: inline-flex;
    align-items: center;
    padding: 0.15rem 0.5rem;
    border-radius: var(--jb-radius-sm);
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.02em;
    line-height: 1.2;
    background: color-mix(in srgb, var(--jb-accent) 22%, var(--jb-bg-elevated));
    color: var(--jb-accent);
    border: 1px solid color-mix(in srgb, var(--jb-accent) 48%, transparent);
  }

  .extensions-settings__description,
  .extensions-settings__meta,
  .extensions-settings__installed {
    margin: 0;
    font-size: 0.8125rem;
    line-height: 1.5;
    color: var(--jb-text-muted);
  }

  .extensions-settings__meta,
  .extensions-settings__installed {
    font-size: 0.75rem;
  }

  .extensions-settings__installed {
    color: color-mix(in srgb, var(--jb-text-muted) 85%, var(--jb-text));
  }

  .extensions-settings__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
    margin-top: 0.125rem;
  }
</style>
