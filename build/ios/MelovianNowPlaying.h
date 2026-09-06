// MelovianNowPlaying.h — MPNowPlayingInfoCenter bridge for CarPlay Now Playing.
#ifndef MELOVIAN_NOW_PLAYING_H
#define MELOVIAN_NOW_PLAYING_H

#ifdef __cplusplus
extern "C" {
#endif

/** Install remote command handlers. Safe to call more than once. */
void melovian_nowplaying_init(void);

/**
 * Update Now Playing from JSON:
 * {"active":bool,"playing":bool,"title","artist","album","artwork",
 *  "duration":seconds,"position":seconds}
 */
void melovian_nowplaying_set_state(const char *json);

/**
 * Optional queue sync (no CarPlay browse without Apple entitlement).
 * JSON: {"index":int,"tracks":[...]}
 */
void melovian_nowplaying_set_queue(const char *json);

/** Dispatch a transport action into the WKWebView as native-media-action. */
void melovian_nowplaying_dispatch_action(const char *action);

#ifdef __cplusplus
}
#endif

#endif
