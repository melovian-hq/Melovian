package com.wails.app;

import android.os.Bundle;
import android.support.v4.media.MediaBrowserCompat;
import android.support.v4.media.MediaDescriptionCompat;
import android.support.v4.media.MediaMetadataCompat;
import android.support.v4.media.session.MediaSessionCompat;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.media.MediaBrowserServiceCompat;

import java.util.ArrayList;
import java.util.List;

/**
 * MediaBrowserService for Android Auto / system media browsers.
 * Exposes Now Playing and the current queue pushed from the WebView.
 */
public class MelovianMediaBrowserService extends MediaBrowserServiceCompat {
    public static final String ROOT_ID = "root";
    public static final String NOW_PLAYING_ID = "nowplaying";
    public static final String QUEUE_ID = "queue";

    private static MelovianMediaBrowserService instance;

    public static void notifyQueueChanged() {
        MelovianMediaBrowserService svc = instance;
        if (svc == null) return;
        svc.notifyChildrenChanged(ROOT_ID);
        svc.notifyChildrenChanged(NOW_PLAYING_ID);
        svc.notifyChildrenChanged(QUEUE_ID);
    }

    @Override
    public void onCreate() {
        super.onCreate();
        instance = this;
        MediaSessionCompat session = MelovianMediaSession.getOrCreate(this);
        setSessionToken(session.getSessionToken());
    }

    @Override
    public void onDestroy() {
        if (instance == this) {
            instance = null;
        }
        super.onDestroy();
    }

    @Nullable
    @Override
    public BrowserRoot onGetRoot(
            @NonNull String clientPackageName,
            int clientUid,
            @Nullable Bundle rootHints) {
        // Android Auto and system media browsers may connect while the UI is backgrounded.
        Bundle extras = new Bundle();
        extras.putBoolean(BrowserRoot.EXTRA_OFFLINE, false);
        return new BrowserRoot(ROOT_ID, extras);
    }

    @Override
    public void onLoadChildren(
            @NonNull String parentId,
            @NonNull Result<List<MediaBrowserCompat.MediaItem>> result) {
        List<MediaBrowserCompat.MediaItem> items = new ArrayList<>();
        if (ROOT_ID.equals(parentId)) {
            items.add(browseable("Now Playing", NOW_PLAYING_ID, "Current track"));
            items.add(browseable("Queue", QUEUE_ID, "Up next"));
        } else if (NOW_PLAYING_ID.equals(parentId)) {
            String title = MelovianMediaSession.getLastTrack();
            if (title == null || title.isEmpty()) {
                title = "Nothing playing";
            }
            items.add(playable(
                    NOW_PLAYING_ID,
                    title,
                    MelovianMediaSession.getLastArtist(),
                    MelovianMediaSession.getLastAlbum()));
        } else if (QUEUE_ID.equals(parentId)) {
            List<MelovianMediaSession.QueueItem> queue = MelovianMediaSession.getQueue();
            for (int i = 0; i < queue.size(); i++) {
                MelovianMediaSession.QueueItem q = queue.get(i);
                String title = q.title.isEmpty() ? ("Track " + (i + 1)) : q.title;
                items.add(playable("queue:" + i, title, q.artist, q.album));
            }
            if (items.isEmpty()) {
                items.add(playable(
                        NOW_PLAYING_ID,
                        MelovianMediaSession.getLastTrack().isEmpty()
                                ? "Queue empty"
                                : MelovianMediaSession.getLastTrack(),
                        MelovianMediaSession.getLastArtist(),
                        MelovianMediaSession.getLastAlbum()));
            }
        }
        result.sendResult(items);
    }

    private static MediaBrowserCompat.MediaItem browseable(String title, String id, String subtitle) {
        MediaDescriptionCompat desc = new MediaDescriptionCompat.Builder()
                .setMediaId(id)
                .setTitle(title)
                .setSubtitle(subtitle)
                .build();
        return new MediaBrowserCompat.MediaItem(desc, MediaBrowserCompat.MediaItem.FLAG_BROWSABLE);
    }

    private static MediaBrowserCompat.MediaItem playable(
            String id, String title, String artist, String album) {
        MediaDescriptionCompat.Builder b = new MediaDescriptionCompat.Builder()
                .setMediaId(id)
                .setTitle(title == null || title.isEmpty() ? "Melovian" : title)
                .setSubtitle(artist == null ? "" : artist);
        if (album != null && !album.isEmpty()) {
            b.setDescription(album);
        }
        Bundle extras = new Bundle();
        extras.putString(MediaMetadataCompat.METADATA_KEY_ARTIST, artist == null ? "" : artist);
        b.setExtras(extras);
        return new MediaBrowserCompat.MediaItem(
                b.build(), MediaBrowserCompat.MediaItem.FLAG_PLAYABLE);
    }
}
