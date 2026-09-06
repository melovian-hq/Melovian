<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import {
    clearRemoteServerUrl,
    getRemoteServerUrl,
    normalizeRemoteServerUrl,
    probeRemoteServer,
    setRemoteServerUrl,
  } from "$lib/config/remote-server";
  import { toast } from "$lib/ui/toast.svelte";
  import { APP_NAME, APP_SLUG } from "$lib/brand";
  import "$lib/settings/settings-page.css";

  let url = $state(getRemoteServerUrl());
  let connectedUrl = $state(getRemoteServerUrl());
  let testing = $state(false);
  let saving = $state(false);

  async function testConnection() {
    testing = true;
    try {
      const result = await probeRemoteServer(url);
      url = result.url;
      const mode = result.authEnabled
        ? "auth on"
        : result.demoMode
          ? "demo mode"
          : "no account login";
      toast.success(`Reached ${result.url} (${mode})`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Connection failed");
    } finally {
      testing = false;
    }
  }

  async function connect() {
    saving = true;
    try {
      const result = await probeRemoteServer(url);
      setRemoteServerUrl(result.url);
      connectedUrl = result.url;
      if (result.authEnabled && result.url.startsWith("http://")) {
        toast.info(
          `Connected over HTTP. Account login cookies need HTTPS. Use a no-auth server, or put ${APP_NAME} behind TLS.`,
        );
      } else if (!result.authEnabled) {
        toast.success(`Using ${result.url} (no account login). Reloading.`);
      } else {
        toast.success(`Using ${result.url}. Reloading.`);
      }
      window.location.reload();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not connect");
      saving = false;
    }
  }

  function disconnect() {
    clearRemoteServerUrl();
    connectedUrl = "";
    toast.success(`Using the on-device ${APP_NAME} backend. Reloading.`);
    window.location.reload();
  }

  function onUrlBlur() {
    const normalized = normalizeRemoteServerUrl(url);
    if (normalized) {
      url = normalized;
    }
  }
</script>

<SettingsCard
  id="melovian-host"
  title={`${APP_NAME} server`}
  description={`Point this app at your Docker or ${APP_SLUG}-server host. Works over http on Tailscale/Netbird, or https on the public internet. No-auth and demo servers are supported.`}
>
  <Field
    label="Server URL"
    hint="Examples: http://100.64.1.2:8080 or https://melovian.example.com"
  >
    <Input
      bind:value={url}
      type="url"
      placeholder="http://100.64.1.2:8080"
      autocomplete="url"
      inputmode="url"
      enterkeyhint="go"
      autocapitalize="off"
      spellcheck={false}
      onblur={onUrlBlur}
    />
  </Field>

  {#if connectedUrl}
    <p class="settings-page__meta">
      Connected to {connectedUrl}. Device sync and listen together use this
      host.
    </p>
  {:else}
    <p class="settings-page__meta">
      Not connected. The app is using its built-in backend on this device.
    </p>
  {/if}

  {#snippet footer()}
    <Button
      variant="ghost"
      disabled={testing || saving || !url.trim()}
      onclick={() => void testConnection()}
    >
      {testing ? "Testing…" : "Test"}
    </Button>
    {#if connectedUrl}
      <Button variant="ghost" disabled={saving} onclick={disconnect}>
        Disconnect
      </Button>
    {/if}
    <Button
      variant="primary"
      disabled={testing || saving || !url.trim()}
      onclick={() => void connect()}
    >
      {saving ? "Connecting…" : "Connect"}
    </Button>
  {/snippet}
</SettingsCard>
