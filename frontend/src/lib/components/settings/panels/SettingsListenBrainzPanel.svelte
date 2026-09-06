<script lang="ts">
  import { onMount } from "svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import * as musicApi from "$lib/music/api";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import { toast } from "$lib/ui/toast.svelte";
  import "$lib/settings/settings-page.css";

  let settings = $state<musicApi.ListenBrainzSettings | null>(null);
  let token = $state("");
  let endpoint = $state("");
  let testing = $state(false);
  let saving = $state(false);

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
    if (!value) return;
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

  async function save() {
    const value = token.trim();
    if (!value) {
      toast.error("Enter a token to save");
      return;
    }
    const ep = endpoint.trim() || "https://api.listenbrainz.org";
    saving = true;
    try {
      const next = await musicApi.saveListenBrainzSettings({
        token: value,
        endpoint: ep,
      });
      if (next) {
        settings = next;
        token = "";
        toast.success("ListenBrainz settings saved");
      } else {
        toast.error("Could not save settings");
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not save settings");
    } finally {
      saving = false;
    }
  }
</script>

<SettingsCard
  title="ListenBrainz"
  description="Scrobble listens to ListenBrainz or a self-hosted compatible endpoint."
>
  <div class="listenbrainz-settings">
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
      disabled={!settings}
    />
    {#if settings?.hasToken}
      <p class="listenbrainz-settings__status">
        A token is already saved. Type above to replace it.
      </p>
    {/if}
    <div class="listenbrainz-settings__actions">
      <Button
        size="sm"
        disabled={!token.trim() || testing || saving || !settings}
        onclick={() => void test()}
      >
        {testing ? "Testing..." : "Test token"}
      </Button>
      <Button
        size="sm"
        disabled={!token.trim() || saving || testing || !settings}
        onclick={() => void save()}
      >
        {saving ? "Saving..." : "Save token"}
      </Button>
    </div>
  </div>
</SettingsCard>

<style>
  .listenbrainz-settings {
    display: grid;
    gap: var(--jb-space-3);
    max-width: 32rem;
  }

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

  .listenbrainz-settings__actions {
    display: flex;
    gap: var(--jb-space-3);
  }
</style>
