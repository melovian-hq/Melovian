// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * In-browser API shim for VITE_STATIC_DEMO builds (GH Pages, frontend-only).
 * Intercepts fetch for /api/* and Subsonic rest paths. Uses fixtures from
 * /demo/catalog.json (exported from democatalog).
 */

import { ApiPaths } from "$lib/core/http/api-paths";
import { parseJson } from "$lib/core/http/parse";
import { demoCatalogSchema } from "./schemas";

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
let installPromise: Promise<void> | null = null;
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

function demoVersion(): string {
  return (
    (typeof import.meta !== "undefined" &&
      (import.meta.env?.VITE_APP_VERSION as string | undefined)) ||
    "0.1.0"
  );
}

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      "Content-Type": "application/json",
      "X-Melovian-Server-Version": demoVersion(),
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
  catalog = await parseJson(demoCatalogSchema, res, "demo catalog");
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
      const pool = [...c.songs];
      for (let i = pool.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [pool[i], pool[j]] = [pool[j], pool[i]];
      }
      const songs = pool.slice(0, size).map(songMap);
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

function playlistTrackEntry(c: DemoCatalog, sid: string, position: number) {
  const s = findSong(c, sid);
  if (!s) return null;
  return {
    trackId: s.ID,
    trackTitle: s.Title,
    artistName: s.Artist,
    albumId: s.AlbumID,
    albumTitle: s.Album,
    durationMs: s.Duration * 1000,
    coverArtId: s.CoverArt,
    position,
  };
}

function playlistApiPayload(c: DemoCatalog) {
  return c.playlists.map((p) => ({
    id: p.ID,
    name: p.Name,
    kind: "static",
    rulesJson: "",
    createdAt: p.Created,
    updatedAt: p.Changed,
    trackCount: p.SongIDs.length,
    durationMs: p.SongIDs.reduce(
      (ms, sid) => ms + (findSong(c, sid)?.Duration ?? 0) * 1000,
      0,
    ),
    coverArtIds: [
      ...new Set(
        p.SongIDs.slice(0, 8)
          .map((sid) => findSong(c, sid)?.CoverArt)
          .filter((id): id is string => Boolean(id)),
      ),
    ].slice(0, 4),
  }));
}

function listenEntry(
  s: DemoSong,
  overrides: {
    positionMs?: number;
    played?: boolean;
    playCount?: number;
    listenedMs?: number;
    lastPlayedAt: string;
  },
) {
  return {
    trackId: s.ID,
    trackTitle: s.Title,
    artistName: s.Artist,
    albumId: s.AlbumID,
    albumTitle: s.Album,
    positionMs: overrides.positionMs ?? 0,
    durationMs: s.Duration * 1000,
    played: overrides.played ?? true,
    playCount: overrides.playCount ?? Math.max(1, s.PlayCount),
    listenedMs: overrides.listenedMs ?? s.Duration * 1000,
    lastPlayedAt: overrides.lastPlayedAt,
    coverArtId: s.CoverArt,
  };
}

// Recent-history fixtures so the static demo shows the same populated home
// shelves the seeded server demo produces.
function seededListenHistory(c: DemoCatalog) {
  const sorted = [...c.songs].sort((a, b) => b.PlayCount - a.PlayCount);
  const base = Date.UTC(2025, 7, 1, 12, 0, 0);
  return sorted.slice(0, 20).map((s, i) =>
    listenEntry(s, {
      lastPlayedAt: new Date(base - i * 30 * 60 * 1000).toISOString(),
    }),
  );
}

function seededResumeItems(c: DemoCatalog) {
  const top = [...c.songs].sort((a, b) => b.PlayCount - a.PlayCount)[0];
  if (!top) return [];
  return [
    listenEntry(top, {
      positionMs: Math.floor(top.Duration * 1000 * 0.45),
      played: false,
      listenedMs: Math.floor(top.Duration * 1000 * 0.45),
      lastPlayedAt: new Date(Date.UTC(2025, 7, 1, 12, 0, 0)).toISOString(),
    }),
  ];
}

