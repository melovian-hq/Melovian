// MelovianNowPlaying.m — system Now Playing + remote commands (CarPlay Now Playing).
#import "MelovianNowPlaying.h"
#import <AVFoundation/AVFoundation.h>
#import <Foundation/Foundation.h>
#import <MediaPlayer/MediaPlayer.h>
#import <UIKit/UIKit.h>
#import <WebKit/WebKit.h>

// Soft declarations so this unit links without Wails headers at clang time.
@interface WailsViewController : UIViewController
@property (nonatomic, strong) WKWebView *webView;
- (void)executeJavaScript:(NSString *)js;
@end

@interface WailsAppDelegate : UIResponder
@property (nonatomic, strong) NSMutableArray *viewControllers;
@end

extern WailsAppDelegate *appDelegate;

static BOOL g_jjnp_initialized = NO;

static void jjnpRunOnMain(void (^block)(void)) {
    if ([NSThread isMainThread]) {
        block();
    } else {
        dispatch_async(dispatch_get_main_queue(), block);
    }
}

static NSDictionary *jjnpParseJSON(const char *json) {
    if (json == NULL) return @{};
    NSString *str = [NSString stringWithUTF8String:json];
    if (!str) return @{};
    NSData *data = [str dataUsingEncoding:NSUTF8StringEncoding];
    if (!data) return @{};
    id obj = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
    return [obj isKindOfClass:[NSDictionary class]] ? (NSDictionary *)obj : @{};
}

void melovian_nowplaying_dispatch_action(const char *action) {
    if (action == NULL || action[0] == '\0') return;
    NSString *safe = [[NSString stringWithUTF8String:action]
        stringByReplacingOccurrencesOfString:@"\\" withString:@"\\\\"];
    safe = [safe stringByReplacingOccurrencesOfString:@"'" withString:@"\\'"];
    NSString *js = [NSString stringWithFormat:
        @"(function(){var d={detail:{action:'%@'}};"
         "window.dispatchEvent(new CustomEvent('native-media-action',d));"
         "window.dispatchEvent(new CustomEvent('ios-media-action',d));})();",
        safe];
    jjnpRunOnMain(^{
        if (!appDelegate) return;
        for (id obj in appDelegate.viewControllers) {
            WailsViewController *vc = (WailsViewController *)obj;
            if ([vc respondsToSelector:@selector(executeJavaScript:)]) {
                [vc executeJavaScript:js];
            } else if (vc.webView) {
                [vc.webView evaluateJavaScript:js completionHandler:nil];
            }
        }
    });
}

void melovian_nowplaying_init(void) {
    jjnpRunOnMain(^{
        if (g_jjnp_initialized) return;
        g_jjnp_initialized = YES;

        [[AVAudioSession sharedInstance]
            setCategory:AVAudioSessionCategoryPlayback
            error:nil];
        [[AVAudioSession sharedInstance] setActive:YES error:nil];

        MPRemoteCommandCenter *center = [MPRemoteCommandCenter sharedCommandCenter];
        [center.playCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
            melovian_nowplaying_dispatch_action("play");
            return MPRemoteCommandHandlerStatusSuccess;
        }];
        [center.pauseCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
            melovian_nowplaying_dispatch_action("pause");
            return MPRemoteCommandHandlerStatusSuccess;
        }];
        [center.togglePlayPauseCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
            melovian_nowplaying_dispatch_action("toggle");
            return MPRemoteCommandHandlerStatusSuccess;
        }];
        [center.nextTrackCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
            melovian_nowplaying_dispatch_action("next");
            return MPRemoteCommandHandlerStatusSuccess;
        }];
        [center.previousTrackCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
            melovian_nowplaying_dispatch_action("previous");
            return MPRemoteCommandHandlerStatusSuccess;
        }];
        [center.changePlaybackPositionCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
            MPChangePlaybackPositionCommandEvent *pos =
                (MPChangePlaybackPositionCommandEvent *)event;
            NSString *action = [NSString stringWithFormat:@"seek:%f", pos.positionTime];
            melovian_nowplaying_dispatch_action([action UTF8String]);
            return MPRemoteCommandHandlerStatusSuccess;
        }];
        center.playCommand.enabled = YES;
        center.pauseCommand.enabled = YES;
        center.togglePlayPauseCommand.enabled = YES;
        center.nextTrackCommand.enabled = YES;
        center.previousTrackCommand.enabled = YES;
        center.changePlaybackPositionCommand.enabled = YES;
    });
}

void melovian_nowplaying_set_state(const char *json) {
    melovian_nowplaying_init();
    NSDictionary *opts = jjnpParseJSON(json);
    jjnpRunOnMain(^{
        BOOL active = opts[@"active"] ? [opts[@"active"] boolValue] : YES;
        if (!active) {
            [MPNowPlayingInfoCenter defaultCenter].nowPlayingInfo = nil;
            [MPNowPlayingInfoCenter defaultCenter].playbackState =
                MPNowPlayingPlaybackStateStopped;
            return;
        }

        BOOL playing = [opts[@"playing"] boolValue];
        NSString *title = [opts[@"title"] isKindOfClass:[NSString class]] ? opts[@"title"] : @"";
        NSString *artist = [opts[@"artist"] isKindOfClass:[NSString class]] ? opts[@"artist"] : @"";
        NSString *album = [opts[@"album"] isKindOfClass:[NSString class]] ? opts[@"album"] : @"";
        NSString *artworkURL = [opts[@"artwork"] isKindOfClass:[NSString class]] ? opts[@"artwork"] : @"";
        double duration = [opts[@"duration"] doubleValue];
        double position = [opts[@"position"] doubleValue];

        NSMutableDictionary *info = [NSMutableDictionary dictionary];
        if (title.length) info[MPMediaItemPropertyTitle] = title;
        else info[MPMediaItemPropertyTitle] = @"Melovian";
        if (artist.length) info[MPMediaItemPropertyArtist] = artist;
        if (album.length) info[MPMediaItemPropertyAlbumTitle] = album;
        if (duration > 0) info[MPMediaItemPropertyPlaybackDuration] = @(duration);
        info[MPNowPlayingInfoPropertyElapsedPlaybackTime] = @(position);
        info[MPNowPlayingInfoPropertyPlaybackRate] = playing ? @(1.0) : @(0.0);

        if (artworkURL.length && ![artworkURL hasPrefix:@"data:"]) {
            NSURL *url = [NSURL URLWithString:artworkURL];
            if (url) {
                MPMediaItemArtwork *art = [[MPMediaItemArtwork alloc]
                    initWithBoundsSize:CGSizeMake(512, 512)
                    requestHandler:^UIImage * _Nonnull(CGSize size) {
                        NSData *data = [NSData dataWithContentsOfURL:url];
                        if (!data) return [[UIImage alloc] init];
                        UIImage *img = [UIImage imageWithData:data];
                        return img ?: [[UIImage alloc] init];
                    }];
                info[MPMediaItemPropertyArtwork] = art;
            }
        }

        MPNowPlayingInfoCenter *np = [MPNowPlayingInfoCenter defaultCenter];
        np.nowPlayingInfo = info;
        np.playbackState = playing
            ? MPNowPlayingPlaybackStatePlaying
            : MPNowPlayingPlaybackStatePaused;
    });
}

void melovian_nowplaying_set_queue(const char *json) {
    (void)json;
}
