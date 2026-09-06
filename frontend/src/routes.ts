// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { RouteDefinition } from "$lib/router/router.svelte";
import { APP_NAME } from "$lib/brand";
import MusicPage from "./pages/MusicPage.svelte";

export const routes: RouteDefinition[] = [
  {
    path: "/account/login",
    load: () => import("./pages/AccountLoginPage.svelte"),
    title: "Sign in",
    description: `Sign in to your ${APP_NAME} account.`,
  },
  {
    path: "/setup",
    load: () => import("./pages/SetupPage.svelte"),
    title: "Set up music source",
    description: "Connect a Subsonic server or local music folder.",
  },
  {
    path: "/share/:token",
    load: () => import("./pages/SharePage.svelte"),
    title: "Shared playlist",
    description: "Open a playlist shared with you.",
  },
  {
    path: "/listen/:token",
    load: () => import("./pages/ListenJoinPage.svelte"),
    title: "Join listen together",
    description: "Join a listen-together session.",
  },
  {
    path: "/login",
    component: MusicPage,
    title: "Music",
    description: "Home for your local and Subsonic libraries.",
  },
  {
    path: "/",
    component: MusicPage,
    title: "Music",
    description: "Home for your local and Subsonic libraries.",
  },
  {
    path: "/music",
    component: MusicPage,
    title: "Music",
    description: "Home for your local and Subsonic libraries.",
  },
  {
    path: "/music/search",
    load: () => import("./pages/SearchPage.svelte"),
    title: "Search",
    description: "Search your music library.",
  },
  {
    path: "/music/artists",
    load: () => import("./pages/ArtistsPage.svelte"),
    title: "Artists",
    description: "Browse artists in your library.",
  },
  {
    path: "/music/albums",
    load: () => import("./pages/AlbumsPage.svelte"),
    title: "Albums",
    description: "Browse albums in your library.",
  },
  {
    path: "/music/genres",
    load: () => import("./pages/GenresPage.svelte"),
    title: "Genres",
    description: "Browse genres in your library.",
  },
  {
    path: "/music/genre/:genre",
    load: () => import("./pages/GenrePage.svelte"),
    title: "Genre",
    description: "Browse tracks in this genre.",
  },
  {
    path: "/music/album/:albumId",
    load: () => import("./pages/AlbumPage.svelte"),
    title: "Album",
    description: "Album details and track list.",
  },
  {
    path: "/music/artist/:artistId",
    load: () => import("./pages/ArtistPage.svelte"),
    title: "Artist",
    description: "Artist discography and related artists.",
  },
  {
    path: "/music/mix/:mixId",
    load: () => import("./pages/MixPage.svelte"),
    title: "Mix",
    description: `A personalized ${APP_NAME} mix.`,
  },
  {
    path: "/music/playlists",
    load: () => import("./pages/PlaylistsPage.svelte"),
    title: "Playlists",
    description: "Your playlists and smart playlists.",
  },
  {
    path: "/music/shared",
    load: () => import("./pages/SharedInboxPage.svelte"),
    title: "Shared with you",
    description: "Playlists others have shared with you.",
  },
  {
    path: "/music/favorites",
    load: () => import("./pages/FavoritesPage.svelte"),
    title: "Favorites",
    description: "Starred tracks, albums, and artists.",
  },
  {
    path: "/music/history",
    load: () => import("./pages/HistoryPage.svelte"),
    title: "History",
    description: "Recently played tracks.",
  },
  {
    path: "/music/videos",
    load: () => import("./pages/VideosPage.svelte"),
    title: "Videos",
    description: "Music videos linked to your library.",
  },
  {
    path: "/play/:itemId",
    load: () => import("./pages/VideoPlayerPage.svelte"),
    title: "Video",
    description: "Watch a music video.",
  },
  {
    path: "/music/now-playing",
    load: () => import("./pages/NowPlayingPage.svelte"),
    title: "Now playing",
    description: "Full-screen now playing view.",
  },
  {
    path: "/music/lyrics",
    load: () => import("./pages/LyricsPage.svelte"),
    title: "Lyrics",
    description: "Synced and plain lyrics for the current track.",
  },
  {
    path: "/music/server-playlist/:playlistId",
    load: () => import("./pages/ServerPlaylistPage.svelte"),
    title: "Server playlist",
    description: "A playlist from your music server.",
  },
  {
    path: "/music/playlist/:playlistId",
    load: () => import("./pages/PlaylistPage.svelte"),
    title: "Playlist",
    description: "Playlist tracks and editing.",
  },
  {
    path: "/music/metadata",
    load: () => import("./pages/MetadataEditorPage.svelte"),
    title: "Metadata editor",
    description: "Edit track and album metadata.",
    index: false,
  },
  {
    path: "/settings/:tab",
    load: () => import("./pages/SettingsPage.svelte"),
    title: "Settings",
    description: "App, library, and playback settings.",
    index: false,
  },
  {
    path: "/settings",
    load: () => import("./pages/SettingsPage.svelte"),
    title: "Settings",
    description: "App, library, and playback settings.",
    index: false,
  },
  {
    path: "/instances",
    load: () => import("./pages/SettingsPage.svelte"),
    title: "Settings",
    description: "Manage music sources and servers.",
    index: false,
  },
];
