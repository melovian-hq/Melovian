<script lang="ts">
  import { onMount } from "svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import SettingsSaveStatus from "$lib/components/settings/SettingsSaveStatus.svelte";
  import { useDebounce } from "runed";
  import * as musicApi from "$lib/music/api";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import { toast } from "$lib/ui/toast.svelte";
  import { createSaveStatus } from "$lib/settings/save-status.svelte";

  const listenBrainzStatus = createSaveStatus();

  let settings = $state<musicApi.ListenBrainzSettings | null>(null);
  let token = $state("");
  let endpoint = $state("");
  let testing = $state(false);

  const enabled = $derived(extensionFeatures.isEnabled("listenbrainz"));

  onMount(() => {
    void load();
  });

  async function load() {
    const next = await musicApi.getListenBrainzSettings().catch(() => null);
    if (!next) {
      settings = { enabled: false, hasToken: false, token: "", endpoint: "" };
      return;
    }
    settings = next;
    token = "";
    endpoint = next.endpoint;
  }

  async function test() {
    const value = token.trim();
    const ep = endpoint.trim() || "https://api.listenbrainz.org";
    if (!value && !settings?.hasToken) return;
    testing = true;
    try {
      const result = await musicApi.testListenBrainzToken(value, ep);
      if (!result) {
        toast.error("Could not validate token");
        return;
      }
      if (result.ok) {
        toast.success(`Token valid for ${result.userName ?? "ListenBrainz"}`);
      } else {
        toast.error("Token is not valid");
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Token test failed");
    } finally {
      testing = false;
    }
  }

  // The server stores the payload verbatim, so a save only runs while the
  // token field is filled. An empty token would wipe the stored one.
  async function save() {
    const value = token.trim();
    if (!value) return;
    const ep = endpoint.trim() || "https://api.listenbrainz.org";
    listenBrainzStatus.begin();
    try {
      const next = await musicApi.saveListenBrainzSettings({
        token: value,
        endpoint: ep,
      });
      if (next) {
        settings = next;
        endpoint = next.endpoint;
        listenBrainzStatus.saved();
      } else {
        listenBrainzStatus.failed("Could not save ListenBrainz settings");
        toast.error("Could not save ListenBrainz settings");
      }
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : "Could not save ListenBrainz settings";
      listenBrainzStatus.failed(message);
      toast.error(message);
    }
  }

  const debouncedSave = useDebounce(() => save(), 500);

  function queueSave() {
    void debouncedSave().catch(() => {});
  }
</script>

<SettingsCard
  title="ListenBrainz"
  description="Scrobble listens to ListenBrainz or a self-hosted compatible endpoint."
>
  {#snippet status()}
    <SettingsSaveStatus status={listenBrainzStatus} />
  {/snippet}
  {#if !enabled}
    <p class="listenbrainz-settings__note">
      Enable the ListenBrainz extension in the Extensions tab to start
      scrobbling.
    </p>
  {/if}
  <label class="listenbrainz-settings__label" for="listenbrainz-endpoint">
    Endpoint
  </label>
  <Input
    id="listenbrainz-endpoint"
    type="url"
    placeholder="https://api.listenbrainz.org"
    bind:value={endpoint}
    oninput={queueSave}
    disabled={!settings}
  />
  <label class="listenbrainz-settings__label" for="listenbrainz-token">
    API token
  </label>
  <Input
    id="listenbrainz-token"
    type="password"
    placeholder="Paste your ListenBrainz token"
    bind:value={token}
    oninput={queueSave}
    disabled={!settings}
  />
  {#if settings?.hasToken && !token.trim()}
    <p class="listenbrainz-settings__status">
      A token is saved. Enter it to update the endpoint or replace it.
    </p>
  {/if}
  {#snippet footer()}
    <Button
      size="sm"
      disabled={(!token.trim() && !settings?.hasToken) || testing || !settings}
      onclick={() => void test()}
    >
      {testing ? "Testing..." : "Test token"}
    </Button>
  {/snippet}
</SettingsCard>

<style>
  .listenbrainz-settings__note {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .listenbrainz-settings__label {
    font-size: 0.8125rem;
    font-weight: 600;
  }

  .listenbrainz-settings__status {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }
</style>
