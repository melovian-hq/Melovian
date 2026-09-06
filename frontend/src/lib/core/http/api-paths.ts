// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export const ApiPaths = {
  authStatus: "/api/auth/status",
  authSetup: "/api/auth/setup",
  authLogin: "/api/auth/login",
  authLogout: "/api/auth/logout",

  instances: "/api/instances",
  instancesActive: "/api/instances/active",
  instancesTest: "/api/instances/test",
  instanceById: (id: string) => `/api/instances/${id}`,
  instancePing: (id: string) => `/api/instances/${id}/ping`,
  instanceActivate: (id: string) => `/api/instances/${id}/activate`,

  localLibraries: "/api/local-libraries",
  localLibrariesActive: "/api/local-libraries/active",
  localLibraryById: (id: string) => `/api/local-libraries/${id}`,
  localLibraryActivate: (id: string) => `/api/local-libraries/${id}/activate`,
  localLibraryScan: (id: string) => `/api/local-libraries/${id}/scan`,
  config: "/api/config",

  sourcesStatus: "/api/sources/status",
  sourcesMultiLocalLibrary: "/api/sources/multi-local-library",
  sourcesViewMode: "/api/sources/view-mode",

  notifications: "/api/notifications",
  notificationsUnreadCount: "/api/notifications/unread-count",
  notificationsReadAll: "/api/notifications/read-all",
  notificationById: (id: string) => `/api/notifications/${id}`,
  notificationRead: (id: string) => `/api/notifications/${id}/read`,

  musicStatus: "/api/music/status",
  musicLibraryStats: "/api/music/library-stats",
  musicLibraryRefresh: "/api/music/library/refresh",
  musicHistory: "/api/music/history",
  musicListenEvents: "/api/music/listen-events",
  musicListenEventYears: "/api/music/listen-events/years",
  musicResume: "/api/music/resume",
  musicStats: "/api/music/stats",
  musicBatch: "/api/music/batch",
  musicItem: (trackId: string) => `/api/music/items/${trackId}`,
  musicItemPlayed: (trackId: string) => `/api/music/items/${trackId}/played`,
  musicPlaylists: "/api/music/playlists",
  musicPlaylist: (id: string) => `/api/music/playlists/${id}`,
  musicPlaylistTracks: (playlistId: string) =>
    `/api/music/playlists/${playlistId}/tracks`,
  musicPlaylistTrack: (playlistId: string, trackId: string) =>
    `/api/music/playlists/${playlistId}/tracks/${trackId}`,
  musicFavorites: "/api/music/favorites",
  musicFavorite: (trackId: string) => `/api/music/favorites/${trackId}`,
  musicSettingsCache: "/api/music/settings/cache",
  musicSettingsEq: "/api/music/settings/eq",
  musicSettingsConnection: "/api/music/settings/connection",
  musicSettingsRocksky: "/api/music/settings/rocksky",
  musicRockskyTest: "/api/music/settings/rocksky/test",
  musicRockskyNowPlaying: "/api/music/rocksky/now-playing",
  musicRockskyScrobble: "/api/music/rocksky/scrobble",
  musicListenBrainzSettings: "/api/music/settings/listenbrainz",
  musicListenBrainzTest: "/api/music/settings/listenbrainz/test",
  musicListenBrainzNowPlaying: "/api/music/listenbrainz/now-playing",
  musicListenBrainzScrobble: "/api/music/listenbrainz/scrobble",
  musicLastFMSettings: "/api/music/settings/lastfm",
  musicLastFMTest: "/api/music/settings/lastfm/test",
  musicLastFMNowPlaying: "/api/music/lastfm/now-playing",
  musicLastFMScrobble: "/api/music/lastfm/scrobble",
  musicSettingsLyrics: "/api/music/settings/lyrics",
  musicShares: "/api/music/shares",
  musicSharesInbox: "/api/music/shares/inbox",
  musicShare: (id: string) => `/api/music/shares/${encodeURIComponent(id)}`,
  musicLyrics: (trackId: string) =>
    `/api/music/lyrics/${encodeURIComponent(trackId)}`,
  musicLyricsFetch: (trackId: string) =>
    `/api/music/lyrics/${encodeURIComponent(trackId)}/fetch`,
  musicLyricsWhisper: (trackId: string) =>
    `/api/music/lyrics/${encodeURIComponent(trackId)}/whisper`,
  musicLyricsCache: "/api/music/lyrics/cache",
  musicSmartPlaylistsSupport: "/api/music/smart-playlists/support",
  musicSmartPlaylists: "/api/music/smart-playlists",

  localMusicVideos: "/api/local-music/videos",
  localMusicVideo: (id: string) =>
    `/api/local-music/videos/${encodeURIComponent(id)}`,
  videoSettings: "/api/video/settings",
  videoSearch: "/api/video/search",
  videoResolve: "/api/video/resolve",
  videoLink: (trackId: string) =>
    `/api/video/links/${encodeURIComponent(trackId)}`,

  downloads: "/api/downloads",
  downloadsDir: "/api/downloads/dir",
  downloadsReveal: "/api/downloads/reveal",
  downloadsExportZip: "/api/downloads/export.zip",
  download: (trackId: string) =>
    `/api/downloads/${encodeURIComponent(trackId)}`,
  downloadExport: (trackId: string) =>
    `/api/downloads/${encodeURIComponent(trackId)}/export`,

  mediaTrackDownload: (trackId: string) =>
    `/api/media/tracks/${encodeURIComponent(trackId)}/download`,
  mediaAlbumDownloadZip: (albumId: string) =>
    `/api/media/albums/${encodeURIComponent(albumId)}/download.zip`,
  mediaPlaylistDownloadZip: (playlistId: string) =>
    `/api/media/playlists/${encodeURIComponent(playlistId)}/download.zip`,
  mediaServerPlaylistDownloadZip: (playlistId: string) =>
    `/api/media/server-playlists/${encodeURIComponent(playlistId)}/download.zip`,

  partyPrefix: "/api/party/",
  partyInviteLink: "/api/party/invite-link",
  partyInvite: "/api/party/invite",
  partyJoin: "/api/party/join",
  partyStatus: (sessionId: string) =>
    `/api/party/${encodeURIComponent(sessionId)}/status`,
  partyStream: (sessionId: string, trackId: string) =>
    `/api/party/${encodeURIComponent(sessionId)}/stream/${encodeURIComponent(trackId)}`,
  partyCover: (sessionId: string, trackId: string) =>
    `/api/party/${encodeURIComponent(sessionId)}/cover/${encodeURIComponent(trackId)}`,
} as const;
