// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { RouteDefinition } from "$lib/router/router.svelte";
import MusicPage from "./pages/MusicPage.svelte";

export const routes: RouteDefinition[] = [
  {
    path: "/account/login",
    load: () => import("./pages/AccountLoginPage.svelte"),
    title: "Sign in",
  },
  {
    path: "/setup",
    load: () => import("./pages/SetupPage.svelte"),
    title: "Set up music source",
  },
  {
    path: "/share/:token",
    load: () => import("./pages/SharePage.svelte"),
    title: "Shared playlist",
  },
  {
    path: "/listen/:token",
    load: () => import("./pages/ListenJoinPage.svelte"),
    title: "Join listen together",
  },
  { path: "/login", component: MusicPage, title: "Music" },
  { path: "/", component: MusicPage, title: "Music" },
  { path: "/music", component: MusicPage, title: "Music" },
  {
    path: "/music/search",
    load: () => import("./pages/SearchPage.svelte"),
    title: "Search",
  },
  {
    path: "/music/artists",
    load: () => import("./pages/ArtistsPage.svelte"),
    title: "Artists",
  },
  {
    path: "/music/albums",
    load: () => import("./pages/AlbumsPage.svelte"),
    title: "Albums",
  },
  {
    path: "/music/genres",
    load: () => import("./pages/GenresPage.svelte"),
    title: "Genres",
  },
  {
    path: "/music/genre/:genre",
    load: () => import("./pages/GenrePage.svelte"),
    title: "Genre",
  },
  {
    path: "/music/album/:albumId",
    load: () => import("./pages/AlbumPage.svelte"),
    title: "Album",
  },
  {
    path: "/music/artist/:artistId",
    load: () => import("./pages/ArtistPage.svelte"),
    title: "Artist",
  },
  {
    path: "/music/mix/:mixId",
    load: () => import("./pages/MixPage.svelte"),
    title: "Mix",
  },
  {
    path: "/music/playlists",
    load: () => import("./pages/PlaylistsPage.svelte"),
    title: "Playlists",
  },
  {
    path: "/music/shared",
    load: () => import("./pages/SharedInboxPage.svelte"),
    title: "Shared with you",
  },
  {
    path: "/music/favorites",
    load: () => import("./pages/FavoritesPage.svelte"),
    title: "Favorites",
  },
  {
    path: "/music/history",
    load: () => import("./pages/HistoryPage.svelte"),
    title: "History",
  },
  {
    path: "/music/videos",
    load: () => import("./pages/VideosPage.svelte"),
    title: "Videos",
  },
  {
    path: "/play/:itemId",
    load: () => import("./pages/VideoPlayerPage.svelte"),
    title: "Video",
  },
  {
    path: "/music/now-playing",
    load: () => import("./pages/NowPlayingPage.svelte"),
    title: "Now playing",
  },
  {
    path: "/music/lyrics",
    load: () => import("./pages/LyricsPage.svelte"),
    title: "Lyrics",
  },
  {
    path: "/music/server-playlist/:playlistId",
    load: () => import("./pages/ServerPlaylistPage.svelte"),
    title: "Server playlist",
  },
  {
    path: "/music/playlist/:playlistId",
    load: () => import("./pages/PlaylistPage.svelte"),
    title: "Playlist",
  },
  {
    path: "/music/metadata",
    load: () => import("./pages/MetadataEditorPage.svelte"),
    title: "Metadata editor",
  },
  {
    path: "/settings/:tab",
    load: () => import("./pages/SettingsPage.svelte"),
    title: "Settings",
  },
  {
    path: "/settings",
    load: () => import("./pages/SettingsPage.svelte"),
    title: "Settings",
  },
  {
    path: "/instances",
    load: () => import("./pages/SettingsPage.svelte"),
    title: "Settings",
  },
];
