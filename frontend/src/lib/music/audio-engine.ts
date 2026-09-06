// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { EQ_BAND_COUNT, defaultBandParams, type EqSettings } from "./eq";
import type {
  ImmersiveAudioMode,
  ImmersiveAudioSettings,
} from "./immersive-audio-settings";
import type { PlaybackEngine } from "./playback-engine";
import { setMobileMediaPlaying } from "$lib/config/runtime";

type Handler = () => void;

type CrossfeedNodes = {
  splitter: ChannelSplitterNode;
  merger: ChannelMergerNode;
  delayL: DelayNode;
  delayR: DelayNode;
  crossL: GainNode;
  crossR: GainNode;
  dryL: GainNode;
  dryR: GainNode;
};

/**
 * AudioEngine drives playback with two HTMLAudioElements that share a single
 * Web Audio EQ graph. The second element pre-buffers the next track so that
 * advancing performs a pointer swap instead of a fresh load, yielding a
 * gapless transition without rewiring the audio graph.
 */
export class AudioEngine implements PlaybackEngine {
  readonly usesNative = false;
  /**
   * DECLICK_SEC is a short master-gain fade applied around transport changes
   * (loads, swaps, and playback start) so cutting a playing waveform never
   * produces a click or pop. It is short enough to feel instant.
   */
  private static readonly DECLICK_SEC = 0.018;
  private static readonly CROSSFEED_DELAY_SEC = 0.0003;
  private static readonly CROSSFEED_AMOUNT = 0.4;
  private elements: HTMLAudioElement[];
  private activeIndex = 0;
  private context: AudioContext | null = null;
  private sources: (MediaElementAudioSourceNode | null)[] = [null, null];
  private sourceGains: (GainNode | null)[] = [null, null];
  private filters: BiquadFilterNode[] = [];
  private gainNode: GainNode | null = null;
  private crossfeed: CrossfeedNodes | null = null;
  private immersiveMode: ImmersiveAudioMode = "auto";
  private loadedSrc = "";
  private preparedUrl = "";
  private currentVolume = 1;
  private prefetchEls = new Map<string, HTMLAudioElement>();
  private crossfadeInFlight = false;
  private suppressEndedIndex: number | null = null;

  private endedHandlers = new Set<Handler>();
  private timeHandlers = new Set<Handler>();
  private metaHandlers = new Set<Handler>();
  private errorHandlers = new Set<Handler>();

  constructor() {
    this.elements = [this.createElement(), this.createElement()];
    this.attachInternalListeners();
  }

  private createElement(): HTMLAudioElement {
    const el = new Audio();
    el.preload = "auto";
    el.crossOrigin = "anonymous";
    return el;
  }

  private get active(): HTMLAudioElement {
    return this.elements[this.activeIndex];
  }

  private get standby(): HTMLAudioElement {
    return this.elements[1 - this.activeIndex];
  }

  private elementIndex(el: HTMLAudioElement): number {
    return el === this.elements[0] ? 0 : 1;
  }

  /** audio exposes the active element for media-session and diagnostics. */
  get audio(): HTMLAudioElement {
    return this.active;
  }

  private attachInternalListeners() {
    for (const el of this.elements) {
      el.addEventListener("ended", () => {
        const index = this.elementIndex(el);
        if (this.suppressEndedIndex === index) return;
        if (el === this.active) this.endedHandlers.forEach((h) => h());
      });
      el.addEventListener("timeupdate", () => {
        if (el === this.active) this.timeHandlers.forEach((h) => h());
      });
      el.addEventListener("loadedmetadata", () => {
        if (el === this.active) this.metaHandlers.forEach((h) => h());
      });
      el.addEventListener("error", () => {
        if (el === this.active) this.errorHandlers.forEach((h) => h());
      });
    }
  }

  private ensureContext() {
    if (this.context) return;
    try {
      this.context = new AudioContext();
      this.gainNode = this.context.createGain();
      this.gainNode.gain.value = 1;

      this.filters = defaultBandParams().map((band) => {
        const filter = this.context!.createBiquadFilter();
        filter.type = "peaking";
        filter.frequency.value = band.frequency;
        filter.Q.value = band.q;
        filter.gain.value = 0;
        return filter;
      });

      for (let i = 0; i < this.filters.length - 1; i++) {
        this.filters[i].connect(this.filters[i + 1]);
      }
      this.wireOutputGraph();

      this.elements.forEach((el, i) => {
        const src = this.context!.createMediaElementSource(el);
        const sourceGain = this.context!.createGain();
        sourceGain.gain.value = 1;
        src.connect(sourceGain);
        sourceGain.connect(this.filters[0]);
        this.sources[i] = src;
        this.sourceGains[i] = sourceGain;
      });

      for (const el of this.elements) el.volume = 1;
      this.gainNode.gain.value = this.currentVolume;
    } catch {
      this.context = null;
      this.source_reset();
    }
  }

