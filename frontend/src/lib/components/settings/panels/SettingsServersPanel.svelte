<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { APP_NAME } from "$lib/brand";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import InstanceForm from "$lib/components/instances/InstanceForm.svelte";
  import InstanceList from "$lib/components/instances/InstanceList.svelte";
  import LocalLibraryForm from "$lib/components/instances/LocalLibraryForm.svelte";
  import LocalLibraryList from "$lib/components/instances/LocalLibraryList.svelte";
  import MusicServerSettings from "$lib/components/settings/MusicServerSettings.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import { router } from "$lib/router/router.svelte";
  import { music } from "$lib/config/music.svelte";
  import { toast } from "$lib/ui/toast.svelte";
  import type { InstanceInput } from "$lib/features/instances/types";
  import type { LocalLibraryInput } from "$lib/features/local-libraries/types";
  import { localLibraries } from "$lib/features/local-libraries/store.svelte";
  import { sources } from "$lib/features/sources/store.svelte";
  import { isWailsMobile } from "$lib/config/runtime";
  import RemoteMelovianCard from "$lib/components/settings/RemoteMelovianCard.svelte";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import "$lib/settings/settings-page.css";

  let testing = $state(false);
  let saving = $state(false);
  let savingLibrary = $state(false);
  let showAddServer = $state(false);

  const showServerForm = $derived(instances.items.length > 0 && showAddServer);

  async function handleTest(input: InstanceInput) {
    testing = true;
    try {
      const serverName = await instances.test(input);
      toast.success(`Connected to ${serverName}`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Connection test failed",
      );
    } finally {
      testing = false;
    }
  }

  async function handleSubmit(input: InstanceInput) {
    saving = true;
    try {
      await instances.add(input);
      await sources.refreshStatus();
      await music.connect({ force: true });
      toast.success("Server added");
      showAddServer = false;
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to add instance",
      );
    } finally {
      saving = false;
    }
  }

  async function handleLocalLibrarySubmit(input: LocalLibraryInput) {
    savingLibrary = true;
    try {
      const created = await localLibraries.add(input);
      await sources.refreshStatus();
      await music.connect({ force: true });
      toast.info(`Added ${created.name}. Scanning in background…`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to add local library",
      );
    } finally {
      savingLibrary = false;
    }
  }
</script>

<section class="servers-panel">
  {#if isWailsMobile()}
    <RemoteMelovianCard />
  {/if}

  <SettingsCard
    title="Music servers"
    description="Connect Navidrome or any Subsonic-compatible server."
  >
    {#if instances.items.length > 0}
      <div class="servers-panel__toolbar">
        {#if instances.activeId}
          <Button
            size="sm"
            variant="surface"
            disabled={music.libraryRefreshing || !music.connected}
            onclick={() => void music.refreshLibrary()}
          >
            <MdiIcon name="refresh" size={16} />
            {music.libraryRefreshing ? "Refreshing..." : "Refresh library"}
          </Button>
        {/if}
        {#if !showAddServer}
          <Button
            size="sm"
            variant="surface"
            onclick={() => router.navigate("/setup")}
          >
            <MdiIcon name="plus" size={16} />
            Add server
          </Button>
        {/if}
      </div>
      <InstanceList />
    {:else}
      <EmptyState
        title="No servers yet"
        message="Use the guided setup to connect a Subsonic server."
        sourceKind="subsonic"
        embedded
      >
        {#snippet actions()}
          <Button onclick={() => router.navigate("/setup")}>
            <MdiIcon name="server" size={16} />
            Connect a server
          </Button>
        {/snippet}
      </EmptyState>
    {/if}

    {#if showServerForm}
      <div class="servers-panel__form">
        <h3 class="servers-panel__form-title">Add server</h3>
        <InstanceForm
          {testing}
          {saving}
          stickyActions
          ontest={handleTest}
          onsubmit={handleSubmit}
          oncancel={() => (showAddServer = false)}
        />
      </div>
    {/if}
  </SettingsCard>

  {#if localLibraries.enabled}
    <SettingsCard
      id="local-library"
      title="Local libraries"
      description="Index audio folders on this machine."
    >
      {#if isWailsMobile()}
        <p class="settings-page__meta">
          Local folders are not available in the Android or iOS app. Connect a
          {APP_NAME} server above, add a Subsonic server here, or index folders from
          the desktop app.
        </p>
      {:else}
        {#if localLibraries.items.length === 0}
          <EmptyState
            title="No local libraries yet"
            message="Browse for a music folder on this machine, or paste an absolute path."
            sourceKind="local"
            embedded
          >
            {#snippet actions()}
              <Button onclick={() => router.navigate("/setup?source=local")}>
                <MdiIcon name="folderMusic" size={16} />
                Add local folder
              </Button>
            {/snippet}
          </EmptyState>
        {:else}
          <p class="settings-page__meta">
            {localLibraries.items.length}
            {localLibraries.items.length === 1 ? "library" : "libraries"} saved. Switch
            the active library from the source menu.
          </p>
          <LocalLibraryList />
          {#if extensionFeatures.metadata}
            <div class="settings-page__metadata-link">
              <Button
                variant="surface"
                onclick={() => router.navigate("/music/metadata")}
              >
                <MdiIcon name="tag" size={18} />
                Open metadata editor
              </Button>
            </div>
          {/if}
        {/if}

        <div class="servers-panel__form">
          <h3 class="servers-panel__form-title">
            {localLibraries.items.length === 0
              ? "Add a local library"
              : "Add another library"}
          </h3>
          <LocalLibraryForm
            saving={savingLibrary}
            onsubmit={handleLocalLibrarySubmit}
          />
        </div>
      {/if}
    </SettingsCard>
  {/if}

  <SettingsCard
    title="This host"
    description={`REST API, shares, and options for this ${APP_NAME} process.`}
  >
    <MusicServerSettings />
  </SettingsCard>
</section>

<style>
  .servers-panel {
    display: flex;
    flex-direction: column;
  }

  .servers-panel__toolbar {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
    margin-bottom: var(--jb-space-3);
  }

  .servers-panel__form {
    display: grid;
    gap: var(--jb-space-3);
    margin-top: var(--jb-space-4);
    padding-top: var(--jb-space-4);
    border-top: 1px solid var(--jb-border);
  }

  .servers-panel__form-title {
    margin: 0;
    font-size: 0.875rem;
    font-weight: 700;
  }
</style>