function seededListenStats(c: DemoCatalog) {
  const byArtist = new Map<string, number>();
  const byAlbum = new Map<string, number>();
  let totalPlays = 0;
  let totalListeningMs = 0;
  let uniqueTracks = 0;
  for (const s of c.songs) {
    if (s.PlayCount <= 0) continue;
    uniqueTracks += 1;
    totalPlays += s.PlayCount;
    totalListeningMs += s.PlayCount * s.Duration * 1000;
    byArtist.set(s.Artist, (byArtist.get(s.Artist) ?? 0) + s.PlayCount);
    byAlbum.set(s.Album, (byAlbum.get(s.Album) ?? 0) + s.PlayCount);
  }
  const toEntries = (m: Map<string, number>) =>
    [...m.entries()]
      .sort((a, b) => b[1] - a[1])
      .slice(0, 8)
      .map(([label, count]) => ({ key: label, label, count }));
  const topTracks = [...c.songs]
    .filter((s) => s.PlayCount > 0)
    .sort((a, b) => b.PlayCount - a.PlayCount)
    .slice(0, 8)
    .map((s) => ({ key: s.ID, label: s.Title, count: s.PlayCount }));
  return {
    totalPlays,
    uniqueTracks,
    totalListeningMs,
    topArtists: toEntries(byArtist),
    topTracks,
    topAlbums: toEntries(byAlbum),
  };
}

