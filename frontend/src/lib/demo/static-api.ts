// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * In-browser API shim for VITE_STATIC_DEMO builds (GH Pages, frontend-only).
 * Intercepts fetch for /api/* and Subsonic rest paths. Uses fixtures from
 * /demo/catalog.json (exported from democatalog).
 */

type DemoArtist = {
  ID: string;
  Name: string;
  CoverArt: string;
  AlbumIDs: string[];
  Biography?: string;
  SimilarIDs?: string[];
};

type DemoAlbum = {
  ID: string;
  Name: string;
  ArtistID: string;
  Artist: string;
  Year: number;
  Genre: string;
  CoverArt: string;
  SongIDs: string[];
  CreatedAt: string;
};

type DemoSong = {
  ID: string;
  Title: string;
  AlbumID: string;
  Album: string;
  ArtistID: string;
  Artist: string;
  Track: number;
  Duration: number;
  Year: number;
  Genre: string;
  CoverArt: string;
  BitRate: number;
  Starred: boolean;
  PlayCount: number;
  ContentType: string;
  Suffix: string;
};

type DemoPlaylist = {
  ID: string;
  Name: string;
  Comment: string;
  SongIDs: string[];
  Created: string;
  Changed: string;
  Public: boolean;
  Owner: string;
};

type DemoGenre = {
  Name: string;
  SongCount: number;
  AlbumCount: number;
};

type DemoCatalog = {
  artists: DemoArtist[];
  albums: DemoAlbum[];
  songs: DemoSong[];
  playlists: DemoPlaylist[];
  genres: DemoGenre[];
};

const DEMO_INSTANCE = {
  id: "demo-instance",
  name: "Home Library",
  serverUrl: "fake://melovian-demo",
  username: "demo",
  serverName: "Home Library",
  createdAt: "2025-08-01T00:00:00Z",
  updatedAt: "2025-08-01T00:00:00Z",
};

let catalog: DemoCatalog | null = null;
let installed = false;
let originalFetch: typeof fetch | null = null;

function baseUrl(): string {
  const base =
    (typeof import.meta !== "undefined" && import.meta.env?.BASE_URL) || "/";
  return base.endsWith("/") ? base.slice(0, -1) : base;
}

function asset(path: string): string {
  const p = path.startsWith("/") ? path : `/${path}`;
  return `${baseUrl()}${p}`;
}

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      "Content-Type": "application/json",
      "X-Melovian-Server-Version": "0.1.0",
      "X-Melovian-API-Version": "1",
    },
  });
}

function textResponse(body: string, status: number, type: string): Response {
  return new Response(body, {
    status,
    headers: { "Content-Type": type },
  });
}

function forbid(): Response {
  return jsonResponse(
    { error: "demo_readonly", message: "Demo mode is read-only" },
    403,
  );
}

function subsonicOK(extra: Record<string, unknown> = {}): Response {
  return jsonResponse({
    "subsonic-response": {
      status: "ok",
      version: "1.16.1",
      type: "Home Library",
      serverVersion: "1.16.1",
      openSubsonic: true,
      ...extra,
    },
  });
}

function artistJSON(a: DemoArtist) {
  return {
    id: a.ID,
    name: a.Name,
    coverArt: a.CoverArt,
    albumCount: a.AlbumIDs?.length ?? 0,
  };
}

function albumMap(al: DemoAlbum) {
  return {
    id: al.ID,
    name: al.Name,
    artist: al.Artist,
    artistId: al.ArtistID,
    coverArt: al.CoverArt,
    songCount: al.SongIDs?.length ?? 0,
    duration: 0,
    created: al.CreatedAt,
    year: al.Year,
    genre: al.Genre,
  };
}

function songMap(s: DemoSong) {
  return {
    id: s.ID,
    parent: s.AlbumID,
    title: s.Title,
    album: s.Album,
    artist: s.Artist,
    artistId: s.ArtistID,
    albumId: s.AlbumID,
    track: s.Track,
    year: s.Year,
    genre: s.Genre,
    coverArt: s.CoverArt,
    size: 0,
    contentType: s.ContentType || "audio/wav",
    suffix: s.Suffix || "wav",
    duration: s.Duration,
    bitRate: s.BitRate || 128,
    path: `${s.Artist}/${s.Album}/${s.Title}.wav`,
    isDir: false,
    type: "music",
    starred: s.Starred ? "2025-01-01T00:00:00Z" : undefined,
    playCount: s.PlayCount,
  };
}

