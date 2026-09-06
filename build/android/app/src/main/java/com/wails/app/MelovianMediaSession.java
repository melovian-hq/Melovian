package com.wails.app;

import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.graphics.Bitmap;
import android.media.AudioAttributes;
import android.media.AudioFocusRequest;
import android.media.AudioManager;
import android.os.Build;
import android.support.v4.media.MediaDescriptionCompat;
import android.support.v4.media.MediaMetadataCompat;
import android.support.v4.media.session.MediaSessionCompat;
import android.support.v4.media.session.PlaybackStateCompat;
import android.util.Log;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

/**
 * Shared MediaSessionCompat + audio focus used by the foreground notification
 * service and Android Auto MediaBrowserService.
 */
public final class MelovianMediaSession {
    private static final String TAG = "MelovianMedia";
    private static final String SESSION_TAG = "melovian-media-session";
    public static final long MEDIA_ACTIONS = PlaybackStateCompat.ACTION_PLAY
            | PlaybackStateCompat.ACTION_PAUSE
            | PlaybackStateCompat.ACTION_PLAY_PAUSE
            | PlaybackStateCompat.ACTION_SKIP_TO_NEXT
            | PlaybackStateCompat.ACTION_SKIP_TO_PREVIOUS
            | PlaybackStateCompat.ACTION_SEEK_TO
            | PlaybackStateCompat.ACTION_STOP
            | PlaybackStateCompat.ACTION_PLAY_FROM_MEDIA_ID
            | PlaybackStateCompat.ACTION_SKIP_TO_QUEUE_ITEM;

    public static final class QueueItem {
        public final String id;
        public final String title;
        public final String artist;
        public final String album;
        public final String artwork;

        public QueueItem(String id, String title, String artist, String album, String artwork) {
            this.id = id == null ? "" : id;
            this.title = title == null ? "" : title;
            this.artist = artist == null ? "" : artist;
            this.album = album == null ? "" : album;
            this.artwork = artwork == null ? "" : artwork;
        }
    }

    private static MediaSessionCompat mediaSession;
    private static AudioManager audioManager;
    private static AudioFocusRequest focusRequest;
    private static boolean hasFocus;
    private static String lastTrack = "";
    private static String lastArtist = "";
    private static String lastAlbum = "";
    private static String lastArtwork = "";
    private static boolean lastPlaying = false;
    private static long lastDurationMs = 0;
    private static long lastPositionMs = 0;
    private static Bitmap lastArtBitmap = null;
    private static String lastArtLoadedUrl = "";
    private static final List<QueueItem> queue = new ArrayList<>();
    private static int queueIndex = -1;

    private MelovianMediaSession() {}

    public static synchronized MediaSessionCompat getOrCreate(Context context) {
        if (mediaSession != null) {
            return mediaSession;
        }
        Context app = context.getApplicationContext();
        audioManager = (AudioManager) app.getSystemService(Context.AUDIO_SERVICE);
        mediaSession = new MediaSessionCompat(app, SESSION_TAG);
        mediaSession.setFlags(MediaSessionCompat.FLAG_HANDLES_MEDIA_BUTTONS
                | MediaSessionCompat.FLAG_HANDLES_TRANSPORT_CONTROLS);
        mediaSession.setCallback(new MediaSessionCompat.Callback() {
            @Override
            public void onPlay() {
                requestFocus(app);
                WailsBridge.dispatchMediaAction("play");
            }

            @Override
            public void onPause() {
                WailsBridge.dispatchMediaAction("pause");
            }

            @Override
            public void onSkipToNext() {
                WailsBridge.dispatchMediaAction("next");
            }

            @Override
            public void onSkipToPrevious() {
                WailsBridge.dispatchMediaAction("previous");
            }

            @Override
            public void onStop() {
                WailsBridge.dispatchMediaAction("pause");
                abandonFocus();
            }

            @Override
            public void onSeekTo(long pos) {
                WailsBridge.dispatchMediaAction("seek:" + (pos / 1000.0));
            }

            @Override
            public void onPlayFromMediaId(String mediaId, android.os.Bundle extras) {
                if (mediaId == null || mediaId.isEmpty()) return;
                if (mediaId.startsWith("queue:")) {
                    try {
                        int index = Integer.parseInt(mediaId.substring(6));
                        WailsBridge.dispatchMediaAction("play-queue-index:" + index);
                    } catch (NumberFormatException ignored) {
                    }
                    return;
                }
                if ("nowplaying".equals(mediaId)) {
                    WailsBridge.dispatchMediaAction("play");
                }
            }

            @Override
            public void onSkipToQueueItem(long id) {
                if (id < 0 || id > Integer.MAX_VALUE) return;
                WailsBridge.dispatchMediaAction("play-queue-index:" + (int) id);
            }
        });

        Intent launch = new Intent(app, MainActivity.class);
        launch.setAction(Intent.ACTION_MAIN);
        launch.addCategory(Intent.CATEGORY_LAUNCHER);
        launch.addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP | Intent.FLAG_ACTIVITY_CLEAR_TOP);
        mediaSession.setSessionActivity(
                PendingIntent.getActivity(app, 0, launch, pendingFlags()));