async function handleApi(
  method: string,
  path: string,
  url: URL,
): Promise<Response> {
  const c = await loadCatalog();
  const write = method !== "GET" && method !== "HEAD" && method !== "OPTIONS";

  if (path.startsWith(ApiPaths.subsonicPrefix)) {
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
        version: demoVersion(),
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
    case ApiPaths.config:
      return jsonResponse({
        dataDir: "",
        listenAddr: "",
        publicUrl: typeof window !== "undefined" ? window.location.origin : "",
        authEnabled: false,
        oidcEnabled: false,
        serverMode: true,
        demoMode: true,
        fakeCatalog: true,
        version: demoVersion(),
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
    case ApiPaths.authStatus:
      return jsonResponse({
        enabled: false,
        authenticated: false,
        setupRequired: false,
        oidcEnabled: false,
        demoMode: true,
        fakeCatalog: true,
      });
    case ApiPaths.instances:
      return jsonResponse({ instances: [DEMO_INSTANCE] });
    case ApiPaths.instancesActive:
      // The real endpoint returns the bare instance object, or {} when unset.
      return jsonResponse(DEMO_INSTANCE);
    case ApiPaths.sourcesStatus:
      return jsonResponse({
        mode: "subsonic",
        activeInstanceId: DEMO_INSTANCE.id,
        activeLocalId: "",
        multiLocalLibrary: false,
        unifiedAvailable: false,
      });
    case ApiPaths.localLibraries:
      return jsonResponse({ libraries: [] });
    case ApiPaths.localLibrariesActive:
      // The real endpoint returns {} when no library is active.
      return jsonResponse({});
    case ApiPaths.extensions: {
      // Mirror a fresh server: auto-installed bundled extensions enabled,
      // optional bundled listed but not installed. The metadata feature gate
      // is derived from this list, so an empty list would silently disable
      // artwork lookups in the demo.
      const bundledItem = (
        id: string,
        name: string,
        version: string,
        description: string,
        enabled: boolean,
      ) => ({
        id,
        name,
        version,
        description,
        author: "Melovian",
        enabled,
        installed: enabled,
        bundled: true,
        hasScript: false,
        scriptSafe: false,
        hasWasm: false,
      });
      const enabled = [
        bundledItem(
          "lyrics",
          "Lyrics",
          "0.1.1",
          "Lyrics panel, providers, settings, and now-playing lyrics tab.",
          true,
        ),
        bundledItem(
          "metadata",
          "Metadata",
          "0.1.1",
          "Local metadata editor, catalog lookups (iTunes, MusicBrainz, Deezer, TheAudioDB), and library metadata tools.",
          true,
        ),
      ];
      const optional = [
        bundledItem("rocksky", "Rocksky", "0.1.0", "", false),
        bundledItem("lastfm", "Last.fm", "0.1.0", "", false),
        bundledItem("listenbrainz", "ListenBrainz", "0.1.0", "", false),
        bundledItem("lyrics-whisper", "Lyrics Whisper", "0.1.0", "", false),
      ];
      return jsonResponse({
        items: [...enabled, ...optional],
        manifests: enabled.map((e) => ({
          id: e.id,
          name: e.name,
          version: e.version,
          description: e.description,
          author: e.author,
        })),
        dir: "",
      });
    }
    case ApiPaths.musicStatus:
      return jsonResponse({
        enabled: true,
        connected: true,
        serverName: "Home Library",
        version: "1.16.1",
        source: "subsonic",
      });
    case ApiPaths.musicLibraryStats:
      return jsonResponse({
        songCount: c.songs.length,
        albumCount: c.albums.length,
        artistCount: c.artists.length,
        folderCount: 1,
        scanning: false,
        lastScan: "2025-08-01T00:00:00Z",
      });
    case ApiPaths.musicHistory:
      return jsonResponse({ items: seededListenHistory(c) });
    case ApiPaths.musicListenEvents:
      return jsonResponse({
        items: seededListenHistory(c).map((entry, i) => ({
          id: i + 1,
          trackId: entry.trackId,
          trackTitle: entry.trackTitle,
          artistName: entry.artistName,
          albumId: entry.albumId,
          albumTitle: entry.albumTitle,
          durationMs: entry.durationMs,
          coverArtId: entry.coverArtId,
          playedAt: entry.lastPlayedAt,
        })),
        hasMore: false,
      });
    case ApiPaths.musicListenEventYears:
      return jsonResponse({ years: [2025] });
    case ApiPaths.musicResume:
      return jsonResponse({ items: seededResumeItems(c) });
    case ApiPaths.musicStats:
      return jsonResponse(seededListenStats(c));
    case ApiPaths.musicBatch:
      // The real endpoint is a trackId -> listen entry map.
      return jsonResponse({});
    case ApiPaths.musicPlaylists:
      return jsonResponse({ playlists: playlistApiPayload(c) });
    case ApiPaths.musicSmartPlaylistsSupport:
      return jsonResponse({
        supported: false,
        reason: "Smart playlists are not available in the demo",
      });
    case ApiPaths.musicFavorites:
      return jsonResponse({
        items: c.songs
          .filter((s) => s.Starred)
          .map((s) => ({
            trackId: s.ID,
            trackTitle: s.Title,
            artistName: s.Artist,
            albumId: s.AlbumID,
            albumTitle: s.Album,
            durationMs: s.Duration * 1000,
            coverArtId: s.CoverArt,
            favoritedAt: "2025-01-01T00:00:00Z",
          })),
      });
    case ApiPaths.devices:
      return jsonResponse({ devices: [] });
    case ApiPaths.settingsSentry:
      return jsonResponse({ clientReporting: false });
    case ApiPaths.ws:
      return jsonResponse({ error: "websocket_unavailable" }, 400);
    default: {
      if (path.startsWith(`${ApiPaths.musicPlaylists}/`)) {
        const id = decodeURIComponent(path.split("/").pop() || "");
        const p = findPlaylist(c, id);
        if (!p) return jsonResponse({ error: "not_found" }, 404);
        // The real endpoint returns the bare playlist object with tracks.
        return jsonResponse({
          ...playlistApiPayload(c).find((x) => x.id === id),
          tracks: p.SongIDs.map((sid, i) =>
            playlistTrackEntry(c, sid, i),
          ).filter(Boolean),
        });
      }
      if (path.startsWith(ApiPaths.musicItemsPrefix)) {
        return jsonResponse({});
      }
      if (path.startsWith(`${ApiPaths.instances}/`) && path.endsWith("/ping")) {
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
    path === "/health" ||
    path.startsWith(ApiPaths.apiPrefix) ||
    path.startsWith("/rest/")
  );
}

/**
 * Install the static demo fetch shim. Safe to call more than once: the
 * in-flight promise dedupes concurrent callers (so a second caller never
 * captures the shim itself as the "original" fetch), and installed only
 * latches after the shim is live so a failed catalog load stays retryable.
 */
export async function installStaticDemoApi(): Promise<void> {
  if (installed || typeof window === "undefined") return;
  if (!installPromise) {
    installPromise = doInstall().finally(() => {
      if (!installed) installPromise = null;
    });
  }
  return installPromise;
}

async function doInstall(): Promise<void> {
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
  installed = true;
}

export function isStaticDemoInstalled(): boolean {
  return installed;
}
