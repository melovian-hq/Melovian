<script lang="ts">
  import { onMount } from "svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import Toggle from "$lib/components/ui/Toggle.svelte";
  import { loadExtensions } from "$lib/extensions/registry";
  import { APP_NAME, EXTENSION_MANIFEST } from "$lib/brand";
  import {
    clearRegistryConfig,
    fetchExtensionRegistry,
    fetchExtensions,
    installExtension,
    installExtensionDir,
    installRemoteExtension,
    saveRegistryConfig,
    reinstallExtension,
    saveExtensionSettings,
    setExtensionEnabled,
    uninstallExtension,
    type ExtensionListItem,
    type RegistryItem,
  } from "$lib/extensions/api";
  import type {
    ExtensionManifest,
    ExtensionSettingField,
  } from "$lib/extensions/types";
  import { confirmDialog } from "$lib/ui/confirm.svelte";
  import { toast } from "$lib/ui/toast.svelte";
  import { router } from "$lib/router/router.svelte";
  import { extensionDeepLinkId } from "$lib/desktop/deep-link";
  import {
    extensionAutoUpdate,
    setExtensionAutoUpdate,
  } from "$lib/extensions/updates.svelte";

  let items = $state<ExtensionListItem[]>([]);
  let extensionsDir = $state("");
  let loading = $state(true);
  let loadError = $state<string | null>(null);
  let busyId = $state<string | null>(null);
  let uploading = $state(false);
  let fileInput = $state<HTMLInputElement | undefined>();
  let dragOver = $state(false);
  let registryItems = $state<RegistryItem[]>([]);
  let registryLoading = $state(false);
  let registryError = $state<string | null>(null);
  let registrySigned = $state(false);
  let handledDeepLink = $state("");
  let manifests = $state<ExtensionManifest[]>([]);
  let settingsDrafts = $state<Record<string, Record<string, unknown>>>({});
  let settingsSavingId = $state<string | null>(null);
  let devPath = $state("");
  let registryUrl = $state("");
  let registryKey = $state("");
  let registryCustom = $state(false);
  let registryActiveUrl = $state("");
  let registryConfigBusy = $state(false);

  async function applyPayload(payload: {
    items: ExtensionListItem[];
    dir: string;
    manifests?: ExtensionManifest[];
  }) {
    items = payload.items;
    extensionsDir = payload.dir;
    manifests = payload.manifests ?? manifests;
    ensureSettingsDrafts();
    await loadExtensions();
  }

  async function refresh() {
    loading = true;
    loadError = null;
    try {
      const payload = await fetchExtensions();
      items = payload.items;
      extensionsDir = payload.dir;
      manifests = payload.manifests ?? [];
      ensureSettingsDrafts();
    } catch (err) {
      items = [];
      extensionsDir = "";
      loadError =
        err instanceof Error ? err.message : "Could not load extensions";
    } finally {
      loading = false;
    }
  }

  async function loadRegistry() {
    registryLoading = true;
    registryError = null;
    try {
      const payload = await fetchExtensionRegistry();
      registryItems = payload.items;
      registrySigned = payload.signed;
      registryCustom = payload.custom;
      registryActiveUrl = payload.url;
    } catch (err) {
      registryItems = [];
      registryError =
        err instanceof Error ? err.message : "Could not load registry";
    } finally {
      registryLoading = false;
    }
  }

  function installConfirmMessage(item: RegistryItem): string {
    const lines = [`Install ${item.name} v${item.version}?`];
    if (item.author) lines.push(`Author: ${item.author}`);
    const caps: string[] = [];
    if (item.hasScript) caps.push("runs a sandboxed script");
    if (item.hasWasm) caps.push("ships WASM");
    if (item.styles) caps.push(`${item.styles} stylesheet(s)`);
    if (item.appTheme) caps.push("app theme");
    if (item.trackRules) caps.push(`${item.trackRules} track rule(s)`);
    if (caps.length) lines.push(`It ${caps.join(", ")}.`);
    if (item.risk && item.risk !== "low") {
      lines.push(`Audit risk rating: ${item.risk}.`);
    }
    if (item.delisted) {
      lines.push(
        `This extension was delisted${item.delisted.reason ? `: ${item.delisted.reason}` : "."}`,
      );
    }
    if (item.requires?.length) {
      lines.push(`Requires ${item.requires.join(", ")} to be installed first.`);
    }
    if (item.minAppVersion) {
      lines.push(`Needs Melovian ${item.minAppVersion} or newer.`);
    }
    if (item.externalUrls?.length) {
      lines.push(
        `References ${item.externalUrls.length} external URL(s), listed on the extension page.`,
      );
    }
    if (!registrySigned) {
      lines.push("This registry did not provide a verified signature.");
    }
    return lines.join("\n\n");
  }

  async function handleInstallRemote(item: RegistryItem, version?: string) {
    const confirmed = await confirmDialog.confirm({
      title: `Install ${item.name}`,
      message: installConfirmMessage(item),
      confirmLabel: item.installed ? "Update" : "Install",
    });
    if (!confirmed) return;
    busyId = item.id;
    try {
      const payload = await installRemoteExtension(item.id, version);
      await applyPayload(payload);
      toast.success(`Installed ${item.name}`);
      await loadRegistry();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not install extension",
      );
    } finally {
      busyId = null;
    }
  }

  // Published versions the row can roll back to: everything except the
  // currently installed one. The downgrade is explicit so the registry
  // guard lets it through, and the dialog makes the risk clear.
  const REPORT_ISSUE_URL =
    "https://github.com/melovian-hq/Melovian-Extensions/issues/new";

  function reportUrl(item: RegistryItem): string {
    const title = encodeURIComponent(`Report extension: ${item.id}`);
    const body = encodeURIComponent(
      `Extension: ${item.id}\nVersion: ${item.version}\n\nReason:\n`,
    );
    return `${REPORT_ISSUE_URL}?title=${title}&body=${body}`;
  }

  function olderVersions(item: RegistryItem): string[] {
    return (item.versions ?? [])
      .map((v) => v.version)
      .filter((v) => v !== item.installedVersion);
  }

  async function handleRollback(item: RegistryItem, version: string) {
    const confirmed = await confirmDialog.confirm({
      title: `Roll back ${item.name}`,
      message: `Install ${item.name} v${version} over the installed v${item.installedVersion ?? item.version}?\n\nOlder versions are signed and checksummed like current releases, but a rollback reintroduces whatever the newer release fixed.`,
      confirmLabel: `Install v${version}`,
    });
    if (!confirmed) return;
    busyId = item.id;
    try {
      const payload = await installRemoteExtension(item.id, version);
      await applyPayload(payload);
      toast.success(`Rolled back ${item.name} to v${version}`);
      await loadRegistry();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not roll back extension",
      );
    } finally {
      busyId = null;
    }
  }

  // A melovian://install-extension/<id> deep link or
  // ?install-extension=<id> query lands here. Prompt once per id, then
  // strip the param so reloads do not re-prompt.
  async function handleDeepLink(search: string) {
    const id = extensionDeepLinkId(search);
    if (!id || id === handledDeepLink) return;
    handledDeepLink = id;
    router.navigate("/settings/extensions", true);
    if (registryLoading) return;
    const item = registryItems.find((entry) => entry.id === id);
    if (!item) {
      if (!registryError) await loadRegistry();
      const retry = registryItems.find((entry) => entry.id === id);
      if (!retry) {
        toast.error(`Extension ${id} is not in the registry`);
        return;
      }
      await handleInstallRemote(retry);
      return;
    }
    await handleInstallRemote(item);
  }

  function settingsFieldsFor(item: ExtensionListItem): ExtensionSettingField[] {
    if (!item.installed) return [];
    return manifests.find((m) => m.id === item.id)?.settings ?? [];
  }

  // Editable copy per extension: saved values merged over field defaults.
  // Drafts persist across refreshes so unsaved edits are not lost when the
  // list reloads. They are created in ensureSettingsDrafts from payload
  // handlers, never during render, because writing state inside a template
  // expression throws state_unsafe_mutation.
  function ensureSettingsDrafts() {
    for (const item of items) {
      const fields = settingsFieldsFor(item);
      if (!item.installed || !fields.length || settingsDrafts[item.id]) {
        continue;
      }
      const draft: Record<string, unknown> = {};
      for (const field of fields) {
        draft[field.key] =
          item.settings?.[field.key] ?? field.default ?? defaultValue(field);
      }
      settingsDrafts[item.id] = draft;
    }
  }

  function settingsDraftFor(item: ExtensionListItem): Record<string, unknown> {
    return settingsDrafts[item.id] ?? {};
  }

  function defaultValue(field: ExtensionSettingField): unknown {
    if (field.type === "boolean") return false;
    if (field.type === "choice") return field.options?.[0] ?? "";
    return "";
  }

  // Dev installs symlink a work tree into the extensions dir, so
  // reloading re-reads manifest, styles, and script from disk.
  async function handleInstallDir() {
    const path = devPath.trim();
    if (!path) return;
    busyId = "__dev__";
    try {
      const payload = await installExtensionDir(path);
      await applyPayload(payload);
      devPath = "";
      toast.success("Linked extension directory");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not link extension",
      );
    } finally {
      busyId = null;
    }
  }

  async function handleReloadDev(item: ExtensionListItem) {
    busyId = item.id;
    try {
      const payload = await fetchExtensions();
      items = payload.items;
      manifests = payload.manifests ?? [];
      ensureSettingsDrafts();
      await loadExtensions();
      toast.success(`Reloaded ${item.name}`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not reload extension",
      );
    } finally {
      busyId = null;
    }
  }

  // A custom registry is a trust decision: its key signs every package
  // the app will install, so the confirm dialog spells that out.
  async function handleSaveRegistry() {
    const url = registryUrl.trim();
    const key = registryKey.trim();
    if (!url || !key) {
      toast.error("Registry URL and public key are both required");
      return;
    }
    const confirmed = await confirmDialog.confirm({
      title: "Use a custom registry",
      message: `${url}\n\nEvery extension you install after this will be signed by the key you paste here, not the official Melovian key. Only continue if you trust whoever runs this registry.`,
      confirmLabel: "Trust registry",
    });
    if (!confirmed) return;
    registryConfigBusy = true;
    try {
      await saveRegistryConfig(url, [key]);
      registryUrl = "";
      registryKey = "";
      await loadRegistry();
      toast.success("Custom registry saved");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not save registry",
      );
    } finally {
      registryConfigBusy = false;
    }
  }

  async function handleClearRegistry() {
    registryConfigBusy = true;
    try {
      await clearRegistryConfig();
      await loadRegistry();
      toast.success("Back to the official registry");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not reset registry",
      );
    } finally {
      registryConfigBusy = false;
    }
  }

  async function handleSaveSettings(item: ExtensionListItem) {
    settingsSavingId = item.id;
    try {
      const saved = await saveExtensionSettings(
        item.id,
        settingsDraftFor(item),
      );
      item.settings = saved;
      await loadExtensions();
      toast.success(`Saved ${item.name} settings`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not save settings",
      );
    } finally {
      settingsSavingId = null;
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
    if (item.dev) parts.push("dev link");
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

  function registryMeta(item: RegistryItem): string {
    const parts: string[] = [];
    if (item.author) parts.push(item.author);
    if (item.trackRules) parts.push(`${item.trackRules} rules`);
    if (item.styles) parts.push(`${item.styles} styles`);
    if (item.hasScript) parts.push("script");
    if (item.hasWasm) parts.push("WASM");
    if (item.appTheme) parts.push("theme");
    return parts.join(" · ");
  }

  onMount(() => {
    void refresh();
    void loadRegistry().then(() => handleDeepLink(window.location.search));
  });

  $effect(() => {
    void handleDeepLink(router.search);
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

    <div class="extensions-settings__dev">
      <p class="extensions-settings__dev-title">Development</p>
      <p class="extensions-settings__browse-note">
        Link a local extension folder. Edits apply when you press Reload on its
        row. Scripts still run in the sandbox.
      </p>
      <div class="extensions-settings__dev-form">
        <input
          type="text"
          class="extensions-settings__input extensions-settings__input--wide"
          placeholder="/path/to/my-extension"
          aria-label="Extension directory path"
          bind:value={devPath}
        />
        <Button
          size="sm"
          variant="ghost"
          disabled={!devPath.trim() || busyId !== null || uploading}
          onclick={() => void handleInstallDir()}
        >
          Link
        </Button>
      </div>
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
                  {#if item.installed && !item.required}
                    <Button
                      size="sm"
                      variant="ghost"
                      disabled={busyId === item.id || uploading}
                      onclick={() => void handleUninstall(item)}
                    >
                      Uninstall
                    </Button>
                  {/if}
                  {#if item.dev && item.installed}
                    <Button
                      size="sm"
                      variant="ghost"
                      disabled={busyId === item.id || uploading}
                      onclick={() => void handleReloadDev(item)}
                    >
                      Reload
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
                {#if item.installed && settingsFieldsFor(item).length > 0}
                  {@const draft = settingsDraftFor(item)}
                  <div class="extensions-settings__fields">
                    {#each settingsFieldsFor(item) as field (field.key)}
                      <label class="extensions-settings__field">
                        <span class="extensions-settings__field-label">
                          {field.label || field.key}
                        </span>
                        {#if field.type === "boolean"}
                          <input
                            type="checkbox"
                            class="extensions-settings__checkbox"
                            checked={draft[field.key] === true}
                            onchange={(e) =>
                              (draft[field.key] = e.currentTarget.checked)}
                          />
                        {:else if field.type === "choice"}
                          <select
                            class="extensions-settings__input"
                            value={String(draft[field.key] ?? "")}
                            onchange={(e) =>
                              (draft[field.key] = e.currentTarget.value)}
                          >
                            {#each field.options ?? [] as opt (opt)}
                              <option value={opt}>{opt}</option>
                            {/each}
                          </select>
                        {:else}
                          <input
                            type="text"
                            class="extensions-settings__input"
                            maxlength="256"
                            value={String(draft[field.key] ?? "")}
                            oninput={(e) =>
                              (draft[field.key] = e.currentTarget.value)}
                          />
                        {/if}
                      </label>
                    {/each}
                    <Button
                      size="sm"
                      variant="ghost"
                      disabled={settingsSavingId === item.id || uploading}
                      onclick={() => void handleSaveSettings(item)}
                    >
                      {settingsSavingId === item.id
                        ? "Saving..."
                        : "Save settings"}
                    </Button>
                  </div>
                {/if}
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

    <div class="extensions-settings__browse">
      <p class="extensions-settings__browse-title">
        Browse the registry
        {#if registrySigned}
          <span class="extensions-settings__signed">signed</span>
        {/if}
        {#if registryCustom}
          <span class="extensions-settings__custom">custom</span>
        {/if}
      </p>
      {#if registryActiveUrl}
        <p
          class="extensions-settings__browse-note extensions-settings__registry-url"
        >
          {registryActiveUrl}
        </p>
      {/if}
      <div class="extensions-settings__registry-config">
        <input
          type="url"
          class="extensions-settings__input extensions-settings__input--wide"
          placeholder="https://example.com/registry.json"
          aria-label="Custom registry URL"
          bind:value={registryUrl}
        />
        <input
          type="text"
          class="extensions-settings__input extensions-settings__input--wide"
          placeholder="Ed25519 public key (hex)"
          aria-label="Registry public key"
          bind:value={registryKey}
        />
        <div class="extensions-settings__actions">
          <Button
            size="sm"
            variant="ghost"
            disabled={registryConfigBusy ||
              (!registryUrl.trim() && !registryKey.trim())}
            onclick={() => void handleSaveRegistry()}
          >
            Use custom registry
          </Button>
          {#if registryCustom}
            <Button
              size="sm"
              variant="ghost"
              disabled={registryConfigBusy}
              onclick={() => void handleClearRegistry()}
            >
              Use official
            </Button>
          {/if}
        </div>
      </div>
      <div class="extensions-settings__auto-update">
        <Toggle
          checked={extensionAutoUpdate()}
          ariaLabel="Auto-update low-risk extensions"
          onchange={(v) => setExtensionAutoUpdate(v)}
        />
        <span class="extensions-settings__browse-note">
          Automatically install updates flagged low risk
        </span>
      </div>
      {#if registryLoading}
        <Spinner class="extensions-settings__spinner" />
      {:else if registryError}
        <p class="extensions-settings__browse-note">
          Could not reach the extension registry: {registryError}
        </p>
        <Button size="sm" variant="ghost" onclick={() => void loadRegistry()}>
          Retry
        </Button>
      {:else if registryItems.length === 0}
        <p class="extensions-settings__browse-note">
          The registry returned no extensions.
        </p>
      {:else}
        <div class="extensions-settings__list">
          {#each registryItems as item (item.id)}
            <div class="extensions-settings__row">
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
                    {#if item.delisted}
                      <span class="extensions-settings__delisted">delisted</span
                      >
                    {:else if item.auditStatus === "pass"}
                      <span class="extensions-settings__audited">audited</span>
                    {/if}
                  </div>
                  {#if item.description}
                    <p class="extensions-settings__description">
                      {item.description}
                    </p>
                  {/if}
                  {#if registryMeta(item)}
                    <p class="extensions-settings__meta">
                      {registryMeta(item)}
                    </p>
                  {/if}
                  <div class="extensions-settings__actions">
                    {#if item.delisted}
                      <span
                        class="extensions-settings__delisted-note"
                        title={item.delisted.reason}
                      >
                        Removed from the registry
                      </span>
                    {:else if item.installed && item.updateAvailable}
                      <Button
                        size="sm"
                        disabled={busyId === item.id || uploading}
                        onclick={() => void handleInstallRemote(item)}
                      >
                        Update to v{item.version}
                      </Button>
                    {:else if item.installed}
                      <span class="extensions-settings__installed">
                        Installed{item.installedVersion
                          ? ` v${item.installedVersion}`
                          : ""}
                      </span>
                      {#if olderVersions(item).length > 0}
                        <select
                          class="extensions-settings__versions"
                          disabled={busyId === item.id || uploading}
                          aria-label={`Roll back ${item.name}`}
                          onchange={(e) => {
                            const v = e.currentTarget.value;
                            e.currentTarget.value = "";
                            if (v) void handleRollback(item, v);
                          }}
                        >
                          <option value="">Other versions</option>
                          {#each olderVersions(item) as v (v)}
                            <option value={v}>v{v}</option>
                          {/each}
                        </select>
                      {/if}
                    {:else}
                      <Button
                        size="sm"
                        disabled={busyId === item.id || uploading}
                        onclick={() => void handleInstallRemote(item)}
                      >
                        Install
                      </Button>
                    {/if}
                    <a
                      class="extensions-settings__report"
                      href={reportUrl(item)}
                      target="_blank"
                      rel="noopener noreferrer"
                      title={`Report ${item.name} to the registry maintainers`}
                    >
                      Report
                    </a>
                  </div>
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
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
    gap: var(--jb-space-2);
  }

  .extensions-settings__row {
    display: grid;
    gap: var(--jb-space-2);
    padding: var(--jb-space-2) 0;
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
    gap: var(--jb-space-1);
  }

  .extensions-settings__name-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--jb-space-2);
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
    margin-top: var(--jb-space-1);
  }

  .extensions-settings__browse {
    display: grid;
    gap: var(--jb-space-2);
    padding-top: var(--jb-space-3);
    border-top: 1px solid
      color-mix(in srgb, var(--jb-border, currentColor) 50%, transparent);
  }

  .extensions-settings__browse-title {
    margin: 0;
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-text);
  }

  .extensions-settings__browse-note {
    margin: 0;
    font-size: 0.8125rem;
    line-height: 1.5;
    color: var(--jb-text-muted);
  }

  .extensions-settings__audited {
    display: inline-flex;
    align-items: center;
    padding: 0.15rem 0.5rem;
    border-radius: var(--jb-radius-sm);
    font-size: 0.6875rem;
    font-weight: 700;
    letter-spacing: 0.02em;
    line-height: 1.2;
    background: color-mix(in srgb, var(--jb-success) 18%, transparent);
    color: var(--jb-success);
    border: 1px solid color-mix(in srgb, var(--jb-success) 40%, transparent);
  }

  .extensions-settings__delisted {
    display: inline-flex;
    align-items: center;
    padding: 0.15rem 0.5rem;
    border-radius: var(--jb-radius-sm);
    font-size: 0.6875rem;
    font-weight: 700;
    letter-spacing: 0.02em;
    line-height: 1.2;
    background: color-mix(in srgb, var(--jb-danger) 18%, transparent);
    color: var(--jb-danger);
    border: 1px solid color-mix(in srgb, var(--jb-danger) 40%, transparent);
  }

  .extensions-settings__auto-update {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    margin-bottom: var(--jb-space-2);
  }

  .extensions-settings__versions {
    background: var(--jb-surface);
    color: var(--jb-text-muted);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-sm);
    font-size: 0.75rem;
    padding: 0.2rem 0.4rem;
  }

  .extensions-settings__report {
    font-size: 0.75rem;
    color: var(--jb-text-muted);
    text-decoration: none;
    align-self: center;
  }

  .extensions-settings__report:hover {
    color: var(--jb-danger);
    text-decoration: underline;
  }

  .extensions-settings__delisted-note {
    font-size: 0.8125rem;
    color: var(--jb-danger);
  }

  .extensions-settings__fields {
    display: grid;
    gap: var(--jb-space-2);
    justify-items: start;
    margin-top: var(--jb-space-2);
    padding-top: var(--jb-space-2);
    border-top: 1px solid
      color-mix(in srgb, var(--jb-border, currentColor) 25%, transparent);
  }

  .extensions-settings__field {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .extensions-settings__field-label {
    min-width: 6rem;
  }

  .extensions-settings__input {
    background: var(--jb-surface);
    color: var(--jb-text);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-sm);
    font-size: 0.8125rem;
    padding: 0.2rem 0.5rem;
    max-width: 14rem;
  }

  .extensions-settings__checkbox {
    accent-color: var(--jb-accent);
  }

  .extensions-settings__dev {
    display: grid;
    gap: var(--jb-space-2);
    justify-items: start;
    padding: var(--jb-space-3);
    border: 1px solid
      color-mix(in srgb, var(--jb-border, currentColor) 35%, transparent);
    border-radius: var(--jb-radius-md);
  }

  .extensions-settings__dev-title {
    margin: 0;
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-text);
  }

  .extensions-settings__dev-form {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
  }

  .extensions-settings__input--wide {
    min-width: 16rem;
  }

  .extensions-settings__registry-config {
    display: grid;
    gap: var(--jb-space-2);
    justify-items: start;
    margin-bottom: var(--jb-space-2);
  }

  .extensions-settings__registry-url {
    word-break: break-all;
  }

  .extensions-settings__custom {
    display: inline-flex;
    align-items: center;
    margin-left: var(--jb-space-2);
    padding: 0.1rem 0.45rem;
    border-radius: var(--jb-radius-sm);
    font-size: 0.6875rem;
    font-weight: 700;
    letter-spacing: 0.02em;
    line-height: 1.2;
    vertical-align: middle;
    background: color-mix(in srgb, var(--jb-warning) 18%, transparent);
    color: var(--jb-warning);
    border: 1px solid color-mix(in srgb, var(--jb-warning) 40%, transparent);
  }

  .extensions-settings__signed {
    display: inline-flex;
    align-items: center;
    margin-left: var(--jb-space-2);
    padding: 0.1rem 0.45rem;
    border-radius: var(--jb-radius-sm);
    font-size: 0.6875rem;
    font-weight: 700;
    letter-spacing: 0.02em;
    line-height: 1.2;
    vertical-align: middle;
    background: color-mix(in srgb, var(--jb-accent) 18%, transparent);
    color: var(--jb-accent);
    border: 1px solid color-mix(in srgb, var(--jb-accent) 40%, transparent);
  }
</style>
