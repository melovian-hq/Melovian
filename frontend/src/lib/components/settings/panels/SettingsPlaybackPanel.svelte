<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Select from "$lib/components/ui/Select.svelte";
  import { music } from "$lib/config/music.svelte";
  import { APP_NAME } from "$lib/brand";
  import { isWailsMobile, nativeDesktopAvailable } from "$lib/config/runtime";
  import {
    loadNativeBackendPref,
    saveNativeBackendPref,
    saveNativePlaybackPref,
    type NativeBackendPref,
  } from "$lib/music/prefs";
  import {
    TRANSCODE_BITRATE_OPTIONS,
    TRANSCODE_FORMAT_LABELS,
    mergeTranscodingSettings,
    type TranscodeFormat,
    type TranscodingSettings,
  } from "$lib/music/transcoding-settings";
  import {
    QUEUE_SIZE_OPTIONS,
    mergeQueueSettings,
    type QueueSettings,
  } from "$lib/music/queue-settings";
  import {
    CROSSFADE_DURATION_OPTIONS,
    mergePlaybackSettings,
    type PlaybackSettings,
  } from "$lib/music/playback-settings";
  import {
    IMMERSIVE_AUDIO_MODE_HINTS,
    IMMERSIVE_AUDIO_MODE_LABELS,
    mergeImmersiveAudioSettings,
    type ImmersiveAudioMode,
    type ImmersiveAudioSettings,
  } from "$lib/music/immersive-audio-settings";
  import { toast } from "$lib/ui/toast.svelte";
  import "$lib/settings/settings-page.css";

  let nativeBackendPref = $state<NativeBackendPref>(loadNativeBackendPref());
  let nativePlaybackSaving = $state(false);
  let transcodingSettings = $state<TranscodingSettings>(
    mergeTranscodingSettings(music.transcodingSettings),
  );
  let immersiveAudioSettings = $state<ImmersiveAudioSettings>(
    mergeImmersiveAudioSettings(music.immersiveAudioSettings),
  );
  let queueSettings = $state<QueueSettings>(
    mergeQueueSettings(music.queueSettings),
  );
  let playbackSettings = $state<PlaybackSettings>(
    mergePlaybackSettings(music.playbackSettings),
  );

  const transcodeFormatOptions = (
    Object.entries(TRANSCODE_FORMAT_LABELS) as [TranscodeFormat, string][]
  ).map(([value, label]) => ({ value, label }));

  const immersiveModeOptions = (
    Object.entries(IMMERSIVE_AUDIO_MODE_LABELS) as [
      ImmersiveAudioMode,
      string,
    ][]
  ).map(([value, label]) => ({ value, label }));

  const queueSizeOptions = QUEUE_SIZE_OPTIONS.map((option) => ({
    value: String(option.value),
    label: option.label,
  }));

  const crossfadeDurationOptions = CROSSFADE_DURATION_OPTIONS.map((option) => ({
    value: String(option.value),
    label: option.label,
  }));

  const transcodeBitrateOptions = TRANSCODE_BITRATE_OPTIONS.map((option) => ({
    value: String(option.value),
    label: option.label,
  }));

  const nativeBackendOptions = $derived<
    { value: NativeBackendPref; label: string; disabled?: boolean }[]
  >([
    { value: "auto", label: "Auto (prefer mpv)" },
    {
      value: "mpv",
      label: `mpv${music.mpvAvailable ? "" : " (libmpv not found)"}`,
      disabled: !music.mpvAvailable,
    },
    {
      value: "vlc",
      label: `VLC${music.vlcAvailable ? "" : " (libvlc not found)"}`,
      disabled: !music.vlcAvailable,
    },
  ]);

  $effect(() => {
    transcodingSettings = mergeTranscodingSettings(music.transcodingSettings);
    immersiveAudioSettings = mergeImmersiveAudioSettings(
      music.immersiveAudioSettings,
    );
    playbackSettings = mergePlaybackSettings(music.playbackSettings);
    queueSettings = mergeQueueSettings(music.queueSettings);
    nativeBackendPref = loadNativeBackendPref();
  });

  $effect(() => {
    if (nativeDesktopAvailable()) {
      void music.refreshNativeCapabilities();
    }
  });

  function applyTranscodingSettings() {
    music.updateTranscodingSettings(transcodingSettings);
    toast.success("Transcoding settings saved");
  }

  function applyImmersiveAudioSettings() {
    music.updateImmersiveAudioSettings(immersiveAudioSettings);
    toast.success("Immersive audio settings saved");
  }

  function applyQueueSettings() {
    music.updateQueueSettings(queueSettings);
    toast.success("Queue settings saved");
  }

  function savePlaybackSettings() {
    music.updatePlaybackSettings(playbackSettings);
    toast.success("Playback settings saved");
  }

  function saveQueueSize(value: number) {
    queueSettings = mergeQueueSettings({ maxQueueSize: value });
    applyQueueSettings();
  }

  function saveTranscoding(patch: Partial<TranscodingSettings>) {
    transcodingSettings = mergeTranscodingSettings({
      ...transcodingSettings,
      ...patch,
    });
    applyTranscodingSettings();
  }

  function saveImmersive(patch: Partial<ImmersiveAudioSettings>) {
    immersiveAudioSettings = mergeImmersiveAudioSettings({
      ...immersiveAudioSettings,
      ...patch,
    });
    applyImmersiveAudioSettings();
  }

  async function applyNativeBackend(backend: NativeBackendPref) {
    nativePlaybackSaving = true;
    try {
      nativeBackendPref = backend;
      saveNativeBackendPref(backend);
      await music.setNativeBackend(backend);
      if (music.nativeInitError) {
        toast.error(music.nativeInitError);
        return;
      }
      if (!music.nativeAvailable) {
        toast.error("Could not initialize the selected native backend");
        return;
      }
      const label =
        music.nativeBackend === "vlc"
          ? "VLC"
          : music.nativeBackend === "mpv"
            ? "mpv"
            : backend;
      toast.success(`Native backend set to ${label}`);
    } finally {
      nativePlaybackSaving = false;
    }
  }

  async function applyNativePlayback(enabled: boolean) {
    if (enabled && !music.nativeAvailable) {
      toast.error(
        "No working native playback backend is available on this system",
      );
      return;
    }
    nativePlaybackSaving = true;
    try {
      saveNativePlaybackPref(enabled);
      await music.setNativePlaybackEnabled(enabled);
      toast.success(
        enabled ? "Native playback enabled" : "Web playback enabled",
      );
    } finally {
      nativePlaybackSaving = false;
    }
  }