  private buildCrossfeed(ctx: AudioContext): CrossfeedNodes {
    const splitter = ctx.createChannelSplitter(2);
    const merger = ctx.createChannelMerger(2);
    const delayL = ctx.createDelay(0.01);
    const delayR = ctx.createDelay(0.01);
    delayL.delayTime.value = AudioEngine.CROSSFEED_DELAY_SEC;
    delayR.delayTime.value = AudioEngine.CROSSFEED_DELAY_SEC;
    const crossL = ctx.createGain();
    const crossR = ctx.createGain();
    const dryL = ctx.createGain();
    const dryR = ctx.createGain();
    crossL.gain.value = AudioEngine.CROSSFEED_AMOUNT;
    crossR.gain.value = AudioEngine.CROSSFEED_AMOUNT;
    dryL.gain.value = 1;
    dryR.gain.value = 1;

    splitter.connect(dryL, 0);
    splitter.connect(dryR, 1);
    splitter.connect(delayR, 0);
    splitter.connect(delayL, 1);
    delayL.connect(crossL);
    delayR.connect(crossR);
    dryL.connect(merger, 0, 0);
    crossL.connect(merger, 0, 0);
    dryR.connect(merger, 0, 1);
    crossR.connect(merger, 0, 1);

    return { splitter, merger, delayL, delayR, crossL, crossR, dryL, dryR };
  }

  private wireOutputGraph() {
    if (!this.context || !this.gainNode || this.filters.length === 0) return;
    const lastFilter = this.filters[this.filters.length - 1];
    lastFilter.disconnect();
    this.gainNode.disconnect();
    if (this.crossfeed) {
      this.crossfeed.splitter.disconnect();
      this.crossfeed.merger.disconnect();
      this.crossfeed = null;
    }

    if (this.immersiveMode === "binaural") {
      this.crossfeed = this.buildCrossfeed(this.context);
      lastFilter.connect(this.crossfeed.splitter);
      this.crossfeed.merger.connect(this.gainNode);
    } else {
      lastFilter.connect(this.gainNode);
    }
    this.gainNode.connect(this.context.destination);
  }

  private source_reset() {
    this.sources = [null, null];
    this.sourceGains = [null, null];
    this.gainNode = null;
    this.filters = [];
    this.crossfeed = null;
  }

  async resumeContext() {
    if (!this.context) return;
    if (this.context.state === "suspended") {
      await this.context.resume();
    }
  }

  setVolume(value: number) {
    this.currentVolume = value;
    if (this.gainNode && this.context) {
      for (const el of this.elements) el.volume = 1;
      const now = this.context.currentTime;
      this.gainNode.gain.cancelScheduledValues(now);
      this.gainNode.gain.setValueAtTime(value, now);
      return;
    }
    for (const el of this.elements) el.volume = value;
  }

  /**
   * rampMasterGain schedules a short linear fade of the master gain to target,
   * cancelling any in-flight automation first so transitions never fight each
   * other and leave the gain stuck at a stale value.
   */
  private rampMasterGain(
    target: number,
    seconds = AudioEngine.DECLICK_SEC,
  ): void {
    if (!this.context || !this.gainNode) return;
    const now = this.context.currentTime;
    const gain = this.gainNode.gain;
    gain.cancelScheduledValues(now);
    gain.setValueAtTime(gain.value, now);
    gain.linearRampToValueAtTime(target, now + seconds);
  }

  /**
   * silenceForSwap fades the master gain to zero and waits for the fade to
   * complete so the outgoing element can be paused or reloaded without a click.
   * Resolves immediately when no audio graph exists yet.
   */
  private async silenceForSwap(): Promise<void> {
    if (!this.context || !this.gainNode) return;
    this.rampMasterGain(0);
    await new Promise<void>((resolve) => {
      window.setTimeout(resolve, AudioEngine.DECLICK_SEC * 1000 + 5);
    });
  }

  applyEq(settings: EqSettings) {
    if (!settings.enabled) return;
    this.ensureContext();
    if (!this.context || this.filters.length !== EQ_BAND_COUNT) return;
    for (let i = 0; i < this.filters.length; i++) {
      const band = settings.bands[i];
      if (settings.enabled) {
        this.filters[i].frequency.value = band.frequency;
        this.filters[i].Q.value = band.q;
        this.filters[i].gain.value = band.gain;
      } else {
        this.filters[i].gain.value = 0;
      }
    }
  }

