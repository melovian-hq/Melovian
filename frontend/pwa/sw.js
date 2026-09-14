// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Melovian service worker. The build injects __VERSION__ and __PRECACHE__
// with the hashed asset list so every release maps to a distinct cache.

const VERSION = "__VERSION__";
const SHELL_CACHE = `melovian-shell-${VERSION}`;
const PRECACHE = __PRECACHE__;
const NAV_FALLBACK_URL = "__BASE_URL__index.html";
// On a slow link a navigation fetch can hang forever. Serve the cached shell
// after this window instead and let the app retry its API calls itself.
const NAV_TIMEOUT_MS = 6000;

self.addEventListener("install", (event) => {
  // No skipWaiting here. The page decides when a new worker may activate so
  // an update never reloads mid-playback. On first install (no existing
  // controller) the worker activates on its own.
  event.waitUntil(
    caches.open(SHELL_CACHE).then((cache) => cache.addAll(PRECACHE)),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(
          keys
            .filter(
              (key) => key.startsWith("melovian-shell-") && key !== SHELL_CACHE,
            )
            .map((key) => caches.delete(key)),
        ),
      )
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("message", (event) => {
  if (event.data && event.data.type === "SKIP_WAITING") {
    void self.skipWaiting();
  }
});

function isStaticAsset(url) {
  return (
    url.origin === self.location.origin &&
    (PRECACHE.includes(url.pathname) || url.pathname.includes("/assets/"))
  );
}

self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.method !== "GET") return;

  const url = new URL(request.url);

  // Never touch API, stream, websocket, or cross-origin traffic. Returning
  // without respondWith keeps media playback completely outside the worker.
  if (url.origin !== self.location.origin) return;
  if (
    url.pathname.startsWith("/api/") ||
    url.pathname.startsWith("/rest/") ||
    url.pathname.startsWith("/ws") ||
    url.pathname.startsWith("/demo/")
  ) {
    return;
  }

  if (request.mode === "navigate") {
    event.respondWith(handleNavigation(request));
    return;
  }

  if (isStaticAsset(url)) {
    event.respondWith(cacheFirst(request));
    return;
  }
});

async function handleNavigation(request) {
  const cache = await caches.open(SHELL_CACHE);
  const network = fetch(request)
    .then((response) => {
      if (response.ok) {
        void cache.put(NAV_FALLBACK_URL, response.clone());
      }
      return response;
    })
    .catch(() => null);

  const response = await Promise.race([
    network,
    new Promise((resolve) => setTimeout(resolve, NAV_TIMEOUT_MS, null)),
  ]);
  if (response) return response;

  const cached = await cache.match(NAV_FALLBACK_URL);
  if (cached) return cached;
  const slow = await network;
  if (slow) return slow;
  return new Response("Melovian is offline and has no cached shell.", {
    status: 503,
    headers: { "Content-Type": "text/plain" },
  });
}

async function cacheFirst(request) {
  const cache = await caches.open(SHELL_CACHE);
  const cached = await cache.match(request);
  if (cached) return cached;
  try {
    const response = await fetch(request);
    if (response.ok) void cache.put(request, response.clone());
    return response;
  } catch {
    return Response.error();
  }
}