async function loadCatalog(): Promise<DemoCatalog> {
  if (catalog) return catalog;
  const res = await (originalFetch ?? fetch)(asset("/demo/catalog.json"));
  if (!res.ok) {
    throw new Error(`Failed to load demo catalog: ${res.status}`);
  }
  catalog = (await res.json()) as DemoCatalog;
  return catalog;
}

function findSong(c: DemoCatalog, id: string): DemoSong | undefined {
  return c.songs.find((s) => s.ID === id);
}

function findArtist(c: DemoCatalog, id: string): DemoArtist | undefined {
  return c.artists.find((a) => a.ID === id);
}

function findAlbum(c: DemoCatalog, id: string): DemoAlbum | undefined {
  return c.albums.find((a) => a.ID === id);
}

function findPlaylist(c: DemoCatalog, id: string): DemoPlaylist | undefined {
  return c.playlists.find((p) => p.ID === id);
}

function coverSVG(id: string): string {
  const hue = Array.from(id).reduce((a, ch) => a + ch.charCodeAt(0), 0) % 360;
  return `<svg xmlns="http://www.w3.org/2000/svg" width="300" height="300" viewBox="0 0 300 300">
  <defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
    <stop offset="0%" stop-color="hsl(${hue},45%,28%)"/>
    <stop offset="100%" stop-color="hsl(${(hue + 40) % 360},40%,18%)"/>
  </linearGradient></defs>
  <rect width="300" height="300" fill="url(#g)"/>
  <text x="24" y="260" fill="#fafafa" font-family="sans-serif" font-size="18">${id.slice(0, 18)}</text>
</svg>`;
}

