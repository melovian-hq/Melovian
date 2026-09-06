package com.wails.app;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Intent;
import android.content.pm.ServiceInfo;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.os.Build;
import android.os.IBinder;
import android.support.v4.media.session.MediaSessionCompat;
import android.util.Log;

import androidx.annotation.Nullable;
import androidx.core.app.NotificationCompat;
import androidx.media.app.NotificationCompat.MediaStyle;

import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;

/**
 * Foreground media playback service. Publishes MediaSession metadata and a
 * transport notification for lock screen and notification shade controls.
 */
public class WailsForegroundService extends android.app.Service {
    public static final String ACTION_START = "com.melovian.app.FGS_START";
    public static final String ACTION_PLAY = "com.melovian.app.MEDIA_PLAY";
    public static final String ACTION_PAUSE = "com.melovian.app.MEDIA_PAUSE";
    public static final String ACTION_NEXT = "com.melovian.app.MEDIA_NEXT";
    public static final String ACTION_PREVIOUS = "com.melovian.app.MEDIA_PREVIOUS";
    public static final String EXTRA_PLAYING = "playing";
    public static final String EXTRA_TRACK = "track";
    public static final String EXTRA_ARTIST = "artist";
    public static final String EXTRA_ALBUM = "album";
    public static final String EXTRA_ARTWORK = "artwork";
    public static final String EXTRA_DURATION_MS = "durationMs";
    public static final String EXTRA_POSITION_MS = "positionMs";

    private static final String TAG = "MelovianMedia";
    private static final String CHANNEL_ID = "melovian_media";
    private static final int NOTIFICATION_ID = 0x57A1;
    private static int artworkLoadSeq = 0;

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        String action = intent != null ? intent.getAction() : null;
        if (ACTION_PLAY.equals(action)) {
            MelovianMediaSession.requestFocus(this);
            WailsBridge.dispatchMediaAction("play");
            return START_STICKY;
        }
        if (ACTION_PAUSE.equals(action)) {
            WailsBridge.dispatchMediaAction("pause");
            return START_STICKY;
        }
        if (ACTION_NEXT.equals(action)) {
            WailsBridge.dispatchMediaAction("next");
            return START_STICKY;
        }
        if (ACTION_PREVIOUS.equals(action)) {
            WailsBridge.dispatchMediaAction("previous");
            return START_STICKY;
        }

        if (intent != null && ACTION_START.equals(intent.getAction())) {
            boolean playing = intent.hasExtra(EXTRA_PLAYING)
                    ? intent.getBooleanExtra(EXTRA_PLAYING, false)
                    : MelovianMediaSession.isPlaying();
            MelovianMediaSession.updateState(
                    intent.hasExtra(EXTRA_TRACK) ? nvl(intent.getStringExtra(EXTRA_TRACK)) : MelovianMediaSession.getLastTrack(),
                    intent.hasExtra(EXTRA_ARTIST) ? nvl(intent.getStringExtra(EXTRA_ARTIST)) : MelovianMediaSession.getLastArtist(),
                    intent.hasExtra(EXTRA_ALBUM) ? nvl(intent.getStringExtra(EXTRA_ALBUM)) : MelovianMediaSession.getLastAlbum(),
                    intent.hasExtra(EXTRA_ARTWORK) ? nvl(intent.getStringExtra(EXTRA_ARTWORK)) : MelovianMediaSession.getLastArtwork(),
                    playing,
                    intent.hasExtra(EXTRA_DURATION_MS)
                            ? intent.getLongExtra(EXTRA_DURATION_MS, 0)
                            : MelovianMediaSession.getLastDurationMs(),
                    intent.hasExtra(EXTRA_POSITION_MS)
                            ? intent.getLongExtra(EXTRA_POSITION_MS, 0)
                            : MelovianMediaSession.getLastPositionMs());
            if (playing) {
                MelovianMediaSession.requestFocus(this);
            } else {
                MelovianMediaSession.abandonFocus();
            }
        }