        Intent mediaButton = new Intent(Intent.ACTION_MEDIA_BUTTON);
        mediaButton.setClass(app, androidx.media.session.MediaButtonReceiver.class);
        mediaSession.setMediaButtonReceiver(
                PendingIntent.getBroadcast(app, 0, mediaButton, pendingFlags()));
        mediaSession.setActive(true);
        publishLocked();
        return mediaSession;
    }

    public static synchronized MediaSessionCompat.Token getSessionToken(Context context) {
        return getOrCreate(context).getSessionToken();
    }

    public static synchronized void updateState(
            String track,
            String artist,
            String album,
            String artwork,
            boolean playing,
            long durationMs,
            long positionMs) {
        if (track != null) lastTrack = track;
        if (artist != null) lastArtist = artist;
        if (album != null) lastAlbum = album;
        if (artwork != null) lastArtwork = artwork;
        lastPlaying = playing;
        if (durationMs >= 0) lastDurationMs = durationMs;
        if (positionMs >= 0) lastPositionMs = positionMs;
        publishLocked();
    }

    public static synchronized void setQueue(List<QueueItem> items, int activeIndex) {
        queue.clear();
        if (items != null) {
            queue.addAll(items);
        }
        queueIndex = activeIndex;
        if (mediaSession != null) {
            publishQueueLocked();
        }
    }

    private static void publishQueueLocked() {
        if (mediaSession == null) return;
        List<MediaSessionCompat.QueueItem> sessionQueue = new ArrayList<>();
        for (int i = 0; i < queue.size(); i++) {
            QueueItem q = queue.get(i);
            String title = q.title.isEmpty() ? ("Track " + (i + 1)) : q.title;
            MediaDescriptionCompat desc = new MediaDescriptionCompat.Builder()
                    .setMediaId("queue:" + i)
                    .setTitle(title)
                    .setSubtitle(q.artist)
                    .setDescription(q.album)
                    .build();
            sessionQueue.add(new MediaSessionCompat.QueueItem(desc, i));
        }
        mediaSession.setQueue(sessionQueue);
        mediaSession.setQueueTitle("Queue");
        if (queueIndex >= 0 && queueIndex < queue.size()) {
            PlaybackStateCompat state = mediaSession.getController().getPlaybackState();
            long position = lastPositionMs >= 0
                    ? lastPositionMs
                    : PlaybackStateCompat.PLAYBACK_POSITION_UNKNOWN;
            int pbState = lastPlaying
                    ? PlaybackStateCompat.STATE_PLAYING
                    : PlaybackStateCompat.STATE_PAUSED;
            PlaybackStateCompat.Builder b = new PlaybackStateCompat.Builder()
                    .setActions(MEDIA_ACTIONS)
                    .setActiveQueueItemId(queueIndex)
                    .setState(pbState, position, lastPlaying ? 1f : 0f);
            if (state != null) {
                b.setBufferedPosition(state.getBufferedPosition());
            }
            mediaSession.setPlaybackState(b.build());
        }
    }

    public static synchronized List<QueueItem> getQueue() {
        return Collections.unmodifiableList(new ArrayList<>(queue));
    }

    public static synchronized int getQueueIndex() {
        return queueIndex;
    }

    public static synchronized String getLastTrack() {
        return lastTrack;
    }

    public static synchronized String getLastArtist() {
        return lastArtist;
    }

    public static synchronized String getLastAlbum() {
        return lastAlbum;
    }

    public static synchronized String getLastArtwork() {
        return lastArtwork;
    }

    public static synchronized boolean isPlaying() {
        return lastPlaying;
    }

    public static synchronized long getLastDurationMs() {
        return lastDurationMs;
    }

    public static synchronized long getLastPositionMs() {
        return lastPositionMs;
    }

    public static synchronized Bitmap getLastArtBitmap() {
        return lastArtBitmap;
    }

    public static synchronized String getLastArtLoadedUrl() {
        return lastArtLoadedUrl;
    }

    public static synchronized void setArtBitmap(Bitmap bitmap, String url) {
        lastArtBitmap = bitmap;
        lastArtLoadedUrl = url == null ? "" : url;
        publishLocked();
    }

    public static synchronized boolean requestFocus(Context context) {
        if (hasFocus) return true;
        if (audioManager == null) {
            audioManager = (AudioManager) context.getApplicationContext()
                    .getSystemService(Context.AUDIO_SERVICE);
        }
        if (audioManager == null) return false;
        AudioManager.OnAudioFocusChangeListener listener = focusChange -> {
            if (focusChange == AudioManager.AUDIOFOCUS_LOSS
                    || focusChange == AudioManager.AUDIOFOCUS_LOSS_TRANSIENT) {
                hasFocus = false;
                WailsBridge.dispatchMediaAction("pause");
            } else if (focusChange == AudioManager.AUDIOFOCUS_LOSS_TRANSIENT_CAN_DUCK) {
                WailsBridge.dispatchMediaAction("pause");
            } else if (focusChange == AudioManager.AUDIOFOCUS_GAIN) {
                hasFocus = true;
            }
        };
        int result;
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            focusRequest = new AudioFocusRequest.Builder(AudioManager.AUDIOFOCUS_GAIN)
                    .setAudioAttributes(new AudioAttributes.Builder()
                            .setUsage(AudioAttributes.USAGE_MEDIA)
                            .setContentType(AudioAttributes.CONTENT_TYPE_MUSIC)
                            .build())
                    .setOnAudioFocusChangeListener(listener)
                    .setAcceptsDelayedFocusGain(true)
                    .build();
            result = audioManager.requestAudioFocus(focusRequest);
        } else {
            result = audioManager.requestAudioFocus(
                    listener,
                    AudioManager.STREAM_MUSIC,
                    AudioManager.AUDIOFOCUS_GAIN);
        }
        hasFocus = result == AudioManager.AUDIOFOCUS_REQUEST_GRANTED;
        if (!hasFocus) {
            Log.w(TAG, "audio focus not granted: " + result);
        }
        return hasFocus;
    }

    public static synchronized void abandonFocus() {
        if (audioManager == null || !hasFocus) {
            hasFocus = false;
            return;
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O && focusRequest != null) {
            audioManager.abandonAudioFocusRequest(focusRequest);
        } else {
            audioManager.abandonAudioFocus(null);
        }
        hasFocus = false;
    }

    public static synchronized void release() {
        abandonFocus();
        if (mediaSession != null) {
            mediaSession.setActive(false);
            mediaSession.release();
            mediaSession = null;
        }
        lastArtBitmap = null;
        lastArtLoadedUrl = "";
    }

    private static void publishLocked() {
        if (mediaSession == null) return;
        int state = lastPlaying
                ? PlaybackStateCompat.STATE_PLAYING
                : PlaybackStateCompat.STATE_PAUSED;
        long position = lastPositionMs >= 0
                ? lastPositionMs
                : PlaybackStateCompat.PLAYBACK_POSITION_UNKNOWN;
        mediaSession.setPlaybackState(new PlaybackStateCompat.Builder()
                .setActions(MEDIA_ACTIONS)
                .setActiveQueueItemId(queueIndex >= 0
                        ? queueIndex
                        : MediaSessionCompat.QueueItem.UNKNOWN_ID)
                .setState(state, position, lastPlaying ? 1f : 0f)
                .build());

        String title = lastTrack.isEmpty() ? "Melovian" : lastTrack;
        String artist = lastArtist.isEmpty() ? "Unknown artist" : lastArtist;
        MediaMetadataCompat.Builder meta = new MediaMetadataCompat.Builder()
                .putString(MediaMetadataCompat.METADATA_KEY_TITLE, title)
                .putString(MediaMetadataCompat.METADATA_KEY_ARTIST, artist)
                .putString(MediaMetadataCompat.METADATA_KEY_ALBUM, lastAlbum)
                .putString(MediaMetadataCompat.METADATA_KEY_DISPLAY_TITLE, title)
                .putString(MediaMetadataCompat.METADATA_KEY_DISPLAY_SUBTITLE, artist)
                .putString(MediaMetadataCompat.METADATA_KEY_MEDIA_ID, "nowplaying");
        if (!lastAlbum.isEmpty()) {
            meta.putString(MediaMetadataCompat.METADATA_KEY_DISPLAY_DESCRIPTION, lastAlbum);
        }
        if (lastDurationMs > 0) {
            meta.putLong(MediaMetadataCompat.METADATA_KEY_DURATION, lastDurationMs);
        }
        if (lastArtBitmap != null && !lastArtBitmap.isRecycled()) {
            meta.putBitmap(MediaMetadataCompat.METADATA_KEY_ALBUM_ART, lastArtBitmap);
            meta.putBitmap(MediaMetadataCompat.METADATA_KEY_ART, lastArtBitmap);
        }
        mediaSession.setMetadata(meta.build());
        mediaSession.setActive(true);
    }

    static int pendingFlags() {
        return Build.VERSION.SDK_INT >= Build.VERSION_CODES.M
                ? PendingIntent.FLAG_IMMUTABLE | PendingIntent.FLAG_UPDATE_CURRENT
                : PendingIntent.FLAG_UPDATE_CURRENT;
    }
}