async function handleSubsonic(
  endpoint: string,
  url: URL,
  c: DemoCatalog,
): Promise<Response> {
  const q = url.searchParams;
  switch (endpoint) {
    case "ping":
    case "getLicense":
      return subsonicOK();
    case "getCoverArt": {
      const id = q.get("id") || "cover";
      return textResponse(coverSVG(id), 200, "image/svg+xml");
    }
    case "stream":
    case "download": {
      const streamRes = await (originalFetch ?? fetch)(
        asset("/demo/silent.wav"),
      );
      return new Response(streamRes.body, {
        status: 200,
        headers: {
          "Content-Type": "audio/wav",
          "Accept-Ranges": "bytes",
        },
      });
    }
    case "getArtists": {
      const indexes: Record<string, ReturnType<typeof artistJSON>[]> = {};
      for (const a of c.artists) {
        const letter = (a.Name[0] || "#").toUpperCase();
        (indexes[letter] ??= []).push(artistJSON(a));
      }
      const index = Object.entries(indexes).map(([name, artist]) => ({
        name,
        artist,
      }));
      return subsonicOK({ artists: { index } });
    }
    case "getArtist": {
      const a = findArtist(c, q.get("id") || "");
      if (!a) return subsonicOK({ error: { code: 70, message: "not found" } });
      const albums = (a.AlbumIDs || [])
        .map((id) => findAlbum(c, id))
        .filter(Boolean)
        .map((al) => albumMap(al as DemoAlbum));
      return subsonicOK({
        artist: { ...artistJSON(a), album: albums },
      });
    }
    case "getArtistInfo":
    case "getArtistInfo2": {
      const a = findArtist(c, q.get("id") || "");
      return subsonicOK({
        artistInfo: {
          biography: a?.Biography || "",
          musicBrainzId: "",
          lastFmUrl: "",
          smallImageUrl: "",
          mediumImageUrl: "",
          largeImageUrl: "",
        },
        artistInfo2: {
          biography: a?.Biography || "",
          similarArtist: (a?.SimilarIDs || [])
            .map((id) => findArtist(c, id))
            .filter(Boolean)
            .map((x) => artistJSON(x as DemoArtist)),
        },
      });
    }
    case "getAlbum": {
      const al = findAlbum(c, q.get("id") || "");
      if (!al) return subsonicOK({ error: { code: 70, message: "not found" } });
      const songs = (al.SongIDs || [])
        .map((id) => findSong(c, id))
        .filter(Boolean)
        .map((s) => songMap(s as DemoSong));
      return subsonicOK({ album: { ...albumMap(al), song: songs } });
    }
    case "getAlbumList":
    case "getAlbumList2": {
      const size = Number(q.get("size") || 20);
      const offset = Number(q.get("offset") || 0);
      const genre = q.get("genre");
      let list = [...c.albums];
      if (genre) {
        list = list.filter(
          (a) => a.Genre.toLowerCase() === genre.toLowerCase(),
        );
      }
      const slice = list.slice(offset, offset + size).map(albumMap);
      const key = endpoint === "getAlbumList2" ? "albumList2" : "albumList";
      return subsonicOK({ [key]: { album: slice } });
    }
    case "getSong": {
      const s = findSong(c, q.get("id") || "");
      if (!s) return subsonicOK({ error: { code: 70, message: "not found" } });
      return subsonicOK({ song: songMap(s) });
    }
    case "getRandomSongs": {
      const size = Number(q.get("size") || 20);
      const songs = [...c.songs].slice(0, size).map(songMap);
      return subsonicOK({ randomSongs: { song: songs } });
    }
    case "getGenres":
      return subsonicOK({
        genres: {
          genre: c.genres.map((g) => ({
            value: g.Name,
            songCount: g.SongCount,
            albumCount: g.AlbumCount,
          })),
        },
      });
    case "getSongsByGenre": {
      const genre = q.get("genre") || "";
      const count = Number(q.get("count") || 20);
      const offset = Number(q.get("offset") || 0);
      const songs = c.songs
        .filter((s) => s.Genre.toLowerCase() === genre.toLowerCase())
        .slice(offset, offset + count)
        .map(songMap);
      return subsonicOK({ songsByGenre: { song: songs } });
    }
    case "search2":
    case "search3": {
      const query = (q.get("query") || "").toLowerCase();
      const artists = c.artists
        .filter((a) => a.Name.toLowerCase().includes(query))
        .slice(0, Number(q.get("artistCount") || 20))
        .map(artistJSON);
      const albums = c.albums
        .filter((a) => a.Name.toLowerCase().includes(query))
        .slice(0, Number(q.get("albumCount") || 20))
        .map(albumMap);
      const songs = c.songs
        .filter((s) => s.Title.toLowerCase().includes(query))
        .slice(0, Number(q.get("songCount") || 20))
        .map(songMap);
      const key = endpoint === "search3" ? "searchResult3" : "searchResult2";
      return subsonicOK({
        [key]: { artist: artists, album: albums, song: songs },
      });
    }
    case "getPlaylists":
      return subsonicOK({
        playlists: {
          playlist: c.playlists.map((p) => ({
            id: p.ID,
            name: p.Name,
            comment: p.Comment,
            owner: p.Owner,
            public: p.Public,
            songCount: p.SongIDs.length,
            created: p.Created,
            changed: p.Changed,
            coverArt: p.SongIDs[0]
              ? findSong(c, p.SongIDs[0])?.CoverArt
              : undefined,
          })),
        },
      });
    case "getPlaylist": {
      const p = findPlaylist(c, q.get("id") || "");
      if (!p) return subsonicOK({ error: { code: 70, message: "not found" } });
      const entry = p.SongIDs.map((id) => findSong(c, id))
        .filter(Boolean)
        .map((s) => songMap(s as DemoSong));
      return subsonicOK({
        playlist: {
          id: p.ID,
          name: p.Name,
          comment: p.Comment,
          owner: p.Owner,
          public: p.Public,
          songCount: p.SongIDs.length,
          created: p.Created,
          changed: p.Changed,
          entry,
        },
      });
    }
    case "getStarred":
    case "getStarred2": {
      const songs = c.songs.filter((s) => s.Starred).map(songMap);
      const key = endpoint === "getStarred2" ? "starred2" : "starred";
      return subsonicOK({ [key]: { song: songs, album: [], artist: [] } });
    }
    case "getSimilarSongs":
    case "getSimilarSongs2": {
      const size = Number(q.get("count") || 20);
      return subsonicOK({
        similarSongs: { song: c.songs.slice(0, size).map(songMap) },
        similarSongs2: { song: c.songs.slice(0, size).map(songMap) },
      });
    }
    case "getScanStatus":
      return subsonicOK({
        scanStatus: {
          scanning: false,
          count: c.songs.length,
          folderCount: 1,
          lastScan: "2025-08-01T00:00:00Z",
        },
      });
    case "getInternetRadioStations":
      return subsonicOK({
        internetRadioStations: { internetRadioStation: [] },
      });
    case "getLyrics":
    case "getLyricsBySongId":
      return subsonicOK({ lyrics: { value: "" } });
    case "star":
    case "unstar":
    case "scrobble":
    case "createPlaylist":
    case "updatePlaylist":
    case "deletePlaylist":
      return subsonicOK();
    default:
      return subsonicOK({
        error: { code: 0, message: `unsupported demo endpoint: ${endpoint}` },
      });
  }
}

