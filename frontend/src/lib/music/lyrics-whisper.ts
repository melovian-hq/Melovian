// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { ApiPaths } from "$lib/core/http/api-paths";
import { apiHeaders } from "$lib/core/http/client";
import { isRemoteClient, resolveApiUrl } from "$lib/config/remote-server";
import { normalizeParsedLyrics, type ParsedLyrics } from "$lib/music/lyrics";

export interface WhisperSegment {
  startMs: number;
  text: string;
}

/**
 * WhisperClientEngine is implemented by optional WASM packages (for example a
 * transcriptasm build) that transcribe 16 kHz mono PCM in the browser. When
 * registered, transcription stays on the client and only the timed segments
 * are posted to the server for storage.
 */
export interface WhisperClientEngine {
  transcribe(pcm: Float32Array, sampleRate: number): Promise<WhisperSegment[]>;
}

let clientEngine: WhisperClientEngine | null = null;

export function registerWhisperClientEngine(engine: WhisperClientEngine): void {
  clientEngine = engine;
}

export function hasWhisperClientEngine(): boolean {
  return clientEngine != null;
}

const WHISPER_SAMPLE_RATE = 16000;
const WHISPER_REQUEST_TIMEOUT_MS = 10 * 60 * 1000;

async function fetchAudioBuffer(url: string): Promise<ArrayBuffer> {
  const response = await fetch(url, {
    credentials: isRemoteClient() ? "include" : "same-origin",
  });
  if (!response.ok) {
    throw new Error(
      `Could not download track audio (status ${response.status})`,
    );
  }
  return response.arrayBuffer();
}

async function decodeToMonoPcm(data: ArrayBuffer): Promise<Float32Array> {
  const decodeContext = new AudioContext();
  let buffer: AudioBuffer;
  try {
    buffer = await decodeContext.decodeAudioData(data);
  } finally {
    void decodeContext.close();
  }
  const offline = new OfflineAudioContext(
    1,
    Math.ceil(buffer.duration * WHISPER_SAMPLE_RATE),
    WHISPER_SAMPLE_RATE,
  );
  const source = offline.createBufferSource();
  source.buffer = buffer;
  source.connect(offline.destination);
  source.start();
  const rendered = await offline.startRendering();
  return rendered.getChannelData(0);
}

/** Encode 16 kHz mono float PCM as a 16-bit WAV blob. */
export function encodeWav16(
  pcm: Float32Array,
  sampleRate = WHISPER_SAMPLE_RATE,
): Blob {
  const dataLength = pcm.length * 2;
  const buffer = new ArrayBuffer(44 + dataLength);
  const view = new DataView(buffer);
  const writeString = (offset: number, value: string) => {
    for (let i = 0; i < value.length; i++) {
      view.setUint8(offset + i, value.charCodeAt(i));
    }
  };
  writeString(0, "RIFF");
  view.setUint32(4, 36 + dataLength, true);
  writeString(8, "WAVE");
  writeString(12, "fmt ");
  view.setUint32(16, 16, true);
  view.setUint16(20, 1, true);
  view.setUint16(22, 1, true);
  view.setUint32(24, sampleRate, true);
  view.setUint32(28, sampleRate * 2, true);
  view.setUint16(32, 2, true);
  view.setUint16(34, 16, true);
  writeString(36, "data");
  view.setUint32(40, dataLength, true);
  for (let i = 0; i < pcm.length; i++) {
    const sample = Math.max(-1, Math.min(1, pcm[i] ?? 0));
    view.setInt16(
      44 + i * 2,
      sample < 0 ? sample * 0x8000 : sample * 0x7fff,
      true,
    );
  }
  return new Blob([buffer], { type: "audio/wav" });
}

export type WhisperTrackMeta = {
  artist?: string;
  title?: string;
  album?: string;
  durationSec?: number;
  language?: string;
};

async function postWhisperRequest(
  input: RequestInit,
  trackId: string,
): Promise<ParsedLyrics | null> {
  const controller = new AbortController();
  const timer = setTimeout(
    () => controller.abort(),
    WHISPER_REQUEST_TIMEOUT_MS,
  );
  try {
    const response = await fetch(
      resolveApiUrl(ApiPaths.musicLyricsWhisper(trackId)),
      {
        credentials: isRemoteClient() ? "include" : "same-origin",
        ...input,
        signal: controller.signal,
        headers: { ...apiHeaders(), ...(input.headers ?? {}) },
      },
    );
    if (response.status === 404) return null;
    const payload = (await response.json().catch(() => null)) as
      (Partial<ParsedLyrics> & { message?: string }) | null;
    if (!response.ok) {
      throw new Error(payload?.message ?? "Whisper transcription failed");
    }
    return normalizeParsedLyrics(payload ?? null);
  } finally {
    clearTimeout(timer);
  }
}

/**
 * Send already-transcribed segments to the server for storage as synced
 * lyrics. Used by client-side whisper engines.
 */
export async function saveWhisperSegments(
  trackId: string,
  meta: WhisperTrackMeta,
  segments: WhisperSegment[],
): Promise<ParsedLyrics | null> {
  return postWhisperRequest(
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ ...meta, segments }),
    },
    trackId,
  );
}

async function postWhisperAudio(
  trackId: string,
  meta: WhisperTrackMeta,
  wav: Blob,
): Promise<ParsedLyrics | null> {
  const form = new FormData();
  form.append("file", wav, "track.wav");
  if (meta.artist) form.append("artist", meta.artist);
  if (meta.title) form.append("title", meta.title);
  if (meta.album) form.append("album", meta.album);
  return postWhisperRequest({ method: "POST", body: form }, trackId);
}

/**
 * Transcribe a track to synced lyrics. Downloads the audio from the resolved
 * stream URL, decodes to 16 kHz mono PCM, then either runs a registered
 * client-side whisper engine or uploads a WAV to the server, which forwards
 * it to the configured whisper.cpp server.
 */
export async function transcribeTrackLyrics(
  trackId: string,
  audioUrl: string,
  meta: WhisperTrackMeta,
): Promise<ParsedLyrics | null> {
  const data = await fetchAudioBuffer(audioUrl);
  const pcm = await decodeToMonoPcm(data);
  const engine = clientEngine;
  if (engine) {
    const segments = await engine.transcribe(pcm, WHISPER_SAMPLE_RATE);
    return saveWhisperSegments(trackId, meta, segments);
  }
  return postWhisperAudio(trackId, meta, encodeWav16(pcm));
}