  applyImmersiveAudio(settings: ImmersiveAudioSettings) {
    const nextMode = settings.mode;
    if (this.immersiveMode === nextMode && this.context) return;
    this.immersiveMode = nextMode;
    if (!this.context) {
      if (nextMode === "binaural") this.ensureContext();
      return;
    }
    this.wireOutputGraph();
  }

  canPlay(): boolean {
    return (
      this.loadedSrc !== "" &&
      this.active.readyState >= HTMLMediaElement.HAVE_FUTURE_DATA
    );
  }

  private loadInto(el: HTMLAudioElement, url: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const cleanup = () => {
        el.removeEventListener("canplay", onCanPlay);
        el.removeEventListener("error", onError);
      };
      const onCanPlay = () => {
        cleanup();
        resolve();
      };
      const onError = () => {
        cleanup();
        const code = el.error?.code ?? 0;
        reject(
          new Error(
            `Audio failed to load (${code})${el.currentSrc ? `: ${el.currentSrc}` : ""}`,
          ),
        );
      };
      el.addEventListener("canplay", onCanPlay);
      el.addEventListener("error", onError);
      // When the Web Audio graph is active, the audio element must feed it at
      // full level and the master gain node controls volume. Otherwise the
      // element's own volume is the only control.
      el.volume = this.context ? 1 : this.currentVolume;
      el.src = url;
      el.load();
    });
  }

  async loadSource(url: string): Promise<void> {
    if (this.loadedSrc === url && this.canPlay()) {
      return;
    }
    this.cancelCrossfade();
    await this.silenceForSwap();
    this.active.pause();
    this.preparedUrl = "";
    this.releaseStandby();
    await this.loadInto(this.active, url);
    this.loadedSrc = url;
    this.resetSourceGains();
  }

  /**
   * prepareNext buffers an upcoming track in the standby element so the next
   * advance can swap to it instantly. Passing an empty or already-loaded url
   * clears any pending preparation and drops standby media buffers.
   */
  prepareNext(url: string): void {
    if (!url || url === this.loadedSrc) {
      this.preparedUrl = "";
      this.releaseStandby();
      return;
    }
    if (this.preparedUrl === url) return;
    this.preparedUrl = url;
    void this.loadInto(this.standby, url).catch(() => {
      if (this.preparedUrl === url) this.preparedUrl = "";
    });
  }

  /** Drop standby media buffers when nothing is prepared for gapless advance. */
  private releaseStandby(): void {
    const el = this.standby;
    el.pause();
    if (el.getAttribute("src") || el.src) {
      el.removeAttribute("src");
      el.load();
    }
  }

  hasPrepared(url: string): boolean {
    return (
      url !== "" &&
      this.preparedUrl === url &&
      this.standby.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA
    );
  }

  private resetSourceGains() {
    if (!this.context) return;
    const now = this.context.currentTime;
    for (const gain of this.sourceGains) {
      if (gain) {
        gain.gain.cancelScheduledValues(now);
        gain.gain.setValueAtTime(1, now);
      }
    }
  }

  private cancelCrossfade() {
    this.crossfadeInFlight = false;
    this.suppressEndedIndex = null;
    this.resetSourceGains();
  }

  /**
   * activatePrepared swaps to the pre-buffered standby element. When
   * crossfadeSec is greater than zero, ramps between the outgoing and incoming
   * tracks instead of cutting instantly.
   */
  async activatePrepared(url: string, crossfadeSec = 0): Promise<boolean> {
    if (!this.hasPrepared(url)) return false;
    if (crossfadeSec > 0) {
      return this.crossfadePrepared(url, crossfadeSec);
    }
    return this.instantPrepared(url);
  }

  private async instantPrepared(url: string): Promise<boolean> {
    const next = this.standby;
    await this.silenceForSwap();
    this.active.pause();
    this.activeIndex = 1 - this.activeIndex;
    this.loadedSrc = url;
    this.preparedUrl = "";
    this.resetSourceGains();
    try {
      try {
        next.currentTime = 0;
      } catch {
        /* some browsers disallow seeking until metadata is ready */
      }
      if (this.context) await this.resumeContext();
      await next.play();
      return true;
    } finally {
      this.rampMasterGain(this.currentVolume);
    }
  }

  private async crossfadePrepared(
    url: string,
    crossfadeSec: number,
  ): Promise<boolean> {
    this.ensureContext();
    if (!this.context) {
      return this.instantPrepared(url);
    }

    const outgoingIndex = this.activeIndex;
    const incomingIndex = 1 - outgoingIndex;
    const outgoing = this.elements[outgoingIndex];
    const incoming = this.elements[incomingIndex];
    const outGain = this.sourceGains[outgoingIndex];
    const inGain = this.sourceGains[incomingIndex];
    if (!outGain || !inGain) {
      return this.instantPrepared(url);
    }

    this.crossfadeInFlight = true;
    this.suppressEndedIndex = outgoingIndex;

    const duration = Math.max(0.25, crossfadeSec);
    const now = this.context.currentTime;

    try {
      await this.resumeContext();
      this.rampMasterGain(this.currentVolume);
      try {
        incoming.currentTime = 0;
      } catch {
        /* ignore */
      }
      inGain.gain.cancelScheduledValues(now);
      outGain.gain.cancelScheduledValues(now);
      inGain.gain.setValueAtTime(0, now);
      outGain.gain.setValueAtTime(1, now);
      await incoming.play();
      inGain.gain.linearRampToValueAtTime(1, now + duration);
      outGain.gain.linearRampToValueAtTime(0, now + duration);

      await new Promise<void>((resolve) => {
        window.setTimeout(resolve, duration * 1000);
      });

      outgoing.pause();
      this.activeIndex = incomingIndex;
      this.loadedSrc = url;
      this.preparedUrl = "";
      inGain.gain.setValueAtTime(1, this.context.currentTime);
      outGain.gain.setValueAtTime(0, this.context.currentTime);
      return true;
    } catch {
      outgoing.pause();
      this.resetSourceGains();
      return false;
    } finally {
      this.crossfadeInFlight = false;
      this.suppressEndedIndex = null;
    }
  }

  /**
   * prefetch warms the browser media cache for additional track URLs (for
   * example the previous track) without occupying the standby element.
   */
  prefetch(urls: string[]): void {
    const wanted = urls.filter(
      (url) => url && url !== this.loadedSrc && url !== this.preparedUrl,
    );

    for (const [url, el] of this.prefetchEls) {
      if (!wanted.includes(url)) {
        el.removeAttribute("src");
        el.load();
        this.prefetchEls.delete(url);
      }
    }

    for (const url of wanted) {
      if (this.prefetchEls.has(url)) continue;
      const el = new Audio();
      el.preload = "auto";
      el.crossOrigin = "anonymous";
      el.src = url;
      el.load();
      this.prefetchEls.set(url, el);
    }
  }

  async play() {
    this.ensureContext();
    if (this.context) {
      await this.resumeContext();
    }
    const result = this.active.play();
    this.rampMasterGain(this.currentVolume);
    setMobileMediaPlaying(true);
    return result;
  }

  pause() {
    this.cancelCrossfade();
    for (const el of this.elements) {
      el.pause();
    }
    setMobileMediaPlaying(false);
  }

  get currentTime() {
    return this.active.currentTime;
  }

  get duration() {
    return this.active.duration || 0;
  }

  get mediaErrorCode(): number {
    return this.active.error?.code ?? 0;
  }

  isNetworkError(): boolean {
    return this.mediaErrorCode === MediaError.MEDIA_ERR_NETWORK;
  }

  isDecodeError(): boolean {
    const code = this.mediaErrorCode;
    return (
      code === MediaError.MEDIA_ERR_DECODE ||
      code === MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED
    );
  }

  getLastError(): string {
    return this.active.error?.message ?? "";
  }

  set currentTime(value: number) {
    this.cancelCrossfade();
    this.active.currentTime = value;
  }

  set src(value: string) {
    this.active.src = value;
    this.loadedSrc = value;
  }

  onEnded(handler: Handler) {
    this.endedHandlers.add(handler);
    return () => this.endedHandlers.delete(handler);
  }

  onTimeUpdate(handler: Handler) {
    this.timeHandlers.add(handler);
    return () => this.timeHandlers.delete(handler);
  }

  onLoadedMetadata(handler: Handler) {
    this.metaHandlers.add(handler);
    return () => this.metaHandlers.delete(handler);
  }

  onError(handler: Handler) {
    this.errorHandlers.add(handler);
    return () => this.errorHandlers.delete(handler);
  }

  destroy() {
    this.cancelCrossfade();
    for (const el of this.elements) {
      el.pause();
      el.removeAttribute("src");
      el.load();
    }
    this.loadedSrc = "";
    this.preparedUrl = "";
    for (const el of this.prefetchEls.values()) {
      el.removeAttribute("src");
      el.load();
    }
    this.prefetchEls.clear();
    if (this.context) {
      void this.context.close();
    }
  }
}