function playlistApiPayload(c: DemoCatalog) {
  return c.playlists.map((p) => ({
    id: p.ID,
    name: p.Name,
    comment: p.Comment,
    owner: p.Owner,
    public: p.Public,
    songCount: p.SongIDs.length,
    createdAt: p.Created,
    updatedAt: p.Changed,
    trackIds: p.SongIDs,
  }));
}

async function handleApi(
  method: string,
  path: string,
  url: URL,
): Promise<Response> {
  const c = await loadCatalog();
  const write = method !== "GET" && method !== "HEAD" && method !== "OPTIONS";

  if (path.startsWith("/api/subsonic")) {
    if (write && !path.includes("stream") && !path.includes("download")) {
      // Subsonic mutations still return OK in server demo, match that.
    }
    let rest = path.replace(/^\/api\/subsonic\/?/, "");
    rest = rest.replace(/^rest\//, "");
    const endpoint = rest.replace(/\.view$/, "").split("?")[0] || "";
    return handleSubsonic(endpoint, url, c);
  }

  if (write) {
    return forbid();
  }

  switch (path) {
    case "/health":
      return jsonResponse({
        status: "ok",
        version: "0.1.0",
        apiVersion: 1,
        minClientVersion: "0.1.0",
        minServerVersion: "0.1.0",
        capabilities: [
          "browse",
          "playback",
          "playlists",
          "favorites",
          "history",
          "mixes",
          "personal_radio",
          "lyrics",
          "eq",
        ],
      });
    case "/api/config":
      return jsonResponse({
        dataDir: "",
        listenAddr: "",
        publicUrl: typeof window !== "undefined" ? window.location.origin : "",
        authEnabled: false,
        oidcEnabled: false,
        serverMode: true,
        demoMode: true,
        fakeCatalog: true,
        version: "0.1.0",
        apiVersion: 1,
        minClientVersion: "0.1.0",
        minServerVersion: "0.1.0",
        capabilities: [
          "browse",
          "playback",
          "playlists",
          "favorites",
          "history",
          "mixes",
          "personal_radio",
          "lyrics",
          "eq",
        ],
        connectionDefaults: {},
        subsonicServer: { enabled: false, restUrl: "/rest" },
        transcoding: { available: false },
        extensions: { dir: "" },
        dlna: { enabled: false, port: 0 },
        jukebox: { enabled: false },
        localLibrary: {
          enabled: false,
          defaultPath: "",
          allowCustomPath: false,
        },
      });
    case "/api/auth/status":
      return jsonResponse({
        enabled: false,
        authenticated: false,
        setupRequired: false,
        oidcEnabled: false,
        demoMode: true,
        fakeCatalog: true,
      });
    case "/api/instances":
      return jsonResponse({ instances: [DEMO_INSTANCE] });
    case "/api/instances/active":
      return jsonResponse({ instance: DEMO_INSTANCE });
    case "/api/sources/status":
      return jsonResponse({
        mode: "subsonic",
        activeInstanceId: DEMO_INSTANCE.id,
        activeLocalId: "",
        multiLocalLibrary: false,
        unifiedAvailable: false,
      });
    case "/api/local-libraries":
      return jsonResponse({ libraries: [] });
    case "/api/local-libraries/active":
      return jsonResponse({ library: null });
    case "/api/extensions":
      return jsonResponse({ extensions: [] });
    case "/api/music/status":
      return jsonResponse({
        enabled: true,
        connected: true,
        serverName: "Home Library",
        version: "1.16.1",
        source: "subsonic",
      });
    case "/api/music/library-stats":
      return jsonResponse({
        artists: c.artists.length,
        albums: c.albums.length,
        songs: c.songs.length,
        playlists: c.playlists.length,
      });
    case "/api/music/history":
      return jsonResponse({ entries: [] });
    case "/api/music/listen-events":
      return jsonResponse({ events: [] });
    case "/api/music/listen-events/years":
      return jsonResponse({ years: [] });
    case "/api/music/resume":
      return jsonResponse({ track: null });
    case "/api/music/stats":
      return jsonResponse({ plays: 0, minutes: 0 });
    case "/api/music/batch":
      return jsonResponse({
        favorites: c.songs.filter((s) => s.Starred).map((s) => ({ id: s.ID })),
        playlists: playlistApiPayload(c),
        history: [],
      });
    case "/api/music/playlists":
      return jsonResponse({ playlists: playlistApiPayload(c) });
    case "/api/music/favorites":
      return jsonResponse({
        tracks: c.songs.filter((s) => s.Starred).map((s) => songMap(s)),
      });
    case "/api/devices":
      return jsonResponse({ devices: [] });
    case "/api/settings/sentry":
      return jsonResponse({ clientReporting: false });
    case "/api/ws":
      return jsonResponse({ error: "websocket_unavailable" }, 400);
    default: {
      if (path.startsWith("/api/music/playlists/")) {
        const id = decodeURIComponent(path.split("/").pop() || "");
        const p = findPlaylist(c, id);
        if (!p) return jsonResponse({ error: "not_found" }, 404);
        return jsonResponse({
          playlist: {
            ...playlistApiPayload(c).find((x) => x.id === id),
            tracks: p.SongIDs.map((sid) => findSong(c, sid))
              .filter(Boolean)
              .map((s) => songMap(s as DemoSong)),
          },
        });
      }
      if (path.startsWith("/api/music/items/")) {
        return jsonResponse({ item: null });
      }
      if (path.startsWith("/api/instances/") && path.endsWith("/ping")) {
        return jsonResponse({
          ok: true,
          serverName: DEMO_INSTANCE.serverName,
          version: "1.16.1",
        });
      }
      // Soft-empty defaults keep browse working when optional endpoints fire.
      return jsonResponse({});
    }
  }
}

function normalizePath(pathname: string): string {
  const base = baseUrl();
  if (base && pathname.startsWith(base)) {
    const trimmed = pathname.slice(base.length);
    const next = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
    return next || "/";
  }
  return pathname || "/";
}

function pathFromInput(input: RequestInfo | URL): {
  path: string;
  url: URL;
  method: string;
  request?: Request;
} {
  if (typeof input === "string") {
    const url = new URL(input, window.location.origin);
    return { path: normalizePath(url.pathname), url, method: "GET" };
  }
  if (input instanceof URL) {
    return {
      path: normalizePath(input.pathname),
      url: input,
      method: "GET",
    };
  }
  const req = input as Request;
  const url = new URL(req.url, window.location.origin);
  return {
    path: normalizePath(url.pathname),
    url,
    method: req.method,
    request: req,
  };
}

function shouldIntercept(path: string): boolean {
  return (
    path === "/health" || path.startsWith("/api/") || path.startsWith("/rest/")
  );
}

/** Install the static demo fetch shim. Safe to call once. */
export async function installStaticDemoApi(): Promise<void> {
  if (installed || typeof window === "undefined") return;
  installed = true;
  originalFetch = window.fetch.bind(window);

  await loadCatalog();

  const shim: typeof fetch = async (input, init) => {
    const { path, url, method, request } = pathFromInput(input);
    const verb = (
      init?.method ||
      request?.method ||
      method ||
      "GET"
    ).toUpperCase();
    if (!shouldIntercept(path)) {
      return originalFetch!(input, init);
    }
    try {
      return await handleApi(verb, path, url);
    } catch (err) {
      console.error("static demo API error", path, err);
      return jsonResponse(
        { error: "demo_error", message: "Demo API request failed" },
        500,
      );
    }
  };

  window.fetch = shim;
  globalThis.fetch = shim;
}

export function isStaticDemoInstalled(): boolean {
  return installed;
}