</script>

{#if !isWailsMobile() && (music.nativeAvailable || music.mpvAvailable || music.vlcAvailable)}
  <SettingsCard
    title="Native playback"
    description="Use libmpv or VLC for broad codec support. The equalizer and crossfade are disabled while native playback is active."
  >
    <SettingsToggleRow
      label="Use native playback"
      description="Stream through a native audio backend instead of the browser."
      checked={music.nativePlayback}
      disabled={nativePlaybackSaving || !music.nativeAvailable}
      onchange={(checked) => void applyNativePlayback(checked)}
    />

    {#if !music.nativeAvailable}
      <p class="settings-page__meta">
        {#if music.mpvAvailable || music.vlcAvailable}
          {#if music.mpvLoadError}
            mpv load failed: {music.mpvLoadError}
          {:else if music.nativeInitError}
            {music.nativeInitError}
          {:else}
            A native library was detected but could not be initialized. Install
            mpv or VLC with its plugins, or use web playback.
          {/if}
        {:else}
          Install libmpv or VLC on this system to enable native playback. The
          mpv package must include libmpv (not just the mpv binary).
        {/if}
      </p>
    {/if}

    {#if music.mpvAvailable || music.vlcAvailable}
      <Field
        label="Native backend"
        hint="Choose mpv, VLC, or Auto. Auto tries mpv first when both libraries are installed."
      >
        <Select
          value={nativeBackendPref}
          options={nativeBackendOptions}
          disabled={nativePlaybackSaving}
          onchange={(backend) => void applyNativeBackend(backend)}
        />
      </Field>
      {#if music.nativeAvailable && music.nativeBackend}
        <p class="settings-page__meta">
          Active backend: {music.nativeBackend === "vlc" ? "VLC" : "mpv"}
        </p>
      {/if}
    {/if}

    {#if music.nativePlayback}
      <p class="settings-page__meta">
        Native playback is active
        {#if music.nativeBackend}
          ({music.nativeBackend === "vlc" ? "VLC" : "mpv"}).
        {/if}
      </p>
    {:else if music.nativeAvailable}
      <p class="settings-page__meta">
        Native playback is available but web playback is selected.
      </p>
    {:else}
      <p class="settings-page__meta">
        Selected native backend is unavailable. Choose another backend or use
        web playback.
      </p>
    {/if}
  </SettingsCard>
{/if}

<SettingsCard
  title="Immersive audio"
  description={`Surround PCM, Dolby/DTS bitstream passthrough for Atmos-capable receivers, and headphone binaural crossfeed. ${APP_NAME} does not ship a licensed Dolby decoder. Atmos object audio is preserved only when passthrough reaches a receiver that can decode it.`}
>
  <Field
    label="Output mode"
    hint={IMMERSIVE_AUDIO_MODE_HINTS[immersiveAudioSettings.mode]}
  >
    <Select
      value={immersiveAudioSettings.mode}
      options={immersiveModeOptions}
      onchange={(mode) => saveImmersive({ mode })}
    />
  </Field>

  <SettingsToggleRow
    label="Exclusive output for passthrough"
    description="Ask the OS for exclusive device access when using Dolby/DTS passthrough. Helps HDMI and WASAPI receivers lock the bitstream."
    checked={immersiveAudioSettings.exclusiveOutput}
    disabled={immersiveAudioSettings.mode !== "passthrough"}
    onchange={(checked) => saveImmersive({ exclusiveOutput: checked })}
  />

  <SettingsToggleRow
    label="Preserve immersive streams"
    description="Do not force stereo MP3 transcoding for multichannel, Atmos-named, or surround bitstream tracks."
    checked={immersiveAudioSettings.preserveImmersiveStreams}
    onchange={(checked) => saveImmersive({ preserveImmersiveStreams: checked })}
  />

  {#if immersiveAudioSettings.mode === "passthrough" || immersiveAudioSettings.mode === "surround"}
    <p class="settings-page__meta">
      {#if music.nativePlayback && music.nativeBackend === "mpv"}
        Native mpv is active. Surround and passthrough can use the system audio
        device.
      {:else if music.nativeAvailable}
        Enable native playback (mpv preferred) for surround PCM and Dolby/DTS
        passthrough. The web player cannot bitstream Atmos to an AVR.
      {:else}
        Install libmpv and enable native playback for surround and passthrough.
        Headphone binaural still works in the web player.
      {/if}
    </p>
  {/if}
</SettingsCard>

{#if !isWailsMobile() && music.mpvAvailable}
  <SettingsCard
    title="Remote audio output"
    description="Send decoded audio to extra targets in addition to the speakers. Applies to native mpv playback only. The stream is raw 48 kHz s16 stereo PCM."
  >
    <Field
      label="Output targets"
      hint="Comma separated: device, stdout, fifo:/path, tcp:host:port, tcp-listen:0.0.0.0:4987, unix:/path.sock, unix-listen:/path.sock"
    >
      <input
        class="settings-input"
        type="text"
        placeholder="tcp-listen:0.0.0.0:4987"
        value={immersiveAudioSettings.remoteOutputs}
        onchange={(e) =>
          saveImmersive({
            remoteOutputs: (e.currentTarget as HTMLInputElement).value,
          })}
      />
    </Field>
  </SettingsCard>
{/if}

<SettingsCard
  title="Session"
  description={`Control how ${APP_NAME} resumes when you reopen the app.`}
>
  <SettingsToggleRow
    label="Continue playback on launch"
    description="Restore your queue and playback position from the last session."
    checked={playbackSettings.continuePlaybackOnLaunch}
    onchange={(checked) => {
      playbackSettings = {
        ...playbackSettings,
        continuePlaybackOnLaunch: checked,
      };
      savePlaybackSettings();
    }}
  />
</SettingsCard>

<SettingsCard
  title="Playback queue"
  description={`Limit how many tracks ${APP_NAME} keeps in the playback queue. Continuous modes (library shuffle, personal radio, random radio) refill up to this size as you listen.`}
>
  <Field
    label="Maximum queue size"
    hint="Unlimited allows the queue to grow without a cap."
  >
    <Select
      value={String(queueSettings.maxQueueSize)}
      options={queueSizeOptions}
      onchange={(value) => saveQueueSize(Number(value))}
    />
  </Field>
</SettingsCard>

<SettingsCard
  title="Crossfade"
  description="Blend the end of one track into the start of the next during sequential playback."
>
  <SettingsToggleRow
    label="Crossfade between tracks"
    description="Web playback only. Shuffle, repeat-one, and native playback disable crossfade."
    checked={playbackSettings.crossfadeEnabled}
    disabled={music.nativePlayback}
    onchange={(checked) => {
      playbackSettings = {
        ...playbackSettings,
        crossfadeEnabled: checked,
      };
      savePlaybackSettings();
    }}
  />

  <Field
    label="Crossfade duration"
    hint="How long tracks overlap before the next song takes over."
  >
    <Select
      value={String(playbackSettings.crossfadeDurationSec)}
      options={crossfadeDurationOptions}
      disabled={!playbackSettings.crossfadeEnabled || music.nativePlayback}
      onchange={(value) => {
        playbackSettings = {
          ...playbackSettings,
          crossfadeDurationSec: Number(value),
        };
        savePlaybackSettings();
      }}
    />
  </Field>
</SettingsCard>

<SettingsCard
  title="Streaming and transcoding"
  description={`Control how ${APP_NAME} requests audio from your server.`}
>
  <SettingsToggleRow
    label="Always transcode streams"
    description="Request transcoded audio even when the browser might decode the original format."
    checked={transcodingSettings.alwaysTranscode}
    onchange={(checked) => saveTranscoding({ alwaysTranscode: checked })}
  />

  <Field
    label="Max bitrate"
    hint="Limits transcoded stream quality. Off uses 320 kbps only when always transcoding or after a decode failure."
  >
    <Select
      value={String(transcodingSettings.maxBitRate)}
      options={transcodeBitrateOptions}
      onchange={(value) => saveTranscoding({ maxBitRate: Number(value) })}
    />
  </Field>

  <Field
    label="Preferred format"
    hint="Sent to the server as the stream format when transcoding."
  >
    <Select
      value={transcodingSettings.format}
      options={transcodeFormatOptions}
      onchange={(format) => saveTranscoding({ format })}
    />
  </Field>
</SettingsCard>
