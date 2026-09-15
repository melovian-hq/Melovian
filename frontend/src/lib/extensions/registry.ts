// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { isLocalMusicId } from "$lib/music/library-adapter";
import { ApiPaths } from "$lib/core/http/api-paths";
import { parseJson } from "$lib/core/http/parse";
import { isRemoteClient, resolveApiUrl } from "$lib/config/remote-server";
import { extensionFeatures } from "./features.svelte";
import { extensionsPayloadSchema } from "./schemas";
import { syncExtensionStyles } from "./styles";
import type {
  DecorateTrackContext,
  ExtensionAPI,
  ExtensionManifest,
  TrackDecoration,
  TrackRule,
} from "./types";

type ScriptHook = (api: ExtensionAPI, ctx: DecorateTrackContext) => void;

const declarativeRules: TrackRule[] = [];
const scriptDecorators: Array<
  (ctx: DecorateTrackContext) => TrackDecoration | null | undefined
> = [];

function contains(haystack: string | undefined, needle: string | undefined) {
  if (!needle) return true;
  return (haystack ?? "").toLowerCase().includes(needle.toLowerCase());
}

function matchesRule(ctx: DecorateTrackContext, match: TrackRule["match"]) {
  if (match.isLocal != null && match.isLocal !== ctx.isLocal) return false;
  if (!contains(ctx.track.title, match.titleContains)) return false;
  if (!contains(ctx.track.artist, match.artistContains)) return false;
  if (!contains(ctx.track.album, match.albumContains)) return false;
  const genre = ctx.genre ?? ctx.track.genre;
  if (!contains(genre, match.genreContains)) return false;
  if (match.titleRegex) {
    try {
      if (!new RegExp(match.titleRegex, "i").test(ctx.track.title ?? "")) {
        return false;
      }
    } catch {
      return false;
    }
  }
  if (match.tagEquals) {
    const tags = ctx.tags ?? [];
    if (
      !tags.some((tag) => tag.toLowerCase() === match.tagEquals!.toLowerCase())
    ) {
      return false;
    }
  }
  if (match.minRating != null && (ctx.rating ?? 0) < match.minRating)
    return false;
  return true;
}

function mergeDecorations(
  base: TrackDecoration,
  next: TrackDecoration,
): TrackDecoration {
  return {
    progressColor: next.progressColor ?? base.progressColor,
    progressGradient: next.progressGradient ?? base.progressGradient,
    progressThumbUrl: next.progressThumbUrl ?? base.progressThumbUrl,
    progressParticleUrl: next.progressParticleUrl ?? base.progressParticleUrl,
    icon: next.icon ?? base.icon,
    iconUrl: next.iconUrl ?? base.iconUrl,
    titlePrefix: next.titlePrefix ?? base.titlePrefix,
    coverOverlayIcon: next.coverOverlayIcon ?? base.coverOverlayIcon,
    playerTheme: next.playerTheme ?? base.playerTheme,
  };
}

export function decorateTrack(
  track: DecorateTrackContext["track"],
  extra: Omit<DecorateTrackContext, "track" | "isLocal"> = {},
): TrackDecoration {
  const ctx: DecorateTrackContext = {
    track,
    isLocal: isLocalMusicId(track.id),
    ...extra,
  };
  let decoration: TrackDecoration = {};
  for (const rule of declarativeRules) {
    if (matchesRule(ctx, rule.match)) {
      decoration = mergeDecorations(decoration, rule.decoration);
    }
  }
  for (const hook of scriptDecorators) {
    const next = hook(ctx);
    if (next) {
      decoration = mergeDecorations(decoration, next);
    }
  }
  return decoration;
}

function createExtensionAPI(
  settings: Record<string, unknown>,
): ExtensionAPI {
  return {
    registerTrackRule(rule) {
      declarativeRules.push(rule);
    },
    decorateTrack(ctx) {
      for (const rule of declarativeRules) {
        if (matchesRule(ctx, rule.match)) {
          return rule.decoration;
        }
      }
      return null;
    },
    settings: Object.freeze({ ...settings }),
  };
}