        MelovianMediaSession.getOrCreate(this);
        ensureChannel();
        startForegroundWithMediaNotification();
        maybeLoadArtwork(MelovianMediaSession.getLastArtwork());
        return START_STICKY;
    }

    private void ensureChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return;
        NotificationManager nm = (NotificationManager) getSystemService(NOTIFICATION_SERVICE);
        NotificationChannel ch = new NotificationChannel(
                CHANNEL_ID, "Now playing", NotificationManager.IMPORTANCE_LOW);
        ch.setDescription("System media player for lock screen and notification controls");
        ch.setLockscreenVisibility(Notification.VISIBILITY_PUBLIC);
        ch.setShowBadge(false);
        nm.createNotificationChannel(ch);
    }

    private void startForegroundWithMediaNotification() {
        String title = MelovianMediaSession.getLastTrack().isEmpty()
                ? "Melovian"
                : MelovianMediaSession.getLastTrack();
        String artist = MelovianMediaSession.getLastArtist().isEmpty()
                ? "Now playing"
                : MelovianMediaSession.getLastArtist();
        String album = MelovianMediaSession.getLastAlbum();
        boolean playing = MelovianMediaSession.isPlaying();
        Bitmap art = MelovianMediaSession.getLastArtBitmap();
        MediaSessionCompat session = MelovianMediaSession.getOrCreate(this);

        Intent launch = new Intent(this, MainActivity.class);
        launch.setAction(Intent.ACTION_MAIN);
        launch.addCategory(Intent.CATEGORY_LAUNCHER);
        launch.addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP | Intent.FLAG_ACTIVITY_CLEAR_TOP);
        PendingIntent contentIntent = PendingIntent.getActivity(
                this, 0, launch, MelovianMediaSession.pendingFlags());

        PendingIntent previousIntent = PendingIntent.getService(
                this, 1001, new Intent(this, WailsForegroundService.class).setAction(ACTION_PREVIOUS),
                MelovianMediaSession.pendingFlags());
        PendingIntent playIntent = PendingIntent.getService(
                this, 1002, new Intent(this, WailsForegroundService.class).setAction(ACTION_PLAY),
                MelovianMediaSession.pendingFlags());
        PendingIntent pauseIntent = PendingIntent.getService(
                this, 1003, new Intent(this, WailsForegroundService.class).setAction(ACTION_PAUSE),
                MelovianMediaSession.pendingFlags());
        PendingIntent nextIntent = PendingIntent.getService(
                this, 1004, new Intent(this, WailsForegroundService.class).setAction(ACTION_NEXT),
                MelovianMediaSession.pendingFlags());

        NotificationCompat.Builder builder = new NotificationCompat.Builder(this, CHANNEL_ID)
                .setSmallIcon(R.mipmap.ic_launcher)
                .setContentTitle(title)
                .setContentText(artist)
                .setSubText(album)
                .setVisibility(NotificationCompat.VISIBILITY_PUBLIC)
                .setCategory(NotificationCompat.CATEGORY_TRANSPORT)
                .setPriority(NotificationCompat.PRIORITY_LOW)
                .setOnlyAlertOnce(true)
                .setOngoing(true)
                .setShowWhen(false)
                .setContentIntent(contentIntent)
                .addAction(android.R.drawable.ic_media_previous, "Previous", previousIntent);
        if (playing) {
            builder.addAction(android.R.drawable.ic_media_pause, "Pause", pauseIntent);
        } else {
            builder.addAction(android.R.drawable.ic_media_play, "Play", playIntent);
        }
        builder.addAction(android.R.drawable.ic_media_next, "Next", nextIntent);
        if (art != null && !art.isRecycled()) {
            builder.setLargeIcon(art);
        }
        builder.setStyle(new MediaStyle()
                .setMediaSession(session.getSessionToken())
                .setShowActionsInCompactView(0, 1, 2)
                .setShowCancelButton(false));

        Notification n = builder.build();
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            startForeground(NOTIFICATION_ID, n, ServiceInfo.FOREGROUND_SERVICE_TYPE_MEDIA_PLAYBACK);
        } else {
            startForeground(NOTIFICATION_ID, n);
        }
    }

    private void maybeLoadArtwork(String url) {
        if (url == null || url.isEmpty()) return;
        if (url.equals(MelovianMediaSession.getLastArtLoadedUrl())
                && MelovianMediaSession.getLastArtBitmap() != null) {
            return;
        }
        final int seq = ++artworkLoadSeq;
        final String fetchUrl = url;
        new Thread(() -> {
            Bitmap bitmap = downloadBitmap(fetchUrl);
            if (bitmap == null || seq != artworkLoadSeq) return;
            MelovianMediaSession.setArtBitmap(bitmap, fetchUrl);
            startForegroundWithMediaNotification();
        }, "melovian-artwork").start();
    }

    private static Bitmap downloadBitmap(String src) {
        HttpURLConnection conn = null;
        try {
            String fetch = src == null ? "" : src.trim();
            if (fetch.isEmpty() || fetch.startsWith("data:")) {
                return null;
            }
            if (fetch.startsWith("/")) {
                fetch = "https://wails.localhost" + fetch;
            }
            if (!fetch.startsWith("http://") && !fetch.startsWith("https://")) {
                return null;
            }
            URL url = new URL(fetch);
            conn = (HttpURLConnection) url.openConnection();
            conn.setConnectTimeout(4000);
            conn.setReadTimeout(6000);
            conn.setInstanceFollowRedirects(true);
            conn.connect();
            if (conn.getResponseCode() >= 400) return null;
            try (InputStream in = conn.getInputStream()) {
                Bitmap raw = BitmapFactory.decodeStream(in);
                if (raw == null) return null;
                int max = 512;
                if (raw.getWidth() <= max && raw.getHeight() <= max) return raw;
                float scale = Math.min(
                        max / (float) raw.getWidth(),
                        max / (float) raw.getHeight());
                Bitmap scaled = Bitmap.createScaledBitmap(
                        raw,
                        Math.max(1, Math.round(raw.getWidth() * scale)),
                        Math.max(1, Math.round(raw.getHeight() * scale)),
                        true);
                if (scaled != raw) raw.recycle();
                return scaled;
            }
        } catch (Exception e) {
            Log.w(TAG, "artwork download failed", e);
            return null;
        } finally {
            if (conn != null) conn.disconnect();
        }
    }

    private static String nvl(String value) {
        return value == null ? "" : value;
    }

    @Override
    public void onDestroy() {
        MelovianMediaSession.abandonFocus();
        super.onDestroy();
    }

    @Nullable
    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }
}
