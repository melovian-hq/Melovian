// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"fmt"
	"time"

	"melovian/internal/appconfig"
)

type DB struct {
	sql     *sql.DB
	dialect Dialect
}

// Open opens the configured database (SQLite by default, Postgres via DatabaseURL).
func Open(cfg appconfig.Config) (*DB, error) {
	dialect, dsn, err := resolveDialect(cfg)
	if err != nil {
		return nil, err
	}

	conn, err := dialect.Open(dsn)
	if err != nil {
		return nil, err
	}

	db := &DB{sql: conn, dialect: dialect}
	if err := dialect.Setup(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := db.runMigrations(cfg); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return db, nil
}

// OpenDB opens a SQLite database at path. Prefer Open for production entrypoints.
func OpenDB(path string, legacy appconfig.Config) (*DB, error) {
	cfg := legacy
	cfg.DatabasePath = path
	cfg.DatabaseURL = ""
	return Open(cfg)
}

func (db *DB) runMigrations(legacy appconfig.Config) error {
	if err := db.migrate(); err != nil {
		return err
	}
	if err := db.migratePlaylistSmartColumns(); err != nil {
		return err
	}
	if err := db.migrateListenProgressListenedMs(); err != nil {
		return err
	}
	if err := db.migrateLocalTrackMetadataColumns(); err != nil {
		return err
	}
	if err := db.migrateLocalTrackMediaKind(); err != nil {
		return err
	}
	if err := db.migrateTrackVideoLinks(); err != nil {
		return err
	}
	if err := db.migrateLocalTrackMediaKindIndex(); err != nil {
		return err
	}
	if err := NewListenStore(db).BackfillListenEvents(); err != nil {
		return err
	}
	if err := db.migrateInstancesUserID(); err != nil {
		return err
	}
	if err := db.migrateLegacyInstance(legacy); err != nil {
		return err
	}
	if err := NewAuthStore(db, "").migrate(); err != nil {
		return err
	}
	if err := NewShareStore(db).migrate(); err != nil {
		return err
	}
	if err := NewNotificationStore(db).migrate(); err != nil {
		return err
	}
	return nil
}

func (db *DB) Close() error {
	return db.sql.Close()
}

func (db *DB) Dialect() Dialect {
	return db.dialect
}

func (db *DB) migrate() error {
	autoInc := db.dialect.AutoIncPK("id")
	schema := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS listen_progress (
	user_id TEXT NOT NULL,
	track_id TEXT NOT NULL,
	position_ms INTEGER NOT NULL DEFAULT 0,
	played INTEGER NOT NULL DEFAULT 0,
	play_count INTEGER NOT NULL DEFAULT 0,
	listened_ms INTEGER NOT NULL DEFAULT 0,
	last_played_at INTEGER NOT NULL,
	track_title TEXT NOT NULL DEFAULT '',
	artist_name TEXT NOT NULL DEFAULT '',
	album_id TEXT NOT NULL DEFAULT '',
	album_title TEXT NOT NULL DEFAULT '',
	duration_ms INTEGER NOT NULL DEFAULT 0,
	cover_art_id TEXT NOT NULL DEFAULT '',
	updated_at INTEGER NOT NULL,
	PRIMARY KEY (user_id, track_id)
);

CREATE INDEX IF NOT EXISTS idx_listen_progress_user ON listen_progress(user_id, last_played_at DESC);

CREATE INDEX IF NOT EXISTS idx_listen_progress_resume ON listen_progress(user_id, last_played_at DESC)
	WHERE played = 0 AND position_ms > 5000;

CREATE TABLE IF NOT EXISTS listen_events (
	%s,
	user_id TEXT NOT NULL,
	track_id TEXT NOT NULL,
	played_at INTEGER NOT NULL,
	track_title TEXT NOT NULL DEFAULT '',
	artist_name TEXT NOT NULL DEFAULT '',
	album_id TEXT NOT NULL DEFAULT '',
	album_title TEXT NOT NULL DEFAULT '',
	duration_ms INTEGER NOT NULL DEFAULT 0,
	cover_art_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_listen_events_user_played ON listen_events(user_id, played_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS music_playlists (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	name TEXT NOT NULL,
	kind TEXT NOT NULL DEFAULT 'static',
	rules_json TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_music_playlists_user ON music_playlists(user_id);

CREATE TABLE IF NOT EXISTS music_playlist_tracks (
	playlist_id TEXT NOT NULL REFERENCES music_playlists(id) ON DELETE CASCADE,
	track_id TEXT NOT NULL,
	position INTEGER NOT NULL,
	track_title TEXT NOT NULL DEFAULT '',
	artist_name TEXT NOT NULL DEFAULT '',
	album_id TEXT NOT NULL DEFAULT '',
	album_title TEXT NOT NULL DEFAULT '',
	duration_ms INTEGER NOT NULL DEFAULT 0,
	cover_art_id TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (playlist_id, track_id)
);

CREATE INDEX IF NOT EXISTS idx_music_playlist_tracks_order ON music_playlist_tracks(playlist_id, position);

CREATE TABLE IF NOT EXISTS music_favorites (
	user_id TEXT NOT NULL,
	track_id TEXT NOT NULL,
	track_title TEXT NOT NULL DEFAULT '',
	artist_name TEXT NOT NULL DEFAULT '',
	album_id TEXT NOT NULL DEFAULT '',
	album_title TEXT NOT NULL DEFAULT '',
	duration_ms INTEGER NOT NULL DEFAULT 0,
	cover_art_id TEXT NOT NULL DEFAULT '',
	favorited_at INTEGER NOT NULL,
	PRIMARY KEY (user_id, track_id)
);

CREATE INDEX IF NOT EXISTS idx_music_favorites_user ON music_favorites(user_id, favorited_at DESC);

CREATE TABLE IF NOT EXISTS app_settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_preferences (
	user_id TEXT NOT NULL,
	key TEXT NOT NULL,
	value TEXT NOT NULL,
	updated_at INTEGER NOT NULL,
	PRIMARY KEY (user_id, key)
);

CREATE INDEX IF NOT EXISTS idx_user_preferences_user ON user_preferences(user_id);

CREATE TABLE IF NOT EXISTS subsonic_instances (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	server_url TEXT NOT NULL,
	username TEXT NOT NULL,
	password TEXT NOT NULL,
	server_name TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	last_used_at INTEGER
);

CREATE TABLE IF NOT EXISTS downloaded_tracks (
	instance_id TEXT NOT NULL,
	track_id TEXT NOT NULL,
	path TEXT NOT NULL,
	content_type TEXT NOT NULL DEFAULT '',
	size INTEGER NOT NULL DEFAULT 0,
	track_title TEXT NOT NULL DEFAULT '',
	artist_name TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	PRIMARY KEY (instance_id, track_id)
);

CREATE INDEX IF NOT EXISTS idx_downloaded_tracks_instance ON downloaded_tracks(instance_id, created_at DESC);

CREATE TABLE IF NOT EXISTS local_libraries (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL,
	path TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	last_scanned_at INTEGER,
	track_count INTEGER NOT NULL DEFAULT 0,
	missing_count INTEGER NOT NULL DEFAULT 0,
	duplicate_count INTEGER NOT NULL DEFAULT 0,
	scan_status TEXT NOT NULL DEFAULT 'idle',
	scan_error TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_local_libraries_user ON local_libraries(user_id, name ASC);

CREATE TABLE IF NOT EXISTS local_tracks (
	id TEXT PRIMARY KEY,
	library_id TEXT NOT NULL REFERENCES local_libraries(id) ON DELETE CASCADE,
	rel_path TEXT NOT NULL,
	abs_path TEXT NOT NULL,
	file_sig TEXT NOT NULL DEFAULT '',
	content_hash TEXT NOT NULL DEFAULT '',
	size INTEGER NOT NULL DEFAULT 0,
	mtime INTEGER NOT NULL DEFAULT 0,
	title TEXT NOT NULL DEFAULT '',
	artist TEXT NOT NULL DEFAULT '',
	album TEXT NOT NULL DEFAULT '',
	album_artist TEXT NOT NULL DEFAULT '',
	track_num INTEGER NOT NULL DEFAULT 0,
	disc_num INTEGER NOT NULL DEFAULT 0,
	duration_ms INTEGER NOT NULL DEFAULT 0,
	format TEXT NOT NULL DEFAULT '',
	media_kind TEXT NOT NULL DEFAULT 'audio',
	status TEXT NOT NULL DEFAULT 'present',
	duplicate_of TEXT NOT NULL DEFAULT '',
	updated_at INTEGER NOT NULL,
	UNIQUE(library_id, rel_path)
);

CREATE INDEX IF NOT EXISTS idx_local_tracks_library ON local_tracks(library_id, status);
CREATE INDEX IF NOT EXISTS idx_local_tracks_hash ON local_tracks(library_id, content_hash);
CREATE INDEX IF NOT EXISTS idx_local_tracks_library_media ON local_tracks(library_id, media_kind, status);
`, autoInc)
	return db.execScript(schema)
}

func (db *DB) migrateInstancesUserID() error {
	_, err := db.exec(`ALTER TABLE subsonic_instances ADD COLUMN user_id TEXT NOT NULL DEFAULT ''`)
	if err != nil && !db.dialect.IsDuplicateColumn(err) {
		return err
	}
	return nil
}

func (db *DB) migratePlaylistSmartColumns() error {
	alters := []string{
		`ALTER TABLE music_playlists ADD COLUMN kind TEXT NOT NULL DEFAULT 'static'`,
		`ALTER TABLE music_playlists ADD COLUMN rules_json TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range alters {
		_, err := db.exec(stmt)
		if err != nil && !db.dialect.IsDuplicateColumn(err) {
			return err
		}
	}
	return nil
}

func (db *DB) migrateListenProgressListenedMs() error {
	_, err := db.exec(`ALTER TABLE listen_progress ADD COLUMN listened_ms INTEGER NOT NULL DEFAULT 0`)
	if err != nil && !db.dialect.IsDuplicateColumn(err) {
		return err
	}
	return nil
}

func (db *DB) migrateLocalTrackMetadataColumns() error {
	alters := []string{
		`ALTER TABLE local_tracks ADD COLUMN genre TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE local_tracks ADD COLUMN year INTEGER NOT NULL DEFAULT 0`,
	}
	for _, stmt := range alters {
		_, err := db.exec(stmt)
		if err != nil && !db.dialect.IsDuplicateColumn(err) {
			return err
		}
	}
	return nil
}

func (db *DB) migrateLocalTrackMediaKind() error {
	_, err := db.exec(`ALTER TABLE local_tracks ADD COLUMN media_kind TEXT NOT NULL DEFAULT 'audio'`)
	if err != nil && !db.dialect.IsDuplicateColumn(err) {
		return err
	}
	return nil
}

func (db *DB) migrateTrackVideoLinks() error {
	return db.execScript(`
CREATE TABLE IF NOT EXISTS track_video_links (
	user_id TEXT NOT NULL DEFAULT '',
	track_id TEXT NOT NULL,
	source TEXT NOT NULL,
	video_id TEXT NOT NULL,
	title TEXT NOT NULL DEFAULT '',
	updated_at INTEGER NOT NULL,
	PRIMARY KEY (user_id, track_id)
);
`)
}

func (db *DB) migrateLocalTrackMediaKindIndex() error {
	return db.execScript(`
CREATE INDEX IF NOT EXISTS idx_local_tracks_library_media
ON local_tracks(library_id, media_kind, status);
`)
}

func nowUnix() int64 {
	return time.Now().Unix()
}

func (db *DB) getSetting(key string) (string, error) {
	var value string
	err := db.queryRow(`SELECT value FROM app_settings WHERE key = ?`, key).Scan(&value)
	return value, err
}

func (db *DB) setSetting(key, value string) error {
	_, err := db.exec(
		`INSERT INTO app_settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

func (db *DB) migrateLegacyInstance(legacy appconfig.Config) error {
	var count int
	if err := db.queryRow(`SELECT COUNT(*) FROM subsonic_instances`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	server := legacy.LegacyServer
	user := legacy.LegacyUser
	pass := legacy.LegacyPass

	if v, err := db.getSetting(appconfig.SettingNavidromeServer); err == nil && v != "" {
		server = v
	}
	if v, err := db.getSetting(appconfig.SettingNavidromeUser); err == nil && v != "" {
		user = v
	}
	if v, err := db.getSetting(appconfig.SettingNavidromePassword); err == nil && v != "" {
		pass = v
	}

	if server == "" || user == "" || pass == "" {
		return nil
	}

	instances := NewInstanceStore(db)
	inst, err := instances.Create(CreateInstanceInput{
		Name:      "Default",
		ServerURL: server,
		Username:  user,
		Password:  pass,
	})
	if err != nil {
		return err
	}
	return instances.SetActive(inst.ID)
}