// Identifiers shadowed to undefined inside the runner. The audit scans
// for these statically, but a regex is not a boundary: shadowing is what
// actually removes them from the script's scope chain. Anything not
// listed stays reachable (math, strings, JSON, Date), which is all the
// extension API needs.
export const SHADOWED_GLOBALS = [
  "window",
  "document",
  "self",
  "globalThis",
  "frames",
  "top",
  "parent",
  "opener",
  "fetch",
  "XMLHttpRequest",
  "WebSocket",
  "EventSource",
  "navigator",
  "location",
  "history",
  "localStorage",
  "sessionStorage",
  "indexedDB",
  "caches",
  "open",
  "alert",
  "confirm",
  "prompt",
  // eval and arguments cannot be parameter names in strict mode. The
  // audit's static blocklist covers eval instead.
  "Function",
  "process",
  "require",
  "module",
  "exports",
  "importScripts",
  "postMessage",
  "customElements",
  "crypto",
  "SharedArrayBuffer",
  "Worker",
  "setTimeout",
  "setInterval",
  "queueMicrotask",
  "requestAnimationFrame",
  "close",
] as const;

// Runs an extension script with every dangerous global rebound to
// undefined. `new Function` still shares the global scope, so the shadow
// list is what makes the audit's static blocklist real at runtime.
// Exported for the sandbox test matrix.
export function runExtensionScript(
  source: string,
  api: ExtensionAPI,
): unknown {
  const runner = new Function(
    "api",
    "register",
    ...SHADOWED_GLOBALS,
    `"use strict";\n${source}\n;return typeof register === 'function' ? register(api) : undefined;`,
  ) as (
    api: ExtensionAPI,
    register: ScriptHook,
    ...shadowed: undefined[]
  ) => unknown;
  return runner(api, undefined as unknown as ScriptHook);
}

async function loadScriptExtension(
  manifest: ExtensionManifest,
  settings: Record<string, unknown>,
) {
  if (!manifest.script) return;
  const response = await fetch(
    resolveApiUrl(ApiPaths.extensionScript(manifest.id)),
    {
      credentials: isRemoteClient() ? "include" : "same-origin",
    },
  );
  if (!response.ok) return;
  const source = await response.text();
  if (
    /\b(import|require\s*\(|fetch\s*\(|eval\s*\(|Function\s*\(|window\b|document\b)\b/i.test(
      source,
    )
  ) {
    return;
  }
  const api = createExtensionAPI(settings);
  runExtensionScript(source, api);
}

export async function loadExtensions() {
  const response = await fetch(resolveApiUrl(ApiPaths.extensions), {
    credentials: isRemoteClient() ? "include" : "same-origin",
  });
  if (!response.ok) {
    return;
  }
  const payload = await parseJson(
    extensionsPayloadSchema,
    response,
    "extensions",
  );
  extensionFeatures.applyFromItems(payload.items ?? []);
  declarativeRules.length = 0;
  scriptDecorators.length = 0;
  const settingsById = new Map<string, Record<string, unknown>>();
  for (const item of payload.items ?? []) {
    if (item.settings) settingsById.set(item.id, item.settings);
  }
  const manifests = payload.manifests ?? [];
  for (const manifest of manifests) {
    for (const rule of manifest.trackRules ?? []) {
      declarativeRules.push(rule);
    }
    await loadScriptExtension(manifest, settingsById.get(manifest.id) ?? {});
  }
  extensionFeatures.applyAppThemeFromManifests(manifests);
  syncExtensionStyles(manifests);
}

export function resetExtensionsForTests() {
  declarativeRules.length = 0;
  scriptDecorators.length = 0;
  extensionFeatures.resetForTests([]);
  syncExtensionStyles([]);
}

export function registerTrackRuleForTests(rule: TrackRule) {
  declarativeRules.push(rule);
}
